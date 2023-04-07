package validator_test

import (
	"context"
	"time"

	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/validator"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var (
	// Rollup contract address.
	Rollup = common.Address{}
	// EdgeChallengeManager contract address.
	EdgeChallengeManager = common.Address{}
)

func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create your backend connection to an L1 blockchain.
	backend, err := ethclient.DialContext(ctx, "http://localhost:8545")
	if err != nil {
		// An error is unlikely to occur at this point, but your application should handle any
		// error as a critical error.
		panic(err)
	}

	// Create a L1 header reader with the default configuration. This is used to monitor the L1
	// blockchain for updates.
	hr := headerreader.New(backend, func() *headerreader.Config {
		return &headerreader.DefaultConfig
	})

	// Load your transaction options from an account you wish to send transactions from.
	txOpts := loadTransactOpts()

	// Setup assertion chain abstraction.
	chain, err := solimpl.NewAssertionChain(
		ctx,
		Rollup,
		txOpts,
		backend,
		hr, // headerReader
	)
	if err != nil {
		// An error is unlikely to occur at this point, but your application should handle any
		// error as a critical error.
		panic(err)
	}

	// Bring your own implementation of statemanager.Manager. The statemanager.Manager maintains
	// information about the state of the L2 chain. It is called upon to provide supporting
	// data during a challenge.
	//
	// This example will use a simulated implementation with fake data.
	sm := exampleStateManager()

	v, err := validator.New(
		ctx,
		chain,
		backend,
		sm,
		Rollup,
		// Optional configuration.
		validator.WithAddress(txOpts.From),
		validator.WithName("example-validator"),
		validator.WithNewAssertionCheckInterval(time.Second),
		// Any additional validator.Opt options can be passed here.
	)
	if err != nil {
		// An error is unlikely to occur at this point, but your application should handle any
		// error as a critical error.
		panic(err)
	}

	// Start the validator routine. The validator will now wait for new assertions to be posted
	// and participate in any challenges.
	go v.Start(ctx)

	// Wait for the validator to exit or for the context to expire in this example.
	<-ctx.Done()
}

func exampleStateManager() statemanager.Manager {
	// This is an example of a state manager that is populated with fake data and complies with the
	// statemanager.Manager interface.
	sm, err := statemanager.New(
		[]common.Hash{ // stateRoots
			common.HexToHash("0x1"),
			common.HexToHash("0x2"),
			common.HexToHash("0x3"),
			common.HexToHash("0x4"),
			common.HexToHash("0x5"),
		},
		// any relevant statemanager.Opt options can be passed here.
		statemanager.WithMaxWavmOpcodesPerBlock(49),
		statemanager.WithNumOpcodesPerBigStep(7),
	)
	if err != nil {
		panic(err)
	}
	return sm
}

func loadTransactOpts() *bind.TransactOpts {
	// This example doesn't create real transaction options, but your application should.
	// You will want to use something like bind.NewKeyedTransactorWithChainID to create these
	// TransactOpts.
	return &bind.TransactOpts{}
}
