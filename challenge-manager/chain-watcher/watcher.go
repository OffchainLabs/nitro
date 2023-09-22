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

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"
	"github.com/OffchainLabs/bold/containers"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pkg/errors"
)

var (
	srvlog                         = log.New("service", "chain-watcher")
	edgeAddedCounter               = metrics.NewRegisteredCounter("arb/validator/watcher/edge_added", nil)
	edgeConfirmedByChildrenCounter = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_children", nil)
	edgeConfirmedByTimeCounter     = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_time", nil)
	edgeConfirmedByOSPCounter      = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_osp", nil)
	edgeConfirmedByClaimCounter    = metrics.NewRegisteredCounter("arb/validator/watcher/confirmed_by_claim", nil)
)

func init() {
	srvlog.SetHandler(log.StreamHandler(os.Stdout, log.LogfmtFormat()))
}

// EdgeManager provides a method to track edges, via edge tracker goroutines.
type EdgeManager interface {
	TrackEdge(ctx context.Context, edge protocol.SpecEdge) error
}

// ConfirmationMetadataChecker defines a struct which can retrieve information about
// an edge to determine if it can be confirmed via different means. For example,
// checking if a confirmed edge exists that claims a specified edge id as its claim id,
// or retrieving the cumulative, honest path timer for an edge and its honest ancestors.
// This information is used in order to confirm edges onchain.
type ConfirmationMetadataChecker interface {
	ConfirmedEdgeWithClaimExists(
		topLevelAssertionHash protocol.AssertionHash,
		claimId protocol.ClaimId,
	) (protocol.EdgeId, bool)
	ComputeHonestPathTimer(
		ctx context.Context,
		topLevelAssertionHash protocol.AssertionHash,
		edgeId protocol.EdgeId,
	) (challengetree.PathTimer, challengetree.HonestAncestors, error)
}

// Represents a set of honest edges being tracked in a top-level challenge and all the
// associated subchallenge honest edges along with some more metadata used for
// computing information needed for confirmations. Each time an edge is created onchain,
// the challenge watcher service will add it to its respective "trackedChallenge"
// namespaced under the top-level assertion hash the edge belongs to.
type trackedChallenge struct {
	honestEdgeTree                 *challengetree.HonestChallengeTree
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
	histChecker          l2stateprovider.HistoryChecker
	chain                protocol.AssertionChain
	edgeManager          EdgeManager
	pollEventsInterval   time.Duration
	challenges           *threadsafe.Map[protocol.AssertionHash, *trackedChallenge]
	backend              bind.ContractBackend
	validatorName        string
	initialSyncCompleted atomic.Bool
}

// New initializes a watcher service for frequently scanning the chain
// for edge creations and confirmations.
func New(
	chain protocol.AssertionChain,
	edgeManager EdgeManager,
	histChecker l2stateprovider.HistoryChecker,
	backend bind.ContractBackend,
	interval time.Duration,
	validatorName string,
) *Watcher {
	return &Watcher{
		chain:              chain,
		edgeManager:        edgeManager,
		pollEventsInterval: interval,
		challenges:         threadsafe.NewMap[protocol.AssertionHash, *trackedChallenge](),
		backend:            backend,
		histChecker:        histChecker,
		validatorName:      validatorName,
	}
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

// ComputeHonestPathTimer computes the honest path timer for an edge id within an assertion hash challenge
// namespace. This is used during the confirmation process for edges in
// edge tracker goroutine logic.
func (w *Watcher) ComputeHonestPathTimer(
	ctx context.Context,
	topLevelAssertionHash protocol.AssertionHash,
	edgeId protocol.EdgeId,
) (challengetree.PathTimer, challengetree.HonestAncestors, error) {
	header, err := w.backend.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	if !header.Number.IsUint64() {
		return 0, nil, errors.New("latest block header number is not a uint64")
	}
	blockNumber := header.Number.Uint64()
	chal, ok := w.challenges.TryGet(topLevelAssertionHash)
	if !ok {
		return 0, nil, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			topLevelAssertionHash,
		)
	}
	response, err := chal.honestEdgeTree.ComputeAncestorsWithTimers(ctx, edgeId, blockNumber)
	if err != nil {
		return 0, nil, err
	}
	pathTimer, err := chal.honestEdgeTree.ComputeHonestPathTimer(ctx, edgeId, response.AncestorLocalTimers, blockNumber)
	if err != nil {
		return 0, nil, err
	}
	return pathTimer, response.AncestorEdgeIds, nil
}

func (w *Watcher) IsSynced() bool {
	return w.initialSyncCompleted.Load()
}

// Start watching the chain via a polling mechanism for all edge added and confirmation events
// in order to process some of this data into internal representations for confirmation purposes.
func (w *Watcher) Start(ctx context.Context) {
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
		return true, w.checkForEdgeConfirmedByChildren(ctx, filterer, filterOpts)
	})
	if err != nil {
		srvlog.Error("Could not check for edge confirmed by children", log.Ctx{"err": err})
		return
	}
	_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
		return true, w.checkForEdgeConfirmedByClaim(ctx, filterer, filterOpts)
	})
	if err != nil {
		srvlog.Error("Could not check for edge confirmed by claim", log.Ctx{"err": err})
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
			latestBlock, err := w.backend.HeaderByNumber(ctx, nil)
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
			if err = w.checkForEdgeConfirmedByChildren(ctx, filterer, filterOpts); err != nil {
				srvlog.Error("Could not check for edge confirmed by children", log.Ctx{"err": err})
				continue
			}
			if err = w.checkForEdgeConfirmedByTime(ctx, filterer, filterOpts); err != nil {
				srvlog.Error("Could not check for edge confirmed by time", log.Ctx{"err": err})
				continue
			}
			if err = w.checkForEdgeConfirmedByClaim(ctx, filterer, filterOpts); err != nil {
				srvlog.Error("Could not check for edge confirmed by claim", log.Ctx{"err": err})
				continue
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

// GetEdges returns all edges in the watcher.
func (w *Watcher) GetEdges() []protocol.SpecEdge {
	syncEdges := make([]protocol.SpecEdge, 0)
	//nolint:err
	_ = w.challenges.ForEach(func(AssertionHash protocol.AssertionHash, t *trackedChallenge) error {
		//nolint:err
		_ = t.honestEdgeTree.GetEdges().ForEach(func(edgeId protocol.EdgeId, edge protocol.SpecEdge) error {
			syncEdges = append(syncEdges, edge)
			return nil
		})
		return nil
	})
	return syncEdges
}

// AddVerifiedHonestEdge adds an edge known to be honest to the chain watcher's internally
// tracked challenge trees and spawns an edge tracker for it. Should be called after the challenge
// manager creates a new edge, or bisects an edge and produces two children from that move.
func (w *Watcher) AddVerifiedHonestEdge(ctx context.Context, edge protocol.VerifiedHonestEdge) error {
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
			w.validatorName,
		)
		chal = &trackedChallenge{
			honestEdgeTree:                 tree,
			confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap[protocol.ClaimId, protocol.EdgeId](),
		}
		w.challenges.Put(assertionHash, chal)
	}
	// Add the edge to a local challenge tree of honest edges and, if needed,
	// we also spawn a tracker for the edge.
	if err := chal.honestEdgeTree.AddHonestEdge(edge); err != nil {
		return errors.Wrap(err, "could not add honest edge to challenge tree")
	}
	return w.edgeManager.TrackEdge(ctx, edge)
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
		_, processErr := retry.UntilSucceeds(ctx, func() (bool, error) {
			return true, w.processEdgeAddedEvent(ctx, it.Event)
		})
		if processErr != nil {
			return processErr
		}
		edgeAddedCounter.Inc(1)
	}
	return nil
}

// Processes an edge added event by adding it to the honest challenge tree if it is honest.
func (w *Watcher) processEdgeAddedEvent(
	ctx context.Context,
	event *challengeV2gen.EdgeChallengeManagerEdgeAdded,
) error {
	challengeManager, err := w.chain.SpecChallengeManager(ctx)
	if err != nil {
		return err
	}
	edgeOpt, err := challengeManager.GetEdge(ctx, protocol.EdgeId{Hash: event.EdgeId})
	if err != nil {
		return err
	}
	if edgeOpt.IsNone() {
		return fmt.Errorf("no edge found with id %#x", event.EdgeId)
	}
	edge := edgeOpt.Unwrap()

	assertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return err
	}
	chal, ok := w.challenges.TryGet(assertionHash)
	if !ok {
		tree := challengetree.New(
			protocol.AssertionHash{Hash: event.OriginId},
			w.chain,
			w.histChecker,
			w.validatorName,
		)
		chal = &trackedChallenge{
			honestEdgeTree:                 tree,
			confirmedLevelZeroEdgeClaimIds: threadsafe.NewMap[protocol.ClaimId, protocol.EdgeId](),
		}
		w.challenges.Put(assertionHash, chal)
	}
	// Add the edge to a local challenge tree of tracked edges. If it is honest,
	// we also spawn a tracker for the edge.
	agreement, err := chal.honestEdgeTree.AddEdge(ctx, edge)
	if err != nil {
		return errors.Wrap(err, "could not add edge to challenge tree")
	}
	if agreement.IsHonestEdge {
		return w.edgeManager.TrackEdge(ctx, edge)
	}
	return nil
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

// Filters for edge confirmed by children within a range.
// and processes any events found.
func (w *Watcher) checkForEdgeConfirmedByChildren(
	ctx context.Context,
	filterer *challengeV2gen.EdgeChallengeManagerFilterer,
	filterOpts *bind.FilterOpts,
) error {
	it, err := filterer.FilterEdgeConfirmedByChildren(filterOpts, nil, nil)
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
		edgeConfirmedByChildrenCounter.Inc(1)
	}
	return nil
}

// Filters for edge confirmed by claim within a range.
// and processes any events found.
func (w *Watcher) checkForEdgeConfirmedByClaim(
	ctx context.Context,
	filterer *challengeV2gen.EdgeChallengeManagerFilterer,
	filterOpts *bind.FilterOpts,
) error {
	it, err := filterer.FilterEdgeConfirmedByClaim(filterOpts, nil, nil)
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
		edgeConfirmedByClaimCounter.Inc(1)
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
	assertionHash, err := edge.AssertionHash(ctx)
	if err != nil {
		return err
	}

	// If an edge does not have a claim ID, it is not a level zero edge, and thus we can return early,
	// as the following operations only operate on level zero edges.
	if edge.ClaimId().IsNone() {
		return nil
	}

	claimId := edge.ClaimId().Unwrap()
	chal, ok := w.challenges.TryGet(assertionHash)
	if !ok {
		return nil
	}

	// Check if we should confirm the assertion by challenge winner.
	challengeLevel, err := edge.GetChallengeLevel()
	if err != nil {
		return err
	}
	if challengeLevel == protocol.NewBlockChallengeLevel() {
		if confirmAssertionErr := w.chain.ConfirmAssertionByChallengeWinner(ctx, protocol.AssertionHash{Hash: common.Hash(claimId)}, edgeId); confirmAssertionErr != nil {
			return confirmAssertionErr
		}
		srvlog.Info("Assertion confirmed by challenge win", log.Ctx{
			"assertionHash": containers.Trunc(assertionHash.Bytes()),
		})
	}

	chal.confirmedLevelZeroEdgeClaimIds.Put(claimId, edge.Id())
	w.challenges.Put(assertionHash, chal)
	return nil
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
	firstBlock, err := latestConfirmed.CreatedAtBlock()
	if err != nil {
		return filterRange{}, err
	}
	startBlock := firstBlock
	header, err := w.backend.HeaderByNumber(ctx, nil)
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
