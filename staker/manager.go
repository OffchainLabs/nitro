package staker

import (
	"context"
	"time"

	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	challengemanager "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/types"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"

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
	blockValidator *BlockValidator,
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
	bigStepEdgeHeight, err := managerBinding.LAYERZEROBIGSTEPEDGEHEIGHT(&callOpts)
	if err != nil {
		return nil, err
	}
	smallStepEdgeHeight, err := managerBinding.LAYERZEROSMALLSTEPEDGEHEIGHT(&callOpts)
	if err != nil {
		return nil, err
	}
	stateManager, err := NewStateManager(
		statelessBlockValidator,
		blockValidator,
		smallStepEdgeHeight.Uint64(),
		bigStepEdgeHeight.Uint64()*smallStepEdgeHeight.Uint64(),
	)
	if err != nil {
		return nil, err
	}
	manager, err := challengemanager.New(
		ctx,
		chain,
		client,
		stateManager,
		rollupAddress,
		challengemanager.WithMode(types.MakeMode),
		challengemanager.WithAssertionPostingInterval(time.Second*5),
		challengemanager.WithAssertionScanningInterval(time.Second),
	)
	if err != nil {
		return nil, err
	}
	return manager, nil
}
