// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/storage"
)

const (
	baseFeesOffset uint64 = iota
)

// MultiGasBaseFees defines the base fees tracked for multiple gas resource kinds.
type MultiGasBaseFees struct {
	baseFees [multigas.NumResourceKind]storage.StorageBackedBigInt
}

// OpenMultiGasBaseFees opens or initializes base fees in the given storage subspace.
func OpenMultiGasBaseFees(sto *storage.Storage) *MultiGasBaseFees {
	r := &MultiGasBaseFees{
		baseFees: [multigas.NumResourceKind]storage.StorageBackedBigInt{},
	}
	for i := range int(multigas.NumResourceKind) {
		// #nosec G115 safe: NumResourceKind < 2^32
		offset := baseFeesOffset + uint64(i)
		r.baseFees[i] = sto.OpenStorageBackedBigInt(offset)
	}
	return r
}

// Get retrieves the base fee for the given resource kind.
func (bf *MultiGasBaseFees) Get(kind multigas.ResourceKind) (*big.Int, error) {
	return bf.baseFees[kind].Get()
}

// Set sets the base fee for the given resource kind.
func (bf *MultiGasBaseFees) Set(kind multigas.ResourceKind, v *big.Int) error {
	return bf.baseFees[kind].SetChecked(v)
}
