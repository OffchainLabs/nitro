// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package watcher implements the main monitoring logic for protocol validators.
// The challenge watcher is a singleton service available to all spawned edge
// trackers and it tracks common information such as the edges' ancestors and an
// edge's time unrivaled.
//
// See: [github.com/offchainlabs/bold/challenge-manager/edge-tracker]
package watcher

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/bold/api"
	"github.com/offchainlabs/bold/api/db"
	protocol "github.com/offchainlabs/bold/chain-abstraction"
	solimpl "github.com/offchainlabs/bold/chain-abstraction/sol-implementation"
	challengetree "github.com/offchainlabs/bold/challenge-manager/challenge-tree"
	"github.com/offchainlabs/bold/containers/option"
	"github.com/offchainlabs/bold/containers/threadsafe"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/logs/ephemeral"
	retry "github.com/offchainlabs/bold/runtime"
	"github.com/offchainlabs/bold/util/stopwaiter"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
)

var (
	edgeAddedCounter                        = metrics.NewRegisteredCounter("arb/validator/watcher/edge_added", nil)
	edgeConfirmedByTimeCounter              = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_time", nil)
	edgeConfirmedByOSPCounter               = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_osp", nil)
	errorConfirmingAssertionByWinnerCounter = metrics.NewRegisteredCounter("arb/validator/watcher/error_confirming_assertion_by_winner", nil)
	assertionConfirmedCounter               = metrics.GetOrRegisterCounter("arb/validator/scanner/assertion_confirmed", nil)
)

// EdgeManager provides a method to track edges, via edge tracker goroutines.
type EdgeManager interface {
	TrackEdge(ctx context.Context, edge protocol.VerifiedRoyalEdge) error
}

// Represents a set of honest edges being tracked in a top-level challenge and
// all the associated subchallenge honest edges along with some more metadata
// used for computing information needed for confirmations. Each time an edge is
// created onchain, the challenge watcher service will add it to its respective
// "trackedChallenge" namespaced under the top-level assertion hash the edge
// belongs to.
type trackedChallenge struct {
	honestEdgeTree                 *challengetree.RoyalChallengeTree
	confirmedLevelZeroEdgeClaimIds *threadsafe.Map[protocol.ClaimId, protocol.EdgeId]
}

// The Watcher implements a service in the validator runtime that is in charge
// of scanning through all edge creation events via a polling mechanism. It will
// keep track of edges the validator's state provider agrees with within
// trackedChallenge instances. The challenge watcher provides two useful
// methods: (a) the ability to compute the honest path timer of an edge, and (b)
// the ability to check if an edge with a certain claim id has been confirmed.
// Both are used during the confirmation process in edge tracker goroutines.
type Watcher struct {
	stopwaiter.StopWaiter
	histChecker                 l2stateprovider.HistoryChecker
	chain                       protocol.AssertionChain
	edgeManager                 EdgeManager
	pollEventsInterval          time.Duration
	challenges                  *threadsafe.Map[protocol.AssertionHash, *trackedChallenge]
	backend                     protocol.ChainBackend
	validatorName               string
	numBigStepLevels            uint8
	initialSyncCompleted        atomic.Bool
	apiDB                       db.Database
	assertionConfirmingInterval time.Duration
	averageTimeForBlockCreation time.Duration
	evilEdgesByLevel            *threadsafe.Map[protocol.ChallengeLevel, *threadsafe.Set[protocol.EdgeId]]
	// Only track challenges for these parent assertion hashes.
	// Track all if empty / nil.
	trackChallengeParentAssertionHashes []protocol.AssertionHash
	maxGetLogBlocks                     uint64
}

// New initializes a watcher service for frequently scanning the chain
// for edge creations and confirmations.
func New(
	chain protocol.AssertionChain,
	histChecker l2stateprovider.HistoryChecker,
	validatorName string,
	apiDB db.Database,
	assertionConfirmingInterval time.Duration,
	averageTimeForBlockCreation time.Duration,
	trackChallengeParentAssertionHashes []protocol.AssertionHash,
	maxGetLogBlocks uint64,
) (*Watcher, error) {
	return &Watcher{
		chain:                               chain,
		edgeManager:                         nil, // Must be set after construction.
		pollEventsInterval:                  time.Millisecond * 500,
		challenges:                          threadsafe.NewMap(threadsafe.MapWithMetric[protocol.AssertionHash, *trackedChallenge]("challenges")),
		backend:                             chain.Backend(),
		histChecker:                         histChecker,
		numBigStepLevels:                    chain.SpecChallengeManager().NumBigSteps(),
		validatorName:                       validatorName,
		apiDB:                               apiDB,
		assertionConfirmingInterval:         assertionConfirmingInterval,
		averageTimeForBlockCreation:         averageTimeForBlockCreation,
		evilEdgesByLevel:                    threadsafe.NewMap(threadsafe.MapWithMetric[protocol.ChallengeLevel, *threadsafe.Set[protocol.EdgeId]]("evilEdgesByLevel")),
		trackChallengeParentAssertionHashes: trackChallengeParentAssertionHashes,
		maxGetLogBlocks:                     maxGetLogBlocks,
	}, nil
}

// SetEdgeManager sets the EdgeManager that will track the royal edges.
func (w *Watcher) SetEdgeManager(em EdgeManager) {
	w.edgeManager = em
}

// AvgBlockTime returns the average time for block creation.
func (w *Watcher) AvgBlockTime() time.Duration {
	return w.averageTimeForBlockCreation
}

// HonestBlockChallengeRootEdge gets the honest block challenge root edge for a
// given challenge by challenged assertion id if it exists.
func (w *Watcher) HonestBlockChallengeRootEdge(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
) (protocol.ReadOnlyEdge, error) {
	chal, ok := w.challenges.TryGet(assertionHash)
	if !ok {
		return nil, fmt.Errorf("no challenge for assertion hash %#x", assertionHash)
	}
	return chal.honestEdgeTree.RoyalBlockChallengeRootEdge()
}

// ConfirmedEdgeWithClaimExists checks if a confirmed, level zero edge exists
// that claims a particular edge id for a tracked challenge. This is used during
// the confirmation process of edges within edge tracker goroutines. Returns the
// claiming edge id.
func (w *Watcher) ConfirmedEdgeWithClaimExists(
	topLevelAssertionHash protocol.AssertionHash,
	claimId protocol.ClaimId,
) (protocol.EdgeId, bool) {
	challenge, ok := w.challenges.TryGet(topLevelAssertionHash)
	if !ok {
		return protocol.EdgeId{}, false
	}
	return challenge.confirmedLevelZeroEdgeClaimIds.TryGet(claimId)
}

func (w *Watcher) IsRoyal(assertionHash protocol.AssertionHash, edgeId protocol.EdgeId) bool {
	chal, ok := w.challenges.TryGet(assertionHash)
	if !ok {
		return false
	}
	return chal.honestEdgeTree.HasRoyalEdge(edgeId)
}

func (w *Watcher) InheritedTimerForEdge(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (protocol.InheritedTimer, error) {
	chalManager := w.chain.SpecChallengeManager()
	edgeOpt, err := chalManager.GetEdge(ctx, edgeId)
	if err != nil {
		return 0, err
	}
	if edgeOpt.IsNone() {
		return 0, fmt.Errorf("no edge found with id %#x", edgeId.Hash)

	}
	return edgeOpt.Unwrap().LatestInheritedTimer(ctx)
}

func (w *Watcher) IsSynced() bool {
	return w.initialSyncCompleted.Load()
}

// Start watching the chain via a polling mechanism for all edge added and
// confirmation events in order to process some of this data into internal
// representations for confirmation purposes.
func (w *Watcher) Start(ctx context.Context) {
	w.StopWaiter.Start(ctx, w)
	scanRange, err := retry.UntilSucceeds(ctx, func() (filterRange, error) {
		return w.getStartEndBlockNum(ctx)
	})
	if err != nil {
		log.Error("Could not get start and end block num", "err", err)
		return
	}
	fromBlock := scanRange.startBlockNum
	toBlock := scanRange.endBlockNum

	// Get a challenge manager instance and filterer.
	challengeManager := w.chain.SpecChallengeManager()
	filterer, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerFilterer, error) {
		return challengeV2gen.NewEdgeChallengeManagerFilterer(challengeManager.Address(), w.backend)
	})
	if err != nil {
		log.Error("Could not initialize edge challenge manager filterer", "err", err)
		return
	}
	for startBlock := fromBlock; startBlock <= toBlock; startBlock = startBlock + w.maxGetLogBlocks {
		endBlock := startBlock + w.maxGetLogBlocks
		if endBlock > toBlock {
			endBlock = toBlock
		}
		filterOpts := &bind.FilterOpts{
			Start:   startBlock,
			End:     &endBlock,
			Context: ctx,
		}

		// Checks for different events right away before we start polling.
		_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			return true, w.checkForEdgeAdded(ctx, filterer, filterOpts)
		})
		if err != nil {
			log.Error("Could not check for edge added", "err", err)
			return
		}
		_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			return true, w.checkForEdgeConfirmedByOneStepProof(ctx, filterer, filterOpts)
		})
		if err != nil {
			log.Error("Could not check for edge confirmed by osp", "err", err)
			return
		}
		_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			return true, w.checkForEdgeConfirmedByTime(ctx, filterer, filterOpts)
		})
		if err != nil {
			log.Error("Could not check for edge confirmed by time", "err", err)
			return
		}
	}

	fromBlock = toBlock
	ticker := time.NewTicker(w.pollEventsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			toBlock, err := w.chain.DesiredHeaderU64(ctx)
			if err != nil {
				log.Error("Could not get latest header", "err", err)
				continue
			}
			// AssertionChain's rpcHeadBlockNumber is set to finalized and this might occur due to l1 backends of load balancer
			// not being in consensus wrt finalized. In which case we ignore and continue
			if fromBlock > toBlock {
				continue
			}
			if fromBlock == toBlock {
				w.initialSyncCompleted.Store(true)
				continue
			}
			// Get a challenge manager instance and filterer.
			challengeManager := w.chain.SpecChallengeManager()
			filterer, err = retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerFilterer, error) {
				return challengeV2gen.NewEdgeChallengeManagerFilterer(challengeManager.Address(), w.backend)
			})
			if err != nil {
				log.Error("Could not get challenge manager filterer", "err", err)
				return
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			if err = w.checkForEdgeAdded(ctx, filterer, filterOpts); err != nil {
				log.Error("Could not check for edge added", "err", err)
				continue
			}
			if err = w.checkForEdgeConfirmedByOneStepProof(ctx, filterer, filterOpts); err != nil {
				log.Error("Could not check for edge confirmed by osp", "err", err)
				continue
			}
			if err = w.checkForEdgeConfirmedByTime(ctx, filterer, filterOpts); err != nil {
				log.Error("Could not check for edge confirmed by time", "err", err)
				continue
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

// GetRoyalEdges returns all royal, tracked edges in the watcher by assertion
// hash.
func (w *Watcher) GetRoyalEdges(ctx context.Context) (map[protocol.AssertionHash][]*api.JsonTrackedRoyalEdge, error) {
	l1BlockNum, err := w.chain.DesiredL1HeaderU64(ctx)
	if err != nil {
		return nil, err
	}
	response := make(map[protocol.AssertionHash][]*api.JsonTrackedRoyalEdge)
	if err = w.challenges.ForEach(func(assertionHash protocol.AssertionHash, t *trackedChallenge) error {
		return t.honestEdgeTree.GetEdges().ForEach(func(edgeId protocol.EdgeId, edge protocol.SpecEdge) error {
			start, startRoot := edge.StartCommitment()
			end, endRoot := edge.EndCommitment()
			createdAt, err2 := edge.CreatedAtBlock()
			if err2 != nil {
				return err2
			}
			unrivaled, err2 := t.honestEdgeTree.IsUnrivaledAtBlockNum(edge, l1BlockNum)
			if err2 != nil {
				return err2
			}
			hasRival := !unrivaled
			timeUnrivaled, err2 := t.honestEdgeTree.TimeUnrivaled(ctx, edge, l1BlockNum)
			if err2 != nil {
				return err2
			}
			var miniStaker common.Address
			if edge.MiniStaker().IsSome() {
				miniStaker = edge.MiniStaker().Unwrap()
			}
			var claimId common.Hash
			if edge.ClaimId().IsSome() {
				claimId = common.Hash(edge.ClaimId().Unwrap())
			}
			response[assertionHash] = append(
				response[assertionHash],
				&api.JsonTrackedRoyalEdge{
					Id:               edgeId.Hash,
					ChallengeLevel:   uint8(edge.GetChallengeLevel()),
					StartHistoryRoot: startRoot,
					StartHeight:      uint64(start),
					EndHeight:        uint64(end),
					EndHistoryRoot:   endRoot,
					CreatedAtBlock:   createdAt,
					MutualId:         common.Hash(edge.MutualId()),
					OriginId:         common.Hash(edge.OriginId()),
					ClaimId:          claimId,
					HasRival:         hasRival,
					TimeUnrivaled:    timeUnrivaled,
					MiniStaker:       miniStaker,
				},
			)
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return response, nil
}

func (w *Watcher) BlockChallengeRootEdge(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
) (protocol.SpecEdge, error) {
	chal, ok := w.challenges.TryGet(challengedAssertionHash)
	if !ok {
		return nil, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			challengedAssertionHash,
		)
	}
	return chal.honestEdgeTree.BlockChallengeRootEdge(ctx)
}

func (w *Watcher) LowerMostRoyalEdges(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
) ([]protocol.SpecEdge, error) {
	chal, ok := w.challenges.TryGet(challengedAssertionHash)
	if !ok {
		return nil, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			challengedAssertionHash,
		)
	}
	return chal.honestEdgeTree.GetAllRoyalLeaves(ctx)
}

func (w *Watcher) ComputeAncestors(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
	edgeId protocol.EdgeId,
) ([]protocol.ReadOnlyEdge, error) {
	chal, ok := w.challenges.TryGet(challengedAssertionHash)
	if !ok {
		return nil, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			challengedAssertionHash,
		)
	}
	l1BlockHeaderNumber, err := w.chain.DesiredL1HeaderU64(ctx)
	if err != nil {
		return nil, err
	}
	return chal.honestEdgeTree.ComputeAncestors(ctx, edgeId, l1BlockHeaderNumber)
}

func (w *Watcher) ClosestEssentialAncestor(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
	edge protocol.VerifiedRoyalEdge,
) (protocol.ReadOnlyEdge, error) {
	chal, ok := w.challenges.TryGet(challengedAssertionHash)
	if !ok {
		return nil, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			challengedAssertionHash,
		)
	}
	return chal.honestEdgeTree.ClosestEssentialAncestor(ctx, edge)
}

func (w *Watcher) IsEssentialAncestorConfirmable(
	ctx context.Context,
	edge protocol.SpecEdge,
	challengedAssertionHash protocol.AssertionHash,
	confirmationThreshold uint64,
) (bool, error) {
	chal, ok := w.challenges.TryGet(challengedAssertionHash)
	if !ok {
		return false, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			challengedAssertionHash,
		)
	}
	blockL1HeaderNumber, err := w.chain.DesiredL1HeaderU64(ctx)
	if err != nil {
		return false, err
	}
	if !chal.honestEdgeTree.HasRoyalEdge(edge.Id()) {
		return false, fmt.Errorf("edge with id %#x is not yet tracked locally", edge.Id().Hash)
	}
	essentialAncestor, err := chal.honestEdgeTree.ClosestEssentialAncestor(ctx, edge)
	if err != nil {
		return false, err
	}
	pathWeight, err := chal.honestEdgeTree.ComputePathWeight(ctx, challengetree.ComputePathWeightArgs{
		Child:    edge.Id(),
		Ancestor: essentialAncestor.Id(),
		BlockNum: blockL1HeaderNumber,
	})
	if err != nil {
		return false, err
	}
	return pathWeight >= confirmationThreshold, nil
}

func (w *Watcher) IsConfirmableEssentialEdge(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
	essentialEdgeId protocol.EdgeId,
	confirmationThreshold uint64,
) (bool, []challengetree.EssentialPath, uint64, error) {
	chal, ok := w.challenges.TryGet(challengedAssertionHash)
	if !ok {
		return false, nil, 0, fmt.Errorf("could not get challenge for top level assertion %#x", challengedAssertionHash)
	}
	blockL1HeaderNumber, err := w.chain.DesiredL1HeaderU64(ctx)
	if err != nil {
		return false, nil, 0, err
	}
	confirmable, essentialPaths, timer, err := chal.honestEdgeTree.IsConfirmableEssentialEdge(
		ctx,
		challengetree.IsConfirmableArgs{
			EssentialEdge:         essentialEdgeId,
			BlockNum:              blockL1HeaderNumber,
			ConfirmationThreshold: confirmationThreshold,
		},
	)
	return confirmable, essentialPaths, timer, err
}

func (w *Watcher) AllowTrackingEdgeWithParentHash(parentHash protocol.AssertionHash) bool {
	if len(w.trackChallengeParentAssertionHashes) == 0 {
		return true
	}
	for _, hash := range w.trackChallengeParentAssertionHashes {
		if hash == parentHash {
			return true
		}
	}
	return false
}

// AddVerifiedHonestEdge adds an edge known to be honest to the chain watcher's
// internally tracked challenge trees and spawns an edge tracker for it. Should
// be called after the challenge manager creates a new edge, or bisects an edge
// and produces two children from that move.
func (w *Watcher) AddVerifiedHonestEdge(ctx context.Context, edge protocol.VerifiedRoyalEdge) error {
	assertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return err
	}
	// If a challenge is not yet being tracked locally by the watcher for the
	// edge's assertion hash, it adds an entry to the map.
	chal, ok := w.challenges.TryGet(assertionHash)
	if !ok {
		tree := challengetree.New(
			assertionHash,
			w.chain,
			w.histChecker,
			w.numBigStepLevels,
			w.validatorName,
		)
		chal = &trackedChallenge{
			honestEdgeTree:                 tree,
			confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap(threadsafe.MapWithMetric[protocol.ClaimId, protocol.EdgeId]("confirmedLevelZeroEdgeClaimIds")),
		}
		w.challenges.Put(assertionHash, chal)
	}
	// Add the edge to a local challenge tree of honest edges and, if needed, we
	// also spawn a tracker for the edge.
	start, startRoot := edge.StartCommitment()
	end, endRoot := edge.EndCommitment()
	fields := []any{
		"edgeId", fmt.Sprintf("%#x", edge.Id().Bytes()[:4]),
		"challengeLevel", edge.GetChallengeLevel(),
		"challengedAssertionHash", fmt.Sprintf("%#x", assertionHash.Bytes()[:4]),
		"startHeight", start,
		"endHeight", end,
		"startCommit", fmt.Sprintf("%#x", startRoot[:4]),
		"endCommit", fmt.Sprintf("%#x", endRoot[:4]),
		"validatorName", w.validatorName,
		"isHonestEdge", true,
	}
	log.Info("Observed honest edge", fields...)
	if err = chal.honestEdgeTree.AddRoyalEdge(edge); err != nil {
		log.Error("Could not add verified honest edge to local cache", "err", err)
		return errors.Wrap(err, "could not add honest edge to challenge tree")
	}
	go func() {
		if _, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			if innerErr := w.saveEdgeToDB(ctx, edge, true /* is royal */); innerErr != nil {
				log.Error("Could not save edge to db", "err", innerErr)
				return false, innerErr
			}
			return false, nil
		}); err != nil {
			log.Error("Could not save edge to db", "err", err)
		}
	}()
	return nil
}

// Filters for all edge added events within a range and processes them.
func (w *Watcher) checkForEdgeAdded(
	ctx context.Context,
	filterer *challengeV2gen.EdgeChallengeManagerFilterer,
	filterOpts *bind.FilterOpts,
) error {
	it, err := filterer.FilterEdgeAdded(filterOpts, nil, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err = it.Close(); err != nil {
			log.Error("Could not close filter iterator", "err", err)
		}
	}()
	for it.Next() {
		if it.Error() != nil {
			return errors.Wrapf(
				err,
				"got iterator error when scanning edge creations from block %d to %d",
				filterOpts.Start,
				*filterOpts.End,
			)
		}
		edgeAdded, processErr := retry.UntilSucceeds(ctx, func() (bool, error) {
			return w.processEdgeAddedEvent(ctx, it.Event)
		})
		if processErr != nil {
			return processErr
		}
		if edgeAdded {
			edgeAddedCounter.Inc(1)
		}
	}
	return nil
}

// AddEdge to watcher. If it is honest, it will be tracked.
func (w *Watcher) AddEdge(ctx context.Context, edge protocol.SpecEdge) (bool, error) {
	challengeParentAssertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return false, err
	}
	start, startRoot := edge.StartCommitment()
	end, endRoot := edge.EndCommitment()
	chal, ok := w.challenges.TryGet(challengeParentAssertionHash)
	if !ok {
		tree := challengetree.New(
			challengeParentAssertionHash,
			w.chain,
			w.histChecker,
			w.numBigStepLevels,
			w.validatorName,
		)
		chal = &trackedChallenge{
			honestEdgeTree:                 tree,
			confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap(threadsafe.MapWithMetric[protocol.ClaimId, protocol.EdgeId]("confirmedLevelZeroEdgeClaimIds")),
		}
		w.challenges.Put(challengeParentAssertionHash, chal)
	}
	// Add the edge to a local challenge tree of tracked edges. If it is honest,
	// we also spawn a tracker for the edge.
	if err = chal.honestEdgeTree.AddEdge(ctx, edge); err != nil {
		if !errors.Is(err, challengetree.ErrAlreadyBeingTracked) {
			return false, errors.Wrap(err, "could not add edge to challenge tree")
		}
		// If the error is that we are already tracking the edge, we exit early.
		return false, nil
	}
	royalEdge, isRoyal := edge.AsVerifiedHonest()
	if isRoyal {
		err = w.edgeManager.TrackEdge(ctx, royalEdge)
		if err != nil {
			return false, err
		}
	}
	fields := []any{
		"edgeId", fmt.Sprintf("%#x", edge.Id().Bytes()[:4]),
		"challengeLevel", edge.GetChallengeLevel(),
		"challengedAssertionHash", fmt.Sprintf("%#x", challengeParentAssertionHash.Bytes()[:4]),
		"startHeight", start,
		"endHeight", end,
		"startCommit", fmt.Sprintf("%#x", startRoot[:4]),
		"endCommit", fmt.Sprintf("%#x", endRoot[:4]),
		"isHonestEdge", isRoyal,
		"validatorName", w.validatorName,
	}
	if isRoyal {
		log.Info("Observed honest edge", fields...)
	} else {
		if edge.ClaimId().IsSome() {
			evilEdges, ok := w.evilEdgesByLevel.TryGet(edge.GetChallengeLevel())
			if !ok {
				evilEdges = threadsafe.NewSet(threadsafe.SetWithMetric[protocol.EdgeId]("evilEdges"))
				w.evilEdgesByLevel.Put(edge.GetChallengeLevel(), evilEdges)
			}
			if evilEdges.NumItems() < 5 {
				evilEdges.Insert(edge.Id())
			}
			if evilEdges.NumItems() >= 5 {
				log.Warn("High number of evil edges observed", "numEvilEdges", evilEdges.NumItems(), "challengeLevel", edge.GetChallengeLevel())
				metrics.GetOrRegisterCounter("arb/validator/watcher/high_num_evil_edges_at_level_"+fmt.Sprint(edge.GetChallengeLevel()), nil).Inc(1)
			}
		}
		log.Info("Observed evil edge", fields...)
	}
	go func() {
		if _, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			if innerErr := w.saveEdgeToDB(ctx, edge, isRoyal); innerErr != nil {
				log.Error("Could not save edge to db", "err", innerErr)
				return false, innerErr
			}
			return false, nil
		}); err != nil {
			log.Error("Could not save edge to db", "err", err)
		}
	}()
	return true, nil
}

// Processes an edge added event by adding it to the honest challenge tree if it
// is honest.
func (w *Watcher) processEdgeAddedEvent(
	ctx context.Context,
	event *challengeV2gen.EdgeChallengeManagerEdgeAdded,
) (bool, error) {
	challengeManager := w.chain.SpecChallengeManager()
	edgeOpt, err := challengeManager.GetEdge(ctx, protocol.EdgeId{Hash: event.EdgeId})
	if err != nil {
		return false, err
	}
	if edgeOpt.IsNone() {
		return false, fmt.Errorf("no edge found with id %#x", event.EdgeId)
	}
	edge := edgeOpt.Unwrap()
	challengeParentAssertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return false, err
	}
	if !w.allowTrackingEdgeWithChallengeParentAssertionHash(challengeParentAssertionHash) {
		return false, nil
	}
	return w.AddEdge(ctx, edgeOpt.Unwrap())
}

func (w *Watcher) allowTrackingEdgeWithChallengeParentAssertionHash(challengeParentAssertionHash protocol.AssertionHash) bool {
	if len(w.trackChallengeParentAssertionHashes) == 0 {
		return true
	}
	for _, hash := range w.trackChallengeParentAssertionHashes {
		if hash == challengeParentAssertionHash {
			return true
		}
	}
	return false
}

// Filters for edge confirmed by one step proof events within a range and
// processes any events found.
func (w *Watcher) checkForEdgeConfirmedByOneStepProof(
	ctx context.Context,
	filterer *challengeV2gen.EdgeChallengeManagerFilterer,
	filterOpts *bind.FilterOpts,
) error {
	it, err := filterer.FilterEdgeConfirmedByOneStepProof(filterOpts, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err = it.Close(); err != nil {
			log.Error("Could not close filter iterator", "err", err)
		}
	}()
	for it.Next() {
		if it.Error() != nil {
			return errors.Wrapf(
				err,
				"got iterator error when scanning edge creations from block %d to %d",
				filterOpts.Start,
				*filterOpts.End,
			)
		}
		_, processErr := retry.UntilSucceeds(ctx, func() (bool, error) {
			return true, w.processEdgeConfirmation(ctx, protocol.EdgeId{
				Hash: it.Event.EdgeId,
			})
		})
		if processErr != nil {
			return processErr
		}
		edgeConfirmedByOSPCounter.Inc(1)
	}
	return nil
}

// Filters for edge confirmed by time within a range and processes any events
// found.
func (w *Watcher) checkForEdgeConfirmedByTime(
	ctx context.Context,
	filterer *challengeV2gen.EdgeChallengeManagerFilterer,
	filterOpts *bind.FilterOpts,
) error {
	it, err := filterer.FilterEdgeConfirmedByTime(filterOpts, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err = it.Close(); err != nil {
			log.Error("Could not close filter iterator", "err", err)
		}
	}()
	for it.Next() {
		if it.Error() != nil {
			return errors.Wrapf(
				err,
				"got iterator error when scanning edge creations from block %d to %d",
				filterOpts.Start,
				*filterOpts.End,
			)
		}
		_, processErr := retry.UntilSucceeds(ctx, func() (bool, error) {
			return true, w.processEdgeConfirmation(ctx, protocol.EdgeId{
				Hash: it.Event.EdgeId,
			})
		})
		if processErr != nil {
			return processErr
		}
		edgeConfirmedByTimeCounter.Inc(1)
	}
	return nil
}

// Processes an edge confirmation event by checking if it claims an edge. If so,
// we add the claim id to the confirmed, level zero edge claim ids map for the
// associated assertion-level challenge the edge is a part of.
func (w *Watcher) processEdgeConfirmation(
	ctx context.Context,
	edgeId protocol.EdgeId,
) error {
	challengeManager := w.chain.SpecChallengeManager()
	edgeOpt, err := challengeManager.GetEdge(ctx, edgeId)
	if err != nil {
		return err
	}
	if edgeOpt.IsNone() {
		return errors.New("no edge found")
	}
	edge := edgeOpt.Unwrap()
	challengeParentAssertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return err
	}

	if !w.allowTrackingEdgeWithChallengeParentAssertionHash(challengeParentAssertionHash) {
		return nil
	}

	// If an edge does not have a claim ID, it is not a level zero edge, and thus
	// we can return early, as the following operations only operate on level zero
	// edges.
	if edge.ClaimId().IsNone() {
		return nil
	}

	claimId := edge.ClaimId().Unwrap()
	chal, ok := w.challenges.TryGet(challengeParentAssertionHash)
	if !ok {
		return nil
	}

	challengeComplete, err := w.chain.IsChallengeComplete(ctx, challengeParentAssertionHash)
	if err != nil {
		return errors.Wrapf(
			err,
			"could not check if edge with parent assertion hash %#x is part of a completed challenge",
			challengeParentAssertionHash.Hash,
		)
	}
	if challengeComplete {
		return nil
	}

	// Check if we should confirm the assertion by challenge winner.
	challengeLevel := edge.GetChallengeLevel()
	if challengeLevel == protocol.NewBlockChallengeLevel() {
		claimedAssertion := protocol.AssertionHash{Hash: common.Hash(claimId)}
		w.LaunchThread(func(ctx context.Context) {
			w.confirmAssertionByChallengeWinner(ctx, edge, claimedAssertion, challengeParentAssertionHash)
		})
	}

	chal.confirmedLevelZeroEdgeClaimIds.Put(claimId, edge.Id())
	w.challenges.Put(challengeParentAssertionHash, chal)
	return nil
}

func (w *Watcher) confirmAssertionByChallengeWinner(ctx context.Context, edge protocol.SpecEdge, claimedAssertion protocol.AssertionHash, challengeParentAssertionHash protocol.AssertionHash) {
	edgeConfirmedAtBlock, err := retry.UntilSucceeds(ctx, func() (uint64, error) {
		return edge.ConfirmedAtBlock(ctx)
	})
	if err != nil {
		log.Error("Could not get edge confirmed at block", "err", err)
		return
	}
	challengeGracePeriodBlocks, err := retry.UntilSucceeds(ctx, func() (uint64, error) {
		return w.chain.RollupUserLogic().ChallengeGracePeriodBlocks(w.chain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	})
	if err != nil {
		log.Error("Could not get challenge grace period blocks", "err", err)
		return
	}
	assertionCreationInfo, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
		return w.chain.ReadAssertionCreationInfo(ctx, claimedAssertion)
	})
	if err != nil {
		log.Error("Could not get assertion creation info", "err", err)
		return
	}
	parentCreationInfo, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
		return w.chain.ReadAssertionCreationInfo(
			ctx, assertionCreationInfo.ParentAssertionHash,
		)
	})
	if err != nil {
		log.Error("Could not get parent assertion creation info", "err", err)
		return
	}
	confirmableAtBlock := challengedAssertionConfirmableBlock(
		parentCreationInfo,
		edgeConfirmedAtBlock,
		assertionCreationInfo,
		challengeGracePeriodBlocks,
	)

	exceedsMaxMempoolSizeEphemeralErrorHandler := ephemeral.NewEphemeralErrorHandler(10*time.Minute, "posting this transaction will exceed max mempool size", 0)
	gasEstimationEphemeralErrorHandler := ephemeral.NewEphemeralErrorHandler(10*time.Minute, "gas estimation errored for tx with hash", 0)

	// Compute the number of blocks until we reach the assertion's
	// deadline for confirmation.
	ticker := time.NewTicker(w.assertionConfirmingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			confirmed, err := solimpl.TryConfirmingAssertion(
				ctx,
				claimedAssertion,
				confirmableAtBlock,
				w.chain,
				w.averageTimeForBlockCreation,
				option.Some(edge.Id()),
			)
			if err != nil {
				logLevel := log.Error
				logLevel = exceedsMaxMempoolSizeEphemeralErrorHandler.LogLevel(err, logLevel)
				logLevel = gasEstimationEphemeralErrorHandler.LogLevel(err, logLevel)

				logLevel("Could not confirm assertion", "err", err, "assertionHash", claimedAssertion)
				errorConfirmingAssertionByWinnerCounter.Inc(1)
				continue
			}

			exceedsMaxMempoolSizeEphemeralErrorHandler.Reset()
			gasEstimationEphemeralErrorHandler.Reset()

			if confirmed {
				assertionConfirmedCounter.Inc(1)
				log.Info("Confirmed assertion by challenge win", "assertionHash", claimedAssertion)
				return
			}
		}
	}
}

func challengedAssertionConfirmableBlock(
	parentInfo *protocol.AssertionCreatedInfo,
	winningEdgeConfirmationBlock uint64,
	info *protocol.AssertionCreatedInfo,
	challengeGracePeriodBlocks uint64,
) uint64 {
	confirmableAtBlock := info.CreationL1Block + parentInfo.ConfirmPeriodBlocks
	if winningEdgeConfirmationBlock+challengeGracePeriodBlocks > confirmableAtBlock {
		confirmableAtBlock = winningEdgeConfirmationBlock + challengeGracePeriodBlocks
	}
	return confirmableAtBlock
}

type filterRange struct {
	startBlockNum uint64
	endBlockNum   uint64
}

// Gets the start and end block numbers for our filter queries, starting from
// the latest confirmed assertion's block number up to the latest block number.
func (w *Watcher) getStartEndBlockNum(ctx context.Context) (filterRange, error) {
	latestConfirmedAssertion, err := retry.UntilSucceeds(ctx, func() (protocol.Assertion, error) {
		return w.chain.LatestConfirmed(ctx, w.chain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx}))
	})
	if err != nil {
		return filterRange{}, err
	}
	latestDesiredBlockNum, err := retry.UntilSucceeds(ctx, func() (uint64, error) {
		return w.chain.DesiredHeaderU64(ctx)
	})
	if err != nil {
		return filterRange{}, err
	}
	latestConfirmedAssertionCreationBlock, err := w.chain.GetAssertionCreationParentBlock(ctx, latestConfirmedAssertion.Id().Hash)
	if err != nil {
		return filterRange{}, err
	}
	return filterRange{
		startBlockNum: latestConfirmedAssertionCreationBlock,
		endBlockNum:   latestDesiredBlockNum,
	}, nil
}

func (w *Watcher) saveEdgeToDB(
	ctx context.Context,
	edge protocol.SpecEdge,
	isRoyal bool,
) error {
	if api.IsNil(w.apiDB) {
		return nil
	}
	start, startCommit := edge.StartCommitment()
	end, endCommit := edge.EndCommitment()
	creation, err := edge.CreatedAtBlock()
	if err != nil {
		return err
	}
	var miniStaker common.Address
	if edge.MiniStaker().IsSome() {
		miniStaker = edge.MiniStaker().Unwrap()
	}
	assertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return err
	}
	var claimId common.Hash
	if edge.ClaimId().IsSome() {
		claimId = common.Hash(edge.ClaimId().Unwrap())
	}
	inheritedTimer, err := w.InheritedTimerForEdge(ctx, edge.Id())
	if err != nil {
		return err
	}
	lowerChild, err := edge.LowerChild(ctx)
	if err != nil {
		return err
	}
	upperChild, err := edge.UpperChild(ctx)
	if err != nil {
		return err
	}
	var lowerChildId, upperChildId common.Hash
	var hasChildren bool
	if lowerChild.IsSome() {
		hasChildren = true
		lowerChildId = lowerChild.Unwrap().Hash
	}
	if upperChild.IsSome() {
		hasChildren = true
		upperChildId = upperChild.Unwrap().Hash
	}
	status, err := edge.Status(ctx)
	if err != nil {
		return err
	}
	timeUnrivaled, err := edge.TimeUnrivaled(ctx)
	if err != nil {
		return err
	}
	hasRival, err := edge.HasRival(ctx)
	if err != nil {
		return err
	}
	hasLengthOneRival, err := edge.HasLengthOneRival(ctx)
	if err != nil {
		return err
	}
	inherited := inheritedTimer
	if inherited == math.MaxUint64 {
		inherited = (1 << 63) - 1
	}
	cumulative := inheritedTimer
	if cumulative == math.MaxUint64 {
		cumulative = (1 << 63) - 1
	}
	return w.apiDB.InsertEdge(&api.JsonEdge{
		Id:                  edge.Id().Hash,
		ChallengeLevel:      uint8(edge.GetChallengeLevel()),
		StartHistoryRoot:    startCommit,
		StartHeight:         uint64(start),
		EndHistoryRoot:      endCommit,
		EndHeight:           uint64(end),
		CreatedAtBlock:      creation,
		MutualId:            common.Hash(edge.MutualId()),
		OriginId:            common.Hash(edge.OriginId()),
		ClaimId:             claimId,
		MiniStaker:          miniStaker,
		AssertionHash:       assertionHash.Hash,
		Status:              status.String(),
		LowerChildId:        lowerChildId,
		UpperChildId:        upperChildId,
		HasChildren:         hasChildren,
		IsRoyal:             isRoyal,
		InheritedTimer:      uint64(inherited),
		CumulativePathTimer: uint64(cumulative),
		TimeUnrivaled:       timeUnrivaled,
		HasRival:            hasRival,
		HasLengthOneRival:   hasLengthOneRival,
	})
}
