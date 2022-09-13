// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbosState

import (
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"

	"github.com/offchainlabs/nitro/arbos/blockhash"
	"github.com/offchainlabs/nitro/arbos/l2pricing"

	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/arbos/burn"

	"github.com/offchainlabs/nitro/arbos/addressTable"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"

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
	chainOwners       *addressSet.AddressSet
	sendMerkle        *merkleAccumulator.MerkleAccumulator
	blockhashes       *blockhash.Blockhashes
	chainId           storage.StorageBackedBigInt
	genesisBlockNum   storage.StorageBackedUint64
	infraFeeAccount   storage.StorageBackedAddress
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
		addressSet.OpenAddressSet(backingStorage.OpenSubStorage(chainOwnerSubspace)),
		merkleAccumulator.OpenMerkleAccumulator(backingStorage.OpenSubStorage(sendMerkleSubspace)),
		blockhash.OpenBlockhashes(backingStorage.OpenSubStorage(blockhashesSubspace)),
		backingStorage.OpenStorageBackedBigInt(uint64(chainIdOffset)),
		backingStorage.OpenStorageBackedUint64(uint64(genesisBlockNumOffset)),
		backingStorage.OpenStorageBackedAddress(uint64(infraFeeAccountOffset)),
		backingStorage,
		burner,
	}, nil
}

func OpenSystemArbosState(stateDB vm.StateDB, tracingInfo *util.TracingInfo, readOnly bool) (*ArbosState, error) {
	burner := burn.NewSystemBurner(tracingInfo, readOnly)
	state, err := OpenArbosState(stateDB, burner)
	burner.Restrict(err)
	return state, err
}

func OpenSystemArbosStateOrPanic(stateDB vm.StateDB, tracingInfo *util.TracingInfo, readOnly bool) *ArbosState {
	state, err := OpenSystemArbosState(stateDB, tracingInfo, readOnly)
	if err != nil {
		panic(err)
	}
	return state
}

// Create and initialize a memory-backed ArbOS state (for testing only)
func NewArbosMemoryBackedArbOSState() (*ArbosState, *state.StateDB) {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		log.Fatal("failed to init empty statedb", err)
	}
	burner := burn.NewSystemBurner(nil, false)
	state, err := InitializeArbosState(statedb, burner, params.ArbitrumDevTestChainConfig())
	if err != nil {
		log.Fatal("failed to open the ArbOS state", err)
	}
	return state, statedb
}

// Get the ArbOS version
func ArbOSVersion(stateDB vm.StateDB) uint64 {
	backingStorage := storage.NewGeth(stateDB, burn.NewSystemBurner(nil, false))
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
	networkFeeAccountOffset
	chainIdOffset
	genesisBlockNumOffset
	infraFeeAccountOffset
)

type ArbosStateSubspaceID []byte

var (
	l1PricingSubspace    ArbosStateSubspaceID = []byte{0}
	l2PricingSubspace    ArbosStateSubspaceID = []byte{1}
	retryablesSubspace   ArbosStateSubspaceID = []byte{2}
	addressTableSubspace ArbosStateSubspaceID = []byte{3}
	chainOwnerSubspace   ArbosStateSubspaceID = []byte{4}
	sendMerkleSubspace   ArbosStateSubspaceID = []byte{5}
	blockhashesSubspace  ArbosStateSubspaceID = []byte{6}
)

// Returns a list of precompiles that only appear in Arbitrum chains (i.e. ArbOS precompiles) at the genesis block
func getArbitrumOnlyPrecompiles(chainConfig *params.ChainConfig) []common.Address {
	rules := chainConfig.Rules(big.NewInt(0), false)
	arbPrecompiles := vm.ActivePrecompiles(rules)
	rules.IsArbitrum = false
	ethPrecompiles := vm.ActivePrecompiles(rules)

	ethPrecompilesSet := make(map[common.Address]bool)
	for _, addr := range ethPrecompiles {
		ethPrecompilesSet[addr] = true
	}

	var arbOnlyPrecompiles []common.Address
	for _, addr := range arbPrecompiles {
		if !ethPrecompilesSet[addr] {
			arbOnlyPrecompiles = append(arbOnlyPrecompiles, addr)
		}
	}
	return arbOnlyPrecompiles
}

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

	desiredArbosVersion := chainConfig.ArbitrumChainParams.InitialArbOSVersion
	if desiredArbosVersion == 0 {
		return nil, errors.New("cannot initialize to ArbOS version 0")
	}

	// Solidity requires call targets have code, but precompiles don't.
	// To work around this, we give precompiles fake code.
	for _, precompile := range getArbitrumOnlyPrecompiles(chainConfig) {
		stateDB.SetCode(precompile, []byte{byte(vm.INVALID)})
	}

	// may be the zero address
	initialChainOwner := chainConfig.ArbitrumChainParams.InitialChainOwner

	_ = sto.SetUint64ByUint64(uint64(versionOffset), 1) // initialize to version 1; upgrade at end of this func if needed
	_ = sto.SetUint64ByUint64(uint64(upgradeVersionOffset), 0)
	_ = sto.SetUint64ByUint64(uint64(upgradeTimestampOffset), 0)
	if desiredArbosVersion >= 2 {
		_ = sto.SetByUint64(uint64(networkFeeAccountOffset), util.AddressToHash(initialChainOwner))
	} else {
		_ = sto.SetByUint64(uint64(networkFeeAccountOffset), common.Hash{}) // the 0 address until an owner sets it
	}
	_ = sto.SetByUint64(uint64(chainIdOffset), common.BigToHash(chainConfig.ChainID))
	_ = sto.SetUint64ByUint64(uint64(genesisBlockNumOffset), chainConfig.ArbitrumChainParams.GenesisBlockNum)

	initialRewardsRecipient := l1pricing.BatchPosterAddress
	if desiredArbosVersion >= 2 {
		initialRewardsRecipient = initialChainOwner
	}
	_ = l1pricing.InitializeL1PricingState(sto.OpenSubStorage(l1PricingSubspace), initialRewardsRecipient)
	_ = l2pricing.InitializeL2PricingState(sto.OpenSubStorage(l2PricingSubspace))
	_ = retryables.InitializeRetryableState(sto.OpenSubStorage(retryablesSubspace))
	addressTable.Initialize(sto.OpenSubStorage(addressTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(sto.OpenSubStorage(sendMerkleSubspace))
	blockhash.InitializeBlockhashes(sto.OpenSubStorage(blockhashesSubspace))

	ownersStorage := sto.OpenSubStorage(chainOwnerSubspace)
	_ = addressSet.Initialize(ownersStorage)
	_ = addressSet.OpenAddressSet(ownersStorage).Add(initialChainOwner)

	aState, err := OpenArbosState(stateDB, burner)
	if err != nil {
		return nil, err
	}
	if desiredArbosVersion > 1 {
		aState.UpgradeArbosVersion(desiredArbosVersion, true)
	}
	return aState, err
}

func (state *ArbosState) UpgradeArbosVersionIfNecessary(currentTimestamp uint64) {
	upgradeTo, err := state.upgradeVersion.Get()
	state.Restrict(err)
	flagday, _ := state.upgradeTimestamp.Get()
	if state.arbosVersion < upgradeTo && currentTimestamp >= flagday {
		state.UpgradeArbosVersion(upgradeTo, false)
	}
}

func (state *ArbosState) UpgradeArbosVersion(upgradeTo uint64, firstTime bool) {
	for state.arbosVersion < upgradeTo {
		ensure := func(err error) {
			if err != nil {
				message := fmt.Sprintf(
					"Failed to upgrade ArbOS version %v to version %v: %v",
					state.arbosVersion, state.arbosVersion+1, err,
				)
				panic(message)
			}
		}

		switch state.arbosVersion {
		case 1:
			ensure(state.l1PricingState.SetLastSurplus(common.Big0, 1))
		case 2:
			ensure(state.l1PricingState.SetPerBatchGasCost(0))
			ensure(state.l1PricingState.SetAmortizedCostCapBips(math.MaxUint64))
		case 3:
			// no state changes needed
		case 4:
			// no state changes needed
		case 5:
			// no state changes needed
		case 6:
			// no state changes needed
		default:
			panic("Unable to perform requested ArbOS upgrade")
		}
		state.arbosVersion++
	}

	if firstTime && upgradeTo >= 6 {
		state.Restrict(state.l1PricingState.SetPerBatchGasCost(l1pricing.InitialPerBatchGasCostV6))
		state.Restrict(state.l1PricingState.SetEquilibrationUnits(l1pricing.InitialEquilibrationUnitsV6))
		state.Restrict(state.l2PricingState.SetSpeedLimitPerSecond(l2pricing.InitialSpeedLimitPerSecondV6))
		state.Restrict(state.l2PricingState.SetMaxPerBlockGasLimit(l2pricing.InitialPerBlockGasLimitV6))
	}

	state.Restrict(state.backingStorage.SetUint64ByUint64(uint64(versionOffset), state.arbosVersion))
}

func (state *ArbosState) ScheduleArbOSUpgrade(newVersion uint64, timestamp uint64) error {
	err := state.upgradeVersion.Set(newVersion)
	if err != nil {
		return err
	}
	return state.upgradeTimestamp.Set(timestamp)
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

func (state *ArbosState) NetworkFeeAccount() (common.Address, error) {
	return state.networkFeeAccount.Get()
}

func (state *ArbosState) SetNetworkFeeAccount(account common.Address) error {
	return state.networkFeeAccount.Set(account)
}

func (state *ArbosState) InfraFeeAccount() (common.Address, error) {
	return state.infraFeeAccount.Get()
}

func (state *ArbosState) SetInfraFeeAccount(account common.Address) error {
	return state.infraFeeAccount.Set(account)
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

func (state *ArbosState) GenesisBlockNum() (uint64, error) {
	return state.genesisBlockNum.Get()
}
