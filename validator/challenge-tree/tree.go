package challengetree

import (
	"context"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util/commitments"
	"github.com/OffchainLabs/challenge-protocol-v2/util/option"
	"github.com/OffchainLabs/challenge-protocol-v2/util/threadsafe"
	"github.com/pkg/errors"
)

// MetadataReader can read certain information about edges from the backend.
type MetadataReader interface {
	AssertionUnrivaledTime(ctx context.Context, assertionId protocol.AssertionId) (uint64, error)
	TopLevelAssertion(ctx context.Context, edgeId protocol.EdgeId) (protocol.AssertionId, error)
	TopLevelClaimHeights(ctx context.Context, edgeId protocol.EdgeId) (*protocol.OriginHeights, error)
	SpecChallengeManager(ctx context.Context) (protocol.SpecChallengeManager, error)
	GetAssertionNum(ctx context.Context, assertionHash protocol.AssertionId) (protocol.AssertionSequenceNumber, error)
	ReadAssertionCreationInfo(
		ctx context.Context, seqNum protocol.AssertionSequenceNumber,
	) (*protocol.AssertionCreatedInfo, error)
}

type creationTime uint64

// An honestChallengeTree keeps track of edges the honest node agrees with in a particular challenge.
// All edges tracked in this data structure are part of the same, top-level assertion challenge.
type HonestChallengeTree struct {
	edges                         *threadsafe.Map[protocol.EdgeId, protocol.SpecEdge]
	mutualIds                     *threadsafe.Map[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]]
	topLevelAssertionId           protocol.AssertionId
	honestBlockChalLevelZeroEdge  option.Option[protocol.ReadOnlyEdge]
	honestBigStepLevelZeroEdges   *threadsafe.Slice[protocol.ReadOnlyEdge]
	honestSmallStepLevelZeroEdges *threadsafe.Slice[protocol.ReadOnlyEdge]
	metadataReader                MetadataReader
	histChecker                   statemanager.HistoryChecker
	validatorName                 string
}

func New(
	prevAssertionId protocol.AssertionId,
	metadataReader MetadataReader,
	histChecker statemanager.HistoryChecker,
	validatorName string,
) *HonestChallengeTree {
	return &HonestChallengeTree{
		edges:                         threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds:                     threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		topLevelAssertionId:           prevAssertionId,
		honestBlockChalLevelZeroEdge:  option.None[protocol.ReadOnlyEdge](),
		honestBigStepLevelZeroEdges:   threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		honestSmallStepLevelZeroEdges: threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		metadataReader:                metadataReader,
		histChecker:                   histChecker,
		validatorName:                 validatorName,
	}
}

// AddEdge to the honest challenge tree. Only honest edges are tracked, but we also keep track
// of rival ids in a mutual ids mapping internally for extra book-keeping.
func (ht *HonestChallengeTree) AddEdge(ctx context.Context, eg protocol.SpecEdge) error {
	prevAssertionId, err := ht.metadataReader.TopLevelAssertion(ctx, eg.Id())
	if err != nil {
		return errors.Wrapf(err, "could not get top level assertion for edge %#x", eg.Id())
	}
	if ht.topLevelAssertionId != prevAssertionId {
		// Do nothing - this edge should not be part of this challenge tree.
		return nil
	}
	prevAssertionSeqNum, err := ht.metadataReader.GetAssertionNum(ctx, prevAssertionId)
	if err != nil {
		return err
	}
	prevCreationInfo, err := ht.metadataReader.ReadAssertionCreationInfo(ctx, prevAssertionSeqNum)
	if err != nil {
		return err
	}

	// We only track edges we fully agree with (honest edges).
	startHeight, startCommit := eg.StartCommitment()
	endHeight, endCommit := eg.EndCommitment()
	heights, err := ht.metadataReader.TopLevelClaimHeights(ctx, eg.Id())
	if err != nil {
		return errors.Wrapf(err, "could not get claim heights for edge %#x", eg.Id())
	}
	agreement, err := ht.histChecker.AgreesWithHistoryCommitment(
		ctx,
		eg.GetType(),
		prevCreationInfo.InboxMaxCount.Uint64(),
		heights,
		commitments.History{
			Height: uint64(startHeight),
			Merkle: startCommit,
		},
		commitments.History{
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
		id := eg.Id()
		ht.edges.Put(id, eg)
		if !eg.ClaimId().IsNone() {
			switch eg.GetType() {
			case protocol.BlockChallengeEdge:
				ht.honestBlockChalLevelZeroEdge = option.Some(protocol.ReadOnlyEdge(eg))
			case protocol.BigStepChallengeEdge:
				ht.honestBigStepLevelZeroEdges.Push(eg)
			case protocol.SmallStepChallengeEdge:
				ht.honestSmallStepLevelZeroEdges.Push(eg)
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
			ht.mutualIds.Put(mutualId, threadsafe.NewMap[protocol.EdgeId, creationTime]())
			mutuals = ht.mutualIds.Get(mutualId)
		}
		mutuals.Put(eg.Id(), creationTime(eg.CreatedAtBlock()))
	}
	return nil
}

func (ht *HonestChallengeTree) GetEdges() *threadsafe.Map[protocol.EdgeId, protocol.SpecEdge] {
	return ht.edges
}
