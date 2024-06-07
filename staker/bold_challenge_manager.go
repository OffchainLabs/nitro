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

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
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
	// A name identifier for the validator for cosmetic purposes.
	ValidatorName string `koanf:"validator-name"`
	// Path to a filesystem directory that will cache machine hashes for BOLD.
	MachineLeavesCachePath string `koanf:"machine-leaves-cache-path"`
	// How often to post assertions onchain.
	AssertionPostingIntervalSeconds uint64 `koanf:"assertion-posting-interval-seconds"`
	// How often to scan for newly created assertions onchain.
	AssertionScanningIntervalSeconds uint64 `koanf:"assertion-scanning-interval-seconds"`
	// How often to confirm assertions onchain.
	AssertionConfirmingIntervalSeconds  uint64   `koanf:"assertion-confirming-interval-seconds"`
	API                                 bool     `koanf:"api"`
	APIHost                             string   `koanf:"api-host"`
	APIPort                             uint16   `koanf:"api-port"`
	APIDBPath                           string   `koanf:"api-db-path"`
	TrackChallengeParentAssertionHashes []string `koanf:"track-challenge-parent-assertion-hashes"`
}

var DefaultBoldConfig = BoldConfig{
	Enable:                              false,
	Mode:                                "make-mode",
	BlockChallengeLeafHeight:            1 << 26,
	BigStepLeafHeight:                   1 << 23,
	SmallStepLeafHeight:                 1 << 19,
	NumBigSteps:                         1,
	ValidatorName:                       "default-validator",
	MachineLeavesCachePath:              "/tmp/machine-leaves-cache",
	AssertionPostingIntervalSeconds:     900, // Every 15 minutes.
	AssertionScanningIntervalSeconds:    60,  // Every minute.
	AssertionConfirmingIntervalSeconds:  60,  // Every minute.
	API:                                 false,
	APIHost:                             "127.0.0.1",
	APIPort:                             9393,
	APIDBPath:                           "/tmp/bold-api-db",
	TrackChallengeParentAssertionHashes: []string{},
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
	f.String(prefix+".validator-name", DefaultBoldConfig.ValidatorName, "name identifier for cosmetic purposes")
	f.String(prefix+".machine-leaves-cache-path", DefaultBoldConfig.MachineLeavesCachePath, "path to machine cache")
	f.Uint64(prefix+".assertion-posting-interval-seconds", DefaultBoldConfig.AssertionPostingIntervalSeconds, "assertion posting interval")
	f.Uint64(prefix+".assertion-scanning-interval-seconds", DefaultBoldConfig.AssertionScanningIntervalSeconds, "scan assertion interval")
	f.Uint64(prefix+".assertion-confirming-interval-seconds", DefaultBoldConfig.AssertionConfirmingIntervalSeconds, "confirm assertion interval")
	f.Bool(prefix+".api", DefaultBoldConfig.API, "enable api")
	f.String(prefix+".api-host", DefaultBoldConfig.APIHost, "bold api host")
	f.Uint16(prefix+".api-port", DefaultBoldConfig.APIPort, "bold api port")
	f.String(prefix+".api-db-path", DefaultBoldConfig.APIDBPath, "bold api db path")
	f.StringSlice(prefix+".track-challenge-parent-assertion-hashes", DefaultBoldConfig.TrackChallengeParentAssertionHashes, "only track challenges/edges with these parent assertion hashes")
}

// NewBOLDChallengeManager sets up a BOLD challenge manager implementation by providing it with
// its necessary dependencies and configuration. The challenge manager can then be started, as it
// implements the StopWaiter pattern as part of the Nitro validator.
func NewBOLDChallengeManager(
	ctx context.Context,
	rollupAddress common.Address,
	txOpts *bind.TransactOpts,
	client arbutil.L1Interface,
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
		statelessBlockValidator,
		config.MachineLeavesCachePath,
		// Specify the height constants needed for the state provider.
		// TODO: Fetch these from the smart contract instead.
		[]l2stateprovider.Height{
			blockChallengeLeafHeight,
			bigStepHeight,
			smallStepHeight,
		},
		config.ValidatorName,
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
	postingInterval := time.Second * time.Duration(config.AssertionPostingIntervalSeconds)
	// The interval at which the manager will scan for newly created assertions onchain.
	scanningInterval := time.Second * time.Duration(config.AssertionScanningIntervalSeconds)
	// The interval at which the manager will attempt to confirm assertions.
	confirmingInterval := time.Second * time.Duration(config.AssertionConfirmingIntervalSeconds)
	opts := []challengemanager.Opt{
		challengemanager.WithName(config.ValidatorName),
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
