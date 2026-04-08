// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package assertions

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/challenge/types"
	"github.com/offchainlabs/nitro/bold/protocol"
	challenge_testing "github.com/offchainlabs/nitro/bold/testing"
	stateprovider "github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

// TestPostAssertionCatchesUpThroughExistingAssertions verifies that when a
// validator's assertion poster is behind the onchain state (e.g., assertions
// were posted by another validator), calling PostAssertion will advance through
// all existing assertions before posting a new one.
//
// This is a regression test for a bug where the poster would find that its
// computed assertion already existed onchain and return happily without
// advancing latestAgreedAssertion, causing it to get stuck in a loop.
func TestPostAssertionCatchesUpThroughExistingAssertions(t *testing.T) {
	ctx := context.Background()

	numExistingAssertions := 3

	cfg, err := setup.ChainsWithEdgeChallengeManager(
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

	chain := cfg.Chains[0]
	backend := cfg.Backend

	genesisHash, err := chain.GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisInfo, err := chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	stateManagerOpts := cfg.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(uint64(numExistingAssertions+2)),
	)
	stateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	// Post numExistingAssertions assertions directly using the chain,
	// simulating another validator that posted them while we were offline.
	// We add sequencer messages to the bridge for each subsequent batch.
	parentInfo := genesisInfo
	assertionHashes := make([]protocol.AssertionHash, 0, numExistingAssertions)
	var messageCount int64 = 1
	for i := 0; i < numExistingAssertions; i++ {
		if i > 0 {
			// Add a sequencer message so the next batch accumulator exists on the bridge.
			enqueueSeqMessage(
				t,
				cfg.Accounts[0].TxOpts,
				cfg.Addrs.UpgradeExecutor,
				backend,
				cfg.Addrs.Bridge,
				[32]byte{byte(i)},
				big.NewInt(1),
				big.NewInt(messageCount),
				big.NewInt(messageCount+1),
			)
			messageCount++
		}

		goGlobalState := protocol.GoGlobalStateFromSolidity(parentInfo.AfterState.GlobalState)
		execState, err := stateManager.ExecutionStateAfterPreviousState(
			ctx, parentInfo.InboxMaxCount.Uint64(), goGlobalState,
		)
		require.NoError(t, err)

		var assertion protocol.Assertion
		if i == 0 {
			assertion, err = chain.NewStakeOnNewAssertion(ctx, parentInfo, execState)
		} else {
			assertion, err = chain.StakeOnNewAssertion(ctx, parentInfo, execState)
		}
		require.NoError(t, err)

		info, err := chain.ReadAssertionCreationInfo(ctx, assertion.Id())
		require.NoError(t, err)
		assertionHashes = append(assertionHashes, assertion.Id())
		parentInfo = info
	}
	lastExistingInfo := parentInfo

	// Add one more sequencer message so the poster can post the next assertion.
	enqueueSeqMessage(
		t,
		cfg.Accounts[0].TxOpts,
		cfg.Addrs.UpgradeExecutor,
		backend,
		cfg.Addrs.Bridge,
		[32]byte{byte(numExistingAssertions)},
		big.NewInt(1),
		big.NewInt(messageCount),
		big.NewInt(messageCount+1),
	)

	// Create a Manager whose latestAgreedAssertion is at genesis,
	// simulating being behind by numExistingAssertions assertions.
	manager, err := NewManager(
		chain,
		stateManager,
		"catchup-test",
		types.DefensiveMode,
		WithDangerousReadyToPost(),
		WithMinimumGapToParentAssertion(0),
	)
	require.NoError(t, err)

	// Manually initialize the chain data to genesis (what syncAssertions would do).
	genesisAssertionHash := protocol.AssertionHash{Hash: genesisHash}
	manager.assertionChainData.latestAgreedAssertion = genesisAssertionHash
	manager.assertionChainData.canonicalAssertions[genesisAssertionHash] = genesisInfo

	// Verify we start at genesis.
	require.Equal(t, genesisAssertionHash, manager.assertionChainData.latestAgreedAssertion)

	// Call PostAssertion once. It should:
	// 1. Advance through all numExistingAssertions existing assertions
	// 2. Post a new (numExistingAssertions+1)th assertion
	posted, err := manager.PostAssertion(ctx)
	require.NoError(t, err)

	// Verify we advanced past genesis.
	require.NotEqual(t, genesisAssertionHash, manager.assertionChainData.latestAgreedAssertion,
		"latestAgreedAssertion should have advanced past genesis")

	// Verify all existing assertions are now in the canonical chain.
	for i, hash := range assertionHashes {
		_, ok := manager.assertionChainData.canonicalAssertions[hash]
		require.True(t, ok, "existing assertion %d (%s) should be in canonicalAssertions", i, hash)
	}

	// Verify we advanced past the last existing assertion (a new one was posted).
	require.NotEqual(t, lastExistingInfo.AssertionHash, manager.assertionChainData.latestAgreedAssertion,
		"should have posted a new assertion beyond the existing ones")

	// The returned assertion should be the newly posted one.
	require.True(t, posted.IsSome(), "PostAssertion should have returned the newly posted assertion")
	newAssertionHash := posted.Unwrap().Id()

	// The new assertion should not be any of the pre-existing ones.
	for _, hash := range assertionHashes {
		require.NotEqual(t, hash, newAssertionHash,
			"returned assertion should be new, not one of the pre-existing ones")
	}

	// latestAgreedAssertion should point to the newly posted assertion.
	require.Equal(t, newAssertionHash, manager.assertionChainData.latestAgreedAssertion)

	// Verify the total canonical chain length is correct:
	// genesis + numExistingAssertions + 1 newly posted
	require.Equal(t, numExistingAssertions+2, len(manager.assertionChainData.canonicalAssertions))
}

// enqueueSeqMessage adds a sequencer message directly to the bridge via the
// upgrade executor. This matches the pattern used in manager_test.go's
// enqueueSequencerMessageAsExecutor.
func enqueueSeqMessage(
	t *testing.T,
	opts *bind.TransactOpts,
	executor common.Address,
	backend *setup.SimulatedBackendWrapper,
	bridge common.Address,
	dataHash [32]byte,
	afterDelayedMessagesRead *big.Int,
	prevMessageCount *big.Int,
	newMessageCount *big.Int,
) {
	t.Helper()
	execBindings, err := mocksgen.NewUpgradeExecutorMock(executor, backend)
	require.NoError(t, err)
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.AbsBridgeABI))
	require.NoError(t, err)
	data, err := seqInboxABI.Pack("setSequencerInbox", executor)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, data)
	require.NoError(t, err)
	backend.Commit()

	data, err = seqInboxABI.Pack(
		"enqueueSequencerMessage",
		dataHash, afterDelayedMessagesRead, prevMessageCount, newMessageCount,
	)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, data)
	require.NoError(t, err)
	backend.Commit()
}
