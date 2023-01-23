package protocol

import (
	"testing"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/stretchr/testify/require"
)

func Test_canCreateSubChallenge(t *testing.T) {

}

func TestChallengeVertex_hasUnexpiredChildren(t *testing.T) {
	t.Run("no challenge for vertex", func(t *testing.T) {
		chain := &AssertionChain{}
		v := &ChallengeVertex{
			Challenge: util.None[*Challenge](),
		}
		_, err := hasUnexpiredChildren(chain, v)
		require.ErrorIs(t, err, ErrNoChallenge)
	})
	t.Run("vertices not found for challenge", func(t *testing.T) {
		m := make(map[ChallengeCommitHash]map[VertexCommitHash]*ChallengeVertex)
		chain := &AssertionChain{
			challengeVerticesByCommitHash: m,
		}
		v := &ChallengeVertex{
			Challenge: util.Some(&Challenge{
				rootAssertion: util.None[*Assertion](),
			}),
		}
		_, err := hasUnexpiredChildren(chain, v)
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
			m := make(map[ChallengeCommitHash]map[VertexCommitHash]*ChallengeVertex)
			timeRef := util.NewArtificialTimeReference()
			chain := &AssertionChain{
				challengePeriod:               5 * time.Second,
				challengeVerticesByCommitHash: m,
				timeReference:                 timeRef,
			}
			parent := &ChallengeVertex{
				Challenge: util.Some(&Challenge{
					rootAssertion: util.None[*Assertion](),
				}),
			}
			challengeHash := ChallengeCommitHash((StateCommitment{}).Hash())

			vertices := make(map[VertexCommitHash]*ChallengeVertex, testCase.numChildren)
			for i := uint(0); i < testCase.numChildren; i++ {

				// Children are expired by default for these tests.
				timer := util.NewCountUpTimer(timeRef)
				timer.Add(2 * chain.challengePeriod)

				v := &ChallengeVertex{
					Prev: util.Some(parent),
					Commitment: util.HistoryCommitment{
						Height: uint64(i),
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

			got, err := hasUnexpiredChildren(chain, parent)
			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestChallenge_hasEnded(t *testing.T) {
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
		got := chal.hasEnded(chain)
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
		got := v.chessClockExpired(challengePeriod)
		require.Equal(t, tt.want, got)
	}
}
