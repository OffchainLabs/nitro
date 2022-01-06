//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math/big"

	"github.com/offchainlabs/arbstate/arbos/addressSet"

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
	maxGasPriceWei *big.Int // the max gas price ArbOS can set without breaking geth
	l1PricingState *l1pricing.L1PricingState
	retryableState *retryables.RetryableState
	addressTable   *addressTable.AddressTable
	chainOwners    *addressSet.AddressSet
	sendMerkle     *merkleAccumulator.MerkleAccumulator
	timestamp      *storage.StorageBackedUint64
	backingStorage *storage.Storage
}

func OpenArbosState(stateDB vm.StateDB) *ArbosState {
	backingStorage := storage.NewGeth(stateDB)

	for tryStorageUpgrade(backingStorage) {
	}

	return &ArbosState{
		backingStorage.GetUint64ByUint64(uint64(versionKey)),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		backingStorage.OpenStorageBackedUint64(util.UintToHash(uint64(timestampKey))),
		backingStorage,
	}
}

func tryStorageUpgrade(backingStorage *storage.Storage) bool {
	formatVersion := backingStorage.GetUint64ByUint64(uint64(versionKey))
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
	maxPriceKey
	timestampKey
)

type ArbosStateSubspaceID []byte

var (
	l1PricingSubspace    ArbosStateSubspaceID = []byte{0}
	retryablesSubspace   ArbosStateSubspaceID = []byte{1}
	addressTableSubspace ArbosStateSubspaceID = []byte{2}
	chainOwnerSubspace   ArbosStateSubspaceID = []byte{3}
	sendMerkleSubspace   ArbosStateSubspaceID = []byte{4}
)

func upgrade_0_to_1(backingStorage *storage.Storage) {
	backingStorage.SetUint64ByUint64(uint64(versionKey), 1)
	backingStorage.SetUint64ByUint64(uint64(gasPoolKey), GasPoolMax)
	backingStorage.SetUint64ByUint64(uint64(smallGasPoolKey), SmallGasPoolMax)
	backingStorage.SetUint64ByUint64(uint64(gasPriceKey), InitialGasPriceWei)
	backingStorage.SetUint64ByUint64(uint64(maxPriceKey), 2*InitialGasPriceWei)
	backingStorage.SetUint64ByUint64(uint64(timestampKey), 0)
	l1pricing.InitializeL1PricingState(backingStorage.OpenSubStorage(l1PricingSubspace))
	retryables.InitializeRetryableState(backingStorage.OpenSubStorage(retryablesSubspace))
	addressTable.Initialize(backingStorage.OpenSubStorage(addressTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(backingStorage.OpenSubStorage(sendMerkleSubspace))

	// the zero address is the initial chain owner
	ZeroAddressL2 := util.RemapL1Address(common.Address{})
	ownersStorage := backingStorage.OpenSubStorage(chainOwnerSubspace)
	addressSet.Initialize(ownersStorage)
	addressSet.OpenAddressSet(ownersStorage).Add(ZeroAddressL2)

	backingStorage.SetUint64ByUint64(uint64(versionKey), 1)
}

func (state *ArbosState) FormatVersion() uint64 {
	return state.formatVersion
}

func (state *ArbosState) SetFormatVersion(val uint64) {
	state.formatVersion = val
	state.backingStorage.SetUint64ByUint64(uint64(versionKey), val)
}

func (state *ArbosState) GasPool() int64 {
	if state.gasPool == nil {
		state.gasPool = state.backingStorage.OpenStorageBackedInt64(util.UintToHash(uint64(gasPoolKey)))
	}
	return state.gasPool.Get()
}

func (state *ArbosState) SetGasPool(val int64) {
	if state.gasPool == nil {
		state.gasPool = state.backingStorage.OpenStorageBackedInt64(util.UintToHash(uint64(gasPoolKey)))
	}
	state.gasPool.Set(val)
}

func (state *ArbosState) SmallGasPool() int64 {
	if state.smallGasPool == nil {
		state.smallGasPool = state.backingStorage.OpenStorageBackedInt64(util.UintToHash(uint64(smallGasPoolKey)))
	}
	return state.smallGasPool.Get()
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	if state.smallGasPool == nil {
		state.smallGasPool = state.backingStorage.OpenStorageBackedInt64(util.UintToHash(uint64(smallGasPoolKey)))
	}
	state.smallGasPool.Set(val)
}

func (state *ArbosState) GasPriceWei() *big.Int {
	if state.gasPriceWei == nil {
		state.gasPriceWei = state.backingStorage.GetByUint64(uint64(gasPriceKey)).Big()
	}
	return state.gasPriceWei
}

func (state *ArbosState) SetGasPriceWei(val *big.Int) {
	state.gasPriceWei = val
	state.backingStorage.SetByUint64(uint64(gasPriceKey), common.BigToHash(val))
}

func (state *ArbosState) MaxGasPriceWei() *big.Int {
	if state.maxGasPriceWei == nil {
		state.maxGasPriceWei = state.backingStorage.GetByUint64(uint64(maxPriceKey)).Big()
	}
	return state.maxGasPriceWei
}

func (state *ArbosState) SetMaxGasPriceWei(val *big.Int) {
	state.maxGasPriceWei = val
	state.backingStorage.SetByUint64(uint64(maxPriceKey), common.BigToHash(val))
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

func (state *ArbosState) ChainOwners() *addressSet.AddressSet {
	if state.chainOwners == nil {
		state.chainOwners = addressSet.OpenAddressSet(state.backingStorage.OpenSubStorage(chainOwnerSubspace))
	}
	return state.chainOwners
}

func (state *ArbosState) SendMerkleAccumulator() *merkleAccumulator.MerkleAccumulator {
	if state.sendMerkle == nil {
		state.sendMerkle = merkleAccumulator.OpenMerkleAccumulator(state.backingStorage.OpenSubStorage(sendMerkleSubspace))
	}
	return state.sendMerkle
}

func (state *ArbosState) LastTimestampSeen() uint64 {
	return state.timestamp.Get()
}

func (state *ArbosState) SetLastTimestampSeen(val uint64) {
	ts := state.timestamp.Get()
	if val < ts {
		panic("timestamp decreased")
	}
	if val > ts {
		delta := val - ts
		state.timestamp.Set(val)
		state.notifyGasPricerThatTimeElapsed(delta)
	}
}
