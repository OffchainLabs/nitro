// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbosState

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/arbos/addressTable"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/blockhash"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

// ArbosState contains ArbOS-related state. It is backed by ArbOS's storage in the persistent stateDB.
// Modifications to the ArbosState are written through to the underlying StateDB so that the StateDB always
// has the definitive state, stored persistently. (Note that some tests use memory-backed StateDB's that aren't
// persisted beyond the end of the test.)

type ArbosState struct {
	arbosVersion                  uint64                      // version of the ArbOS storage format and semantics
	maxArbosVersionSupported      uint64                      // maximum ArbOS version supported by this code
	maxDebugArbosVersionSupported uint64                      // maximum ArbOS version supported by this code in debug mode
	upgradeVersion                storage.StorageBackedUint64 // version we're planning to upgrade to, or 0 if not planning to upgrade
	upgradeTimestamp              storage.StorageBackedUint64 // when to do the planned upgrade
	networkFeeAccount             storage.StorageBackedAddress
	l1PricingState                *l1pricing.L1PricingState
	l2PricingState                *l2pricing.L2PricingState
	retryableState                *retryables.RetryableState
	addressTable                  *addressTable.AddressTable
	chainOwners                   *addressSet.AddressSet
	sendMerkle                    *merkleAccumulator.MerkleAccumulator
	programs                      *programs.Programs
	blockhashes                   *blockhash.Blockhashes
	chainId                       storage.StorageBackedBigInt
	chainConfig                   storage.StorageBackedBytes
	genesisBlockNum               storage.StorageBackedUint64
	infraFeeAccount               storage.StorageBackedAddress
	brotliCompressionLevel        storage.StorageBackedUint64 // brotli compression level used for pricing
	backingStorage                *storage.Storage
	Burner                        burn.Burner
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
		30,
		30,
		backingStorage.OpenStorageBackedUint64(uint64(upgradeVersionOffset)),
		backingStorage.OpenStorageBackedUint64(uint64(upgradeTimestampOffset)),
		backingStorage.OpenStorageBackedAddress(uint64(networkFeeAccountOffset)),
		l1pricing.OpenL1PricingState(backingStorage.OpenCachedSubStorage(l1PricingSubspace)),
		l2pricing.OpenL2PricingState(backingStorage.OpenCachedSubStorage(l2PricingSubspace)),
		retryables.OpenRetryableState(backingStorage.OpenCachedSubStorage(retryablesSubspace), stateDB),
		addressTable.Open(backingStorage.OpenCachedSubStorage(addressTableSubspace)),
		addressSet.OpenAddressSet(backingStorage.OpenCachedSubStorage(chainOwnerSubspace)),
		merkleAccumulator.OpenMerkleAccumulator(backingStorage.OpenCachedSubStorage(sendMerkleSubspace)),
		programs.Open(backingStorage.OpenSubStorage(programsSubspace)),
		blockhash.OpenBlockhashes(backingStorage.OpenCachedSubStorage(blockhashesSubspace)),
		backingStorage.OpenStorageBackedBigInt(uint64(chainIdOffset)),
		backingStorage.OpenStorageBackedBytes(chainConfigSubspace),
		backingStorage.OpenStorageBackedUint64(uint64(genesisBlockNumOffset)),
		backingStorage.OpenStorageBackedAddress(uint64(infraFeeAccountOffset)),
		backingStorage.OpenStorageBackedUint64(uint64(brotliCompressionLevelOffset)),
		backingStorage,
		burner,
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
	raw := rawdb.NewMemoryDatabase()
	// TODO pathdb
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		log.Crit("failed to init empty statedb", "error", err)
	}
	burner := burn.NewSystemBurner(nil, false)
	chainConfig := params.ArbitrumDevTestChainConfig()
	newState, err := InitializeArbosState(statedb, burner, chainConfig, arbostypes.TestInitMessage)
	if err != nil {
		log.Crit("failed to open the ArbOS state", "error", err)
	}
	return newState, statedb
}

// ArbOSVersion returns the ArbOS version
func ArbOSVersion(stateDB vm.StateDB) uint64 {
	backingStorage := storage.NewGeth(stateDB, burn.NewSystemBurner(nil, false))
	arbosVersion, err := backingStorage.GetUint64ByUint64(uint64(versionOffset))
	if err != nil {
		log.Crit("failed to get the ArbOS version", "error", err)
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
)

type SubspaceID []byte

var (
	l1PricingSubspace    SubspaceID = []byte{0}
	l2PricingSubspace    SubspaceID = []byte{1}
	retryablesSubspace   SubspaceID = []byte{2}
	addressTableSubspace SubspaceID = []byte{3}
	chainOwnerSubspace   SubspaceID = []byte{4}
	sendMerkleSubspace   SubspaceID = []byte{5}
	blockhashesSubspace  SubspaceID = []byte{6}
	chainConfigSubspace  SubspaceID = []byte{7}
	programsSubspace     SubspaceID = []byte{8}
)

var PrecompileMinArbOSVersions = make(map[common.Address]uint64)

func InitializeArbosState(stateDB vm.StateDB, burner burn.Burner, chainConfig *params.ChainConfig, initMessage *arbostypes.ParsedInitMessage) (*ArbosState, error) {
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
			stateDB.SetCode(addr, []byte{byte(vm.INVALID)})
		}
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
	chainConfigStorage := sto.OpenStorageBackedBytes(chainConfigSubspace)
	_ = chainConfigStorage.Set(initMessage.SerializedChainConfig)
	_ = sto.SetUint64ByUint64(uint64(genesisBlockNumOffset), chainConfig.ArbitrumChainParams.GenesisBlockNum)
	_ = sto.SetUint64ByUint64(uint64(brotliCompressionLevelOffset), 0) // default brotliCompressionLevel for fast compression is 0

	initialRewardsRecipient := l1pricing.BatchPosterAddress
	if desiredArbosVersion >= 2 {
		initialRewardsRecipient = initialChainOwner
	}
	_ = l1pricing.InitializeL1PricingState(sto.OpenCachedSubStorage(l1PricingSubspace), initialRewardsRecipient, initMessage.InitialL1BaseFee)
	_ = l2pricing.InitializeL2PricingState(sto.OpenCachedSubStorage(l2PricingSubspace))
	_ = retryables.InitializeRetryableState(sto.OpenCachedSubStorage(retryablesSubspace))
	addressTable.Initialize(sto.OpenCachedSubStorage(addressTableSubspace))
	merkleAccumulator.InitializeMerkleAccumulator(sto.OpenCachedSubStorage(sendMerkleSubspace))
	blockhash.InitializeBlockhashes(sto.OpenCachedSubStorage(blockhashesSubspace))

	ownersStorage := sto.OpenCachedSubStorage(chainOwnerSubspace)
	_ = addressSet.Initialize(ownersStorage)
	_ = addressSet.OpenAddressSet(ownersStorage).Add(initialChainOwner)

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
		case 2:
			ensure(state.l1PricingState.SetLastSurplus(common.Big0, 1))
		case 3:
			ensure(state.l1PricingState.SetPerBatchGasCost(0))
			ensure(state.l1PricingState.SetAmortizedCostCapBips(math.MaxUint64))
		case 4:
			// no state changes needed
		case 5:
			// no state changes needed
		case 6:
			// no state changes needed
		case 7:
			// no state changes needed
		case 8:
			// no state changes needed
		case 9:
			// no state changes needed
		case 10:
			ensure(state.l1PricingState.SetL1FeesAvailable(stateDB.GetBalance(
				l1pricing.L1PricerFundsPoolAddress,
			).ToBig()))

		case 11:
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

		case 20:
			// Update Brotli compression level for fast compression from 0 to 1
			ensure(state.SetBrotliCompressionLevel(1))

		case 21, 22, 23, 24, 25, 26, 27, 28, 29:
			// these versions are left to Orbit chains for custom upgrades.

		case 30:
			programs.Initialize(state.backingStorage.OpenSubStorage(programsSubspace))

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
				stateDB.SetCode(addr, []byte{byte(vm.INVALID)})
			}
		}

		state.arbosVersion = nextArbosVersion
	}

	if firstTime && upgradeTo >= 6 {
		if upgradeTo < 11 {
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

func (state *ArbosState) MaxArbosVersionSupported() uint64 {
	return state.maxArbosVersionSupported
}

func (state *ArbosState) MaxDebugArbosVersionSupported() uint64 {
	return state.maxDebugArbosVersionSupported
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
		state.sendMerkle = merkleAccumulator.OpenMerkleAccumulator(state.backingStorage.OpenCachedSubStorage(sendMerkleSubspace))
	}
	return state.sendMerkle
}

func (state *ArbosState) Programs() *programs.Programs {
	return state.programs
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
