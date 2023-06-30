// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestRelayConfig(t *testing.T) {
	args := strings.Split("--node.feed.output.port 9652 --node.feed.input.url ws://sequencer:9642/feed", " ")
	_, err := relay.ParseRelay(context.Background(), args)
	testhelpers.RequireImpl(t, err)
}
