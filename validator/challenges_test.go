package validator

import (
	"bytes"
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"encoding/binary"
	"math"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
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
	// TODO: These are brittle and could break if the event sigs change in Solidity.
	vertexAddedEventSig = hexutil.MustDecode("0x4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a675349")
	mergeEventSig       = hexutil.MustDecode("0x72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a")
	bisectEventSig      = hexutil.MustDecode("0x69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27")
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
			assertionDivergenceHeight:    3,
			numBigStepsAtAssertionHeight: 7,
			bigStepDivergenceHeight:      3,
			numSmallStepsAtBigStep:       7,
			smallStepDivergenceHeight:    3,
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
		cfg.expectedVerticesAdded = 6 // TODO: Rename to leaf
		cfg.expectedBisections = 12
		cfg.expectedMerges = 6
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 2")
		AssertLogsContain(t, hook, "Reached one-step-fork at 2")
		AssertLogsContain(t, hook, "Checking one-step-proof against protocol")
	})
	t.Run("two validators opening leaves at same height, fork point is a power of two", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &challengeProtocolTestConfig{
			currentChainHeight:        8,
			aliceHeight:               8,
			bobHeight:                 8,
			assertionDivergenceHeight: 5,
		}
		cfg.expectedVerticesAdded = 2
		cfg.expectedBisections = 5
		cfg.expectedMerges = 1
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	})
	t.Run("two validators opening leaves at heights 6 and 256", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &challengeProtocolTestConfig{
			currentChainHeight:        256,
			aliceHeight:               6,
			bobHeight:                 256,
			assertionDivergenceHeight: 4,
		}
		// With Alice starting at 256 and bisecting all the way down to 4
		// will take 6 bisections. Then, Alice bisects from 4 to 3. Bob bisects twice to 4 and 2.
		// We should see a total of 9 bisections and 2 merges.
		cfg.expectedVerticesAdded = 2
		cfg.expectedBisections = 9
		cfg.expectedMerges = 2
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	})
	t.Run("two validators opening leaves at heights 129 and 256", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &challengeProtocolTestConfig{
			currentChainHeight:        256,
			aliceHeight:               129,
			bobHeight:                 256,
			assertionDivergenceHeight: 4,
		}
		// Same as the test case above but bob has 4 more bisections to perform
		// if Bob starts at 129.
		cfg.expectedVerticesAdded = 2
		cfg.expectedBisections = 14
		cfg.expectedMerges = 2
		hook := test.NewGlobal()
		runChallengeIntegrationTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
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
	expectedBisections    uint64
	expectedMerges        uint64
	expectedVerticesAdded uint64
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
	var genesis protocol.Assertion
	err := chain.Call(func(tx protocol.ActiveTx) error {
		genesisAssertion, err := chain.AssertionBySequenceNum(ctx, tx, 0)
		require.NoError(t, err)
		genesis = genesisAssertion
		return nil
	})
	require.NoError(t, err)

	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	genesisStateHash := protocol.ComputeStateHash(genesisState, prevInboxMaxCount)
	require.Equal(t, genesisStateHash, genesis.StateHash(), "Genesis state hash unequal")

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

func runChallengeIntegrationTest(t testing.TB, hook *test.Hook, cfg *challengeProtocolTestConfig) {
	ctx := context.Background()
	ref := util.NewRealTimeReference()
	chains, accs, addrs, backend := setupAssertionChains(t, 3) // 0th is admin chain.
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
		chains[1],
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
	aliceAddr := accs[1].accountAddr
	alice, err := New(
		ctx,
		chains[1], // Chain 0 is reserved for admin controls.
		backend,
		honestManager,
		addrs.Rollup,
		WithName("alice"),
		WithAddress(aliceAddr),
		WithDisableLeafCreation(),
		WithTimeReference(ref),
		WithChallengeVertexWakeInterval(time.Millisecond*10),
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
	bobAddr := accs[2].accountAddr
	bob, err := New(
		ctx,
		chains[2], // Chain 0 is reserved for admin controls.
		backend,
		maliciousManager,
		addrs.Rollup,
		WithName("bob"),
		WithAddress(bobAddr),
		WithDisableLeafCreation(),
		WithTimeReference(ref),
		WithChallengeVertexWakeInterval(time.Millisecond*10),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// We fire off each validator's background routines.
	go alice.Start(ctx)
	go bob.Start(ctx)

	var managerAddr common.Address
	err = chains[1].Call(func(tx protocol.ActiveTx) error {
		manager, err := chains[1].CurrentChallengeManager(ctx, tx)
		require.NoError(t, err)
		managerAddr = manager.Address()
		return nil
	})
	require.NoError(t, err)

	var totalVertexAdded uint64
	var totalBisections uint64
	var totalMerges uint64
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
				case bytes.Equal(topic[:], vertexAddedEventSig):
					totalVertexAdded++
				case bytes.Equal(topic[:], bisectEventSig):
					totalBisections++
				case bytes.Equal(topic[:], mergeEventSig):
					totalMerges++
				default:
				}
			}
		}
	}()

	time.Sleep(time.Second * 2)

	// Submit leaf creation manually for each validator.
	_, err = alice.SubmitLeafCreation(ctx)
	require.NoError(t, err)
	_, err = bob.SubmitLeafCreation(ctx)
	require.NoError(t, err)
	AssertLogsContain(t, hook, "Submitted assertion")

	wg.Wait()
	assert.Equal(t, cfg.expectedVerticesAdded, totalVertexAdded, "Did not get expected challenge leaf creations")
	assert.Equal(t, cfg.expectedBisections, totalBisections, "Did not get expected total bisections")
	assert.Equal(t, cfg.expectedMerges, totalMerges, "Did not get expected total merges")
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
