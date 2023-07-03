// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

package challengetree

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/threadsafe"
	bisection "github.com/OffchainLabs/challenge-protocol-v2/math"
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

// HonestPathTimer computes the honest path timer of a specified honest edge
// at a block number and its list of honest ancestors.
//
// Consider the following set of edges in a challenge where evil
// edges are marked with a ' and a *:
//
//		     /---6---8
//	  0-----4
//		     \---6'--8'
//
// The honest branch is the one that goes from 0-8. The evil edge is 0-8'.
// The evil edge 0-8' bisects, but agrees with the honest one from 0-4.
// Therefore, there is only a single 0-4 edge in the set.
//
// In this case, the set of honest ancestors for 4-6 is the following:
//
//	{4-8, 0-8}
func (ht *HonestChallengeTree) HonestPathTimer(
	ctx context.Context,
	queryingFor protocol.EdgeId,
	blockNumber uint64,
) (PathTimer, HonestAncestors, error) {
	wantedEdge, ok := ht.edges.TryGet(queryingFor)
	if !ok {
		return 0, nil, errNotFound(queryingFor)
	}
	if ht.honestBlockChalLevelZeroEdge.IsNone() {
		return 0, nil, ErrNoHonestTopLevelEdge
	}
	honestLevelZero := ht.honestBlockChalLevelZeroEdge.Unwrap()

	// Get assertion's unrivaled time and use that as the start
	// of our path timer.
	timer, err := ht.metadataReader.AssertionUnrivaledBlocks(ctx, ht.topLevelAssertionHash)
	if err != nil {
		return 0, nil, err
	}
	pathTimer := PathTimer(timer)

	// Figure out what kind of edge this is, and apply different logic based on it.
	switch wantedEdge.GetType() {
	case protocol.BlockChallengeEdge:
		// If the edge is a block challenge edge, we simply search for the wanted edge's ancestors
		// in the block challenge starting from the honest, level zero edge and return
		// the computed ancestor ids list.
		start := honestLevelZero
		searchFor := wantedEdge
		blockChalTimer, ancestry, err := ht.findAncestorsInChallenge(ctx, start, searchFor, blockNumber)
		if err != nil {
			return 0, nil, err
		}
		// The solidity confirmations function expects a child-to-parent ordering,
		// which is the reverse of our computed list.
		containers.Reverse(ancestry)
		pathTimer += blockChalTimer
		return pathTimer, ancestry, nil
	case protocol.BigStepChallengeEdge:
		ancestry := make([]protocol.EdgeId, 0)
		originId := wantedEdge.OriginId()

		// If the edge is a big step challenge edge, we first find out the honest big step
		// level zero edge it is a child of.
		bigStepLevelZero, ok := findOriginEdge(originId, ht.honestBigStepLevelZeroEdges)
		if !ok {
			return 0, ancestry, errNoLevelZero(originId)
		}

		// From there, we compute its ancestors.
		start := bigStepLevelZero
		searchFor := wantedEdge
		bigStepTimer, bigStepAncestry, err := ht.findAncestorsInChallenge(ctx, start, searchFor, blockNumber)
		if err != nil {
			return 0, ancestry, err
		}
		pathTimer += bigStepTimer

		// Next, we go up to the block challenge level by getting the edge the big step
		// level zero edge claims as its claim id.
		claimedEdge, err := ht.getClaimedEdge(bigStepLevelZero)
		if err != nil {
			return 0, ancestry, err
		}

		// We compute the block ancestry from there.
		start = honestLevelZero
		searchFor = claimedEdge
		blockChalTimer, blockChalAncestry, err := ht.findAncestorsInChallenge(ctx, start, searchFor, blockNumber)
		if err != nil {
			return 0, ancestry, err
		}
		pathTimer += blockChalTimer

		// Finally, the solidity confirmations function expects a child-to-parent ordering,
		// which is the reverse of our computed list. This list should contain
		// the block challenge claimed edge id that links the edge between challenge types.
		ancestry = append(ancestry, blockChalAncestry...)
		ancestry = append(ancestry, claimedEdge.Id())
		ancestry = append(ancestry, bigStepAncestry...)

		containers.Reverse(ancestry)
		return pathTimer, ancestry, nil
	case protocol.SmallStepChallengeEdge:
		ancestry := make([]protocol.EdgeId, 0)
		originId := wantedEdge.OriginId()

		// If the edge is a small step challenge edge, we first find out the honest small step
		// level zero edge it is a child of.
		smallStepLevelZero, ok := findOriginEdge(originId, ht.honestSmallStepLevelZeroEdges)
		if !ok {
			return 0, nil, errNoLevelZero(originId)
		}

		// From there, we compute its ancestors.
		start := smallStepLevelZero
		searchFor := wantedEdge
		smallStepTimer, smallStepAncestry, err := ht.findAncestorsInChallenge(ctx, start, searchFor, blockNumber)
		if err != nil {
			return 0, nil, err
		}
		pathTimer += smallStepTimer

		// Next, we go up to the big step challenge level by getting the edge the small step
		// level zero edge claims as its claim id.
		claimedBigStepEdge, err := ht.getClaimedEdge(smallStepLevelZero)
		if err != nil {
			return 0, nil, err
		}

		originId = claimedBigStepEdge.OriginId()
		bigStepLevelZero, ok := findOriginEdge(originId, ht.honestBigStepLevelZeroEdges)
		if !ok {
			return 0, nil, errNoLevelZero(originId)
		}

		// From there, we compute its ancestors.
		start = bigStepLevelZero
		searchFor = claimedBigStepEdge
		bigStepTimer, bigStepAncestry, err := ht.findAncestorsInChallenge(ctx, start, searchFor, blockNumber)
		if err != nil {
			return 0, nil, err
		}
		pathTimer += bigStepTimer

		// Next, we go up to the block challenge level by getting the edge the big step
		// level zero edge claims as its claim id.
		claimedBlockEdge, err := ht.getClaimedEdge(bigStepLevelZero)
		if err != nil {
			return 0, nil, err
		}
		start = honestLevelZero
		searchFor = claimedBlockEdge
		blockChalTimer, blockAncestry, err := ht.findAncestorsInChallenge(ctx, start, searchFor, blockNumber)
		if err != nil {
			return 0, nil, err
		}
		pathTimer += blockChalTimer

		// Finally, the solidity confirmations function expects a child-to-parent ordering,
		// which is the reverse of our computed list. This list should contain
		// the claimed edge ids that link the edge between challenge types.
		ancestry = append(ancestry, blockAncestry...)
		ancestry = append(ancestry, claimedBlockEdge.Id())
		ancestry = append(ancestry, bigStepAncestry...)
		ancestry = append(ancestry, claimedBigStepEdge.Id())
		ancestry = append(ancestry, smallStepAncestry...)
		containers.Reverse(ancestry)
		return pathTimer, ancestry, nil
	default:
		return 0, nil, fmt.Errorf("edge with type %v not supported", wantedEdge.GetType())
	}
}

// Computes the list of ancestors in a challenge type from a starting edge down
// to a specified child edge. The edge we are querying must be a child of this start edge
// for this function to succeed without error.
func (ht *HonestChallengeTree) findAncestorsInChallenge(
	ctx context.Context,
	start protocol.ReadOnlyEdge,
	queryingFor protocol.ReadOnlyEdge,
	blockNumber uint64,
) (PathTimer, []protocol.EdgeId, error) {
	found := false
	curr := start
	pathTimer, err := ht.localTimer(curr, blockNumber)
	if err != nil {
		return 0, nil, err
	}

	ancestry := make([]protocol.EdgeId, 0)
	wantedEdgeStart, _ := queryingFor.StartCommitment()

	for {
		if ctx.Err() != nil {
			return 0, nil, ctx.Err()
		}
		if curr.Id() == queryingFor.Id() {
			found = true
			break
		}
		ancestry = append(ancestry, curr.Id())

		currStart, _ := curr.StartCommitment()
		currEnd, _ := curr.EndCommitment()
		bisectTo, err := bisection.Bisect(uint64(currStart), uint64(currEnd))
		if err != nil {
			return 0, nil, errors.Wrapf(err, "could not bisect start=%d, end=%d", currStart, currEnd)
		}
		// If the wanted edge's start commitment is < the bisection height of the current
		// edge in the loop, it means it is part of its lower children.
		if uint64(wantedEdgeStart) < bisectTo {
			lowerSnapshot, lowerErr := curr.LowerChild(ctx)
			if lowerErr != nil {
				return 0, nil, errors.Wrapf(lowerErr, "could not get lower child for edge %#x", curr.Id())
			}
			if lowerSnapshot.IsNone() {
				return 0, nil, fmt.Errorf("edge %#x had no lower child", curr.Id())
			}
			curr = ht.edges.Get(lowerSnapshot.Unwrap())
		} else {
			// Else, it is part of the upper children.
			upperSnapshot, upperErr := curr.UpperChild(ctx)
			if upperErr != nil {
				return 0, nil, errors.Wrapf(upperErr, "could not get upper child for edge %#x", curr.Id())
			}
			if upperSnapshot.IsNone() {
				return 0, nil, fmt.Errorf("edge %#x had no upper child", curr.Id())
			}
			curr = ht.edges.Get(upperSnapshot.Unwrap())
		}
		timer, err := ht.localTimer(curr, blockNumber)
		if err != nil {
			return 0, nil, err
		}
		pathTimer += timer
	}
	if !found {
		return 0, nil, errNotFound(queryingFor.Id())
	}
	return PathTimer(pathTimer), ancestry, nil
}

// Gets the edge a specified edge claims, if any.
func (ht *HonestChallengeTree) getClaimedEdge(edge protocol.ReadOnlyEdge) (protocol.SpecEdge, error) {
	if edge.ClaimId().IsNone() {
		return nil, errors.New("does not claim any edge")
	}
	claimId := edge.ClaimId().Unwrap()
	claimedBlockEdge, ok := ht.edges.TryGet(protocol.EdgeId(claimId))
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

func errNoLevelZero(originId protocol.OriginId) error {
	return errors.Wrapf(ErrNoLevelZero, "originId=%#x", originId)
}
