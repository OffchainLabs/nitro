// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"math/big"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type DataPricer struct {
	backingStorage *storage.Storage
	demand         storage.StorageBackedUint32
	bytesPerSecond storage.StorageBackedUint32
	lastUpdateTime storage.StorageBackedUint64
	minPrice       storage.StorageBackedUint32
	inertia        storage.StorageBackedUint32
}

const (
	demandOffset uint64 = iota
	bytesPerSecondOffset
	lastUpdateTimeOffset
	minPriceOffset
	inertiaOffset
)

const initialDemand = 0                                      // no demand
const InitialHourlyBytes = 1 * (1 << 40) / (365 * 24)        // 1Tb total footprint
const initialBytesPerSecond = InitialHourlyBytes / (60 * 60) // refill each second
const initialLastUpdateTime = 1421388000                     // the day it all began
const initialMinPrice = 82928201                             // 5Mb = $1
const initialInertia = 21360419                              // expensive at 1Tb

func initDataPricer(sto *storage.Storage) {
	demand := sto.OpenStorageBackedUint32(demandOffset)
	bytesPerSecond := sto.OpenStorageBackedUint32(bytesPerSecondOffset)
	lastUpdateTime := sto.OpenStorageBackedUint64(lastUpdateTimeOffset)
	minPrice := sto.OpenStorageBackedUint32(minPriceOffset)
	inertia := sto.OpenStorageBackedUint32(inertiaOffset)
	_ = demand.Set(initialDemand)
	_ = bytesPerSecond.Set(initialBytesPerSecond)
	_ = lastUpdateTime.Set(initialLastUpdateTime)
	_ = minPrice.Set(initialMinPrice)
	_ = inertia.Set(initialInertia)
}

func openDataPricer(sto *storage.Storage) *DataPricer {
	return &DataPricer{
		backingStorage: sto,
		demand:         sto.OpenStorageBackedUint32(demandOffset),
		bytesPerSecond: sto.OpenStorageBackedUint32(bytesPerSecondOffset),
		lastUpdateTime: sto.OpenStorageBackedUint64(lastUpdateTimeOffset),
		minPrice:       sto.OpenStorageBackedUint32(minPriceOffset),
		inertia:        sto.OpenStorageBackedUint32(inertiaOffset),
	}
}

func (p *DataPricer) UpdateModel(tempBytes uint32, time uint64) (*big.Int, error) {
	demand, _ := p.demand.Get()
	bytesPerSecond, _ := p.bytesPerSecond.Get()
	lastUpdateTime, _ := p.lastUpdateTime.Get()
	minPrice, _ := p.minPrice.Get()
	inertia, err := p.inertia.Get()
	if err != nil {
		return nil, err
	}

	timeDelta := arbmath.SaturatingUUCast[uint32](time - lastUpdateTime)
	credit := arbmath.SaturatingUMul(bytesPerSecond, timeDelta)
	demand = arbmath.SaturatingUSub(demand, credit)
	demand = arbmath.SaturatingUAdd(demand, tempBytes)

	if err := p.demand.Set(demand); err != nil {
		return nil, err
	}
	if err := p.lastUpdateTime.Set(time); err != nil {
		return nil, err
	}

	exponent := arbmath.OneInBips * arbmath.Bips(demand) / arbmath.Bips(inertia)
	multiplier := arbmath.ApproxExpBasisPoints(exponent, 12).Uint64()
	costPerByte := arbmath.SaturatingUMul(uint64(minPrice), multiplier) / 10000
	costInWei := arbmath.SaturatingUMul(costPerByte, uint64(tempBytes))
	return arbmath.UintToBig(costInWei), nil
}
