package solimpl

import (
	"context"
	"testing"

	"crypto/rand"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/stretchr/testify/require"
	"math/big"
)

var (
	_ = protocol.SpecEdge(&SpecEdge{})
	_ = protocol.SpecChallengeManager(&SpecChallengeManager{})
)

func TestEdgeChallengeManager(t *testing.T) {
	ctx := context.Background()
	height := protocol.Height(3)

	createdData := createTwoValidatorFork(t, ctx, &createForkConfig{
		numBlocks:     uint64(height) + 1,
		divergeHeight: 0,
	})

	honestStateManager, err := statemanager.New(
		createdData.honestValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
	)
	require.NoError(t, err)

	evilStateManager, err := statemanager.New(
		createdData.evilValidatorStateRoots,
		statemanager.WithNumOpcodesPerBigStep(1),
		statemanager.WithMaxWavmOpcodesPerBlock(1),
		statemanager.WithBigStepStateDivergenceHeight(1),
		statemanager.WithSmallStepStateDivergenceHeight(1),
	)
	require.NoError(t, err)

	challengeManager, err := createdData.chains[0].SpecChallengeManager(ctx)
	require.NoError(t, err)

	genesis, err := createdData.chains[0].AssertionBySequenceNum(ctx, 0)
	require.NoError(t, err)

	// Honest assertion being added.
	startCommit := util.HistoryCommitment{
		Height: 0,
		Merkle: common.Hash{},
	}
	leafAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddBlockChallengeLevelZeroEdge(
			ctx,
			genesis,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}

	honestEndCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	t.Log("Alice creates level zero block edge")
	honestEdge := leafAdder(honestEndCommit)
	require.Equal(t, protocol.BlockChallengeEdge, honestEdge.GetType())
	isPs, err := honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isPs)
	t.Log("Alice is presumptive")

	evilEndCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, uint64(height))
	require.NoError(t, err)

	t.Log("Bob creates level zero block edge")
	evilEdge := leafAdder(evilEndCommit)
	require.Equal(t, protocol.BlockChallengeEdge, evilEdge.GetType())

	// Honest and evil edge are rivals, neither is presumptive.
	isPs, err = honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)

	isPs, err = evilEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)
	t.Log("Neither is presumptive")

	// Attempt bisections down to one step fork.
	honestBisectCommit, err := honestStateManager.HistoryCommitmentUpTo(ctx, 1)
	require.NoError(t, err)
	honestProof, err := honestStateManager.PrefixProof(ctx, 1, 3)
	require.NoError(t, err)

	t.Log("Alice bisects")
	_, _, err = honestEdge.Bisect(ctx, honestBisectCommit.Merkle, honestProof)
	require.NoError(t, err)

	evilBisectCommit, err := evilStateManager.HistoryCommitmentUpTo(ctx, 1)
	require.NoError(t, err)
	evilProof, err := evilStateManager.PrefixProof(ctx, 1, 3)
	require.NoError(t, err)

	t.Log("Bob bisects")
	oneStepForkSourceEdge, _, err := evilEdge.Bisect(ctx, evilBisectCommit.Merkle, evilProof)
	require.NoError(t, err)

	isAtOneStepFork, err := oneStepForkSourceEdge.IsOneStepForkSource(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Lower child of bisection is at one step fork")

	// Now opening big step level zero leaves
	bigStepAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddSubChallengeLevelZeroEdge(
			ctx,
			oneStepForkSourceEdge,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}

	honestBigStepCommit, err := honestStateManager.BigStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to big step */
	)
	require.NoError(t, err)

	t.Log("Alice creates level zero big step challenge edge")
	honestEdge = bigStepAdder(honestBigStepCommit)
	require.Equal(t, protocol.BigStepChallengeEdge, honestEdge.GetType())
	isPs, err = honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isPs)

	t.Log("Alice is presumptive")

	evilBigStepCommit, err := evilStateManager.BigStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to big step */
	)
	require.NoError(t, err)

	t.Log("Bob creates level zero big step challenge edge")
	evilEdge = bigStepAdder(evilBigStepCommit)
	require.Equal(t, protocol.BigStepChallengeEdge, evilEdge.GetType())

	isPs, err = honestEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)
	isPs, err = evilEdge.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)

	t.Log("Neither is presumptive")

	isAtOneStepFork, err = honestEdge.IsOneStepForkSource(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Reached one step fork at big step challenge level")

	claimHeight, err := evilEdge.TopLevelClaimHeight(ctx)
	require.NoError(t, err)
	t.Logf("Got top level claim height %d", claimHeight)

	// Now opening small step level zero leaves
	smallStepAdder := func(endCommit util.HistoryCommitment) protocol.SpecEdge {
		leaf, err := challengeManager.AddSubChallengeLevelZeroEdge(
			ctx,
			honestEdge,
			startCommit,
			endCommit,
		)
		require.NoError(t, err)
		return leaf
	}

	honestSmallStepCommit, err := honestStateManager.SmallStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to pc */
	)
	require.NoError(t, err)

	t.Log("Alice creates level zero small step challenge edge")
	smallStepHonest := smallStepAdder(honestSmallStepCommit)
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepHonest.GetType())
	isPs, err = smallStepHonest.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isPs)

	t.Log("Alice is presumptive")

	evilSmallStepCommit, err := evilStateManager.SmallStepCommitmentUpTo(
		ctx, 0 /* from assertion */, 1 /* to assertion */, 1, /* to pc */
	)
	require.NoError(t, err)

	t.Log("Bob creates level zero small step challenge edge")
	smallStepEvil := smallStepAdder(evilSmallStepCommit)
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepEvil.GetType())

	isPs, err = smallStepHonest.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)
	isPs, err = smallStepEvil.IsPresumptive(ctx)
	require.NoError(t, err)
	require.Equal(t, false, isPs)

	t.Log("Neither is presumptive")

	claimHeight, err = smallStepEvil.TopLevelClaimHeight(ctx)
	require.NoError(t, err)
	t.Logf("Got top level claim height %d", claimHeight)

	// Get the lower-level edge of either vertex we just bisected.
	require.Equal(t, protocol.SmallStepChallengeEdge, smallStepHonest.GetType())

	isAtOneStepFork, err = smallStepHonest.IsOneStepForkSource(ctx)
	require.NoError(t, err)
	require.Equal(t, true, isAtOneStepFork)

	t.Log("Reached one step proof!!!")
}

type createdValidatorFork struct {
	leaf1                     protocol.Assertion
	leaf2                     protocol.Assertion
	chains                    []*AssertionChain
	accounts                  []*testAccount
	backend                   *backends.SimulatedBackend
	honestValidatorStateRoots []common.Hash
	evilValidatorStateRoots   []common.Hash
	addrs                     *rollupAddresses
}

type createForkConfig struct {
	numBlocks     uint64
	divergeHeight uint64
}

func createTwoValidatorFork(
	t *testing.T,
	ctx context.Context,
	cfg *createForkConfig,
) *createdValidatorFork {
	divergenceHeight := cfg.divergeHeight
	numBlocks := cfg.numBlocks

	chains, accs, addresses, backend, _ := setupChainsWithEdgeChallengeManager(t)
	prevInboxMaxCount := big.NewInt(1)

	// Advance the backend by some blocks to get over time delta failures when
	// using the assertion chain.
	for i := 0; i < 100; i++ {
		backend.Commit()
	}

	genesis, err := chains[0].AssertionBySequenceNum(ctx, 0)
	require.NoError(t, err)

	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	genesisStateHash := protocol.ComputeStateHash(genesisState, big.NewInt(1))

	actualGenesisStateHash, err := genesis.StateHash()
	require.NoError(t, err)
	require.Equal(t, genesisStateHash, actualGenesisStateHash, "Genesis state hash unequal")

	height := uint64(0)
	honestValidatorStateRoots := make([]common.Hash, 0)
	evilValidatorStateRoots := make([]common.Hash, 0)
	honestValidatorStateRoots = append(honestValidatorStateRoots, genesisStateHash)
	evilValidatorStateRoots = append(evilValidatorStateRoots, genesisStateHash)

	var honestBlockHash common.Hash
	for i := uint64(1); i < numBlocks; i++ {
		height += 1
		honestBlockHash = backend.Commit()

		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestBlockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		honestValidatorStateRoots = append(honestValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))

		// Before the divergence height, the evil validator agrees.
		if i < divergenceHeight {
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
		} else {
			junkRoot := make([]byte, 32)
			_, err := rand.Read(junkRoot)
			require.NoError(t, err)
			blockHash := crypto.Keccak256Hash(junkRoot)
			state.GlobalState.BlockHash = blockHash
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
		}

	}

	height += 1
	honestBlockHash = backend.Commit()
	assertion, err := chains[0].CreateAssertion(
		ctx,
		height,
		genesis.SeqNum(),
		genesisState,
		&protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestBlockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		},
		prevInboxMaxCount,
	)
	require.NoError(t, err)

	assertionStateHash, err := assertion.StateHash()
	require.NoError(t, err)
	honestValidatorStateRoots = append(honestValidatorStateRoots, assertionStateHash)

	evilPostState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.BytesToHash([]byte("evilcommit")),
			Batch:     1,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	forkedAssertion, err := chains[1].CreateAssertion(
		ctx,
		height,
		genesis.SeqNum(),
		genesisState,
		evilPostState,
		prevInboxMaxCount,
	)
	require.NoError(t, err)

	forkedAssertionStateHash, err := forkedAssertion.StateHash()
	require.NoError(t, err)
	evilValidatorStateRoots = append(evilValidatorStateRoots, forkedAssertionStateHash)

	return &createdValidatorFork{
		leaf1:                     assertion,
		leaf2:                     forkedAssertion,
		chains:                    chains,
		accounts:                  accs,
		backend:                   backend,
		addrs:                     addresses,
		honestValidatorStateRoots: honestValidatorStateRoots,
		evilValidatorStateRoots:   evilValidatorStateRoots,
	}
}

func setupChainsWithEdgeChallengeManager(t *testing.T) (
	[]*AssertionChain, []*testAccount, *rollupAddresses, *backends.SimulatedBackend, *headerreader.HeaderReader,
) {
	t.Helper()
	ctx := context.Background()
	accs, backend := setupAccounts(t, 3)
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := accs[0].accountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := common.Address{}
	challengePeriodSeconds := big.NewInt(100)
	miniStake := big.NewInt(1)
	cfg := generateRollupConfig(
		prod,
		wasmModuleRoot,
		rollupOwner,
		chainId,
		loserStakeEscrow,
		challengePeriodSeconds,
		miniStake,
	)
	addresses := deployFullRollupStack(
		t,
		ctx,
		backend,
		accs[0].txOpts,
		common.Address{}, // Sequencer addr.
		cfg,
	)
	headerReader := headerreader.New(util.SimulatedBackendWrapper{SimulatedBackend: backend}, func() *headerreader.Config { return &headerreader.TestConfig })
	headerReader.Start(ctx)
	chains := make([]*AssertionChain, 2)
	chain1, err := NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[1].txOpts,
		&bind.CallOpts{},
		accs[1].accountAddr,
		backend,
		headerReader,
		addresses.EdgeChallengeManager,
	)
	require.NoError(t, err)
	chains[0] = chain1
	chain2, err := NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[2].txOpts,
		&bind.CallOpts{},
		accs[2].accountAddr,
		backend,
		headerReader,
		addresses.EdgeChallengeManager,
	)
	require.NoError(t, err)
	chains[1] = chain2
	return chains, accs, addresses, backend, headerReader
}
