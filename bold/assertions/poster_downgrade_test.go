// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package assertions

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	"github.com/offchainlabs/nitro/bold/protocol"
)

// TestRecordAgreedAssertionDoesNotDowngradeLatestAgreedAssertion verifies the
// parent-link guard: a slow catchup write for an ancestor of the current
// latestAgreedAssertion must not downgrade the cursor. Without the guard,
// the next sync chunk would skip assertions whose parent is the previous
// (newer) cursor value and self-challenge them.
func TestRecordAgreedAssertionDoesNotDowngradeLatestAgreedAssertion(t *testing.T) {
	a0 := protocol.AssertionHash{Hash: hashFromString("A0")}
	a1 := protocol.AssertionHash{Hash: hashFromString("A1")}
	a2 := protocol.AssertionHash{Hash: hashFromString("A2")}

	a0Info := &protocol.AssertionCreatedInfo{AssertionHash: a0, InboxMaxCount: big.NewInt(1)}
	a1Info := &protocol.AssertionCreatedInfo{AssertionHash: a1, ParentAssertionHash: a0, InboxMaxCount: big.NewInt(2)}
	a2Info := &protocol.AssertionCreatedInfo{AssertionHash: a2, ParentAssertionHash: a1, InboxMaxCount: big.NewInt(3)}

	m := managerWithCanonical(t, a2, map[protocol.AssertionHash]*protocol.AssertionCreatedInfo{
		a0: a0Info,
		a1: a1Info,
		a2: a2Info,
	})

	// Slow catchup belatedly applies A1 (an ancestor of latestAgreedAssertion).
	// A1's parent is A0, not A2, so the cursor must NOT move.
	m.applyRecordAgreedAssertion(a1Info)

	require.Equal(t,
		a2,
		m.assertionChainData.latestAgreedAssertion,
		"latestAgreedAssertion was downgraded from A2 back to an ancestor — "+
			"this is the bug that triggers honest-validator self-challenge")

	// canonicalAssertions/submittedAssertions must still be append-only.
	_, hasA1 := m.assertionChainData.canonicalAssertions[a1]
	require.True(t, hasA1, "A1 must still be present in canonicalAssertions")
	require.True(t, m.submittedAssertions.Has(a1), "A1 must be tracked in submittedAssertions")
}

// TestRecordAgreedAssertionAdvancesOnDirectChild verifies the happy path: a
// direct child of the current cursor advances it forward. Without this the
// catchup loop can't make progress.
func TestRecordAgreedAssertionAdvancesOnDirectChild(t *testing.T) {
	a0 := protocol.AssertionHash{Hash: hashFromString("A0")}
	a1 := protocol.AssertionHash{Hash: hashFromString("A1")}

	a0Info := &protocol.AssertionCreatedInfo{AssertionHash: a0, InboxMaxCount: big.NewInt(1)}
	a1Info := &protocol.AssertionCreatedInfo{AssertionHash: a1, ParentAssertionHash: a0, InboxMaxCount: big.NewInt(2)}

	m := managerWithCanonical(t, a0, map[protocol.AssertionHash]*protocol.AssertionCreatedInfo{a0: a0Info})

	m.applyRecordAgreedAssertion(a1Info)

	require.Equal(t, a1, m.assertionChainData.latestAgreedAssertion,
		"latestAgreedAssertion should advance to a direct child of the current value")
	_, hasA1 := m.assertionChainData.canonicalAssertions[a1]
	require.True(t, hasA1)
}

// TestRecordAgreedAssertionAllowsOverflowAdvance pins parent-hash linkage
// (not numeric InboxMaxCount ordering) as the advance check. Overflow
// assertions share InboxMaxCount with their parent, so a numeric check
// would pin catchup at the overflow parent forever.
func TestRecordAgreedAssertionAllowsOverflowAdvance(t *testing.T) {
	parent := protocol.AssertionHash{Hash: hashFromString("parent")}
	child := protocol.AssertionHash{Hash: hashFromString("child-overflow")}

	const sharedInboxMaxCount = 42
	parentInfo := &protocol.AssertionCreatedInfo{
		AssertionHash: parent,
		InboxMaxCount: big.NewInt(sharedInboxMaxCount),
	}
	childInfo := &protocol.AssertionCreatedInfo{
		AssertionHash:       child,
		ParentAssertionHash: parent,
		// Same as parent — the overflow case. A numeric check would refuse.
		InboxMaxCount: big.NewInt(sharedInboxMaxCount),
	}

	m := managerWithCanonical(t, parent, map[protocol.AssertionHash]*protocol.AssertionCreatedInfo{parent: parentInfo})

	m.applyRecordAgreedAssertion(childInfo)

	require.Equal(t, child, m.assertionChainData.latestAgreedAssertion,
		"overflow child (same InboxMaxCount as parent) must be allowed to advance "+
			"the chain pointer — a numeric-ordering check would have left catchup stuck")
}

// TestNoSelfChallengeAfterCursorDowngradeAttempt walks the full prod scenario
// end-to-end: a slow catchup writes an ancestor, then a sync chunk arrives
// with a new canonical assertion. Asserts cursor stability, downstream
// classification, and absence of spurious rivals. For finer-grained coverage
// of each fix in isolation see TestRecordAgreedAssertion* and
// TestSelfChallengeBugWhenStateProviderReturnsTransientWrongRoot.
func TestNoSelfChallengeAfterCursorDowngradeAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := numToAssertionHash(0)
	a14 := numToAssertionHash(14)
	a15 := numToAssertionHash(15)

	gInfo := &protocol.AssertionCreatedInfo{
		AssertionHash: g,
		InboxMaxCount: big.NewInt(14),
		AfterState:    numToState(0, t),
	}
	a14Info := &protocol.AssertionCreatedInfo{
		AssertionHash:       a14,
		ParentAssertionHash: g,
		InboxMaxCount:       big.NewInt(15),
		AfterState:          numToState(14, t),
	}
	a15Info := &protocol.AssertionCreatedInfo{
		AssertionHash:       a15,
		ParentAssertionHash: a14,
		InboxMaxCount:       big.NewInt(16),
		AfterState:          numToState(15, t),
	}

	// mockStateProvider keys on parent.InboxMaxCount; A14's value is 15.
	provider := &mockStateProvider{
		agreesWith: map[uint64]*protocol.AssertionCreatedInfo{
			15: a15Info,
		},
	}

	m := &Manager{
		execProvider:                provider,
		observedCanonicalAssertions: make(chan protocol.AssertionHash, 16),
		submittedAssertions: threadsafe.NewLruSet(
			1024,
			threadsafe.LruSetWithMetric[protocol.AssertionHash]("submittedAssertions"),
		),
		confirming: threadsafe.NewLruSet[protocol.AssertionHash](1024),
		assertionChainData: &assertionChainData{
			latestAgreedAssertion: a14,
			canonicalAssertions: map[protocol.AssertionHash]*protocol.AssertionCreatedInfo{
				g:   gInfo,
				a14: a14Info,
			},
		},
	}
	// Drain the confirmation queue so sendToConfirmationQueue doesn't block.
	go func() {
		for {
			select {
			case <-m.observedCanonicalAssertions:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Slow catchup tries to advance to G (ancestor of A14).
	m.applyRecordAgreedAssertion(gInfo)
	require.Equal(t, a14, m.assertionChainData.latestAgreedAssertion,
		"cursor downgraded to ancestor — parent-link guard regressed")

	// Sync chunk carrying A15. findCanonicalAssertionBranch should classify
	// it as canonical because the cursor is still A14.
	chunk := []assertionAndParentCreationInfo{
		{parent: a14Info, assertion: a15Info},
	}
	require.NoError(t, m.findCanonicalAssertionBranch(ctx, chunk))
	_, hasA15 := m.assertionChainData.canonicalAssertions[a15]
	require.True(t, hasA15, "A15 skipped — cursor was downgraded")

	// respondToAnyInvalidAssertions must not fire on A15.
	poster := &recordingMockRivalPoster{}
	require.NoError(t, m.respondToAnyInvalidAssertions(ctx, chunk, poster))
	require.Empty(t, poster.calls,
		"rival path fired against a canonical assertion")
}

// recordingMockRivalPoster captures every maybePostRivalAssertionAndChallenge
// invocation. Returns nil so the caller treats the call as a no-op.
type recordingMockRivalPoster struct {
	calls []protocol.AssertionHash
}

func (r *recordingMockRivalPoster) maybePostRivalAssertionAndChallenge(
	_ context.Context,
	args rivalPosterArgs,
) (*protocol.AssertionCreatedInfo, error) {
	r.calls = append(r.calls, args.invalidAssertion.AssertionHash)
	return nil, nil
}

// managerWithCanonical builds a Manager with only the fields recordAgreedAssertion
// touches, pre-seeded with the supplied canonical map and latestAgreedAssertion.
func managerWithCanonical(
	t *testing.T,
	latestAgreed protocol.AssertionHash,
	canonical map[protocol.AssertionHash]*protocol.AssertionCreatedInfo,
) *Manager {
	t.Helper()
	return &Manager{
		assertionChainData: &assertionChainData{
			latestAgreedAssertion: latestAgreed,
			canonicalAssertions:   canonical,
		},
		submittedAssertions: threadsafe.NewLruSet(
			1024,
			threadsafe.LruSetWithMetric[protocol.AssertionHash]("submittedAssertions"),
		),
	}
}

func hashFromString(s string) [32]byte {
	var h [32]byte
	copy(h[:], s)
	return h
}
