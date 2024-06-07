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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
)

var BoldModes = map[string]boldtypes.Mode{
	"watchtower-mode": boldtypes.WatchTowerMode,
	"resolve-mode":    boldtypes.ResolveMode,
	"defensive-mode":  boldtypes.DefensiveMode,
	"make-mode":       boldtypes.MakeMode,
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
