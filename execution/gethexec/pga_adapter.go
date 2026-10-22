// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/execution/gethexec/pga"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var _ pga.Tx = txQueueItem{}

func (i txQueueItem) ComputePgaPriority(baseFee *big.Int) bool {
	tip, err := i.tx.EffectiveGasTip(baseFee)
	if err != nil {
		if errors.Is(err, types.ErrGasFeeCapTooLow) {
			// Preserve the existing fee-cap-too-low error message from sequencer.go.
			err = fmt.Errorf("%w: maxFeePerGas: %s baseFee: %s", core.ErrFeeCapTooLow, i.tx.GasFeeCap(), baseFee)
		} else {
			err = fmt.Errorf("unexpected EffectiveGasTip error for tx %v: %w", i.tx.Hash(), err)
		}
		i.returnResult(err)
		return false
	}
	i.SetTip(arbmath.SaturatingCastToUint(tip))
	return true
}

func (i txQueueItem) Validate() bool {
	if err := i.ctx.Err(); err != nil {
		i.returnResult(err)
		return false
	}
	return true
}

func (i txQueueItem) GetFirstAppearance() time.Time { return i.firstAppearance }
