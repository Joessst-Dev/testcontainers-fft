package fft_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	fft "github.com/Joessst-Dev/testcontainers-fft"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	c, err := fft.Run(ctx, fft.DefaultImage)
	testcontainers.CleanupContainer(t, c)
	require.NoError(t, err)

	// The readiness endpoint answers without a token once the emulator is up.
	base := c.MustBaseURL(ctx)
	_, status := get(t, base+"/api/status")
	require.Equal(t, http.StatusOK, status)

	// A stateful collection is reachable and starts empty.
	body, status := get(t, base+"/api/facilities")
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, `"total":0`)
}

func TestRunWithSeed(t *testing.T) {
	ctx := context.Background()

	c, err := fft.Run(ctx, fft.DefaultImage, fft.WithSeed("testdata/fixtures"))
	testcontainers.CleanupContainer(t, c)
	require.NoError(t, err)

	// The pinned id from testdata/fixtures/facilities.json survives seeding, so a
	// list of the seeded collection names it.
	body, status := get(t, c.MustBaseURL(ctx)+"/api/facilities")
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, "11111111-1111-4111-8111-111111111111")
	require.Contains(t, body, "Berlin Warehouse")
}

// get is a small HTTP GET helper returning the body and status.
func get(t *testing.T, url string) (string, int) {
	t.Helper()
	res, err := http.Get(url)
	require.NoError(t, err)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return string(b), res.StatusCode
}
