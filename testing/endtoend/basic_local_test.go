package endtoend

import (
	"context"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	validator "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/endtoend/internal/backend"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type ChallengeScenario struct {
	Name string

	// Validator knowledge
	AliceStateManager   l2stateprovider.Provider
	BobStateManager     l2stateprovider.Provider
	CharlieStateManager l2stateprovider.Provider

	// Expectations
	Expectations []expect
}

func TestTotalWasmOpcodes(t *testing.T) {
	t.Run("2^43 production value", func(t *testing.T) {
		layerZeroHeights := &protocol.LayerZeroHeights{
			BlockChallengeHeight:     1 << 10,
			BigStepChallengeHeight:   1 << 10,
			SmallStepChallengeHeight: 1 << 13,
		}
		numBigSteps := uint8(3)
		require.Equal(t, uint64(1<<43), totalWasmOpcodes(layerZeroHeights, numBigSteps))
	})
	t.Run("minimal configuration", func(t *testing.T) {
		layerZeroHeights := &protocol.LayerZeroHeights{
			BlockChallengeHeight:     1 << 5,
			BigStepChallengeHeight:   1 << 5,
			SmallStepChallengeHeight: 1 << 5,
		}
		numBigSteps := uint8(1)
		require.Equal(t, uint64(1<<10), totalWasmOpcodes(layerZeroHeights, numBigSteps))
	})
}

func TestChallengeProtocol_AliceAndBob_AnvilLocal(t *testing.T) {
	be, err := backend.NewAnvilLocal(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if err := be.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := be.Stop(); err != nil {
			t.Logf("error stopping backend: %v", err)
		}
	}()

	layerZeroHeights := &protocol.LayerZeroHeights{
		BlockChallengeHeight:     1 << 5,
		BigStepChallengeHeight:   1 << 5,
		SmallStepChallengeHeight: 1 << 5,
	}
	numBigSteps := uint8(3)
	totalOpcodes := totalWasmOpcodes(layerZeroHeights, numBigSteps)

	// Diverge exactly at the halfway opcode within the block.
	machineDivergenceStep := totalOpcodes / 2

	scenario := &ChallengeScenario{
		Name: "two forked assertions at the same height",
		AliceStateManager: func() l2stateprovider.Provider {
			sm, err := statemanager.NewForSimpleMachine(statemanager.WithLayerZeroHeights(layerZeroHeights, numBigSteps))
			if err != nil {
				t.Fatal(err)
			}
			return sm
		}(),
		BobStateManager: func() l2stateprovider.Provider {
			assertionDivergenceHeight := uint64(4)
			assertionBlockHeightDifference := int64(4)
			sm, err := statemanager.NewForSimpleMachine(
				statemanager.WithLayerZeroHeights(layerZeroHeights, numBigSteps),
				statemanager.WithMachineDivergenceStep(machineDivergenceStep),
				statemanager.WithBlockDivergenceHeight(assertionDivergenceHeight),
				statemanager.WithDivergentBlockHeightOffset(assertionBlockHeightDifference),
			)
			if err != nil {
				t.Fatal(err)
			}
			return sm
		}(),
		Expectations: []expect{
			expectAssertionConfirmedByChallengeWinner,
			expectAliceAndBobStaked,
		},
	}

	testChallengeProtocol_AliceAndBob(
		t,
		be,
		scenario,
		challenge_testing.WithLayerZeroHeights(layerZeroHeights),
		challenge_testing.WithNumBigStepLevels(numBigSteps),
	)
}

func TestSync_HonestBobStopsCharlieJoins(t *testing.T) {
	be, err := backend.NewAnvilLocal(context.Background())
	require.NoError(t, err)
	require.NoError(t, be.Start())
	defer func() {
		require.NoError(t, be.Stop(), "error stopping backend")
	}()

	layerZeroHeights := &protocol.LayerZeroHeights{
		BlockChallengeHeight:     1 << 5,
		BigStepChallengeHeight:   1 << 3,
		SmallStepChallengeHeight: 1 << 5,
	}
	numBigSteps := uint8(3)
	totalWasmOpcodes := uint64(1)
	for i := uint8(0); i < numBigSteps; i++ {
		totalWasmOpcodes *= layerZeroHeights.BigStepChallengeHeight
	}
	totalWasmOpcodes *= layerZeroHeights.SmallStepChallengeHeight
	machineDivergenceStep := totalWasmOpcodes / 2

	scenario := &ChallengeScenario{
		Name: "honest bob stops and charlie joins in late to defend his honest edges",
		AliceStateManager: func() l2stateprovider.Provider {
			assertionDivergenceHeight := uint64(4)
			sm, err := statemanager.NewForSimpleMachine(
				statemanager.WithLayerZeroHeights(layerZeroHeights, numBigSteps),
				statemanager.WithMachineDivergenceStep(machineDivergenceStep),
				statemanager.WithBlockDivergenceHeight(assertionDivergenceHeight),
				statemanager.WithDivergentBlockHeightOffset(0),
			)
			if err != nil {
				t.Fatal(err)
			}
			return sm
		}(),
		BobStateManager: func() l2stateprovider.Provider {
			sm, err := statemanager.NewForSimpleMachine(statemanager.WithLayerZeroHeights(layerZeroHeights, numBigSteps))
			if err != nil {
				t.Fatal(err)
			}
			return sm
		}(),
		CharlieStateManager: func() l2stateprovider.Provider {
			sm, err := statemanager.NewForSimpleMachine(statemanager.WithLayerZeroHeights(layerZeroHeights, numBigSteps))
			if err != nil {
				t.Fatal(err)
			}
			return sm
		}(),
		Expectations: []expect{
			expectAssertionConfirmedByChallengeWinner,
		},
	}

	testSyncBobStopsCharlieJoins(
		t,
		be,
		scenario,
		challenge_testing.WithLayerZeroHeights(layerZeroHeights),
		challenge_testing.WithNumBigStepLevels(numBigSteps),
	)
}

func testChallengeProtocol_AliceAndBob(t *testing.T, be backend.Backend, scenario *ChallengeScenario, opts ...challenge_testing.Opt) {
	t.Run(scenario.Name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
		defer cancel()

		rollup, err := be.DeployRollup(opts...)
		if err != nil {
			t.Fatal(err)
		}

		a, aChain, err := setupValidator(ctx, be, rollup, scenario.AliceStateManager, be.Alice(), "alice")
		if err != nil {
			t.Fatal(err)
		}
		b, bChain, err := setupValidator(ctx, be, rollup, scenario.BobStateManager, be.Bob(), "bob")
		if err != nil {
			t.Fatal(err)
		}

		// Post assertions.
		alicePoster, err := assertions.NewPoster(aChain, scenario.AliceStateManager, "alice", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		bobPoster, err := assertions.NewPoster(bChain, scenario.BobStateManager, "bob", time.Hour)
		if err != nil {
			t.Fatal(err)
		}

		aliceLeaf, err := alicePoster.PostAssertionAndNewStake(ctx)
		if err != nil {
			t.Fatal(err)
		}
		bobLeaf, err := bobPoster.PostAssertionAndNewStake(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Scan for created assertions.
		aliceScanner, err := assertions.NewScanner(aChain, scenario.AliceStateManager, be.Client(), a, rollup, "alice", time.Hour, time.Second*10)
		if err != nil {
			t.Fatal(err)
		}
		bobScanner, err := assertions.NewScanner(bChain, scenario.BobStateManager, be.Client(), b, rollup, "bob", time.Hour, time.Second*10)
		if err != nil {
			t.Fatal(err)
		}

		if err := aliceScanner.ProcessAssertionCreation(ctx, aliceLeaf.Id()); err != nil {
			t.Fatal(err)
		}
		if err := bobScanner.ProcessAssertionCreation(ctx, bobLeaf.Id()); err != nil {
			t.Fatal(err)
		}

		a.Start(ctx)
		b.Start(ctx)

		g, ctx := errgroup.WithContext(ctx)
		for _, e := range scenario.Expectations {
			fn := e // loop closure
			g.Go(func() error {
				return fn(t, ctx, be)
			})
		}

		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}

		trackedBackend, ok := aChain.Backend().(*solimpl.TrackedContractBackend)
		if !ok {
			t.Fatal("Not a tracked contract backend")
		}
		t.Log("Printing Alice's ethclient metrics at the end of a challenge")
		trackedBackend.PrintMetrics()
	})
}

// testSyncBobStopsCharlieJoins tests the scenario where Bob stops and Charlie joins.
func testSyncBobStopsCharlieJoins(t *testing.T, be backend.Backend, s *ChallengeScenario, opts ...challenge_testing.Opt) {
	t.Run(s.Name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
		defer cancel()

		rollup, err := be.DeployRollup(opts...)
		require.NoError(t, err)

		// Bad Alice
		aChain, err := solimpl.NewAssertionChain(ctx, rollup, be.Alice(), be.Client())
		require.NoError(t, err)
		alice, err := validator.New(ctx, aChain, be.Client(), s.AliceStateManager, rollup, validator.WithAddress(be.Alice().From), validator.WithName("alice"), validator.WithMode(types.MakeMode), validator.WithEdgeTrackerWakeInterval(100*time.Millisecond))
		require.NoError(t, err)

		// Good Bob
		bobCtx, bobCancelCtx := context.WithCancel(ctx)
		bChain, err := solimpl.NewAssertionChain(bobCtx, rollup, be.Bob(), be.Client())
		require.NoError(t, err)
		bob, err := validator.New(bobCtx, bChain, be.Client(), s.BobStateManager, rollup, validator.WithAddress(be.Bob().From), validator.WithName("bob"), validator.WithMode(types.MakeMode), validator.WithEdgeTrackerWakeInterval(100*time.Millisecond))
		require.NoError(t, err)

		alicePoster, err := assertions.NewPoster(aChain, s.AliceStateManager, "alice", time.Hour)
		require.NoError(t, err)
		bobPoster, err := assertions.NewPoster(bChain, s.BobStateManager, "bob", time.Hour)
		require.NoError(t, err)
		aliceLeaf, err := alicePoster.PostAssertionAndNewStake(ctx)
		require.NoError(t, err)
		bobLeaf, err := bobPoster.PostAssertionAndNewStake(bobCtx)
		require.NoError(t, err)
		aliceScanner, err := assertions.NewScanner(aChain, s.AliceStateManager, be.Client(), alice, rollup, "alice", time.Hour, time.Second*10)
		require.NoError(t, err)
		bobScanner, err := assertions.NewScanner(bChain, s.BobStateManager, be.Client(), bob, rollup, "bob", time.Hour, time.Second*10)
		require.NoError(t, err)
		require.NoError(t, aliceScanner.ProcessAssertionCreation(ctx, aliceLeaf.Id()))
		require.NoError(t, bobScanner.ProcessAssertionCreation(bobCtx, bobLeaf.Id()))

		// Alice and bob starts to challenge each other.
		alice.Start(ctx)
		bob.Start(bobCtx)

		// 10s later, bob shuts down
		time.Sleep(10 * time.Second)
		bobCancelCtx()

		// Good Charlie joins
		cChain, err := solimpl.NewAssertionChain(ctx, rollup, be.Charlie(), be.Client())
		require.NoError(t, err)
		charlie, err := validator.New(ctx, cChain, be.Client(), s.CharlieStateManager, rollup, validator.WithAddress(be.Charlie().From), validator.WithName("charlie"), validator.WithMode(types.DefensiveMode), validator.WithEdgeTrackerWakeInterval(100*time.Millisecond)) // Defensive is good enough here.
		require.NoError(t, err)
		charlie.Start(ctx)

		g, ctx := errgroup.WithContext(ctx)
		for _, e := range s.Expectations {
			fn := e // loop closure
			g.Go(func() error {
				return fn(t, ctx, be)
			})
		}

		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}

	})
}

// setupValidator initializes a validator with the minimum required configuration.
func setupValidator(
	ctx context.Context,
	be backend.Backend,
	rollup common.Address,
	sm l2stateprovider.Provider,
	txOpts *bind.TransactOpts,
	name string,
) (*validator.Manager, protocol.Protocol, error) {
	chain, err := solimpl.NewAssertionChain(
		ctx,
		rollup,
		txOpts,
		be.Client(),
		solimpl.WithTrackedContractBackend(),
	)
	if err != nil {
		return nil, nil, err
	}

	v, err := validator.New(
		ctx,
		chain,
		be.Client(),
		sm,
		rollup,
		validator.WithAddress(txOpts.From),
		validator.WithName(name),
		validator.WithEdgeTrackerWakeInterval(time.Millisecond*250),
		validator.WithMode(types.MakeMode),
	)
	if err != nil {
		return nil, nil, err
	}

	return v, chain, nil
}

func totalWasmOpcodes(heights *protocol.LayerZeroHeights, numBigSteps uint8) uint64 {
	totalWasmOpcodes := uint64(1)
	for i := uint8(0); i < numBigSteps; i++ {
		totalWasmOpcodes *= heights.BigStepChallengeHeight
	}
	totalWasmOpcodes *= heights.SmallStepChallengeHeight
	return totalWasmOpcodes
}
