// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE.md
package blocksreexecutor

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

func newTestBlockChain(t *testing.T, blocks int) *core.BlockChain {
	t.Helper()

	engine := ethash.NewFaker()
	gspec := &core.Genesis{Config: params.TestChainConfig}
	db, generated, _ := core.GenerateChainWithGenesis(gspec, engine, blocks, nil)
	chain, err := core.NewBlockChain(db, nil, gspec, engine, core.DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err != nil {
		t.Fatalf("failed to create blockchain: %v", err)
	}
	if _, err := chain.InsertChain(generated); err != nil {
		chain.Stop()
		t.Fatalf("failed to insert test chain: %v", err)
	}
	return chain
}

func TestGetStateForStartBlockSearchesBackward(t *testing.T) {
	chain := newTestBlockChain(t, 6)
	defer chain.Stop()

	executor := &BlocksReExecutor{
		StopWaiter: stopwaiter.StopWaiter{},
		config:     nil,
		scheme:     rawdb.PathScheme,
		room:       0,
		db:         nil,
		blockchain: chain,
		stateFor: func(header *types.Header) (*state.StateDB, arbitrum.StateReleaseFunc, error) {
			if header.Number.Uint64() == 2 {
				return nil, arbitrum.NoopStateRelease, nil
			}
			return nil, arbitrum.NoopStateRelease, errors.New("state unavailable")
		},
		done:         nil,
		fatalErrChan: nil,
		blocks:       nil,
		mutex:        sync.Mutex{},
		success:      nil,
	}

	startHeader := chain.GetHeaderByNumber(5)
	if startHeader == nil {
		t.Fatal("missing start header")
	}

	_, foundHeader, _, err := executor.getStateForStartBlock(context.Background(), startHeader)
	if err != nil {
		t.Fatalf("expected backward search to find an earlier anchor, got err: %v", err)
	}
	if got, want := foundHeader.Number.Uint64(), uint64(2); got != want {
		t.Fatalf("unexpected fallback anchor, got %d want %d", got, want)
	}
}

func TestGetStateForStartBlockReturnsErrorWhenNoStateExists(t *testing.T) {
	chain := newTestBlockChain(t, 4)
	defer chain.Stop()

	executor := &BlocksReExecutor{
		StopWaiter: stopwaiter.StopWaiter{},
		config:     nil,
		scheme:     rawdb.PathScheme,
		room:       0,
		db:         nil,
		blockchain: chain,
		stateFor: func(header *types.Header) (*state.StateDB, arbitrum.StateReleaseFunc, error) {
			return nil, arbitrum.NoopStateRelease, errors.New("state unavailable")
		},
		done:         nil,
		fatalErrChan: nil,
		blocks:       nil,
		mutex:        sync.Mutex{},
		success:      nil,
	}

	startHeader := chain.GetHeaderByNumber(3)
	if startHeader == nil {
		t.Fatal("missing start header")
	}

	_, _, _, err := executor.getStateForStartBlock(context.Background(), startHeader)
	if err == nil {
		t.Fatal("expected an error when no earlier state exists")
	}
	if !strings.Contains(err.Error(), "moved beyond genesis") {
		t.Fatalf("expected error to report exhausting the search, got: %v", err)
	}
}
