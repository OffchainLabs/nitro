// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

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
)

// PathTimer for an honest edge defined as the cumulative unrivaled time
// of it and its honest ancestors all the way up to the assertion chain level.
// This also includes the time the assertion, which the challenge corresponds to,
// has been unrivaled.
type PathTimer uint64

// HonestAncestors of an edge id all the way up to and including the
// block challenge level zero edge.
type HonestAncestors []protocol.EdgeId

// EdgeLocalTimer is the local, unrivaled timer of a specific edge.
type EdgeLocalTimer uint64

// AncestorsQueryResponse contains a list of ancestor edge ids and
// their respective local timers. Both slices have the same length and correspond
// to each other.
type AncestorsQueryResponse struct {
	AncestorLocalTimers []EdgeLocalTimer
	AncestorEdgeIds     HonestAncestors
}

// HasConfirmableAncestor checks if any of an edge's honest ancestors have a cumulative path timer
// that is greater than or equal to a challenge period worth of blocks. It takes in a list of
// local timers for an edge's ancestors and a block number to compute each entry's cumulative
// timer from this list.
//
// IMPORTANT: The list of ancestors must be ordered from child to root edge, where the root edge timer is
// the last element in the list of local timers.
func (ht *HonestChallengeTree) HasConfirmableAncestor(
	ctx context.Context,
	honestAncestorLocalTimers []EdgeLocalTimer,
	challengePeriodBlocks uint64,
) (bool, error) {
	if len(honestAncestorLocalTimers) == 0 {
		return false, nil
	}
	assertionUnrivaledNumBlocks, err := ht.metadataReader.AssertionUnrivaledBlocks(
		ctx, ht.topLevelAssertionHash,
	)
	if err != nil {
		return false, err
	}

	// Computes the cumulative sum for each element in the list.
	cumulativeTimers := make([]PathTimer, 0)
	lastAncestorTimer := honestAncestorLocalTimers[len(honestAncestorLocalTimers)-1] + EdgeLocalTimer(assertionUnrivaledNumBlocks)

	// If we only have a single honest ancestor, check if it plus the assertion unrivaled
	// timer is enough to be confirmable and return.
	if len(honestAncestorLocalTimers) == 1 {
		return uint64(lastAncestorTimer) >= challengePeriodBlocks, nil
	}

	// We start with the last ancestor, which should also include the top-level assertion's unrivaled timer.
	cumulativeTimers = append(cumulativeTimers, PathTimer(lastAncestorTimer))

	// Loop over everything except the last element, which is the root edge as we already
	// appended it in the lines above.
	for i, ancestorTimer := range honestAncestorLocalTimers[:len(honestAncestorLocalTimers)-1] {
		cumulativeTimers = append(cumulativeTimers, cumulativeTimers[i]+PathTimer(ancestorTimer))
	}

	// Then checks if any of them has a cumulative timer greater than
	// or equal to a challenge period worth of blocks. We loop in reverse because the cumulative timers slice is monotonically
	// increasing and this could help us exit the loop earlier in case we find an ancestor that is confirmable.
	for i := len(cumulativeTimers) - 1; i >= 0; i-- {
		if uint64(cumulativeTimers[i]) >= challengePeriodBlocks {
			return true, nil
		}
	}
	return false, nil
}

// ComputeHonestPathTimer for an honest edge at a block number given its ancestors'
// local timers. It adds up all their values including the assertion
// unrivaled timer and the edge's local timer.
func (ht *HonestChallengeTree) ComputeHonestPathTimer(
	ctx context.Context,
	edgeId protocol.EdgeId,
	ancestorLocalTimers []EdgeLocalTimer,
	blockNumber uint64,
) (PathTimer, error) {
	edge, ok := ht.edges.TryGet(edgeId)
	if !ok {
		return 0, errNotFound(edgeId)
	}
	edgeLocalTimer, err := ht.localTimer(edge, blockNumber)
	if err != nil {
		return 0, err
	}
	total := PathTimer(edgeLocalTimer)
	assertionUnrivaledTimer, err := ht.metadataReader.AssertionUnrivaledBlocks(
		ctx, ht.topLevelAssertionHash,
	)
	if err != nil {
		return 0, err
	}
	total += PathTimer(assertionUnrivaledTimer)
	for _, timer := range ancestorLocalTimers {
		total += PathTimer(timer)
	}
	return total, nil
}

// ComputeAncestorsWithTimers computes the ancestors of the given edge and their respective path timers, even
// across challenge levels. Ancestor lists are linked through challenge levels via claimed edges. It is generalized
// to any number of challenge levels in the protocol.
func (ht *HonestChallengeTree) ComputeAncestorsWithTimers(
	ctx context.Context,
	edgeId protocol.EdgeId,
	blockNumber uint64,
) (*AncestorsQueryResponse, error) {
	// Checks if the edge exists before performing any computation.
	startEdge, ok := ht.edges.TryGet(edgeId)
	if !ok {
		return nil, errNotFound(edgeId)
	}
	currentChallengeLevel := startEdge.GetReversedChallengeLevel()

	// Set a cursor at the edge we start from. We will update this cursor
	// as we advance in this function.
	currentEdge := startEdge

	ancestry := make([]protocol.EdgeId, 0)
	localTimers := make([]EdgeLocalTimer, 0)

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
		ancestorLocalTimers, ancestorsAtLevel, err := ht.findHonestAncestorsWithinChallengeLevel(ctx, rootEdge, currentEdge, blockNumber)
		if err != nil {
			return nil, err
		}

		// Expand the total ancestry and timers slices. We want ancestors from
		// the bottom-up, so we must reverse the output slice from the find function.
		containers.Reverse(ancestorLocalTimers)
		containers.Reverse(ancestorsAtLevel)
		ancestry = append(ancestry, ancestorsAtLevel...)
		localTimers = append(localTimers, ancestorLocalTimers...)

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
		claimEdgeLocalTimer, err := ht.localTimer(nextLevelClaimedEdge, blockNumber)
		if err != nil {
			return nil, err
		}

		// Update the cursor to be the claimed edge at the next challenge level.
		currentEdge = nextLevelClaimedEdge

		// Include the next level claimed edge in the ancestry list.
		ancestry = append(ancestry, nextLevelClaimedEdge.Id())
		localTimers = append(localTimers, EdgeLocalTimer(claimEdgeLocalTimer))
	}

	// If the ancestry is empty, we just return an empty response.
	if len(ancestry) == 0 {
		return &AncestorsQueryResponse{
			AncestorLocalTimers: make([]EdgeLocalTimer, 0),
			AncestorEdgeIds:     ancestry,
		}, nil
	}

	return &AncestorsQueryResponse{
		AncestorLocalTimers: localTimers,
		AncestorEdgeIds:     ancestry,
	}, nil
}

// Computes the list of ancestors in a challenge level from a root edge down
// to a specified child edge within the same level. The edge we are querying must be
// a child of this start edge for this function to succeed without error.
func (ht *HonestChallengeTree) findHonestAncestorsWithinChallengeLevel(
	ctx context.Context,
	rootEdge protocol.ReadOnlyEdge,
	queryingFor protocol.ReadOnlyEdge,
	blockNumber uint64,
) ([]EdgeLocalTimer, []protocol.EdgeId, error) {
	found := false
	cursor := rootEdge
	ancestry := make([]protocol.EdgeId, 0)
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
		ancestry = append(ancestry, cursor.Id())
		timer, err := ht.localTimer(cursor, blockNumber)
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
				return nil, nil, fmt.Errorf("edge %#x had no lower child", cursor.Id())
			}
			cursor = ht.edges.Get(lowerChild.Unwrap())
		} else {
			// Else, it is part of the upper children.
			upperChild, upperErr := cursor.UpperChild(ctx)
			if upperErr != nil {
				return nil, nil, errors.Wrapf(upperErr, "could not get upper child for edge %#x", cursor.Id())
			}
			if upperChild.IsNone() {
				return nil, nil, fmt.Errorf("edge %#x had no upper child", cursor.Id())
			}
			cursor = ht.edges.Get(upperChild.Unwrap())
		}
	}
	if !found {
		return nil, nil, errNotFound(queryingFor.Id())
	}
	return localTimers, ancestry, nil
}

// Computes the root edge for a given child edge at a challenge level.
// In a challenge that looks like this:
//
//	      /--5---6-----8-----------16A = Alice
//	0-----4
//	      \--5'--6'----8'----------16B = Bob
//
// where Alice is the honest party, edge 0-16A is the honest root edge.
func (ht *HonestChallengeTree) honestRootAncestorAtChallengeLevel(
	childEdge protocol.ReadOnlyEdge,
	challengeLevel protocol.ChallengeLevel,
) (protocol.ReadOnlyEdge, error) {
	originId := childEdge.OriginId()
	// // Otherwise, finds the honest root edge at the appropriate challenge level.
	rootEdgesAtLevel, ok := ht.honestRootEdgesByLevel.TryGet(challengeLevel)
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
func (ht *HonestChallengeTree) getClaimedEdge(edge protocol.ReadOnlyEdge) (protocol.SpecEdge, error) {
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
func findOriginEdge(originId protocol.OriginId, edges *threadsafe.Slice[protocol.ReadOnlyEdge]) (protocol.ReadOnlyEdge, bool) {
	var originEdge protocol.ReadOnlyEdge
	found := edges.Find(func(_ int, e protocol.ReadOnlyEdge) bool {
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
