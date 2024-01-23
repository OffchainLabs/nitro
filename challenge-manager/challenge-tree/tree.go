// Package challengetree includes logic for keeping track of honest edges within a challenge
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
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/common"
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

// HonestChallengeTree keeps track of edges the honest node agrees with in a particular challenge.
// All edges tracked in this data structure are part of the same, top-level assertion challenge.
type HonestChallengeTree struct {
	edges                  *threadsafe.Map[protocol.EdgeId, protocol.SpecEdge]
	mutualIds              *threadsafe.Map[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]]
	topLevelAssertionHash  protocol.AssertionHash
	metadataReader         MetadataReader
	histChecker            l2stateprovider.HistoryChecker
	validatorName          string
	totalChallengeLevels   uint8
	honestRootEdgesByLevel *threadsafe.Map[protocol.ChallengeLevel, *threadsafe.Slice[protocol.ReadOnlyEdge]]
}

func New(
	assertionHash protocol.AssertionHash,
	metadataReader MetadataReader,
	histChecker l2stateprovider.HistoryChecker,
	numBigStepLevels uint8,
	validatorName string,
) *HonestChallengeTree {
	return &HonestChallengeTree{
		edges:                 threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds:             threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		topLevelAssertionHash: assertionHash,
		metadataReader:        metadataReader,
		histChecker:           histChecker,
		validatorName:         validatorName,
		// The total number of challenge levels include block challenges, small step challenges, and N big step challenges.
		totalChallengeLevels:   numBigStepLevels + 2,
		honestRootEdgesByLevel: threadsafe.NewMap[protocol.ChallengeLevel, *threadsafe.Slice[protocol.ReadOnlyEdge]](),
	}
}

// HonestBlockChallengeRootEdge gets the honest, root challenge block edge for the top level assertion
// being challenged.
func (ht *HonestChallengeTree) HonestBlockChallengeRootEdge() (protocol.ReadOnlyEdge, error) {
	// In our locally tracked challenge tree implementation, the
	// block challenge level is equal to the total challenge levels - 1.
	blockChallengeLevel := protocol.ChallengeLevel(ht.totalChallengeLevels) - 1
	if rootEdges, ok := ht.honestRootEdgesByLevel.TryGet(blockChallengeLevel); ok {
		if rootEdges.Len() != 1 {
			return nil, fmt.Errorf(
				"expected one honest root block challenge edge for challenged assertion %#x",
				ht.topLevelAssertionHash,
			)
		}
		return rootEdges.Get(0).Unwrap(), nil
	}
	return nil, fmt.Errorf("no honest root edges for block challenge level for assertion %#x", ht.topLevelAssertionHash)
}

var (
	ErrAlreadyBeingTracked              = errors.New("edge already being tracked")
	ErrMismatchedChallengeAssertionHash = errors.New("edge challenged assertion hash is not the expected one for the challenge")
)

// AddEdge to the honest challenge tree. Only honest edges are tracked, but we also keep track
// of rival ids in a mutual ids mapping internally for extra book-keeping.
func (ht *HonestChallengeTree) AddEdge(ctx context.Context, eg protocol.SpecEdge) (protocol.Agreement, error) {
	if _, ok := ht.edges.TryGet(eg.Id()); ok {
		return protocol.Agreement{}, ErrAlreadyBeingTracked
	}
	assertionHash, err := ht.metadataReader.TopLevelAssertion(ctx, eg.Id())
	if err != nil {
		return protocol.Agreement{}, errors.Wrapf(err, "could not get top level assertion for edge %#x", eg.Id())
	}
	if ht.topLevelAssertionHash != assertionHash {
		// This edge should not be part of this challenge tree.
		return protocol.Agreement{}, ErrMismatchedChallengeAssertionHash
	}

	var claimedAssertionHash protocol.AssertionHash
	challengeLevel := eg.GetChallengeLevel()

	// If this is a root challege level zero edge.
	if challengeLevel == protocol.NewBlockChallengeLevel() && !eg.ClaimId().IsNone() {
		claimedAssertionHash = protocol.AssertionHash{Hash: common.Hash(eg.ClaimId().Unwrap())}
	} else {
		honestLevelZeroEdge, honestErr := ht.HonestBlockChallengeRootEdge()
		if honestErr != nil {
			return protocol.Agreement{}, honestErr
		}
		if honestLevelZeroEdge.ClaimId().IsNone() {
			return protocol.Agreement{}, errors.New("honest level zero edge has no claim id")
		}
		claimedAssertionHash = protocol.AssertionHash{Hash: common.Hash(honestLevelZeroEdge.ClaimId().Unwrap())}
	}

	// We get the batch range for the claimed assertion of the edge which is needed to compute
	// history commitments. We need to figure out from what batch to what batch the assertion
	// is claiming its data for.
	creationInfo, err := ht.metadataReader.ReadAssertionCreationInfo(ctx, claimedAssertionHash)
	if err != nil {
		return protocol.Agreement{}, err
	}
	fromBatch := l2stateprovider.Batch(protocol.GoGlobalStateFromSolidity(creationInfo.BeforeState.GlobalState).Batch)
	toBatch := l2stateprovider.Batch(protocol.GoGlobalStateFromSolidity(creationInfo.AfterState.GlobalState).Batch)

	// We only track edges we fully agree with (honest edges).
	startHeight, startCommit := eg.StartCommitment()
	endHeight, endCommit := eg.EndCommitment()
	heights, err := ht.metadataReader.TopLevelClaimHeights(ctx, eg.Id())
	if err != nil {
		return protocol.Agreement{}, errors.Wrapf(err, "could not get claim heights for edge %#x", eg.Id())
	}

	startHeights := make([]l2stateprovider.Height, len(heights.ChallengeOriginHeights))
	for i, h := range heights.ChallengeOriginHeights {
		startHeights[i] = l2stateprovider.Height(h)
	}

	var isHonestEdge bool
	var agreesWithStart bool
	if challengeLevel == protocol.NewBlockChallengeLevel() {
		request := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              creationInfo.WasmModuleRoot,
			FromBatch:                   fromBatch,
			ToBatch:                     toBatch,
			FromHeight:                  0,
			UpperChallengeOriginHeights: make([]l2stateprovider.Height, 0),
			UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
		}
		isHonestEdge, err = ht.histChecker.AgreesWithHistoryCommitment(
			ctx,
			challengeLevel,
			request,
			l2stateprovider.History{
				Height:     uint64(endHeight),
				MerkleRoot: endCommit,
			},
		)
		if err != nil {
			return protocol.Agreement{}, errors.Wrapf(err, "could not check if agrees with history commit for edge %#x", eg.Id())
		}
		agreesWithStart, err = ht.histChecker.AgreesWithHistoryCommitment(
			ctx,
			challengeLevel,
			request,
			l2stateprovider.History{
				Height:     uint64(startHeight),
				MerkleRoot: startCommit,
			},
		)
		if err != nil {
			return protocol.Agreement{}, errors.Wrapf(err, "could not check if agrees with history commit for edge %#x", eg.Id())
		}
	} else {
		if len(startHeights) == 0 {
			return protocol.Agreement{}, errors.New("start height cannot be zero")
		}
		request := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              creationInfo.WasmModuleRoot,
			FromBatch:                   fromBatch,
			ToBatch:                     toBatch,
			FromHeight:                  l2stateprovider.Height(0),
			UpperChallengeOriginHeights: startHeights,
			UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
		}
		isHonestEdge, err = ht.histChecker.AgreesWithHistoryCommitment(
			ctx,
			challengeLevel,
			request,
			l2stateprovider.History{
				Height:     uint64(endHeight),
				MerkleRoot: endCommit,
			},
		)
		if err != nil {
			return protocol.Agreement{}, errors.Wrapf(err, "could not check if agrees with history commit for edge %#x", eg.Id())
		}
		agreesWithStart, err = ht.histChecker.AgreesWithHistoryCommitment(
			ctx,
			challengeLevel,
			request,
			l2stateprovider.History{
				Height:     uint64(startHeight),
				MerkleRoot: startCommit,
			},
		)
		if err != nil {
			return protocol.Agreement{}, errors.Wrapf(err, "could not check if agrees with history commit for edge %#x", eg.Id())
		}
	}
	// If we agree with the edge, we add it to our edges mapping and if it is level zero,
	// we keep track of it specifically in our struct.
	if isHonestEdge {
		id := eg.Id()
		ht.edges.Put(id, eg)
		if !eg.ClaimId().IsNone() {
			reversedChallengeLevel := eg.GetReversedChallengeLevel()
			rootEdgesAtLevel, ok := ht.honestRootEdgesByLevel.TryGet(reversedChallengeLevel)
			if !ok || rootEdgesAtLevel == nil {
				honestRootEdges := threadsafe.NewSlice[protocol.ReadOnlyEdge]()
				honestRootEdges.Push(eg)
				ht.honestRootEdgesByLevel.Put(reversedChallengeLevel, honestRootEdges)
			} else {
				rootEdgesAtLevel.Push(eg)
				ht.honestRootEdgesByLevel.Put(reversedChallengeLevel, rootEdgesAtLevel)
			}
		}
	}

	// Check if the edge id should be added to the rivaled edges set.
	// Here we only care about edges here that are either honest or those whose start
	// history commitments we agree with.
	if agreesWithStart || isHonestEdge {
		mutualId := eg.MutualId()
		mutuals := ht.mutualIds.Get(mutualId)
		if mutuals == nil {
			ht.mutualIds.Put(mutualId, threadsafe.NewMap[protocol.EdgeId, creationTime]())
			mutuals = ht.mutualIds.Get(mutualId)
		}
		createdAtBlock, err := eg.CreatedAtBlock()
		if err != nil {
			return protocol.Agreement{}, err
		}
		mutuals.Put(eg.Id(), creationTime(createdAtBlock))
	}
	return protocol.Agreement{
		IsHonestEdge:          isHonestEdge,
		AgreesWithStartCommit: agreesWithStart,
	}, nil
}

func (ht *HonestChallengeTree) HasRoyalEdge(edgeId protocol.EdgeId) bool {
	return ht.edges.Has(edgeId)
}

// AddHonestEdge known to be honest, such as those created by the local validator.
func (ht *HonestChallengeTree) AddHonestEdge(eg protocol.VerifiedHonestEdge) error {
	id := eg.Id()
	if _, ok := ht.edges.TryGet(id); ok {
		// Already being tracked.
		return nil
	}
	ht.edges.Put(id, eg)
	// If the edge has a claim id, it means it is a level zero edge and we keep track of it.
	if !eg.ClaimId().IsNone() {
		reversedChallengeLevel := eg.GetReversedChallengeLevel()
		rootEdgesAtLevel, ok := ht.honestRootEdgesByLevel.TryGet(reversedChallengeLevel)
		if !ok || rootEdgesAtLevel == nil {
			honestRootEdges := threadsafe.NewSlice[protocol.ReadOnlyEdge]()
			honestRootEdges.Push(eg)
			ht.honestRootEdgesByLevel.Put(reversedChallengeLevel, honestRootEdges)
		} else {
			rootEdgesAtLevel.Push(eg)
			ht.honestRootEdgesByLevel.Put(reversedChallengeLevel, rootEdgesAtLevel)
		}
	}
	// We add the edge id to the list of mutual ids we are tracking.
	mutualId := eg.MutualId()
	mutuals := ht.mutualIds.Get(mutualId)
	if mutuals == nil {
		ht.mutualIds.Put(mutualId, threadsafe.NewMap[protocol.EdgeId, creationTime]())
		mutuals = ht.mutualIds.Get(mutualId)
	}
	createdAtBlock, err := eg.CreatedAtBlock()
	if err != nil {
		return err
	}
	mutuals.Put(eg.Id(), creationTime(createdAtBlock))
	return nil
}

func (ht *HonestChallengeTree) GetEdges() *threadsafe.Map[protocol.EdgeId, protocol.SpecEdge] {
	return ht.edges
}
