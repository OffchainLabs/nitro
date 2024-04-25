package challengetree

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	bisection "github.com/OffchainLabs/bold/math"
	"github.com/pkg/errors"
)

var (
	ErrNoHonestTopLevelEdge = errors.New("no honest block challenge edge being tracked")
	ErrNotFound             = errors.New("not found in honest challenge tree")
	ErrNoLevelZero          = errors.New("no level zero edge with origin id found")
	ErrNoLowerChildYet      = errors.New("edge does not yet have a lower child")
)

// HonestAncestors of an edge id all the way up to and including the
// block challenge level zero edge.
type HonestAncestors []protocol.ReadOnlyEdge

// EdgeLocalTimer is the local, unrivaled timer of a specific edge.
type EdgeLocalTimer uint64

// ComputeAncestors gathers all royal ancestors of a given edge across challenge levels.
// Ancestor lists are linked through challenge levels via claimed edges. It is generalized
// to any number of challenge levels in the protocol.
func (ht *RoyalChallengeTree) ComputeAncestors(
	ctx context.Context,
	edgeId protocol.EdgeId,
	blockNumber uint64,
) (HonestAncestors, error) {
	// Checks if the edge exists before performing any computation.
	startEdge, ok := ht.edges.TryGet(edgeId)
	if !ok {
		return nil, errNotFound(edgeId)
	}
	currentChallengeLevel := startEdge.GetReversedChallengeLevel()

	// Set a cursor at the edge we start from. We will update this cursor
	// as we advance in this function.
	currentEdge := startEdge

	ancestry := make([]protocol.ReadOnlyEdge, 0)

	// Challenge levels go from lowest to highest, where lowest is the smallest challenge level
	// (where challenges are over individual, WASM opcodes). If we have 3 challenge levels,
	// we will go from 0, 1, 2. We want the ancestors for an edge across an entire challenge
	// tree, even across levels.
	for currentChallengeLevel < protocol.ChallengeLevel(ht.totalChallengeLevels) {
		// Compute the root edge for the current challenge level.
		rootEdge, err := ht.honestRootAncestorAtChallengeLevel(currentEdge, currentChallengeLevel)
		if err != nil {
			return nil, err
		}

		// Compute the ancestors for the current edge in the current challenge level.
		ancestorLocalTimers, ancestorsAtLevel, err := ht.findHonestAncestorsWithinChallengeLevel(
			ctx, rootEdge, currentEdge, blockNumber,
		)
		if err != nil {
			return nil, err
		}

		// Expand the total ancestry and timers slices. We want ancestors from
		// the bottom-up, so we must reverse the output slice from the find function.
		containers.Reverse(ancestorLocalTimers)
		containers.Reverse(ancestorsAtLevel)
		ancestry = append(ancestry, ancestorsAtLevel...)

		// Advance the challenge level.
		currentChallengeLevel += 1

		if currentChallengeLevel == protocol.ChallengeLevel(ht.totalChallengeLevels) {
			break
		}

		// Update the current edge to the one the root edge at this challenge claims
		// at the next challenge level to link between levels.
		nextLevelClaimedEdge, err := ht.getClaimedEdge(rootEdge)
		if err != nil {
			return nil, err
		}
		// Update the cursor to be the claimed edge at the next challenge level.
		currentEdge = nextLevelClaimedEdge

		// Include the next level claimed edge in the ancestry list.
		ancestry = append(ancestry, nextLevelClaimedEdge)
	}
	return ancestry, nil
}

// Computes the list of ancestors in a challenge level from a root edge down
// to a specified child edge within the same level. The edge we are querying must be
// a child of this start edge for this function to succeed without error.
func (ht *RoyalChallengeTree) findHonestAncestorsWithinChallengeLevel(
	ctx context.Context,
	rootEdge protocol.ReadOnlyEdge,
	queryingFor protocol.ReadOnlyEdge,
	blockNumber uint64,
) ([]EdgeLocalTimer, []protocol.ReadOnlyEdge, error) {
	found := false
	cursor := rootEdge
	ancestry := make([]protocol.ReadOnlyEdge, 0)
	localTimers := make([]EdgeLocalTimer, 0)
	wantedEdgeStart, _ := queryingFor.StartCommitment()

	for {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		if cursor.Id() == queryingFor.Id() {
			found = true
			break
		}
		// We expand the ancestry and timers' slices using the cursor edge.
		ancestry = append(ancestry, cursor)
		timer, err := ht.LocalTimer(cursor, blockNumber)
		if err != nil {
			return nil, nil, err
		}
		localTimers = append(localTimers, EdgeLocalTimer(timer))

		currStart, _ := cursor.StartCommitment()
		currEnd, _ := cursor.EndCommitment()
		bisectTo, err := bisection.Bisect(uint64(currStart), uint64(currEnd))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not bisect start=%d, end=%d", currStart, currEnd)
		}
		// If the wanted edge's start commitment is less than the bisection height of the current
		// edge in the loop, it means it is part of its lower children.
		if uint64(wantedEdgeStart) < bisectTo {
			lowerChild, lowerErr := cursor.LowerChild(ctx)
			if lowerErr != nil {
				return nil, nil, errors.Wrapf(lowerErr, "could not get lower child for edge %#x", cursor.Id())
			}
			if lowerChild.IsNone() {
				return nil, nil, errNotFound(queryingFor.Id())
			}
			cursor = ht.edges.Get(lowerChild.Unwrap())
		} else {
			// Else, it is part of the upper children.
			upperChild, upperErr := cursor.UpperChild(ctx)
			if upperErr != nil {
				return nil, nil, errors.Wrapf(upperErr, "could not get upper child for edge %#x", cursor.Id())
			}
			if upperChild.IsNone() {
				return nil, nil, errNotFound(queryingFor.Id())
			}
			cursor = ht.edges.Get(upperChild.Unwrap())
		}
	}
	if !found {
		return nil, nil, errNotFound(queryingFor.Id())
	}
	return localTimers, ancestry, nil
}

func (ht *RoyalChallengeTree) hasHonestAncestry(ctx context.Context, eg protocol.SpecEdge) (bool, error) {
	chalLevel := eg.GetChallengeLevel()
	claimId := eg.ClaimId()

	// If the edge is a root edge at the block challenge level, then we return true early.
	if chalLevel == protocol.NewBlockChallengeLevel() && claimId.IsSome() {
		return true, nil
	}
	ancestry, err := ht.ComputeAncestors(
		ctx,
		eg.Id(),
		0, /* block num (unimportant here) */
	)
	if err != nil {
		// If the edge we were looking for had no direct ancestry links, we return false.
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		// Otherwise, we received a real error in the computation.
		return false, err
	}
	return len(ancestry) > 0, nil
}

// Computes the root edge for a given child edge at a challenge level.
// In a challenge that looks like this:
//
//	      /--5---6-----8-----------16A = Alice
//	0-----4
//	      \--5'--6'----8'----------16B = Bob
//
// where Alice is the honest party, edge 0-16A is the honest root edge.
func (ht *RoyalChallengeTree) honestRootAncestorAtChallengeLevel(
	childEdge protocol.ReadOnlyEdge,
	challengeLevel protocol.ChallengeLevel,
) (protocol.ReadOnlyEdge, error) {
	originId := childEdge.OriginId()
	// // Otherwise, finds the honest root edge at the appropriate challenge level.
	rootEdgesAtLevel, ok := ht.royalRootEdgesByLevel.TryGet(challengeLevel)
	if !ok || rootEdgesAtLevel == nil {
		return nil, fmt.Errorf("no honest edges found at challenge level %d", challengeLevel)
	}
	rootAncestor, found := findOriginEdge(originId, rootEdgesAtLevel)
	if !found {
		return nil, fmt.Errorf("no honest root edge with origin id %#x found at challenge level %d", originId, challengeLevel)
	}
	return rootAncestor, nil
}

// Gets the edge a specified edge claims, if any.
func (ht *RoyalChallengeTree) getClaimedEdge(edge protocol.ReadOnlyEdge) (protocol.SpecEdge, error) {
	if edge.ClaimId().IsNone() {
		return nil, errors.New("does not claim any edge")
	}
	claimId := edge.ClaimId().Unwrap()
	claimIdHash := [32]byte(claimId)
	claimedBlockEdge, ok := ht.edges.TryGet(protocol.EdgeId{Hash: claimIdHash})
	if !ok {
		return nil, errors.New("claimed edge not found")
	}
	return claimedBlockEdge, nil
}

// Finds an edge in a list with a specified origin id.
func findOriginEdge(originId protocol.OriginId, edges *threadsafe.Slice[protocol.SpecEdge]) (protocol.SpecEdge, bool) {
	var originEdge protocol.SpecEdge
	found := edges.Find(func(_ int, e protocol.SpecEdge) bool {
		if e.OriginId() == originId {
			originEdge = e
			return true
		}
		return false
	})
	return originEdge, found
}

func errNotFound(id protocol.EdgeId) error {
	return errors.Wrapf(ErrNotFound, "id=%#x", id)
}
