package log

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestIOLogger(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	logOut := &bytes.Buffer{}

	logger := slog.New(slog.NewTextHandler(logOut, nil))
	ioLogger := NewIOLogger(in, out, logger)

	// Test Read
	in.WriteString("hello")
	p := make([]byte, 5)
	n, err := ioLogger.Read(p)
	if err != nil {
		t.Errorf("unexpected error on read: %v", err)
	}
	if n != 5 {
		t.Errorf("expected to read 5 bytes, got %d", n)
	}
	if string(p) != "hello" {
		t.Errorf("expected to read 'hello', got '%s'", string(p))
	}
	if !strings.Contains(logOut.String(), "hello") {
		t.Errorf("expected log to contain 'hello', got '%s'", logOut.String())
	}

	// Test Write
	logOut.Reset()
	n, err = ioLogger.Write([]byte("world"))
	if err != nil {
		t.Errorf("unexpected error on write: %v", err)
	}
	if n != 5 {
		t.Errorf("expected to write 5 bytes, got %d", n)
	}
	if out.String() != "world" {
		t.Errorf("expected to write 'world', got '%s'", out.String())
	}
	if !strings.Contains(logOut.String(), "world") {
		t.Errorf("expected log to contain 'world', got '%s'", logOut.String())
	}

	// Test Close
	err = ioLogger.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}

	// Test Read from closed
	_, err = ioLogger.Read(p)
	if err != io.EOF {
		t.Errorf("expected EOF on read from closed, got %v", err)
	}

	// Test Write to closed
	_, err = ioLogger.Write([]byte("world"))
	if err != io.ErrClosedPipe {
		t.Errorf("expected ErrClosedPipe on write to closed, got %v", err)
	}
}

func TestRedaction(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	logOut := &bytes.Buffer{}

	logger := slog.New(slog.NewTextHandler(logOut, nil))
	ioLogger := NewIOLogger(in, out, logger)

	// Test Write with token
	token := "ghp_123456789012345678901234567890123456"
	input := `{"token":"` + token + `"}`
	n, err := ioLogger.Write([]byte(input))
	if err != nil {
		t.Errorf("unexpected error on write: %v", err)
	}
	if n != len(input) {
		t.Errorf("expected to write %d bytes, got %d", len(input), n)
	}
	if out.String() != input {
		t.Errorf("expected to write '%s', got '%s'", input, out.String())
	}
	if strings.Contains(logOut.String(), token) {
		t.Errorf("expected log not to contain token, got '%s'", logOut.String())
	}
	if !strings.Contains(logOut.String(), "ghp_************************************") {
		t.Errorf("expected log to contain redacted token, got '%s'", logOut.String())
	}
}
