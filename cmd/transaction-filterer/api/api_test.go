// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode"
)

func startTestAPI(t *testing.T) *TransactionFiltererAPI {
	t.Helper()
	api := NewTransactionFiltererAPI(nil, &bind.TransactOpts{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := api.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(api.StopAndWait)
	return api
}

func TestFilterContextCancelledWhenQueueFull(t *testing.T) {
	api := NewTransactionFiltererAPI(nil, &bind.TransactOpts{}, nil)

	for range filterQueueSize {
		api.filterQueue <- common.Hash{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := api.Filter(ctx, common.Hash{2})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestUnfilterContextCancelledWhenQueueFull(t *testing.T) {
	api := NewTransactionFiltererAPI(nil, &bind.TransactOpts{}, nil)

	for range filterQueueSize {
		api.unfilterQueue <- common.Hash{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := api.Unfilter(ctx, common.Hash{2})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestFilterConsumesFromQueue(t *testing.T) {
	api := startTestAPI(t)

	const n = 5
	for i := range n {
		if err := api.Filter(context.Background(), common.BytesToHash([]byte{byte(i)})); err != nil {
			t.Fatal(err)
		}
	}

	timeout := time.After(5 * time.Second)
	for len(api.filterQueue) > 0 {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for queue to drain")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestUnfilterConsumesFromQueue(t *testing.T) {
	api := startTestAPI(t)

	const n = 5
	for i := range n {
		if err := api.Unfilter(context.Background(), common.BytesToHash([]byte{byte(i)})); err != nil {
			t.Fatal(err)
		}
	}

	timeout := time.After(5 * time.Second)
	for len(api.unfilterQueue) > 0 {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for queue to drain")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

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
	stack, api, err := NewStack(&stackConfig, txOpts, nil, nil)
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

func TestValidatePruneOptions(t *testing.T) {
	validClient := &ethclient.Client{}
	validBridge := &arbnode.DelayedBridge{}
	validConfig := PruneConfig{
		Enable:               true,
		PollInterval:         time.Second,
		ParentBlockChunkSize: 100,
	}
	base := PruneOptions{
		Config:            validConfig,
		ChainId:           big.NewInt(1),
		ParentChainClient: validClient,
		ChildChainClient:  validClient,
		DelayedBridge:     validBridge,
	}
	withMod := func(mod func(*PruneOptions)) *PruneOptions {
		opts := base
		mod(&opts)
		return &opts
	}

	cases := []struct {
		name    string
		opts    *PruneOptions
		wantErr string
	}{
		{"nil options", nil, ""},
		{"disabled", &PruneOptions{Config: PruneConfig{Enable: false}}, ""},
		{"zero poll interval", withMod(func(o *PruneOptions) { o.Config.PollInterval = 0 }), "poll-interval"},
		{"zero chunk size", withMod(func(o *PruneOptions) { o.Config.ParentBlockChunkSize = 0 }), "parent-block-chunk-size"},
		{"nil chain id", withMod(func(o *PruneOptions) { o.ChainId = nil }), "chain ID"},
		{"nil parent client", withMod(func(o *PruneOptions) { o.ParentChainClient = nil }), "parent chain client"},
		{"nil child client", withMod(func(o *PruneOptions) { o.ChildChainClient = nil }), "child chain client"},
		{"nil bridge", withMod(func(o *PruneOptions) { o.DelayedBridge = nil }), "delayed bridge"},
		{"all valid", withMod(func(*PruneOptions) {}), ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePruneOptions(tc.opts)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}
