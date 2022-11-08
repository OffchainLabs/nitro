package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const testChallengePeriod = 100 * time.Second

func TestAssertionChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(200)
	wrongBlockHashes := wrongBlockHashesForTest(200)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	chain := NewAssertionChain(ctx, timeRef, testChallengePeriod).inner
	require.Equal(t, 1, len(chain.assertions))
	require.Equal(t, 0, chain.confirmedLatest)
	genesis := chain.LatestConfirmed()
	require.Equal(t, StateCommitment{
		height: 0,
		state:  common.Hash{},
	}, genesis)

	eventChan := chain.feed.Subscribe(ctx)

	// add an assertion, then confirm it
	comm := StateCommitment{1, correctBlockHashes[99]}
	newAssertion, err := chain.CreateLeaf(genesis, comm, staker1)
	require.NoError(t, err)
	require.Equal(t, 2, len(chain.assertions))
	require.Equal(t, genesis, chain.LatestConfirmed())
	verifyCreateLeafEventInFeed(t, eventChan, 1, 0, staker1, comm)

	err = newAssertion.ConfirmNoRival()
	require.ErrorIs(t, err, ErrNotYet)
	timeRef.Add(testChallengePeriod + time.Second)
	require.NoError(t, newAssertion.ConfirmForWin())

	require.Equal(t, newAssertion, chain.LatestConfirmed())
	require.Equal(t, ConfirmedAssertionState, newAssertion.status)
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
	verifyStartChallengeEventInFeed(t, eventChan, newAssertion.sequenceNum)
	chal1, err := challenge.AddLeaf(branch1, util.HistoryCommitment{100, util.ExpansionFromLeaves(correctBlockHashes[99:200]).Root()})
	require.NoError(t, err)
	_, err = challenge.AddLeaf(branch2, util.HistoryCommitment{100, util.ExpansionFromLeaves(wrongBlockHashes[99:200]).Root()})
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

func verifyCreateLeafEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum, prevSeqNum uint64, staker common.Address, comm StateCommitment) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *CreateLeafEvent:
		if e.seqNum != seqNum || e.prevSeqNum != prevSeqNum || e.staker != staker || e.commitment != comm {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func verifyConfirmEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *ConfirmEvent:
		if e.seqNum != seqNum {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func verifyRejectEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *RejectEvent:
		if e.seqNum != seqNum {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func verifyStartChallengeEventInFeed(t *testing.T, c <-chan AssertionChainEvent, parentSeqNum uint64) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *StartChallengeEvent:
		if e.parentSeqNum != parentSeqNum {
			t.Fatal(e)
		}
	default:
		t.Fatal()
	}
}

func TestChallengeBisections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(200)
	wrongBlockHashes := wrongBlockHashesForTest(200)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	chain := NewAssertionChain(ctx, timeRef, testChallengePeriod).inner
	correctBranch, err := chain.CreateLeaf(chain.LatestConfirmed(), StateCommitment{100, correctBlockHashes[100]}, staker1)
	require.NoError(t, err)
	wrongBranch, err := chain.CreateLeaf(chain.LatestConfirmed(), StateCommitment{100, wrongBlockHashes[100]}, staker2)
	require.NoError(t, err)
	challenge, err := chain.LatestConfirmed().CreateChallenge(ctx)
	require.NoError(t, err)
	correctLeaf, err := challenge.AddLeaf(correctBranch, util.HistoryCommitment{100, util.ExpansionFromLeaves(correctBlockHashes[101:200]).Root()})
	require.NoError(t, err)
	wrongLeaf, err := challenge.AddLeaf(wrongBranch, util.HistoryCommitment{100, util.ExpansionFromLeaves(wrongBlockHashes[101:200]).Root()})
	require.NoError(t, err)

	_ = correctLeaf
	_ = wrongLeaf
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
