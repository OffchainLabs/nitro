package main

import (
	"context"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	validator "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/testing/toys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var (
	// The heights at which Alice and Bob diverge at each challenge level.
	divergeHeightAtL2 = uint64(4)
	// How often an edge tracker needs to wake and perform its responsibilities.
	edgeTrackerWakeInterval = time.Second
	// How often the validator polls the chain to see if new assertions have been posted.
	checkForAssertionsInteral = time.Second
	// How often the validator will post its latest assertion to the chain.
	postNewAssertionInterval = time.Second * 5
	// How often we advance the blockchain's latest block in the background using a simulated backend.
	advanceChainInterval = time.Second * 5
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

	aliceStateManager, err := statemanager.NewForSimpleMachine()
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
		statemanager.WithMachineDivergenceStep(cfg.bigStepDivergenceHeight*protocol.LevelZeroSmallStepEdgeHeight+cfg.smallStepDivergenceHeight),
		statemanager.WithBlockDivergenceHeight(cfg.assertionDivergenceHeight),
		statemanager.WithDivergentBlockHeightOffset(cfg.assertionBlockHeightDifference),
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
		validator.WithNewAssertionCheckInterval(checkForAssertionsInteral),
		validator.WithPostAssertionsInterval(postNewAssertionInterval),
	)
	if err != nil {
		return nil, err
	}

	return v, nil
}
