package endtoend

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	validator "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/endtoend/internal/backend"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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

func TestChallengeProtocol_AliceAndBob_AnvilLocal_InMiddleOfBlock_WithoutFlakyEthClient(t *testing.T) {
	aliceAndBobInMiddleOfBlock(t, false)
}

func aliceAndBobInMiddleOfBlock(t *testing.T, useFlakyEthClient bool) {
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
		Name: "disagreement in middle of block",
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
		useFlakyEthClient,
		scenario,
		challenge_testing.WithLayerZeroHeights(layerZeroHeights),
		challenge_testing.WithNumBigStepLevels(numBigSteps),
	)
}

func TestChallengeProtocol_AliceAndBob_AnvilLocal_LastOpcode(t *testing.T) {
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

	// Diverge exactly at the last opcode within the block.
	machineDivergenceStep := totalOpcodes - 1

	scenario := &ChallengeScenario{
		Name: "disagreement at last opcode",
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
		false,
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

type FlakyEthClient struct {
	*ethclient.Client
}

func (f *FlakyEthClient) flaky() error {
	// 10% chance of failure
	if rand.Intn(10) > 8 {
		return errors.New("flaky error")
	}
	return nil
}

func (f *FlakyEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*eth_types.Receipt, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.TransactionReceipt(ctx, txHash)
}

func (f *FlakyEthClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.CodeAt(ctx, contract, blockNumber)
}

func (f *FlakyEthClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.CallContract(ctx, call, blockNumber)
}

func (f *FlakyEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*eth_types.Header, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.HeaderByNumber(ctx, number)
}

func (f *FlakyEthClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.PendingCodeAt(ctx, account)
}

func (f *FlakyEthClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if err := f.flaky(); err != nil {
		return 0, err
	}
	return f.Client.PendingNonceAt(ctx, account)
}

func (f *FlakyEthClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.SuggestGasPrice(ctx)
}

func (f *FlakyEthClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.SuggestGasTipCap(ctx)
}

func (f *FlakyEthClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	if err := f.flaky(); err != nil {
		return 0, err
	}
	return f.Client.EstimateGas(ctx, call)
}

func (f *FlakyEthClient) SendTransaction(ctx context.Context, tx *eth_types.Transaction) error {
	if err := f.flaky(); err != nil {
		return err
	}
	return f.Client.SendTransaction(ctx, tx)
}

func (f *FlakyEthClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]eth_types.Log, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.FilterLogs(ctx, query)
}
func (f *FlakyEthClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- eth_types.Log) (ethereum.Subscription, error) {
	if err := f.flaky(); err != nil {
		return nil, err
	}
	return f.Client.SubscribeFilterLogs(ctx, query, ch)
}

func testChallengeProtocol_AliceAndBob(t *testing.T, be backend.Backend, useFlakyEthClient bool, scenario *ChallengeScenario, opts ...challenge_testing.Opt) {
	t.Run(scenario.Name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
		defer cancel()

		rollup, err := be.DeployRollup(opts...)
		if err != nil {
			t.Fatal(err)
		}
		var ethClient protocol.ChainBackend
		if useFlakyEthClient {
			ethClient = &FlakyEthClient{be.Client()}
		} else {
			ethClient = be.Client()
		}
		a, aChain, err := retry.UntilSucceedsMultipleReturnValue(ctx, func() (*validator.Manager, protocol.Protocol, error) {
			return setupValidator(ctx, ethClient, rollup, scenario.AliceStateManager, be.Alice(), "alice")
		})
		if err != nil {
			t.Fatal(err)
		}
		b, bChain, err := retry.UntilSucceedsMultipleReturnValue(ctx, func() (*validator.Manager, protocol.Protocol, error) {
			return setupValidator(ctx, ethClient, rollup, scenario.BobStateManager, be.Bob(), "bob")
		})
		if err != nil {
			t.Fatal(err)
		}

		// Post assertions.
		alicePoster, err := retry.UntilSucceeds(ctx, func() (*assertions.Manager, error) {
			return assertions.NewManager(aChain, scenario.AliceStateManager, be.Client(), a, rollup, "alice", time.Hour, time.Second*10, scenario.AliceStateManager, time.Hour, time.Second)
		})
		if err != nil {
			t.Fatal(err)
		}
		bobPoster, err := retry.UntilSucceeds(ctx, func() (*assertions.Manager, error) {
			return assertions.NewManager(bChain, scenario.BobStateManager, be.Client(), b, rollup, "bob", time.Hour, time.Second*10, scenario.BobStateManager, time.Hour, time.Second)
		})
		if err != nil {
			t.Fatal(err)
		}

		aliceLeaf, err := retry.UntilSucceeds(ctx, func() (protocol.Assertion, error) {
			return alicePoster.PostAssertion(ctx)
		})
		if err != nil {
			t.Fatal(err)
		}
		bobLeaf, err := retry.UntilSucceeds(ctx, func() (protocol.Assertion, error) {
			return bobPoster.PostAssertion(ctx)
		})
		if err != nil {
			t.Fatal(err)
		}

		// Scan for created assertions.
		if _, err := retry.UntilSucceeds(ctx, func() (protocol.Assertion, error) {
			return nil, alicePoster.ProcessAssertionCreationEvent(ctx, aliceLeaf.Id())
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := retry.UntilSucceeds(ctx, func() (protocol.Assertion, error) {
			return nil, bobPoster.ProcessAssertionCreationEvent(ctx, bobLeaf.Id())
		}); err != nil {
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

		alicePoster, err := assertions.NewManager(aChain, s.AliceStateManager, be.Client(), alice, rollup, "alice", time.Hour, time.Second*10, s.AliceStateManager, time.Hour, time.Second)
		require.NoError(t, err)
		bobPoster, err := assertions.NewManager(bChain, s.BobStateManager, be.Client(), bob, rollup, "bob", time.Hour, time.Second*10, s.BobStateManager, time.Hour, time.Second)
		require.NoError(t, err)
		aliceLeaf, err := alicePoster.PostAssertion(ctx)
		require.NoError(t, err)
		bobLeaf, err := bobPoster.PostAssertion(bobCtx)
		require.NoError(t, err)
		require.NoError(t, alicePoster.ProcessAssertionCreationEvent(ctx, aliceLeaf.Id()))
		require.NoError(t, bobPoster.ProcessAssertionCreationEvent(bobCtx, bobLeaf.Id()))

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
	backend protocol.ChainBackend,
	rollup common.Address,
	sm l2stateprovider.Provider,
	txOpts *bind.TransactOpts,
	name string,
) (*validator.Manager, protocol.Protocol, error) {
	chain, err := solimpl.NewAssertionChain(
		ctx,
		rollup,
		txOpts,
		backend,
		solimpl.WithTrackedContractBackend(),
	)
	if err != nil {
		return nil, nil, err
	}

	v, err := validator.New(
		ctx,
		chain,
		backend,
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
