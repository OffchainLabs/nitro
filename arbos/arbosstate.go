//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math/big"

	"github.com/offchainlabs/arbstate/arbos/addressTable"
	"github.com/offchainlabs/arbstate/arbos/l1pricing"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ArbosState struct {
	formatVersion  uint64
	gasPool        *storage.StorageBackedInt64
	smallGasPool   *storage.StorageBackedInt64
	gasPriceWei    *big.Int
	l1PricingState *l1pricing.L1PricingState
	retryableState *retryables.RetryableState
	addressTable   *addressTable.AddressTable
	sendMerkle     *merkleAccumulator.MerkleAccumulator
	timestamp      *uint64
	backingStorage *storage.Storage
}

func OpenArbosState(stateDB vm.StateDB) *ArbosState {
	backingStorage := storage.NewGeth(stateDB)

	for tryStorageUpgrade(backingStorage) {
	}

	return &ArbosState{
		backingStorage.GetByInt64(int64(versionKey)).Big().Uint64(),
		nil,
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
	formatVersion := backingStorage.GetByInt64(int64(versionKey)).Big().Uint64()
	switch formatVersion {
	case 0:
		upgrade_0_to_1(backingStorage)
		return true
	default:
		return false
	}
}

// Don't change the positions of items in the following const block, because they are part of the storage format
//     definition that ArbOS uses, so changes would break format compatibility.
type ArbosStateOffset int64

const (
	versionKey ArbosStateOffset = iota
	gasPoolKey
	smallGasPoolKey
	gasPriceKey
	timestampKey
)

type ArbosStateSubspaceID []byte

var (
	l1PricingSubspace    ArbosStateSubspaceID = []byte{0}
	retryablesSubspace   ArbosStateSubspaceID = []byte{1}
	addressTableSubspace ArbosStateSubspaceID = []byte{2}
	sendMerkleSubspace   ArbosStateSubspaceID = []byte{3}
)

func upgrade_0_to_1(backingStorage *storage.Storage) {
	backingStorage.SetByInt64(int64(versionKey), util.IntToHash(1))
	backingStorage.SetByInt64(int64(gasPoolKey), util.IntToHash(GasPoolMax))
	backingStorage.SetByInt64(int64(smallGasPoolKey), util.IntToHash(SmallGasPoolMax))
	backingStorage.SetByInt64(int64(gasPriceKey), util.IntToHash(InitialGasPriceWei)) // 1 gwei
	backingStorage.SetByInt64(int64(timestampKey), util.IntToHash(0))
	l1pricing.InitializeL1PricingState(backingStorage.OpenSubStorage(l1PricingSubspace))
	retryables.InitializeRetryableState(backingStorage.OpenSubStorage(retryablesSubspace))
	addressTable.Initialize(backingStorage.OpenSubStorage(addressTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(backingStorage.OpenSubStorage(sendMerkleSubspace))
}

func (state *ArbosState) FormatVersion() uint64 {
	return state.formatVersion
}

func (state *ArbosState) SetFormatVersion(val uint64) {
	state.formatVersion = val
	state.backingStorage.SetByInt64(int64(versionKey), util.IntToHash(int64(val)))
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

func (state *ArbosState) AddToGasPools(val int64) {
	state.SetGasPool(state.GasPool() + val)
	state.SetSmallGasPool(state.SmallGasPool() + val)
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

func (state *ArbosState) RetryableState() *retryables.RetryableState {
	if state.retryableState == nil {
		state.retryableState = retryables.OpenRetryableState(state.backingStorage.OpenSubStorage(retryablesSubspace))
	}
	return state.retryableState
}

func (state *ArbosState) L1PricingState() *l1pricing.L1PricingState {
	if state.l1PricingState == nil {
		state.l1PricingState = l1pricing.OpenL1PricingState(state.backingStorage.OpenSubStorage(l1PricingSubspace))
	}
	return state.l1PricingState
}

func (state *ArbosState) AddressTable() *addressTable.AddressTable {
	if state.addressTable == nil {
		state.addressTable = addressTable.Open(state.backingStorage.OpenSubStorage(addressTableSubspace))
	}
	return state.addressTable
}

func (state *ArbosState) SendMerkleAccumulator() *merkleAccumulator.MerkleAccumulator {
	if state.sendMerkle == nil {
		state.sendMerkle = merkleAccumulator.OpenMerkleAccumulator(state.backingStorage.OpenSubStorage(sendMerkleSubspace))
	}
	return state.sendMerkle
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
