// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package challengetree includes logic for keeping track of royal edges within a challenge
// with utilities for computing cumulative path timers for said edges. This is helpful during
// the confirmation process needed by edge trackers.
package challengetree

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/offchainlabs/nitro/bold/chainabstraction"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	"github.com/offchainlabs/nitro/bold/layer2stateprovider"
)

// MetadataReader can read certain information about edges from the backend.
type MetadataReader interface {
	AssertionUnrivaledBlocks(ctx context.Context, assertionHash chainabstraction.AssertionHash) (uint64, error)
	TopLevelAssertion(ctx context.Context, edgeId chainabstraction.EdgeId) (chainabstraction.AssertionHash, error)
	TopLevelClaimHeights(ctx context.Context, edgeId chainabstraction.EdgeId) (chainabstraction.OriginHeights, error)
	SpecChallengeManager() chainabstraction.SpecChallengeManager
	ReadAssertionCreationInfo(
		ctx context.Context, id chainabstraction.AssertionHash,
	) (*chainabstraction.AssertionCreatedInfo, error)
}

type creationTime uint64

// OriginPlusMutualId combines a mutual id and origin id as a key for a mapping.
// This is used for computing the rivals of an edge, as all rivals share a mutual id.
// However, we also add the origin id as that allows us to namespace to lookup
// to a specific challenge.
type OriginPlusMutualId [64]byte

func buildEdgeCreationTimeKey(originId chainabstraction.OriginId, mutualId chainabstraction.MutualId) OriginPlusMutualId {
	var key OriginPlusMutualId
	copy(key[0:32], originId[:])
	copy(key[32:64], mutualId[:])
	return key
}

// RoyalChallengeTree keeps track of royal edges the honest validator agrees with in a particular challenge.
// All edges tracked in this data structure are part of the same, top-level assertion challenge.
type RoyalChallengeTree struct {
	edges                 *threadsafe.Map[chainabstraction.EdgeId, chainabstraction.SpecEdge]
	edgeCreationTimes     *threadsafe.Map[OriginPlusMutualId, *threadsafe.Map[chainabstraction.EdgeId, creationTime]]
	topLevelAssertionHash chainabstraction.AssertionHash
	metadataReader        MetadataReader
	histChecker           layer2stateprovider.HistoryChecker
	validatorName         string
	totalChallengeLevels  uint8
	royalRootEdgesByLevel *threadsafe.Map[chainabstraction.ChallengeLevel, *threadsafe.Slice[chainabstraction.SpecEdge]]
}

func New(
	assertionHash chainabstraction.AssertionHash,
	metadataReader MetadataReader,
	histChecker layer2stateprovider.HistoryChecker,
	numBigStepLevels uint8,
	validatorName string,
) *RoyalChallengeTree {
	return &RoyalChallengeTree{
		edges:                 threadsafe.NewMap[chainabstraction.EdgeId, chainabstraction.SpecEdge](threadsafe.MapWithMetric[chainabstraction.EdgeId, chainabstraction.SpecEdge]("edges")),
		edgeCreationTimes:     threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[chainabstraction.EdgeId, creationTime]](threadsafe.MapWithMetric[OriginPlusMutualId, *threadsafe.Map[chainabstraction.EdgeId, creationTime]]("edgeCreationTimes")),
		topLevelAssertionHash: assertionHash,
		metadataReader:        metadataReader,
		histChecker:           histChecker,
		validatorName:         validatorName,
		// The total number of challenge levels include block challenges, small step challenges, and N big step challenges.
		totalChallengeLevels:  numBigStepLevels + 2,
		royalRootEdgesByLevel: threadsafe.NewMap[chainabstraction.ChallengeLevel, *threadsafe.Slice[chainabstraction.SpecEdge]](threadsafe.MapWithMetric[chainabstraction.ChallengeLevel, *threadsafe.Slice[chainabstraction.SpecEdge]]("royalRootEdgesByLevel")),
	}
}

// RoyalBlockChallengeRootEdge gets the royal, root challenge block edge for the top level assertion
// being challenged.
func (ht *RoyalChallengeTree) RoyalBlockChallengeRootEdge() (chainabstraction.ReadOnlyEdge, error) {
	// In our locally tracked challenge tree implementation, the
	// block challenge level is equal to the total challenge levels - 1.
	blockChallengeLevel := chainabstraction.ChallengeLevel(ht.totalChallengeLevels) - 1
	if rootEdges, ok := ht.royalRootEdgesByLevel.TryGet(blockChallengeLevel); ok {
		if rootEdges.Len() != 1 {
			return nil, fmt.Errorf(
				"expected one royal root block challenge edge for challenged assertion %#x",
				ht.topLevelAssertionHash,
			)
		}
		return rootEdges.Get(0).Unwrap(), nil
	}
	return nil, fmt.Errorf("no royal root edges for block challenge level for assertion %#x", ht.topLevelAssertionHash)
}

var (
	ErrAlreadyBeingTracked              = errors.New("edge already being tracked")
	ErrMismatchedChallengeAssertionHash = errors.New("edge challenged assertion hash is not the expected one for the challenge")
)

func (ht *RoyalChallengeTree) GetEdges() *threadsafe.Map[chainabstraction.EdgeId, chainabstraction.SpecEdge] {
	return ht.edges
}

func (ht *RoyalChallengeTree) GetEdge(edgeId chainabstraction.EdgeId) (chainabstraction.SpecEdge, bool) {
	return ht.edges.TryGet(edgeId)
}

func (ht *RoyalChallengeTree) HasRoyalEdge(edgeId chainabstraction.EdgeId) bool {
	return ht.edges.Has(edgeId)
}

func (ht *RoyalChallengeTree) IsUnrivaledAtBlockNum(edge chainabstraction.ReadOnlyEdge, blockNum uint64) (bool, error) {
	return ht.UnrivaledAtBlockNum(edge, blockNum)
}

func (ht *RoyalChallengeTree) TimeUnrivaled(ctx context.Context, edge chainabstraction.ReadOnlyEdge, blockNum uint64) (uint64, error) {
	return ht.LocalTimer(ctx, edge, blockNum)
}

// Obtains the lowermost edges across all subchallenges that are royal.
// To do this, we fetch all royal, tracked edges that do not have children.
func (ht *RoyalChallengeTree) GetAllRoyalLeaves(ctx context.Context) ([]chainabstraction.SpecEdge, error) {
	royalLeaves := make([]chainabstraction.SpecEdge, 0)
	if err := ht.edges.ForEach(func(_ chainabstraction.EdgeId, edge chainabstraction.SpecEdge) error {
		hasChildren, err := edge.HasChildren(ctx)
		if err != nil {
			return err
		}
		if !hasChildren {
			royalLeaves = append(royalLeaves, edge)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return royalLeaves, nil
}

func (ht *RoyalChallengeTree) BlockChallengeRootEdge(ctx context.Context) (chainabstraction.SpecEdge, error) {
	blockChalEdges, ok := ht.royalRootEdgesByLevel.TryGet(chainabstraction.ChallengeLevel(ht.totalChallengeLevels) - 1)
	if !ok {
		return nil, errors.New("no block challenge root edge found")
	}
	if blockChalEdges.Len() != 1 {
		return nil, errors.New("expected exactly one block challenge root edge")
	}
	return blockChalEdges.Get(0).Unwrap(), nil
}

func (ht *RoyalChallengeTree) findClaimingEdge(claimedEdge chainabstraction.EdgeId) (chainabstraction.SpecEdge, bool) {
	var foundEdge chainabstraction.SpecEdge
	var ok bool
	_ = ht.edges.ForEach(func(_ chainabstraction.EdgeId, edge chainabstraction.SpecEdge) error {
		if edge.ClaimId().IsNone() {
			return nil
		}
		if edge.ClaimId().Unwrap() == chainabstraction.ClaimId(claimedEdge.Hash) {
			foundEdge = edge
			ok = true
		}
		return nil
	})
	return foundEdge, ok
}
