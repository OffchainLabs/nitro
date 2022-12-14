package validator

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	statemanager "github.com/OffchainLabs/new-rollup-exploration/state-manager"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
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
		aliceAddr := common.BytesToAddress([]byte{1})
		bobAddr := common.BytesToAddress([]byte{2})
		cfg := &blockChallengeTestConfig{
			numValidators: 2,
			latestStateHeightByAddress: map[common.Address]uint64{
				aliceAddr: 6,
				bobAddr:   6,
			},
			validatorAddrs: []common.Address{aliceAddr, bobAddr},
			validatorNamesByAddress: map[common.Address]string{
				aliceAddr: "alice",
				bobAddr:   "bob",
			},
			// The heights at which the validators diverge in histories. In this test,
			// alice and bob agree up to and including height 3.
			divergenceHeightsByAddress: map[common.Address]uint64{
				aliceAddr: 3,
				bobAddr:   3,
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
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
		AssertLogsContain(t, hook, "Reached one-step-fork at 3")
	})
}

type blockChallengeTestConfig struct {
	// Number of validators we want to enter a block challenge with.
	numValidators uint16
	// The heights at which each validator by address diverges histories.
	divergenceHeightsByAddress map[common.Address]uint64
	// Validator human-readable names by address.
	validatorNamesByAddress map[common.Address]string
	// Latest state height by address.
	latestStateHeightByAddress map[common.Address]uint64
	// List of validator addresses to initialize in order.
	validatorAddrs []common.Address
	// Events we want to assert are fired from the protocol.
	eventsToAssert map[protocol.ChallengeEvent]uint
}

func runBlockChallengeTest(t testing.TB, hook *test.Hook, cfg *blockChallengeTestConfig) {
	ctx := context.Background()
	ref := util.NewArtificialTimeReference()
	chain := protocol.NewAssertionChain(ctx, ref, time.Second)

	// Increase the balance for each validator in the test.
	bal := big.NewInt(0).Mul(protocol.Gwei, big.NewInt(100))
	err := chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		for addr := range cfg.validatorNamesByAddress {
			chain.AddToBalance(tx, addr, bal)
		}
		return nil
	})
	require.NoError(t, err)

	// Initialize each validator associated state roots which diverge
	// at specified points in the test config.
	validatorStateRoots := make([][]common.Hash, cfg.numValidators)
	for i := uint16(0); i < cfg.numValidators; i++ {
		addr := cfg.validatorAddrs[i]
		numRoots := cfg.latestStateHeightByAddress[addr] + 1
		divergenceHeight := cfg.divergenceHeightsByAddress[addr]
		stateRoots := make([]common.Hash, numRoots)
		for i := uint64(0); i < numRoots; i++ {
			if divergenceHeight == 0 || i < divergenceHeight {
				stateRoots[i] = util.HashForUint(i)
			} else {
				divergingRoot := make([]byte, 32)
				_, err = rand.Read(divergingRoot)
				require.NoError(t, err)
				stateRoots[i] = common.BytesToHash(divergingRoot)
			}
		}
		validatorStateRoots[i] = stateRoots
	}

	// Initialize each validator.
	validators := make([]*Validator, cfg.numValidators)
	for i := 0; i < len(validators); i++ {
		manager := statemanager.New(validatorStateRoots[i])
		addr := cfg.validatorAddrs[i]
		v, err2 := New(
			ctx,
			chain,
			manager,
			WithName(cfg.validatorNamesByAddress[addr]),
			WithAddress(addr),
			WithDisableLeafCreation(),
			WithTimeReference(ref),
		)
		require.NoError(t, err2)
		validators[i] = v
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	harnessObserver := make(chan protocol.ChallengeEvent, 100)
	chain.SubscribeChallengeEvents(ctx, harnessObserver)

	// Submit leaf creation manually for each validator.
	for _, val := range validators {
		_, err = val.submitLeafCreation(ctx)
		require.NoError(t, err)
		AssertLogsContain(t, hook, "Submitted leaf creation")
	}

	// We fire off each validator's background routines.
	for _, val := range validators {
		go val.Start(ctx)
	}

	// Sleep before reading events for cleaner logs below.
	time.Sleep(time.Second)

	totalEventsWanted := uint16(0)
	for _, count := range cfg.eventsToAssert {
		totalEventsWanted += uint16(count)
	}
	totalEventsSeen := uint16(0)
	seenEventCount := make(map[string]uint)
	for ev := range harnessObserver {
		if totalEventsSeen > totalEventsWanted {
			t.Fatalf("Received more events than expected, saw an extra %+T", ev)
		}
		switch e := ev.(type) {
		case *protocol.ChallengeLeafEvent:
			fmt.Println("ChallengeLeafEvent")
			fmt.Printf(
				"validator=%s height=%d commit=%#x\n",
				cfg.validatorNamesByAddress[ev.ValidatorAddress()],
				e.History.Height,
				e.History.Merkle,
			)
			fmt.Println("")
		case *protocol.ChallengeMergeEvent:
			fmt.Println("ChallengeMergeEvent")
			fmt.Printf(
				"validator=%s to=%d commit=%#x\n",
				cfg.validatorNamesByAddress[ev.ValidatorAddress()],
				e.History.Height,
				e.History.Merkle,
			)
			fmt.Println("")
		case *protocol.ChallengeBisectEvent:
			fmt.Println("ChallengeBisectEvent")
			fmt.Printf(
				"validator=%s to=%d commit=%#x\n",
				cfg.validatorNamesByAddress[ev.ValidatorAddress()],
				e.History.Height,
				e.History.Merkle,
			)
			fmt.Println("")
		default:
			fmt.Printf(
				"Seen event %+T: %+v from validator %s\n",
				ev,
				ev,
				cfg.validatorNamesByAddress[ev.ValidatorAddress()],
			)
		}
		typ := fmt.Sprintf("%+T", ev)
		seenEventCount[typ]++
		totalEventsSeen++
	}
	for ev, wantedCount := range cfg.eventsToAssert {
		typ := fmt.Sprintf("%+T", ev)
		seenCount, ok := seenEventCount[typ]
		if !ok {
			t.Fatalf("Wanted to see %+T event, but none received", ev)
		}
		require.Equal(
			t,
			wantedCount,
			seenCount,
			fmt.Sprintf("Did not see the expected number of %+T events", ev),
		)
	}
}
