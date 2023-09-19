// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package staker

import (
	"context"

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

func NewManager(
	ctx context.Context,
	rollupAddress common.Address,
	txOpts *bind.TransactOpts,
	callOpts bind.CallOpts,
	client arbutil.L1Interface,
	statelessBlockValidator *StatelessBlockValidator,
	historyCacheBaseDir string,
) (*challengemanager.Manager, error) {
	chain, err := solimpl.NewAssertionChain(
		ctx,
		rollupAddress,
		txOpts,
		client,
	)
	if err != nil {
		return nil, err
	}
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
	managerBinding, err := challengeV2gen.NewEdgeChallengeManager(challengeManagerAddr, client)
	if err != nil {
		return nil, err
	}
	numBigStepLevel, err := managerBinding.NUMBIGSTEPLEVEL(&callOpts)
	if err != nil {
		return nil, err
	}
	challengeLeafHeights := make([]l2stateprovider.Height, numBigStepLevel.Uint64()+2)
	for i := uint64(0); i <= numBigStepLevel.Uint64()+1; i++ {
		leafHeight, err := managerBinding.GetLayerZeroEndHeight(&callOpts, uint8(i))
		if err != nil {
			return nil, err
		}
		challengeLeafHeights[i] = l2stateprovider.Height(leafHeight.Uint64())
	}

	stateManager, err := NewStateManager(
		statelessBlockValidator,
		historyCacheBaseDir,
		challengeLeafHeights,
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
	)
	manager, err := challengemanager.New(
		ctx,
		chain,
		client,
		provider,
		rollupAddress,
		challengemanager.WithMode(types.MakeMode),
	)
	if err != nil {
		return nil, err
	}
	return manager, nil
}
