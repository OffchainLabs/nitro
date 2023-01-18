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

func TestAssertionChain_ConfirmAndRefund(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(200)
	staker := common.BytesToAddress([]byte{1})

	assertionsChain := NewAssertionChain(ctx, timeRef, testChallengePeriod)
	require.Equal(t, 1, len(assertionsChain.assertions))
	require.Equal(t, AssertionSequenceNumber(0), assertionsChain.latestConfirmed)
	err := assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
		chain := p.(*AssertionChain)

		chain.SetBalance(tx, staker, AssertionStake)
		genesis := p.LatestConfirmed(tx)
		comm := StateCommitment{Height: 1, StateRoot: correctBlockHashes[99]}
		a1, err := chain.CreateLeaf(tx, genesis, comm, staker)
		require.NoError(t, err)
		require.Equal(t, uint64(0), chain.GetBalance(tx, staker).Uint64())

		comm = StateCommitment{2, correctBlockHashes[199]}
		a2, err := chain.CreateLeaf(tx, a1, comm, staker)
		require.NoError(t, err)
		require.Equal(t, uint64(0), chain.GetBalance(tx, staker).Uint64())
		timeRef.Add(testChallengePeriod + time.Second)

		// Parent is confirmed. Staker should not get a refund because it's not a leaf.
		require.NoError(t, a1.ConfirmNoRival(tx))
		require.Equal(t, uint64(0), chain.GetBalance(tx, staker).Uint64())

		// Child is confirmed. Staker should get a refund because it's a leaf.
		require.NoError(t, a2.ConfirmNoRival(tx))
		require.Equal(t, AssertionStake.Uint64(), chain.GetBalance(tx, staker).Uint64())

		return nil
	})

	require.NoError(t, err)
}

func TestAssertionChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(200)
	wrongBlockHashes := wrongBlockHashesForTest(200)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	assertionsChain := NewAssertionChain(ctx, timeRef, testChallengePeriod)
	require.Equal(t, 1, len(assertionsChain.assertions))
	require.Equal(t, AssertionSequenceNumber(0), assertionsChain.latestConfirmed)
	eventChan := make(chan AssertionChainEvent)
	err := assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
		chain := p.(*AssertionChain)
		genesis := p.LatestConfirmed(tx)
		require.Equal(t, StateCommitment{
			Height:    0,
			StateRoot: common.Hash{},
		}, genesis.StateCommitment)
		chain.SetBalance(tx, staker1, big.NewInt(0).Add(AssertionStake, ChallengeVertexStake))
		chain.SetBalance(tx, staker2, big.NewInt(0).Add(AssertionStake, ChallengeVertexStake))

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
		newAssertion, err := chain.CreateLeaf(tx, genesis, comm, staker1)
		require.NoError(t, err)
		require.Equal(t, 2, len(chain.assertions))
		require.Equal(t, genesis, chain.LatestConfirmed(tx))
		verifyCreateLeafEventInFeed(t, eventChan, 1, 0, staker1, comm)

		err = newAssertion.ConfirmNoRival(tx)
		require.ErrorIs(t, err, ErrNotYet)
		timeRef.Add(testChallengePeriod + time.Second)
		require.NoError(t, newAssertion.ConfirmNoRival(tx))

		require.Equal(t, newAssertion, chain.LatestConfirmed(tx))
		require.Equal(t, ConfirmedAssertionState, int(newAssertion.status))
		verifyConfirmEventInFeed(t, eventChan, AssertionSequenceNumber(1))

		// try to create a duplicate assertion
		_, err = chain.CreateLeaf(tx, genesis, StateCommitment{1, correctBlockHashes[99]}, staker1)
		require.ErrorIs(t, err, ErrVertexAlreadyExists)

		// create a fork, let first branch win by timeout
		comm = StateCommitment{2, correctBlockHashes[199]}

		branch1, err := chain.CreateLeaf(tx, newAssertion, comm, staker1)
		require.NoError(t, err)

		timeRef.Add(5 * time.Second)
		verifyCreateLeafEventInFeed(t, eventChan, 2, 1, staker1, comm)
		comm = StateCommitment{2, wrongBlockHashes[199]}
		branch2, err := chain.CreateLeaf(tx, newAssertion, comm, staker2)
		require.NoError(t, err)
		verifyCreateLeafEventInFeed(t, eventChan, 3, 1, staker2, comm)
		challenge, err := newAssertion.CreateChallenge(tx, ctx, staker2)
		require.NoError(t, err)
		verifyStartChallengeEventInFeed(t, eventChan, newAssertion.SequenceNum)

		chal1, err := challenge.AddLeaf(tx, branch1, util.HistoryCommitment{Height: 100, Merkle: util.ExpansionFromLeaves(correctBlockHashes[99:200]).Root()}, staker1)
		require.NoError(t, err)

		_, err = challenge.AddLeaf(tx, branch2, util.HistoryCommitment{Height: 100, Merkle: util.ExpansionFromLeaves(wrongBlockHashes[99:200]).Root()}, staker2)
		require.NoError(t, err)
		err = chal1.ConfirmForPsTimer(tx)
		require.ErrorIs(t, err, ErrNotYet)

		timeRef.Add(testChallengePeriod)
		require.NoError(t, chal1.ConfirmForPsTimer(tx))
		require.Equal(t, ChallengeVertexStake, chain.GetBalance(tx, chal1.Validator)) // Should receive challenge vertex stake back.
		require.NoError(t, branch1.ConfirmForWin(tx))
		require.Equal(t, branch1, chain.LatestConfirmed(tx))

		verifyConfirmEventInFeed(t, eventChan, 2)
		require.NoError(t, branch2.RejectForLoss(tx))
		verifyRejectEventInFeed(t, eventChan, 3)
		return nil
	})
	require.NoError(t, err)

	// verify that feed is empty
	time.Sleep(500 * time.Millisecond)
	select {
	case ev := <-eventChan:
		t.Fatal(ev)
	default:
	}
}

func TestAssertionChain_CreateLeaf_MustHaveValidParent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	staker := common.BytesToAddress([]byte{1})

	assertionsChain := NewAssertionChain(ctx, timeRef, testChallengePeriod)
	require.Equal(t, 1, len(assertionsChain.assertions))
	require.Equal(t, AssertionSequenceNumber(0), assertionsChain.latestConfirmed)
	err := assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
		chain := p.(*AssertionChain)
		genesis := p.LatestConfirmed(tx)
		require.Equal(t, StateCommitment{
			Height:    0,
			StateRoot: common.Hash{},
		}, genesis.StateCommitment)

		bigBalance := new(big.Int).Mul(AssertionStake, big.NewInt(1000))
		chain.SetBalance(tx, staker, bigBalance)

		foo := common.BytesToHash([]byte("foo"))
		bar := common.BytesToHash([]byte("bar"))
		_ = bar
		comm := StateCommitment{Height: 1, StateRoot: foo}
		leaf, err := chain.CreateLeaf(tx, genesis, comm, staker)
		require.NoError(t, err)

		// Trying to create a new leaf with the same commitment as before should fail.
		leaf.StateCommitment = StateCommitment{Height: 0, StateRoot: bar} // Mutate leaf.
		_, err = chain.CreateLeaf(tx, leaf, comm, staker)
		require.ErrorIs(t, err, ErrVertexAlreadyExists)

		// Trying to create a new leaf on top of a non-existent parent should fail.
		leaf.StateCommitment = StateCommitment{Height: 0, StateRoot: bar} // Mutate leaf.
		comm = StateCommitment{Height: 2, StateRoot: foo}
		_, err = chain.CreateLeaf(tx, leaf, comm, staker)
		require.ErrorIs(t, err, ErrParentDoesNotExist)
		return nil
	})
	require.NoError(t, err)
}

func TestAssertionChain_LeafCreationThroughDiffStakers(t *testing.T) {
	ctx := context.Background()
	assertionsChain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)

	require.NoError(t, assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
		chain := p.(*AssertionChain)
		oldStaker := common.BytesToAddress([]byte{1})
		staker := common.BytesToAddress([]byte{2})
		require.Equal(t, chain.GetBalance(tx, oldStaker), big.NewInt(0)) // Old staker has 0 because it's already staked.
		chain.SetBalance(tx, staker, AssertionStake)
		require.Equal(t, chain.GetBalance(tx, staker), AssertionStake) // New staker has full balance because it's not yet staked.

		lc := chain.LatestConfirmed(tx)
		lc.Staker = util.Some[common.Address](oldStaker)
		_, err := chain.CreateLeaf(tx, lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
		require.NoError(t, err)

		require.Equal(t, chain.GetBalance(tx, staker), big.NewInt(0))     // New staker has 0 balance after staking.
		require.Equal(t, chain.GetBalance(tx, oldStaker), AssertionStake) // Old staker has full balance after unstaking.
		return nil
	}))
}

func TestAssertionChain_LeafCreationsInsufficientStakes(t *testing.T) {
	ctx := context.Background()
	assertionsChain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)

	require.NoError(t, assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
		chain := p.(*AssertionChain)
		lc := chain.LatestConfirmed(tx)
		staker := common.BytesToAddress([]byte{1})
		lc.Staker = util.None[common.Address]()
		_, err := chain.CreateLeaf(tx, lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
		require.ErrorIs(t, err, ErrInsufficientBalance)

		diffStaker := common.BytesToAddress([]byte{2})
		lc.Staker = util.Some[common.Address](diffStaker)
		_, err = chain.CreateLeaf(tx, lc, StateCommitment{Height: 1, StateRoot: common.Hash{}}, staker)
		require.ErrorIs(t, err, ErrInsufficientBalance)
		return nil
	}))
}

func verifyCreateLeafEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum, prevSeqNum AssertionSequenceNumber, staker common.Address, comm StateCommitment) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *CreateLeafEvent:
		if e.SeqNum != seqNum || e.PrevSeqNum != prevSeqNum || e.Validator != staker || e.StateCommitment != comm {
			t.Fatal(e)
		}
	default:
		t.Fatal(e)
	}
}

func verifyConfirmEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum AssertionSequenceNumber) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *ConfirmEvent:
		require.Equal(t, seqNum, e.SeqNum)
	default:
		t.Fatal()
	}
}

func verifyRejectEventInFeed(t *testing.T, c <-chan AssertionChainEvent, seqNum AssertionSequenceNumber) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *RejectEvent:
		require.Equal(t, seqNum, e.SeqNum)
	default:
		t.Fatal()
	}
}

func verifyStartChallengeEventInFeed(t *testing.T, c <-chan AssertionChainEvent, parentSeqNum AssertionSequenceNumber) {
	t.Helper()
	ev := <-c
	switch e := ev.(type) {
	case *StartChallengeEvent:
		require.Equal(t, parentSeqNum, e.ParentSeqNum)
	default:
		t.Fatal()
	}
}

func TestIsAtOneStepFork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	assertionsChain := NewAssertionChain(ctx, timeRef, testChallengePeriod)
	genesisCommitHash := ChallengeCommitHash((StateCommitment{}).Hash())

	tests := []struct {
		name         string
		vertexHeight uint64
		parentHeight uint64
		vertices     map[VertexCommitHash]*ChallengeVertex
		want         bool
	}{
		{
			name:         "height difference != 1 in inputs",
			vertexHeight: 2,
			parentHeight: 0,
			want:         false,
			vertices:     nil,
		},
		{
			name:         "empty list of vertices despite height difference == 1 in inputs",
			vertexHeight: 1,
			parentHeight: 0,
			want:         false,
			vertices:     nil,
		},
		{
			name:         "only one vertex",
			vertexHeight: 1,
			parentHeight: 0,
			want:         false,
			vertices: map[VertexCommitHash]*ChallengeVertex{
				VertexCommitHash{1}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 1,
						Merkle: common.BytesToHash([]byte{1}),
					},
				},
			},
		},
		{
			name:         "no vertices with matching parent commitment",
			vertexHeight: 1,
			parentHeight: 0,
			want:         false,
			vertices: map[VertexCommitHash]*ChallengeVertex{
				VertexCommitHash{1}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{
							Height: 5,
							Merkle: common.BytesToHash([]byte{5}),
						},
					}),
					Commitment: util.HistoryCommitment{
						Height: 6,
						Merkle: common.BytesToHash([]byte{6}),
					},
				},
			},
		},
		{
			name:         "two vertices but only one is has height difference == 1",
			vertexHeight: 1,
			parentHeight: 0,
			want:         false,
			vertices: map[VertexCommitHash]*ChallengeVertex{
				VertexCommitHash{1}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 1,
						Merkle: common.BytesToHash([]byte{1}),
					},
				},
				VertexCommitHash{2}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 2,
						Merkle: common.BytesToHash([]byte{1}),
					},
				},
			},
		},
		{
			name:         "two vertices at one-step-fork",
			vertexHeight: 1,
			parentHeight: 0,
			want:         true,
			vertices: map[VertexCommitHash]*ChallengeVertex{
				VertexCommitHash{1}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 1,
						Merkle: common.BytesToHash([]byte{1}),
					},
				},
				VertexCommitHash{2}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 1,
						Merkle: common.BytesToHash([]byte{2}),
					},
				},
			},
		},
		{
			name:         "three vertices with only two at one-step-fork",
			vertexHeight: 1,
			parentHeight: 0,
			want:         false,
			vertices: map[VertexCommitHash]*ChallengeVertex{
				VertexCommitHash{1}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 1,
						Merkle: common.BytesToHash([]byte{1}),
					},
				},
				VertexCommitHash{2}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 1,
						Merkle: common.BytesToHash([]byte{2}),
					},
				},
				VertexCommitHash{3}: {
					Prev: util.Some(&ChallengeVertex{
						Commitment: util.HistoryCommitment{},
					}),
					Commitment: util.HistoryCommitment{
						Height: 2,
						Merkle: common.BytesToHash([]byte{3}),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
				vertexCommit := util.HistoryCommitment{
					Height: tt.vertexHeight,
				}
				parentCommit := util.HistoryCommitment{
					Height: tt.parentHeight,
				}
				assertionsChain.challengeVerticesByCommitHash = make(map[ChallengeCommitHash]map[VertexCommitHash]*ChallengeVertex)
				assertionsChain.challengeVerticesByCommitHash[genesisCommitHash] = tt.vertices
				ok, err := assertionsChain.IsAtOneStepFork(
					tx,
					genesisCommitHash,
					vertexCommit,
					parentCommit,
				)
				require.NoError(t, err)
				require.Equal(t, tt.want, ok)
				return nil
			})
			require.NoError(t, err)
		})
	}
}

func TestChallengeVertexByHistoryCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	assertionsChain := NewAssertionChain(ctx, timeRef, testChallengePeriod)

	err := assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
		chain := p.(*AssertionChain)

		genesisCommitHash := ChallengeCommitHash((StateCommitment{}).Hash())
		t.Run("vertices not found for challenge", func(t *testing.T) {
			vertexCommit := util.HistoryCommitment{
				Height: 1,
			}
			_, err := chain.ChallengeVertexByCommitHash(
				tx,
				genesisCommitHash,
				VertexCommitHash(vertexCommit.Merkle),
			)
			require.ErrorContains(t, err, "challenge vertices not found")
		})
		t.Run("vertex with commit not found", func(t *testing.T) {
			vertexCommit := util.HistoryCommitment{
				Height: 1,
			}
			vertices := map[VertexCommitHash]*ChallengeVertex{}
			chain.challengeVerticesByCommitHash[genesisCommitHash] = vertices
			_, err := chain.ChallengeVertexByCommitHash(
				tx,
				genesisCommitHash,
				VertexCommitHash(vertexCommit.Merkle),
			)
			require.ErrorContains(t, err, "not found")
		})
		t.Run("vertex found", func(t *testing.T) {
			vertexCommit := util.HistoryCommitment{
				Height: 1,
				Merkle: common.Hash(VertexCommitHash{10}),
			}
			want := &ChallengeVertex{
				Commitment: vertexCommit,
			}
			vertices := map[VertexCommitHash]*ChallengeVertex{
				VertexCommitHash{10}: want,
			}
			chain.challengeVerticesByCommitHash[genesisCommitHash] = vertices
			got, err := chain.ChallengeVertexByCommitHash(
				tx,
				genesisCommitHash,
				VertexCommitHash(vertexCommit.Merkle),
			)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
		return nil
	})
	require.NoError(t, err)
}

func TestAssertionChain_Bisect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeRef := util.NewArtificialTimeReference()
	correctBlockHashes := correctBlockHashesForTest(10)
	wrongBlockHashes := wrongBlockHashesForTest(10)
	staker1 := common.BytesToAddress([]byte{1})
	staker2 := common.BytesToAddress([]byte{2})

	assertionsChain := NewAssertionChain(ctx, timeRef, testChallengePeriod)

	err := assertionsChain.Tx(func(tx *ActiveTx, p OnChainProtocol) error {
		chain := p.(*AssertionChain)
		// We create a fork with genesis as the rootAssertion, where one branch is a higher depth than the other.
		genesis := chain.LatestConfirmed(tx)
		bigBalance := new(big.Int).Mul(AssertionStake, big.NewInt(1000))
		chain.SetBalance(tx, staker1, bigBalance)
		chain.SetBalance(tx, staker2, bigBalance)

		correctBranch, err := chain.CreateLeaf(tx, genesis, StateCommitment{6, correctBlockHashes[6]}, staker1)
		require.NoError(t, err)
		wrongBranch, err := chain.CreateLeaf(tx, genesis, StateCommitment{7, wrongBlockHashes[7]}, staker2)
		require.NoError(t, err)

		challenge, err := genesis.CreateChallenge(tx, ctx, staker2)
		require.NoError(t, err)

		// Add some leaves to the mix...
		expectedBisectionHeight := uint64(4)
		lo := expectedBisectionHeight
		hi := uint64(7)
		loExp := util.ExpansionFromLeaves(wrongBlockHashes[:lo])
		hiExp := util.ExpansionFromLeaves(wrongBlockHashes[:hi])

		cl1, err := challenge.AddLeaf(
			tx,
			wrongBranch,
			util.HistoryCommitment{
				Height: 6,
				Merkle: util.ExpansionFromLeaves(correctBlockHashes[:7]).Root(),
			},
			staker1,
		)
		require.NoError(t, err)
		cl2, err := challenge.AddLeaf(
			tx,
			correctBranch,
			util.HistoryCommitment{
				Height: 7,
				Merkle: hiExp.Root(),
			},
			staker2,
		)
		require.NoError(t, err)

		// Ensure the lower height challenge vertex is the ps.
		require.Equal(t, true, cl1.IsPresumptiveSuccessor())
		require.Equal(t, false, cl2.IsPresumptiveSuccessor())

		// Next, only the vertex that is not the presumptive successor can start a bisection move.
		bisectionHeight, err := cl2.requiredBisectionHeight()
		require.NoError(t, err)
		require.Equal(t, expectedBisectionHeight, bisectionHeight)

		proof := util.GeneratePrefixProof(lo, loExp, correctBlockHashes[lo:6])
		_, err = cl1.Bisect(
			tx,
			util.HistoryCommitment{
				Height: lo,
				Merkle: loExp.Root(),
			},
			proof,
			staker1,
		)
		require.ErrorIs(t, err, ErrWrongState)

		// Generate a prefix proof for the associated history commitments from the bisection
		// height up to the height of the state commitment for the non-presumptive challenge leaf.
		proof = util.GeneratePrefixProof(lo, loExp, wrongBlockHashes[lo:hi])
		bisection, err := cl2.Bisect(
			tx,
			util.HistoryCommitment{
				Height: lo,
				Merkle: loExp.Root(),
			},
			proof,
			staker2,
		)
		require.NoError(t, err)

		// Ensure the prev value of cl2 is set to the vertex we just bisected to.
		require.Equal(t, bisection, cl2.Prev.Unwrap())

		// The rootAssertion of the bisectoin should be the rootVertex of this challenge and the bisection
		// should be the new presumptive successor.
		require.Equal(t, challenge.rootVertex.Unwrap().Commitment.Merkle, bisection.Prev.Unwrap().Commitment.Merkle)
		require.Equal(t, true, bisection.Prev.Unwrap().IsPresumptiveSuccessor())
		return nil
	})

	require.NoError(t, err)
}

func TestAssertionChain_Merge(t *testing.T) {
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	t.Run("past deadline", func(t *testing.T) {
		timeRef := util.NewArtificialTimeReference()
		counter := util.NewCountUpTimer(timeRef)
		counter.Add(2 * time.Minute)
		rootAssertion := util.Some(&Assertion{
			chain: &AssertionChain{
				challengePeriod: time.Minute,
			},
		})
		ps := util.Some(&ChallengeVertex{
			PsTimer: counter,
			Commitment: util.HistoryCommitment{
				Height: 1,
			},
		})
		mergingTo := &ChallengeVertex{
			Challenge: util.Some(&Challenge{
				rootAssertion: rootAssertion,
			}),
			PresumptiveSuccessor: ps,
		}
		mergingFrom := &ChallengeVertex{}
		err := mergingFrom.Merge(
			tx,
			mergingTo,
			[]common.Hash{},
			common.Address{},
		)
		require.ErrorIs(t, err, ErrPastDeadline)
	})
	t.Run("invalid bisection point", func(t *testing.T) {
		mergingTo := &ChallengeVertex{}
		mergingFrom := &ChallengeVertex{
			Prev: util.Some(&ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 3,
				},
			}),
			Commitment: util.HistoryCommitment{
				Height: 4,
			},
		}
		err := mergingFrom.Merge(
			tx,
			mergingTo,
			[]common.Hash{},
			common.Address{},
		)
		require.ErrorIs(t, err, util.ErrUnableToBisect)
	})
	t.Run("invalid height", func(t *testing.T) {
		mergingTo := &ChallengeVertex{
			Commitment: util.HistoryCommitment{
				Height: 2,
			},
		}
		mergingFrom := &ChallengeVertex{
			Prev: util.Some[*ChallengeVertex](&ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 2,
				},
			}),
			Commitment: util.HistoryCommitment{
				Height: 4,
			},
		}
		err := mergingFrom.Merge(
			tx,
			mergingTo,
			[]common.Hash{},
			common.Address{},
		)
		require.ErrorIs(t, err, ErrInvalidHeight)
	})
	t.Run("invalid prefix proof", func(t *testing.T) {
		mergingTo := &ChallengeVertex{
			Commitment: util.HistoryCommitment{
				Height: 3,
			},
		}
		mergingFrom := &ChallengeVertex{
			Prev: util.Some[*ChallengeVertex](&ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 2,
				},
			}),
			Commitment: util.HistoryCommitment{
				Height: 4,
			},
		}
		err := mergingFrom.Merge(
			tx,
			mergingTo,
			[]common.Hash{},
			common.Address{},
		)
		require.ErrorIs(t, err, util.ErrIncorrectProof)
	})
	t.Run("OK", func(t *testing.T) {
		ctx := context.Background()
		timeRef := util.NewArtificialTimeReference()
		counter := util.NewCountUpTimer(timeRef)
		stateRoots := correctBlockHashesForTest(10)

		loExp := util.ExpansionFromLeaves(stateRoots[:3])
		proof := util.GeneratePrefixProof(
			3,
			loExp,
			stateRoots[3:4],
		)

		exp := util.ExpansionFromLeaves(stateRoots[:3])
		mergingToCommit := util.HistoryCommitment{
			Height: 3,
			Merkle: exp.Root(),
		}
		mergingTo := &ChallengeVertex{
			PsTimer:    counter,
			Commitment: mergingToCommit,
		}
		exp = util.ExpansionFromLeaves(stateRoots[:4])
		mergingFromCommit := util.HistoryCommitment{
			Height: 4,
			Merkle: exp.Root(),
		}
		mergingFrom := &ChallengeVertex{
			PsTimer: counter,
			Challenge: util.Some[*Challenge](&Challenge{
				rootAssertion: util.Some[*Assertion](&Assertion{
					chain: &AssertionChain{
						challengesFeed: NewEventFeed[ChallengeEvent](ctx),
					},
				}),
			}),
			Prev: util.Some[*ChallengeVertex](&ChallengeVertex{
				Commitment: util.HistoryCommitment{
					Height: 2,
				},
			}),
			Commitment: mergingFromCommit,
		}
		err := mergingFrom.Merge(
			tx,
			mergingTo,
			proof,
			common.Address{},
		)
		require.NoError(t, err)
	})
}

func correctBlockHashesForTest(numBlocks uint64) []common.Hash {
	var ret []common.Hash
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(i))
	}
	return ret
}

func wrongBlockHashesForTest(numBlocks uint64) []common.Hash {
	var ret []common.Hash
	for i := uint64(0); i < numBlocks; i++ {
		ret = append(ret, util.HashForUint(71285937102384-i))
	}
	return ret
}

func TestAssertionChain_StakerInsufficientBalance(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	require.ErrorContains(t, chain.DeductFromBalance(
		&ActiveTx{TxStatus: ReadWriteTxStatus},
		common.BytesToAddress([]byte{1}),
		AssertionStake,
	), "0 < 100000000000000000000: insufficient balance")
}

func TestAssertionChain_ChallengePeriodLength(t *testing.T) {
	ctx := context.Background()
	cp := 123 * time.Second
	tx := &ActiveTx{TxStatus: ReadOnlyTxStatus}
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), cp)
	require.Equal(t, chain.ChallengePeriodLength(tx), cp)
}

func TestAssertionChain_Inbox(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	require.Equal(t, chain.Inbox().messages, NewInbox(ctx).messages)
}

func TestAssertionChain_RetrieveAssertions(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	require.Equal(t, chain.Inbox().messages, NewInbox(ctx).messages)
	staker := common.BytesToAddress([]byte{1})
	bigBalance := new(big.Int).Mul(AssertionStake, big.NewInt(1000))
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	chain.SetBalance(tx, staker, bigBalance)
	p := chain.LatestConfirmed(tx)
	a, err := chain.CreateLeaf(tx, p, StateCommitment{Height: 1}, staker)
	require.NoError(t, err)
	require.Equal(t, chain.NumAssertions(tx), uint64(2))
	got, err := chain.AssertionBySequenceNum(tx, 0)
	require.NoError(t, err)
	require.Equal(t, got, p)
	got, err = chain.AssertionBySequenceNum(tx, 1)
	require.NoError(t, err)
	require.Equal(t, got, a)
}

func TestAssertionChain_LeafCreationErrors(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	badChain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod+1)
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	lc := chain.LatestConfirmed(tx)
	_, err := badChain.CreateLeaf(tx, lc, StateCommitment{}, common.BytesToAddress([]byte{}))
	require.ErrorIs(t, err, ErrWrongChain)
	_, err = chain.CreateLeaf(tx, lc, StateCommitment{}, common.BytesToAddress([]byte{}))
	require.ErrorIs(t, err, ErrInvalidOp)
}

func TestAssertion_ErrWrongState(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	a := chain.LatestConfirmed(tx)
	require.ErrorIs(t, a.RejectForPrev(tx), ErrWrongState)
	require.ErrorIs(t, a.RejectForLoss(tx), ErrWrongState)
	require.ErrorIs(t, a.ConfirmForWin(tx), ErrWrongState)
}

func TestAssertion_ErrWrongPredecessorState(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	staker := common.BytesToAddress([]byte{1})
	bigBalance := new(big.Int).Mul(AssertionStake, big.NewInt(1000))
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	chain.SetBalance(tx, staker, bigBalance)
	newA, err := chain.CreateLeaf(tx, chain.LatestConfirmed(tx), StateCommitment{Height: 1}, staker)
	require.NoError(t, err)
	require.ErrorIs(t, newA.RejectForPrev(tx), ErrWrongPredecessorState)
	require.ErrorIs(t, newA.ConfirmForWin(tx), ErrWrongPredecessorState)
}

func TestAssertion_ErrNotYet(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	staker := common.BytesToAddress([]byte{1})
	bigBalance := new(big.Int).Mul(AssertionStake, big.NewInt(1000))
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	chain.SetBalance(tx, staker, bigBalance)
	newA, err := chain.CreateLeaf(tx, chain.LatestConfirmed(tx), StateCommitment{Height: 1}, staker)
	require.NoError(t, err)
	require.ErrorIs(t, newA.ConfirmNoRival(tx), ErrNotYet)
}

func TestAssertion_ErrInvalid(t *testing.T) {
	ctx := context.Background()
	chain := NewAssertionChain(ctx, util.NewArtificialTimeReference(), testChallengePeriod)
	staker := common.BytesToAddress([]byte{1})
	bigBalance := new(big.Int).Mul(AssertionStake, big.NewInt(1000))
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	chain.SetBalance(tx, staker, bigBalance)
	newA, err := chain.CreateLeaf(tx, chain.LatestConfirmed(tx), StateCommitment{Height: 1}, staker)
	require.NoError(t, err)
	newA.Prev = util.None[*Assertion]()
	require.ErrorIs(t, newA.RejectForPrev(tx), ErrInvalidOp)
	require.ErrorIs(t, newA.RejectForLoss(tx), ErrInvalidOp)
	require.ErrorIs(t, newA.ConfirmNoRival(tx), ErrInvalidOp)
	require.ErrorIs(t, newA.ConfirmForWin(tx), ErrInvalidOp)
}

func TestAssertion_HasConfirmedAboveSeqNumber(t *testing.T) {
	c := &Challenge{}
	tx := &ActiveTx{TxStatus: ReadOnlyTxStatus}
	require.False(t, c.HasConfirmedAboveSeqNumber(tx, 0))
	a := util.Some(&Assertion{
		chain: &AssertionChain{
			challengeVerticesByCommitHash: make(map[ChallengeCommitHash]map[VertexCommitHash]*ChallengeVertex),
		}})
	c.rootAssertion = a
	require.False(t, c.HasConfirmedAboveSeqNumber(tx, 0))

	h := c.ParentStateCommitment().Hash()
	c.rootAssertion.Unwrap().chain.challengeVerticesByCommitHash[ChallengeCommitHash(h)] = map[VertexCommitHash]*ChallengeVertex{
		VertexCommitHash(h): {SequenceNum: 100, Status: ConfirmedAssertionState},
	}

	require.True(t, c.HasConfirmedAboveSeqNumber(tx, 99))
	require.False(t, c.HasConfirmedAboveSeqNumber(tx, 100))
	require.False(t, c.HasConfirmedAboveSeqNumber(tx, 101))
}
