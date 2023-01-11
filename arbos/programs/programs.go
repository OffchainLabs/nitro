// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package programs

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type Programs struct {
	backingStorage *storage.Storage
	wasmGasPrice   storage.StorageBackedUBips
	wasmMaxDepth   storage.StorageBackedUint32
	wasmHeapBound  storage.StorageBackedUint32
	version        storage.StorageBackedUint32
}

const (
	versionOffset uint64 = iota
	wasmGasPriceOffset
	wasmMaxDepthOffset
	wasmHeapBoundOffset
)

func Initialize(sto *storage.Storage) {
	wasmGasPrice := sto.OpenStorageBackedBips(wasmGasPriceOffset)
	wasmMaxDepth := sto.OpenStorageBackedUint32(wasmMaxDepthOffset)
	wasmHeapBound := sto.OpenStorageBackedUint32(wasmHeapBoundOffset)
	version := sto.OpenStorageBackedUint32(versionOffset)
	_ = wasmGasPrice.Set(0)
	_ = wasmMaxDepth.Set(math.MaxUint32)
	_ = wasmHeapBound.Set(math.MaxUint32)
	_ = version.Set(1)
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage: sto,
		wasmGasPrice:   sto.OpenStorageBackedUBips(wasmGasPriceOffset),
		wasmMaxDepth:   sto.OpenStorageBackedUint32(wasmMaxDepthOffset),
		wasmHeapBound:  sto.OpenStorageBackedUint32(wasmHeapBoundOffset),
		version:        sto.OpenStorageBackedUint32(versionOffset),
	}
}

func (p Programs) StylusVersion() (uint32, error) {
	return p.version.Get()
}

func (p Programs) WasmGasPrice() (arbmath.UBips, error) {
	return p.wasmGasPrice.Get()
}

func (p Programs) SetWasmGasPrice(price arbmath.UBips) error {
	return p.wasmGasPrice.Set(price)
}

func (p Programs) WasmMaxDepth() (uint32, error) {
	return p.wasmMaxDepth.Get()
}

func (p Programs) SetWasmMaxDepth(depth uint32) error {
	return p.wasmMaxDepth.Set(depth)
}

func (p Programs) WasmHeapBound() (uint32, error) {
	return p.wasmHeapBound.Get()
}

func (p Programs) SetWasmHeapBound(bound uint32) error {
	return p.wasmHeapBound.Set(bound)
}
