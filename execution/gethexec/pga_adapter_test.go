// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestTxQueueItemComputePgaPriority covers the concrete pga.Tx priority-fee
// computation over real dynamic-fee transactions: the EffectiveGasTip bounds,
// uint64 saturation, the core.ErrFeeCapTooLow mapping for a fee cap below the
// basefee, and propagation of other EffectiveGasTip errors such as
// types.ErrUint256Overflow for >256-bit fee/tip/basefee values. Dropped txs
// must get the error on their result channel.
func TestTxQueueItemComputePgaPriority(t *testing.T) {
	// over64 exceeds uint64, so the effective tip saturates to math.MaxUint64.
	over64 := new(big.Int).Lsh(big.NewInt(1), 70)
	// over256 exceeds uint256, so EffectiveGasTip returns types.ErrUint256Overflow rather than fee-cap-too-low.
	over256 := new(big.Int).Lsh(big.NewInt(1), 300)
	for _, tc := range []struct {
		name      string
		gasFeeCap *big.Int
		gasTipCap *big.Int
		baseFee   *big.Int
		want      uint64
		wantErr   error
	}{
		{name: "tip bound", gasFeeCap: big.NewInt(100), gasTipCap: big.NewInt(5), baseFee: big.NewInt(40), want: 5},
		{name: "cap bound", gasFeeCap: big.NewInt(60), gasTipCap: big.NewInt(50), baseFee: big.NewInt(40), want: 20},
		{name: "zero tip", gasFeeCap: big.NewInt(100), gasTipCap: big.NewInt(0), baseFee: big.NewInt(40), want: 0},
		{name: "fee cap equals base fee", gasFeeCap: big.NewInt(40), gasTipCap: big.NewInt(10), baseFee: big.NewInt(40), want: 0},
		{name: "fee cap below base fee", gasFeeCap: big.NewInt(30), gasTipCap: big.NewInt(10), baseFee: big.NewInt(40), wantErr: core.ErrFeeCapTooLow},
		{name: "saturates to max uint64", gasFeeCap: over64, gasTipCap: over64, baseFee: big.NewInt(1), want: math.MaxUint64},
		{name: "fee cap exceeds 256 bits", gasFeeCap: over256, gasTipCap: over256, baseFee: big.NewInt(40), wantErr: types.ErrUint256Overflow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   big.NewInt(1),
				GasTipCap: tc.gasTipCap,
				GasFeeCap: tc.gasFeeCap,
				Gas:       21000,
				To:        &common.Address{},
				Value:     big.NewInt(0),
			})
			resultChan := make(chan error, 1)
			item := newTxQueueItem(context.Background(), tx, nil, resultChan)
			ok := item.ComputePgaPriority(tc.baseFee)
			if tc.wantErr != nil {
				if ok {
					t.Fatal("ComputePgaPriority = true, want the tx dropped")
				}
				select {
				case err := <-resultChan:
					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("result error = %v, want errors.Is(%v)", err, tc.wantErr)
					}
				default:
					t.Fatalf("no result reported, want %v", tc.wantErr)
				}
				return
			}
			if !ok {
				t.Fatal("ComputePgaPriority = false, want true")
			}
			if item.GetPriority() != tc.want {
				t.Fatalf("priority = %d, want %d", item.GetPriority(), tc.want)
			}
		})
	}
}
