package validator

import (
	"context"
	"encoding/binary"
	"math"
	"math/big"
	"testing"
	"time"

	"bytes"
	"sync"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	bisectEventSig    = hexutil.MustDecode("0xddd14992ee7cd971b2a5cc510ebc7a33a1a7bd11dd74c3c5a83000328a0d5906")
	leafAddedEventSig = hexutil.MustDecode("0x7340510d24b7ec9b5c100f5500d93429d80d00d46f0d18e4e85d0c4cc22b9924")
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
			// The latest assertion height each validator has seen.
			aliceHeight: 7,
			bobHeight:   7,
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob start diverging at height 3 at all subchallenge levels.
			assertionDivergenceHeight: 4,
			bigStepDivergenceHeight:   4,
			smallStepDivergenceHeight: 4,
		}
		cfg.expectedLeavesAdded = 30
		cfg.expectedBisections = 60
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at start height 3")
		AssertLogsContain(t, hook, "Succeeded one-step-proof for edge and confirmed it as winner")
	})
	t.Run("two validators opening leaves at height 31", func(t *testing.T) {
		// TODO: we would use a larger height here but we're limited by protocol.LayerZeroBlockEdgeHeight
		cfg := &challengeProtocolTestConfig{
			aliceHeight:               31,
			bobHeight:                 31,
			assertionDivergenceHeight: 4,
			bigStepDivergenceHeight:   4,
			smallStepDivergenceHeight: 4,
		}
		cfg.expectedLeavesAdded = 30
		cfg.expectedBisections = 60
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at start height 3")
		AssertLogsContain(t, hook, "Succeeded one-step-proof for edge and confirmed it as winner")
	})
	t.Run("two validators disagreeing on the number of blocks", func(t *testing.T) {
		// TODO: we would use a larger height here but we're limited by protocol.LayerZeroBlockEdgeHeight
		cfg := &challengeProtocolTestConfig{
			aliceHeight:               7,
			bobHeight:                 8,
			assertionDivergenceHeight: 8,
			bigStepDivergenceHeight:   4,
			smallStepDivergenceHeight: 4,
		}
		cfg.expectedLeavesAdded = 30
		cfg.expectedBisections = 60
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at start height 3")
		AssertLogsContain(t, hook, "Succeeded one-step-proof for edge and confirmed it as winner")
	})
}

type challengeProtocolTestConfig struct {
	// The latest heights by index at the assertion chain level.
	aliceHeight uint64
	bobHeight   uint64
	// The height in the assertion chain at which the validators diverge.
	assertionDivergenceHeight uint64
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

func prepareHonestStates(
	ctx context.Context,
	t testing.TB,
	chain protocol.Protocol,
	backend *backends.SimulatedBackend,
	honestHashes []common.Hash,
	chainHeight uint64,
	prevInboxMaxCount *big.Int,
) ([]*protocol.ExecutionState, []*big.Int) {
	t.Helper()
	// Initialize each validator's associated state roots which diverge
	genesis, err := chain.AssertionBySequenceNum(ctx, 1)
	require.NoError(t, err)

	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	genesisStateHash := protocol.ComputeStateHash(genesisState, prevInboxMaxCount)
	actualGenesisStateHash, err := genesis.StateHash()
	require.NoError(t, err)

	require.Equal(t, genesisStateHash, actualGenesisStateHash, "Genesis state hash unequal")

	// Initialize each validator associated state roots which diverge
	// at specified points in the test config.
	honestStates := make([]*protocol.ExecutionState, chainHeight+1)
	honestInboxCounts := make([]*big.Int, chainHeight+1)
	honestStates[0] = genesisState
	honestInboxCounts[0] = big.NewInt(1)

	for i := uint64(1); i <= chainHeight; i++ {
		backend.Commit()
		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash:  honestHashes[i],
				Batch:      0,
				PosInBatch: i,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}
		if i == chainHeight {
			state.GlobalState.Batch = 1
			state.GlobalState.PosInBatch = 0
		}

		honestStates[i] = state
		honestInboxCounts[i] = big.NewInt(2)
	}
	return honestStates, honestInboxCounts
}

func prepareMaliciousStates(
	cfg *challengeProtocolTestConfig,
	evilHashes []common.Hash,
	honestStates []*protocol.ExecutionState,
	honestInboxCounts []*big.Int,
) ([]*protocol.ExecutionState, []*big.Int) {
	divergenceHeight := cfg.assertionDivergenceHeight
	numRoots := cfg.bobHeight + 1
	states := make([]*protocol.ExecutionState, numRoots)
	inboxCounts := make([]*big.Int, numRoots)

	for j := uint64(0); j < numRoots; j++ {
		if divergenceHeight == 0 || j < divergenceHeight {
			evilState := *honestStates[j]
			if j < cfg.bobHeight {
				evilState.GlobalState.Batch = 0
				evilState.GlobalState.PosInBatch = j
			}
			states[j] = &evilState
			inboxCounts[j] = honestInboxCounts[j]
		} else {
			evilState := &protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash:  evilHashes[j],
					Batch:      0,
					PosInBatch: j,
				},
				MachineStatus: protocol.MachineStatusFinished,
			}
			if j == cfg.bobHeight {
				evilState.GlobalState.Batch = 1
				evilState.GlobalState.PosInBatch = 0
			}
			states[j] = evilState
			inboxCounts[j] = big.NewInt(2)
		}
	}
	return states, inboxCounts
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

	honestHashes := honestHashesForUints(0, cfg.aliceHeight+1)
	evilHashes := evilHashesForUints(0, cfg.bobHeight+1)

	honestStates, honestInboxCounts := prepareHonestStates(
		ctx,
		t,
		chains[0],
		backend,
		honestHashes,
		cfg.aliceHeight,
		prevInboxMaxCount,
	)

	maliciousStates, maliciousInboxCounts := prepareMaliciousStates(
		cfg,
		evilHashes,
		honestStates,
		honestInboxCounts,
	)

	// Initialize each validator.
	honestManager, err := statemanager.NewWithAssertionStates(
		honestStates,
		honestInboxCounts,
		statemanager.WithNumOpcodesPerBigStep(protocol.LayerZeroSmallStepEdgeHeight),
		statemanager.WithMaxWavmOpcodesPerBlock(protocol.LayerZeroBigStepEdgeHeight*protocol.LayerZeroSmallStepEdgeHeight),
	)
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

	maliciousManager, err := statemanager.NewWithAssertionStates(
		maliciousStates,
		maliciousInboxCounts,
		statemanager.WithNumOpcodesPerBigStep(protocol.LayerZeroSmallStepEdgeHeight),
		statemanager.WithMaxWavmOpcodesPerBlock(protocol.LayerZeroBigStepEdgeHeight*protocol.LayerZeroSmallStepEdgeHeight),
		statemanager.WithBigStepStateDivergenceHeight(cfg.bigStepDivergenceHeight),
		statemanager.WithSmallStepStateDivergenceHeight(cfg.smallStepDivergenceHeight),
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
				case bytes.Equal(topic[:], leafAddedEventSig):
					totalLeavesAdded++
				case bytes.Equal(topic[:], bisectEventSig):
					totalBisections++
				default:
				}
			}
		}
	}()

	// Submit leaf creation manually for each validator.
	latestHonest, err := honestManager.LatestAssertionCreationData(ctx, 0)
	require.NoError(t, err)
	inboxMaxCount := big.NewInt(2)
	// TODO: this field is broken :/ see the comment in LatestAssertionCreationData
	//assert.Equal(t, latestHonest.InboxMaxCount, inboxSize, "honest assertion has an incorrect InboxMaxCount")
	leaf1, err := alice.chain.CreateAssertion(
		ctx,
		latestHonest.Height,
		1,
		latestHonest.PreState,
		latestHonest.PostState,
		latestHonest.InboxMaxCount,
	)
	require.NoError(t, err)
	leaf1State, err := leaf1.StateHash()
	require.NoError(t, err)
	expectedLeaf1State := protocol.ComputeStateHash(latestHonest.PostState, inboxMaxCount)
	assert.Equal(t, leaf1State, expectedLeaf1State, "created honest leaf1 with an unexpected state hash")

	latestEvil, err := maliciousManager.LatestAssertionCreationData(ctx, 0)
	require.NoError(t, err)
	leaf2, err := bob.chain.CreateAssertion(
		ctx,
		latestEvil.Height,
		1,
		latestEvil.PreState,
		latestEvil.PostState,
		latestEvil.InboxMaxCount,
	)
	require.NoError(t, err)
	leaf2State, err := leaf2.StateHash()
	require.NoError(t, err)
	expectedLeaf2State := protocol.ComputeStateHash(latestEvil.PostState, inboxMaxCount)
	assert.Equal(t, leaf2State, expectedLeaf2State, "created evil leaf2 with an unexpected state hash")

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
	honestEndCommit, err := honestManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LayerZeroBlockEdgeHeight, 1)
	require.NoError(t, err)
	honestPrefixProof, err := honestManager.PrefixProofUpToBatch(ctx, 0, 0, protocol.LayerZeroBlockEdgeHeight, 1)
	require.NoError(t, err)

	t.Log("Alice creates level zero block edge")

	honestEdge := leafAdder(honestStartCommit, honestEndCommit, honestPrefixProof, leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	evilStartCommit, err := maliciousManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := maliciousManager.HistoryCommitmentUpToBatch(ctx, 0, protocol.LayerZeroBlockEdgeHeight, 1)
	require.NoError(t, err)
	evilPrefixProof, err := maliciousManager.PrefixProofUpToBatch(ctx, 0, 0, protocol.LayerZeroBlockEdgeHeight, 1)
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

func evilHashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, hashForUint(math.MaxUint64-i))
	}
	return ret
}

func honestHashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, hashForUint(i))
	}
	return ret
}

func hashForUint(x uint64) common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, x))
}
