// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package assertions_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/OffchainLabs/bold/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestPostAssertion(t *testing.T) {
	setup, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockBridge(),
		setup.WithMockOneStepProver(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
	)
	require.NoError(t, err)

	bridgeBindings, err := mocksgen.NewBridgeStub(setup.Addrs.Bridge, setup.Backend)
	require.NoError(t, err)

	rollupAdminBindings, err := rollupgen.NewRollupAdminLogic(setup.Addrs.Rollup, setup.Backend)
	require.NoError(t, err)
	_, err = rollupAdminBindings.SetMinimumAssertionPeriod(setup.Accounts[0].TxOpts, big.NewInt(1))
	require.NoError(t, err)
	setup.Backend.Commit()

	msgCount, err := bridgeBindings.SequencerMessageCount(util.GetSafeCallOpts(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := setup.Chains[0]

	ctx := context.Background()

	stateManagerOpts := setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithNumBatchesRead(5),
	)
	stateManager, err := statemanager.NewForSimpleMachine(stateManagerOpts...)
	require.NoError(t, err)

	chalManager, err := challengemanager.New(
		ctx,
		aliceChain,
		stateManager,
		setup.Addrs.Rollup,
		challengemanager.WithMode(types.DefensiveMode),
		challengemanager.WithEdgeTrackerWakeInterval(time.Hour),
	)
	require.NoError(t, err)
	chalManager.Start(ctx)

	postState, err := stateManager.ExecutionStateAfterBatchCount(ctx, 1)
	require.NoError(t, err)

	assertionManager, err := assertions.NewManager(
		aliceChain,
		stateManager,
		setup.Backend,
		chalManager,
		aliceChain.RollupAddress(),
		chalManager.ChallengeManagerAddress(),
		"alice",
		time.Millisecond*200, // poll interval for assertions
		time.Hour,            // confirmation attempt interval
		stateManager,
		time.Millisecond*100, // poll interval
		time.Second*1,
		nil,
		assertions.WithDangerousReadyToPost(),
		assertions.WithPostingDisabled(),
	)
	require.NoError(t, err)

	go assertionManager.Start(ctx)

	time.Sleep(time.Second)

	posted, err := assertionManager.PostAssertion(ctx)
	require.NoError(t, err)
	require.Equal(t, true, posted.IsSome())
	require.Equal(t, postState, protocol.GoExecutionStateFromSolidity(posted.Unwrap().AfterState))
}
