package validator

import (
	"bytes"
	"context"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	vertexAddedEventSig = hexutil.MustDecode("0x4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a675349")
	mergeEventSig       = hexutil.MustDecode("0x72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a")
	bisectEventSig      = hexutil.MustDecode("0x69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27")
)

func TestBlockChallenge(t *testing.T) {
	// Tests that validators are able to reach a one step fork correctly
	// by playing the challenge game on their own upon observing leaves
	// they disagree with. Here's the example with Alice and Bob.
	//
	//                [2]-[3]-[7]-alice
	//               /
	// [genesis]-[1]-
	//               \[2]-[3]-[7]-bob
	//
	t.Run("two validators opening leaves at same height", func(t *testing.T) {
		cfg := &blockChallengeTestConfig{
			numValidators:      2,
			currentChainHeight: 7,
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
			},
			latestHeightsByIndex: map[uint64]uint64{
				0: 7,
				1: 7,
			},
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob start diverging at height 3.
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 2,
				1: 2,
			},
		}
		// Alice adds a challenge leaf 6, is presumptive.
		// Bob adds leaf 6.
		// Bob bisects to 4, is presumptive.
		// Alice bisects to 4.
		// Alice bisects to 2, is presumptive.
		// Bob merges to 2.
		// Bob bisects from 4 to 3, is presumptive.
		// Alice bisects from 4 to 3.
		// Both challengers are now at a one-step fork, we now await subchallenge resolution.
		cfg.expectedVerticesAdded = 2
		cfg.expectedBisections = 5
		cfg.expectedMerges = 1
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 1")
		AssertLogsContain(t, hook, "Reached one-step-fork at 1")
	})
	t.Run("two validators opening leaves at same height, fork point is a power of two", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &blockChallengeTestConfig{
			numValidators:      2,
			currentChainHeight: 8,
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
			},
			latestHeightsByIndex: map[uint64]uint64{
				0: 8,
				1: 8,
			},
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob start diverging at height 3.
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 5,
				1: 5,
			},
		}
		cfg.expectedVerticesAdded = 2
		cfg.expectedBisections = 5
		cfg.expectedMerges = 1
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	})
	t.Run("two validators opening leaves at heights 6 and 256", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &blockChallengeTestConfig{
			numValidators:      2,
			currentChainHeight: 256,
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
			},
			latestHeightsByIndex: map[uint64]uint64{
				0: 6,
				1: 256,
			},
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 4,
				1: 4,
			},
		}
		// With Alice starting at 256 and bisecting all the way down to 4
		// will take 6 bisections. Then, Alice bisects from 4 to 3. Bob bisects twice to 4 and 2.
		// We should see a total of 9 bisections and 2 merges.
		cfg.expectedVerticesAdded = 2
		cfg.expectedBisections = 9
		cfg.expectedMerges = 2
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	})
	t.Run("two validators opening leaves at heights 129 and 256", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &blockChallengeTestConfig{
			numValidators:      2,
			currentChainHeight: 256,
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
			},
			latestHeightsByIndex: map[uint64]uint64{
				0: 129,
				1: 256,
			},
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 4,
				1: 4,
			},
		}
		// Same as the test case above but bob has 4 more bisections to perform
		// if Bob starts at 129.
		cfg.expectedVerticesAdded = 2
		cfg.expectedBisections = 14
		cfg.expectedMerges = 2
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	})
	//
	//                   [4]-[6]-alice
	//                  /
	// [genesis]-[2]-[3]-[4]-[6]-bob
	//                  \
	//                   [4]-[6]-charlie
	//
	t.Run("three validators opening leaves at same height same fork point", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &blockChallengeTestConfig{
			numValidators:      3,
			currentChainHeight: 6,
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
				2: "charlie",
			},
			latestHeightsByIndex: map[uint64]uint64{
				0: 6,
				1: 6,
				2: 6,
			},
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 4,
				1: 4,
				2: 4,
			},
		}
		cfg.expectedVerticesAdded = 3
		cfg.expectedBisections = 5
		cfg.expectedMerges = 4
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	})
	//
	//                   [4]-alice
	//                  /
	// [genesis]-[2]-[3]    -[6]-bob
	//                  \  /
	//                   [4]-[6]-charlie
	//
	t.Run("three validators opening leaves at same height different fork points", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &blockChallengeTestConfig{
			numValidators:      3,
			currentChainHeight: 6,
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
				2: "charlie",
			},
			latestHeightsByIndex: map[uint64]uint64{
				0: 6,
				1: 6,
				2: 6,
			},
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 3,
				1: 5,
				2: 5,
			},
		}
		cfg.expectedVerticesAdded = 3
		cfg.expectedBisections = 7
		cfg.expectedMerges = 2
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 2")
		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	})
	//
	//                   [3]-----------[6]--alice
	//                  /
	// [genesis]-[2]---------[4]--[5]--bob
	//                  \  /
	//                   [3]-[4]--[4]--charlie
	//
	t.Run("three validators opening leaves at different height different fork points", func(t *testing.T) {
		t.Skip("Flakey")
		cfg := &blockChallengeTestConfig{
			numValidators:      3,
			currentChainHeight: 64,
			latestHeightsByIndex: map[uint64]uint64{
				0: 6,
				1: 5,
				2: 5,
			},
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
				2: "charlie",
			},
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob agree up to and including height 3.
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 3,
				1: 4,
				2: 4,
			},
		}

		cfg.expectedVerticesAdded = 3
		cfg.expectedBisections = 6
		cfg.expectedMerges = 3
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 2")
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	})
}

type blockChallengeTestConfig struct {
	// Number of validators we want to enter a block challenge with.
	numValidators uint16
	// The heights at which each validator diverges histories.
	divergenceHeightsByIndex map[uint64]uint64
	// The latest heights by index
	latestHeightsByIndex map[uint64]uint64
	// Validator human-readable names by index.
	validatorNamesByIndex map[uint64]string
	currentChainHeight    uint64
	// Events we want to assert are fired from the goimpl.
	expectedBisections    uint64
	expectedMerges        uint64
	expectedVerticesAdded uint64
}

func runBlockChallengeTest(t testing.TB, hook *test.Hook, cfg *blockChallengeTestConfig) {
	require.Equal(t, true, cfg.numValidators > 1, "Need at least 2 validators")
	ctx := context.Background()
	tx := &solimpl.ActiveTx{ReadWriteTx: true}
	ref := util.NewRealTimeReference()
	chains, accs, addrs, backend := setupAssertionChains(t, uint64(cfg.numValidators)+1)
	prevInboxMaxCount := big.NewInt(1)

	// Advance the chain by 100 blocks as there needs to be a minimum period of time
	// before any assertions can be made on-chain.
	var honestBlockHash common.Hash
	for i := 0; i < 100; i++ {
		backend.Commit()
		//nolint:all
		honestBlockHash = backend.Commit()
	}

	// Initialize each validator's associated state roots which diverge
	var genesis protocol.Assertion
	err := chains[1].Call(func(tx protocol.ActiveTx) error {
		genesisAssertion, err := chains[1].AssertionBySequenceNum(ctx, tx, 0)
		if err != nil {
			return err
		}
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
	actualGenesisStateHash, err := genesis.StateHash()
	if err != nil {
		return
	}
	require.Equal(t, genesisStateHash, actualGenesisStateHash, "Genesis state hash unequal")

	// Initialize each validator associated state roots which diverge
	// at specified points in the test config.
	honestRoots := make([]common.Hash, cfg.currentChainHeight)
	honestStates := make([]*protocol.ExecutionState, cfg.currentChainHeight)
	honestInboxCounts := make([]*big.Int, cfg.currentChainHeight)
	honestRoots[0] = genesisStateHash
	honestStates[0] = genesisState
	honestInboxCounts[0] = big.NewInt(1)

	height := uint64(0)

	for i := uint64(1); i < cfg.currentChainHeight; i++ {
		height += 1
		honestBlockHash = backend.Commit()
		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestBlockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		honestRoots[i] = protocol.ComputeStateHash(state, prevInboxMaxCount)
		honestStates[i] = state
		honestInboxCounts[i] = big.NewInt(1)
	}

	vRoots := make([][]common.Hash, cfg.numValidators)
	vStates := make([][]*protocol.ExecutionState, cfg.numValidators)
	vInboxCounts := make([][]*big.Int, cfg.numValidators)

	// Creates validator states for each validator index where each index can diverge from
	// others at specific points. For example, with Alice, Bob, and Charlie,
	// All of them agree up to height 3, then Alice diverges starting at height 4 from Bob and Charlie.
	// Meanwhile, Bob and Charlie agree up until height 5, and start to diverge then.
	for i := uint64(0); i < uint64(cfg.numValidators); i++ {
		numRoots := cfg.latestHeightsByIndex[i] + 1
		divergenceHeight := cfg.divergenceHeightsByIndex[i]

		stateRoots := make([]common.Hash, numRoots)
		states := make([]*protocol.ExecutionState, numRoots)
		inboxCounts := make([]*big.Int, numRoots)

		for j := uint64(0); j < numRoots; j++ {
			if divergenceHeight == 0 || j < divergenceHeight {
				stateRoots[j] = honestRoots[j]
				states[j] = honestStates[j]
				inboxCounts[j] = honestInboxCounts[j]
			} else {
				junkRoot := make([]byte, 32)
				_, err := rand.Read(junkRoot)
				require.NoError(t, err)
				blockHash := crypto.Keccak256Hash(junkRoot)
				evilState := &protocol.ExecutionState{
					GlobalState: protocol.GoGlobalState{
						BlockHash: blockHash,
						Batch:     1,
					},
					MachineStatus: protocol.MachineStatusFinished,
				}
				stateRoots[j] = protocol.ComputeStateHash(evilState, prevInboxMaxCount)
				states[j] = evilState
				inboxCounts[j] = big.NewInt(1)
			}
		}
		vRoots[i] = stateRoots
		vStates[i] = states
		vInboxCounts[i] = inboxCounts
	}

	// Initialize each validator.
	validators := make([]*Validator, cfg.numValidators)
	for i := 0; i < len(validators); i++ {
		manager, err := statemanager.NewWithAssertionStates(vStates[i], vInboxCounts[i])
		require.NoError(t, err)
		addr := accs[i+1].accountAddr
		v, valErr := New(
			ctx,
			chains[i+1], // Chain 0 is reserved for admin
			backend,
			manager,
			addrs.Rollup,
			WithName(cfg.validatorNamesByIndex[uint64(i)]),
			WithAddress(addr),
			WithDisableLeafCreation(),
			WithTimeReference(ref),
			WithChallengeVertexWakeInterval(time.Millisecond*10),
			WithNewAssertionCheckInterval(time.Millisecond),
		)
		require.NoError(t, valErr)
		validators[i] = v
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// We fire off each validator's background routines.
	for _, val := range validators {
		go val.Start(ctx, tx)
	}

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

	time.Sleep(time.Millisecond * 100)

	// Submit leaf creation manually for each validator.
	for _, val := range validators {
		_, err := val.SubmitLeafCreation(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Submitted assertion")
	}

	wg.Wait()
	assert.Equal(t, cfg.expectedVerticesAdded, totalVertexAdded, "Did not get expected challenge leaf creations")
	assert.Equal(t, cfg.expectedBisections, totalBisections, "Did not get expected total bisections")
	assert.Equal(t, cfg.expectedMerges, totalMerges, "Did not get expected total merges")
}
