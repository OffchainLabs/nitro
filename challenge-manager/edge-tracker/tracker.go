// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

// Package edgetracker contains the logic for tracking an edge in the challenge manager. It keeps
// track of edges created and their own state transitions until an eventual confirmation.
package edgetracker

import (
	"context"
	"fmt"
	"os"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"
	"github.com/OffchainLabs/bold/containers"
	"github.com/OffchainLabs/bold/containers/fsm"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/math"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	utilTime "github.com/OffchainLabs/bold/time"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pkg/errors"
)

var (
	srvlog               = log.New("service", "edge-tracker")
	errBadOneStepProof   = errors.New("bad one step proof data")
	errNotYetConfirmable = errors.New("edge is not yet confirmable")
	spawnedCounter       = metrics.NewRegisteredCounter("arb/validator/tracker/spawned", nil)
	bisectedCounter      = metrics.NewRegisteredCounter("arb/validator/tracker/bisected", nil)
	confirmedCounter     = metrics.NewRegisteredCounter("arb/validator/tracker/confirmed", nil)
	layerZeroLeafCounter = metrics.NewRegisteredCounter("arb/validator/tracker/layer_zero_leaves", nil)
)

func init() {
	srvlog.SetHandler(log.StreamHandler(os.Stdout, log.LogfmtFormat()))
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
	AddVerifiedHonestEdge(
		ctx context.Context, verifiedHonest protocol.VerifiedHonestEdge,
	) error
}

type ChallengeTracker interface {
	IsTrackingEdge(protocol.EdgeId) bool
	MarkTrackedEdge(protocol.EdgeId)
}

// AssociatedAssertionMetadata for the tracked edge.
type AssociatedAssertionMetadata struct {
	FromBatch      l2stateprovider.Batch
	ToBatch        l2stateprovider.Batch
	WasmModuleRoot common.Hash
}

type Opt func(et *Tracker)

// WithActInterval sets the duration between actions. The default is one second.
func WithActInterval(d time.Duration) Opt {
	return func(et *Tracker) {
		et.actInterval = d
	}
}

// WithTimeReference allows setting the timer used by the tracker to determine that time
// passed in accordance with the act interval set with [WithActInterval]. The default is
// to use [github.com/offchainlabs/bold/time.NewRealTimeReference].
// This is useful for testing with a fake time reference to avoid waiting for real time.
func WithTimeReference(ref utilTime.Reference) Opt {
	return func(et *Tracker) {
		et.timeRef = ref
	}
}

// WithValidatorName associates a name to the running validator. This name is used only for logging
// and is not exposed externally. This is particularly useful for debugging purposes.
func WithValidatorName(name string) Opt {
	return func(et *Tracker) {
		et.validatorName = name
	}
}

// WithFSMOpts sets any FSM options to be used when creating the tracker's FSM.
func WithFSMOpts(opts ...fsm.Opt[edgeTrackerAction, State]) Opt {
	return func(et *Tracker) {
		et.fsmOpts = opts
	}
}

type Tracker struct {
	edge                        protocol.SpecEdge
	fsm                         *fsm.Fsm[edgeTrackerAction, State]
	fsmOpts                     []fsm.Opt[edgeTrackerAction, State]
	actInterval                 time.Duration
	timeRef                     utilTime.Reference
	validatorName               string
	chain                       protocol.Protocol
	stateProvider               l2stateprovider.Provider
	chainWatcher                ConfirmationMetadataChecker
	challengeManager            ChallengeTracker
	associatedAssertionMetadata *AssociatedAssertionMetadata
}

func New(
	ctx context.Context,
	edge protocol.SpecEdge,
	chain protocol.Protocol,
	stateProvider l2stateprovider.Provider,
	chainWatcher ConfirmationMetadataChecker,
	challengeManager ChallengeTracker,
	assertionCreationInfo *AssociatedAssertionMetadata,
	opts ...Opt,
) (*Tracker, error) {
	tr := &Tracker{
		edge:                        edge,
		chain:                       chain,
		stateProvider:               stateProvider,
		chainWatcher:                chainWatcher,
		challengeManager:            challengeManager,
		associatedAssertionMetadata: assertionCreationInfo,
		actInterval:                 time.Second,
		timeRef:                     utilTime.NewRealTimeReference(),
	}
	for _, o := range opts {
		o(tr)
	}
	if tr.actInterval == 0 {
		return nil, errors.New("edge tracker act interval must be greater than 0")
	}
	fsm, err := newEdgeTrackerFsm(
		EdgeStarted,
		tr.fsmOpts...,
	)
	if err != nil {
		return nil, err
	}
	tr.fsm = fsm
	return tr, nil
}

func (et *Tracker) AssertionInfo() *AssociatedAssertionMetadata {
	return et.associatedAssertionMetadata
}

func (et *Tracker) EdgeId() protocol.EdgeId {
	return et.edge.Id()
}

func (et *Tracker) Watcher() ConfirmationMetadataChecker {
	return et.chainWatcher
}

func (et *Tracker) ChallengeManager() ChallengeTracker {
	return et.challengeManager
}

func (et *Tracker) Spawn(ctx context.Context) {
	// No-op if we are already tracking this edge in our challenge manager.
	if et.challengeManager.IsTrackingEdge(et.edge.Id()) {
		return
	}
	fields := et.uniqueTrackerLogFields()
	srvlog.Info("Tracking edge", fields)
	spawnedCounter.Inc(1)
	et.challengeManager.MarkTrackedEdge(et.edge.Id())
	t := et.timeRef.NewTicker(et.actInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			if et.ShouldDespawn(ctx) {
				srvlog.Info("Tracked edge received notice it should exit - now despawning", fields)
				spawnedCounter.Dec(1)
				return
			}
			if err := et.Act(ctx); err != nil {
				fields["err"] = err
				srvlog.Error("Could not act with edge tracker", fields)
			}
		case <-ctx.Done():
			srvlog.Debug("Edge tracker goroutine exiting", fields)
			spawnedCounter.Dec(1)
			return
		}
	}
}

func (et *Tracker) CurrentState() State {
	return et.fsm.Current().State
}

func (et *Tracker) Act(ctx context.Context) error {
	fields := et.uniqueTrackerLogFields()
	current := et.fsm.Current()
	switch current.State {
	// Start state.
	case EdgeStarted:
		canOsp, err := CanOneStepProve(ctx, et.edge)
		if err != nil {
			fields["err"] = err
			srvlog.Error("Could not check if edge can be one step proven", fields)
			return et.fsm.Do(edgeBackToStart{})
		}
		if canOsp {
			return et.fsm.Do(edgeHandleOneStepProof{})
		}
		wasConfirmed, err := et.tryToConfirm(ctx)
		if err != nil {
			if !errors.Is(err, errNotYetConfirmable) {
				fields["err"] = err
				srvlog.Error("Could not check if edge can be confirmed", fields)
			}
		}
		if wasConfirmed {
			return et.fsm.Do(edgeConfirm{})
		}
		hasRival, err := et.edge.HasRival(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check presumptive")
		}
		if !hasRival {
			return et.fsm.Do(edgeBackToStart{})
		}
		atOneStepFork, err := et.edge.HasLengthOneRival(ctx)
		if err != nil {
			fields["err"] = err
			srvlog.Error("Could not check if edge has length one rival", fields)
			return et.fsm.Do(edgeBackToStart{})
		}
		if atOneStepFork {
			return et.fsm.Do(edgeOpenSubchallengeLeaf{})
		}
		return et.fsm.Do(edgeBisect{})
	// Edge is at a one-step-proof in a small-step challenge.
	case EdgeAtOneStepProof:
		if err := et.submitOneStepProof(ctx); err != nil {
			fields["err"] = err
			srvlog.Trace("Could not submit one step proof", fields)
			return et.fsm.Do(edgeBackToStart{})
		}
		return et.fsm.Do(edgeConfirm{})
	// Edge tracker should add a subchallenge level zero leaf.
	case EdgeAddingSubchallengeLeaf:
		if err := et.openSubchallengeLeaf(ctx); err != nil {
			fields["err"] = err
			srvlog.Error("Could not open subchallenge leaf", fields)
			return et.fsm.Do(edgeBackToStart{})
		}
		layerZeroLeafCounter.Inc(1)
		return et.fsm.Do(edgeAwaitConfirmation{})
	// Edge should bisect.
	case EdgeBisecting:
		lowerChild, upperChild, err := et.bisect(ctx)
		if err != nil {
			fields["err"] = err
			srvlog.Error("Could not bisect", fields)
			return et.fsm.Do(edgeBackToStart{})
		}
		bisectedCounter.Inc(1)

		firstTracker, err := New(
			ctx,
			lowerChild,
			et.chain,
			et.stateProvider,
			et.chainWatcher,
			et.challengeManager,
			et.associatedAssertionMetadata,
			WithActInterval(et.actInterval),
			WithTimeReference(et.timeRef),
			WithValidatorName(et.validatorName),
			WithFSMOpts(et.fsmOpts...),
		)
		if err != nil {
			fields["err"] = err
			srvlog.Error("Could not create new edge tracker", fields)
			return et.fsm.Do(edgeBackToStart{})
		}
		secondTracker, err := New(
			ctx,
			upperChild,
			et.chain,
			et.stateProvider,
			et.chainWatcher,
			et.challengeManager,
			et.associatedAssertionMetadata,
			WithActInterval(et.actInterval),
			WithTimeReference(et.timeRef),
			WithValidatorName(et.validatorName),
			WithFSMOpts(et.fsmOpts...),
		)
		if err != nil {
			fields["err"] = err
			srvlog.Error("Could not create new edge tracker", fields)
			return et.fsm.Do(edgeBackToStart{})
		}
		go firstTracker.Spawn(ctx)
		go secondTracker.Spawn(ctx)
		return et.fsm.Do(edgeAwaitConfirmation{})
	case EdgeConfirming:
		wasConfirmed, err := et.tryToConfirm(ctx)
		if err != nil {
			if !errors.Is(err, errNotYetConfirmable) {
				fields["err"] = err
				srvlog.Error("Could not check if edge can be confirmed", fields)
			}
		}
		if !wasConfirmed {
			return et.fsm.Do(edgeAwaitConfirmation{})
		}
		return et.fsm.Do(edgeConfirm{})
	case EdgeConfirmed:
		srvlog.Info("Edge reached confirmed state", fields)
		return et.fsm.Do(edgeConfirm{})
	default:
		return fmt.Errorf("invalid state: %s", current.State)
	}
}

// ShouldDespawn checks if an edge tracker should despawn and no longer act.
// This is true if the edge's FSM state is the confirmed state or if
// the edge has a confirmable ancestor by time.
func (et *Tracker) ShouldDespawn(ctx context.Context) bool {
	if et.fsm.Current().State == EdgeConfirmed {
		return true
	}
	fields := et.uniqueTrackerLogFields()
	hasConfirmedRival, err := et.edge.HasConfirmedRival(ctx)
	if err != nil {
		fields["err"] = err
		srvlog.Error("Could not check if edge has a confirmed rival", fields)
		return false
	}
	if hasConfirmedRival {
		// Cannot be confirmed if it has a confirmed rival edge. We should despawn the edge.
		srvlog.Info("Edge has a confirmed rival, edge tracker will now despawn")
		return true
	}
	assertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		fields["err"] = err
		srvlog.Error("Could not get assertion hash", fields)
		return false
	}
	_, _, ancestorLocalTimers, err := et.chainWatcher.ComputeHonestPathTimer(ctx, assertionHash, et.edge.Id())
	if err != nil {
		if errors.Is(err, challengetree.ErrNoLowerChildYet) {
			srvlog.Info(
				"Edge does not yet have a child, perhaps its creation event is still being processed",
				et.uniqueTrackerLogFields(),
			)
			return false
		}
		fields["err"] = err
		srvlog.Error("Could not compute honest path timer", fields)
		return false
	}
	chalManager, err := et.chain.SpecChallengeManager(ctx)
	if err != nil {
		fields["err"] = err
		srvlog.Error("Could not get challenge manager", fields)
		return false
	}
	challengePeriodBlocks, err := chalManager.ChallengePeriodBlocks(ctx)
	if err != nil {
		fields["err"] = err
		srvlog.Error("Could not get challenge period blocks", fields)
		return false
	}
	hasConfirmableAncestor, err := et.chainWatcher.HasConfirmableAncestor(
		ctx,
		assertionHash,
		ancestorLocalTimers,
		challengePeriodBlocks,
	)
	if err != nil {
		fields["err"] = err
		srvlog.Error("Could not check if has confirmable ancestor", fields)
		return false
	}
	if hasConfirmableAncestor {
		srvlog.Info("Edge has confirmable ancestor - challenge manager will stop tracking it", fields)
		return true
	}
	return false
}

func (et *Tracker) uniqueTrackerLogFields() log.Ctx {
	startHeight, startCommit := et.edge.StartCommitment()
	endHeight, endCommit := et.edge.EndCommitment()
	chalLevel := et.edge.GetChallengeLevel()
	return log.Ctx{
		"id":            containers.Trunc(et.edge.Id().Bytes()),
		"fromBatch":     et.associatedAssertionMetadata.FromBatch,
		"toBatch":       et.associatedAssertionMetadata.ToBatch,
		"startHeight":   startHeight,
		"startCommit":   containers.Trunc(startCommit.Bytes()),
		"endHeight":     endHeight,
		"endCommit":     containers.Trunc(endCommit.Bytes()),
		"validatorName": et.validatorName,
		"challengeType": chalLevel.String(),
	}
}

func ChildrenAreConfirmed(
	ctx context.Context,
	edge protocol.SpecEdge,
	chalManager protocol.SpecChallengeManager,
) (bool, error) {
	lower, err := edge.LowerChild(ctx)
	if err != nil {
		return false, err
	}
	upper, err := edge.UpperChild(ctx)
	if err != nil {
		return false, err
	}
	if lower.IsNone() || upper.IsNone() {
		return false, nil
	}
	someLowerEdge, err := chalManager.GetEdge(ctx, lower.Unwrap())
	if err != nil {
		return false, err
	}
	someUpperEdge, err := chalManager.GetEdge(ctx, upper.Unwrap())
	if err != nil {
		return false, err
	}
	if someLowerEdge.IsNone() || someUpperEdge.IsNone() {
		return false, nil
	}
	lowerStatus, err := someLowerEdge.Unwrap().Status(ctx)
	if err != nil {
		return false, err
	}
	upperStatus, err := someUpperEdge.Unwrap().Status(ctx)
	if err != nil {
		return false, err
	}
	return lowerStatus == protocol.EdgeConfirmed && upperStatus == protocol.EdgeConfirmed, nil
}

func (et *Tracker) tryToConfirm(ctx context.Context) (bool, error) {
	status, err := et.edge.Status(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not get edge status")
	}
	if status == protocol.EdgeConfirmed {
		return true, nil
	}

	hasConfirmedRival, err := et.edge.HasConfirmedRival(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not check if edge has confirmed rival")
	}
	if hasConfirmedRival {
		// Cannot be confirmed if it has a confirmed rival edge.
		return false, nil
	}

	assertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not get prev assertion hash")
	}
	manager, err := et.chain.SpecChallengeManager(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not get challenge manager")
	}

	// Check if we can confirm by children.
	childrenConfirmed, err := ChildrenAreConfirmed(ctx, et.edge, manager)
	if err != nil {
		return false, errors.Wrap(err, "could not check if children are confirmed")
	}
	if childrenConfirmed {
		if confirmErr := et.edge.ConfirmByChildren(ctx); confirmErr != nil {
			return false, errors.Wrap(confirmErr, "could not confirm by children")
		}
		srvlog.Info("Confirmed by children", et.uniqueTrackerLogFields())
		confirmedCounter.Inc(1)
		return true, nil
	}

	// Check if we can confirm by claim.
	claimingEdge, ok := et.chainWatcher.ConfirmedEdgeWithClaimExists(
		assertionHash,
		protocol.ClaimId(et.edge.Id().Hash),
	)
	if ok {
		if confirmClaimErr := et.edge.ConfirmByClaim(ctx, protocol.ClaimId(claimingEdge.Hash)); confirmClaimErr != nil {
			return false, errors.Wrap(confirmClaimErr, "could not confirm by claim")
		}
		srvlog.Info("Confirmed by claim", et.uniqueTrackerLogFields())
		confirmedCounter.Inc(1)
		return true, nil
	}

	// Check if we can confirm by time.
	timer, ancestors, _, err := et.chainWatcher.ComputeHonestPathTimer(ctx, assertionHash, et.edge.Id())
	if err != nil {
		if errors.Is(err, challengetree.ErrNoLowerChildYet) {
			srvlog.Info(
				"Edge does not yet have a child, perhaps its creation event is still being processed",
				et.uniqueTrackerLogFields(),
			)
			return false, nil
		}
		return false, errors.Wrap(err, "could not compute honest path timer")
	}
	chalPeriod, err := manager.ChallengePeriodBlocks(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not check the challenge period length")
	}
	if timer >= challengetree.PathTimer(chalPeriod) {
		if err := et.edge.ConfirmByTimer(ctx, ancestors); err != nil {
			return false, errors.Wrapf(err, "could not confirm by timer: got timer %d, chal period %d", timer, chalPeriod)
		}
		srvlog.Info("Confirmed by time", et.uniqueTrackerLogFields())
		confirmedCounter.Inc(1)
		return true, nil
	}
	return false, errNotYetConfirmable
}

// Determines the bisection point from parentHeight to toHeight and returns a history
// commitment with a prefix proof for the action based on the challenge type.
func (et *Tracker) DetermineBisectionHistoryWithProof(
	ctx context.Context,
) (commitments.History, []byte, error) {
	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.EndCommitment()
	bisectTo, err := math.Bisect(uint64(startHeight), uint64(endHeight))
	if err != nil {
		return commitments.History{}, nil, errors.Wrapf(err, "determining bisection point errored for %d and %d", startHeight, endHeight)
	}
	challengeLevel := et.edge.GetChallengeLevel()
	if challengeLevel == protocol.NewBlockChallengeLevel() {
		historyCommit, commitErr := et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
				FromBatch:                   et.associatedAssertionMetadata.FromBatch,
				ToBatch:                     et.associatedAssertionMetadata.ToBatch,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(bisectTo)),
			},
		)
		if commitErr != nil {
			return commitments.History{}, nil, commitErr
		}
		proof, proofErr := et.stateProvider.PrefixProof(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
				FromBatch:                   et.associatedAssertionMetadata.FromBatch,
				ToBatch:                     et.associatedAssertionMetadata.ToBatch,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
			},
			l2stateprovider.Height(bisectTo),
		)
		if proofErr != nil {
			return commitments.History{}, nil, proofErr
		}
		return historyCommit, proof, nil
	}
	var historyCommit commitments.History
	var commitErr error
	var proof []byte
	var proofErr error

	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return commitments.History{}, nil, err
	}
	challengeOriginHeights := make([]l2stateprovider.Height, len(originHeights.ChallengeOriginHeights))
	for index, height := range originHeights.ChallengeOriginHeights {
		challengeOriginHeights[index] = l2stateprovider.Height(height)
	}
	// The first challenge origin height must account for the start block height of the assertion.
	historyCommit, commitErr = et.stateProvider.HistoryCommitment(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
			FromBatch:                   et.associatedAssertionMetadata.FromBatch,
			ToBatch:                     et.associatedAssertionMetadata.ToBatch,
			UpperChallengeOriginHeights: challengeOriginHeights,
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.Some(l2stateprovider.Height(bisectTo)),
		},
	)
	if commitErr != nil {
		return commitments.History{}, nil, errors.Wrap(commitErr, "could not produce history commitment")
	}
	proof, proofErr = et.stateProvider.PrefixProof(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
			FromBatch:                   et.associatedAssertionMetadata.FromBatch,
			ToBatch:                     et.associatedAssertionMetadata.ToBatch,
			UpperChallengeOriginHeights: challengeOriginHeights,
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
		},
		l2stateprovider.Height(bisectTo),
	)
	if proofErr != nil {
		return commitments.History{}, nil, errors.Wrap(proofErr, "could not produce prefix proof")
	}
	return historyCommit, proof, nil
}

func (et *Tracker) bisect(ctx context.Context) (protocol.SpecEdge, protocol.SpecEdge, error) {
	historyCommit, proof, err := et.DetermineBisectionHistoryWithProof(ctx)
	if err != nil {
		return nil, nil, err
	}
	endHeight, endCommit := et.edge.EndCommitment()
	bisectTo := historyCommit.Height
	firstChild, secondChild, err := et.edge.Bisect(ctx, historyCommit.Merkle, proof)
	if err != nil {
		return nil, nil, errors.Wrapf(
			err,
			"%s could not bisect to height=%d,commit=%s from height=%d,commit=%s",
			et.validatorName,
			bisectTo,
			containers.Trunc(historyCommit.Merkle.Bytes()),
			endHeight,
			containers.Trunc(endCommit.Bytes()),
		)
	}
	challengeLevel := et.edge.GetChallengeLevel()
	srvlog.Info("Successfully bisected edge", log.Ctx{
		"name":               et.validatorName,
		"challengeType":      challengeLevel,
		"bisectedFrom":       endHeight,
		"bisectedFromMerkle": containers.Trunc(endCommit.Bytes()),
		"bisectedTo":         bisectTo,
		"bisectedToMerkle":   containers.Trunc(historyCommit.Merkle.Bytes()),
	})
	if addVerifiedErr := et.chainWatcher.AddVerifiedHonestEdge(ctx, firstChild); addVerifiedErr != nil {
		// We simply log an error, as if this errored, it will be added later on by the chain watcher
		// scraping events from the chain, but this is a helpful optimization.
		srvlog.Error("Could not add verified honest edge to chain watcher", log.Ctx{"err": addVerifiedErr})
	}
	if addVerifiedErr := et.chainWatcher.AddVerifiedHonestEdge(ctx, secondChild); addVerifiedErr != nil {
		srvlog.Error("Could not add verified honest edge to chain watcher", log.Ctx{"err": addVerifiedErr})
	}
	return firstChild, secondChild, nil
}

func (et *Tracker) openSubchallengeLeaf(ctx context.Context) error {
	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get top level claim height")
	}

	fromBlockChallengeHeight := l2stateprovider.Height(originHeights.ChallengeOriginHeights[0])

	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.EndCommitment()

	fields := log.Ctx{
		"name":                     et.validatorName,
		"edgeStartHeight":          startHeight,
		"edgeEndHeight":            endHeight,
		"fromBlockChallengeHeight": fromBlockChallengeHeight,
	}

	var startHistory commitments.History
	var endHistory commitments.History
	var startParentCommitment commitments.History
	var endParentCommitment commitments.History
	var startEndPrefixProof []byte
	challengeLevel := et.edge.GetChallengeLevel()
	switch challengeLevel {
	case protocol.NewBlockChallengeLevel():
		endHistory, err = et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
				FromBatch:                   et.associatedAssertionMetadata.FromBatch,
				ToBatch:                     et.associatedAssertionMetadata.ToBatch,
				UpperChallengeOriginHeights: []l2stateprovider.Height{fromBlockChallengeHeight},
				FromHeight:                  l2stateprovider.Height(0),
				UpToHeight:                  option.None[l2stateprovider.Height](),
			},
		)
		if err != nil {
			return errors.Wrap(err, "could not compute end history commitment")
		}
		startEndPrefixProof, err = et.stateProvider.PrefixProof(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
				FromBatch:                   et.associatedAssertionMetadata.FromBatch,
				ToBatch:                     et.associatedAssertionMetadata.ToBatch,
				UpperChallengeOriginHeights: []l2stateprovider.Height{fromBlockChallengeHeight},
				FromHeight:                  l2stateprovider.Height(0),
				UpToHeight:                  option.Some(l2stateprovider.Height(endHistory.Height)),
			},
			l2stateprovider.Height(0),
		)
		if err != nil {
			return errors.Wrap(err, "could not compute prefix proof")
		}
		startHistory, err = et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
				FromBatch:                   et.associatedAssertionMetadata.FromBatch,
				ToBatch:                     et.associatedAssertionMetadata.ToBatch,
				UpperChallengeOriginHeights: []l2stateprovider.Height{fromBlockChallengeHeight},
				FromHeight:                  l2stateprovider.Height(0),
				UpToHeight:                  option.Some(l2stateprovider.Height(0)),
			},
		)
		if err != nil {
			return err
		}
		endParentCommitment, err = et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
				FromBatch:                   et.associatedAssertionMetadata.FromBatch,
				ToBatch:                     et.associatedAssertionMetadata.ToBatch,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(fromBlockChallengeHeight + 1),
			},
		)
		if err != nil {
			return err
		}
		startParentCommitment, err = et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
				FromBatch:                   et.associatedAssertionMetadata.FromBatch,
				ToBatch:                     et.associatedAssertionMetadata.ToBatch,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				FromHeight:                  0,
				UpToHeight:                  option.Some(fromBlockChallengeHeight),
			},
		)
		if err != nil {
			return err
		}
	default:
		heights := make([]l2stateprovider.Height, 0)
		for _, h := range originHeights.ChallengeOriginHeights {
			heights = append(heights, l2stateprovider.Height(h))
		}
		heights = append(heights, l2stateprovider.Height(startHeight))
		request := &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
			FromBatch:                   et.associatedAssertionMetadata.FromBatch,
			ToBatch:                     et.associatedAssertionMetadata.ToBatch,
			UpperChallengeOriginHeights: heights,
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.None[l2stateprovider.Height](),
		}
		endHistory, err = et.stateProvider.HistoryCommitment(
			ctx,
			request,
		)
		if err != nil {
			return errors.Wrapf(err, "could not compute child commitment with request %+v", request)
		}
		request = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
			FromBatch:                   et.associatedAssertionMetadata.FromBatch,
			ToBatch:                     et.associatedAssertionMetadata.ToBatch,
			UpperChallengeOriginHeights: heights,
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.Some(l2stateprovider.Height(endHistory.Height)),
		}
		startEndPrefixProof, err = et.stateProvider.PrefixProof(
			ctx,
			request,
			l2stateprovider.Height(0),
		)
		if err != nil {
			return errors.Wrapf(err, "could not compute prefix proof for child with request %+v, up to height %d", request, endHistory.Height)
		}
		request = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
			FromBatch:                   et.associatedAssertionMetadata.FromBatch,
			ToBatch:                     et.associatedAssertionMetadata.ToBatch,
			UpperChallengeOriginHeights: heights,
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		}
		startHistory, err = et.stateProvider.HistoryCommitment(
			ctx,
			request,
		)
		if err != nil {
			return errors.Wrapf(err, "could not compute start history commitment with request %+v", request)
		}
		request = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
			FromBatch:                   et.associatedAssertionMetadata.FromBatch,
			ToBatch:                     et.associatedAssertionMetadata.ToBatch,
			UpperChallengeOriginHeights: heights[:len(heights)-1],
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
		}
		endParentCommitment, err = et.stateProvider.HistoryCommitment(
			ctx,
			request,
		)
		if err != nil {
			return errors.Wrapf(err, "could not compute end parent commitment with request %+v, end height %d", request, endHeight)
		}
		request = &l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              et.associatedAssertionMetadata.WasmModuleRoot,
			FromBatch:                   et.associatedAssertionMetadata.FromBatch,
			ToBatch:                     et.associatedAssertionMetadata.ToBatch,
			UpperChallengeOriginHeights: heights[:len(heights)-1],
			FromHeight:                  l2stateprovider.Height(0),
			UpToHeight:                  option.Some(l2stateprovider.Height(startHeight)),
		}
		startParentCommitment, err = et.stateProvider.HistoryCommitment(
			ctx,
			request,
		)
		if err != nil {
			return errors.Wrapf(err, "could not compute start parent commitment with request %+v, start height %d", request, startHeight)
		}
	}
	fields["firstLeaf"] = containers.Trunc(startHistory.LastLeaf.Bytes())
	fields["lastLeaf"] = containers.Trunc(endHistory.LastLeaf.Bytes())
	fields["parentFirstLeaf"] = containers.Trunc(startParentCommitment.LastLeaf.Bytes())
	fields["parentLastLeaf"] = containers.Trunc(endParentCommitment.LastLeaf.Bytes())
	fields["parentStartHeight"] = startParentCommitment.Height
	fields["parentEndHeight"] = endParentCommitment.Height
	srvlog.Info("Creating subchallenge edge", fields)

	manager, err := et.chain.SpecChallengeManager(ctx)
	if err != nil {
		return err
	}
	addedLeaf, err := manager.AddSubChallengeLevelZeroEdge(
		ctx,
		et.edge,
		startHistory,
		endHistory,
		startParentCommitment.LastLeafProof,
		endParentCommitment.LastLeafProof,
		startEndPrefixProof,
	)
	if err != nil {
		return err
	}
	fields["firstLeaf"] = containers.Trunc(startHistory.FirstLeaf.Bytes())
	fields["startCommitment"] = containers.Trunc(startHistory.Merkle.Bytes())
	addedLeafChallengeLevel := addedLeaf.GetChallengeLevel()
	fields["subChallengeType"] = addedLeafChallengeLevel
	srvlog.Info("Created subchallenge edge", fields)

	if addVerifiedErr := et.chainWatcher.AddVerifiedHonestEdge(ctx, addedLeaf); addVerifiedErr != nil {
		// We simply log an error, as if this errored, it will be added later on by the chain watcher
		// scraping events from the chain, but this is a helpful optimization.
		srvlog.Error("Could not add verified honest edge to chain watcher", log.Ctx{"err": addVerifiedErr})
	}

	tracker, err := New(
		ctx,
		addedLeaf,
		et.chain,
		et.stateProvider,
		et.chainWatcher,
		et.challengeManager,
		et.associatedAssertionMetadata,
		WithActInterval(et.actInterval),
		WithTimeReference(et.timeRef),
		WithValidatorName(et.validatorName),
		WithFSMOpts(et.fsmOpts...),
	)
	if err != nil {
		return err
	}
	go tracker.Spawn(ctx)
	return nil
}

func (et *Tracker) submitOneStepProof(ctx context.Context) error {
	fields := et.uniqueTrackerLogFields()
	srvlog.Info("Submitting one-step-proof to protocol", fields)
	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get top level claim height")
	}
	pc, _ := et.edge.StartCommitment()

	assertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		return err
	}
	parentAssertionCreationInfo, err := et.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return err
	}
	challengeOriginHeights := make([]l2stateprovider.Height, len(originHeights.ChallengeOriginHeights))
	for index, height := range originHeights.ChallengeOriginHeights {
		challengeOriginHeights[index] = l2stateprovider.Height(height)
	}
	data, beforeStateInclusionProof, afterStateInclusionProof, err := et.stateProvider.OneStepProofData(
		ctx,
		parentAssertionCreationInfo.WasmModuleRoot,
		et.associatedAssertionMetadata.FromBatch,
		et.associatedAssertionMetadata.ToBatch,
		challengeOriginHeights,
		0,
		l2stateprovider.Height(pc),
	)
	if err != nil {
		return errors.Wrapf(errBadOneStepProof, "could not get one step data: %v", err)
	}
	manager, err := et.chain.SpecChallengeManager(ctx)
	if err != nil {
		return err
	}
	if err = manager.ConfirmEdgeByOneStepProof(
		ctx,
		et.edge.Id(),
		data,
		beforeStateInclusionProof,
		afterStateInclusionProof,
	); err != nil {
		return errors.Wrap(err, "could not confirm one step proof against protocol")
	}
	srvlog.Info("Succeeded one-step-proof for edge and confirmed it as winner", fields)
	return nil
}

func CanOneStepProve(ctx context.Context, edge protocol.SpecEdge) (bool, error) {
	start, _ := edge.StartCommitment()
	end, _ := edge.EndCommitment()
	// Can never happen in the protocol, but added as an additional defensive check.
	if start >= end {
		return false, fmt.Errorf("start height %d cannot be >= end height %d", start, end)
	}
	challengeLevel := edge.GetChallengeLevel()
	totalChallengeLevels := edge.GetTotalChallengeLevels(ctx)
	return end-start == 1 && challengeLevel.Uint8() == totalChallengeLevels-1, nil
}
