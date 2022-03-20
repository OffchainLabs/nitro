package main

import (
	"context"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"testing"
)

func TestRelayConfig(t *testing.T) {
	_, err := ParseRelay(context.Background(), []string{
		"--feed.output.port", "9652",
		"--feed.input.url", "ws://sequencer:9642/feed",
	})
	testhelpers.RequireImpl(t, err)
}
