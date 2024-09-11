// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	boldtypes "github.com/OffchainLabs/bold/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	boldrollup "github.com/OffchainLabs/bold/solgen/go/rollupgen"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

type BoldConfig struct {
	Enable bool   `koanf:"enable"`
	Mode   string `koanf:"mode"`
	// The height constants at each challenge level for the BOLD challenge manager.
	BlockChallengeLeafHeight uint64 `koanf:"block-challenge-leaf-height"`
	BigStepLeafHeight        uint64 `koanf:"big-step-leaf-height"`
	SmallStepLeafHeight      uint64 `koanf:"small-step-leaf-height"`
	// Number of big step challenges in the BOLD protocol.
	NumBigSteps uint64 `koanf:"num-big-steps"`
	// How often to post assertions onchain.
	AssertionPostingInterval time.Duration `koanf:"assertion-posting-interval"`
	// How often to scan for newly created assertions onchain.
	AssertionScanningInterval time.Duration `koanf:"assertion-scanning-interval"`
	// How often to confirm assertions onchain.
	AssertionConfirmingInterval         time.Duration       `koanf:"assertion-confirming-interval"`
	API                                 bool                `koanf:"api"`
	APIHost                             string              `koanf:"api-host"`
	APIPort                             uint16              `koanf:"api-port"`
	APIDBPath                           string              `koanf:"api-db-path"`
	TrackChallengeParentAssertionHashes []string            `koanf:"track-challenge-parent-assertion-hashes"`
	CheckStakerSwitchInterval           time.Duration       `koanf:"check-staker-switch-interval"`
	StateProviderConfig                 StateProviderConfig `koanf:"state-provider-config"`
}

type StateProviderConfig struct {
	// A name identifier for the validator for cosmetic purposes.
	ValidatorName      string `koanf:"validator-name"`
	CheckBatchFinality bool   `koanf:"check-batch-finality"`
	// Path to a filesystem directory that will cache machine hashes for BOLD.
	MachineLeavesCachePath string `koanf:"machine-leaves-cache-path"`
}

var DefaultStateProviderConfig = StateProviderConfig{
	ValidatorName:          "default-validator",
	CheckBatchFinality:     true,
	MachineLeavesCachePath: "/tmp/machine-leaves-cache",
}

var DefaultBoldConfig = BoldConfig{
	Enable:                              false,
	Mode:                                "make-mode",
	BlockChallengeLeafHeight:            1 << 26,
	BigStepLeafHeight:                   1 << 23,
	SmallStepLeafHeight:                 1 << 19,
	NumBigSteps:                         1,
	AssertionPostingInterval:            time.Minute * 15,
	AssertionScanningInterval:           time.Minute,
	AssertionConfirmingInterval:         time.Minute,
	API:                                 false,
	APIHost:                             "127.0.0.1",
	APIPort:                             9393,
	APIDBPath:                           "/tmp/bold-api-db",
	TrackChallengeParentAssertionHashes: []string{},
	CheckStakerSwitchInterval:           time.Minute, // Every minute, check if the Nitro node staker should switch to using BOLD.
	StateProviderConfig:                 DefaultStateProviderConfig,
}

var BoldModes = map[string]boldtypes.Mode{
	"watchtower-mode": boldtypes.WatchTowerMode,
	"resolve-mode":    boldtypes.ResolveMode,
	"defensive-mode":  boldtypes.DefensiveMode,
	"make-mode":       boldtypes.MakeMode,
}

func BoldConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBoldConfig.Enable, "enable bold challenge protocol")
	f.String(prefix+".mode", DefaultBoldConfig.Mode, "define the bold validator staker strategy")
	f.Uint64(prefix+".block-challenge-leaf-height", DefaultBoldConfig.BlockChallengeLeafHeight, "block challenge leaf height")
	f.Uint64(prefix+".big-step-leaf-height", DefaultBoldConfig.BigStepLeafHeight, "big challenge leaf height")
	f.Uint64(prefix+".small-step-leaf-height", DefaultBoldConfig.SmallStepLeafHeight, "small challenge leaf height")
	f.Uint64(prefix+".num-big-steps", DefaultBoldConfig.NumBigSteps, "num big steps")
	f.Duration(prefix+".assertion-posting-interval", DefaultBoldConfig.AssertionPostingInterval, "assertion posting interval")
	f.Duration(prefix+".assertion-scanning-interval", DefaultBoldConfig.AssertionScanningInterval, "scan assertion interval")
	f.Duration(prefix+".assertion-confirming-interval", DefaultBoldConfig.AssertionConfirmingInterval, "confirm assertion interval")
	f.Duration(prefix+".check-staker-switch-interval", DefaultBoldConfig.CheckStakerSwitchInterval, "how often to check if staker can switch to bold")
	f.Bool(prefix+".api", DefaultBoldConfig.API, "enable api")
	f.String(prefix+".api-host", DefaultBoldConfig.APIHost, "bold api host")
	f.Uint16(prefix+".api-port", DefaultBoldConfig.APIPort, "bold api port")
	f.String(prefix+".api-db-path", DefaultBoldConfig.APIDBPath, "bold api db path")
	f.StringSlice(prefix+".track-challenge-parent-assertion-hashes", DefaultBoldConfig.TrackChallengeParentAssertionHashes, "only track challenges/edges with these parent assertion hashes")
	StateProviderConfigAddOptions(prefix+".state-provider-config", f)
}

func StateProviderConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".validator-name", DefaultStateProviderConfig.ValidatorName, "name identifier for cosmetic purposes")
	f.Bool(prefix+".check-batch-finality", DefaultStateProviderConfig.CheckBatchFinality, "check batch finality")
	f.String(prefix+".machine-leaves-cache-path", DefaultStateProviderConfig.MachineLeavesCachePath, "path to machine cache")
}

type BOLDStaker struct {
	stopwaiter.StopWaiter
	config             *BoldConfig
	chalManager        *challengemanager.Manager
	blockValidator     *BlockValidator
	rollupAddress      common.Address
	client             bind.ContractBackend
	lastWasmModuleRoot common.Hash
	callOpts           bind.CallOpts
	validatorConfig    L1ValidatorConfig
	wallet             ValidatorWalletInterface
	stakedNotifiers    []LatestStakedNotifier
	confirmedNotifiers []LatestConfirmedNotifier
}

func newBOLDStaker(
	ctx context.Context,
	validatorConfig L1ValidatorConfig,
	rollupAddress common.Address,
	callOpts bind.CallOpts,
	txOpts *bind.TransactOpts,
	client arbutil.L1Interface,
	blockValidator *BlockValidator,
	statelessBlockValidator *StatelessBlockValidator,
	config *BoldConfig,
	dataPoster *dataposter.DataPoster,
	wallet ValidatorWalletInterface,
	stakedNotifiers []LatestStakedNotifier,
	confirmedNotifiers []LatestConfirmedNotifier,
) (*BOLDStaker, error) {
	manager, err := newBOLDChallengeManager(ctx, rollupAddress, txOpts, client, blockValidator, statelessBlockValidator, config, dataPoster)
	if err != nil {
		return nil, err
	}
	return &BOLDStaker{
		config:             config,
		chalManager:        manager,
		blockValidator:     blockValidator,
		rollupAddress:      rollupAddress,
		client:             client,
		callOpts:           callOpts,
		validatorConfig:    validatorConfig,
		wallet:             wallet,
		stakedNotifiers:    stakedNotifiers,
		confirmedNotifiers: confirmedNotifiers,
	}, nil
}

// Initialize Updates the block validator module root.
// And updates the init state of the block validator if block validator has not started yet.
func (b *BOLDStaker) Initialize(ctx context.Context) error {
	if err := b.updateBlockValidatorModuleRoot(ctx); err != nil {
		return err
	}
	walletAddressOrZero := b.wallet.AddressOrZero()
	if b.blockValidator != nil && b.validatorConfig.StartValidationFromStaked && !b.blockValidator.Started() {
		rollupUserLogic, err := boldrollup.NewRollupUserLogic(b.rollupAddress, b.client)
		if err != nil {
			return err
		}
		latestStaked, err := rollupUserLogic.LatestStakedAssertion(b.getCallOpts(ctx), walletAddressOrZero)
		if err != nil {
			return err
		}
		if latestStaked == [32]byte{} {
			latestConfirmed, err := rollupUserLogic.LatestConfirmed(&bind.CallOpts{Context: ctx})
			if err != nil {
				return err
			}
			latestStaked = latestConfirmed
		}
		assertion, err := readBoldAssertionCreationInfo(
			ctx,
			rollupUserLogic,
			b.client,
			b.rollupAddress,
			latestStaked,
		)
		if err != nil {
			return err
		}
		afterState := protocol.GoGlobalStateFromSolidity(assertion.AfterState.GlobalState)
		return b.blockValidator.InitAssumeValid(validator.GoGlobalState(afterState))
	}
	return nil
}

func (b *BOLDStaker) Start(ctxIn context.Context) {
	b.StopWaiter.Start(ctxIn, b)
	b.chalManager.Start(ctxIn)
	b.CallIteratively(func(ctx context.Context) time.Duration {
		err := b.updateBlockValidatorModuleRoot(ctx)
		if err != nil {
			log.Warn("error updating latest wasm module root", "err", err)
		}
		agreedMsgCount, agreedGlobalState, err := b.getLatestState(ctx, false)
		if err != nil {
			log.Error("staker: error checking latest agreed", "err", err)
		}

		if agreedGlobalState != nil {
			for _, notifier := range b.stakedNotifiers {
				notifier.UpdateLatestStaked(agreedMsgCount, *agreedGlobalState)
			}
		}
		confirmedMsgCount, confirmedGlobalState, err := b.getLatestState(ctx, true)
		if err != nil {
			log.Error("staker: error checking latest confirmed", "err", err)
		}

		if confirmedGlobalState != nil {
			for _, notifier := range b.confirmedNotifiers {
				notifier.UpdateLatestConfirmed(confirmedMsgCount, *confirmedGlobalState)
			}
		}
		return b.config.AssertionPostingInterval
	})
}

func (b *BOLDStaker) getLatestState(ctx context.Context, confirmed bool) (arbutil.MessageIndex, *validator.GoGlobalState, error) {
	var globalState protocol.GoGlobalState
	var err error
	if confirmed {
		globalState, err = b.chalManager.LatestConfirmedState(ctx)
	} else {
		globalState, err = b.chalManager.LatestAgreedState(ctx)
	}
	var assertionType string
	if confirmed {
		assertionType = "confirmed"
	} else {
		assertionType = "agreed"
	}
	if err != nil {
		return 0, nil, fmt.Errorf("error getting latest %s: %w", assertionType, err)
	}
	caughtUp, count, err := GlobalStateToMsgCount(b.blockValidator.inboxTracker, b.blockValidator.streamer, validator.GoGlobalState(globalState))
	if err != nil {
		if errors.Is(err, ErrGlobalStateNotInChain) {
			return 0, nil, fmt.Errorf("latest %s assertion of %v not yet in our node: %w", assertionType, globalState, err)
		}
		return 0, nil, fmt.Errorf("error getting message count: %w", err)
	}

	if !caughtUp {
		log.Info(fmt.Sprintf("latest %s assertion not yet in our node", assertionType), "state", globalState)
		return 0, nil, nil
	}

	processedCount, err := b.blockValidator.streamer.GetProcessedMessageCount()
	if err != nil {
		return 0, nil, err
	}

	if processedCount < count {
		log.Info("execution catching up to rollup", "rollupCount", count, "processedCount", processedCount)
		return 0, nil, nil
	}

	return count, (*validator.GoGlobalState)(&globalState), nil
}

func (b *BOLDStaker) StopAndWait() {
	b.chalManager.StopAndWait()
	b.StopWaiter.StopAndWait()
}

func (b *BOLDStaker) updateBlockValidatorModuleRoot(ctx context.Context) error {
	if b.blockValidator == nil {
		return nil
	}
	boldRollup, err := boldrollup.NewRollupUserLogic(b.rollupAddress, b.client)
	if err != nil {
		return err
	}
	moduleRoot, err := boldRollup.WasmModuleRoot(b.getCallOpts(ctx))
	if err != nil {
		return err
	}
	if moduleRoot != b.lastWasmModuleRoot {
		err := b.blockValidator.SetCurrentWasmModuleRoot(moduleRoot)
		if err != nil {
			return err
		}
		b.lastWasmModuleRoot = moduleRoot
	} else if (moduleRoot == common.Hash{}) {
		return errors.New("wasmModuleRoot in rollup is zero")
	}
	return nil
}

func (b *BOLDStaker) getCallOpts(ctx context.Context) *bind.CallOpts {
	opts := b.callOpts
	opts.Context = ctx
	return &opts
}

// Sets up a BOLD challenge manager implementation by providing it with
// its necessary dependencies and configuration. The challenge manager can then be started, as it
// implements the StopWaiter pattern as part of the Nitro validator.
func newBOLDChallengeManager(
	ctx context.Context,
	rollupAddress common.Address,
	txOpts *bind.TransactOpts,
	client arbutil.L1Interface,
	blockValidator *BlockValidator,
	statelessBlockValidator *StatelessBlockValidator,
	config *BoldConfig,
	dataPoster *dataposter.DataPoster,
) (*challengemanager.Manager, error) {
	// Initializes the BOLD contract bindings and the assertion chain abstraction.
	rollupBindings, err := boldrollup.NewRollupUserLogic(rollupAddress, client)
	if err != nil {
		return nil, fmt.Errorf("could not create rollup bindings: %w", err)
	}
	chalManager, err := rollupBindings.ChallengeManager(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not get challenge manager: %w", err)
	}
	assertionChain, err := solimpl.NewAssertionChain(ctx, rollupAddress, chalManager, txOpts, client, solimpl.NewDataPosterTransactor(dataPoster))
	if err != nil {
		return nil, fmt.Errorf("could not create assertion chain: %w", err)
	}
	blockChallengeLeafHeight := l2stateprovider.Height(config.BlockChallengeLeafHeight)
	bigStepHeight := l2stateprovider.Height(config.BigStepLeafHeight)
	smallStepHeight := l2stateprovider.Height(config.SmallStepLeafHeight)

	// Sets up the state provider interface that BOLD will use to request data such as
	// execution states for assertions, history commitments for machine execution, and one step proofs.
	stateProvider, err := NewBOLDStateProvider(
		blockValidator,
		statelessBlockValidator,
		// Specify the height constants needed for the state provider.
		// TODO: Fetch these from the smart contract instead.
		blockChallengeLeafHeight,
		&config.StateProviderConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create state manager: %w", err)
	}
	providerHeights := []l2stateprovider.Height{blockChallengeLeafHeight}
	for i := uint64(0); i < config.NumBigSteps; i++ {
		providerHeights = append(providerHeights, bigStepHeight)
	}
	providerHeights = append(providerHeights, smallStepHeight)
	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateProvider,
		stateProvider,
		stateProvider,
		providerHeights,
		stateProvider,
		nil, // Nil API database for the history commitment provider, as it will be provided later. TODO: Improve this dependency injection.
	)
	// The interval at which the challenge manager will attempt to post assertions.
	postingInterval := config.AssertionPostingInterval
	// The interval at which the manager will scan for newly created assertions onchain.
	scanningInterval := config.AssertionScanningInterval
	// The interval at which the manager will attempt to confirm assertions.
	confirmingInterval := config.AssertionConfirmingInterval
	opts := []challengemanager.Opt{
		challengemanager.WithName(config.StateProviderConfig.ValidatorName),
		challengemanager.WithMode(BoldModes[config.Mode]),
		challengemanager.WithAssertionPostingInterval(postingInterval),
		challengemanager.WithAssertionScanningInterval(scanningInterval),
		challengemanager.WithAssertionConfirmingInterval(confirmingInterval),
		challengemanager.WithAddress(txOpts.From),
		// Configure the validator to track only certain challenges if configured to do so.
		challengemanager.WithTrackChallengeParentAssertionHashes(config.TrackChallengeParentAssertionHashes),
	}
	if config.API {
		// Conditionally enables the BOLD API if configured.
		opts = append(opts, challengemanager.WithAPIEnabled(fmt.Sprintf("%s:%d", config.APIHost, config.APIPort), config.APIDBPath))
	}
	manager, err := challengemanager.New(
		ctx,
		assertionChain,
		provider,
		assertionChain.RollupAddress(),
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create challenge manager: %w", err)
	}
	provider.UpdateAPIDatabase(manager.Database())
	return manager, nil
}

// Read the creation info for an assertion by looking up its creation
// event from the rollup contracts.
func readBoldAssertionCreationInfo(
	ctx context.Context,
	rollup *boldrollup.RollupUserLogic,
	client bind.ContractFilterer,
	rollupAddress common.Address,
	assertionHash common.Hash,
) (*protocol.AssertionCreatedInfo, error) {
	var creationBlock uint64
	var topics [][]common.Hash
	if assertionHash == (common.Hash{}) {
		rollupDeploymentBlock, err := rollup.RollupDeploymentBlock(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, err
		}
		if !rollupDeploymentBlock.IsUint64() {
			return nil, errors.New("rollup deployment block was not a uint64")
		}
		creationBlock = rollupDeploymentBlock.Uint64()
		topics = [][]common.Hash{{assertionCreatedId}}
	} else {
		var b [32]byte
		copy(b[:], assertionHash[:])
		node, err := rollup.GetAssertion(&bind.CallOpts{Context: ctx}, b)
		if err != nil {
			return nil, err
		}
		creationBlock = node.CreatedAtBlock
		topics = [][]common.Hash{{assertionCreatedId}, {assertionHash}}
	}
	var query = ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(creationBlock),
		ToBlock:   new(big.Int).SetUint64(creationBlock),
		Addresses: []common.Address{rollupAddress},
		Topics:    topics,
	}
	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, errors.New("no assertion creation logs found")
	}
	if len(logs) > 1 {
		return nil, errors.New("found multiple instances of requested node")
	}
	ethLog := logs[0]
	parsedLog, err := rollup.ParseAssertionCreated(ethLog)
	if err != nil {
		return nil, err
	}
	afterState := parsedLog.Assertion.AfterState
	return &protocol.AssertionCreatedInfo{
		ConfirmPeriodBlocks: parsedLog.ConfirmPeriodBlocks,
		RequiredStake:       parsedLog.RequiredStake,
		ParentAssertionHash: parsedLog.ParentAssertionHash,
		BeforeState:         parsedLog.Assertion.BeforeState,
		AfterState:          afterState,
		InboxMaxCount:       parsedLog.InboxMaxCount,
		AfterInboxBatchAcc:  parsedLog.AfterInboxBatchAcc,
		AssertionHash:       parsedLog.AssertionHash,
		WasmModuleRoot:      parsedLog.WasmModuleRoot,
		ChallengeManager:    parsedLog.ChallengeManager,
		TransactionHash:     ethLog.TxHash,
		CreationBlock:       ethLog.BlockNumber,
	}, nil
}
