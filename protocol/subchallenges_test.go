package protocol

import (
	"context"
	"testing"
	"time"

	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestChallengeVertex_CreateBigStepChallenge(t *testing.T) {
	ctx := context.Background()
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	t.Run("top level challenge must be block challenge", func(t *testing.T) {
		v := setupValidSubChallengeCreation(t, SmallStepChallenge)
		err := v.CreateBigStepChallenge(ctx, tx)
		require.ErrorIs(t, err, ErrWrongChallengeKind)
	})
	t.Run("OK", func(t *testing.T) {
		v := setupValidSubChallengeCreation(t, BlockChallenge)
		err := v.CreateBigStepChallenge(ctx, tx)
		require.NoError(t, err)
		sub := v.SubChallenge.Unwrap()
		subChallengeType, _ := sub.GetChallengeType(ctx, tx)
		require.Equal(t, ChallengeType(BigStepChallenge), subChallengeType)
	})
}

func TestChallengeVertex_CreateSmallStepChallenge(t *testing.T) {
	ctx := context.Background()
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	t.Run("top level challenge must be big step challenge", func(t *testing.T) {
		v := setupValidSubChallengeCreation(t, SmallStepChallenge)
		err := v.CreateSmallStepChallenge(ctx, tx)
		require.ErrorIs(t, err, ErrWrongChallengeKind)
	})
	t.Run("OK", func(t *testing.T) {
		v := setupValidSubChallengeCreation(t, BigStepChallenge)
		err := v.CreateSmallStepChallenge(ctx, tx)
		require.NoError(t, err)
		sub := v.SubChallenge.Unwrap()
		subChallengeType, _ := sub.GetChallengeType(ctx, tx)
		require.Equal(t, ChallengeType(SmallStepChallenge), subChallengeType)
	})
}

func setupValidSubChallengeCreation(t *testing.T, topLevelType ChallengeType) *ChallengeVertex {
	challengePeriod := 5 * time.Second
	ref := util.NewRealTimeReference()
	m := make(map[ChallengeCommitHash]map[VertexCommitHash]ChallengeVertexInterface)
	chain := &AssertionChain{
		challengePeriod:               challengePeriod,
		timeReference:                 ref,
		challengeVerticesByCommitHash: m,
	}

	creationTime := ref.Get()
	chal := &Challenge{
		creationTime:  creationTime,
		challengeType: topLevelType,
		rootAssertion: util.Some(&Assertion{
			challengeManager: chain,
			StateCommitment:  util.StateCommitment{},
		}),
	}
	v := &ChallengeVertex{
		Challenge:    util.Some(ChallengeInterface(chal)),
		SubChallenge: util.None[ChallengeInterface](),
		Status:       PendingAssertionState,
		Commitment: util.HistoryCommitment{
			Height: 0,
			Merkle: common.BytesToHash([]byte("foo")),
		},
	}

	challengeHash := ChallengeCommitHash((util.StateCommitment{}).Hash())
	vertices := make(map[VertexCommitHash]ChallengeVertexInterface, 0)

	// Create child vertices with unexpired chess clocks.
	for i := uint(0); i < 3; i++ {
		timer := util.NewCountUpTimer(ref)
		child := &ChallengeVertex{
			Prev: util.Some(ChallengeVertexInterface(v)),
			Commitment: util.HistoryCommitment{
				Height: v.Commitment.Height + 1,
				Merkle: common.BytesToHash([]byte(fmt.Sprintf("%d", i))),
			},
			PsTimer: timer,
		}
		vHash := VertexCommitHash(child.Commitment.Hash())
		if i == 0 {
			v.Prev = util.None[ChallengeVertexInterface]()
		}
		vertices[vHash] = child
	}
	chain.challengeVerticesByCommitHash[challengeHash] = vertices
	return v
}

func Test_canCreateSubChallenge(t *testing.T) {
	ctx := context.Background()
	tx := &ActiveTx{TxStatus: ReadOnlyTxStatus}
	t.Run("no challenge for vertex", func(t *testing.T) {
		v := &ChallengeVertex{
			Challenge: util.None[ChallengeInterface](),
		}
		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.ErrorIs(t, err, ErrNoChallenge)
	})
	t.Run("block challenge cannot be a subchallenge", func(t *testing.T) {
		v := &ChallengeVertex{
			Challenge: util.Some(ChallengeInterface(&Challenge{})),
		}
		err := v.canCreateSubChallenge(ctx, tx, BlockChallenge)
		require.ErrorIs(t, err, ErrWrongChallengeKind)
	})
	t.Run("parent of big step challenge must be block challenge", func(t *testing.T) {
		v := &ChallengeVertex{
			Challenge: util.Some(ChallengeInterface(&Challenge{
				challengeType: SmallStepChallenge,
			})),
		}
		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.ErrorIs(t, err, ErrWrongChallengeKind)
	})
	t.Run("parent of small step challenge must be big step challenge", func(t *testing.T) {
		v := &ChallengeVertex{
			Challenge: util.Some(ChallengeInterface(&Challenge{
				challengeType: SmallStepChallenge,
			})),
		}
		err := v.canCreateSubChallenge(ctx, tx, SmallStepChallenge)
		require.ErrorIs(t, err, ErrWrongChallengeKind)
	})
	t.Run("challenge must be ongoing", func(t *testing.T) {
		// Create an expired challenge.
		challengePeriod := 5 * time.Second
		ref := util.NewRealTimeReference()
		chain := &AssertionChain{
			challengePeriod: challengePeriod,
			timeReference:   ref,
		}
		creationTime := ref.Get().Add(-2 * challengePeriod)
		chal := &Challenge{
			creationTime:  creationTime,
			challengeType: BlockChallenge,
			rootAssertion: util.Some(&Assertion{
				challengeManager: chain,
			}),
		}
		v := &ChallengeVertex{
			Challenge: util.Some(ChallengeInterface(chal)),
		}
		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.ErrorIs(t, err, ErrChallengeNotRunning)
	})
	t.Run("subchallenge already exists", func(t *testing.T) {
		challengePeriod := 5 * time.Second
		ref := util.NewRealTimeReference()
		chain := &AssertionChain{
			challengePeriod: challengePeriod,
			timeReference:   ref,
		}
		creationTime := ref.Get()
		chal := &Challenge{
			creationTime:  creationTime,
			challengeType: BlockChallenge,
			rootAssertion: util.Some(&Assertion{
				challengeManager: chain,
			}),
		}
		v := &ChallengeVertex{
			Challenge:    util.Some(ChallengeInterface(chal)),
			SubChallenge: util.Some(ChallengeInterface(&Challenge{})),
		}
		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.ErrorIs(t, err, ErrSubchallengeAlreadyExists)
	})
	t.Run("vertex must not be confirmed", func(t *testing.T) {
		challengePeriod := 5 * time.Second
		ref := util.NewRealTimeReference()
		chain := &AssertionChain{
			challengePeriod: challengePeriod,
			timeReference:   ref,
		}
		creationTime := ref.Get()
		chal := &Challenge{
			creationTime:  creationTime,
			challengeType: BlockChallenge,
			rootAssertion: util.Some(&Assertion{
				challengeManager: chain,
			}),
		}
		v := &ChallengeVertex{
			Challenge:    util.Some(ChallengeInterface(chal)),
			SubChallenge: util.None[ChallengeInterface](),
			Status:       ConfirmedAssertionState,
		}
		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.ErrorIs(t, err, ErrWrongState)
	})
	t.Run("checking unexpired children's existence fails", func(t *testing.T) {
		challengePeriod := 5 * time.Second
		ref := util.NewRealTimeReference()
		m := make(map[ChallengeCommitHash]map[VertexCommitHash]ChallengeVertexInterface)
		chain := &AssertionChain{
			challengePeriod:               challengePeriod,
			timeReference:                 ref,
			challengeVerticesByCommitHash: m,
		}
		creationTime := ref.Get()
		chal := &Challenge{
			creationTime:  creationTime,
			challengeType: BlockChallenge,
			rootAssertion: util.Some(&Assertion{
				challengeManager: chain,
				StateCommitment:  util.StateCommitment{},
			}),
		}
		v := &ChallengeVertex{
			Challenge:    util.Some(ChallengeInterface(chal)),
			SubChallenge: util.None[ChallengeInterface](),
			Status:       PendingAssertionState,
		}
		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.ErrorContains(t, err, "vertices not found")
	})
	t.Run("not enough unexpired children", func(t *testing.T) {
		challengePeriod := 5 * time.Second
		ref := util.NewRealTimeReference()
		m := make(map[ChallengeCommitHash]map[VertexCommitHash]ChallengeVertexInterface)
		chain := &AssertionChain{
			challengePeriod:               challengePeriod,
			timeReference:                 ref,
			challengeVerticesByCommitHash: m,
		}

		challengeHash := ChallengeCommitHash((util.StateCommitment{}).Hash())
		vertices := make(map[VertexCommitHash]ChallengeVertexInterface, 0)
		chain.challengeVerticesByCommitHash[challengeHash] = vertices

		creationTime := ref.Get()
		chal := &Challenge{
			creationTime:  creationTime,
			challengeType: BlockChallenge,
			rootAssertion: util.Some(&Assertion{
				challengeManager: chain,
				StateCommitment:  util.StateCommitment{},
			}),
		}
		v := &ChallengeVertex{
			Challenge:    util.Some(ChallengeInterface(chal)),
			SubChallenge: util.None[ChallengeInterface](),
			Status:       PendingAssertionState,
		}
		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.ErrorIs(t, err, ErrNotEnoughValidChildren)
	})
	t.Run("OK", func(t *testing.T) {
		challengePeriod := 5 * time.Second
		ref := util.NewRealTimeReference()
		m := make(map[ChallengeCommitHash]map[VertexCommitHash]ChallengeVertexInterface)
		chain := &AssertionChain{
			challengePeriod:               challengePeriod,
			timeReference:                 ref,
			challengeVerticesByCommitHash: m,
		}

		creationTime := ref.Get()
		chal := &Challenge{
			creationTime:  creationTime,
			challengeType: BlockChallenge,
			rootAssertion: util.Some(&Assertion{
				challengeManager: chain,
				StateCommitment:  util.StateCommitment{},
			}),
		}
		v := &ChallengeVertex{
			Challenge:    util.Some(ChallengeInterface(chal)),
			SubChallenge: util.None[ChallengeInterface](),
			Status:       PendingAssertionState,
		}

		challengeHash := ChallengeCommitHash((util.StateCommitment{}).Hash())
		vertices := make(map[VertexCommitHash]ChallengeVertexInterface, 0)

		// Create child vertices with unexpired chess clocks.
		for i := uint(0); i < 3; i++ {
			timer := util.NewCountUpTimer(ref)
			child := &ChallengeVertex{
				Prev: util.Some(ChallengeVertexInterface(v)),
				Commitment: util.HistoryCommitment{
					Height: v.Commitment.Height + 1,
					Merkle: common.BytesToHash([]byte(fmt.Sprintf("%d", i))),
				},
				PsTimer: timer,
			}
			vHash := VertexCommitHash(child.Commitment.Hash())
			if i == 0 {
				child.Prev = util.None[ChallengeVertexInterface]()
			}
			vertices[vHash] = child
		}
		chain.challengeVerticesByCommitHash[challengeHash] = vertices

		err := v.canCreateSubChallenge(ctx, tx, BigStepChallenge)
		require.NoError(t, err)
	})
}

func TestChallengeVertex_hasUnexpiredChildren(t *testing.T) {
	ctx := context.Background()
	tx := &ActiveTx{TxStatus: ReadOnlyTxStatus}
	t.Run("no challenge for vertex", func(t *testing.T) {
		chain := &AssertionChain{}
		v := &ChallengeVertex{
			Challenge: util.None[ChallengeInterface](),
		}
		_, err := hasUnexpiredChildren(ctx, tx, chain, v)
		require.ErrorIs(t, err, ErrNoChallenge)
	})
	t.Run("vertices not found for challenge", func(t *testing.T) {
		m := make(map[ChallengeCommitHash]map[VertexCommitHash]ChallengeVertexInterface)
		chain := &AssertionChain{
			challengeVerticesByCommitHash: m,
		}
		v := &ChallengeVertex{
			Challenge: util.Some(ChallengeInterface(&Challenge{
				rootAssertion: util.None[*Assertion](),
			})),
		}
		_, err := hasUnexpiredChildren(ctx, tx, chain, v)
		require.ErrorContains(t, err, "vertices not found")
	})

	for _, testCase := range []struct {
		name         string
		numChildren  uint
		numUnexpired uint
		want         bool
	}{
		{
			name: "no children",
			want: false,
		},
		{
			name:        "two children, but both expired",
			numChildren: 2,
			want:        false,
		},
		{
			name:         "one child, unexpired",
			numChildren:  1,
			numUnexpired: 1,
			want:         false,
		},
		{
			name:         "two children, one expired one unexpired",
			numChildren:  2,
			numUnexpired: 1,
			want:         false,
		},
		{
			name:         "two children, both unexpired",
			numChildren:  2,
			numUnexpired: 2,
			want:         true,
		},
		{
			name:         "ten children, all but one unexpired",
			numChildren:  10,
			numUnexpired: 9,
			want:         true,
		},
		{
			name:         "ten children, all unexpired",
			numChildren:  10,
			numUnexpired: 10,
			want:         true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			m := make(map[ChallengeCommitHash]map[VertexCommitHash]ChallengeVertexInterface)
			timeRef := util.NewArtificialTimeReference()
			chain := &AssertionChain{
				challengePeriod:               5 * time.Second,
				challengeVerticesByCommitHash: m,
				timeReference:                 timeRef,
			}
			parent := &ChallengeVertex{
				Challenge: util.Some(ChallengeInterface(&Challenge{
					rootAssertion: util.None[*Assertion](),
				})),
			}
			challengeHash := ChallengeCommitHash((util.StateCommitment{}).Hash())

			vertices := make(map[VertexCommitHash]ChallengeVertexInterface, testCase.numChildren)
			for i := uint(0); i < testCase.numChildren; i++ {

				// Children are expired by default for these tests.
				timer := util.NewCountUpTimer(timeRef)
				timer.Add(2 * chain.challengePeriod)

				v := &ChallengeVertex{
					Prev: util.Some(ChallengeVertexInterface(parent)),
					Commitment: util.HistoryCommitment{
						Height: parent.Commitment.Height + 1,
						Merkle: common.BytesToHash([]byte(fmt.Sprintf("%d", i))),
					},
					PsTimer: timer,
				}
				vHash := VertexCommitHash(v.Commitment.Hash())
				vertices[vHash] = v

				// If we want to mark an vertex as unexpired, we give it
				// a different ps timer.
				if i < testCase.numUnexpired {
					v.PsTimer = util.NewCountUpTimer(timeRef)
				}
			}
			chain.challengeVerticesByCommitHash[challengeHash] = vertices

			got, err := hasUnexpiredChildren(ctx, tx, chain, parent)
			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestChallenge_hasEnded(t *testing.T) {
	ctx := context.Background()
	tx := &ActiveTx{TxStatus: ReadOnlyTxStatus}
	challengePeriod := 5 * time.Second
	for _, tt := range []struct {
		elapsed time.Duration
		want    bool
	}{
		{elapsed: time.Second, want: false},
		{elapsed: challengePeriod, want: false},
		{elapsed: 2 * challengePeriod, want: true},
	} {
		ref := util.NewRealTimeReference()
		creationTime := ref.Get().Add(-tt.elapsed)
		chal := &Challenge{
			creationTime: creationTime,
		}
		chain := &AssertionChain{
			challengePeriod: challengePeriod,
			timeReference:   ref,
		}
		got, _ := chal.HasEnded(ctx, tx, chain)
		require.Equal(t, tt.want, got)
	}

}

func TestChallengeVertex_chessClockExpired(t *testing.T) {
	challengePeriod := 5 * time.Second
	for _, tt := range []struct {
		elapsed time.Duration
		want    bool
	}{
		{elapsed: time.Second, want: false},
		{elapsed: challengePeriod, want: false},
		{elapsed: challengePeriod + time.Millisecond, want: true},
		{elapsed: 2 * challengePeriod, want: true},
	} {
		ref := util.NewArtificialTimeReference()
		timer := util.NewCountUpTimer(ref)
		timer.Add(tt.elapsed)
		v := &ChallengeVertex{
			PsTimer: timer,
		}
		got, err := v.ChessClockExpired(context.Background(), &ActiveTx{}, challengePeriod)
		require.NoError(t, err)
		require.Equal(t, tt.want, got)
	}
}
