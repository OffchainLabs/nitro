package protocol

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var _ = OnChainProtocol(&AssertionChain{})

const testChallengePeriod = 100 * time.Second

func TestAssertionChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(200)
	wrongBlockHashes := wrongBlockHashesForTest(200)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	chain := NewAssertionChain(ctx, timeRef, testChallengePeriod)
	require.Equal(t, 1, len(chain.assertions))
	require.Equal(t, uint64(0), chain.confirmedLatest)
	genesis := chain.LatestConfirmed()
	require.Equal(t, StateCommitment{
		Height:    0,
		StateRoot: common.Hash{},
	}, genesis.StateCommitment)

	bigBalance := new(big.Int).Mul(AssertionStakeWei, big.NewInt(1000))
	chain.SetBalance(staker1, bigBalance)
	chain.SetBalance(staker2, bigBalance)

	eventChan := make(chan AssertionChainEvent)
	chain.feed.SubscribeWithFilter(ctx, eventChan, func(ev AssertionChainEvent) bool {
		switch ev.(type) {
		case *SetBalanceEvent:
			return false
		default:
			return true
		}
	})

	// add an assertion, then confirm it
	comm := StateCommitment{Height: 1, StateRoot: correctBlockHashes[99]}
	newAssertion, err := chain.CreateLeaf(genesis, comm, staker1)
	require.NoError(t, err)
	require.Equal(t, 2, len(chain.assertions))
	require.Equal(t, genesis, chain.LatestConfirmed())
	verifyCreateLeafEventInFeed(t, eventChan, 1, 0, staker1, comm)
	require.True(t, new(big.Int).Add(chain.GetBalance(staker1), AssertionStakeWei).Cmp(bigBalance) == 0)

	err = newAssertion.ConfirmNoRival()
	require.ErrorIs(t, err, ErrNotYet)
	timeRef.Add(testChallengePeriod + time.Second)
	require.NoError(t, newAssertion.ConfirmNoRival())
	require.True(t, chain.GetBalance(staker1).Cmp(bigBalance) == 0)

	require.Equal(t, newAssertion, chain.LatestConfirmed())
	require.Equal(t, ConfirmedAssertionState, int(newAssertion.status))
	verifyConfirmEventInFeed(t, eventChan, 1)

	// try to create a duplicate assertion
	_, err = chain.CreateLeaf(genesis, StateCommitment{1, correctBlockHashes[99]}, staker1)
	require.ErrorIs(t, err, ErrVertexAlreadyExists)

	// create a fork, let first branch win by timeout
	comm = StateCommitment{2, correctBlockHashes[199]}
	branch1, err := chain.CreateLeaf(newAssertion, comm, staker1)
	require.NoError(t, err)
	timeRef.Add(5 * time.Second)
	verifyCreateLeafEventInFeed(t, eventChan, 2, 1, staker1, comm)
	comm = StateCommitment{2, wrongBlockHashes[199]}
	branch2, err := chain.CreateLeaf(newAssertion, comm, staker2)
	require.NoError(t, err)
	verifyCreateLeafEventInFeed(t, eventChan, 3, 1, staker2, comm)
	challenge, err := newAssertion.CreateChallenge(ctx)
	require.NoError(t, err)
	verifyStartChallengeEventInFeed(t, eventChan, newAssertion.SequenceNum)
	chal1, err := challenge.AddLeaf(branch1, util.HistoryCommitment{Height: 100, Merkle: util.ExpansionFromLeaves(correctBlockHashes[99:200]).Root()})
	require.NoError(t, err)
	_, err = challenge.AddLeaf(branch2, util.HistoryCommitment{Height: 100, Merkle: util.ExpansionFromLeaves(wrongBlockHashes[99:200]).Root()})
	require.NoError(t, err)
	err = chal1.ConfirmForPsTimer()
	require.ErrorIs(t, err, ErrNotYet)

	timeRef.Add(testChallengePeriod)
	require.NoError(t, chal1.ConfirmForPsTimer())
	require.NoError(t, branch1.ConfirmForWin())
	require.Equal(t, branch1, chain.LatestConfirmed())

	verifyConfirmEventInFeed(t, eventChan, 2)
	require.NoError(t, branch2.RejectForLoss())
	verifyRejectEventInFeed(t, eventChan, 3)

	// verify that feed is empty
	time.Sleep(500 * time.Millisecond)
	select {
	case ev := <-eventChan:
		t.Fatal(ev)
	default:
	}
}

func TestAssertionChain_LeafCreationThroughDiffStakers(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)

	oldStaker := common.BytesToAddress([]byte{1})
	staker := common.BytesToAddress([]byte{2})
	require.Equal(t, chain.GetBalance(oldStaker), big.NewInt(0)) // Old staker has 0 because it's already staked.
	chain.SetBalance(staker, AssertionStakeWei)
	require.Equal(t, chain.GetBalance(staker), AssertionStakeWei) // New staker has full balance because it's not yet staked.

	lc := chain.LatestConfirmed()
	lc.staker = util.FullOption[common.Address](oldStaker)
	_, err := chain.CreateLeaf(lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
	require.NoError(t, err)

	require.Equal(t, chain.GetBalance(staker), big.NewInt(0))        // New staker has 0 balance after staking.
	require.Equal(t, chain.GetBalance(oldStaker), AssertionStakeWei) // Old staker has full balance after unstaking.
}

func TestAssertionChain_LeafCreationsInsufficientStakes(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	lc := chain.LatestConfirmed()

	staker := common.BytesToAddress([]byte{1})
	lc.staker = util.EmptyOption[common.Address]()
	_, err := chain.CreateLeaf(lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
	require.ErrorIs(t, err, ErrInsufficientBalance)

	diffStaker := common.BytesToAddress([]byte{2})
	lc.staker = util.FullOption[common.Address](diffStaker)
	_, err = chain.CreateLeaf(lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
	require.ErrorIs(t, err, ErrInsufficientBalance)
}

func verifyCreateLeafEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum, prevSeqNum uint64, staker common.Address, comm StateCommitment) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *CreateLeafEvent:
		if e.SeqNum != seqNum || e.PrevSeqNum != prevSeqNum || e.Staker != staker || e.StateCommitment != comm {
			t.Fatal(e)
		}
	default:
		t.Fatal(e)
	}
}

func verifyConfirmEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *ConfirmEvent:
		require.Equal(t, seqNum, e.SeqNum)
	default:
		t.Fatal()
	}
}

func verifyRejectEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *RejectEvent:
		require.Equal(t, seqNum, e.SeqNum)
	default:
		t.Fatal()
	}
}

func verifyStartChallengeEventInFeed(t *testing.T, c <-chan AssertionChainEvent, parentSeqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *StartChallengeEvent:
		require.Equal(t, parentSeqNum, e.ParentSeqNum)
	default:
		t.Fatal()
	}
}

func TestBisectionChallengeGame(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(8)
	wrongBlockHashes := wrongBlockHashesForTest(8)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	chain := NewAssertionChain(ctx, timeRef, testChallengePeriod)

	bigBalance := new(big.Int).Mul(AssertionStakeWei, big.NewInt(1000))
	chain.SetBalance(staker1, bigBalance)
	chain.SetBalance(staker2, bigBalance)

	// We create a fork with genesis as the parent, where one branch is a higher depth than the other.
	genesis := chain.LatestConfirmed()
	correctBranch, err := chain.CreateLeaf(genesis, StateCommitment{6, correctBlockHashes[6]}, staker1)
	require.NoError(t, err)
	wrongBranch, err := chain.CreateLeaf(genesis, StateCommitment{7, wrongBlockHashes[7]}, staker2)
	require.NoError(t, err)

	challenge, err := genesis.CreateChallenge(ctx)
	require.NoError(t, err)

	// Add some leaves to the mix...
	expectedBisectionHeight := uint64(4)
	lo := expectedBisectionHeight
	hi := uint64(7)
	loExp := util.ExpansionFromLeaves(wrongBlockHashes[:lo])
	hiExp := util.ExpansionFromLeaves(wrongBlockHashes[:hi])

	cl1, err := challenge.AddLeaf(
		wrongBranch,
		util.HistoryCommitment{
			Height: 6,
			Merkle: util.ExpansionFromLeaves(correctBlockHashes[:7]).Root(),
		},
	)
	require.NoError(t, err)
	cl2, err := challenge.AddLeaf(
		correctBranch,
		util.HistoryCommitment{
			Height: 7,
			Merkle: hiExp.Root(),
		},
	)
	require.NoError(t, err)

	// Ensure the lower height challenge vertex is the ps.
	require.Equal(t, true, cl1.isPresumptiveSuccessor())
	require.Equal(t, false, cl2.isPresumptiveSuccessor())

	// Next, only the vertex that is not the presumptive successor can start a bisection move.
	bisectionHeight, err := cl2.requiredBisectionHeight()
	require.NoError(t, err)
	require.Equal(t, expectedBisectionHeight, bisectionHeight)

	proof := util.GeneratePrefixProof(lo, loExp, correctBlockHashes[lo:6])
	_, err = cl1.Bisect(
		util.HistoryCommitment{
			Height: lo,
			Merkle: loExp.Root(),
		},
		proof,
	)
	require.ErrorIs(t, err, ErrWrongState)

	// Generate a prefix proof for the associated history commitments from the bisection
	// height up to the height of the state commitment for the non-presumptive challenge leaf.
	proof = util.GeneratePrefixProof(lo, loExp, wrongBlockHashes[lo:hi])
	bisection, err := cl2.Bisect(
		util.HistoryCommitment{
			Height: lo,
			Merkle: loExp.Root(),
		},
		proof,
	)
	require.NoError(t, err)

	// The parent of the bisectoin should be the root of this challenge and the bisection
	// should be the new presumptive successor.
	require.Equal(t, challenge.root.commitment.Merkle, bisection.prev.commitment.Merkle)
	require.Equal(t, true, bisection.prev.isPresumptiveSuccessor())
}

func correctBlockHashesForTest(numBlocks uint64) []common.Hash {
	ret := []common.Hash{}
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(i))
	}
	return ret
}

func wrongBlockHashesForTest(numBlocks uint64) []common.Hash {
	ret := []common.Hash{}
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(71285937102384-i))
	}
	return ret
}
