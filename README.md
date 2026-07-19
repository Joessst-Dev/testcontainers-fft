# testcontainers-fft

A [Testcontainers for Go](https://golang.testcontainers.org) module for the
[fulfillmenttools](https://fulfillmenttools.com) API **emulator** — the offline,
in-memory API server that the [`fft` CLI](https://github.com/Joessst-Dev/fft-cli) ships.
It gives an integration test a fresh, disposable fulfillmenttools API per run: a random
host port, automatic readiness, automatic teardown. No tenant, no credentials, no network
to the real platform.

```go
import (
    "context"
    "net/http"

    "github.com/testcontainers/testcontainers-go"
    fft "github.com/Joessst-Dev/testcontainers-fft"
)

func TestSomething(t *testing.T) {
    ctx := context.Background()

    c, err := fft.Run(ctx, fft.DefaultImage)
    testcontainers.CleanupContainer(t, c) // stops it when the test ends
    require.NoError(t, err)

    base := c.MustBaseURL(ctx) // http://host:<mapped-port>
    res, _ := http.Get(base + "/api/facilities")
    // ... drive the API, assert on responses
}
```

## Install

```sh
go get github.com/Joessst-Dev/testcontainers-fft
```

Requires Go 1.23+ and a Docker daemon reachable by Testcontainers.

## What it does

`fft.Run` starts `ghcr.io/joessst-dev/fft` with `emulator --host 0.0.0.0` and blocks until
the emulator is listening. Readiness is a token-free `GET /api/status` returning `200` (the
emulator needs no auth at all). The image tag is pinned by [`fft.DefaultImage`](./fft.go);
pass a different image to `Run` to override it.

The emulator remembers the top-level REST collections (facilities, listings, stocks,
orders, subscriptions, …): a create is stored, a get reflects it, versions and both
pagination models work, and optimistic-locking `409`s are real. Everything else is answered
from a spec-synthesized response. See the
[emulator guide](https://github.com/Joessst-Dev/fft-cli/blob/main/docs/guide/emulator.md)
for the full model.

## Accessors

| Method | Returns |
| --- | --- |
| `BaseURL(ctx)` / `MustBaseURL(ctx)` | `http://host:<mapped-port>` |
| `HostPort(ctx)` | the mapped host port |

`*fft.Container` embeds `testcontainers.Container`, so its full API (`Endpoint`, `Logs`,
`Exec`, `Terminate`, …) is available too.

## Options

```go
c, err := fft.Run(ctx, fft.DefaultImage,
    fft.WithSeed("testdata/fixtures"), // preload <collection>.json fixtures
    fft.WithVerbose(),                 // one log line per request
)
```

- **`WithSeed(hostDir)`** — preload the emulator from a directory of `<collection>.json`
  files (`facilities.json`, `orders.json`, …), each a single object or an array. A seeded
  document keeps the `id` and `version` it carries, so a fixture can pin the exact ids a
  test asserts on. The directory is copied into the container (not bind mounted), so it
  works with the distroless image and remote Docker hosts.
- **`WithSeedFS(fs.FS)`** — the same, from an embedded (`go:embed`) filesystem, so fixtures
  ship inside the test binary.
- **`WithVerbose()`** — the emulator logs one line per request to the container logs.
- **`WithPubSubHost(hostPort)`** — point eventing at a Pub/Sub emulator you manage. For a
  managed sidecar, use `RunWithPubSub` instead.

## Eventing

The emulator can publish domain events to a local Google Pub/Sub emulator. `RunWithPubSub`
starts both containers on a shared network and wires them together:

```go
stack, err := fft.RunWithPubSub(ctx, fft.DefaultImage)
defer stack.Terminate(ctx)

base := stack.FFT.MustBaseURL(ctx)         // the fft emulator
psHostPort, _ := stack.PubSubEndpoint(ctx) // the Pub/Sub emulator, host:port
```

Register a subscription with a `GOOGLE_CLOUD_PUB_SUB` target, then a create/update/delete on
a stateful collection publishes its lifecycle event (a create on `orders` emits
`ORDER_CREATED`, and so on). See [`pubsub_test.go`](./pubsub_test.go) for a full round-trip.

## Versioning

`fft.DefaultImage` pins a tested emulator release; the tag tracks the
[`fft` CLI](https://github.com/Joessst-Dev/fft-cli/releases)'s own semver. Override it by
passing another image to `Run`. This module is versioned independently and published by git
tag (indexed on [pkg.go.dev](https://pkg.go.dev/github.com/Joessst-Dev/testcontainers-fft)).

## License

MIT — see [LICENSE](./LICENSE).
