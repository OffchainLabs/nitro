// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"math/big"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type dataPricer struct {
	backingStorage *storage.Storage
	demand         storage.StorageBackedUint32
	bytesPerSecond storage.StorageBackedUint32
	lastUpdateTime storage.StorageBackedUint64
	minPrice       storage.StorageBackedUint32
	inertia        storage.StorageBackedUint32
}

const initialDemand = 0                                      // no demand
const initialHourlyBytes = 4 * (1 << 40) / (365 * 24)        // 4Tb total footprint
const initialBytesPerSecond = initialHourlyBytes / (60 * 60) // refill each hour
const initialLastUpdateTime = 1421388000                     // the day it all began
const initialMinPrice = 82928201                             // 5Mb = $1
const initialInertia = 70177364                              // expensive at 4Tb

func openDataPricer(sto *storage.Storage) *dataPricer {
	return &dataPricer{
		backingStorage: sto,
		demand:         sto.OpenStorageBackedUint32(initialDemand),
		bytesPerSecond: sto.OpenStorageBackedUint32(initialBytesPerSecond),
		lastUpdateTime: sto.OpenStorageBackedUint64(initialLastUpdateTime),
		minPrice:       sto.OpenStorageBackedUint32(initialMinPrice),
		inertia:        sto.OpenStorageBackedUint32(initialInertia),
	}
}

func (p *dataPricer) updateModel(tempBytes uint32, time uint64) (*big.Int, error) {
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

	exponent := arbmath.OneInBips * arbmath.Bips(demand) / arbmath.Bips(inertia)
	multiplier := arbmath.ApproxExpBasisPoints(exponent, 12).Uint64()
	costPerByte := arbmath.SaturatingUMul(uint64(minPrice), multiplier)
	costInWei := arbmath.SaturatingUMul(costPerByte, uint64(tempBytes))
	return arbmath.UintToBig(costInWei), nil
}
