// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
)

func startTestFilterer(t *testing.T) *TransactionFiltererService {
	t.Helper()
	txFilterer := NewTransactionFiltererService(nil, &bind.TransactOpts{})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := txFilterer.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(txFilterer.StopAndWait)
	return txFilterer
}

func TestFilterContextCancelledWhenQueueFull(t *testing.T) {
	txFilterer := NewTransactionFiltererService(nil, &bind.TransactOpts{})

	for range filterQueueSize {
		txFilterer.queue <- common.Hash{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := txFilterer.Filter(ctx, common.Hash{2})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestFilterConsumesFromQueue(t *testing.T) {
	txFilterer := startTestFilterer(t)

	const n = 5
	for i := range n {
		if err := txFilterer.Filter(context.Background(), common.BytesToHash([]byte{byte(i)})); err != nil {
			t.Fatal(err)
		}
	}

	timeout := time.After(5 * time.Second)
	for len(txFilterer.queue) > 0 {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for queue to drain")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func newTestStack(t *testing.T) (*node.Node, *TransactionFiltererService) {
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
	stack, txFilterer, err := NewStack(&stackConfig, txOpts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stack.Close() })
	return stack, txFilterer
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
