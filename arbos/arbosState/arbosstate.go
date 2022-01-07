//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"github.com/offchainlabs/arbstate/arbos/addressSet"
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

// ArbosState contains ArbOS-related state. It is backed by ArbOS's storage in the persistent stateDB.
// Modifications to the ArbosState are written through to the underlying StateDB so that the StateDB always
// has the definitive state, stored persistently. (Note that some tests use memory-backed StateDB's that aren't
// persisted beyond the end of the test.)

type ArbosState struct {
	formatVersion  uint64
	gasPool        storage.StorageBackedInt64
	smallGasPool   storage.StorageBackedInt64
	gasPriceWei    storage.StorageBackedBigInt
	maxGasPriceWei storage.StorageBackedBigInt // the maximum price ArbOS can set without breaking geth
	l1PricingState *l1pricing.L1PricingState
	retryableState *retryables.RetryableState
	addressTable   *addressTable.AddressTable
	chainOwners    *addressSet.AddressSet
	sendMerkle     *merkleAccumulator.MerkleAccumulator
	timestamp      storage.StorageBackedUint64
	backingStorage *storage.Storage
}

func OpenArbosState(stateDB vm.StateDB) *ArbosState {
	backingStorage := storage.NewGeth(stateDB)

	for tryStorageUpgrade(backingStorage) {
	}

	return &ArbosState{
		backingStorage.GetByUint64(uint64(versionOffset)).Big().Uint64(),
		backingStorage.OpenStorageBackedInt64(uint64(gasPoolOffset)),
		backingStorage.OpenStorageBackedInt64(uint64(smallGasPoolOffset)),
		backingStorage.OpenStorageBackedBigInt(uint64(gasPriceOffset)),
		backingStorage.OpenStorageBackedBigInt(uint64(maxPriceOffset)),
		l1pricing.OpenL1PricingState(backingStorage.OpenSubStorage(l1PricingSubspace)),
		retryables.OpenRetryableState(backingStorage.OpenSubStorage(retryablesSubspace)),
		addressTable.Open(backingStorage.OpenSubStorage(addressTableSubspace)),
		addressSet.OpenAddressSet(backingStorage.OpenSubStorage(chainOwnerSubspace)),
		merkleAccumulator.OpenMerkleAccumulator(backingStorage.OpenSubStorage(sendMerkleSubspace)),
		backingStorage.OpenStorageBackedUint64(uint64(timestampOffset)),
		backingStorage,
	}
}

// See if we should upgrade the storage format. The format version is stored at location zero of our backing storage.
//
// If we know how to upgrade from the observed storage version, we do so. We return true iff we upgraded.
// It might be that yet another upgrade is possible, so if this returns true, the caller should call this again.
//
// If the latest version is N, then there must always be code to upgrade from every version less than N. Each
// such upgrade must increase the version number by at least 1, so that a sequence of upgrades will converge on
// the latest version.
//
// Because uninitialized storage space always returns 0 for all reads, we define format version 0 to mean that the
// storage space is uninitialized. This uninitialized version 0 will cause an upgrade to version 1, which will
// initialize the storage.
//
// During early development we sometimes change the definition of version 1, for convenience. But as soon as we
// start running long-lived chains, every change to the format will require defining a new version and providing
// upgrade code.
func tryStorageUpgrade(backingStorage *storage.Storage) bool {
	formatVersion := backingStorage.GetUint64ByUint64(uint64(versionOffset))
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
	versionOffset ArbosStateOffset = iota
	gasPoolOffset
	smallGasPoolOffset
	gasPriceOffset
	maxPriceOffset
	timestampOffset
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
	backingStorage.SetUint64ByUint64(uint64(versionOffset), 1)
	backingStorage.SetUint64ByUint64(uint64(gasPoolOffset), GasPoolMax)
	backingStorage.SetUint64ByUint64(uint64(smallGasPoolOffset), SmallGasPoolMax)
	backingStorage.SetUint64ByUint64(uint64(gasPriceOffset), InitialGasPriceWei)
	backingStorage.SetUint64ByUint64(uint64(maxPriceOffset), 2*InitialGasPriceWei)
	backingStorage.SetUint64ByUint64(uint64(timestampOffset), 0)
	l1pricing.InitializeL1PricingState(backingStorage.OpenSubStorage(l1PricingSubspace))
	retryables.InitializeRetryableState(backingStorage.OpenSubStorage(retryablesSubspace))
	addressTable.Initialize(backingStorage.OpenSubStorage(addressTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(backingStorage.OpenSubStorage(sendMerkleSubspace))

	// the zero address is the initial chain owner
	ZeroAddressL2 := util.RemapL1Address(common.Address{})
	ownersStorage := backingStorage.OpenSubStorage(chainOwnerSubspace)
	addressSet.Initialize(ownersStorage)
	addressSet.OpenAddressSet(ownersStorage).Add(ZeroAddressL2)

	backingStorage.SetUint64ByUint64(uint64(versionOffset), 1)
}

func (state *ArbosState) BackingStorage() *storage.Storage {
	return state.backingStorage
}

func (state *ArbosState) FormatVersion() uint64 {
	return state.formatVersion
}

func (state *ArbosState) SetFormatVersion(val uint64) {
	state.formatVersion = val
	state.backingStorage.SetUint64ByUint64(uint64(versionOffset), val)
}

func (state *ArbosState) GasPool() int64 {
	return state.gasPool.Get()
}

func (state *ArbosState) SetGasPool(val int64) {
	state.gasPool.Set(val)
}

func (state *ArbosState) SmallGasPool() int64 {
	return state.smallGasPool.Get()
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	state.smallGasPool.Set(val)
}

func (state *ArbosState) GasPriceWei() *big.Int {
	return state.gasPriceWei.Get()
}

func (state *ArbosState) SetGasPriceWei(val *big.Int) {
	state.gasPriceWei.Set(val)
}

func (state *ArbosState) MaxGasPriceWei() *big.Int { // the max gas price ArbOS can set without breaking geth
	return state.maxGasPriceWei.Get()
}

func (state *ArbosState) SetMaxGasPriceWei(val *big.Int) {
	state.maxGasPriceWei.Set(val)
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
		state.NotifyGasPricerThatTimeElapsed(delta)
	}
}
