package doctor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs-box/devctl/internal/config"
)

// HTTPClient interface abstracts HTTP client behavior for better testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// URLChecker defines the contract for all URL/TCP health checks (Strategy Pattern).
type URLChecker interface {
	Check(context.Context) error
}

// HTTPURLChecker implements URLChecker for HTTP/HTTPS health checks.
type HTTPURLChecker struct {
	hc     config.Healthcheck
	client HTTPClient
}

// Dialer abstracts network dialing behavior for better testability.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// TCPURLChecker implements URLChecker for TCP connectivity checks.
type TCPURLChecker struct {
	hc     config.Healthcheck
	dialer Dialer
}

// NewHTTPURLChecker creates an HTTPURLChecker instance.
func NewHTTPURLChecker(hc config.Healthcheck, client HTTPClient) *HTTPURLChecker {
	return &HTTPURLChecker{
		hc:     hc,
		client: client,
	}
}

// Check performs an HTTP/HTTPS health check returning the elapsed time in error messages.
func (h *HTTPURLChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, h.hc.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.hc.URL, nil)
	if err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	start := time.Now()
	resp, err := h.client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("unreachable (%s): %w", elapsed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("bad status: %d (%s)", resp.StatusCode, elapsed)
	}

	return nil
}

// NewTCPURLChecker creates a TCPURLChecker instance. If dialer is nil, a
// default net.Dialer is used.
func NewTCPURLChecker(hc config.Healthcheck, dialer Dialer) *TCPURLChecker {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	return &TCPURLChecker{
		hc:     hc,
		dialer: dialer,
	}
}

// Check performs a TCP connectivity check returning the elapsed time in error messages.
func (t *TCPURLChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, t.hc.Timeout)
	defer cancel()

	start := time.Now()
	conn, err := t.dialer.DialContext(ctx, "tcp", t.hc.URL)
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("unreachable (%s): %w", elapsed, err)
	}

	defer conn.Close()
	return nil
}

// NewURLCheckCommand creates the appropriate URLCheckCommand based on the
// health check configuration type (Factory Pattern).
type URLCheckCommand struct {
	hc      config.Healthcheck
	checker URLChecker
}

// checkerFactory builds a URLChecker for a given healthcheck configuration.
type checkerFactory func(config.Healthcheck, HTTPClient) URLChecker

// checkerFactories maps each supported healthcheck type to its factory.
// Adding a new type requires only a new map entry, not a switch statement
// edited in multiple places (see also config.validateHealthcheckURLByType).
var checkerFactories = map[config.HealthcheckType]checkerFactory{
	config.HealthcheckHTTP: func(hc config.Healthcheck, client HTTPClient) URLChecker {
		return NewHTTPURLChecker(hc, client)
	},
	config.HealthcheckTCP: func(hc config.Healthcheck, _ HTTPClient) URLChecker {
		return NewTCPURLChecker(hc, nil)
	},
}

func NewURLCheckCommand(hc config.Healthcheck, client HTTPClient) *URLCheckCommand {
	cmd := &URLCheckCommand{hc: hc}

	if factory, ok := checkerFactories[hc.Type]; ok {
		cmd.checker = factory(hc, client)
	}

	return cmd
}

func (u *URLCheckCommand) Name() string {
	return fmt.Sprintf("url %s [%s]", u.hc.Name, u.hc.Type)
}

func (u *URLCheckCommand) Run(ctx context.Context) error {
	if u.checker == nil {
		return fmt.Errorf("unsupported healthcheck type: %s", u.hc.Type)
	}
	return u.checker.Check(ctx)
}
