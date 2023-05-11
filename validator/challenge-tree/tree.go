package challengetree

import (
	"context"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/OffchainLabs/challenge-protocol-v2/util/threadsafe"
	"github.com/pkg/errors"
)

// MetadataReader can read certain information about edges from the backend.
type MetadataReader interface {
	TopLevelAssertion(ctx context.Context, edgeId protocol.EdgeId) (protocol.AssertionId, error)
	ClaimHeights(ctx context.Context, edgeId protocol.EdgeId) (*ClaimHeights, error)
}

// ClaimHeights returns the heights of the claim data for an edge, all the way up to
// the top-level assertion chain.
type ClaimHeights struct {
	AssertionClaimHeight      uint64
	BlockChallengeClaimHeight uint64
	BigStepClaimHeight        uint64
}

// Agreement encompasses whether or not a local node agrees with a edge's commitments.
// Either the edge is honest, we agree with its start commit, or disagree entirely.
type Agreement struct {
	IsHonestEdge          bool
	AgreesWithStartCommit bool
}

// HistoryChecker can verify to what extent we agree with an edge's history commitments locally.
type HistoryChecker interface {
	AgreesWithHistoryCommitment(
		ctx context.Context,
		heights *ClaimHeights,
		startCommit,
		endCommit util.HistoryCommitment,
	) (Agreement, error)
}

// An honestChallengeTree keeps track of edges the honest node agrees with in a particular challenge.
// All edges tracked in this data structure are part of the same, top-level assertion challenge.
type HonestChallengeTree struct {
	edges                            *threadsafe.Map[protocol.EdgeId, protocol.EdgeSnapshot]
	mutualIds                        *threadsafe.Map[protocol.MutualId, *threadsafe.Set[protocol.EdgeId]]
	topLevelAssertionId              protocol.AssertionId
	honestBlockChalLevelZeroEdge     util.Option[protocol.EdgeSnapshot]
	honestBigStepChalLevelZeroEdge   util.Option[protocol.EdgeSnapshot]
	honestSmallStepChalLevelZeroEdge util.Option[protocol.EdgeSnapshot]
	metadataReader                   MetadataReader
	histChecker                      HistoryChecker
}

// AddEdge to the honest challenge tree. Only honest edges are tracked, but we also keep track
// of rival ids in a mutual ids mapping internally for extra book-keeping.
func (ht *HonestChallengeTree) AddEdge(ctx context.Context, eg protocol.EdgeSnapshot) error {
	prevAssertionId, err := ht.metadataReader.TopLevelAssertion(ctx, eg.Id())
	if err != nil {
		return errors.Wrapf(err, "could not get top level assertion for edge %#x", eg.Id())
	}
	if ht.topLevelAssertionId != prevAssertionId {
		// Do nothing - this edge should not be part of this challenge tree.
		return nil
	}

	// We only track edges we fully agree with (honest edges).
	startHeight, startCommit := eg.StartCommitment()
	endHeight, endCommit := eg.EndCommitment()
	heights, err := ht.metadataReader.ClaimHeights(ctx, eg.Id())
	if err != nil {
		return errors.Wrapf(err, "could not get claim heights for edge %#x", eg.Id())
	}
	agreement, err := ht.histChecker.AgreesWithHistoryCommitment(
		ctx,
		heights,
		util.HistoryCommitment{
			Height: uint64(startHeight),
			Merkle: startCommit,
		},
		util.HistoryCommitment{
			Height: uint64(endHeight),
			Merkle: endCommit,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "could not check if agrees with history commit for edge %#x", eg.Id())
	}

	// If we agree with the edge, we add it to our edges mapping and if it is level zero,
	// we keep track of it specifically in our struct.
	if agreement.IsHonestEdge {
		ht.edges.Put(eg.Id(), eg)
		if !eg.ClaimId().IsNone() {
			switch eg.GetType() {
			case protocol.BlockChallengeEdge:
				ht.honestBlockChalLevelZeroEdge = util.Some(eg)
			case protocol.BigStepChallengeEdge:
				ht.honestBigStepChalLevelZeroEdge = util.Some(eg)
			case protocol.SmallStepChallengeEdge:
				ht.honestSmallStepChalLevelZeroEdge = util.Some(eg)
			default:
			}
		}
	}

	// Check if the edge id should be added to the rivaled edges set.
	// Here we only care about edges here that are either honest or those whose start
	// history commitments we agree with.
	if agreement.AgreesWithStartCommit || agreement.IsHonestEdge {
		mutualId := eg.MutualId()
		mutuals := ht.mutualIds.Get(mutualId)
		if mutuals == nil {
			ht.mutualIds.Put(mutualId, threadsafe.NewSet[protocol.EdgeId]())
			mutuals = ht.mutualIds.Get(mutualId)
		}
		mutuals.Insert(eg.Id())
	}
	return nil
}
