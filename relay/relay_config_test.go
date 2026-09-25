// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package relay

import (
	"context"
	"testing"
)

// TestParseRelayRejectsFeedBackfill pins that the shared feed input flags cannot turn on backfill
// for a relay: it rebroadcasts what it receives, so a capped catchup window would be its whole
// backlog. Silently ignoring the flag would let an operator believe it took effect.
func TestParseRelayRejectsFeedBackfill(t *testing.T) {
	ctx := context.Background()
	if _, err := ParseRelay(ctx, []string{"--node.feed.input.rest.enable"}); err == nil {
		t.Fatal("expected the relay to refuse feed backfill")
	}
	if _, err := ParseRelay(ctx, []string{"--node.feed.input.rest.url=http://archive.example"}); err != nil {
		t.Fatalf("a rest url without enable should parse: %v", err)
	}
}
