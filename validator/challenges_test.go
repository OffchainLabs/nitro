package validator

import (
	"context"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestBlockChallenge(t *testing.T) {
	// Tests that validators are able to reach a one step fork correctly
	// by playing the challenge game on their own upon observing leaves
	// they disagree with. Here's the example with Alice and Bob.
	//
	//                   [4]-[6]-alice
	//                  /
	// [genesis]-[2]-[3]
	//                  \[4]-[6]-bob
	//
	t.Run("two validators opening leaves at same height", func(t *testing.T) {
		cfg := &blockChallengeTestConfig{
			numValidators: 2,
			latestHeight:  6,
			validatorNamesByIndex: map[uint64]string{
				0: "alice",
				1: "bob",
			},
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob agree up to and including height 3.
			divergenceHeightsByIndex: map[uint64]uint64{
				0: 3,
				1: 3,
			},
		}
		// Alice adds a challenge leaf 6, is presumptive.
		// Bob adds leaf 6.
		// Bob bisects to 4, is presumptive.
		// Alice bisects to 4.
		// Alice bisects to 2, is presumptive.
		// Bob merges to 2.
		// Bob bisects from 4 to 3, is presumptive.
		// Alice merges to 3.
		// Both challengers are now at a one-step fork, we now await subchallenge resolution.
		cfg.eventsToAssert = map[protocol.ChallengeEvent]uint{
			&protocol.ChallengeLeafEvent{}:   2,
			&protocol.ChallengeBisectEvent{}: 4,
			&protocol.ChallengeMergeEvent{}:  2,
		}
		hook := test.NewGlobal()
		runBlockChallengeTest(t, hook, cfg)
		AssertLogsContain(t, hook, "Reached one-step-fork at 2")
		AssertLogsContain(t, hook, "Reached one-step-fork at 2")
	})
	// 	t.Run("two validators opening leaves at same height, fork point is a power of two", func(t *testing.T) {
	// 		aliceAddr := common.BytesToAddress([]byte{1})
	// 		bobAddr := common.BytesToAddress([]byte{2})
	// 		cfg := &blockChallengeTestConfig{
	// 			numValidators: 2,
	// 			latestStateHeightByAddress: map[common.Address]uint64{
	// 				aliceAddr: 6,
	// 				bobAddr:   6,
	// 			},
	// 			validatorAddrs: []common.Address{aliceAddr, bobAddr},
	// 			validatorNamesByAddress: map[common.Address]string{
	// 				aliceAddr: "alice",
	// 				bobAddr:   "bob",
	// 			},
	// 			// The heights at which the validators diverge in histories. In this test,
	// 			// alice and bob agree up to and including height 3.
	// 			divergenceHeightsByAddress: map[common.Address]uint64{
	// 				aliceAddr: 4,
	// 				bobAddr:   4,
	// 			},
	// 		}
	// 		cfg.eventsToAssert = map[goimpl.ChallengeEvent]uint{
	// 			&goimpl.ChallengeLeafEvent{}:   2,
	// 			&goimpl.ChallengeBisectEvent{}: 3,
	// 			&goimpl.ChallengeMergeEvent{}:  1,
	// 		}
	// 		hook := test.NewGlobal()
	// 		runBlockChallengeTest(t, hook, cfg)
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	// 	})
	// 	t.Run("two validators opening leaves at heights 6 and 256", func(t *testing.T) {
	// 		aliceAddr := common.BytesToAddress([]byte{1})
	// 		bobAddr := common.BytesToAddress([]byte{2})
	// 		cfg := &blockChallengeTestConfig{
	// 			numValidators: 2,
	// 			latestStateHeightByAddress: map[common.Address]uint64{
	// 				aliceAddr: 256,
	// 				bobAddr:   6,
	// 			},
	// 			validatorAddrs: []common.Address{aliceAddr, bobAddr},
	// 			validatorNamesByAddress: map[common.Address]string{
	// 				aliceAddr: "alice",
	// 				bobAddr:   "bob",
	// 			},
	// 			// The heights at which the validators diverge in histories. In this test,
	// 			// alice and bob agree up to and including height 3.
	// 			divergenceHeightsByAddress: map[common.Address]uint64{
	// 				aliceAddr: 3,
	// 				bobAddr:   3,
	// 			},
	// 		}
	// 		// With Alice starting at 256 and bisecting all the way down to 4
	// 		// will take 6 bisections. Then, Alice bisects from 4 to 3. Bob bisects twice to 4 and 2.
	// 		// We should see a total of 9 bisections and 2 merges.
	// 		cfg.eventsToAssert = map[goimpl.ChallengeEvent]uint{
	// 			&goimpl.ChallengeLeafEvent{}:   2,
	// 			&goimpl.ChallengeBisectEvent{}: 9,
	// 			&goimpl.ChallengeMergeEvent{}:  2,
	// 		}
	// 		hook := test.NewGlobal()
	// 		runBlockChallengeTest(t, hook, cfg)
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 	})
	// 	t.Run("two validators opening leaves at heights 129 and 256", func(t *testing.T) {
	// 		aliceAddr := common.BytesToAddress([]byte{1})
	// 		bobAddr := common.BytesToAddress([]byte{2})
	// 		cfg := &blockChallengeTestConfig{
	// 			numValidators: 2,
	// 			latestStateHeightByAddress: map[common.Address]uint64{
	// 				aliceAddr: 256,
	// 				bobAddr:   129,
	// 			},
	// 			validatorAddrs: []common.Address{aliceAddr, bobAddr},
	// 			validatorNamesByAddress: map[common.Address]string{
	// 				aliceAddr: "alice",
	// 				bobAddr:   "bob",
	// 			},
	// 			// The heights at which the validators diverge in histories. In this test,
	// 			// alice and bob agree up to and including height 3.
	// 			divergenceHeightsByAddress: map[common.Address]uint64{
	// 				aliceAddr: 3,
	// 				bobAddr:   3,
	// 			},
	// 		}
	// 		// Same as the test case above but bob has 4 more bisections to perform
	// 		// if Bob starts at 129.
	// 		cfg.eventsToAssert = map[goimpl.ChallengeEvent]uint{
	// 			&goimpl.ChallengeLeafEvent{}:   2,
	// 			&goimpl.ChallengeBisectEvent{}: 14,
	// 			&goimpl.ChallengeMergeEvent{}:  2,
	// 		}
	// 		hook := test.NewGlobal()
	// 		runBlockChallengeTest(t, hook, cfg)
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 	})
	// 	//
	// 	//                   [4]-[6]-alice
	// 	//                  /
	// 	// [genesis]-[2]-[3]-[4]-[6]-bob
	// 	//                  \
	// 	//                   [4]-[6]-charlie
	// 	//
	// 	t.Run("three validators opening leaves at same height same fork point", func(t *testing.T) {
	// 		aliceAddr := common.BytesToAddress([]byte{1})
	// 		bobAddr := common.BytesToAddress([]byte{2})
	// 		charlieAddr := common.BytesToAddress([]byte{3})
	// 		cfg := &blockChallengeTestConfig{
	// 			numValidators:  3,
	// 			validatorAddrs: []common.Address{aliceAddr, bobAddr, charlieAddr},
	// 			latestStateHeightByAddress: map[common.Address]uint64{
	// 				aliceAddr:   6,
	// 				bobAddr:     6,
	// 				charlieAddr: 6,
	// 			},
	// 			validatorNamesByAddress: map[common.Address]string{
	// 				aliceAddr:   "alice",
	// 				bobAddr:     "bob",
	// 				charlieAddr: "charlie",
	// 			},
	// 			// The heights at which the validators diverge in histories. In this test,
	// 			// alice and bob agree up to and including height 3.
	// 			divergenceHeightsByAddress: map[common.Address]uint64{
	// 				aliceAddr:   3,
	// 				bobAddr:     3,
	// 				charlieAddr: 3,
	// 			},
	// 		}
	// 		cfg.eventsToAssert = map[goimpl.ChallengeEvent]uint{
	// 			&goimpl.ChallengeLeafEvent{}:   3,
	// 			&goimpl.ChallengeBisectEvent{}: 5,
	// 			&goimpl.ChallengeMergeEvent{}:  4,
	// 		}
	// 		hook := test.NewGlobal()
	// 		runBlockChallengeTest(t, hook, cfg)
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 	})
	// 	//
	// 	//                   [4]-[6]-alice
	// 	//                  /
	// 	// [genesis]-[2]-[3]    -[6]-bob
	// 	//                  \  /
	// 	//                   [4]-[6]-charlie
	// 	//
	// 	t.Run("three validators opening leaves at same height different fork points", func(t *testing.T) {
	// 		aliceAddr := common.BytesToAddress([]byte{1})
	// 		bobAddr := common.BytesToAddress([]byte{2})
	// 		charlieAddr := common.BytesToAddress([]byte{3})
	// 		cfg := &blockChallengeTestConfig{
	// 			numValidators:  3,
	// 			validatorAddrs: []common.Address{aliceAddr, bobAddr, charlieAddr},
	// 			latestStateHeightByAddress: map[common.Address]uint64{
	// 				aliceAddr:   6,
	// 				bobAddr:     6,
	// 				charlieAddr: 6,
	// 			},
	// 			validatorNamesByAddress: map[common.Address]string{
	// 				aliceAddr:   "alice",
	// 				bobAddr:     "bob",
	// 				charlieAddr: "charlie",
	// 			},
	// 			// The heights at which the validators diverge in histories. In this test,
	// 			// alice and bob agree up to and including height 3.
	// 			divergenceHeightsByAddress: map[common.Address]uint64{
	// 				aliceAddr:   3,
	// 				bobAddr:     4,
	// 				charlieAddr: 4,
	// 			},
	// 		}

	// 		cfg.eventsToAssert = map[goimpl.ChallengeEvent]uint{
	// 			&goimpl.ChallengeLeafEvent{}:   3,
	// 			&goimpl.ChallengeBisectEvent{}: 6,
	// 			&goimpl.ChallengeMergeEvent{}:  3,
	// 		}
	// 		hook := test.NewGlobal()
	// 		runBlockChallengeTest(t, hook, cfg)
	// 		for _, entry := range hook.AllEntries() {
	// 			t.Log(entry.Message)
	// 		}
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	// 		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	// 	})
	// 	//
	// 	//                   [4]-----------------[64]-alice
	// 	//                  /
	// 	// [genesis]-[2]-[3]    -[6]-bob
	// 	//                  \  /
	// 	//                   [4]-[6]-charlie
	// 	//
	// 	t.Run("three validators opening leaves at different height different fork points", func(t *testing.T) {
	// 		aliceAddr := common.BytesToAddress([]byte{1})
	// 		bobAddr := common.BytesToAddress([]byte{2})
	// 		charlieAddr := common.BytesToAddress([]byte{3})
	// 		cfg := &blockChallengeTestConfig{
	// 			numValidators:  3,
	// 			validatorAddrs: []common.Address{aliceAddr, bobAddr, charlieAddr},
	// 			latestStateHeightByAddress: map[common.Address]uint64{
	// 				aliceAddr:   64,
	// 				bobAddr:     6,
	// 				charlieAddr: 6,
	// 			},
	// 			validatorNamesByAddress: map[common.Address]string{
	// 				aliceAddr:   "alice",
	// 				bobAddr:     "bob",
	// 				charlieAddr: "charlie",
	// 			},
	// 			// The heights at which the validators diverge in histories. In this test,
	// 			// alice and bob agree up to and including height 3.
	// 			divergenceHeightsByAddress: map[common.Address]uint64{
	// 				aliceAddr:   3,
	// 				bobAddr:     4,
	// 				charlieAddr: 4,
	// 			},
	// 		}

	//		cfg.eventsToAssert = map[goimpl.ChallengeEvent]uint{
	//			&goimpl.ChallengeLeafEvent{}:   3,
	//			&goimpl.ChallengeBisectEvent{}: 9,
	//			&goimpl.ChallengeMergeEvent{}:  3,
	//		}
	//		hook := test.NewGlobal()
	//		runBlockChallengeTest(t, hook, cfg)
	//		for _, entry := range hook.AllEntries() {
	//			t.Log(entry.Message)
	//		}
	//		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	//		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	//		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	//		AssertLogsContain(t, hook, "Reached one-step-fork at 4")
	//	})
}

type blockChallengeTestConfig struct {
	// Number of validators we want to enter a block challenge with.
	numValidators uint16
	// The heights at which each validator diverges histories.
	divergenceHeightsByIndex map[uint64]uint64
	// Validator human-readable names by index.
	validatorNamesByIndex map[uint64]string
	latestHeight          uint64
	// Events we want to assert are fired from the goimpl.
	eventsToAssert map[protocol.ChallengeEvent]uint
}

func runBlockChallengeTest(t testing.TB, hook *test.Hook, cfg *blockChallengeTestConfig) {
	require.Equal(t, true, cfg.numValidators > 1, "Need at least 2 validators")
	ctx := context.Background()
	ref := util.NewRealTimeReference()
	chains, accs, addrs, backend := setupAssertionChains(t, uint64(cfg.numValidators)+1)
	prevInboxMaxCount := big.NewInt(1)

	// Advance the chain by 100 blocks as there needs to be a minimum period of time
	// before any assertions can be made on-chain.
	honestBlockHash := common.Hash{}
	for i := 0; i < 100; i++ {
		backend.Commit()
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
	require.Equal(t, genesisStateHash, genesis.StateHash(), "Genesis state hash unequal")

	// Only one validator will have the correct result. We'll make that validator 0.
	height := uint64(0)
	validatorStateRoots := make([][]common.Hash, cfg.numValidators)
	validatorStates := make([][]*protocol.ExecutionState, cfg.numValidators)
	validatorInboxMaxCounts := make([][]*big.Int, cfg.numValidators)

	for i := 0; i < len(validatorStateRoots); i++ {
		validatorStateRoots[i] = append(validatorStateRoots[i], genesisStateHash)
		validatorStates[i] = append(validatorStates[i], genesisState)
		validatorInboxMaxCounts[i] = append(validatorInboxMaxCounts[i], big.NewInt(1))
	}

	for i := uint64(1); i < cfg.latestHeight; i++ {
		height += 1
		honestBlockHash = backend.Commit()

		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestBlockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		validatorStateRoots[0] = append(validatorStateRoots[0], protocol.ComputeStateHash(state, prevInboxMaxCount))
		validatorStates[0] = append(validatorStates[0], state)
		validatorInboxMaxCounts[0] = append(validatorInboxMaxCounts[0], big.NewInt(1))

		for j := uint64(1); j < uint64(cfg.numValidators); j++ {
			// Before the divergence height, the evil validator agrees.
			if uint64(i) < cfg.divergenceHeightsByIndex[j] {
				validatorStateRoots[j] = append(validatorStateRoots[j], protocol.ComputeStateHash(state, prevInboxMaxCount))
				validatorStates[j] = append(validatorStates[j], state)
				validatorInboxMaxCounts[j] = append(validatorInboxMaxCounts[j], big.NewInt(1))
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
				validatorStateRoots[j] = append(validatorStateRoots[j], protocol.ComputeStateHash(evilState, prevInboxMaxCount))
				validatorStates[j] = append(validatorStates[j], evilState)
				validatorInboxMaxCounts[j] = append(validatorInboxMaxCounts[j], big.NewInt(1))
			}
		}
	}

	// Initialize each validator.
	validators := make([]*Validator, cfg.numValidators)
	for i := 0; i < len(validators); i++ {
		manager := statemanager.NewWithExecutionStates(validatorStates[i], validatorInboxMaxCounts[i])
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
			WithChallengeVertexWakeInterval(time.Millisecond*100),
		)
		require.NoError(t, valErr)
		validators[i] = v
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*100)
	defer cancel()

	// We fire off each validator's background routines.
	for _, val := range validators {
		go val.Start(ctx)
	}

	time.Sleep(time.Second * 5)

	// Submit leaf creation manually for each validator.
	for _, val := range validators {
		go func(vv *Validator) {
			_, err := vv.SubmitLeafCreation(ctx)
			require.NoError(t, err)
			AssertLogsContain(t, hook, "Submitted assertion")
		}(val)
	}

	time.Sleep(time.Second * 5)

	// totalEventsWanted := uint16(0)
	// for _, count := range cfg.eventsToAssert {
	// 	totalEventsWanted += uint16(count)
	// }
	// totalEventsSeen := uint16(0)
	// seenEventCount := make(map[string]uint)
	// for ev := range harnessObserver {
	// 	if totalEventsSeen > totalEventsWanted {
	// 		t.Logf("Received more events than expected, saw an extra %+T", ev)
	// 	}
	// 	switch e := ev.(type) {
	// 	case *protocol.ChallengeLeafEvent:
	// 		fmt.Println("ChallengeLeafEvent")
	// 		fmt.Printf(
	// 			"validator=%s height=%d commit=%#x\n",
	// 			cfg.validatorNamesByAddress[ev.ValidatorAddress()],
	// 			e.History.Height,
	// 			e.History.Merkle,
	// 		)
	// 		fmt.Println("")
	// 	case *protocol.ChallengeMergeEvent:
	// 		fmt.Println("ChallengeMergeEvent")
	// 		fmt.Printf(
	// 			"validator=%s to=%d commit=%#x\n",
	// 			cfg.validatorNamesByAddress[ev.ValidatorAddress()],
	// 			e.ToHistory.Height,
	// 			e.ToHistory.Merkle,
	// 		)
	// 		fmt.Println("")
	// 	case *protocol.ChallengeBisectEvent:
	// 		fmt.Println("ChallengeBisectEvent")
	// 		fmt.Printf(
	// 			"validator=%s to=%d commit=%#x\n",
	// 			cfg.validatorNamesByAddress[ev.ValidatorAddress()],
	// 			e.ToHistory.Height,
	// 			e.ToHistory.Merkle,
	// 		)
	// 		fmt.Println("")
	// 	default:
	// 		fmt.Printf(
	// 			"Seen event %+T: %+v from validator %s\n",
	// 			ev,
	// 			ev,
	// 			cfg.validatorNamesByAddress[ev.ValidatorAddress()],
	// 		)
	// 	}
	// 	typ := fmt.Sprintf("%+T", ev)
	// 	seenEventCount[typ]++
	// 	totalEventsSeen++
	// }
	// for ev, wantedCount := range cfg.eventsToAssert {
	// 	_ = wantedCount
	// 	typ := fmt.Sprintf("%+T", ev)
	// 	seenCount, ok := seenEventCount[typ]
	// 	if !ok {
	// 		t.Logf("Wanted to see %+T event, but none received", ev)
	// 	}
	// 	_ = seenCount
	// 	require.Equal(
	// 		t,
	// 		wantedCount,
	// 		seenCount,
	// 		fmt.Sprintf("Did not see the expected number of %+T events", ev),
	// 	)
	// }
}

// func TestValidator_verifyAddLeafConditions(t *testing.T) {
// 	tx := &goimpl.ActiveTx{}
// 	badAssertion := &goimpl.Assertion{}
// 	ctx := context.Background()
// 	timeRef := util.NewArtificialTimeReference()
// 	v := &Validator{chain: goimpl.NewAssertionChain(ctx, timeRef, 100*time.Second)}
// 	// Can not add leaf on root assertion
// 	require.ErrorIs(t, v.verifyAddLeafConditions(ctx, tx, badAssertion, &goimpl.Challenge{}), goimpl.ErrInvalidOp)

// 	chain := goimpl.NewAssertionChain(ctx, timeRef, 100*time.Second)
// 	var chal goimpl.ChallengeInterface
// 	var rootAssertion *goimpl.Assertion
// 	var err error
// 	err = chain.Tx(func(tx *goimpl.ActiveTx) error {
// 		require.Equal(t, uint64(1), chain.NumAssertions(tx))
// 		rootAssertion, err = chain.AssertionBySequenceNum(tx, 0)
// 		require.NoError(t, err)
// 		chain.SetBalance(tx, common.Address{}, new(big.Int).Mul(goimpl.AssertionStake, big.NewInt(1000)))
// 		_, err = chain.CreateLeaf(tx, rootAssertion, util.StateCommitment{
// 			Height:    1,
// 			StateRoot: common.Hash{'a'},
// 		}, common.Address{})
// 		require.NoError(t, err)
// 		_, err = chain.CreateLeaf(tx, rootAssertion, util.StateCommitment{
// 			Height:    2,
// 			StateRoot: common.Hash{'b'},
// 		}, common.Address{})
// 		require.NoError(t, err)
// 		chal, err = rootAssertion.CreateChallenge(tx, ctx, common.Address{})
// 		require.NoError(t, err)
// 		// Parent missmatch between challenge and assertion's parent
// 		require.ErrorIs(t, v.verifyAddLeafConditions(ctx, tx, &goimpl.Assertion{Prev: util.Some[*goimpl.Assertion](badAssertion)}, chal), goimpl.ErrInvalidOp)

// 		// Happy case
// 		require.NoError(t, v.verifyAddLeafConditions(ctx, tx, &goimpl.Assertion{Prev: util.Some[*goimpl.Assertion](rootAssertion)}, chal), goimpl.ErrInvalidOp)
// 		return nil
// 	})
// 	require.NoError(t, err)
// }
