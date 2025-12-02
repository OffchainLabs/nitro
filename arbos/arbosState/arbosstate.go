// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbosState

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/arbos/addressTable"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/blockhash"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/features"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

// ArbosState contains ArbOS-related state. It is backed by ArbOS's storage in the persistent stateDB.
// Modifications to the ArbosState are written through to the underlying StateDB so that the StateDB always
// has the definitive state, stored persistently. (Note that some tests use memory-backed StateDB's that aren't
// persisted beyond the end of the test.)

type ArbosState struct {
	arbosVersion           uint64                      // version of the ArbOS storage format and semantics
	upgradeVersion         storage.StorageBackedUint64 // version we're planning to upgrade to, or 0 if not planning to upgrade
	upgradeTimestamp       storage.StorageBackedUint64 // when to do the planned upgrade
	networkFeeAccount      storage.StorageBackedAddress
	l1PricingState         *l1pricing.L1PricingState
	l2PricingState         *l2pricing.L2PricingState
	retryableState         *retryables.RetryableState
	addressTable           *addressTable.AddressTable
	chainOwners            *addressSet.AddressSet
	nativeTokenOwners      *addressSet.AddressSet
	sendMerkle             *merkleAccumulator.MerkleAccumulator
	programs               *programs.Programs
	features               *features.Features
	blockhashes            *blockhash.Blockhashes
	chainId                storage.StorageBackedBigInt
	chainConfig            storage.StorageBackedBytes
	genesisBlockNum        storage.StorageBackedUint64
	infraFeeAccount        storage.StorageBackedAddress
	brotliCompressionLevel storage.StorageBackedUint64 // brotli compression level used for pricing
	nativeTokenEnabledTime storage.StorageBackedUint64
	backingStorage         *storage.Storage
	Burner                 burn.Burner
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
		arbosVersion:           arbosVersion,
		upgradeVersion:         backingStorage.OpenStorageBackedUint64(uint64(upgradeVersionOffset)),
		upgradeTimestamp:       backingStorage.OpenStorageBackedUint64(uint64(upgradeTimestampOffset)),
		networkFeeAccount:      backingStorage.OpenStorageBackedAddress(uint64(networkFeeAccountOffset)),
		l1PricingState:         l1pricing.OpenL1PricingState(backingStorage.OpenCachedSubStorage(l1PricingSubspace), arbosVersion),
		l2PricingState:         l2pricing.OpenL2PricingState(backingStorage.OpenCachedSubStorage(l2PricingSubspace), arbosVersion),
		retryableState:         retryables.OpenRetryableState(backingStorage.OpenCachedSubStorage(retryablesSubspace), stateDB),
		addressTable:           addressTable.Open(backingStorage.OpenCachedSubStorage(addressTableSubspace)),
		chainOwners:            addressSet.OpenAddressSet(backingStorage.OpenCachedSubStorage(chainOwnerSubspace)),
		nativeTokenOwners:      addressSet.OpenAddressSet(backingStorage.OpenCachedSubStorage(nativeTokenOwnerSubspace)),
		sendMerkle:             merkleAccumulator.OpenMerkleAccumulator(backingStorage.OpenCachedSubStorage(sendMerkleSubspace)),
		programs:               programs.Open(arbosVersion, backingStorage.OpenSubStorage(programsSubspace)),
		features:               features.Open(backingStorage.OpenSubStorage(featuresSubspace)),
		blockhashes:            blockhash.OpenBlockhashes(backingStorage.OpenCachedSubStorage(blockhashesSubspace)),
		chainId:                backingStorage.OpenStorageBackedBigInt(uint64(chainIdOffset)),
		chainConfig:            backingStorage.OpenStorageBackedBytes(chainConfigSubspace),
		genesisBlockNum:        backingStorage.OpenStorageBackedUint64(uint64(genesisBlockNumOffset)),
		infraFeeAccount:        backingStorage.OpenStorageBackedAddress(uint64(infraFeeAccountOffset)),
		brotliCompressionLevel: backingStorage.OpenStorageBackedUint64(uint64(brotliCompressionLevelOffset)),
		nativeTokenEnabledTime: backingStorage.OpenStorageBackedUint64(uint64(nativeTokenEnabledFromTimeOffset)),
		backingStorage:         backingStorage,
		Burner:                 burner,
	}, nil
}

func OpenSystemArbosState(stateDB vm.StateDB, tracingInfo *util.TracingInfo, readOnly bool) (*ArbosState, error) {
	burner := burn.NewSystemBurner(tracingInfo, readOnly)
	newState, err := OpenArbosState(stateDB, burner)
	burner.Restrict(err)
	return newState, err
}

func OpenSystemArbosStateOrPanic(stateDB vm.StateDB, tracingInfo *util.TracingInfo, readOnly bool) *ArbosState {
	newState, err := OpenSystemArbosState(stateDB, tracingInfo, readOnly)
	if err != nil {
		panic(err)
	}
	return newState
}

// NewArbosMemoryBackedArbOSState creates and initializes a memory-backed ArbOS state (for testing only)
func NewArbosMemoryBackedArbOSState() (*ArbosState, *state.StateDB) {
	return NewArbosMemoryBackedArbOSStateWithConfig(chaininfo.ArbitrumDevTestChainConfig())
}

// NewArbosMemoryBackedArbOSStateWithConfig creates and initializes a memory-backed ArbOS state with a given config (for testing only)
func NewArbosMemoryBackedArbOSStateWithConfig(chainConfig *params.ChainConfig) (*ArbosState, *state.StateDB) {
	if chainConfig.ArbitrumChainParams.InitialArbOSVersion == 0 {
		chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	}
	raw := rawdb.NewMemoryDatabase()
	trieConfig := &triedb.Config{Preimages: false, PathDB: pathdb.Defaults}
	if env.GetTestStateScheme() == rawdb.HashScheme {
		trieConfig = &triedb.Config{Preimages: false, HashDB: hashdb.Defaults}
	}
	db := state.NewDatabase(triedb.NewDatabase(raw, trieConfig), nil)
	statedb, err := state.New(common.Hash{}, db)
	if err != nil {
		panic("failed to init empty statedb: " + err.Error())
	}
	burner := burn.NewSystemBurner(nil, false)
	// #nosec G115
	newState, err := InitializeArbosState(statedb, burner, chainConfig, nil, arbostypes.TestInitMessage)
	if err != nil {
		panic("failed to open the ArbOS state: " + err.Error())
	}
	return newState, statedb
}

// ArbOSVersion returns the ArbOS version
func ArbOSVersion(stateDB vm.StateDB) uint64 {
	backingStorage := storage.NewGeth(stateDB, burn.NewSystemBurner(nil, false))
	arbosVersion, err := backingStorage.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		panic("failed to get the ArbOS version: " + err.Error())
	}
	return arbosVersion
}

type Offset uint64

const (
	versionOffset Offset = iota
	upgradeVersionOffset
	upgradeTimestampOffset
	networkFeeAccountOffset
	chainIdOffset
	genesisBlockNumOffset
	infraFeeAccountOffset
	brotliCompressionLevelOffset
	nativeTokenEnabledFromTimeOffset
)

type SubspaceID []byte

var (
	l1PricingSubspace        SubspaceID = []byte{0}
	l2PricingSubspace        SubspaceID = []byte{1}
	retryablesSubspace       SubspaceID = []byte{2}
	addressTableSubspace     SubspaceID = []byte{3}
	chainOwnerSubspace       SubspaceID = []byte{4}
	sendMerkleSubspace       SubspaceID = []byte{5}
	blockhashesSubspace      SubspaceID = []byte{6}
	chainConfigSubspace      SubspaceID = []byte{7}
	programsSubspace         SubspaceID = []byte{8}
	featuresSubspace         SubspaceID = []byte{9}
	nativeTokenOwnerSubspace SubspaceID = []byte{10}
)

var PrecompileMinArbOSVersions = make(map[common.Address]uint64)

func InitializeArbosState(stateDB vm.StateDB, burner burn.Burner, chainConfig *params.ChainConfig, genesisArbOSInit *params.ArbOSInit, initMessage *arbostypes.ParsedInitMessage) (*ArbosState, error) {
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
	for addr, version := range PrecompileMinArbOSVersions {
		if version == 0 {
			stateDB.SetCode(addr, []byte{byte(vm.INVALID)}, tracing.CodeChangeUnspecified)
		}
	}

	// may be the zero address
	initialChainOwner := chainConfig.ArbitrumChainParams.InitialChainOwner

	nativeTokenEnabledFromTime := uint64(0)
	if genesisArbOSInit != nil && genesisArbOSInit.NativeTokenSupplyManagementEnabled {
		// Since we're initializing the state from the beginning with the
		// feature enabled, we set the enabled time to 1 (which will always be)
		// lower than the timestamp of the first block of the chain.
		nativeTokenEnabledFromTime = uint64(1)
	}
	err = sto.SetUint64ByUint64(uint64(nativeTokenEnabledFromTimeOffset), nativeTokenEnabledFromTime)
	if err != nil {
		return nil, err
	}

	err = sto.SetUint64ByUint64(uint64(versionOffset), 1) // initialize to version 1; upgrade at end of this func if needed
	if err != nil {
		return nil, err
	}
	err = sto.SetUint64ByUint64(uint64(upgradeVersionOffset), 0)
	if err != nil {
		return nil, err
	}
	err = sto.SetUint64ByUint64(uint64(upgradeTimestampOffset), 0)
	if err != nil {
		return nil, err
	}

	if desiredArbosVersion >= params.ArbosVersion_2 {
		err = sto.SetByUint64(uint64(networkFeeAccountOffset), util.AddressToHash(initialChainOwner))
		if err != nil {
			return nil, err
		}
	} else {
		err = sto.SetByUint64(uint64(networkFeeAccountOffset), common.Hash{}) // the 0 address until an owner sets it
		if err != nil {
			return nil, err
		}
	}
	err = sto.SetByUint64(uint64(chainIdOffset), common.BigToHash(chainConfig.ChainID))
	if err != nil {
		return nil, err
	}

	chainConfigStorage := sto.OpenStorageBackedBytes(chainConfigSubspace)
	err = chainConfigStorage.Set(initMessage.SerializedChainConfig)
	if err != nil {
		return nil, err
	}
	err = sto.SetUint64ByUint64(uint64(genesisBlockNumOffset), chainConfig.ArbitrumChainParams.GenesisBlockNum)
	if err != nil {
		return nil, err
	}
	err = sto.SetUint64ByUint64(uint64(brotliCompressionLevelOffset), 0) // default brotliCompressionLevel for fast compression is 0
	if err != nil {
		return nil, err
	}

	initialRewardsRecipient := l1pricing.BatchPosterAddress
	if desiredArbosVersion >= params.ArbosVersion_2 {
		initialRewardsRecipient = initialChainOwner
	}
	err = l1pricing.InitializeL1PricingState(sto.OpenCachedSubStorage(l1PricingSubspace), initialRewardsRecipient, initMessage.InitialL1BaseFee)
	if err != nil {
		return nil, err
	}
	err = l2pricing.InitializeL2PricingState(sto.OpenCachedSubStorage(l2PricingSubspace))
	if err != nil {
		return nil, err
	}
	err = retryables.InitializeRetryableState(sto.OpenCachedSubStorage(retryablesSubspace))
	if err != nil {
		return nil, err
	}
	addressTable.Initialize(sto.OpenCachedSubStorage(addressTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(sto.OpenCachedSubStorage(sendMerkleSubspace))
	blockhash.InitializeBlockhashes(sto.OpenCachedSubStorage(blockhashesSubspace))

	ownersStorage := sto.OpenCachedSubStorage(chainOwnerSubspace)
	err = addressSet.Initialize(ownersStorage)
	if err != nil {
		return nil, err
	}
	err = addressSet.OpenAddressSet(ownersStorage).Add(initialChainOwner)
	if err != nil {
		return nil, err
	}

	nativeTokenOwnersStorage := sto.OpenCachedSubStorage(nativeTokenOwnerSubspace)
	err = addressSet.Initialize(nativeTokenOwnersStorage)
	if err != nil {
		return nil, err
	}

	aState, err := OpenArbosState(stateDB, burner)
	if err != nil {
		return nil, err
	}
	if desiredArbosVersion > 1 {
		err = aState.UpgradeArbosVersion(desiredArbosVersion, true, stateDB, chainConfig)
		if err != nil {
			return nil, err
		}
	}
	return aState, nil
}

func (state *ArbosState) UpgradeArbosVersionIfNecessary(
	currentTimestamp uint64, stateDB vm.StateDB, chainConfig *params.ChainConfig,
) error {
	upgradeTo, err := state.upgradeVersion.Get()
	state.Restrict(err)
	flagday, _ := state.upgradeTimestamp.Get()
	if state.arbosVersion < upgradeTo && currentTimestamp >= flagday {
		return state.UpgradeArbosVersion(upgradeTo, false, stateDB, chainConfig)
	}
	return nil
}

var ErrFatalNodeOutOfDate = errors.New("please upgrade to the latest version of the node software")

func (state *ArbosState) UpgradeArbosVersion(
	upgradeTo uint64, firstTime bool, stateDB vm.StateDB, chainConfig *params.ChainConfig,
) error {
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

		nextArbosVersion := state.arbosVersion + 1
		switch nextArbosVersion {
		case params.ArbosVersion_2:
			ensure(state.l1PricingState.SetLastSurplus(common.Big0, 1))
		case params.ArbosVersion_3:
			ensure(state.l1PricingState.SetPerBatchGasCost(0))
			ensure(state.l1PricingState.SetAmortizedCostCapBips(math.MaxUint64))
		case params.ArbosVersion_4:
			// no state changes needed
		case params.ArbosVersion_5:
			// no state changes needed
		case params.ArbosVersion_6:
			// no state changes needed
		case params.ArbosVersion_7:
			// no state changes needed
		case params.ArbosVersion_8:
			// no state changes needed
		case params.ArbosVersion_9:
			// no state changes needed
		case params.ArbosVersion_10:
			ensure(state.l1PricingState.SetL1FeesAvailable(stateDB.GetBalance(
				l1pricing.L1PricerFundsPoolAddress,
			).ToBig()))

		case params.ArbosVersion_11:
			// Update the PerBatchGasCost to a more accurate value compared to the old v6 default.
			ensure(state.l1PricingState.SetPerBatchGasCost(l1pricing.InitialPerBatchGasCostV12))

			// We had mistakenly initialized AmortizedCostCapBips to math.MaxUint64 in older versions,
			// but the correct value to disable the amortization cap is 0.
			oldAmortizationCap, err := state.l1PricingState.AmortizedCostCapBips()
			ensure(err)
			if oldAmortizationCap == math.MaxUint64 {
				ensure(state.l1PricingState.SetAmortizedCostCapBips(0))
			}

			// Clear chainOwners list to allow rectification of the mapping.
			if !firstTime {
				ensure(state.chainOwners.ClearList())
			}

		case 12, 13, 14, 15, 16, 17, 18, 19:
			// these versions are left to Orbit chains for custom upgrades.

		case params.ArbosVersion_20:
			// Update Brotli compression level for fast compression from 0 to 1
			ensure(state.SetBrotliCompressionLevel(1))

		case 21, 22, 23, 24, 25, 26, 27, 28, 29:
			// these versions are left to Orbit chains for custom upgrades.

		case params.ArbosVersion_30:
			programs.Initialize(nextArbosVersion, state.backingStorage.OpenSubStorage(programsSubspace))

		case params.ArbosVersion_31:
			params, err := state.Programs().Params()
			ensure(err)
			ensure(params.UpgradeToVersion(2))
			ensure(params.Save())

		case params.ArbosVersion_32:
			// no change state needed

		case 33, 34, 35, 36, 37, 38, 39:
			// these versions are left to Orbit chains for custom upgrades.

		case params.ArbosVersion_40:
			// EIP-2935: Add support for historical block hashes.
			stateDB.SetNonce(params.HistoryStorageAddress, 1, tracing.NonceChangeUnspecified)
			stateDB.SetCode(params.HistoryStorageAddress, params.HistoryStorageCodeArbitrum, tracing.CodeChangeUnspecified)
			// The MaxWasmSize was a constant before arbos version 40, and can
			// be read as a parameter after arbos version 40.
			params, err := state.Programs().Params()
			ensure(err)
			ensure(params.UpgradeToArbosVersion(nextArbosVersion))
			ensure(params.Save())

		case params.ArbosVersion_41:
			// no change state needed

		case 42, 43, 44, 45, 46, 47, 48, 49:
			// these versions are left to Orbit chains for custom upgrades.

		case params.ArbosVersion_50:
			p, err := state.Programs().Params()
			ensure(err)
			ensure(p.UpgradeToArbosVersion(nextArbosVersion))
			ensure(p.Save())
			ensure(state.l2PricingState.SetMaxPerTxGasLimit(l2pricing.InitialPerTxGasLimitV50))

		case params.ArbosVersion_51:
			// nothing

		case 52, 53, 54, 55, 56, 57, 58, 59:
			// these versions are left to Orbit chains for custom upgrades.

		case params.ArbosVersion_60:
			// no change state needed
		default:
			return fmt.Errorf(
				"the chain is upgrading to unsupported ArbOS version %v, %w",
				nextArbosVersion,
				ErrFatalNodeOutOfDate,
			)
		}

		// install any new precompiles
		for addr, version := range PrecompileMinArbOSVersions {
			if version == nextArbosVersion {
				stateDB.SetCode(addr, []byte{byte(vm.INVALID)}, tracing.CodeChangeUnspecified)
			}
		}

		state.arbosVersion = nextArbosVersion
		state.programs.ArbosVersion = nextArbosVersion
		state.l1PricingState.ArbosVersion = nextArbosVersion
		state.l2PricingState.ArbosVersion = nextArbosVersion
	}

	if firstTime && upgradeTo >= params.ArbosVersion_6 {
		if upgradeTo < params.ArbosVersion_11 {
			state.Restrict(state.l1PricingState.SetPerBatchGasCost(l1pricing.InitialPerBatchGasCostV6))
		}
		state.Restrict(state.l1PricingState.SetEquilibrationUnits(l1pricing.InitialEquilibrationUnitsV6))
		state.Restrict(state.l2PricingState.SetSpeedLimitPerSecond(l2pricing.InitialSpeedLimitPerSecondV6))
		state.Restrict(state.l2PricingState.SetMaxPerBlockGasLimit(l2pricing.InitialPerBlockGasLimitV6))
	}

	state.Restrict(state.backingStorage.SetUint64ByUint64(uint64(versionOffset), state.arbosVersion))

	return nil
}

func (state *ArbosState) ScheduleArbOSUpgrade(newVersion uint64, timestamp uint64) error {
	err := state.upgradeVersion.Set(newVersion)
	if err != nil {
		return err
	}
	return state.upgradeTimestamp.Set(timestamp)
}

func (state *ArbosState) GetScheduledUpgrade() (uint64, uint64, error) {
	version, err := state.upgradeVersion.Get()
	if err != nil {
		return 0, 0, err
	}
	timestamp, err := state.upgradeTimestamp.Get()
	if err != nil {
		return 0, 0, err
	}
	return version, timestamp, nil
}

func (state *ArbosState) BackingStorage() *storage.Storage {
	return state.backingStorage
}

func (state *ArbosState) Restrict(err error) {
	state.Burner.Restrict(err)
}

func (state *ArbosState) ArbOSVersion() uint64 {
	return state.arbosVersion
}

func (state *ArbosState) SetFormatVersion(val uint64) {
	state.arbosVersion = val
	state.Restrict(state.backingStorage.SetUint64ByUint64(uint64(versionOffset), val))
}

func (state *ArbosState) BrotliCompressionLevel() (uint64, error) {
	return state.brotliCompressionLevel.Get()
}

func (state *ArbosState) SetBrotliCompressionLevel(val uint64) error {
	if val <= arbcompress.LEVEL_WELL {
		return state.brotliCompressionLevel.Set(val)
	}
	return errors.New("invalid brotli compression level")
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

func (state *ArbosState) NativeTokenManagementFromTime() (uint64, error) {
	return state.nativeTokenEnabledTime.Get()
}

func (state *ArbosState) SetNativeTokenManagementFromTime(val uint64) error {
	return state.nativeTokenEnabledTime.Set(val)
}

func (state *ArbosState) NativeTokenOwners() *addressSet.AddressSet {
	return state.nativeTokenOwners
}

func (state *ArbosState) SendMerkleAccumulator() *merkleAccumulator.MerkleAccumulator {
	if state.sendMerkle == nil {
		state.sendMerkle = merkleAccumulator.OpenMerkleAccumulator(state.backingStorage.OpenCachedSubStorage(sendMerkleSubspace))
	}
	return state.sendMerkle
}

func (state *ArbosState) Programs() *programs.Programs {
	return state.programs
}

func (state *ArbosState) Features() *features.Features {
	return state.features
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

func (state *ArbosState) ChainConfig() ([]byte, error) {
	return state.chainConfig.Get()
}

func (state *ArbosState) SetChainConfig(serializedChainConfig []byte) error {
	return state.chainConfig.Set(serializedChainConfig)
}

func (state *ArbosState) GenesisBlockNum() (uint64, error) {
	return state.genesisBlockNum.Get()
}
