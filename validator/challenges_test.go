package validator

import (
	"bytes"
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	bisectEventSig = crypto.Keccak256Hash([]byte("EdgeBisected(bytes32,bytes32,bytes32,bool)"))
	addedEventSig  = crypto.Keccak256Hash([]byte("EdgeAdded(bytes32,bytes32,bytes32,bytes32,uint256,uint8,bool,bool)"))
)

func TestChallengeProtocol_AliceAndBob(t *testing.T) {
	// Tests that validators are able to reach a one step fork correctly
	// by playing the challenge game on their own upon observing leaves
	// they disagree with. Here's the example with Alice and Bob, in which
	// they narrow down their disagreement to a single WAVM opcode
	// in a small step subchallenge. In this first test, Alice will be the honest
	// validator and will be able to resolve a challenge via a one-step-proof.
	//
	// At the assertion chain level, the fork is at height 2.
	//
	//                [3]-[7]-alice
	//               /
	// [genesis]-[2]-
	//               \
	//                [3]-[7]-bob
	//
	// At the big step challenge level, the fork is at height 2 (big step number 2).
	//
	//                    [3]-[7]-alice
	//                   /
	// [big_step_root]-[2]
	//                   \
	//                    [3]-[7]-bob
	//
	//
	// At the small step challenge level the fork is at height 2 (wavm opcode 2).
	//
	//                      [3]-[7]-alice
	//                     /
	// [small_step_root]-[2]
	//                     \
	//                      [3]-[7]-bob
	//
	t.Run("two forked assertions at the same height", func(t *testing.T) {
		cfg := &challengeProtocolTestConfig{
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob start diverging at height 3 at all subchallenge levels.
			assertionDivergenceHeight: 4,
			bigStepDivergenceHeight:   4,
			smallStepDivergenceHeight: 4,
		}
		cfg.expectedLeavesAdded = 61
		cfg.expectedBisections = 30
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Succeeded one-step-proof for edge and confirmed it as winner")
	})
	t.Run("two validators disagreeing on the number of blocks", func(t *testing.T) {
		cfg := &challengeProtocolTestConfig{
			assertionDivergenceHeight:      7,
			assertionBlockHeightDifference: 1,
			bigStepDivergenceHeight:        4,
			smallStepDivergenceHeight:      4,
		}
		cfg.expectedLeavesAdded = 61
		cfg.expectedBisections = 30
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Succeeded one-step-proof for edge and confirmed it as winner")
	})
}

type challengeProtocolTestConfig struct {
	// The height in the assertion chain at which the validators diverge.
	assertionDivergenceHeight uint64
	// The difference between the malicious assertion block height and the honest assertion block height.
	assertionBlockHeightDifference int64
	// The heights at which the validators diverge in histories at the big step
	// subchallenge level.
	bigStepDivergenceHeight uint64
	// The heights at which the validators diverge in histories at the small step
	// subchallenge level.
	smallStepDivergenceHeight uint64
	// Events we want to assert are fired from the goimpl.
	expectedBisections  uint64
	expectedLeavesAdded uint64
}

func runChallengeIntegrationTest(t *testing.T, _ *test.Hook, cfg *challengeProtocolTestConfig) {
	t.Helper()
	ctx := context.Background()
	ref := util.NewRealTimeReference()
	setupCfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	chains := setupCfg.Chains
	accs := setupCfg.Accounts
	addrs := setupCfg.Addrs
	backend := setupCfg.Backend
	prevInboxMaxCount := big.NewInt(1)

	// Advance the chain by 100 blocks as there needs to be a minimum period of time
	// before any assertions can be made on-chain.
	for i := 0; i < 100; i++ {
		backend.Commit()
	}

	// Initialize each validator.
	honestManager, err := statemanager.NewForSimpleMachine()
	require.NoError(t, err)
	aliceAddr := accs[1].AccountAddr
	alice, err := New(
		ctx,
		chains[0],
		backend,
		honestManager,
		addrs.Rollup,
		WithName("alice"),
		WithAddress(aliceAddr),
		WithTimeReference(ref),
		WithEdgeTrackerWakeInterval(time.Millisecond*100),
		WithNewAssertionCheckInterval(time.Millisecond*50),
	)
	require.NoError(t, err)

	maliciousManager, err := statemanager.NewForSimpleMachine(
		statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*protocol.LevelZeroSmallStepEdgeHeight+cfg.smallStepDivergenceHeight),
		statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight),
		statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
	)
	require.NoError(t, err)
	bobAddr := accs[1].AccountAddr
	bob, err := New(
		ctx,
		chains[1], // Chain 0 is reserved for admin controls.
		backend,
		maliciousManager,
		addrs.Rollup,
		WithName("bob"),
		WithAddress(bobAddr),
		WithTimeReference(ref),
		WithEdgeTrackerWakeInterval(time.Millisecond*100),
		WithNewAssertionCheckInterval(time.Millisecond*50),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	challengeManager, err := chains[1].SpecChallengeManager(ctx)
	require.NoError(t, err)
	managerAddr := challengeManager.Address()

	var totalLeavesAdded uint64
	var totalBisections uint64
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		logs := make(chan types.Log, 100)
		query := ethereum.FilterQuery{
			Addresses: []common.Address{managerAddr},
		}
		sub, err := backend.SubscribeFilterLogs(ctx, query, logs)
		require.NoError(t, err)
		defer sub.Unsubscribe()
		for {
			select {
			case err := <-sub.Err():
				t.Error(err)
				return
			case <-ctx.Done():
				return
			case vLog := <-logs:
				if len(vLog.Topics) == 0 {
					continue
				}
				topic := vLog.Topics[0]
				switch {
				case bytes.Equal(topic[:], addedEventSig.Bytes()):
					totalLeavesAdded++
				case bytes.Equal(topic[:], bisectEventSig.Bytes()):
					totalBisections++
				default:
				}
			}
		}
	}()

	genesisCreation, err := alice.chain.ReadAssertionCreationInfo(ctx, protocol.GenesisAssertionSeqNum)
	require.NoError(t, err)

	// Submit leaf creation manually for each validator.
	genesisState := protocol.GoExecutionStateFromSolidity(genesisCreation.AfterState)
	latestHonest, err := honestManager.LatestExecutionState(ctx)
	require.NoError(t, err)
	leaf1, err := alice.chain.CreateAssertion(
		ctx,
		genesisState,
		latestHonest,
		genesisCreation.InboxMaxCount,
	)
	require.NoError(t, err)

	latestEvil, err := maliciousManager.LatestExecutionState(ctx)
	require.NoError(t, err)
	leaf2, err := bob.chain.CreateAssertion(
		ctx,
		genesisState,
		latestEvil,
		genesisCreation.InboxMaxCount,
	)
	require.NoError(t, err)

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, prefixProof []byte, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
			prefixProof,
		)
		require.NoError(t, err)
		return edge
	}

	honestStartCommit, err := honestManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight, 1)
	require.NoError(t, err)
	honestPrefixProof, err := honestManager.PrefixProofUpToBatch(ctx, 0, 0, protocol.LevelZeroBlockEdgeHeight, 1)
	require.NoError(t, err)

	t.Log("Alice creates level zero block edge")

	honestEdge := leafAdder(honestStartCommit, honestEndCommit, honestPrefixProof, leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	evilStartCommit, err := maliciousManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := maliciousManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LevelZeroBlockEdgeHeight, 1)
	require.NoError(t, err)
	evilPrefixProof, err := maliciousManager.PrefixProofUpToBatch(ctx, 0, 0, protocol.LevelZeroBlockEdgeHeight, 1)
	require.NoError(t, err)

	t.Log("Bob creates level zero block edge")

	evilEdge := leafAdder(evilStartCommit, evilEndCommit, evilPrefixProof, leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	aliceTracker, err := newEdgeTracker(
		ctx,
		&edgeTrackerConfig{
			timeRef:          alice.timeRef,
			actEveryNSeconds: alice.edgeTrackerWakeInterval,
			chain:            alice.chain,
			stateManager:     alice.stateManager,
			validatorName:    alice.name,
			validatorAddress: alice.address,
		},
		honestEdge,
		0,
		prevInboxMaxCount.Uint64(),
	)
	require.NoError(t, err)

	bobTracker, err := newEdgeTracker(
		ctx,
		&edgeTrackerConfig{
			timeRef:          bob.timeRef,
			actEveryNSeconds: bob.edgeTrackerWakeInterval,
			chain:            bob.chain,
			stateManager:     bob.stateManager,
			validatorName:    bob.name,
			validatorAddress: bob.address,
		},
		evilEdge,
		0,
		prevInboxMaxCount.Uint64(),
	)
	require.NoError(t, err)

	go aliceTracker.spawn(ctx)
	go bobTracker.spawn(ctx)

	wg.Wait()
	assert.Equal(t, cfg.expectedLeavesAdded, totalLeavesAdded, "Did not get expected challenge leaf creations")
	assert.Equal(t, cfg.expectedBisections, totalBisections, "Did not get expected total bisections")
}
