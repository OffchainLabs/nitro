// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/node"
)

func newTestStack(t *testing.T) (*node.Node, *FilteringReportAPI) {
	t.Helper()

	stackConfig := DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSPort = 0
	stack, api, err := NewStack(&stackConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stack.Close() })
	return stack, api
}

func TestLiveness(t *testing.T) {
	stack, _ := newTestStack(t)

	resp, err := http.Get(stack.HTTPEndpoint() + "/liveness")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestReadiness(t *testing.T) {
	stack, _ := newTestStack(t)

	resp, err := http.Get(stack.HTTPEndpoint() + "/readiness")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}
