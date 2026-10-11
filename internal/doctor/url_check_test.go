package doctor

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs-box/devctl/internal/config"
)

// mockHTTPClient is a test double for HTTPClient interface.
type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// TestHTTPURLCheckerCheckSuccess verifies successful HTTP health check.
func TestHTTPURLCheckerCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	hc := config.Healthcheck{
		Name:    "test",
		URL:     server.URL,
		Timeout: time.Second,
	}

	checker := NewHTTPURLChecker(hc, server.Client())
	err := checker.Check(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestHTTPURLCheckerCheckBadStatus verifies detection of bad HTTP status codes.
func TestHTTPURLCheckerCheckBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	hc := config.Healthcheck{
		Name:    "test",
		URL:     server.URL,
		Timeout: time.Second,
	}

	checker := NewHTTPURLChecker(hc, server.Client())
	err := checker.Check(context.Background())

	if err == nil {
		t.Fatal("expected error for bad status, got nil")
	}
	if !contains(err.Error(), "bad status: 500") {
		t.Fatalf("expected 'bad status: 500' error, got %v", err)
	}
}

// TestHTTPURLCheckerCheckConnectionError verifies error handling for HTTP connection failures.
func TestHTTPURLCheckerCheckConnectionError(t *testing.T) {
	hc := config.Healthcheck{
		Name:    "test",
		URL:     "http://invalid-url-that-will-fail.local",
		Timeout: 100 * time.Millisecond,
	}

	checker := NewHTTPURLChecker(hc, &mockHTTPClient{
		err: errors.New("connection refused"),
	})
	err := checker.Check(context.Background())

	if err == nil {
		t.Fatal("expected error for connection failure, got nil")
	}
	if !contains(err.Error(), "unreachable") {
		t.Fatalf("expected 'unreachable' error, got %v", err)
	}
}

// TestHTTPURLCheckerCheckTimeout verifies timeout handling in HTTP checks.
func TestHTTPURLCheckerCheckTimeout(t *testing.T) {
	hc := config.Healthcheck{
		Name:    "test",
		URL:     "http://example.com",
		Timeout: time.Nanosecond,
	}

	slowClient := &mockHTTPClient{
		err: context.DeadlineExceeded,
	}

	checker := NewHTTPURLChecker(hc, slowClient)
	err := checker.Check(context.Background())

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestHTTPURLCheckerCheckInvalidURL verifies error handling for malformed URLs.
func TestHTTPURLCheckerCheckInvalidURL(t *testing.T) {
	hc := config.Healthcheck{
		Name:    "test",
		URL:     "ht!tp://[invalid:url",
		Timeout: time.Second,
	}

	checker := NewHTTPURLChecker(hc, &mockHTTPClient{})
	err := checker.Check(context.Background())

	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
	if !contains(err.Error(), "invalid request") {
		t.Fatalf("expected 'invalid request' error, got %v", err)
	}
}

// TestTCPURLCheckerCheckSuccess verifies successful TCP connectivity check.
func TestTCPURLCheckerCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	hc := config.Healthcheck{
		Name:    "test",
		URL:     server.Listener.Addr().String(),
		Timeout: time.Second,
	}

	checker := NewTCPURLChecker(hc, nil)
	err := checker.Check(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestTCPURLCheckerCheckConnectionError verifies error handling for TCP connection failures.
func TestTCPURLCheckerCheckConnectionError(t *testing.T) {
	hc := config.Healthcheck{
		Name:    "test",
		URL:     "localhost:1",
		Timeout: 100 * time.Millisecond,
	}

	checker := NewTCPURLChecker(hc, nil)
	err := checker.Check(context.Background())

	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !contains(err.Error(), "unreachable") {
		t.Fatalf("expected 'unreachable' error, got %v", err)
	}
}

// TestTCPURLCheckerCheckInvalidAddress verifies error handling for invalid TCP addresses.
func TestTCPURLCheckerCheckInvalidAddress(t *testing.T) {
	hc := config.Healthcheck{
		Name:    "test",
		URL:     "not:a:valid:address",
		Timeout: 100 * time.Millisecond,
	}

	checker := NewTCPURLChecker(hc, nil)
	err := checker.Check(context.Background())

	if err == nil {
		t.Fatal("expected error for invalid address, got nil")
	}
}

// mockDialer is a test double for Dialer interface.
type mockDialer struct {
	err error
}

func (m *mockDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, m.err
}

// TestTCPURLCheckerCheckWithInjectedDialer verifies the dialer can be injected
// for testability, without requiring a real network connection.
func TestTCPURLCheckerCheckWithInjectedDialer(t *testing.T) {
	hc := config.Healthcheck{
		Name:    "test",
		URL:     "unused:0",
		Timeout: time.Second,
	}

	checker := NewTCPURLChecker(hc, &mockDialer{err: errors.New("simulated dial failure")})
	err := checker.Check(context.Background())

	if err == nil {
		t.Fatal("expected error from injected dialer, got nil")
	}
	if !contains(err.Error(), "unreachable") {
		t.Fatalf("expected 'unreachable' error, got %v", err)
	}
}

// TestURLCheckCommandFactoryHTTP verifies HTTP checker creation via factory.
func TestURLCheckCommandFactoryHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	hc := config.Healthcheck{
		Name:    "http-check",
		Type:    "http",
		URL:     server.URL,
		Timeout: time.Second,
	}

	cmd := NewURLCheckCommand(hc, server.Client())
	err := cmd.Run(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestURLCheckCommandFactoryTCP verifies TCP checker creation via factory.
func TestURLCheckCommandFactoryTCP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	hc := config.Healthcheck{
		Name:    "tcp-check",
		Type:    "tcp",
		URL:     server.Listener.Addr().String(),
		Timeout: time.Second,
	}

	cmd := NewURLCheckCommand(hc, server.Client())
	err := cmd.Run(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestURLCheckCommandUnsupportedType verifies error for unsupported checker types.
func TestURLCheckCommandUnsupportedType(t *testing.T) {
	hc := config.Healthcheck{
		Name:    "invalid-check",
		Type:    "ftp",
		URL:     "ftp://example.com",
		Timeout: time.Second,
	}

	cmd := NewURLCheckCommand(hc, &mockHTTPClient{})
	err := cmd.Run(context.Background())

	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
	if !contains(err.Error(), "unsupported healthcheck type: ftp") {
		t.Fatalf("expected 'unsupported' error, got %v", err)
	}
}

// TestURLCheckCommandName verifies correct naming format for URL check commands.
func TestURLCheckCommandName(t *testing.T) {
	hc := config.Healthcheck{
		Name: "my-service",
		Type: "http",
		URL:  "http://example.com",
	}

	cmd := NewURLCheckCommand(hc, &mockHTTPClient{})
	expectedName := "url my-service [http]"
	actualName := cmd.Name()

	if actualName != expectedName {
		t.Fatalf("expected name %q, got %q", expectedName, actualName)
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
