package fft_test

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"

	fft "github.com/Joessst-Dev/testcontainers-fft"
)

// ExampleRun starts the emulator, reads its base URL, and tears it down.
func ExampleRun() {
	ctx := context.Background()

	c, err := fft.Run(ctx, fft.DefaultImage)
	defer func() { _ = testcontainers.TerminateContainer(c) }()
	if err != nil {
		fmt.Println("run:", err)
		return
	}

	base, err := c.BaseURL(ctx)
	if err != nil {
		fmt.Println("base url:", err)
		return
	}
	fmt.Println("emulator ready:", base != "")
	// Output: emulator ready: true
}
