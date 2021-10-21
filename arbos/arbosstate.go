//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	retryables2 "github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)


type ArbosState struct {
	formatVersion   *big.Int
	gasPool         *storage.StorageBackedInt64
	smallGasPool    *storage.StorageBackedInt64
	gasPriceWei     *big.Int
	l1PricingState  *L1PricingState
	retryables      *retryables2.RetryableState
	timestamp       *uint64
	backingStorage  *storage.Storage
}

func OpenArbosState(stateDB vm.StateDB) *ArbosState {
	backingStorage := storage.NewGeth(stateDB)

	for tryStorageUpgrade(backingStorage) {
	}

	return &ArbosState{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		backingStorage,
	}
}

func tryStorageUpgrade(backingStorage *storage.Storage) bool {
	formatVersion := backingStorage.Get(util.IntToHash(0))
	switch formatVersion {
	case util.IntToHash(0):
		upgrade_0_to_1(backingStorage)
		return true
	default:
		return false
	}
}

type ArbosStateOffset int64
const (
	versionKey ArbosStateOffset = 0
	gasPoolKey = 1
	smallGasPoolKey = 2
	gasPriceKey = 3
	l1PricingKey = 4
	timestampKey = 5
)

type ArbosStateSubspaceID []byte
var (
	retryablesSubspace ArbosStateSubspaceID = []byte{ 0 }
)

func upgrade_0_to_1(backingStorage *storage.Storage) {
	backingStorage.SetByInt64(int64(versionKey), util.IntToHash(1))
	backingStorage.SetByInt64(int64(gasPoolKey), util.IntToHash(GasPoolMax))
	backingStorage.SetByInt64(int64(smallGasPoolKey), util.IntToHash(SmallGasPoolMax))
	backingStorage.SetByInt64(int64(gasPriceKey), util.IntToHash(1000000000)) // 1 gwei
	backingStorage.SetByInt64(int64(l1PricingKey), util.IntToHash(0))
	backingStorage.SetByInt64(int64(timestampKey), util.IntToHash(0))
	retryables2.InitializeRetryableState(backingStorage.Open(retryablesSubspace))
}

func (state *ArbosState) FormatVersion() *big.Int {
	if state.formatVersion == nil {
		state.formatVersion = state.backingStorage.GetByInt64(int64(versionKey)).Big()
	}
	return state.formatVersion
}

func (state *ArbosState) SetFormatVersion(val *big.Int) {
	state.formatVersion = val
	state.backingStorage.SetByInt64(int64(versionKey), common.BigToHash(state.formatVersion))
}

func (state *ArbosState) GasPool() int64 {
	if state.gasPool == nil {
		state.gasPool = state.backingStorage.OpenStorageBackedInt64(util.IntToHash(int64(gasPoolKey)))
	}
	return state.gasPool.Get()
}

func (state *ArbosState) SetGasPool(val int64) {
	if state.gasPool == nil {
		state.gasPool = state.backingStorage.OpenStorageBackedInt64(util.IntToHash(int64(gasPoolKey)))
	}
	state.gasPool.Set(val)
}

func (state *ArbosState) SmallGasPool() int64 {
	if state.smallGasPool == nil {
		state.smallGasPool = state.backingStorage.OpenStorageBackedInt64(util.IntToHash(int64(smallGasPoolKey)))
	}
	return state.smallGasPool.Get()
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	if state.smallGasPool == nil {
		state.smallGasPool = state.backingStorage.OpenStorageBackedInt64(util.IntToHash(int64(smallGasPoolKey)))
	}
	state.smallGasPool.Set(val)
}

func (state *ArbosState) GasPriceWei() *big.Int {
	if state.gasPriceWei == nil {
		state.gasPriceWei = state.backingStorage.GetByInt64(int64(gasPriceKey)).Big()
	}
	return state.gasPriceWei
}

func (state *ArbosState) SetGasPriceWei(val *big.Int) {
	state.gasPriceWei = val
	state.backingStorage.SetByInt64(int64(gasPriceKey), common.BigToHash(val))
}

func (state *ArbosState) RetryableState() *retryables2.RetryableState {
	return retryables2.OpenRetryableState(state.backingStorage.Open(retryablesSubspace))
}

func (state *ArbosState) L1PricingState() *L1PricingState {
	if state.l1PricingState == nil {
		offset := state.backingStorage.GetByInt64(int64(l1PricingKey))
		if offset == (common.Hash{}) {
			l1PricingState, offset := AllocateL1PricingState(state)
			state.l1PricingState = l1PricingState
			state.backingStorage.SetByInt64(int64(l1PricingKey), offset)
		} else {
			state.l1PricingState = OpenL1PricingState(offset, state.backingStorage)
		}
	}
	return state.l1PricingState
}

func (state *ArbosState) LastTimestampSeen() uint64 {
	if state.timestamp == nil {
		ts := state.backingStorage.GetByInt64(int64(timestampKey)).Big().Uint64()
		state.timestamp = &ts
	}
	return *state.timestamp
}

func (state *ArbosState) SetLastTimestampSeen(val uint64) {
	if state.timestamp == nil {
		ts := state.backingStorage.GetByInt64(int64(timestampKey)).Big().Uint64()
		state.timestamp = &ts
	}
	if val < *state.timestamp {
		panic("timestamp decreased")
	}
	if val > *state.timestamp {
		delta := val - *state.timestamp
		ts := val
		state.timestamp = &ts
		state.backingStorage.SetByInt64(int64(timestampKey), util.IntToHash(int64(ts)))
		state.notifyGasPricerThatTimeElapsed(delta)
	}
}
