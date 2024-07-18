// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	am "github.com/offchainlabs/nitro/util/arbmath"
)

const MaxWasmSize = 128 * 1024      // max decompressed wasm size (programs are also bounded by compressed size)
const initialStackDepth = 4 * 65536 // 4 page stack.
const InitialFreePages = 2          // 2 pages come free (per tx).
const InitialPageGas = 1000         // linear cost per allocation.
const initialPageRamp = 620674314   // targets 8MB costing 32 million gas, minus the linear term.
const initialPageLimit = 128        // reject wasms with memories larger than 8MB.
const initialInkPrice = 10000       // 1 evm gas buys 10k ink.
const initialMinInitGas = 72        // charge 72 * 128 = 9216 gas.
const initialMinCachedGas = 11      // charge 11 *  32 = 352 gas.
const initialInitCostScalar = 50    // scale costs 1:1 (100%)
const initialCachedCostScalar = 50  // scale costs 1:1 (100%)
const initialExpiryDays = 365       // deactivate after 1 year.
const initialKeepaliveDays = 31     // wait a month before allowing reactivation.
const initialRecentCacheSize = 32   // cache the 32 most recent programs.

const v2MinInitGas = 69 // charge 69 * 128 = 8832 gas (minCachedGas will also be charged in v2).

const MinCachedGasUnits = 32 /// 32 gas for each unit
const MinInitGasUnits = 128  // 128 gas for each unit
const CostScalarPercent = 2  // 2% for each unit

// This struct exists to collect the many Stylus configuration parameters into a single word.
// The items here must only be modified in ArbOwner precompile methods (or in ArbOS upgrades).
type StylusParams struct {
	backingStorage   *storage.Storage
	Version          uint16 // must only be changed during ArbOS upgrades
	InkPrice         uint24
	MaxStackDepth    uint32
	FreePages        uint16
	PageGas          uint16
	PageRamp         uint64
	PageLimit        uint16
	MinInitGas       uint8 // measured in 128-gas increments
	MinCachedInitGas uint8 // measured in 32-gas increments
	InitCostScalar   uint8 // measured in 2% increments
	CachedCostScalar uint8 // measured in 2% increments
	ExpiryDays       uint16
	KeepaliveDays    uint16
	BlockCacheSize   uint16
}

// Provides a view of the Stylus parameters. Call Save() to persist.
// Note: this method never returns nil.
func (p Programs) Params() (*StylusParams, error) {
	sto := p.backingStorage.OpenCachedSubStorage(paramsKey)

	// assume reads are warm due to the frequency of access
	if err := sto.Burner().Burn(1 * params.WarmStorageReadCostEIP2929); err != nil {
		return &StylusParams{}, err
	}

	// paid for the reads above
	next := uint64(0)
	data := []byte{}
	take := func(count int) []byte {
		if len(data) < count {
			word := sto.GetFree(util.UintToHash(next))
			data = word[:]
			next += 1
		}
		value := data[:count]
		data = data[count:]
		return value
	}

	// order matters!
	return &StylusParams{
		backingStorage:   sto,
		Version:          am.BytesToUint16(take(2)),
		InkPrice:         am.BytesToUint24(take(3)),
		MaxStackDepth:    am.BytesToUint32(take(4)),
		FreePages:        am.BytesToUint16(take(2)),
		PageGas:          am.BytesToUint16(take(2)),
		PageRamp:         initialPageRamp,
		PageLimit:        am.BytesToUint16(take(2)),
		MinInitGas:       am.BytesToUint8(take(1)),
		MinCachedInitGas: am.BytesToUint8(take(1)),
		InitCostScalar:   am.BytesToUint8(take(1)),
		CachedCostScalar: am.BytesToUint8(take(1)),
		ExpiryDays:       am.BytesToUint16(take(2)),
		KeepaliveDays:    am.BytesToUint16(take(2)),
		BlockCacheSize:   am.BytesToUint16(take(2)),
	}, nil
}

// Writes the params to permanent storage.
func (p *StylusParams) Save() error {
	if p.backingStorage == nil {
		log.Error("tried to Save invalid StylusParams")
		return errors.New("invalid StylusParams")
	}

	// order matters!
	data := am.ConcatByteSlices(
		am.Uint16ToBytes(p.Version),
		am.Uint24ToBytes(p.InkPrice),
		am.Uint32ToBytes(p.MaxStackDepth),
		am.Uint16ToBytes(p.FreePages),
		am.Uint16ToBytes(p.PageGas),
		am.Uint16ToBytes(p.PageLimit),
		am.Uint8ToBytes(p.MinInitGas),
		am.Uint8ToBytes(p.MinCachedInitGas),
		am.Uint8ToBytes(p.InitCostScalar),
		am.Uint8ToBytes(p.CachedCostScalar),
		am.Uint16ToBytes(p.ExpiryDays),
		am.Uint16ToBytes(p.KeepaliveDays),
		am.Uint16ToBytes(p.BlockCacheSize),
	)

	slot := uint64(0)
	for len(data) != 0 {
		next := am.MinInt(32, len(data))
		info := data[:next]
		data = data[next:]

		word := common.Hash{}
		copy(word[:], info) // right-pad with zeros
		if err := p.backingStorage.SetByUint64(slot, word); err != nil {
			return err
		}
		slot += 1
	}
	return nil
}

func (p *StylusParams) UpgradeToVersion(version uint16) error {
	if version != 2 {
		return errors.New("dest version not supported for upgrade")
	}
	if p.Version != 1 {
		return errors.New("existing version not supported for upgrade")
	}
	p.Version = 2
	p.MinInitGas = v2MinInitGas
	return nil
}

func initStylusParams(sto *storage.Storage) {
	params := &StylusParams{
		backingStorage:   sto,
		Version:          1,
		InkPrice:         initialInkPrice,
		MaxStackDepth:    initialStackDepth,
		FreePages:        InitialFreePages,
		PageGas:          InitialPageGas,
		PageRamp:         initialPageRamp,
		PageLimit:        initialPageLimit,
		MinInitGas:       initialMinInitGas,
		MinCachedInitGas: initialMinCachedGas,
		InitCostScalar:   initialInitCostScalar,
		CachedCostScalar: initialCachedCostScalar,
		ExpiryDays:       initialExpiryDays,
		KeepaliveDays:    initialKeepaliveDays,
		BlockCacheSize:   initialRecentCacheSize,
	}
	_ = params.Save()
}
