// Package challengetree includes logic for keeping track of honest edges within a challenge
// with utilities for computing cumulative path timers for said edges. This is helpful during
// the confirmation process needed by edge trackers.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package challengetree

import (
	"context"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
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

// HonestChallengeTree keeps track of edges the honest node agrees with in a particular challenge.
// All edges tracked in this data structure are part of the same, top-level assertion challenge.
type HonestChallengeTree struct {
	edges                         *threadsafe.Map[protocol.EdgeId, protocol.SpecEdge]
	mutualIds                     *threadsafe.Map[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]]
	topLevelAssertionHash         protocol.AssertionHash
	honestBlockChalLevelZeroEdge  option.Option[protocol.ReadOnlyEdge]
	honestBigStepLevelZeroEdges   *threadsafe.Slice[protocol.ReadOnlyEdge]
	honestSmallStepLevelZeroEdges *threadsafe.Slice[protocol.ReadOnlyEdge]
	metadataReader                MetadataReader
	histChecker                   l2stateprovider.HistoryChecker
	validatorName                 string
	totalChallengeLevels          uint8
	honestRootEdgesByLevel        *threadsafe.Map[protocol.ChallengeLevel, *threadsafe.Slice[protocol.ReadOnlyEdge]]
}

func New(
	assertionHash protocol.AssertionHash,
	metadataReader MetadataReader,
	histChecker l2stateprovider.HistoryChecker,
	validatorName string,
) *HonestChallengeTree {
	return &HonestChallengeTree{
		edges:                         threadsafe.NewMap[protocol.EdgeId, protocol.SpecEdge](),
		mutualIds:                     threadsafe.NewMap[protocol.MutualId, *threadsafe.Map[protocol.EdgeId, creationTime]](),
		topLevelAssertionHash:         assertionHash,
		honestBlockChalLevelZeroEdge:  option.None[protocol.ReadOnlyEdge](),
		honestBigStepLevelZeroEdges:   threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		honestSmallStepLevelZeroEdges: threadsafe.NewSlice[protocol.ReadOnlyEdge](),
		metadataReader:                metadataReader,
		histChecker:                   histChecker,
		validatorName:                 validatorName,
		totalChallengeLevels:          3,
		honestRootEdgesByLevel:        threadsafe.NewMap[protocol.ChallengeLevel, *threadsafe.Slice[protocol.ReadOnlyEdge]](),
	}
}

// AddEdge to the honest challenge tree. Only honest edges are tracked, but we also keep track
// of rival ids in a mutual ids mapping internally for extra book-keeping.
func (ht *HonestChallengeTree) AddEdge(ctx context.Context, eg protocol.SpecEdge) (protocol.Agreement, error) {
	if _, ok := ht.edges.TryGet(eg.Id()); ok {
		// Already being tracked.
		return protocol.Agreement{}, nil
	}
	assertionHash, err := ht.metadataReader.TopLevelAssertion(ctx, eg.Id())
	if err != nil {
		return protocol.Agreement{}, errors.Wrapf(err, "could not get top level assertion for edge %#x", eg.Id())
	}
	if ht.topLevelAssertionHash != assertionHash {
		// Do nothing - this edge should not be part of this challenge tree.
		return protocol.Agreement{}, nil
	}
	creationInfo, err := ht.metadataReader.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return protocol.Agreement{}, err
	}
	if !creationInfo.InboxMaxCount.IsUint64() {
		return protocol.Agreement{}, errors.New("inbox max count was not a uint64")
	}

	// We only track edges we fully agree with (honest edges).
	startHeight, startCommit := eg.StartCommitment()
	endHeight, endCommit := eg.EndCommitment()
	heights, err := ht.metadataReader.TopLevelClaimHeights(ctx, eg.Id())
	if err != nil {
		return protocol.Agreement{}, errors.Wrapf(err, "could not get claim heights for edge %#x", eg.Id())
	}
	parentAssertionInfo, err := ht.metadataReader.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: creationInfo.ParentAssertionHash})
	if err != nil {
		return protocol.Agreement{}, err
	}
	parentAssertionAfterState := protocol.GoExecutionStateFromSolidity(parentAssertionInfo.AfterState)

	challengeLevel, err := eg.GetChallengeLevel()
	if err != nil {
		return protocol.Agreement{}, err
	}

	startHeights := make([]l2stateprovider.Height, len(heights.ChallengeOriginHeights))
	for i, h := range heights.ChallengeOriginHeights {
		startHeights[i] = l2stateprovider.Height(h)
	}

	isHonestEdge, err := ht.histChecker.AgreesWithHistoryCommitment(
		ctx,
		challengeLevel,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              creationInfo.WasmModuleRoot,
			Batch:                       l2stateprovider.Batch(creationInfo.InboxMaxCount.Uint64()),
			FromHeight:                  l2stateprovider.Height(parentAssertionAfterState.GlobalState.Batch),
			UpperChallengeOriginHeights: startHeights,
		},
		l2stateprovider.History{
			Height:     uint64(endHeight),
			MerkleRoot: endCommit,
		},
	)
	if err != nil {
		return protocol.Agreement{}, errors.Wrapf(err, "could not check if agrees with history commit for edge %#x", eg.Id())
	}
	agreesWithStart, err := ht.histChecker.AgreesWithHistoryCommitment(
		ctx,
		challengeLevel,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              creationInfo.WasmModuleRoot,
			Batch:                       l2stateprovider.Batch(creationInfo.InboxMaxCount.Uint64()),
			FromHeight:                  l2stateprovider.Height(parentAssertionAfterState.GlobalState.Batch),
			UpperChallengeOriginHeights: startHeights,
		},
		l2stateprovider.History{
			Height:     uint64(startHeight),
			MerkleRoot: startCommit,
		},
	)
	if err != nil {
		return protocol.Agreement{}, errors.Wrapf(err, "could not check if agrees with history commit for edge %#x", eg.Id())
	}

	// If we agree with the edge, we add it to our edges mapping and if it is level zero,
	// we keep track of it specifically in our struct.
	if isHonestEdge {
		id := eg.Id()
		ht.edges.Put(id, eg)
		if !eg.ClaimId().IsNone() {
			switch challengeLevel {
			case protocol.NewBlockChallengeLevel():
				ht.honestBlockChalLevelZeroEdge = option.Some(protocol.ReadOnlyEdge(eg))
			default:
				reversedChallengeLevel, err := eg.GetReversedChallengeLevel()
				if err != nil {
					return protocol.Agreement{}, err
				}
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
			level, err := eg.GetChallengeLevel()
			if err != nil {
				return protocol.Agreement{}, err
			}
			rootEdges, ok := ht.honestRootEdgesByLevel.TryGet(level)
			if !ok {
				ht.honestRootEdgesByLevel.Put(level, threadsafe.NewSlice[protocol.ReadOnlyEdge]())
			} else {
				rootEdges.Push(eg)
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
		challengeLevel, err := eg.GetChallengeLevel()
		if err != nil {
			return err
		}
		switch challengeLevel {
		case protocol.NewBlockChallengeLevel():
			ht.honestBlockChalLevelZeroEdge = option.Some(protocol.ReadOnlyEdge(eg))
		default:
			reversedChallengeLevel, err := eg.GetReversedChallengeLevel()
			if err != nil {
				return err
			}
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
