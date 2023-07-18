// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

package main

import (
	"context"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/assertions"
	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	validator "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	challenge_testing "github.com/OffchainLabs/challenge-protocol-v2/testing"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/testing/toys/state-provider"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var (
	// The heights at which Alice and Bob diverge at each challenge level.
	divergeHeightAtL2 = uint64(4)
	// How often an edge tracker needs to wake and perform its responsibilities.
	edgeTrackerWakeInterval = time.Millisecond * 250
	// How often the validator polls the chain to see if new assertions have been posted.
	checkForAssertionsInterval = time.Second
	// How often the validator will post its latest assertion to the chain.
	postNewAssertionInterval = time.Hour
	// How often we advance the blockchain's latest block in the background using a simulated backend.
	advanceChainInterval = time.Second * 2
	// Heights
	levelZeroBlockHeight     = uint64(1 << 5)
	levelZeroBigStepHeight   = uint64(1 << 5)
	levelZeroSmallStepHeight = uint64(1 << 5)
)

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

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	setupCfg, err := setup.ChainsWithEdgeChallengeManager()
	if err != nil {
		panic(err)
	}
	chains := setupCfg.Chains
	accs := setupCfg.Accounts
	addrs := setupCfg.Addrs
	backend := setupCfg.Backend

	// Advance the chain by 100 blocks as there needs to be a minimum period of time
	// before any assertions can be made on-chain.
	for i := 0; i < 100; i++ {
		backend.Commit()
	}

	aliceStateManager, err := statemanager.NewForSimpleMachine(statemanager.WithLevelZeroEdgeHeights(&challenge_testing.LevelZeroHeights{
		BlockChallengeHeight:     levelZeroBlockHeight,
		BigStepChallengeHeight:   levelZeroBigStepHeight,
		SmallStepChallengeHeight: levelZeroSmallStepHeight,
	}))
	if err != nil {
		panic(err)
	}
	cfg := &challengeProtocolTestConfig{
		// The heights at which the validators diverge in histories. In this test,
		// alice and bob start diverging at height 3 at all subchallenge levels.
		assertionDivergenceHeight: divergeHeightAtL2,
		bigStepDivergenceHeight:   divergeHeightAtL2,
		smallStepDivergenceHeight: divergeHeightAtL2,
	}
	bobStateManager, err := statemanager.NewForSimpleMachine(
		statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*levelZeroSmallStepHeight+cfg.smallStepDivergenceHeight),
		statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight),
		statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
		statemanager.WithLevelZeroEdgeHeights(&challenge_testing.LevelZeroHeights{
			BlockChallengeHeight:     levelZeroBlockHeight,
			BigStepChallengeHeight:   levelZeroBigStepHeight,
			SmallStepChallengeHeight: levelZeroSmallStepHeight,
		}),
	)
	if err != nil {
		panic(err)
	}
	charlieStateManager, err := statemanager.NewForSimpleMachine(
		statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*levelZeroSmallStepHeight+(cfg.smallStepDivergenceHeight+2)),
		statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight+2),
		statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
		statemanager.WithLevelZeroEdgeHeights(&challenge_testing.LevelZeroHeights{
			BlockChallengeHeight:     levelZeroBlockHeight,
			BigStepChallengeHeight:   levelZeroBigStepHeight,
			SmallStepChallengeHeight: levelZeroSmallStepHeight,
		}),
		statemanager.WithMaliciousMachineIndex(1),
	)
	if err != nil {
		panic(err)
	}

	a, err := setupValidator(ctx, chains[0], backend, addrs.Rollup, aliceStateManager, "alice", accs[0].TxOpts.From)
	if err != nil {
		panic(err)
	}
	b, err := setupValidator(ctx, chains[1], backend, addrs.Rollup, bobStateManager, "bob", accs[1].TxOpts.From)
	if err != nil {
		panic(err)
	}
	c, err := setupValidator(ctx, chains[2], backend, addrs.Rollup, charlieStateManager, "charlie", accs[2].TxOpts.From)
	if err != nil {
		panic(err)
	}

	// Post assertions in the background.
	alicePoster := assertions.NewPoster(chains[0], aliceStateManager, "alice", postNewAssertionInterval)
	bobPoster := assertions.NewPoster(chains[1], bobStateManager, "bob", postNewAssertionInterval)
	charliePoster := assertions.NewPoster(chains[2], charlieStateManager, "charlie", postNewAssertionInterval)

	aliceLeaf, err := alicePoster.PostLatestAssertion(ctx)
	if err != nil {
		panic(err)
	}
	bobLeaf, err := bobPoster.PostLatestAssertion(ctx)
	if err != nil {
		panic(err)
	}
	charlieLeaf, err := charliePoster.PostLatestAssertion(ctx)
	if err != nil {
		panic(err)
	}

	// Scan for created assertions in the background.
	aliceScanner := assertions.NewScanner(chains[0], aliceStateManager, backend, a, addrs.Rollup, "alice", checkForAssertionsInterval)
	bobScanner := assertions.NewScanner(chains[1], bobStateManager, backend, b, addrs.Rollup, "bob", checkForAssertionsInterval)
	charlieScanner := assertions.NewScanner(chains[2], charlieStateManager, backend, c, addrs.Rollup, "charlie", checkForAssertionsInterval)

	if err := aliceScanner.ProcessAssertionCreation(ctx, aliceLeaf.Id()); err != nil {
		panic(err)
	}
	if err := bobScanner.ProcessAssertionCreation(ctx, bobLeaf.Id()); err != nil {
		panic(err)
	}
	if err := charlieScanner.ProcessAssertionCreation(ctx, charlieLeaf.Id()); err != nil {
		panic(err)
	}

	// Advance the blockchain in the background.
	go func() {
		tick := time.NewTicker(advanceChainInterval)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				backend.Commit()
			case <-ctx.Done():
				return
			}
		}
	}()

	a.Start(ctx)
	b.Start(ctx)
	c.Start(ctx)

	<-ctx.Done()
}

// setupValidator initializes a validator with the minimum required configuration.
func setupValidator(
	ctx context.Context,
	chain protocol.AssertionChain,
	backend bind.ContractBackend,
	rollup common.Address,
	sm l2stateprovider.Provider,
	name string,
	addr common.Address,
) (*validator.Manager, error) {
	v, err := validator.New(
		ctx,
		chain,
		backend,
		sm,
		rollup,
		validator.WithAddress(addr),
		validator.WithName(name),
		validator.WithEdgeTrackerWakeInterval(edgeTrackerWakeInterval),
		validator.WithMode(types.MakeMode),
	)
	if err != nil {
		return nil, err
	}

	return v, nil
}
