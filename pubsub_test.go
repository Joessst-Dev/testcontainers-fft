package fft_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	fft "github.com/Joessst-Dev/testcontainers-fft"
)

// TestRunWithPubSub exercises eventing end to end: a create on a stateful
// collection auto-publishes a lifecycle event, which lands on the Pub/Sub
// emulator and is pulled back. It is the container-native form of the emulator
// guide's walkthrough.
func TestRunWithPubSub(t *testing.T) {
	ctx := context.Background()

	stack, err := fft.RunWithPubSub(ctx, fft.DefaultImage)
	t.Cleanup(func() { _ = stack.Terminate(ctx) })
	require.NoError(t, err)

	base := stack.FFT.MustBaseURL(ctx)
	ps, err := stack.PubSubEndpoint(ctx)
	require.NoError(t, err)
	psURL := "http://" + ps

	// A topic and a pull subscription on the Pub/Sub emulator to read from.
	put(t, psURL+"/v1/projects/local/topics/orders", nil)
	put(t, psURL+"/v1/projects/local/subscriptions/reader",
		[]byte(`{"topic":"projects/local/topics/orders"}`))

	// Tell the emulator to publish ORDER_CREATED to that topic.
	post(t, base+"/api/subscriptions", []byte(`{
		"name":"orders","event":"ORDER_CREATED",
		"target":{"type":"GOOGLE_CLOUD_PUB_SUB","projectId":"local","topicId":"orders"}
	}`))

	// Creating an order auto-emits ORDER_CREATED.
	post(t, base+"/api/orders", []byte(`{"tenantOrderId":"order-1"}`))

	// Pull the published message and confirm it is our event.
	body := post(t, psURL+"/v1/projects/local/subscriptions/reader:pull",
		[]byte(`{"maxMessages":10,"returnImmediately":true}`))

	var pull struct {
		ReceivedMessages []struct {
			Message struct {
				Data       string            `json:"data"`
				Attributes map[string]string `json:"attributes"`
			} `json:"message"`
		} `json:"receivedMessages"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &pull))
	require.NotEmpty(t, pull.ReceivedMessages, "expected a published message")

	msg := pull.ReceivedMessages[0].Message
	require.Equal(t, "ORDER_CREATED", msg.Attributes["event"])

	decoded, err := base64.StdEncoding.DecodeString(msg.Data)
	require.NoError(t, err)
	require.Contains(t, string(decoded), `"event":"ORDER_CREATED"`)
}

func put(t *testing.T, url string, body []byte) {
	t.Helper()
	do(t, http.MethodPut, url, body)
}

func post(t *testing.T, url string, body []byte) string {
	t.Helper()
	return do(t, http.MethodPost, url, body)
}

func do(t *testing.T, method, url string, body []byte) string {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Lessf(t, res.StatusCode, 300, "%s %s -> %d: %s", method, url, res.StatusCode, b)
	return string(b)
}
