// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/bold/blob/main/LICENSE.md

package assertions_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/offchainlabs/bold/assertions"
	protocol "github.com/offchainlabs/bold/chain-abstraction"
	cm "github.com/offchainlabs/bold/challenge-manager"
	"github.com/offchainlabs/bold/challenge-manager/types"
	"github.com/offchainlabs/bold/solgen/go/mocksgen"
	challenge_testing "github.com/offchainlabs/bold/testing"
	statemanager "github.com/offchainlabs/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/bold/testing/setup"
)

func TestPostAssertion(t *testing.T) {
	ctx := context.Background()
	setup, err := setup.ChainsWithEdgeChallengeManager(
		// setup.WithMockBridge(),
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

	msgCount, err := bridgeBindings.SequencerMessageCount(setup.Chains[0].GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := setup.Chains[0]

	stateManagerOpts := setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		statemanager.WithNumBatchesRead(5),
	)
	stateManager, err := statemanager.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	// Set MinimumGapToBlockCreationTime as 1 second to verify that a new assertion is only posted after 1 sec has passed
	// from parent assertion creation. This will make the test run for ~19 seconds as the parent assertion time is
	// ~18 seconds in the future
	assertionManager, err := assertions.NewManager(
		aliceChain,
		stateManager,
		"alice",
		types.DefensiveMode,
		assertions.WithPollingInterval(time.Millisecond*200),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(time.Second),
	)
	require.NoError(t, err)

	chalManager, err := cm.NewChallengeStack(
		aliceChain,
		stateManager,
		cm.StackWithMode(types.DefensiveMode),
		cm.StackWithName("alice"),
		cm.OverrideAssertionManager(assertionManager),
	)
	require.NoError(t, err)
	chalManager.Start(ctx)

	preState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	require.NoError(t, err)
	postState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 1, preState.GlobalState)
	require.NoError(t, err)

	time.Sleep(time.Second)

	posted, err := assertionManager.PostAssertion(ctx)
	require.NoError(t, err)
	require.Equal(t, true, posted.IsSome())
	creationInfo, err := aliceChain.ReadAssertionCreationInfo(ctx, posted.Unwrap().Id())
	require.NoError(t, err)
	require.Equal(t, postState, protocol.GoExecutionStateFromSolidity(creationInfo.AfterState))
}
