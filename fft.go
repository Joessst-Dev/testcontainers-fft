// Package fft runs the fulfillmenttools API emulator as a Testcontainers
// container, so an integration test gets a fresh, disposable API on a random
// port with automatic readiness and teardown.
//
// The emulator is the same binary the fft CLI ships (image
// ghcr.io/joessst-dev/fft). It holds all state in memory and makes no request to
// any tenant, and it needs no authentication — a test just points an HTTP client
// at [Container.BaseURL].
package fft

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DefaultImage is the emulator image pinned to a tested release. Pass another
// image to [Run] to override it — the tag tracks the fft CLI's own semver, so a
// test can hold a specific emulator build. Never default to a floating tag: a
// moving image makes a test suite non-reproducible.
const DefaultImage = "ghcr.io/joessst-dev/fft:0.3.0"

// port is the container's fixed listen port. The host port is always mapped
// dynamically, so this is only ever used inside the container and for the wait.
const port = nat.Port("8080/tcp")

// Container is a running fft emulator.
type Container struct {
	testcontainers.Container
}

// Run starts an fft emulator from img and blocks until it is ready to serve.
//
// Readiness is a token-free HTTP 200 from GET /api/status: the emulator answers
// it the moment it is listening, and it needs no auth. (The emulator serves a
// list envelope there rather than the real API's {"status":"UP"}, so the wait
// asserts the status code, not the body.)
//
// On failure Run may still return a non-nil *Container so the caller can hand it
// to testcontainers.CleanupContainer / TerminateContainer for teardown.
func Run(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        img,
		ExposedPorts: []string{string(port)},
		// --host 0.0.0.0 is mandatory: the emulator defaults to 127.0.0.1, which
		// answers only inside the container, so the mapped port would be dead.
		Cmd:        []string{"emulator", "--host", "0.0.0.0"},
		WaitingFor: statusWait(),
	}

	generic := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}
	for _, opt := range opts {
		if err := opt.Customize(&generic); err != nil {
			return nil, fmt.Errorf("customize fft container request: %w", err)
		}
	}

	ctr, err := testcontainers.GenericContainer(ctx, generic)
	var c *Container
	if ctr != nil {
		c = &Container{Container: ctr}
	}
	if err != nil {
		return c, fmt.Errorf("run fft emulator: %w", err)
	}
	return c, nil
}

// statusWait waits for GET /api/status to return 200. The emulator answers every
// route with 200 once its port is bound, so the status code alone is the ready
// signal; the response body is not asserted (see [Run]).
func statusWait() wait.Strategy {
	return wait.ForHTTP("/api/status").
		WithPort(port).
		WithStartupTimeout(60 * time.Second).
		WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK })
}

// BaseURL is the emulator's base URL from the host, http://host:<mapped-port>.
// Point an HTTP client or the fft CLI's FFT_BASE_URL at it.
func (c *Container) BaseURL(ctx context.Context) (string, error) {
	return c.PortEndpoint(ctx, port, "http")
}

// MustBaseURL is [Container.BaseURL] for a test that treats a failure to read the
// endpoint as fatal. It panics on error.
func (c *Container) MustBaseURL(ctx context.Context) string {
	url, err := c.BaseURL(ctx)
	if err != nil {
		panic(fmt.Errorf("fft base url: %w", err))
	}
	return url
}

// HostPort is the host port the emulator's 8080 is published on.
func (c *Container) HostPort(ctx context.Context) (nat.Port, error) {
	return c.MappedPort(ctx, port)
}
