// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"math/big"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
)

func newTestStack(t *testing.T) (*node.Node, *TransactionFiltererAPI) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	txOpts, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}

	stackConfig := DefaultStackConfig
	stackConfig.HTTPHost = "127.0.0.1"
	stackConfig.HTTPPort = 0
	stack, api, err := NewStack(&stackConfig, txOpts, nil)
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
