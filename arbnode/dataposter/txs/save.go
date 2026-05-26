// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txs

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
)

func SaveTx(ctx context.Context, p dataPoster, s *state.LockedInternalState, prevTx, newTx *storage.QueuedTransaction) error {
	if prevTx != nil {
		if prevTx.FullTx.Nonce() != newTx.FullTx.Nonce() {
			return fmt.Errorf("prevTx nonce %v doesn't match newTx nonce %v", prevTx.FullTx.Nonce(), newTx.FullTx.Nonce())
		}

		// Check if prevTx is the same as newTx and we don't need to do anything
		oldEnc, err := rlp.EncodeToBytes(prevTx)
		if err != nil {
			return fmt.Errorf("failed to encode prevTx: %w", err)
		}
		newEnc, err := rlp.EncodeToBytes(newTx)
		if err != nil {
			return fmt.Errorf("failed to encode newTx: %w", err)
		}
		if bytes.Equal(oldEnc, newEnc) {
			// No need to save newTx as it's the same as prevTx
			return nil
		}
	}
	if err := s.Queue.Put(ctx, newTx.FullTx.Nonce(), prevTx, newTx); err != nil {
		return fmt.Errorf("putting new tx in the queue: %w", err)
	}
	return nil
}
