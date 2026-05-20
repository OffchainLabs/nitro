// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package lifecycle

import (
	"context"
	"math/big"

	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
)

// Updates dataposter balance to balance at pending block.
func UpdateBalance(ctx context.Context, p dataPoster, s *state.LockedInternalState) error {
	// Use the pending (represented as -1) balance because we're looking at batches we'd post,
	// so we want to see how much gas we could afford with our pending state.
	balance, err := p.Client().BalanceAt(ctx, p.Sender(), big.NewInt(-1))
	if err != nil {
		return err
	}
	s.Balance = balance
	return nil
}
