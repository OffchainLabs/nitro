// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/storage"
)

const (
	currentBlockFeesOffset uint64 = iota * uint64(multigas.NumResourceKind)
	lastBlockFeesOffset
)

// MultiGasFees tracks per–resource-kind base fees for current and last blocks.
type MultiGasFees struct {
	current [multigas.NumResourceKind]storage.StorageBackedBigInt
	last    [multigas.NumResourceKind]storage.StorageBackedBigInt
}

// OpenMultiGasFees opens or initializes base fees in the given storage subspace.
func OpenMultiGasFees(sto *storage.Storage) *MultiGasFees {
	r := &MultiGasFees{}
	for offset := range uint64(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		r.current[offset] = sto.OpenStorageBackedBigInt(currentBlockFeesOffset + offset)
		r.last[offset] = sto.OpenStorageBackedBigInt(lastBlockFeesOffset + offset)
	}
	return r
}

// GetLast returns the last-committed base fee for the given resource kind.
func (bf *MultiGasFees) GetLast(kind multigas.ResourceKind) (*big.Int, error) {
	return bf.last[kind].Get()
}

// GetCurrent returns the current-block base fee for the given resource kind.
func (bf *MultiGasFees) GetCurrent(kind multigas.ResourceKind) (*big.Int, error) {
	return bf.current[kind].Get()
}

// SetCurrent sets the current-block base fee for the given resource kind.
func (bf *MultiGasFees) SetCurrent(kind multigas.ResourceKind, v *big.Int) error {
	return bf.current[kind].SetChecked(v)
}

// CommitCurrentToLast rotates current-block fees into last-block fees and clears current-block fees.
func (bf *MultiGasFees) CommitCurrentToLast() error {
	for i := range int(multigas.NumResourceKind) {
		cur, err := bf.current[i].Get()
		if err != nil {
			return err
		}
		if cur == nil {
			cur = big.NewInt(0)
		}

		// Set current to last.
		if err := bf.last[i].SetChecked(cur); err != nil {
			return err
		}

		// Zeroize current.
		if err := bf.current[i].SetChecked(big.NewInt(0)); err != nil {
			return err
		}
	}
	return nil
}
