// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"math/big"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type dataPricer struct {
	backingStorage     *storage.Storage
	poolBytes          storage.StorageBackedInt64
	poolBytesPerSecond storage.StorageBackedInt64
	maxPoolBytes       storage.StorageBackedInt64
	lastUpdateTime     storage.StorageBackedUint64
	minPrice           storage.StorageBackedUint32
	inertia            storage.StorageBackedUint64
}

const initialPoolBytes = initialMaxPoolBytes
const initialPoolBytesPerSecond = initialMaxPoolBytes / (60 * 60) // refill each hour
const initialMaxPoolBytes = 4 * (1 << 40) / (365 * 24)            // 4Tb total footprint
const initialLastUpdateTime = 1421388000                          // the day it all began
const initialMinPrice = 10                                        // one USD
const initialInertia = 70832408                                   // expensive at 4Tb

func openDataPricer(sto *storage.Storage) *dataPricer {
	return &dataPricer{
		backingStorage:     sto,
		poolBytes:          sto.OpenStorageBackedInt64(initialPoolBytes),
		poolBytesPerSecond: sto.OpenStorageBackedInt64(initialPoolBytesPerSecond),
		maxPoolBytes:       sto.OpenStorageBackedInt64(initialMaxPoolBytes),
		lastUpdateTime:     sto.OpenStorageBackedUint64(initialLastUpdateTime),
		minPrice:           sto.OpenStorageBackedUint32(initialMinPrice),
		inertia:            sto.OpenStorageBackedUint64(initialInertia),
	}
}

func (p *dataPricer) updateModel(tempBytes int64, time uint64) (*big.Int, error) {
	poolBytes, _ := p.poolBytes.Get()
	poolBytesPerSecond, _ := p.poolBytesPerSecond.Get()
	maxPoolBytes, _ := p.maxPoolBytes.Get()
	lastUpdateTime, _ := p.lastUpdateTime.Get()
	minPrice, _ := p.minPrice.Get()
	inertia, err := p.inertia.Get()
	if err != nil {
		return nil, err
	}

	timeDelta := arbmath.SaturatingCast(time - lastUpdateTime)
	credit := arbmath.SaturatingMul(poolBytesPerSecond, timeDelta)
	poolBytes = arbmath.MinInt(arbmath.SaturatingAdd(poolBytes, credit), maxPoolBytes)
	poolBytes = arbmath.SaturatingSub(poolBytes, tempBytes)

	if err := p.poolBytes.Set(poolBytes); err != nil {
		return nil, err
	}

	cost := big.NewInt(arbmath.SaturatingMul(int64(minPrice), tempBytes))

	if poolBytes < 0 {
		excess := arbmath.SaturatingNeg(poolBytes)
		exponent := arbmath.NaturalToBips(excess) / arbmath.Bips(inertia)
		cost = arbmath.BigMulByBips(cost, arbmath.ApproxExpBasisPoints(exponent, 12))
	}
	return cost, nil
}
