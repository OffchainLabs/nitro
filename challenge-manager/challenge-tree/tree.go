// Package challengetree includes logic for keeping track of royal edges within a challenge
// with utilities for computing cumulative path timers for said edges. This is helpful during
// the confirmation process needed by edge trackers.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package challengetree

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/pkg/errors"
)

// MetadataReader can read certain information about edges from the backend.
type MetadataReader interface {
	AssertionUnrivaledBlocks(ctx context.Context, assertionHash protocol.AssertionHash) (uint64, error)
	TopLevelAssertion(ctx context.Context, edgeId protocol.EdgeId) (protocol.AssertionHash, error)
	TopLevelClaimHeights(ctx context.Context, edgeId protocol.EdgeId) (protocol.OriginHeights, error)
	SpecChallengeManager(ctx context.Context) (protocol.SpecChallengeManager, error)
	ReadAssertionCreationInfo(
		ctx context.Context, id protocol.AssertionHash,
	) (*protocol.AssertionCreatedInfo, error)
}

type creationTime uint64

// OriginPlusMutualId combines a mutual id and origin id as a key for a mapping.
// This is used for computing the rivals of an edge, as all rivals share a mutual id.
// However, we also add the origin id as that allows us to namespace to lookup
// to a specific challenge.
type OriginPlusMutualId [64]byte

func buildEdgeCreationTimeKey(originId protocol.OriginId, mutualId protocol.MutualId) OriginPlusMutualId {
	var key OriginPlusMutualId
	copy(key[0:32], originId[:])
	copy(key[32:64], mutualId[:])
	return key
}

// RoyalChallengeTree keeps track of royal edges the honest node agrees with in a particular challenge.
// All edges tracked in this data structure are part of the same, top-level assertion challenge.
type RoyalChallengeTree struct {
	edges                 *threadsafe.Map[protocol.EdgeId, protocol.SpecEdge]
	edgeCreationTimes     *threadsafe.Map[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]]
	topLevelAssertionHash protocol.AssertionHash
	metadataReader        MetadataReader
	histChecker           l2stateprovider.HistoryChecker
	validatorName         string
	totalChallengeLevels  uint8
	royalRootEdgesByLevel *threadsafe.Map[protocol.ChallengeLevel, *threadsafe.Slice[protocol.SpecEdge]]
}

func New(
	assertionHash protocol.AssertionHash,
	metadataReader MetadataReader,
	histChecker l2stateprovider.HistoryChecker,
	numBigStepLevels uint8,
	validatorName string,
) *RoyalChallengeTree {
	return &RoyalChallengeTree{
		edges:                 threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](threadsafe.MapWithMetric[protocol.EdgeId, protocol.SpecEdge]("edges")),
		edgeCreationTimes:     threadsafe.NewMap[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](threadsafe.MapWithMetric[OriginPlusMutualId, *threadsafe.Map[protocol.EdgeId, creationTime]]("edgeCreationTimes")),
		topLevelAssertionHash: assertionHash,
		metadataReader:        metadataReader,
		histChecker:           histChecker,
		validatorName:         validatorName,
		// The total number of challenge levels include block challenges, small step challenges, and N big step challenges.
		totalChallengeLevels:  numBigStepLevels + 2,
		royalRootEdgesByLevel: threadsafe.NewMap[protocol.ChallengeLevel, *threadsafe.Slice[protocol.SpecEdge]](threadsafe.MapWithMetric[protocol.ChallengeLevel, *threadsafe.Slice[protocol.SpecEdge]]("royalRootEdgesByLevel")),
	}
}

// RoyalBlockChallengeRootEdge gets the royal, root challenge block edge for the top level assertion
// being challenged.
func (ht *RoyalChallengeTree) RoyalBlockChallengeRootEdge() (protocol.ReadOnlyEdge, error) {
	// In our locally tracked challenge tree implementation, the
	// block challenge level is equal to the total challenge levels - 1.
	blockChallengeLevel := protocol.ChallengeLevel(ht.totalChallengeLevels) - 1
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

func (ht *RoyalChallengeTree) GetEdges() *threadsafe.Map[protocol.EdgeId, protocol.SpecEdge] {
	return ht.edges
}

func (ht *RoyalChallengeTree) HasRoyalEdge(edgeId protocol.EdgeId) bool {
	return ht.edges.Has(edgeId)
}

func (ht *RoyalChallengeTree) IsUnrivaledAtBlockNum(edge protocol.ReadOnlyEdge, blockNum uint64) (bool, error) {
	return ht.UnrivaledAtBlockNum(edge, blockNum)
}

func (ht *RoyalChallengeTree) TimeUnrivaled(edge protocol.ReadOnlyEdge, blockNum uint64) (uint64, error) {
	return ht.LocalTimer(edge, blockNum)
}

// Obtains the lowermost edges across all subchallenges that are royal.
// To do this, we fetch all royal, tracked edges that do not have children.
func (ht *RoyalChallengeTree) GetAllRoyalLeaves(ctx context.Context) ([]protocol.SpecEdge, error) {
	royalLeaves := make([]protocol.SpecEdge, 0)
	if err := ht.edges.ForEach(func(_ protocol.EdgeId, edge protocol.SpecEdge) error {
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

func (ht *RoyalChallengeTree) BlockChallengeRootEdge(ctx context.Context) (protocol.SpecEdge, error) {
	blockChalEdges, ok := ht.royalRootEdgesByLevel.TryGet(protocol.ChallengeLevel(ht.totalChallengeLevels) - 1)
	if !ok {
		return nil, errors.New("no block challenge root edge found")
	}
	if blockChalEdges.Len() != 1 {
		return nil, errors.New("expected exactly one block challenge root edge")
	}
	return blockChalEdges.Get(0).Unwrap(), nil
}

func (ht *RoyalChallengeTree) findClaimingEdge(
	ctx context.Context, claimedEdge protocol.EdgeId,
) (protocol.SpecEdge, bool) {
	var foundEdge protocol.SpecEdge
	var ok bool
	_ = ht.edges.ForEach(func(_ protocol.EdgeId, edge protocol.SpecEdge) error {
		if edge.ClaimId().IsNone() {
			return nil
		}
		if edge.ClaimId().Unwrap() == protocol.ClaimId(claimedEdge.Hash) {
			foundEdge = edge
			ok = true
		}
		return nil
	})
	return foundEdge, ok
}
