//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"errors"
	"log"
	"math/big"

	"github.com/offchainlabs/arbstate/arbos/blockhash"
	"github.com/offchainlabs/arbstate/arbos/l2pricing"

	"github.com/offchainlabs/arbstate/arbos/addressSet"
	"github.com/offchainlabs/arbstate/arbos/blsTable"
	"github.com/offchainlabs/arbstate/arbos/burn"

	"github.com/offchainlabs/arbstate/arbos/addressTable"
	"github.com/offchainlabs/arbstate/arbos/l1pricing"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// ArbosState contains ArbOS-related state. It is backed by ArbOS's storage in the persistent stateDB.
// Modifications to the ArbosState are written through to the underlying StateDB so that the StateDB always
// has the definitive state, stored persistently. (Note that some tests use memory-backed StateDB's that aren't
// persisted beyond the end of the test.)

type ArbosState struct {
	arbosVersion      uint64                      // version of the ArbOS storage format and semantics
	upgradeVersion    storage.StorageBackedUint64 // version we're planning to upgrade to, or 0 if not planning to upgrade
	upgradeTimestamp  storage.StorageBackedUint64 // when to do the planned upgrade
	networkFeeAccount storage.StorageBackedAddress
	l1PricingState    *l1pricing.L1PricingState
	l2PricingState    *l2pricing.L2PricingState
	retryableState    *retryables.RetryableState
	addressTable      *addressTable.AddressTable
	blsTable          *blsTable.BLSTable
	chainOwners       *addressSet.AddressSet
	sendMerkle        *merkleAccumulator.MerkleAccumulator
	timestamp         storage.StorageBackedUint64
	blockhashes       *blockhash.Blockhashes
	chainId           storage.StorageBackedBigInt
	backingStorage    *storage.Storage
	Burner            burn.Burner
}

var ErrUninitializedArbOS = errors.New("ArbOS uninitialized")
var ErrAlreadyInitialized = errors.New("ArbOS is already initialized")

func OpenArbosState(stateDB vm.StateDB, burner burn.Burner) (*ArbosState, error) {
	backingStorage := storage.NewGeth(stateDB, burner)
	arbosVersion, err := backingStorage.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		return nil, err
	}
	if arbosVersion == 0 {
		return nil, ErrUninitializedArbOS
	}
	return &ArbosState{
		arbosVersion,
		backingStorage.OpenStorageBackedUint64(uint64(upgradeVersionOffset)),
		backingStorage.OpenStorageBackedUint64(uint64(upgradeTimestampOffset)),
		backingStorage.OpenStorageBackedAddress(uint64(networkFeeAccountOffset)),
		l1pricing.OpenL1PricingState(backingStorage.OpenSubStorage(l1PricingSubspace)),
		l2pricing.OpenL2PricingState(backingStorage.OpenSubStorage(l2PricingSubspace)),
		retryables.OpenRetryableState(backingStorage.OpenSubStorage(retryablesSubspace), stateDB),
		addressTable.Open(backingStorage.OpenSubStorage(addressTableSubspace)),
		blsTable.Open(backingStorage.OpenSubStorage(blsTableSubspace)),
		addressSet.OpenAddressSet(backingStorage.OpenSubStorage(chainOwnerSubspace)),
		merkleAccumulator.OpenMerkleAccumulator(backingStorage.OpenSubStorage(sendMerkleSubspace)),
		backingStorage.OpenStorageBackedUint64(uint64(timestampOffset)),
		blockhash.OpenBlockhashes(backingStorage.OpenSubStorage(blockhashesSubspace)),
		backingStorage.OpenStorageBackedBigInt(uint64(chainIdOffset)),
		backingStorage,
		burner,
	}, nil
}

func OpenSystemArbosState(stateDB vm.StateDB, readOnly bool) (*ArbosState, error) {
	burner := burn.NewSystemBurner(readOnly)
	state, err := OpenArbosState(stateDB, burner)
	burner.Restrict(err)
	return state, err
}

// Create and initialize a memory-backed ArbOS state (for testing only)
func NewArbosMemoryBackedArbOSState() (*ArbosState, *state.StateDB) {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		log.Fatal("failed to init empty statedb", err)
	}
	burner := burn.NewSystemBurner(false)
	state, err := InitializeArbosState(statedb, burner, params.ArbitrumDevTestChainConfig())
	if err != nil {
		log.Fatal("failed to open the ArbOS state", err)
	}
	return state, statedb
}

// Get the ArbOS version
func ArbOSVersion(stateDB vm.StateDB) uint64 {
	backingStorage := storage.NewGeth(stateDB, burn.NewSystemBurner(false))
	arbosVersion, err := backingStorage.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		log.Fatal("faled to get the ArbOS version", err)
	}
	return arbosVersion
}

type ArbosStateOffset uint64

const (
	versionOffset ArbosStateOffset = iota
	upgradeVersionOffset
	upgradeTimestampOffset
	timestampOffset
	networkFeeAccountOffset
	chainIdOffset
)

type ArbosStateSubspaceID []byte

var (
	l1PricingSubspace    ArbosStateSubspaceID = []byte{0}
	l2PricingSubspace    ArbosStateSubspaceID = []byte{1}
	retryablesSubspace   ArbosStateSubspaceID = []byte{2}
	addressTableSubspace ArbosStateSubspaceID = []byte{3}
	blsTableSubspace     ArbosStateSubspaceID = []byte{4}
	chainOwnerSubspace   ArbosStateSubspaceID = []byte{5}
	sendMerkleSubspace   ArbosStateSubspaceID = []byte{6}
	blockhashesSubspace  ArbosStateSubspaceID = []byte{7}
)

// During early development we sometimes change the storage format of version 1, for convenience. But as soon as we
// start running long-lived chains, every change to the storage format will require defining a new version and
// providing upgrade code.

func InitializeArbosState(stateDB vm.StateDB, burner burn.Burner, chainConfig *params.ChainConfig) (*ArbosState, error) {
	sto := storage.NewGeth(stateDB, burner)
	arbosVersion, err := sto.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		return nil, err
	}
	if arbosVersion != 0 {
		return nil, ErrAlreadyInitialized
	}

	_ = sto.SetUint64ByUint64(uint64(versionOffset), 1)
	_ = sto.SetUint64ByUint64(uint64(upgradeVersionOffset), 0)
	_ = sto.SetUint64ByUint64(uint64(upgradeTimestampOffset), 0)
	_ = sto.SetUint64ByUint64(uint64(timestampOffset), 0)
	_ = sto.SetUint64ByUint64(uint64(networkFeeAccountOffset), 0) // the 0 address until an owner sets it
	_ = sto.SetByUint64(uint64(chainIdOffset), common.BigToHash(chainConfig.ChainID))
	_ = l1pricing.InitializeL1PricingState(sto.OpenSubStorage(l1PricingSubspace))
	_ = l2pricing.InitializeL2PricingState(sto.OpenSubStorage(l2PricingSubspace))
	_ = retryables.InitializeRetryableState(sto.OpenSubStorage(retryablesSubspace))
	addressTable.Initialize(sto.OpenSubStorage(addressTableSubspace))
	_ = blsTable.InitializeBLSTable(sto.OpenSubStorage(blsTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(sto.OpenSubStorage(sendMerkleSubspace))
	blockhash.InitializeBlockhashes(sto.OpenSubStorage(blockhashesSubspace))

	// by default, the remapped zero address is the initial chain owner
	initialChainOwner := util.RemapL1Address(common.Address{})
	if chainConfig.ArbitrumChainParams.InitialChainOwner != (common.Address{}) {
		initialChainOwner = chainConfig.ArbitrumChainParams.InitialChainOwner
	}
	ownersStorage := sto.OpenSubStorage(chainOwnerSubspace)
	_ = addressSet.Initialize(ownersStorage)
	_ = addressSet.OpenAddressSet(ownersStorage).Add(initialChainOwner)

	_ = sto.SetUint64ByUint64(uint64(versionOffset), 1)

	return OpenArbosState(stateDB, burner)
}

func (state *ArbosState) UpgradeArbosVersionIfNecessary(currentTimestamp uint64) {
	upgradeTo, err := state.upgradeVersion.Get()
	state.Restrict(err)
	flagday, _ := state.upgradeTimestamp.Get()
	if upgradeTo > state.arbosVersion && currentTimestamp >= flagday {
		// code to upgrade to future versions will be put here
		// for now, no upgrades are enabled
		panic("Unable to perform requested ArbOS upgrade")
	}
}

func (state *ArbosState) BackingStorage() *storage.Storage {
	return state.backingStorage
}

func (state *ArbosState) Restrict(err error) {
	state.Burner.Restrict(err)
}

func (state *ArbosState) FormatVersion() uint64 {
	return state.arbosVersion
}

func (state *ArbosState) SetFormatVersion(val uint64) {
	state.arbosVersion = val
	state.Restrict(state.backingStorage.SetUint64ByUint64(uint64(versionOffset), val))
}

func (state *ArbosState) RetryableState() *retryables.RetryableState {
	return state.retryableState
}

func (state *ArbosState) L1PricingState() *l1pricing.L1PricingState {
	return state.l1PricingState
}

func (state *ArbosState) L2PricingState() *l2pricing.L2PricingState {
	return state.l2PricingState
}

func (state *ArbosState) AddressTable() *addressTable.AddressTable {
	return state.addressTable
}

func (state *ArbosState) BLSTable() *blsTable.BLSTable {
	return state.blsTable
}

func (state *ArbosState) ChainOwners() *addressSet.AddressSet {
	return state.chainOwners
}

func (state *ArbosState) SendMerkleAccumulator() *merkleAccumulator.MerkleAccumulator {
	if state.sendMerkle == nil {
		state.sendMerkle = merkleAccumulator.OpenMerkleAccumulator(state.backingStorage.OpenSubStorage(sendMerkleSubspace))
	}
	return state.sendMerkle
}

func (state *ArbosState) Blockhashes() *blockhash.Blockhashes {
	return state.blockhashes
}

func (state *ArbosState) LastTimestampSeen() (uint64, error) {
	return state.timestamp.Get()
}

func (state *ArbosState) SetLastTimestampSeen(val uint64) {
	ts, err := state.timestamp.Get()
	state.Restrict(err)
	if val < ts {
		panic("timestamp decreased")
	}
	if val > ts {
		delta := val - ts
		state.Restrict(state.timestamp.Set(val))
		state.l2PricingState.NotifyGasPricerThatTimeElapsed(delta)
	}
}

func (state *ArbosState) NetworkFeeAccount() (common.Address, error) {
	return state.networkFeeAccount.Get()
}

func (state *ArbosState) SetNetworkFeeAccount(account common.Address) error {
	return state.networkFeeAccount.Set(account)
}

func (state *ArbosState) Keccak(data ...[]byte) ([]byte, error) {
	return state.backingStorage.Keccak(data...)
}

func (state *ArbosState) KeccakHash(data ...[]byte) (common.Hash, error) {
	return state.backingStorage.KeccakHash(data...)
}

func (state *ArbosState) ChainId() (*big.Int, error) {
	return state.chainId.Get()
}
