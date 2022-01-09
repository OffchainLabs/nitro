//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"math/big"

	"github.com/offchainlabs/arbstate/arbos/addressSet"
	"github.com/offchainlabs/arbstate/arbos/burn"

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
	arbosVersion     uint64                      // version of the ArbOS storage format and semantics
	upgradeVersion   storage.StorageBackedUint64 // version we're planning to upgrade to, or 0 if not planning to upgrade
	upgradeTimestamp storage.StorageBackedUint64 // when to do the planned upgrade
	gasPool          storage.StorageBackedInt64
	smallGasPool     storage.StorageBackedInt64
	gasPriceWei      storage.StorageBackedBigInt
	maxGasPriceWei   storage.StorageBackedBigInt // the maximum price ArbOS can set without breaking geth
	l1PricingState   *l1pricing.L1PricingState
	retryableState   *retryables.RetryableState
	addressTable     *addressTable.AddressTable
	chainOwners      *addressSet.AddressSet
	sendMerkle       *merkleAccumulator.MerkleAccumulator
	timestamp        storage.StorageBackedUint64
	backingStorage   *storage.Storage
	Burner           burn.Burner
}

func OpenArbosState(stateDB vm.StateDB, burner burn.Burner) *ArbosState {
	backingStorage := storage.NewGeth(stateDB, burner)
	arbosVersion := backingStorage.GetUint64ByUint64(uint64(versionOffset))
	if arbosVersion == 0 {
		// we found a zero at storage location 0, so storage hasn't been initialized yet
		initializeStorage(backingStorage)
	}
	return &ArbosState{
		arbosVersion,
		backingStorage.OpenStorageBackedUint64(uint64(upgradeVersionOffset)),
		backingStorage.OpenStorageBackedUint64(uint64(upgradeTimestampOffset)),
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
		burner,
	}
}

type ArbosStateOffset uint64

const (
	versionOffset ArbosStateOffset = iota
	upgradeVersionOffset
	upgradeTimestampOffset
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

// During early development we sometimes change the storage format of version 1, for convenience. But as soon as we
// start running long-lived chains, every change to the storage format will require defining a new version and
// providing upgrade code.
func initializeStorage(backingStorage *storage.Storage) {
	backingStorage.SetUint64ByUint64(uint64(versionOffset), 1)
	backingStorage.SetUint64ByUint64(uint64(upgradeVersionOffset), 0)
	backingStorage.SetUint64ByUint64(uint64(upgradeTimestampOffset), 0)
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

func (state *ArbosState) UpgradeArbosVersionIfNecessary(currentTimestamp uint64) {
	upgradeTo := state.upgradeVersion.Get()
	if upgradeTo > state.arbosVersion && currentTimestamp >= state.upgradeTimestamp.Get() {
		// code to upgrade to future versions will be put here
		// for now, no upgrades are enabled
		panic("Unable to perform requested ArbOS upgrade")
	}
}

func (state *ArbosState) BackingStorage() *storage.Storage {
	return state.backingStorage
}

func (state *ArbosState) FormatVersion() uint64 {
	return state.arbosVersion
}

func (state *ArbosState) SetFormatVersion(val uint64) {
	state.arbosVersion = val
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
