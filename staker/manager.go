// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package staker

import (
	"context"
	"time"

	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

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
	callOpts bind.CallOpts,
	client arbutil.L1Interface,
	statelessBlockValidator *StatelessBlockValidator,
	config *BoldConfig,
) (*challengemanager.Manager, error) {
	userLogic, err := rollupgen.NewRollupUserLogic(
		rollupAddress, client,
	)
	if err != nil {
		return nil, err
	}
	challengeManagerAddr, err := userLogic.RollupUserLogicCaller.ChallengeManager(
		&bind.CallOpts{Context: ctx},
	)
	if err != nil {
		return nil, err
	}
	chain, err := solimpl.NewAssertionChain(
		ctx,
		rollupAddress,
		challengeManagerAddr,
		txOpts,
		client,
		solimpl.NewChainBackendTransactor(client),
	)
	if err != nil {
		return nil, err
	}
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(challengeManagerAddr, client)
	if err != nil {
		return nil, err
	}
	numBigStepLevel, err := managerBinding.NUMBIGSTEPLEVEL(&callOpts)
	if err != nil {
		return nil, err
	}
	challengeLeafHeights := make([]l2stateprovider.Height, numBigStepLevel+2)
	for i := uint8(0); i <= numBigStepLevel+1; i++ {
		leafHeight, err := managerBinding.GetLayerZeroEndHeight(&callOpts, i)
		if err != nil {
			return nil, err
		}
		challengeLeafHeights[i] = l2stateprovider.Height(leafHeight.Uint64())
	}

	stateManager, err := NewStateManager(
		statelessBlockValidator,
		config.MachineLeavesCachePath,
		challengeLeafHeights,
		config.ValidatorName,
	)
	if err != nil {
		return nil, err
	}
	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateManager,
		stateManager,
		stateManager,
		challengeLeafHeights,
		stateManager,
		nil,
	)
	manager, err := challengemanager.New(
		ctx,
		chain,
		provider,
		rollupAddress,
		challengemanager.WithName(config.ValidatorName),
		challengemanager.WithMode(BoldModes[config.Mode]),
		challengemanager.WithAssertionPostingInterval(time.Duration(config.AssertionPostingIntervalSeconds)),
		challengemanager.WithAssertionScanningInterval(time.Duration(config.AssertionScanningIntervalSeconds)),
		challengemanager.WithAssertionConfirmingInterval(time.Duration(config.AssertionConfirmingIntervalSeconds)),
		challengemanager.WithEdgeTrackerWakeInterval(time.Duration(config.EdgeTrackerWakeIntervalSeconds)),
		challengemanager.WithAddress(txOpts.From),
	)
	if err != nil {
		return nil, err
	}
	provider.UpdateAPIDatabase(manager.Database())
	return manager, nil
}
