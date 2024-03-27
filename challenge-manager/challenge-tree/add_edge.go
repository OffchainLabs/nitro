package challengetree

import (
	"context"
	"strings"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// AddRoyalEdge known to be honest, such as those created by the local validator.
func (ht *RoyalChallengeTree) AddRoyalEdge(eg protocol.VerifiedRoyalEdge) error {
	id := eg.Id()
	if _, ok := ht.edges.TryGet(id); ok {
		// Already being tracked.
		return nil
	}
	if err := ht.keepTrackOfCreationTime(eg); err != nil {
		return err
	}
	ht.keepTrackOfHonestEdge(eg)
	return nil
}

// AddEdge to the honest challenge tree. Only honest edges are tracked, but we also keep track
// of rival ids in a mutual ids mapping internally for extra book-keeping.
func (ht *RoyalChallengeTree) AddEdge(ctx context.Context, eg protocol.SpecEdge) (isRoyal bool, err error) {
	edgeId := eg.Id()

	// Check if edge is already being tracked.
	if _, ok := ht.edges.TryGet(edgeId); ok {
		return false, ErrAlreadyBeingTracked
	}
	// Check if assertion hash is correct.
	if err = ht.checkAssertionHash(ctx, edgeId); err != nil {
		return false, errors.Wrapf(err, "could not check if the edge's assertion hash is correct %#x", edgeId)
	}
	if err = ht.keepTrackOfCreationTime(eg); err != nil {
		return false, errors.Wrapf(err, "could not track mutual id: %#x", edgeId)
	}
	hasHonestAncestry, err := ht.hasHonestAncestry(ctx, eg)
	if err != nil {
		return false, errors.Wrapf(err, "could not check if edge has honest ancestors: %#x", edgeId)
	}
	if !hasHonestAncestry {
		return false, nil
	}
	claimedAssertionHash, err := ht.claimedAssertionHash(ctx, eg)
	if err != nil {
		return false, errors.Wrapf(err, "could not fetch claimed assertion hash for edge: %#x", edgeId)
	}
	historyCommitRequest, err := ht.prepareHistoryCommitmentRequest(ctx, eg, claimedAssertionHash)
	if err != nil {
		return false, errors.Wrapf(err, "could not prepare history commitment request for edge: %#x", edgeId)
	}
	endHeight, endCommit := eg.EndCommitment()
	challengeLevel := eg.GetChallengeLevel()
	isHonestEdge, err := ht.histChecker.AgreesWithHistoryCommitment(
		ctx,
		challengeLevel,
		historyCommitRequest,
		l2stateprovider.History{
			Height:     uint64(endHeight),
			MerkleRoot: endCommit,
		},
	)
	if err != nil {
		if strings.Contains(err.Error(), "accumulator not found") {
			return false, errors.New("validator is still syncing the chain, will retry later")
		}
		return false, errors.Wrapf(err, "could not check history commitment agreement for edge: %#x", edgeId)
	}
	// Edges are royal if they have an honest ancestry and are also honest from our perspective.
	isRoyal = hasHonestAncestry && isHonestEdge
	if isRoyal {
		ht.keepTrackOfHonestEdge(eg)
	}
	return isRoyal, nil
}

func (ht *RoyalChallengeTree) checkAssertionHash(ctx context.Context, id protocol.EdgeId) error {
	assertionHash, err := ht.metadataReader.TopLevelAssertion(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get top level assertion for edge %#x", id)
	}
	if ht.topLevelAssertionHash != assertionHash {
		// This edge should not be part of this challenge tree.
		return ErrMismatchedChallengeAssertionHash
	}
	return nil
}

func (ht *RoyalChallengeTree) claimedAssertionHash(ctx context.Context, eg protocol.SpecEdge) (protocol.AssertionHash, error) {
	challengeLevel := eg.GetChallengeLevel()
	// If this is a root challege level zero edge.
	if challengeLevel == protocol.NewBlockChallengeLevel() && !eg.ClaimId().IsNone() {
		return protocol.AssertionHash{Hash: common.Hash(eg.ClaimId().Unwrap())}, nil
	}
	honestLevelZeroEdge, honestErr := ht.RoyalBlockChallengeRootEdge()
	if honestErr != nil {
		return protocol.AssertionHash{}, honestErr
	}
	if honestLevelZeroEdge.ClaimId().IsNone() {
		return protocol.AssertionHash{}, errors.New("honest level zero edge has no claim id")
	}
	return protocol.AssertionHash{Hash: common.Hash(honestLevelZeroEdge.ClaimId().Unwrap())}, nil
}

func (ht *RoyalChallengeTree) prepareHistoryCommitmentRequest(
	ctx context.Context,
	eg protocol.SpecEdge,
	claimedAssertionHash protocol.AssertionHash,
) (*l2stateprovider.HistoryCommitmentRequest, error) {
	// We get the batch range for the claimed assertion of the edge which is needed to compute
	// history commitments. We need to figure out from what batch to what batch the assertion
	// is claiming its data for.
	creationInfo, err := ht.metadataReader.ReadAssertionCreationInfo(ctx, claimedAssertionHash)
	if err != nil {
		return nil, err
	}
	challengeLevel := eg.GetChallengeLevel()
	fromBatch := l2stateprovider.Batch(protocol.GoGlobalStateFromSolidity(creationInfo.BeforeState.GlobalState).Batch)
	toBatch := l2stateprovider.Batch(protocol.GoGlobalStateFromSolidity(creationInfo.AfterState.GlobalState).Batch)
	endHeight, _ := eg.EndCommitment()
	heights, err := ht.metadataReader.TopLevelClaimHeights(ctx, eg.Id())
	if err != nil {
		return nil, errors.Wrapf(err, "could not get claim heights for edge %#x", eg.Id())
	}
	startHeights := make([]l2stateprovider.Height, len(heights.ChallengeOriginHeights))
	for i, h := range heights.ChallengeOriginHeights {
		startHeights[i] = l2stateprovider.Height(h)
	}
	if challengeLevel == protocol.NewBlockChallengeLevel() {
		return &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              creationInfo.WasmModuleRoot,
			FromBatch:                   fromBatch,
			ToBatch:                     toBatch,
			FromHeight:                  0,
			UpperChallengeOriginHeights: make([]l2stateprovider.Height, 0),
			UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
		}, nil
	}

	if len(startHeights) == 0 {
		return nil, errors.New("start height cannot be zero")
	}
	return &l2stateprovider.HistoryCommitmentRequest{
		WasmModuleRoot:              creationInfo.WasmModuleRoot,
		FromBatch:                   fromBatch,
		ToBatch:                     toBatch,
		FromHeight:                  l2stateprovider.Height(0),
		UpperChallengeOriginHeights: startHeights,
		UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
	}, nil
}

// Check if the edge id should be added to the rivaled edges set.
// Here we only care about edges here that are either honest or those whose start
// history commitments we agree with.
func (ht *RoyalChallengeTree) keepTrackOfCreationTime(eg protocol.SpecEdge) error {
	key := buildEdgeCreationTimeKey(eg.OriginId(), eg.MutualId())
	mutuals := ht.edgeCreationTimes.Get(key)
	if mutuals == nil {
		ht.edgeCreationTimes.Put(key, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals = ht.edgeCreationTimes.Get(key)
	}
	createdAtBlock, err := eg.CreatedAtBlock()
	if err != nil {
		return err
	}
	mutuals.Put(eg.Id(), creationTime(createdAtBlock))
	ht.edgeCreationTimes.Put(key, mutuals)
	return nil
}

// If we agree with the edge, we add it to our edges mapping and if it is level zero,
// we keep track of it specifically in our struct.
func (ht *RoyalChallengeTree) keepTrackOfHonestEdge(eg protocol.SpecEdge) {
	id := eg.Id()
	ht.edges.Put(id, eg)
	if eg.ClaimId().IsNone() {
		return
	}
	reversedChallengeLevel := eg.GetReversedChallengeLevel()
	rootEdgesAtLevel, ok := ht.royalRootEdgesByLevel.TryGet(reversedChallengeLevel)
	if !ok || rootEdgesAtLevel == nil {
		honestRootEdges := threadsafe.NewSlice[protocol.SpecEdge]()
		honestRootEdges.Push(eg)
		ht.royalRootEdgesByLevel.Put(reversedChallengeLevel, honestRootEdges)
	} else {
		rootEdgesAtLevel.Push(eg)
		ht.royalRootEdgesByLevel.Put(reversedChallengeLevel, rootEdgesAtLevel)
	}
}
