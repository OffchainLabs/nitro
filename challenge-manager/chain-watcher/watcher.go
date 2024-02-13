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
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
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

const (
	ConfirmableByChildren = "confirmable_by_children"
	ConfirmableByClaim    = "confirmable_by_claim"
	ConfirmableByTimer    = "confirmable_by_timer"
	ConfirmableByOSP      = "confirmable_by_osp"
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
	) (challengetree.PathTimer, challengetree.HonestAncestors, []challengetree.EdgeLocalTimer, error)
	HasConfirmableAncestor(
		ctx context.Context,
		topLevelAssertionHash protocol.AssertionHash,
		ancestorLocalTimers []challengetree.EdgeLocalTimer,
		challengePeriodBlocks uint64,
	) (bool, error)
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
	histChecker          l2stateprovider.HistoryChecker
	chain                protocol.AssertionChain
	edgeManager          EdgeManager
	pollEventsInterval   time.Duration
	challenges           *threadsafe.Map[protocol.AssertionHash, *trackedChallenge]
	backend              bind.ContractBackend
	validatorName        string
	numBigStepLevels     uint8
	initialSyncCompleted atomic.Bool
	apiDB                db.Database
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
) (*Watcher, error) {
	if interval == 0 {
		return nil, errors.New("chain watcher polling interval must be greater than 0")
	}
	return &Watcher{
		chain:              chain,
		edgeManager:        edgeManager,
		pollEventsInterval: interval,
		challenges:         threadsafe.NewMap[protocol.AssertionHash, *trackedChallenge](threadsafe.MapWithMetric[protocol.AssertionHash, *trackedChallenge]("challenges")),
		backend:            backend,
		histChecker:        histChecker,
		numBigStepLevels:   numBigStepLevels,
		validatorName:      validatorName,
		apiDB:              apiDB,
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

// ComputeHonestPathTimer computes the honest path timer for an edge id within an assertion hash challenge
// namespace. This is used during the confirmation process for edges in
// edge tracker goroutine logic.
func (w *Watcher) ComputeHonestPathTimer(
	ctx context.Context,
	topLevelAssertionHash protocol.AssertionHash,
	edgeId protocol.EdgeId,
) (challengetree.PathTimer, challengetree.HonestAncestors, []challengetree.EdgeLocalTimer, error) {
	header, err := w.backend.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, nil, nil, err
	}
	if !header.Number.IsUint64() {
		return 0, nil, nil, errors.New("latest block header number is not a uint64")
	}
	blockNumber := header.Number.Uint64()
	return w.ComputeHonestPathTimerByBlockNumber(ctx, topLevelAssertionHash, edgeId, blockNumber)
}

func (w *Watcher) IsRoyal(assertionHash protocol.AssertionHash, edgeId protocol.EdgeId) bool {
	chal, ok := w.challenges.TryGet(assertionHash)
	if !ok {
		return false
	}
	return chal.honestEdgeTree.HasRoyalEdge(edgeId)
}

func (w *Watcher) ComputeHonestPathTimerByBlockNumber(
	ctx context.Context,
	topLevelAssertionHash protocol.AssertionHash,
	edgeId protocol.EdgeId,
	blockNumber uint64,
) (challengetree.PathTimer, challengetree.HonestAncestors, []challengetree.EdgeLocalTimer, error) {
	chal, ok := w.challenges.TryGet(topLevelAssertionHash)
	if !ok {
		return 0, nil, nil, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			topLevelAssertionHash,
		)
	}
	response, err := chal.honestEdgeTree.ComputeAncestorsWithTimers(ctx, edgeId, blockNumber)
	if err != nil {
		return 0, nil, nil, err
	}
	pathTimer, err := chal.honestEdgeTree.ComputeHonestPathTimer(ctx, edgeId, response.AncestorLocalTimers, blockNumber)
	if err != nil {
		return 0, nil, nil, err
	}
	return pathTimer, response.AncestorEdgeIds, response.AncestorLocalTimers, nil
}

func (w *Watcher) HasConfirmableAncestor(
	ctx context.Context,
	topLevelAssertionHash protocol.AssertionHash,
	ancestorLocalTimers []challengetree.EdgeLocalTimer,
	challengePeriodBlocks uint64,
) (bool, error) {
	chal, ok := w.challenges.TryGet(topLevelAssertionHash)
	if !ok {
		return false, fmt.Errorf(
			"could not get challenge for top level assertion %#x",
			topLevelAssertionHash,
		)
	}
	return chal.honestEdgeTree.HasConfirmableAncestor(ctx, ancestorLocalTimers, challengePeriodBlocks)
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

func (w *Watcher) GetEdge(ctx context.Context, edgeId common.Hash) (protocol.SpecEdge, error) {
	challengeManager, err := w.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	edgeOpt, err := challengeManager.GetEdge(ctx, protocol.EdgeId{Hash: edgeId})
	if err != nil {
		return nil, err
	}
	if edgeOpt.IsNone() {
		return nil, fmt.Errorf("no edge found with id %#x", edgeId)
	}
	return edgeOpt.Unwrap(), nil
}

func (w *Watcher) GetEdges(ctx context.Context) ([]protocol.SpecEdge, error) {
	scanRange, err := retry.UntilSucceeds(ctx, func() (filterRange, error) {
		return w.getStartEndBlockNum(ctx)
	})
	if err != nil {
		return nil, err
	}
	fromBlock := scanRange.startBlockNum
	toBlock := scanRange.endBlockNum

	// Get a challenge manager instance and filterer.
	challengeManager, err := retry.UntilSucceeds(ctx, func() (protocol.SpecChallengeManager, error) {
		return w.chain.SpecChallengeManager(ctx)
	})
	if err != nil {
		return nil, err
	}
	filterer, err := retry.UntilSucceeds(ctx, func() (*challengeV2gen.EdgeChallengeManagerFilterer, error) {
		return challengeV2gen.NewEdgeChallengeManagerFilterer(challengeManager.Address(), w.backend)
	})
	if err != nil {
		return nil, err
	}
	filterOpts := &bind.FilterOpts{
		Start:   fromBlock,
		End:     &toBlock,
		Context: ctx,
	}

	return retry.UntilSucceeds(ctx, func() ([]protocol.SpecEdge, error) {
		return w.getAllEdges(ctx, filterer, filterOpts)
	})
}

func (w *Watcher) getAllEdges(
	ctx context.Context,
	filterer *challengeV2gen.EdgeChallengeManagerFilterer,
	filterOpts *bind.FilterOpts,
) ([]protocol.SpecEdge, error) {
	it, err := filterer.FilterEdgeAdded(filterOpts, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = it.Close(); err != nil {
			srvlog.Error("Could not close filter iterator", log.Ctx{"err": err})
		}
	}()
	edges := make([]protocol.SpecEdge, 0)
	for it.Next() {
		if it.Error() != nil {
			return nil, errors.Wrapf(
				err,
				"got iterator error when scanning edge creations from block %d to %d",
				filterOpts.Start,
				*filterOpts.End,
			)
		}
		edge, err := retry.UntilSucceeds(ctx, func() (protocol.SpecEdge, error) {
			return w.getEdgeFromEvent(ctx, it.Event)
		})
		if err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

func (w *Watcher) getEdgeFromEvent(
	ctx context.Context,
	event *challengeV2gen.EdgeChallengeManagerEdgeAdded,
) (protocol.SpecEdge, error) {
	challengeManager, err := w.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	edgeOpt, err := challengeManager.GetEdge(ctx, protocol.EdgeId{Hash: event.EdgeId})
	if err != nil {
		return nil, err
	}
	if edgeOpt.IsNone() {
		return nil, fmt.Errorf("no edge found with id %#x", event.EdgeId)
	}
	return edgeOpt.Unwrap(), nil
}

// GetRoyalEdges returns all royal, tracked edges in the watcher by assertion hash.
func (w *Watcher) GetRoyalEdges(ctx context.Context) (map[protocol.AssertionHash][]*api.JsonTrackedRoyalEdge, error) {
	header, err := w.chain.Backend().HeaderByNumber(ctx, nil)
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
			ancestorDetails, err2 := t.honestEdgeTree.ComputeAncestorsWithTimers(ctx, edgeId, blockNum)
			if err2 != nil {
				return err2
			}
			pathTimer, err2 := t.honestEdgeTree.ComputeHonestPathTimer(ctx, edgeId, ancestorDetails.AncestorLocalTimers, blockNum)
			if err2 != nil {
				return err2
			}
			ancestors := make([]common.Hash, len(ancestorDetails.AncestorEdgeIds))
			for i := range ancestorDetails.AncestorEdgeIds {
				ancestors[i] = ancestorDetails.AncestorEdgeIds[i].Hash
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
					Id:                  edgeId.Hash,
					ChallengeLevel:      uint8(edge.GetChallengeLevel()),
					StartHistoryRoot:    startRoot,
					StartHeight:         uint64(start),
					EndHeight:           uint64(end),
					EndHistoryRoot:      endRoot,
					CreatedAtBlock:      createdAt,
					MutualId:            common.Hash(edge.MutualId()),
					OriginId:            common.Hash(edge.OriginId()),
					ClaimId:             claimId,
					HasRival:            hasRival,
					CumulativePathTimer: uint64(pathTimer),
					TimeUnrivaled:       timeUnrivaled,
					Ancestors:           ancestors,
					MiniStaker:          miniStaker,
				},
			)
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return response, nil
}

// GetHonestEdges returns all edges in the watcher.
func (w *Watcher) GetHonestEdges() []protocol.SpecEdge {
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

func (w *Watcher) GetHonestConfirmableEdges(ctx context.Context) (map[string][]protocol.SpecEdge, error) {
	honestEdges := w.GetHonestEdges()
	confirmableEdges := make(map[string][]protocol.SpecEdge)
	confirmableEdges[ConfirmableByChildren] = make([]protocol.SpecEdge, 0)
	confirmableEdges[ConfirmableByClaim] = make([]protocol.SpecEdge, 0)
	confirmableEdges[ConfirmableByTimer] = make([]protocol.SpecEdge, 0)
	confirmableEdges[ConfirmableByOSP] = make([]protocol.SpecEdge, 0)
	for _, honestEdge := range honestEdges {
		status, err := honestEdge.Status(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get edge status")
		}
		if status == protocol.EdgeConfirmed {
			continue
		}

		// Check if we can confirm by one step proof.
		canOsp, err := edgetracker.CanOneStepProve(ctx, honestEdge)
		if err != nil {
			return nil, errors.Wrap(err, "could not check if edge can be one step proven")
		}
		if canOsp {
			confirmableEdges[ConfirmableByOSP] = append(confirmableEdges[ConfirmableByOSP], honestEdge)
			continue
		}

		hasConfirmedRival, err := honestEdge.HasConfirmedRival(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not check if edge has confirmed rival")
		}
		if hasConfirmedRival {
			// Cannot be confirmed if it has a confirmed rival edge.
			continue
		}

		assertionHash, err := honestEdge.AssertionHash(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get prev assertion hash")
		}
		manager, err := w.chain.SpecChallengeManager(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get challenge manager")
		}

		// Check if we can confirm by children.
		childrenConfirmed, err := edgetracker.ChildrenAreConfirmed(ctx, honestEdge, manager)
		if err != nil {
			return nil, errors.Wrap(err, "could not check if children are confirmed")
		}
		if childrenConfirmed {
			confirmableEdges[ConfirmableByChildren] = append(confirmableEdges[ConfirmableByChildren], honestEdge)
			continue
		}

		// Check if we can confirm by claim.
		_, ok := w.ConfirmedEdgeWithClaimExists(
			assertionHash,
			protocol.ClaimId(honestEdge.Id().Hash),
		)
		if ok {
			confirmableEdges[ConfirmableByClaim] = append(confirmableEdges[ConfirmableByClaim], honestEdge)
			continue
		}

		// Check if we can confirm by time.
		timer, _, _, err := w.ComputeHonestPathTimer(ctx, assertionHash, honestEdge.Id())
		if err != nil {
			if errors.Is(err, challengetree.ErrNoLowerChildYet) {
				continue
			}
			return nil, errors.Wrap(err, "could not compute honest path timer")
		}
		chalPeriod, err := manager.ChallengePeriodBlocks(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not check the challenge period length")
		}
		if timer >= challengetree.PathTimer(chalPeriod) {
			confirmableEdges[ConfirmableByTimer] = append(confirmableEdges[ConfirmableByTimer], honestEdge)
			continue
		}
	}
	return confirmableEdges, nil
}

func (w *Watcher) GetEvilConfirmedEdges(ctx context.Context) ([]protocol.SpecEdge, error) {
	edges, err := w.GetEdges(ctx)
	if err != nil {
		return nil, err
	}
	honestEdges := w.GetHonestEdges()
	honestEdgesMap := make(map[common.Hash]protocol.SpecEdge)
	for _, honestEdge := range honestEdges {
		honestEdgesMap[honestEdge.Id().Hash] = honestEdge
	}
	evilEdges := make([]protocol.SpecEdge, 0)
	for _, edge := range edges {
		if _, ok := honestEdgesMap[edge.Id().Hash]; !ok {
			evilEdges = append(evilEdges, edge)
		}
	}
	evilConfirmedEdges := make([]protocol.SpecEdge, 0)
	for _, evilEdge := range evilEdges {
		status, err := evilEdge.Status(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get edge status")
		}
		if status == protocol.EdgeConfirmed {
			evilConfirmedEdges = append(evilConfirmedEdges, evilEdge)
		}
	}
	return evilConfirmedEdges, nil
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
	log.Info("Adding verified honest edge to honest edge tree", fields)
	if err := chal.honestEdgeTree.AddRoyalEdge(edge); err != nil {
		log.Error("Could not add verified royal edge to local tree", log.Ctx{"error": err})
		return errors.Wrap(err, "could not add honest edge to challenge tree")
	}
	return w.saveEdgeToDB(ctx, edge, true /* is royal */)
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
		"edgeId":         edge.Id().Hash,
		"challengeLevel": edge.GetChallengeLevel(),
		"assertionHash":  challengeParentAssertionHash.Hash,
		"startHeight":    start,
		"endHeight":      end,
		"startRoot":      startRoot,
		"endRoot":        endRoot,
		"isRoyal":        isRoyalEdge,
	}
	log.Info("Observed edge from onchain event", fields)
	return true, w.saveEdgeToDB(ctx, edge, isRoyalEdge)
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
		if confirmAssertionErr := w.chain.ConfirmAssertionByChallengeWinner(ctx, protocol.AssertionHash{Hash: common.Hash(claimId)}, edgeId); confirmAssertionErr != nil {
			return confirmAssertionErr
		}
		srvlog.Info("Assertion confirmed by challenge win", log.Ctx{
			"challengeParentAssertionHash": containers.Trunc(challengeParentAssertionHash.Bytes()),
		})
	}

	chal.confirmedLevelZeroEdgeClaimIds.Put(claimId, edge.Id())
	w.challenges.Put(challengeParentAssertionHash, chal)
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
	firstBlock := latestConfirmed.CreatedAtBlock()
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
	var pathTimer uint64
	var rawAncestors string
	if isRoyal {
		timer, ancestors, _, err2 := w.ComputeHonestPathTimer(ctx, assertionHash, edge.Id())
		if err2 != nil {
			return err2
		}
		pathTimer = uint64(timer)
		for i, an := range ancestors {
			rawAncestors += an.Hex()
			if i != len(ancestors)-1 {
				rawAncestors += ","
			}
		}
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
		CumulativePathTimer: pathTimer,
		TimeUnrivaled:       timeUnrivaled,
		HasRival:            hasRival,
		HasLengthOneRival:   hasLengthOneRival,
		RawAncestors:        rawAncestors,
	})
}
