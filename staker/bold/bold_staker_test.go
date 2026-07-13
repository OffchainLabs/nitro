// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package bold

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/protocol"
)

func TestResolveWasmModuleRoot(t *testing.T) {
	parentHash := common.HexToHash("0x01")
	parentRoot := common.HexToHash("0xaa")
	fallback := common.HexToHash("0xbb")
	readErr := errors.New("read failed")

	t.Run("has parent uses parent root", func(t *testing.T) {
		fallbackCalled := false
		latest := &protocol.AssertionCreatedInfo{
			ParentAssertionHash: protocol.AssertionHash{Hash: parentHash},
		}
		readInfo := func(hash common.Hash) (*protocol.AssertionCreatedInfo, error) {
			if hash != parentHash {
				t.Fatalf("unexpected hash read: %v", hash)
			}
			return &protocol.AssertionCreatedInfo{WasmModuleRoot: parentRoot}, nil
		}
		fallbackRoot := func() (common.Hash, error) {
			fallbackCalled = true
			return fallback, nil
		}
		got, err := resolveWasmModuleRoot(latest, readInfo, fallbackRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != parentRoot {
			t.Fatalf("got %v, want parent root %v", got, parentRoot)
		}
		if fallbackCalled {
			t.Fatal("fallbackRoot should not be called when a parent exists")
		}
	})

	t.Run("no staked assertion falls back to rollup root", func(t *testing.T) {
		readInfo := func(common.Hash) (*protocol.AssertionCreatedInfo, error) {
			t.Fatal("readInfo should not be called when there is no previous assertion")
			return nil, nil
		}
		fallbackRoot := func() (common.Hash, error) {
			return fallback, nil
		}
		got, err := resolveWasmModuleRoot(nil, readInfo, fallbackRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != fallback {
			t.Fatalf("got %v, want fallback root %v", got, fallback)
		}
	})

	t.Run("genesis falls back to rollup root", func(t *testing.T) {
		latest := &protocol.AssertionCreatedInfo{
			ParentAssertionHash: protocol.AssertionHash{Hash: common.Hash{}},
		}
		readInfo := func(common.Hash) (*protocol.AssertionCreatedInfo, error) {
			t.Fatal("readInfo should not be called for a genesis assertion")
			return nil, nil
		}
		fallbackRoot := func() (common.Hash, error) {
			return fallback, nil
		}
		got, err := resolveWasmModuleRoot(latest, readInfo, fallbackRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != fallback {
			t.Fatalf("got %v, want fallback root %v", got, fallback)
		}
	})

	t.Run("parent read error is propagated", func(t *testing.T) {
		latest := &protocol.AssertionCreatedInfo{
			ParentAssertionHash: protocol.AssertionHash{Hash: parentHash},
		}
		readInfo := func(common.Hash) (*protocol.AssertionCreatedInfo, error) {
			return nil, readErr
		}
		fallbackRoot := func() (common.Hash, error) {
			t.Fatal("fallbackRoot should not be called when the parent read fails")
			return common.Hash{}, nil
		}
		got, err := resolveWasmModuleRoot(latest, readInfo, fallbackRoot)
		if !errors.Is(err, readErr) {
			t.Fatalf("got error %v, want %v", err, readErr)
		}
		if got != (common.Hash{}) {
			t.Fatalf("got %v, want zero hash on error", got)
		}
	})
}
