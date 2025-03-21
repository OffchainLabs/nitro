// Copyright 2025, Offchain Labs, Inc.
// For license information, see
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package features

import (
	"github.com/offchainlabs/nitro/arbos/storage"
)

const (
	// This should work for the first 256 features. After that, either add
	// another member to the Features struct, or switch to StorageBackedBytes.
	increasedCalldata int = iota
)

// Features is a thin wrapper around a storage.StorageBackedBigUint that
// provides accessors for various feature toggles.
type Features struct {
	features storage.StorageBackedBigUint
}

// SetIncreasedCalldataPriceIncrease sets the increased calldata price feature
// on or off depending on the value of enabled.
func (f *Features) SetCalldataPriceIncrease(enabled bool) {
	f.setBit(increasedCalldata, enabled)
}

// IsIncreasedCalldataPriceEnabled returns true if the increased calldata price
// feature is enabled.
func (f *Features) IsIncreasedCalldataPriceEnabled() bool {
	return f.isSet(increasedCalldata)
}

func (f *Features) setBit(index int, enabled bool) {
	bit := uint(1)
	if !enabled {
		bit = 0
	}
	// Features cannot be uninitialized.
	bi, _ := f.features.Get()
	bi.SetBit(bi, index, bit)
	// This won't underflow or overflow. Panic if it does.
	if err := f.features.SetChecked(bi); err != nil {
		panic(err)
	}
}

func (f *Features) isSet(index int) bool {
	// Features cannot be uninitialized.
	bi, _ := f.features.Get()
	return bi.Bit(index) == 1
}

func Open(sto *storage.Storage) *Features {
	return &Features{
		features: sto.OpenStorageBackedBigUint(0),
	}
}
