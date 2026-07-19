package fft

import (
	"context"
	"errors"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// pubSubImage is the Google Cloud SDK image carrying the Pub/Sub emulator.
const pubSubImage = "gcr.io/google.com/cloudsdktool/cloud-sdk:emulators"

// pubSubPort is the Pub/Sub emulator's fixed listen port.
const pubSubPort = "8085/tcp"

// pubSubAlias is the network alias the fft container reaches the Pub/Sub emulator
// by. --pubsub-emulator-host must name a host on the shared network, not
// localhost, so the two containers talk over the Docker network.
const pubSubAlias = "pubsub"

// Stack is an fft emulator wired to a Pub/Sub emulator on a shared network, for
// tests that exercise eventing end to end.
type Stack struct {
	// FFT is the fft emulator, with eventing pointed at PubSub.
	FFT *Container
	// PubSub is the Google Pub/Sub emulator the events are published to.
	PubSub testcontainers.Container

	network *testcontainers.DockerNetwork
}

// RunWithPubSub starts a Pub/Sub emulator and an fft emulator wired to it, both
// on a fresh network, and blocks until both are ready. Mutating a stateful
// collection then publishes a lifecycle event to the Pub/Sub emulator, which a
// test reads back via [Stack.PubSubEndpoint].
//
// Extra opts customise the fft container, exactly as with [Run]; the eventing
// wiring is added on top. On any failure a partial [Stack] may be returned so the
// caller can call [Stack.Terminate] for teardown.
func RunWithPubSub(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*Stack, error) {
	stack := &Stack{}

	net, err := network.New(ctx)
	if err != nil {
		return stack, fmt.Errorf("create network: %w", err)
	}
	stack.network = net

	ps, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          pubSubImage,
			ExposedPorts:   []string{pubSubPort},
			Networks:       []string{net.Name},
			NetworkAliases: map[string][]string{net.Name: {pubSubAlias}},
			Cmd: []string{
				"gcloud", "beta", "emulators", "pubsub", "start",
				"--host-port=0.0.0.0:8085", "--project=local",
			},
			WaitingFor: wait.ForLog("Server started, listening on 8085"),
		},
		Started: true,
	})
	if ps != nil {
		stack.PubSub = ps
	}
	if err != nil {
		return stack, fmt.Errorf("run pubsub emulator: %w", err)
	}

	opts = append(opts,
		network.WithNetwork([]string{"fft"}, net),
		WithPubSubHost(pubSubAlias+":8085"),
	)
	c, err := Run(ctx, img, opts...)
	if c != nil {
		stack.FFT = c
	}
	if err != nil {
		return stack, fmt.Errorf("run fft emulator: %w", err)
	}
	return stack, nil
}

// PubSubEndpoint is the Pub/Sub emulator's host:port from the host — the value to
// give a Pub/Sub client's PUBSUB_EMULATOR_HOST, or to hit its REST API.
func (s *Stack) PubSubEndpoint(ctx context.Context) (string, error) {
	if s.PubSub == nil {
		return "", errors.New("pubsub container is not running")
	}
	return s.PubSub.PortEndpoint(ctx, pubSubPort, "")
}

// Terminate stops both containers and removes the network. It joins every error
// so a single failure does not leak the rest.
func (s *Stack) Terminate(ctx context.Context) error {
	var errs []error
	if s.FFT != nil {
		errs = append(errs, s.FFT.Terminate(ctx))
	}
	if s.PubSub != nil {
		errs = append(errs, s.PubSub.Terminate(ctx))
	}
	if s.network != nil {
		errs = append(errs, s.network.Remove(ctx))
	}
	return errors.Join(errs...)
}
