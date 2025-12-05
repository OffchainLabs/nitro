// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const initialMaxWasmSize = 128 * 1024 // max decompressed wasm size (programs are also bounded by compressed size)
const initialStackDepth = 4 * 65536   // 4 page stack.
const InitialFreePages = 2            // 2 pages come free (per tx).
const InitialPageGas = 1000           // linear cost per allocation.
const initialPageRamp = 620674314     // targets 8MB costing 32 million gas, minus the linear term.
const initialPageLimit = 128          // reject wasms with memories larger than 8MB.
const initialInkPrice = 10000         // 1 evm gas buys 10k ink.
const initialMinInitGas = 72          // charge 72 * 128 = 9216 gas.
const initialMinCachedGas = 11        // charge 11 *  32 = 352 gas.
const initialInitCostScalar = 50      // scale costs 1:1 (100%)
const initialCachedCostScalar = 50    // scale costs 1:1 (100%)
const initialExpiryDays = 365         // deactivate after 1 year.
const initialKeepaliveDays = 31       // wait a month before allowing reactivation.
const initialRecentCacheSize = 32     // cache the 32 most recent programs.

const v2MinInitGas = 69 // charge 69 * 128 = 8832 gas (minCachedGas will also be charged in v2).

const MinCachedGasUnits = 32 /// 32 gas for each unit
const MinInitGasUnits = 128  // 128 gas for each unit
const CostScalarPercent = 2  // 2% for each unit

const arbOS50MaxWasmSize = 22000 // Default wasmer stack depth for ArbOS 50

// This struct exists to collect the many Stylus configuration parameters into a single word.
// The items here must only be modified in ArbOwner precompile methods (or in ArbOS upgrades).
type StylusParams struct {
	backingStorage   *storage.Storage
	arbosVersion     uint64 // not stored
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
	MaxWasmSize      uint32
}

// Provides a view of the Stylus parameters. Call Save() to persist.
// Note: this method never returns nil.
func (p Programs) Params() (*StylusParams, error) {
	sto := p.backingStorage.OpenCachedSubStorage(paramsKey)

	// assume reads are warm due to the frequency of access
	if err := sto.Burner().Burn(multigas.ResourceKindComputation, params.WarmStorageReadCostEIP2929); err != nil {
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
	stylusParams := &StylusParams{
		backingStorage:   sto,
		arbosVersion:     p.ArbosVersion,
		Version:          arbmath.BytesToUint16(take(2)),
		InkPrice:         arbmath.BytesToUint24(take(3)),
		MaxStackDepth:    arbmath.BytesToUint32(take(4)),
		FreePages:        arbmath.BytesToUint16(take(2)),
		PageGas:          arbmath.BytesToUint16(take(2)),
		PageRamp:         initialPageRamp,
		PageLimit:        arbmath.BytesToUint16(take(2)),
		MinInitGas:       arbmath.BytesToUint8(take(1)),
		MinCachedInitGas: arbmath.BytesToUint8(take(1)),
		InitCostScalar:   arbmath.BytesToUint8(take(1)),
		CachedCostScalar: arbmath.BytesToUint8(take(1)),
		ExpiryDays:       arbmath.BytesToUint16(take(2)),
		KeepaliveDays:    arbmath.BytesToUint16(take(2)),
		BlockCacheSize:   arbmath.BytesToUint16(take(2)),
	}
	if p.ArbosVersion >= params.ArbosVersion_40 {
		stylusParams.MaxWasmSize = arbmath.BytesToUint32(take(4))
	} else {
		stylusParams.MaxWasmSize = initialMaxWasmSize
	}
	return stylusParams, nil
}

// Writes the params to permanent storage.
func (p *StylusParams) Save() error {
	if p.backingStorage == nil {
		log.Error("tried to Save invalid StylusParams")
		return errors.New("invalid StylusParams")
	}

	// order matters!
	data := arbmath.ConcatByteSlices(
		arbmath.Uint16ToBytes(p.Version),
		arbmath.Uint24ToBytes(p.InkPrice),
		arbmath.Uint32ToBytes(p.MaxStackDepth),
		arbmath.Uint16ToBytes(p.FreePages),
		arbmath.Uint16ToBytes(p.PageGas),
		arbmath.Uint16ToBytes(p.PageLimit),
		arbmath.Uint8ToBytes(p.MinInitGas),
		arbmath.Uint8ToBytes(p.MinCachedInitGas),
		arbmath.Uint8ToBytes(p.InitCostScalar),
		arbmath.Uint8ToBytes(p.CachedCostScalar),
		arbmath.Uint16ToBytes(p.ExpiryDays),
		arbmath.Uint16ToBytes(p.KeepaliveDays),
		arbmath.Uint16ToBytes(p.BlockCacheSize),
	)
	if p.arbosVersion >= params.ArbosVersion_40 {
		data = append(data, arbmath.Uint32ToBytes(p.MaxWasmSize)...)
	}

	slot := uint64(0)
	for len(data) != 0 {
		next := arbmath.MinInt(32, len(data))
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
	switch version {
	case 2:
		if p.Version != 1 {
			return fmt.Errorf("unexpected upgrade from %d to %d", p.Version, version)
		}
		p.Version = 2
		p.MinInitGas = v2MinInitGas
		return nil
	default:
		return fmt.Errorf("unsupported upgrade to %d. Only 2 is supported", version)
	}
}

func (p *StylusParams) UpgradeToArbosVersion(newArbosVersion uint64) error {
	if newArbosVersion == params.ArbosVersion_50 {
		if p.arbosVersion >= params.ArbosVersion_50 {
			return fmt.Errorf("unexpected arbosVersion upgrade to %d from %d", newArbosVersion, p.arbosVersion)
		}
		if p.MaxStackDepth > arbOS50MaxWasmSize {
			p.MaxStackDepth = arbOS50MaxWasmSize
		}
	}
	if newArbosVersion == params.ArbosVersion_40 {
		if p.arbosVersion >= params.ArbosVersion_40 {
			return fmt.Errorf("unexpected arbosVersion upgrade to %d from %d", newArbosVersion, p.arbosVersion)
		}
		if p.Version != 2 {
			return fmt.Errorf("unexpected arbosVersion upgrade to %d while stylus version %d", newArbosVersion, p.Version)
		}
		p.MaxWasmSize = initialMaxWasmSize
	}
	p.arbosVersion = newArbosVersion
	return nil
}

func initStylusParams(arbosVersion uint64, sto *storage.Storage) {
	stylusParams := &StylusParams{
		backingStorage:   sto,
		arbosVersion:     arbosVersion,
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
	if arbosVersion >= params.ArbosVersion_40 {
		stylusParams.MaxWasmSize = initialMaxWasmSize
	}
	_ = stylusParams.Save()
}
