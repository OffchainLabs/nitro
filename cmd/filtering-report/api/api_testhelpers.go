// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"testing"

	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/util/sqsclient"
)

// NewTestStack creates a filtering-report API stack bound to localhost on
// ephemeral ports. Exported for use in tests across packages.
func NewTestStack(t *testing.T, queueClient sqsclient.QueueClient) *node.Node {
	t.Helper()
	stackConfig := DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSPort = 0
	stack, err := NewStack(&stackConfig, queueClient)
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stack.Close() })
	return stack
}
