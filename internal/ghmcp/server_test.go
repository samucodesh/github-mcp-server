package ghmcp

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMCPServer_CreatesSuccessfully verifies that the server can be created
// with the deps injection middleware properly configured.
func TestNewMCPServer_CreatesSuccessfully(t *testing.T) {
	t.Parallel()

	// Create a minimal server configuration
	cfg := MCPServerConfig{
		Version:           "test",
		Host:              "", // defaults to github.com
		Token:             "test-token",
		EnabledToolsets:   []string{"context"},
		ReadOnly:          false,
		Translator:        translations.NullTranslationHelper,
		ContentWindowSize: 5000,
		LockdownMode:      false,
	}

	// Create the server
	server, err := NewMCPServer(cfg)
	require.NoError(t, err, "expected server creation to succeed")
	require.NotNil(t, server, "expected server to be non-nil")

	// The fact that the server was created successfully indicates that:
	// 1. The deps injection middleware is properly added
	// 2. Tools can be registered without panicking
	//
	// If the middleware wasn't properly added, tool calls would panic with
	// "ToolDependencies not found in context" when executed.
	//
	// The actual middleware functionality and tool execution with ContextWithDeps
	// is already tested in pkg/github/*_test.go.
}

// TestResolveEnabledToolsets verifies the toolset resolution logic.
func TestResolveEnabledToolsets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfg            MCPServerConfig
		expectedResult []string
	}{
		{
			name: "nil toolsets without dynamic mode and no tools - use defaults",
			cfg: MCPServerConfig{
				EnabledToolsets: nil,
				DynamicToolsets: false,
				EnabledTools:    nil,
			},
			expectedResult: nil, // nil means "use defaults"
		},
		{
			name: "nil toolsets with dynamic mode - start empty",
			cfg: MCPServerConfig{
				EnabledToolsets: nil,
				DynamicToolsets: true,
				EnabledTools:    nil,
			},
			expectedResult: []string{}, // empty slice means no toolsets
		},
		{
			name: "explicit toolsets",
			cfg: MCPServerConfig{
				EnabledToolsets: []string{"repos", "issues"},
				DynamicToolsets: false,
			},
			expectedResult: []string{"repos", "issues"},
		},
		{
			name: "empty toolsets - disable all",
			cfg: MCPServerConfig{
				EnabledToolsets: []string{},
				DynamicToolsets: false,
			},
			expectedResult: []string{}, // empty slice means no toolsets
		},
		{
			name: "specific tools without toolsets - no default toolsets",
			cfg: MCPServerConfig{
				EnabledToolsets: nil,
				DynamicToolsets: false,
				EnabledTools:    []string{"get_me"},
			},
			expectedResult: []string{}, // empty slice when tools specified but no toolsets
		},
		{
			name: "dynamic mode with explicit toolsets removes all and default",
			cfg: MCPServerConfig{
				EnabledToolsets: []string{"all", "repos"},
				DynamicToolsets: true,
			},
			expectedResult: []string{"repos"}, // "all" is removed in dynamic mode
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveEnabledToolsets(tc.cfg)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

// mockRoundTripper is a custom http.RoundTripper for testing.
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// TestCheckSubdomainIsolation_CachesResult verifies that the checkSubdomainIsolation
// function caches the result and only makes one network request for the same host.
func TestCheckSubdomainIsolation_CachesResult(t *testing.T) {
	t.Parallel()

	var requestCount int32
	var lastRequestURL string
	scheme := "https"
	hostname := "test.ghes.com"
	expectedURL := "https://raw.test.ghes.com/_ping"

	// Create a custom RoundTripper to intercept requests
	rt := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&requestCount, 1)
			lastRequestURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	client := &http.Client{Transport: rt}

	// Clear cache before running the test to ensure isolation
	cacheKey := fmt.Sprintf("%s://%s", scheme, hostname)
	subdomainIsolationCacheMutex.Lock()
	delete(subdomainIsolationCache, cacheKey)
	subdomainIsolationCacheMutex.Unlock()

	// First call - should make a network request
	result1 := checkSubdomainIsolation(client, scheme, hostname)
	assert.True(t, result1, "expected first call to return true")
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "expected one network request after first call")
	assert.Equal(t, expectedURL, lastRequestURL, "unexpected URL requested")

	// Second call - should use the cache
	result2 := checkSubdomainIsolation(client, scheme, hostname)
	assert.True(t, result2, "expected second call to return true")
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "expected still one network request after second call")

	// Third call - should still use the cache
	result3 := checkSubdomainIsolation(client, scheme, hostname)
	assert.True(t, result3, "expected third call to return true")
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "expected still one network request after third call")
}
