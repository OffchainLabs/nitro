package endtoend

import (
	"context"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	retry "github.com/OffchainLabs/challenge-protocol-v2/runtime"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/endtoend/internal/backend"
)

// expect is a function that will be called asynchronously to verify some success criteria
// for the given scenario.
type expect func(t *testing.T, ctx context.Context, be backend.Backend) error

// Expects that a level zero, block challenge edge was successfully confirmed in an e2e run.
func expectLevelZeroBlockEdgeConfirmed(t *testing.T, ctx context.Context, be backend.Backend) error {
	t.Run("level zero block edge confirmed", func(t *testing.T) {
		ecm, err := edgeManager(be)
		if err != nil {
			t.Fatal(err)
		}

		var edgeConfirmed bool
		for ctx.Err() == nil && !edgeConfirmed {
			i, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerEdgeConfirmedByChildrenIterator, error) {
				return ecm.FilterEdgeConfirmedByChildren(nil, nil, nil)
			})
			if err != nil {
				t.Fatal(err)
			}
			for i.Next() {
				e, err := retry.UntilSucceeds(ctx, func() (challengeV2gen.ChallengeEdge, error) {
					return ecm.GetEdge(nil, i.Event.EdgeId)
				})
				if err != nil {
					t.Fatal(err)
				}
				if e.Status != uint8(protocol.EdgeConfirmed) {
					t.Fatal("Confirmed edge with unfinished state")
				}
				isLevelZero := e.StartHeight.Uint64() == 0 && e.EndHeight.Uint64() == protocol.LevelZeroBlockEdgeHeight
				isBlockEdge := e.EType == uint8(protocol.BlockChallengeEdge)
				if isLevelZero && isBlockEdge {
					edgeConfirmed = true
					break
				}
			}
			time.Sleep(500 * time.Millisecond) // Don't spam the backend.
		}

		if !edgeConfirmed {
			t.Fatal("level zero, block challenge edge was not confirmed")
		}
	})

	return nil
}

// expectOneStepProofSuccessful by waiting for a EdgeConfirmedByOneStepProof event and that
// edge has a status of finished.
func expectOneStepProofSuccessful(t *testing.T, ctx context.Context, be backend.Backend) error {
	t.Run("challenge completed by one step proof", func(t *testing.T) {
		ecm, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManager, error) {
			return edgeManager(be)
		})
		if err != nil {
			t.Fatal(err)
		}

		var edgeConfirmed bool
		for ctx.Err() == nil && !edgeConfirmed {
			i, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerEdgeConfirmedByOneStepProofIterator, error) {
				return ecm.FilterEdgeConfirmedByOneStepProof(nil, nil, nil)
			})
			if err != nil {
				t.Fatal(err)
			}
			for i.Next() {
				edgeConfirmed = true

				e, err := retry.UntilSucceeds(ctx, func() (challengeV2gen.ChallengeEdge, error) {
					return ecm.GetEdge(nil, i.Event.EdgeId)
				})
				if err != nil {
					t.Fatal(err)
				}
				if e.Status != uint8(protocol.EdgeConfirmed) {
					t.Fatal("Confirmed edge with unfinished state")
				}
				break
			}
			time.Sleep(500 * time.Millisecond) // Don't spam the backend.
		}

		if !edgeConfirmed {
			t.Fatal("edge not confirmed by one step proof")
		}
	})

	return nil
}

// expectAliceAndBobStaked monitors EdgeAdded events until Alice and Bob are observed adding edges
// with a stake.
func expectAliceAndBobStaked(t *testing.T, ctx context.Context, be backend.Backend) error {
	t.Run("alice and bob staked", func(t *testing.T) {
		ecm, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManager, error) {
			return edgeManager(be)
		})
		if err != nil {
			t.Fatal(err)
		}

		var aliceStaked, bobStaked bool
		for ctx.Err() == nil && (!aliceStaked || !bobStaked) {
			i, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerEdgeAddedIterator, error) {
				return ecm.FilterEdgeAdded(nil, nil, nil, nil)
			})
			if err != nil {
				t.Fatal(err)
			}
			for i.Next() {
				e, err := retry.UntilSucceeds(ctx, func() (challengeV2gen.ChallengeEdge, error) {
					return ecm.GetEdge(nil, i.Event.EdgeId)
				})
				if err != nil {
					t.Fatal(err)
				}

				switch e.Staker {
				case be.Alice().From:
					aliceStaked = true
				case be.Bob().From:
					bobStaked = true
				}

				time.Sleep(500 * time.Millisecond) // Don't spam the backend.
			}
		}

		if !aliceStaked {
			t.Error("alice did not stake")
		}
		if !bobStaked {
			t.Error("bob did not stake")
		}
	})

	return nil
}
