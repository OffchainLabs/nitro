// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
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
// types.ErrUint256Overflow for >256-bit fee/tip/basefee values.
func TestTxQueueItemComputePgaPriority(t *testing.T) {
	huge := new(big.Int).Lsh(big.NewInt(1), 70) // ~1.18e21, well above math.MaxUint64
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
		{name: "saturates to max uint64", gasFeeCap: huge, gasTipCap: huge, baseFee: big.NewInt(1), want: math.MaxUint64},
		{name: "fee cap exceeds 256 bits", gasFeeCap: over256, gasTipCap: over256, baseFee: big.NewInt(40), wantErr: types.ErrUint256Overflow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := txQueueItem{tx: types.NewTx(&types.DynamicFeeTx{
				ChainID:   big.NewInt(1),
				GasTipCap: tc.gasTipCap,
				GasFeeCap: tc.gasFeeCap,
				Gas:       21000,
				To:        &common.Address{},
				Value:     big.NewInt(0),
			})}
			got, err := item.ComputePgaPriority(tc.baseFee)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want errors.Is(%v)", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("priority fee = %d, want %d", got, tc.want)
			}
		})
	}
}
