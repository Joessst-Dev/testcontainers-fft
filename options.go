package fft

import (
	"bytes"
	"fmt"
	"io/fs"

	"github.com/testcontainers/testcontainers-go"
)

// containerFixtures is where WithSeed / WithSeedFS place the fixtures inside the
// container, and the argument passed to --seed.
const containerFixtures = "/fixtures"

// WithVerbose makes the emulator log one line per request to stderr. The lines
// surface in the container logs.
func WithVerbose() testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Cmd = append(req.Cmd, "--verbose")
		return nil
	}
}

// WithSeed preloads the emulator from a host directory of <collection>.json
// fixtures (facilities.json, orders.json, …), each a single object or an array.
// A seeded document keeps the id and version it carries, so a fixture can pin the
// exact ids a test asserts on.
//
// The directory is copied into the container (tar over the Docker API), not bind
// mounted: the image is distroless with no shell, and a copy also works against a
// remote or rootless Docker host. Mode 0o555 lets the emulator's nonroot user
// (uid 65532) traverse and read the fixtures.
func WithSeed(hostDir string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Files = append(req.Files, testcontainers.ContainerFile{
			HostFilePath:      hostDir,
			ContainerFilePath: containerFixtures,
			FileMode:          0o555,
		})
		req.Cmd = append(req.Cmd, "--seed", containerFixtures)
		return nil
	}
}

// WithSeedFS is [WithSeed] from an in-memory or embedded filesystem (e.g. a
// go:embed tree), so fixtures ship inside the test binary instead of on disk.
// Every regular file in fsys is copied under /fixtures preserving its path.
func WithSeedFS(fsys fs.FS) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		seeded := false
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := fs.ReadFile(fsys, path)
			if err != nil {
				return fmt.Errorf("read seed fixture %q: %w", path, err)
			}
			req.Files = append(req.Files, testcontainers.ContainerFile{
				Reader:            bytes.NewReader(data),
				ContainerFilePath: containerFixtures + "/" + path,
				FileMode:          0o555,
			})
			seeded = true
			return nil
		})
		if err != nil {
			return err
		}
		if seeded {
			req.Cmd = append(req.Cmd, "--seed", containerFixtures)
		}
		return nil
	}
}

// WithPubSubHost points the emulator at an already-running Pub/Sub emulator
// (host:port) and turns eventing on. Use it to wire a sidecar you manage
// yourself; [RunWithPubSub] does this wiring for you.
func WithPubSubHost(hostPort string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Cmd = append(req.Cmd, "--pubsub-emulator-host", hostPort)
		return nil
	}
}
