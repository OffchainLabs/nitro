// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/execution/gethexec/pga"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// txQueueItem implements pga.Tx so it can drive the PGA priority mempool (execution/gethexec/pga) without that package
// depending on gethexec. The methods use value receivers so the mempool can hold txQueueItems by value, matching the
// txQueue channel.
var _ pga.Tx = txQueueItem{}

func (i txQueueItem) ComputePgaPriority(baseFee *big.Int) (uint64, error) {
	tip, err := i.tx.EffectiveGasTip(baseFee)
	if err != nil {
		if errors.Is(err, types.ErrGasFeeCapTooLow) {
			// Preserve the existing fee-cap-too-low error message from sequencer.go.
			return 0, fmt.Errorf("%w: maxFeePerGas: %s baseFee: %s", core.ErrFeeCapTooLow, i.tx.GasFeeCap(), baseFee)
		}
		return 0, fmt.Errorf("unexpected EffectiveGasTip error for tx %v: %w", i.tx.Hash(), err)
	}
	// TODO(NIT-5043): add anti-starvation boost
	return arbmath.BigToUintSaturating(tip), nil
}

func (i txQueueItem) ReportError(err error) { i.returnResult(err) }

func (i txQueueItem) GetContext() context.Context { return i.ctx }

func (i txQueueItem) GetSize() int { return i.txSize }

func (i txQueueItem) GetFirstAppearance() time.Time { return i.firstAppearance }
