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
			currentChainHeight: 7,
			// The latest assertion height each validator has seen.
			aliceHeight: 7,
			bobHeight:   7,
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob start diverging at height 3 at all subchallenge levels.
			assertionDivergenceHeight:    4,
			numBigStepsAtAssertionHeight: 7,
			bigStepDivergenceHeight:      4,
			numSmallStepsAtBigStep:       7,
			smallStepDivergenceHeight:    4,
		}
		// At each challenge level:
		// Alice adds a challenge leaf 7, is presumptive.
		// Bob adds leaf 7.
		// Bob bisects to 3, is presumptive.
		// Alice bisects to 3.
		// Alice bisects to 1, is presumptive.
		// Bob merges to 1.
		// Bob bisects from 3 to 2, is presumptive.
		// Alice merges from 3 to 2.
		// Both challengers are now at a one-step fork, we now await subchallenge resolution.
		cfg.expectedLeavesAdded = 10
		cfg.expectedBisections = 24
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at start height 3")
		AssertLogsContain(t, hook, "Checking one-step-proof against protocol")
	})
	t.Run("two validators opening leaves at height 255", func(t *testing.T) {
		cfg := &challengeProtocolTestConfig{
			currentChainHeight:           255,
			aliceHeight:                  255,
			bobHeight:                    255,
			assertionDivergenceHeight:    4,
			numBigStepsAtAssertionHeight: 7,
			bigStepDivergenceHeight:      4,
			numSmallStepsAtBigStep:       7,
			smallStepDivergenceHeight:    4,
		}
		cfg.expectedLeavesAdded = 20
		cfg.expectedBisections = 44
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at start height 3")
		AssertLogsContain(t, hook, "Checking one-step-proof against protocol")
	})
}

type challengeProtocolTestConfig struct {
	// The latest heights by index at the assertion chain level.
	aliceHeight uint64
	bobHeight   uint64
	// The height in the assertion chain at which the validators diverge.
	assertionDivergenceHeight uint64
	// The number of big steps of WAVM opcodes at the one-step-fork point in a test.
	numBigStepsAtAssertionHeight uint64
	// The heights at which the validators diverge in histories at the big step
	// subchallenge level.
	bigStepDivergenceHeight uint64
	// The number of WAVM opcodes (small steps) at the one-step-fork point of a big step
	// subchallenge in a test.
	numSmallStepsAtBigStep uint64
	// The heights at which the validators diverge in histories at the small step
	// subchallenge level.
	smallStepDivergenceHeight uint64
	currentChainHeight        uint64
	// Events we want to assert are fired from the goimpl.
	expectedBisections  uint64
	expectedLeavesAdded uint64
}

func prepareHonestStates(
	t testing.TB,
	ctx context.Context,
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
				BlockHash: honestHashes[i],
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		honestStates[i] = state
		honestInboxCounts[i] = big.NewInt(1)
	}
	return honestStates, honestInboxCounts
}

func prepareMaliciousStates(
	t testing.TB,
	cfg *challengeProtocolTestConfig,
	evilHashes []common.Hash,
	honestStates []*protocol.ExecutionState,
	honestInboxCounts []*big.Int,
	prevInboxMaxCount *big.Int,
) ([]*protocol.ExecutionState, []*big.Int) {
	divergenceHeight := cfg.assertionDivergenceHeight
	numRoots := cfg.bobHeight + 1
	states := make([]*protocol.ExecutionState, numRoots)
	inboxCounts := make([]*big.Int, numRoots)

	for j := uint64(0); j < numRoots; j++ {
		if divergenceHeight == 0 || j < divergenceHeight {
			states[j] = honestStates[j]
			inboxCounts[j] = honestInboxCounts[j]
		} else {
			evilState := &protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash: evilHashes[j],
					Batch:     1,
				},
				MachineStatus: protocol.MachineStatusFinished,
			}
			states[j] = evilState
			inboxCounts[j] = big.NewInt(1)
		}
	}
	return states, inboxCounts
}

func runChallengeIntegrationTest(t *testing.T, hook *test.Hook, cfg *challengeProtocolTestConfig) {
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

	honestHashes := honestHashesForUints(0, cfg.currentChainHeight+1)
	evilHashes := evilHashesForUints(0, cfg.currentChainHeight+1)
	require.Equal(t, len(honestHashes), len(evilHashes))

	honestStates, honestInboxCounts := prepareHonestStates(
		t,
		ctx,
		chains[0],
		backend,
		honestHashes,
		cfg.currentChainHeight,
		prevInboxMaxCount,
	)

	maliciousStates, maliciousInboxCounts := prepareMaliciousStates(
		t,
		cfg,
		evilHashes,
		honestStates,
		honestInboxCounts,
		prevInboxMaxCount,
	)

	// Initialize each validator.
	honestManager, err := statemanager.NewWithAssertionStates(
		honestStates,
		honestInboxCounts,
		statemanager.WithMaxWavmOpcodesPerBlock(49), // TODO(RJ): Configure.
		statemanager.WithNumOpcodesPerBigStep(7),
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
		statemanager.WithMaxWavmOpcodesPerBlock(49), // TODO(RJ): Configure.
		statemanager.WithNumOpcodesPerBigStep(7),
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
				log.Fatal(err)
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
	leaf1, err := alice.chain.CreateAssertion(
		ctx,
		latestHonest.Height,
		1,
		latestHonest.PreState,
		latestHonest.PostState,
		latestHonest.InboxMaxCount,
	)
	require.NoError(t, err)

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

	// Honest assertion being added.
	leafAdder := func(startCommit, endCommit util.HistoryCommitment, leaf protocol.Assertion) protocol.SpecEdge {
		edge, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			leaf,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return edge
	}

	honestStartCommit, err := honestManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	honestEndCommit, err := honestManager.HistoryCommitmentUpTo(ctx, latestHonest.Height)
	require.NoError(t, err)

	t.Log("Alice creates level zero block edge")

	honestEdge := leafAdder(honestStartCommit, honestEndCommit, leaf1)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())

	hasRival, err := honestEdge.HasRival(ctx)
	require.NoError(t, err)
	require.Equal(t, true, !hasRival)

	evilStartCommit, err := maliciousManager.HistoryCommitmentUpTo(ctx, 0)
	require.NoError(t, err)
	evilEndCommit, err := maliciousManager.HistoryCommitmentUpTo(ctx, uint64(latestEvil.Height))
	require.NoError(t, err)

	t.Log("Bob creates level zero block edge")

	evilEdge := leafAdder(evilStartCommit, evilEndCommit, leaf2)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	aliceTracker, err := newEdgeTracker(
		&edgeTrackerConfig{
			timeRef:          alice.timeRef,
			actEveryNSeconds: alice.edgeTrackerWakeInterval,
			chain:            alice.chain,
			stateManager:     alice.stateManager,
			validatorName:    alice.name,
			validatorAddress: alice.address,
		},
		honestEdge,
	)
	require.NoError(t, err)

	bobTracker, err := newEdgeTracker(
		&edgeTrackerConfig{
			timeRef:          bob.timeRef,
			actEveryNSeconds: bob.edgeTrackerWakeInterval,
			chain:            bob.chain,
			stateManager:     bob.stateManager,
			validatorName:    bob.name,
			validatorAddress: bob.address,
		},
		evilEdge,
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
