package main

import (
	"context"
	"math/big"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	chalTesting "github.com/OffchainLabs/challenge-protocol-v2/testing"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/OffchainLabs/challenge-protocol-v2/validator"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var (
	// The chain id for the backend.
	chainId = big.NewInt(1337)
	// The size of a mini stake that is posted when creating leaf edges in
	// challenges (clarify if gwei?).
	miniStakeSize = big.NewInt(1)
	// The heights at which Alice and Bob diverge at each challenge level.
	divergeHeightAtL2 = uint64(3)
	// How often an edge tracker needs to wake and perform its responsibilities.
	edgeTrackerWakeInterval = time.Millisecond * 500
	// How often the validator polls the chain to see if new assertions have been posted.
	checkForAssertionsInteral = time.Second
	// How often the validator will post its latest assertion to the chain.
	postNewAssertionInterval = time.Second * 5
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// Setup an admin account, Alice and Bob.
	accs, backend, err := setup.SetupAccounts(3)
	if err != nil {
		panic(err)
	}
	addresses, err := deployStack(ctx, accs[0], backend)
	if err != nil {
		panic(err)
	}

	headerReader := headerreader.New(util.SimulatedBackendWrapper{
		SimulatedBackend: backend,
	}, func() *headerreader.Config {
		return &headerreader.TestConfig
	})
	headerReader.Start(ctx)

	// Setup the chain abstractions for Alice and Bob.
	aliceL1ChainWrapper, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[1].TxOpts,
		backend,
		headerReader,
	)
	if err != nil {
		panic(err)
	}

	bobL1ChainWrapper, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[2].TxOpts,
		backend,
		headerReader,
	)
	if err != nil {
		panic(err)
	}

	// Advance the L1 chain by 100 blocks as there needs to be a minimum period of time
	// before any assertions can be submitted to L1.
	for i := 0; i < 100; i++ {
		backend.Commit()
	}

	// Initialize Alice and Bob's respective L2 state managers.
	aliceL2StateManager, err := statemanager.NewForSimpleMachine()
	if err != nil {
		panic(err)
	}

	// Bob diverges from Alice's L2 history at the specified divergence height.
	bobL2StateManager, err := statemanager.NewForSimpleMachine(
		statemanager.WithBlockDivergenceHeight(1),
		statemanager.WithMachineDivergenceStep(divergeHeightAtL2+(divergeHeightAtL2-1)*protocol.LevelZeroSmallStepEdgeHeight),
	)
	if err != nil {
		panic(err)
	}

	timeReference := util.NewRealTimeReference()
	commonValidatorOpts := []validator.Opt{
		validator.WithTimeReference(timeReference),
		validator.WithEdgeTrackerWakeInterval(edgeTrackerWakeInterval),
		validator.WithPostAssertionsInterval(postNewAssertionInterval),
		validator.WithNewAssertionCheckInterval(checkForAssertionsInteral),
	}
	aliceOpts := []validator.Opt{
		validator.WithName("alice"),
		validator.WithAddress(accs[1].AccountAddr),
	}

	// Sets up Alice and Bob validators.
	alice, err := validator.New(
		ctx,
		aliceL1ChainWrapper,
		backend,
		aliceL2StateManager,
		addresses.Rollup,
		append(aliceOpts, commonValidatorOpts...)...,
	)
	if err != nil {
		panic(err)
	}

	bobOpts := []validator.Opt{
		validator.WithName("bob"),
		validator.WithAddress(accs[2].AccountAddr),
	}
	bob, err := validator.New(
		ctx,
		bobL1ChainWrapper,
		backend,
		bobL2StateManager,
		addresses.Rollup,
		append(bobOpts, commonValidatorOpts...)...,
	)
	if err != nil {
		panic(err)
	}

	// Spawns the validators, which should have them post assertions, challenge each other,
	// and have the honest party win.
	go alice.Start(ctx)
	go bob.Start(ctx)

	<-ctx.Done()
}

func deployStack(
	ctx context.Context,
	adminAccount *setup.TestAccount,
	backend *backends.SimulatedBackend,
) (*setup.RollupAddresses, error) {
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := adminAccount
	loserStakeEscrow := common.Address{}
	cfg := chalTesting.GenerateRollupConfig(
		prod,
		wasmModuleRoot,
		rollupOwner.AccountAddr,
		chainId,
		loserStakeEscrow,
		miniStakeSize,
	)
	return setup.DeployFullRollupStack(
		ctx,
		backend,
		adminAccount.TxOpts,
		common.Address{}, // Sequencer addr.
		cfg,
	)
}
