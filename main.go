package main

import (
	"context"
	"math/big"

	"encoding/binary"
	"math"
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var (
	// The chain id for the backend.
	chainId = big.NewInt(1337)
	// The number of seconds in a challenge period.
	challengePeriodSeconds = big.NewInt(100)
	// The size of a mini stake that is posted when creating leaf edges in
	// challenges (clarify if gwei?).
	miniStakeSize = big.NewInt(1)
	// The current L2 chain height for this simulation.
	currentL2ChainHeight = uint64(7)
	// The number of wavm opcodes per block (all blocks are equal in this sim, but not IRL).
	maxWavmOpcodesPerBlock = uint64(49)
	// Number of opcodes in a big step within a big step subchallenge.
	numOpcodesPerBigStep = uint64(7)
	// The heights at which Alice and Bob diverge at each challenge level.
	divergeHeightAtL2 = uint64(3)
	// How often an edge tracker needs to wake and perform its responsibilities.
	edgeTrackerWakeInterval = time.Millisecond * 500
	// How often the validator polls the chain to see if new assertions have been posted.
	checkForAssertionsInteral = time.Second
	// How often the validator will post its latest assertion to the chain.
	postNewAssertionInterval = time.Hour
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

	honestL2StateHashes := honestL2StateHashesForUints(0, currentL2ChainHeight+1)
	evilL2StateHashes := evilL2StateHashesForUints(0, currentL2ChainHeight+1)

	// Creates honest and evil L2 states. These will be equal up to a divergence height.
	// These are toy hashes because this is a simulation and the L1 chain knows nothing about
	// the real L2 state hashes except for what validators claim.
	honestL2States, honestInboxCounts := prepareHonestL2States(
		honestL2StateHashes,
		currentL2ChainHeight,
	)

	evilL2States, evilInboxCounts := prepareMaliciousL2States(
		divergeHeightAtL2,
		evilL2StateHashes,
		honestL2States,
		honestInboxCounts,
	)

	// Initialize Alice and Bob's respective L2 state managers.
	managerOpts := []statemanager.Opt{
		statemanager.WithMaxWavmOpcodesPerBlock(maxWavmOpcodesPerBlock),
		statemanager.WithNumOpcodesPerBigStep(numOpcodesPerBigStep),
	}
	aliceL2StateManager, err := statemanager.NewWithAssertionStates(
		honestL2States,
		honestInboxCounts,
		managerOpts...,
	)
	if err != nil {
		panic(err)
	}

	// Bob diverges from Alice's L2 history at the specified divergence height.
	managerOpts = append(
		managerOpts,
		statemanager.WithMaliciousIntent(),
		statemanager.WithBigStepStateDivergenceHeight(divergeHeightAtL2),
		statemanager.WithSmallStepStateDivergenceHeight(divergeHeightAtL2),
	)
	bobL2StateManager, err := statemanager.NewWithAssertionStates(
		evilL2States,
		evilInboxCounts,
		managerOpts...,
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
		challengePeriodSeconds,
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

func prepareHonestL2States(
	honestHashes []common.Hash,
	chainHeight uint64,
) ([]*protocol.ExecutionState, []*big.Int) {
	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}

	// Initialize each validator associated state roots which diverge
	// at specified points in the test config.
	honestStates := make([]*protocol.ExecutionState, chainHeight+1)
	honestInboxCounts := make([]*big.Int, chainHeight+1)
	honestStates[0] = genesisState
	honestInboxCounts[0] = big.NewInt(1)

	for i := uint64(1); i <= chainHeight; i++ {
		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestHashes[i],
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		honestStates[i] = state
		honestInboxCounts[i] = big.NewInt(1)
	}
	return honestStates, honestInboxCounts
}

func prepareMaliciousL2States(
	assertionDivergenceHeight uint64,
	evilHashes []common.Hash,
	honestStates []*protocol.ExecutionState,
	honestInboxCounts []*big.Int,
) ([]*protocol.ExecutionState, []*big.Int) {
	divergenceHeight := assertionDivergenceHeight
	numRoots := currentL2ChainHeight + 1
	states := make([]*protocol.ExecutionState, numRoots)
	inboxCounts := make([]*big.Int, numRoots)

	for j := uint64(0); j < numRoots; j++ {
		if divergenceHeight == 0 || j < divergenceHeight {
			states[j] = honestStates[j]
			inboxCounts[j] = honestInboxCounts[j]
		} else {
			evilState := &protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash: evilHashes[j],
					Batch:     1,
				},
				MachineStatus: protocol.MachineStatusFinished,
			}
			states[j] = evilState
			inboxCounts[j] = big.NewInt(1)
		}
	}
	return states, inboxCounts
}

func evilL2StateHashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, hashForUint(math.MaxUint64-i))
	}
	return ret
}

func honestL2StateHashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, hashForUint(i))
	}
	return ret
}

func hashForUint(x uint64) common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, x))
}
