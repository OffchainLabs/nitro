package endtoend

import (
	"context"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/endtoend/internal/backend"
)

// expect is a function that will be called asynchronously to verify some success criteria
// for the given scenario.
type expect func(t *testing.T, ctx context.Context, be backend.Backend) error

// expectChallengeCompletedByOneStepProof by waiting for a log to be received where any edge emits
// a EdgeConfirmedByOneStepProof event and that edge has a status of finished.
//
//nolint:unused
func expectChallengeCompletedByOneStepProof(t *testing.T, ctx context.Context, be backend.Backend) error {
	t.Run("challenge completed by one step proof", func(t *testing.T) {
		ecm, err := edgeManager(be)
		if err != nil {
			t.Fatal(err)
		}

		var edgeConfirmed bool
		for ctx.Err() == nil && !edgeConfirmed {
			i, err := ecm.FilterEdgeConfirmedByOneStepProof(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			for i.Next() {
				edgeConfirmed = true

				if e, err := ecm.GetEdge(nil, i.Event.EdgeId); err != nil {
					t.Fatal(err)
				} else if e.Status != uint8(protocol.EdgeConfirmed) {
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
//
//nolint:unused
func expectAliceAndBobStaked(t *testing.T, ctx context.Context, be backend.Backend) error {
	t.Run("alice and bob staked", func(t *testing.T) {
		ecm, err := edgeManager(be)
		if err != nil {
			t.Fatal(err)
		}

		var aliceStaked, bobStaked bool
		for ctx.Err() == nil && (!aliceStaked || !bobStaked) {
			i, err := ecm.FilterEdgeAdded(nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			for i.Next() {
				edge, err := ecm.GetEdge(nil, i.Event.EdgeId)
				if err != nil {
					t.Fatal(err)
				}

				switch edge.Staker {
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
