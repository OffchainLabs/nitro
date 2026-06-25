// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/storage"
)

const (
	nextBlockFeesOffset uint64 = iota * uint64(multigas.NumResourceKind)
	currentBlockFeesOffset
)

// MultiGasFees tracks per–resource-kind base fees.
// The `next` field is the base fee for future blocks. It is updated alongside l2pricing.baseFee whenever `updateMultiGasConstraintsBacklogs` is called.
// The `current` field is the base-fee for the current block, and it is updated in `arbos.ProduceBlockAdvanced` before executing transactions.
type MultiGasFees struct {
	next    [multigas.NumResourceKind]storage.StorageBackedBigInt
	current [multigas.NumResourceKind]storage.StorageBackedBigInt
}

// OpenMultiGasFees opens or initializes base fees in the given storage subspace.
func OpenMultiGasFees(sto *storage.Storage) *MultiGasFees {
	r := &MultiGasFees{}
	for offset := range uint64(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		r.next[offset] = sto.OpenStorageBackedBigInt(nextBlockFeesOffset + offset)
		r.current[offset] = sto.OpenStorageBackedBigInt(currentBlockFeesOffset + offset)
	}
	return r
}

// GetCurrentBlockFee returns the current-block base fee for the given resource kind.
func (bf *MultiGasFees) GetCurrentBlockFee(kind multigas.ResourceKind) (*big.Int, error) {
	return bf.current[kind].Get()
}

// GetNextBlockFee returns the next-block base fee for the given resource kind.
func (bf *MultiGasFees) GetNextBlockFee(kind multigas.ResourceKind) (*big.Int, error) {
	return bf.next[kind].Get()
}

// SetNextBlockFee sets the next-block base fee for the given resource kind.
func (bf *MultiGasFees) SetNextBlockFee(kind multigas.ResourceKind, v *big.Int) error {
	return bf.next[kind].SetChecked(v)
}

// CommitNextToCurrent rotates next-block fees into current-block fees.
func (bf *MultiGasFees) CommitNextToCurrent() error {
	for i := range int(multigas.NumResourceKind) {
		cur, err := bf.next[i].Get()
		if err != nil {
			return err
		}
		if cur == nil {
			cur = big.NewInt(0)
		}

		if err := bf.current[i].SetChecked(cur); err != nil {
			return err
		}
	}
	return nil
}
