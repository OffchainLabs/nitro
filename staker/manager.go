// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package staker

import (
	"context"
	"fmt"
	"time"

	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
)

var BoldModes = map[string]types.Mode{
	"watchtower-mode": types.WatchTowerMode,
	"resolve-mode":    types.ResolveMode,
	"defensive-mode":  types.DefensiveMode,
	"make-mode":       types.MakeMode,
}

func NewManager(
	ctx context.Context,
	rollupAddress common.Address,
	txOpts *bind.TransactOpts,
	client arbutil.L1Interface,
	statelessBlockValidator *StatelessBlockValidator,
	config *BoldConfig,
	dataPoster *dataposter.DataPoster,
) (*challengemanager.Manager, error) {
	rollupBindings, err := rollupgen.NewRollupUserLogic(rollupAddress, client)
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
	stateManager, err := NewStateManager(
		statelessBlockValidator,
		config.MachineLeavesCachePath,
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
		stateManager,
		stateManager,
		stateManager,
		providerHeights,
		stateManager,
		nil,
	)
	postingInterval := time.Second * time.Duration(config.AssertionPostingIntervalSeconds)
	scanningInterval := time.Second * time.Duration(config.AssertionScanningIntervalSeconds)
	confirmingInterval := time.Second * time.Duration(config.AssertionConfirmingIntervalSeconds)
	opts := []challengemanager.Opt{
		challengemanager.WithName(config.ValidatorName),
		challengemanager.WithMode(BoldModes[config.Mode]),
		challengemanager.WithAssertionPostingInterval(postingInterval),
		challengemanager.WithAssertionScanningInterval(scanningInterval),
		challengemanager.WithAssertionConfirmingInterval(confirmingInterval),
		challengemanager.WithAddress(txOpts.From),
	}
	if config.API {
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
