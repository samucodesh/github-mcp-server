## 2024-08-14 - Reuse http.Client for Performance
**Learning:** Creating a new `http.Client` for every HTTP request is a performance anti-pattern in Go. It prevents the reuse of underlying TCP connections, leading to unnecessary overhead from repeated TCP and TLS handshakes.
**Action:** Always create and reuse a single, shared `http.Client` instance for making multiple HTTP requests to the same host. This allows Go's HTTP transport to pool and reuse connections, significantly improving performance.
