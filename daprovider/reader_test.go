// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDangerousAlwaysFallbackReader(t *testing.T) {
	r := &DangerousAlwaysFallbackReader{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const batchNum = uint64(42)

	calls := []struct {
		name string
		fn   func() error
	}{
		{"RecoverPayload", func() error {
			_, err := r.RecoverPayload(batchNum, common.Hash{}, nil).Await(ctx)
			return err
		}},
		{"CollectPreimages", func() error {
			_, err := r.CollectPreimages(batchNum, common.Hash{}, nil).Await(ctx)
			return err
		}},
		{"RecoverPayloadAndPreimages", func() error {
			_, err := r.RecoverPayloadAndPreimages(batchNum, common.Hash{}, nil).Await(ctx)
			return err
		}},
	}

	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			err := c.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrAnyTrustRequiresFallback) {
				t.Fatalf("errors.Is(ErrAnyTrustRequiresFallback) = false, got: %v", err)
			}
			var typed *AnyTrustRequiresFallbackError
			if !errors.As(err, &typed) {
				t.Fatalf("errors.As(*AnyTrustRequiresFallbackError) = false, got: %v", err)
			}
			if typed.BatchNum != batchNum {
				t.Fatalf("BatchNum = %d, want %d", typed.BatchNum, batchNum)
			}
		})
	}
}
