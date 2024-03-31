// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

// Package watcher implements the main monitoring logic for protocol validators.
// The challenge watcher is a singleton service available to all spawned edge trackers
// and it tracks common information such as the edges' ancestors and an edge's time unrivaled.
//
// See: [github.com/OffchainLabs/bold/challenge-manager/edge-tracker]
package watcher

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/OffchainLabs/bold/api"
	"github.com/OffchainLabs/bold/api/db"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/util"
	"github.com/OffchainLabs/bold/util/stopwaiter"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pkg/errors"
)

var (
	srvlog                                  = log.New("service", "chain-watcher")
	edgeAddedCounter                        = metrics.NewRegisteredCounter("arb/validator/watcher/edge_added", nil)
	edgeConfirmedByTimeCounter              = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_time", nil)
	edgeConfirmedByOSPCounter               = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_osp", nil)
	errorConfirmingAssertionByWinnerCounter = metrics.NewRegisteredCounter("arb/validator/watcher/error_confirming_assertion_by_winner", nil)
	assertionConfirmedCounter               = metrics.GetOrRegisterCounter("arb/validator/scanner/assertion_confirmed", nil)
)

func init() {
	srvlog.SetHandler(log.StreamHandler(os.Stdout, log.LogfmtFormat()))
}

// EdgeManager provides a method to track edges, via edge tracker goroutines.
type EdgeManager interface {
	TrackEdge(ctx context.Context, edge protocol.SpecEdge) error
}

// Represents a set of honest edges being tracked in a top-level challenge and all the
// associated subchallenge honest edges along with some more metadata used for
// computing information needed for confirmations. Each time an edge is created onchain,
// the challenge watcher service will add it to its respective "trackedChallenge"
// namespaced under the top-level assertion hash the edge belongs to.
type trackedChallenge struct {
	honestEdgeTree                 *challengetree.RoyalChallengeTree
	confirmedLevelZeroEdgeClaimIds *threadsafe.Map[protocol.ClaimId, protocol.EdgeId]
}

// The Watcher implements a service in the validator runtime
// that is in charge of scanning through all edge creation events via a polling
// mechanism. It will keep track of edges the validator's state provider agrees with
// within trackedChallenge instances. The challenge watcher provides two useful
// methods: (a) the ability to compute the honest path timer of an edge, and
// (b) the ability to check if an edge with a certain claim id has been confirmed. Both
// are used during the confirmation process in edge tracker goroutines.
type Watcher struct {
	stopwaiter.StopWaiter
	histChecker                 l2stateprovider.HistoryChecker
	chain                       protocol.AssertionChain
	edgeManager                 EdgeManager
	pollEventsInterval          time.Duration
	challenges                  *threadsafe.Map[protocol.AssertionHash, *trackedChallenge]
	backend                     bind.ContractBackend
	validatorName               string
	numBigStepLevels            uint8
	initialSyncCompleted        atomic.Bool
	apiDB                       db.Database
	assertionConfirmingInterval time.Duration
	averageTimeForBlockCreation time.Duration
}

// New initializes a watcher service for frequently scanning the chain
// for edge creations and confirmations.
func New(
	chain protocol.AssertionChain,
	edgeManager EdgeManager,
	histChecker l2stateprovider.HistoryChecker,
	backend bind.ContractBackend,
	interval time.Duration,
	numBigStepLevels uint8,
	validatorName string,
	apiDB db.Database,
	assertionConfirmingInterval time.Duration,
	averageTimeForBlockCreation time.Duration,
) (*Watcher, error) {
	if interval == 0 {
		return nil, errors.New("chain watcher polling interval must be greater than 0")
	}
	return &Watcher{
		chain:                       chain,
		edgeManager:                 edgeManager,
		pollEventsInterval:          interval,
		challenges:                  threadsafe.NewMap[protocol.AssertionHash, *trackedChallenge](threadsafe.MapWithMetric[protocol.AssertionHash, *trackedChallenge]("challenges")),
		backend:                     backend,
		histChecker:                 histChecker,
		numBigStepLevels:            numBigStepLevels,
		validatorName:               validatorName,
		apiDB:                       apiDB,
		assertionConfirmingInterval: assertionConfirmingInterval,
		averageTimeForBlockCreation: averageTimeForBlockCreation,
	}, nil
}

// HonestBlockChallengeRootEdge gets the honest block challenge root edge for a given challenge
// by challenged assertion id if it exists.
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

// ConfirmedEdgeWithClaimExists checks if a confirmed, level zero edge exists that claims a particular
// edge id for a tracked challenge. This is used during the confirmation process of edges
// within edge tracker goroutines. Returns the claiming edge id.
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

func (w *Watcher) InheritedTimer(
	ctx context.Context,
	edgeId protocol.EdgeId,
) (protocol.InheritedTimer, error) {
	chalManager, err := w.chain.SpecChallengeManager(ctx)
	if err != nil {
		return 0, err
	}
	edgeOpt, err := chalManager.GetEdge(ctx, edgeId)
	if err != nil {
		return 0, err
	}
	if edgeOpt.IsNone() {
		return 0, fmt.Errorf("no edge found with id %#x", edgeId.Hash)

	}
	return edgeOpt.Unwrap().InheritedTimer(ctx)
}

func (w *Watcher) IsSynced() bool {
	return w.initialSyncCompleted.Load()
}

// Start watching the chain via a polling mechanism for all edge added and confirmation events
// in order to process some of this data into internal representations for confirmation purposes.
func (w *Watcher) Start(ctx context.Context) {
	w.StopWaiter.Start(ctx, w)
	scanRange, err := retry.UntilSucceeds(ctx, func() (filterRange, error) {
		return w.getStartEndBlockNum(ctx)
	})
	if err != nil {
		srvlog.Error("Could not get start and end block num", log.Ctx{"err": err})
		return
	}
	fromBlock := scanRange.startBlockNum
	toBlock := scanRange.endBlockNum

	// Get a challenge manager instance and filterer.
	challengeManager, err := retry.UntilSucceeds(ctx, func() (protocol.SpecChallengeManager, error) {
		return w.chain.SpecChallengeManager(ctx)
	})
	if err != nil {
		srvlog.Error("Could not get spec challenge manager", log.Ctx{"err": err})
		return
	}
	filterer, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerFilterer, error) {
		return challengeV2gen.NewEdgeChallengeManagerFilterer(challengeManager.Address(), w.backend)
	})
	if err != nil {
		srvlog.Error("Could not initialize edge challenge manager filterer", log.Ctx{"err": err})
		return
	}
	filterOpts := &bind.FilterOpts{
		Start:   fromBlock,
		End:     &toBlock,
		Context: ctx,
	}

	// Checks for different events right away before we start polling.
	_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
		return true, w.checkForEdgeAdded(ctx, filterer, filterOpts)
	})
	if err != nil {
		srvlog.Error("Could not check for edge added", log.Ctx{"err": err})
		return
	}
	_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
		return true, w.checkForEdgeConfirmedByOneStepProof(ctx, filterer, filterOpts)
	})
	if err != nil {
		srvlog.Error("Could not check for edge confirmed by osp", log.Ctx{"err": err})
		return
	}
	_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
		return true, w.checkForEdgeConfirmedByTime(ctx, filterer, filterOpts)
	})
	if err != nil {
		srvlog.Error("Could not check for edge confirmed by time", log.Ctx{"err": err})
		return
	}

	w.initialSyncCompleted.Store(true)

	fromBlock = toBlock
	ticker := time.NewTicker(w.pollEventsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := w.backend.HeaderByNumber(ctx, util.GetSafeBlockNumber())
			if err != nil {
				srvlog.Error("Could not get latest header", log.Ctx{"err": err})
				continue
			}
			if !latestBlock.Number.IsUint64() {
				srvlog.Error("latest block header number is not a uint64")
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			// Get a challenge manager instance and filterer.
			challengeManager, err := retry.UntilSucceeds(ctx, func() (protocol.SpecChallengeManager, error) {
				return w.chain.SpecChallengeManager(ctx)
			})
			if err != nil {
				srvlog.Error("Could not get spec challenge manager", log.Ctx{"err": err})
				return
			}
			filterer, err = retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerFilterer, error) {
				return challengeV2gen.NewEdgeChallengeManagerFilterer(challengeManager.Address(), w.backend)
			})
			if err != nil {
				srvlog.Error("Could not get challenge manager filterer", log.Ctx{"err": err})
				return
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			if err = w.checkForEdgeAdded(ctx, filterer, filterOpts); err != nil {
				srvlog.Error("Could not check for edge added", log.Ctx{"err": err})
				continue
			}
			if err = w.checkForEdgeConfirmedByOneStepProof(ctx, filterer, filterOpts); err != nil {
				srvlog.Error("Could not check for edge confirmed by osp", log.Ctx{"err": err})
				continue
			}
			if err = w.checkForEdgeConfirmedByTime(ctx, filterer, filterOpts); err != nil {
				srvlog.Error("Could not check for edge confirmed by time", log.Ctx{"err": err})
				continue
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

// GetRoyalEdges returns all royal, tracked edges in the watcher by assertion hash.
func (w *Watcher) GetRoyalEdges(ctx context.Context) (map[protocol.AssertionHash][]*api.JsonTrackedRoyalEdge, error) {
	header, err := w.chain.Backend().HeaderByNumber(ctx, util.GetSafeBlockNumber())
	if err != nil {
		return nil, err
	}
	if !header.Number.IsUint64() {
		return nil, errors.New("block header is not a uint64")
	}
	blockNum := header.Number.Uint64()
	response := make(map[protocol.AssertionHash][]*api.JsonTrackedRoyalEdge)
	if err = w.challenges.ForEach(func(assertionHash protocol.AssertionHash, t *trackedChallenge) error {
		return t.honestEdgeTree.GetEdges().ForEach(func(edgeId protocol.EdgeId, edge protocol.SpecEdge) error {
			start, startRoot := edge.StartCommitment()
			end, endRoot := edge.EndCommitment()
			createdAt, err2 := edge.CreatedAtBlock()
			if err2 != nil {
				return err2
			}
			unrivaled, err2 := t.honestEdgeTree.IsUnrivaledAtBlockNum(edge, blockNum)
			if err2 != nil {
				return err2
			}
			hasRival := !unrivaled
			timeUnrivaled, err2 := t.honestEdgeTree.TimeUnrivaled(edge, blockNum)
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
	blockHeader, err := w.chain.Backend().HeaderByNumber(ctx, util.GetSafeBlockNumber())
	if err != nil {
		return nil, err
	}
	if !blockHeader.Number.IsUint64() {
		return nil, errors.New("block number is not uint64")
	}
	return chal.honestEdgeTree.ComputeAncestors(ctx, edgeId, blockHeader.Number.Uint64())
}

func (w *Watcher) ComputeRootInheritedTimer(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
) (protocol.InheritedTimer, error) {
	chal, ok := w.challenges.TryGet(challengedAssertionHash)
	if !ok {
		return 0, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			challengedAssertionHash,
		)
	}
	blockHeader, err := w.chain.Backend().HeaderByNumber(ctx, util.GetSafeBlockNumber())
	if err != nil {
		return 0, err
	}
	if !blockHeader.Number.IsUint64() {
		return 0, errors.New("block number is not uint64")
	}
	return chal.honestEdgeTree.ComputeRootInheritedTimer(ctx, challengedAssertionHash, blockHeader.Number.Uint64())
}

// AddVerifiedHonestEdge adds an edge known to be honest to the chain watcher's internally
// tracked challenge trees and spawns an edge tracker for it. Should be called after the challenge
// manager creates a new edge, or bisects an edge and produces two children from that move.
func (w *Watcher) AddVerifiedHonestEdge(ctx context.Context, edge protocol.VerifiedRoyalEdge) error {
	assertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return err
	}
	// If a challenge is not yet being tracked locally by the watcher
	// for the edge's assertion hash, it adds an entry to the map.
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
			confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap[protocol.ClaimId, protocol.EdgeId](threadsafe.MapWithMetric[protocol.ClaimId, protocol.EdgeId]("confirmedLevelZeroEdgeClaimIds")),
		}
		w.challenges.Put(assertionHash, chal)
	}
	// Add the edge to a local challenge tree of honest edges and, if needed,
	// we also spawn a tracker for the edge.
	start, startRoot := edge.StartCommitment()
	end, endRoot := edge.EndCommitment()
	fields := log.Ctx{
		"edgeId":         edge.Id().Hash,
		"challengeLevel": edge.GetChallengeLevel(),
		"assertionHash":  assertionHash.Hash,
		"startHeight":    start,
		"endHeight":      end,
		"startRoot":      startRoot,
		"endRoot":        endRoot,
	}
	srvlog.Info("Observed an honest challenge edge created onchain, now tracking it locally", fields)
	if err = chal.honestEdgeTree.AddRoyalEdge(edge); err != nil {
		log.Error("Could not add verified honest edge to local cache", log.Ctx{"error": err})
		return errors.Wrap(err, "could not add honest edge to challenge tree")
	}
	go func() {
		if _, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			if innerErr := w.saveEdgeToDB(ctx, edge, true /* is royal */); innerErr != nil {
				srvlog.Error("Could not save edge to db", log.Ctx{"err": innerErr})
				return false, innerErr
			}
			return false, nil
		}); err != nil {
			srvlog.Error("Could not save edge to db", log.Ctx{"err": err})
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
			srvlog.Error("Could not close filter iterator", log.Ctx{"err": err})
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
	challengeComplete, err := w.chain.IsChallengeComplete(ctx, challengeParentAssertionHash)
	if err != nil {
		return false, errors.Wrapf(
			err,
			"could not check if edge with parent assertion hash %#x is part of a completed challenge",
			challengeParentAssertionHash.Hash,
		)
	}
	start, startRoot := edge.StartCommitment()
	end, endRoot := edge.EndCommitment()
	if challengeComplete {
		return false, nil
	}
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
			confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap[protocol.ClaimId, protocol.EdgeId](threadsafe.MapWithMetric[protocol.ClaimId, protocol.EdgeId]("confirmedLevelZeroEdgeClaimIds")),
		}
		w.challenges.Put(challengeParentAssertionHash, chal)
	}
	// Add the edge to a local challenge tree of tracked edges. If it is honest,
	// we also spawn a tracker for the edge.
	isRoyalEdge, err := chal.honestEdgeTree.AddEdge(ctx, edge)
	if err != nil {
		if !errors.Is(err, challengetree.ErrAlreadyBeingTracked) {
			return false, errors.Wrap(err, "could not add edge to challenge tree")
		}
		// If the error is that we are already tracking the edge, we exit early.
		return false, nil
	}
	if isRoyalEdge {
		err = w.edgeManager.TrackEdge(ctx, edge)
		if err != nil {
			return false, err
		}
	}
	fields := log.Ctx{
		"edgeId":                  edge.Id().Hash,
		"challengeLevel":          edge.GetChallengeLevel(),
		"challengedAssertionHash": challengeParentAssertionHash.Hash,
		"startHeight":             start,
		"endHeight":               end,
		"startRoot":               startRoot,
		"endRoot":                 endRoot,
		"isHonestEdge":            isRoyalEdge,
	}
	if isRoyalEdge {
		srvlog.Info("Observed an honest challenge edge created onchain, now tracking it locally", fields)
	} else {
		srvlog.Info("Observed an evil edge created onchain from an adversary, will make necessary moves on it", fields)
	}
	go func() {
		if _, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			if innerErr := w.saveEdgeToDB(ctx, edge, isRoyalEdge); innerErr != nil {
				srvlog.Error("Could not save edge to db", log.Ctx{"err": innerErr})
				return false, innerErr
			}
			return false, nil
		}); err != nil {
			srvlog.Error("Could not save edge to db", log.Ctx{"err": err})
		}
	}()
	return true, nil
}

// Processes an edge added event by adding it to the honest challenge tree if it is honest.
func (w *Watcher) processEdgeAddedEvent(
	ctx context.Context,
	event *challengeV2gen.EdgeChallengeManagerEdgeAdded,
) (bool, error) {
	challengeManager, err := w.chain.SpecChallengeManager(ctx)
	if err != nil {
		return false, err
	}
	edgeOpt, err := challengeManager.GetEdge(ctx, protocol.EdgeId{Hash: event.EdgeId})
	if err != nil {
		return false, err
	}
	if edgeOpt.IsNone() {
		return false, fmt.Errorf("no edge found with id %#x", event.EdgeId)
	}
	return w.AddEdge(ctx, edgeOpt.Unwrap())
}

// Filters for edge confirmed by one step proof events within a range.
// and processes any events found.
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
			srvlog.Error("Could not close filter iterator", log.Ctx{"err": err})
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

// Filters for edge confirmed by time within a range.
// and processes any events found.
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
			srvlog.Error("Could not close filter iterator", log.Ctx{"err": err})
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

// Processes an edge confirmation event by checking if it claims an edge. If so, we add
// the claim id to the confirmed, level zero edge claim ids map for the associated
// assertion-level challenge the edge is a part of.
func (w *Watcher) processEdgeConfirmation(
	ctx context.Context,
	edgeId protocol.EdgeId,
) error {
	challengeManager, err := w.chain.SpecChallengeManager(ctx)
	if err != nil {
		return err
	}
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

	// If an edge does not have a claim ID, it is not a level zero edge, and thus we can return early,
	// as the following operations only operate on level zero edges.
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
		w.LaunchThread(func(ctx context.Context) {
			w.confirmAssertionByChallengeWinner(ctx, edge, claimId, challengeParentAssertionHash)
		})
	}

	chal.confirmedLevelZeroEdgeClaimIds.Put(claimId, edge.Id())
	w.challenges.Put(challengeParentAssertionHash, chal)
	return nil
}

func (w *Watcher) confirmAssertionByChallengeWinner(ctx context.Context, edge protocol.SpecEdge, claimId protocol.ClaimId, challengeParentAssertionHash protocol.AssertionHash) {
	edgeConfirmedAtBlock, err := retry.UntilSucceeds(ctx, func() (uint64, error) {
		return edge.ConfirmedAtBlock(ctx)
	})
	if err != nil {
		log.Error("Could not get edge confirmed at block", log.Ctx{"err": err})
		return
	}
	challengeGracePeriodBlocks, err := retry.UntilSucceeds(ctx, func() (uint64, error) {
		return w.chain.RollupUserLogic().RollupUserLogicCaller.ChallengeGracePeriodBlocks(util.GetSafeCallOpts(&bind.CallOpts{Context: ctx}))
	})
	if err != nil {
		log.Error("Could not get challenge grace period blocks", log.Ctx{"err": err})
		return
	}
	ticker := time.NewTicker(w.assertionConfirmingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			confirmed, err := solimpl.TryConfirmingAssertion(ctx, common.Hash(claimId), edgeConfirmedAtBlock+challengeGracePeriodBlocks, w.chain, w.averageTimeForBlockCreation, option.Some(edge.Id()))
			if err != nil {
				srvlog.Error("Could not confirm assertion", log.Ctx{"err": err, "assertionHash": common.Hash(claimId)})
				errorConfirmingAssertionByWinnerCounter.Inc(1)
				continue
			}
			if confirmed {
				assertionConfirmedCounter.Inc(1)
				w.challenges.Delete(challengeParentAssertionHash)
				srvlog.Info("Confirmed assertion by challenge win", log.Ctx{"assertionHash": common.Hash(claimId)})
				return
			}
		}
	}
}

type filterRange struct {
	startBlockNum uint64
	endBlockNum   uint64
}

// Gets the start and end block numbers for our filter queries, starting from the
// latest confirmed assertion's block number up to the latest block number.
func (w *Watcher) getStartEndBlockNum(ctx context.Context) (filterRange, error) {
	latestConfirmed, err := w.chain.LatestConfirmed(ctx)
	if err != nil {
		return filterRange{}, err
	}
	firstBlock := latestConfirmed.CreatedAtBlock()
	startBlock := firstBlock
	header, err := w.backend.HeaderByNumber(ctx, util.GetSafeBlockNumber())
	if err != nil {
		return filterRange{}, err
	}
	if !header.Number.IsUint64() {
		return filterRange{}, errors.New("header number is not a uint64")
	}
	return filterRange{
		startBlockNum: startBlock,
		endBlockNum:   header.Number.Uint64(),
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
	inheritedTimer, err := w.InheritedTimer(ctx, edge.Id())
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
	return w.apiDB.InsertEdge(&api.JsonEdge{
		Id:                edge.Id().Hash,
		ChallengeLevel:    uint8(edge.GetChallengeLevel()),
		StartHistoryRoot:  startCommit,
		StartHeight:       uint64(start),
		EndHistoryRoot:    endCommit,
		EndHeight:         uint64(end),
		CreatedAtBlock:    creation,
		MutualId:          common.Hash(edge.MutualId()),
		OriginId:          common.Hash(edge.OriginId()),
		ClaimId:           claimId,
		MiniStaker:        miniStaker,
		AssertionHash:     assertionHash.Hash,
		Status:            status.String(),
		LowerChildId:      lowerChildId,
		UpperChildId:      upperChildId,
		HasChildren:       hasChildren,
		IsRoyal:           isRoyal,
		InheritedTimer:    uint64(inheritedTimer),
		TimeUnrivaled:     timeUnrivaled,
		HasRival:          hasRival,
		HasLengthOneRival: hasLengthOneRival,
	})
}
