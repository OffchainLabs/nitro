package endtoend

import (
	"context"
	"fmt"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	validator "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/endtoend/internal/backend"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/toys/assertions"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/testing/toys/state-provider"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/headerreader"
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
			t.Log(fmt.Errorf("error stopping backend: %v", err))
		}
	}()

	scenarios := []*ChallengeScenario{
		{
			Name: "two forked assertions at the same height",
			AliceStateManager: func() l2stateprovider.Provider {
				sm, err := statemanager.NewForSimpleMachine()
				if err != nil {
					t.Fatal(err)
				}
				return sm
			}(),
			BobStateManager: func() l2stateprovider.Provider {
				cfg := &challengeProtocolTestConfig{
					// The heights at which the validators diverge in histories. In this test,
					// alice and bob start diverging at height 3 at all subchallenge levels.
					assertionDivergenceHeight: 4,
					bigStepDivergenceHeight:   4,
					smallStepDivergenceHeight: 4,
				}
				sm, err := statemanager.NewForSimpleMachine(
					statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*protocol.LevelZeroSmallStepEdgeHeight+cfg.smallStepDivergenceHeight),
					statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight),
					statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
				)
				if err != nil {
					t.Fatal(err)
				}
				return sm
			}(),
			Expectations: []expect{
				expectChallengeCompletedByOneStepProof,
				expectAliceAndBobStaked,
			},
		},
		{
			Name: "two forked assertions at the different step heights",
			AliceStateManager: func() l2stateprovider.Provider {
				sm, err := statemanager.NewForSimpleMachine()
				if err != nil {
					t.Fatal(err)
				}
				return sm
			}(),
			BobStateManager: func() l2stateprovider.Provider {
				cfg := &challengeProtocolTestConfig{
					// The heights at which the validators diverge in histories. In this test,
					// alice and bob diverge heights at different subchallenge levels.
					assertionDivergenceHeight: 8,
					bigStepDivergenceHeight:   6,
					smallStepDivergenceHeight: 4,
				}
				sm, err := statemanager.NewForSimpleMachine(
					statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*protocol.LevelZeroSmallStepEdgeHeight+cfg.smallStepDivergenceHeight),
					statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight),
					statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
				)
				if err != nil {
					t.Fatal(err)
				}
				return sm
			}(),
			Expectations: []expect{
				expectChallengeCompletedByOneStepProof,
				expectAliceAndBobStaked,
			},
		},
	}

	for _, scenario := range scenarios {
		testChallengeProtocol_AliceAndBob(t, be, scenario)
	}
}

func TestChallengeProtocol_AliceAndBobAndCharlie_AnvilLocal(t *testing.T) {
	t.Skip()
	be, err := backend.NewAnvilLocal(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if err := be.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := be.Stop(); err != nil {
			t.Log(fmt.Errorf("error stopping backend: %v", err))
		}
	}()

	scenarios := []*ChallengeScenario{
		{
			Name: "forked assertion at different heights",
			AliceStateManager: func() l2stateprovider.Provider {
				sm, err := statemanager.NewForSimpleMachine()
				if err != nil {
					t.Fatal(err)
				}
				return sm
			}(),
			BobStateManager: func() l2stateprovider.Provider {
				cfg := &challengeProtocolTestConfig{
					assertionDivergenceHeight: 4,
					bigStepDivergenceHeight:   4,
					smallStepDivergenceHeight: 4,
				}
				sm, err := statemanager.NewForSimpleMachine(
					statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*protocol.LevelZeroSmallStepEdgeHeight+cfg.smallStepDivergenceHeight),
					statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight),
					statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
				)
				if err != nil {
					t.Fatal(err)
				}
				return sm
			}(),
			CharlieStateManager: func() l2stateprovider.Provider {
				cfg := &challengeProtocolTestConfig{
					assertionDivergenceHeight: 5,
					bigStepDivergenceHeight:   5,
					smallStepDivergenceHeight: 5,
				}
				sm, err := statemanager.NewForSimpleMachine(
					statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*protocol.LevelZeroSmallStepEdgeHeight+cfg.smallStepDivergenceHeight),
					statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight),
					statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
				)
				if err != nil {
					t.Fatal(err)
				}
				return sm

			}(),
			Expectations: []expect{
				expectChallengeCompletedByOneStepProof,
			},
		},
	}

	for _, scenario := range scenarios {
		testChallengeProtocol_AliceAndBobAndCharlie(t, be, scenario)
	}
}

func testChallengeProtocol_AliceAndBob(t *testing.T, be backend.Backend, scenario *ChallengeScenario) {
	t.Run(scenario.Name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		rollup, err := be.DeployRollup()
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
		alicePoster := assertions.NewPoster(aChain, scenario.AliceStateManager, "alice", time.Hour)
		bobPoster := assertions.NewPoster(bChain, scenario.BobStateManager, "bob", time.Hour)

		aliceLeaf, err := alicePoster.PostLatestAssertion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		bobLeaf, err := bobPoster.PostLatestAssertion(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Scan for created assertions.
		aliceScanner := assertions.NewScanner(aChain, scenario.AliceStateManager, be.Client(), a, rollup, "alice", time.Hour)
		bobScanner := assertions.NewScanner(bChain, scenario.BobStateManager, be.Client(), b, rollup, "bob", time.Hour)

		if err := aliceScanner.ProcessAssertionCreation(ctx, aliceLeaf.Id()); err != nil {
			panic(err)
		}
		if err := bobScanner.ProcessAssertionCreation(ctx, bobLeaf.Id()); err != nil {
			panic(err)
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
	})
}

func testChallengeProtocol_AliceAndBobAndCharlie(t *testing.T, be backend.Backend, scenario *ChallengeScenario) {
	t.Run(scenario.Name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		rollup, err := be.DeployRollup()
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
		c, cChain, err := setupValidator(ctx, be, rollup, scenario.CharlieStateManager, be.Charlie(), "charlie")
		if err != nil {
			t.Fatal(err)
		}

		// Post assertions.
		alicePoster := assertions.NewPoster(aChain, scenario.AliceStateManager, "alice", time.Hour)
		bobPoster := assertions.NewPoster(bChain, scenario.BobStateManager, "bob", time.Hour)
		charliePoster := assertions.NewPoster(cChain, scenario.CharlieStateManager, "charlie", time.Hour)

		aliceLeaf, err := alicePoster.PostLatestAssertion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		bobLeaf, err := bobPoster.PostLatestAssertion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		charlieLeaf, err := charliePoster.PostLatestAssertion(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Scan for created assertions.
		aliceScanner := assertions.NewScanner(aChain, scenario.AliceStateManager, be.Client(), a, rollup, "alice", time.Hour)
		bobScanner := assertions.NewScanner(bChain, scenario.BobStateManager, be.Client(), b, rollup, "bob", time.Hour)
		charlieScanner := assertions.NewScanner(cChain, scenario.CharlieStateManager, be.Client(), c, rollup, "charlie", time.Hour)

		if err := aliceScanner.ProcessAssertionCreation(ctx, aliceLeaf.Id()); err != nil {
			panic(err)
		}
		if err := bobScanner.ProcessAssertionCreation(ctx, bobLeaf.Id()); err != nil {
			panic(err)
		}
		if err := charlieScanner.ProcessAssertionCreation(ctx, charlieLeaf.Id()); err != nil {
			panic(err)
		}

		a.Start(ctx)
		b.Start(ctx)
		c.Start(ctx)

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
	hr := headerreader.New(be.Client(), func() *headerreader.Config {
		return &headerreader.DefaultConfig
	})

	chain, err := solimpl.NewAssertionChain(
		ctx,
		rollup,
		txOpts,
		be.Client(),
		hr,
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
	)
	if err != nil {
		return nil, nil, err
	}

	return v, chain, nil
}
