// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package edgetracker contains the logic for tracking an edge in the challenge manager. It keeps
// track of edges created and their own state transitions until an eventual confirmation.
package edgetracker

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager/challenge-tree"
	"github.com/offchainlabs/nitro/bold/containers"
	"github.com/offchainlabs/nitro/bold/containers/events"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/math"
	"github.com/offchainlabs/nitro/bold/state-commitments/history"
	utilTime "github.com/offchainlabs/nitro/bold/time"
)

var (
	errBadOneStepProof   = errors.New("bad one step proof data")
	spawnedCounter       = metrics.NewRegisteredCounter("arb/validator/tracker/spawned", nil)
	bisectedCounter      = metrics.NewRegisteredCounter("arb/validator/tracker/bisected", nil)
	layerZeroLeafCounter = metrics.NewRegisteredCounter("arb/validator/tracker/layer_zero_leaves", nil)
)

// HonestChallengeTreeReader defines a type which can retrieve information about
// an edge to determine if it can be confirmed via different means. For example,
// checking if a confirmed edge exists that claims a specified edge id as its claim id,
// or retrieving the cumulative, honest path timer for an edge and its honest ancestors.
// This information is used in order to confirm edges onchain.
type HonestChallengeTreeReader interface {
	LowerMostRoyalEdges(
		ctx context.Context,
		challengedAssertionHash protocol.AssertionHash,
	) ([]protocol.SpecEdge, error)
	ComputeAncestors(
		ctx context.Context,
		challengedAssertionHash protocol.AssertionHash,
		edgeId protocol.EdgeId,
	) ([]protocol.ReadOnlyEdge, error)
	ClosestEssentialAncestor(
		ctx context.Context,
		challengedAssertionHash protocol.AssertionHash,
		edge protocol.VerifiedRoyalEdge,
	) (protocol.ReadOnlyEdge, error)
	IsEssentialAncestorConfirmable(
		ctx context.Context,
		edge protocol.SpecEdge,
		challengedAssertionHash protocol.AssertionHash,
		confirmationThreshold uint64,
	) (bool, error)
	IsConfirmableEssentialEdge(
		ctx context.Context,
		challengedAssertionHash protocol.AssertionHash,
		essentialEdgeId protocol.EdgeId,
		confirmationThreshold uint64,
	) (confirmable bool, essentialPaths []challengetree.EssentialPath, timer uint64, err error)
}

// HonestChallengeTreeWriter defines a type which can not only read information
// about the honest challenge tree, but also add a verified honest edge to the tree.
type HonestChallengeTreeWriter interface {
	HonestChallengeTreeReader
	AddVerifiedHonestEdge(
		ctx context.Context, verifiedHonest protocol.VerifiedRoyalEdge,
	) error
}

// ChallengeTracker defines a type which can keep track of edge spawner goroutines
// and remove them as needed upon confirmation.
type ChallengeTracker interface {
	IsTrackingEdge(protocol.EdgeId) bool
	MarkTrackedEdge(protocol.EdgeId, *Tracker)
	RemovedTrackedEdge(protocol.EdgeId)
	NewBlockSubscriber() *events.Producer[*types.Header]
}

type Opt func(et *Tracker)

// WithTimeReference allows setting the timer used by the tracker to determine that time
// passed in accordance with the act interval set with [WithActInterval]. The default is
// to use [github.com/offchainlabs/nitro/bold/time.NewRealTimeReference].
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
	edge                        protocol.VerifiedRoyalEdge
	fsm                         *fsm.Fsm[edgeTrackerAction, State]
	fsmOpts                     []fsm.Opt[edgeTrackerAction, State]
	timeRef                     utilTime.Reference
	validatorName               string
	chain                       protocol.Protocol
	stateProvider               l2stateprovider.Provider
	chainWatcher                HonestChallengeTreeWriter
	challengeManager            ChallengeTracker
	associatedAssertionMetadata *l2stateprovider.AssociatedAssertionMetadata
	challengeConfirmer          *challengeConfirmer
}

func New(
	ctx context.Context,
	edge protocol.VerifiedRoyalEdge,
	chain protocol.Protocol,
	stateProvider l2stateprovider.Provider,
	chainWatcher HonestChallengeTreeWriter,
	challengeManager ChallengeTracker,
	assertionCreationInfo *l2stateprovider.AssociatedAssertionMetadata,
	opts ...Opt,
) (*Tracker, error) {
	tr := &Tracker{
		edge:                        edge,
		chain:                       chain,
		stateProvider:               stateProvider,
		chainWatcher:                chainWatcher,
		challengeManager:            challengeManager,
		associatedAssertionMetadata: assertionCreationInfo,
		timeRef:                     utilTime.NewRealTimeReference(),
	}
	for _, o := range opts {
		o(tr)
	}
	chalManager := chain.SpecChallengeManager()
	tr.challengeConfirmer = newChallengeConfirmer(chainWatcher, chalManager, chain.Backend(), tr.validatorName, chain)
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

func (et *Tracker) AssertionInfo() *l2stateprovider.AssociatedAssertionMetadata {
	return et.associatedAssertionMetadata
}

func (et *Tracker) EdgeId() protocol.EdgeId {
	return et.edge.Id()
}

func (et *Tracker) ChallengeManager() ChallengeTracker {
	return et.challengeManager
}

type FSMStateSummary struct {
	CurrentState string
	Error        error
}

func (et *Tracker) FSMSummary() *FSMStateSummary {
	curr := et.fsm.Current()
	return &FSMStateSummary{
		CurrentState: curr.State.String(),
		Error:        curr.Error,
	}
}

func (et *Tracker) Spawn(ctx context.Context) {
	// No-op if we are already tracking this edge in our challenge manager.
	if et.challengeManager.IsTrackingEdge(et.edge.Id()) {
		return
	}
	fields := et.uniqueTrackerLogFields()
	log.Info("Now tracking challenge edge locally and making moves", fields...)
	spawnedCounter.Inc(1)
	et.challengeManager.MarkTrackedEdge(et.edge.Id(), et)

	subscription := et.challengeManager.NewBlockSubscriber().Subscribe()
	for {
		_, shouldExit := subscription.Next(ctx)
		if ctx.Err() != nil || shouldExit {
			log.Debug("Edge tracker goroutine exiting", fields...)
			spawnedCounter.Dec(1)
			return
		}
		if et.ShouldDespawn(ctx) {
			log.Info("Tracked edge received notice it should exit - now despawning", fields...)
			spawnedCounter.Dec(1)
			et.challengeManager.RemovedTrackedEdge(et.edge.Id())
			return
		}
		if err := et.Act(ctx); err != nil {
			log.Error("Could not act with edge tracker", append(fields, "err", err)...)
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
		canOsp, err := canOneStepProve(ctx, et.edge)
		if err != nil {
			log.Error("Could not check if edge can be one step proven", append(fields, "err", err)...)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		wasConfirmed, err := et.tryToConfirmEdge(ctx)
		if err != nil {
			log.Error("Could not check if edge can be confirmed from start state", append(fields, "err", err)...)
			et.fsm.MarkError(err)
		}
		if wasConfirmed {
			return et.fsm.Do(edgeAwaitChallengeCompletion{})
		}
		hasRival, err := et.edge.HasRival(ctx)
		if err != nil {
			log.Error("Could not check if edge has rival", append(fields, "err", err)...)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		if !hasRival {
			return et.fsm.Do(edgeBackToStart{})
		}
		if canOsp { // Implicitly, the edge has a rival.
			return et.fsm.Do(edgeHandleOneStepProof{})
		}
		atOneStepFork, err := et.edge.HasLengthOneRival(ctx)
		if err != nil {
			log.Error("Could not check if edge has length one rival", append(fields, "err", err)...)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		if atOneStepFork {
			return et.fsm.Do(edgeOpenSubchallengeLeaf{})
		}
		return et.fsm.Do(edgeBisect{})
	// Edge is at a one-step-proof in a small-step challenge.
	case EdgeAtOneStepProof:
		ok, err := et.isEssentialAncestorConfirmable(ctx)
		if err != nil {
			log.Error("Could not check if closest essential ancestor is confirmable", append(fields, "err", err)...)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		if ok {
			return et.fsm.Do(edgeAwaitChallengeCompletion{})
		}
		if err := et.submitOneStepProof(ctx); err != nil {
			log.Error("Could not submit one step proof", append(fields, "err", err)...)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		return et.fsm.Do(edgeAwaitChallengeCompletion{})
	// Edge tracker should add a subchallenge.
	case EdgeAddingSubchallengeLeaf:
		ok, err := et.isEssentialAncestorConfirmable(ctx)
		if err != nil {
			log.Error("Could not check if closest essential ancestor is confirmable", fields, "err", err)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		if ok {
			return et.fsm.Do(edgeAwaitChallengeCompletion{})
		}
		if err := et.openSubchallenge(ctx); err != nil {
			log.Error("Could not open subchallenge leaf", append(fields, "err", err)...)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		layerZeroLeafCounter.Inc(1)
		return et.fsm.Do(edgeAwaitChallengeCompletion{})
		// Edge should bisect.
	case EdgeBisecting:
		ok, err := et.isEssentialAncestorConfirmable(ctx)
		if err != nil {
			log.Error("Could not check if closest essential ancestor is confirmable", fields, "err", err)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		if ok {
			return et.fsm.Do(edgeAwaitChallengeCompletion{})
		}
		lowerChild, upperChild, err := et.bisect(ctx)
		if err != nil {
			log.Error("Could not bisect", append(fields, "err", err)...)
			et.fsm.MarkError(err)
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
			WithTimeReference(et.timeRef),
			WithValidatorName(et.validatorName),
			WithFSMOpts(et.fsmOpts...),
		)
		if err != nil {
			log.Error("Could not create new edge tracker", append(fields, "err", err)...)
			et.fsm.MarkError(err)
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
			WithTimeReference(et.timeRef),
			WithValidatorName(et.validatorName),
			WithFSMOpts(et.fsmOpts...),
		)
		if err != nil {
			log.Error("Could not create new edge tracker", append(fields, "err", err)...)
			et.fsm.MarkError(err)
			return et.fsm.Do(edgeBackToStart{})
		}
		go firstTracker.Spawn(ctx)
		go secondTracker.Spawn(ctx)
		return et.fsm.Do(edgeAwaitChallengeCompletion{})
	case EdgeAwaitingChallengeCompletion:
		_, err := et.tryToConfirmEdge(ctx)
		if err != nil {
			log.Error("Could not check if edge can be confirmed", append(fields, "err", err)...)
			et.fsm.MarkError(err)
		}
		return et.fsm.Do(edgeAwaitChallengeCompletion{})
	default:
		return fmt.Errorf("invalid state: %s", current.State)
	}
}

// ShouldDespawn checks if an edge tracker should despawn and no longer act.
// Every edge tracker needs to have a despawn condition
// to ensure goroutines are cleaned up.
func (et *Tracker) ShouldDespawn(ctx context.Context) bool {
	// If the edge is an essential root, it should despawn once it is confirmed.
	fields := et.uniqueTrackerLogFields()
	if et.edge.ClaimId().IsSome() {
		status, err := et.edge.Status(ctx)
		if err != nil {
			log.Error("Could not get edge status", append(fields, "err", err)...)
			return false
		}
		return status == protocol.EdgeConfirmed
	}
	// Else if the edge is a NON-essential root:
	canOsp, err := canOneStepProve(ctx, et.edge)
	if err != nil {
		log.Error("Could not check if edge can be one step proven", append(fields, "err", err)...)
		return false
	}
	// If the edge is a small step edge of length one, exit once it is confirmed by OSP.
	if canOsp {
		status, err2 := et.edge.Status(ctx)
		if err2 != nil {
			log.Error("Could not get edge status", append(fields, "err", err2)...)
			return false
		}
		return status == protocol.EdgeConfirmed
	}
	assertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		log.Error("Could not get edge assertion hash", append(fields, "err", err)...)
		return false
	}
	closestEssential, err := et.chainWatcher.ClosestEssentialAncestor(ctx, assertionHash, et.edge)
	if err != nil {
		log.Error("Could not get edge closest essential ancestor", append(fields, "err", err)...)
		return false
	}
	status, err := closestEssential.Status(ctx)
	if err != nil {
		log.Error("Could not get closest essential ancestor status", append(fields, "err", err)...)
		return false
	}
	return status == protocol.EdgeConfirmed
}

func (et *Tracker) uniqueTrackerLogFields() []any {
	startHeight, startCommit := et.edge.StartCommitment()
	endHeight, endCommit := et.edge.EndCommitment()
	chalLevel := et.edge.GetChallengeLevel()
	return []any{
		"id", fmt.Sprintf("%#x", et.edge.Id().Bytes()[:4]),
		"fromBatch", et.associatedAssertionMetadata.FromState.Batch,
		"fromPosInBatch", et.associatedAssertionMetadata.FromState.PosInBatch,
		"batchLimit", et.associatedAssertionMetadata.BatchLimit,
		"claimedAssertionHash", fmt.Sprintf("%#x", et.associatedAssertionMetadata.ClaimedAssertionHash.Hash[:4]),
		"startHeight", startHeight,
		"startCommit", fmt.Sprintf("%#x", startCommit[:4]),
		"endHeight", endHeight,
		"endCommit", fmt.Sprintf("%#x", endCommit[:4]),
		"validatorName", et.validatorName,
		"challengeType", chalLevel.String(),
		"originId", fmt.Sprintf("%#x", common.Hash(et.edge.OriginId()).Bytes()[:4]),
		"mutualId", fmt.Sprintf("%#x", common.Hash(et.edge.MutualId()).Bytes()[:8]),
	}
}

func (et *Tracker) tryToConfirmEdge(ctx context.Context) (bool, error) {
	fields := et.uniqueTrackerLogFields()
	// If the edge is not a root of a challenge or subchallenge, we have nothing to do here.
	if et.edge.ClaimId().IsNone() {
		return false, nil
	}
	status, err := et.edge.Status(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not get edge status")
	}
	if status == protocol.EdgeConfirmed {
		return true, nil
	}
	challengedAssertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		return false, err
	}
	manager := et.chain.SpecChallengeManager()
	chalPeriod := manager.ChallengePeriodBlocks()
	start := time.Now()
	isConfirmable, _, computedTimer, err := et.chainWatcher.IsConfirmableEssentialEdge(
		ctx,
		challengedAssertionHash,
		et.edge.Id(),
		chalPeriod,
	)
	if err != nil {
		// If the error is that the child edges have not yet been observed by our chain watcher,
		// we can simply return false and nil as they will eventually seen. This may occur when the validator
		// is relying on safe or finalized data from the chain watcher.
		if errors.Is(err, challengetree.ErrChildrenNotYetSeen) {
			return false, nil
		}
		return false, errors.Wrap(err, "not check if essential edge is confirmable")
	}
	end := time.Since(start)
	localFields := []any{
		"localTimer", computedTimer,
		"confirmableAfter", chalPeriod,
		"edgeId", fmt.Sprintf("%#x", et.edge.Id().Bytes()[:4]),
		"took", end,
		"batchLimit", et.associatedAssertionMetadata.BatchLimit,
		"claimedAssertion", fmt.Sprintf("%#x", et.associatedAssertionMetadata.ClaimedAssertionHash.Hash[:4]),
	}
	if isConfirmable {
		log.Info("Local computed timer big enough to confirm edge", append(fields, localFields...)...)
		if err := et.challengeConfirmer.beginConfirmationJob(
			ctx,
			challengedAssertionHash,
			computedTimer,
			et.edge,
			et.associatedAssertionMetadata.ClaimedAssertionHash,
			chalPeriod,
		); err != nil {
			log.Error("Could not begin confirmation job", fields...)
			return false, errors.Wrapf(
				err,
				"could not complete confirmation job for essential root edge at level %d",
				et.edge.GetChallengeLevel(),
			)
		}
		// The edge is now confirmed.
		return true, nil
	}
	log.Info("Local computed timer not big enough to confirm edge", append(fields, localFields...)...)
	return false, nil
}

// Checks if the closest essential ancestor of an edge is confirmable. This method is used by the edge
// tracker to determine if it needs to open a subchallenge or bisect. The honest strategy
// aims to avoid unnecessary moves if it can determine they are unnecessary.
func (et *Tracker) isEssentialAncestorConfirmable(ctx context.Context) (bool, error) {
	assertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		return false, err
	}
	manager := et.chain.SpecChallengeManager()
	chalPeriod := manager.ChallengePeriodBlocks()
	return et.chainWatcher.IsEssentialAncestorConfirmable(
		ctx,
		et.edge,
		assertionHash,
		chalPeriod,
	)
}

// Determines the bisection point from parentHeight to toHeight and returns a history
// commitment with a prefix proof for the action based on the challenge type.
func (et *Tracker) DetermineBisectionHistoryWithProof(
	ctx context.Context,
) (history.History, []byte, error) {
	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.EndCommitment()
	bisectTo, err := math.Bisect(uint64(startHeight), uint64(endHeight))
	if err != nil {
		return history.History{}, nil, errors.Wrapf(err, "determining bisection point errored for %d and %d", startHeight, endHeight)
	}
	challengeLevel := et.edge.GetChallengeLevel()
	if challengeLevel == protocol.NewBlockChallengeLevel() {
		historyCommit, commitErr := et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           et.associatedAssertionMetadata,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(bisectTo)),
			},
		)
		if commitErr != nil {
			return history.History{}, nil, commitErr
		}
		proof, proofErr := et.stateProvider.PrefixProof(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           et.associatedAssertionMetadata,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
			},
			l2stateprovider.Height(bisectTo),
		)
		if proofErr != nil {
			return history.History{}, nil, proofErr
		}
		return historyCommit, proof, nil
	}
	var historyCommit history.History
	var commitErr error
	var proof []byte
	var proofErr error

	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return history.History{}, nil, err
	}
	challengeOriginHeights := make([]l2stateprovider.Height, len(originHeights.ChallengeOriginHeights))
	for index, height := range originHeights.ChallengeOriginHeights {
		challengeOriginHeights[index] = l2stateprovider.Height(height)
	}
	// The first challenge origin height must account for the start block height of the assertion.
	historyCommit, commitErr = et.stateProvider.HistoryCommitment(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			AssertionMetadata:           et.associatedAssertionMetadata,
			UpperChallengeOriginHeights: challengeOriginHeights,
			UpToHeight:                  option.Some(l2stateprovider.Height(bisectTo)),
		},
	)
	if commitErr != nil {
		return history.History{}, nil, errors.Wrap(commitErr, "could not produce history commitment")
	}
	proof, proofErr = et.stateProvider.PrefixProof(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			AssertionMetadata:           et.associatedAssertionMetadata,
			UpperChallengeOriginHeights: challengeOriginHeights,
			UpToHeight:                  option.Some(l2stateprovider.Height(endHeight)),
		},
		l2stateprovider.Height(bisectTo),
	)
	if proofErr != nil {
		return history.History{}, nil, errors.Wrap(proofErr, "could not produce prefix proof")
	}
	return historyCommit, proof, nil
}

func (et *Tracker) bisect(ctx context.Context) (protocol.VerifiedRoyalEdge, protocol.VerifiedRoyalEdge, error) {
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
	log.Info("Bisecting honest edge", et.uniqueTrackerLogFields()...)
	if addVerifiedErr := et.chainWatcher.AddVerifiedHonestEdge(ctx, firstChild); addVerifiedErr != nil {
		// We simply log an error, as if this errored, it will be added later on by the chain watcher
		// scraping events from the chain, but this is a helpful optimization.
		log.Error("Could not add verified honest edge to chain watcher", "err", addVerifiedErr)
	}
	if addVerifiedErr := et.chainWatcher.AddVerifiedHonestEdge(ctx, secondChild); addVerifiedErr != nil {
		log.Error("Could not add verified honest edge to chain watcher", "err", addVerifiedErr)
	}
	return firstChild, secondChild, nil
}

func (et *Tracker) openSubchallenge(ctx context.Context) error {
	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get top level claim height")
	}

	fromBlockChallengeHeight := l2stateprovider.Height(originHeights.ChallengeOriginHeights[0])

	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.EndCommitment()

	fields := et.uniqueTrackerLogFields()

	var startHistory history.History
	var endHistory history.History
	var startParentCommitment history.History
	var endParentCommitment history.History
	var startEndPrefixProof []byte
	challengeLevel := et.edge.GetChallengeLevel()
	switch challengeLevel {
	case protocol.NewBlockChallengeLevel():
		endHistory, err = et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           et.associatedAssertionMetadata,
				UpperChallengeOriginHeights: []l2stateprovider.Height{fromBlockChallengeHeight},
				UpToHeight:                  option.None[l2stateprovider.Height](),
			},
		)
		if err != nil {
			return errors.Wrap(err, "could not compute end history commitment")
		}
		startEndPrefixProof, err = et.stateProvider.PrefixProof(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           et.associatedAssertionMetadata,
				UpperChallengeOriginHeights: []l2stateprovider.Height{fromBlockChallengeHeight},
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
				AssertionMetadata:           et.associatedAssertionMetadata,
				UpperChallengeOriginHeights: []l2stateprovider.Height{fromBlockChallengeHeight},
				UpToHeight:                  option.Some(l2stateprovider.Height(0)),
			},
		)
		if err != nil {
			return err
		}
		endParentCommitment, err = et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           et.associatedAssertionMetadata,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
				UpToHeight:                  option.Some(fromBlockChallengeHeight + 1),
			},
		)
		if err != nil {
			return err
		}
		startParentCommitment, err = et.stateProvider.HistoryCommitment(
			ctx,
			&l2stateprovider.HistoryCommitmentRequest{
				AssertionMetadata:           et.associatedAssertionMetadata,
				UpperChallengeOriginHeights: []l2stateprovider.Height{},
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
			AssertionMetadata:           et.associatedAssertionMetadata,
			UpperChallengeOriginHeights: heights,
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
			AssertionMetadata:           et.associatedAssertionMetadata,
			UpperChallengeOriginHeights: heights,
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
			AssertionMetadata:           et.associatedAssertionMetadata,
			UpperChallengeOriginHeights: heights,
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
			AssertionMetadata:           et.associatedAssertionMetadata,
			UpperChallengeOriginHeights: heights[:len(heights)-1],
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
			AssertionMetadata:           et.associatedAssertionMetadata,
			UpperChallengeOriginHeights: heights[:len(heights)-1],
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
	fields = append(
		fields,
		"firstLeaf", containers.Trunc(startHistory.FirstLeaf.Bytes()),
		"lastLeaf", containers.Trunc(endHistory.LastLeaf.Bytes()),
		"parentFirstLeaf", containers.Trunc(startParentCommitment.LastLeaf.Bytes()),
		"parentLastLeaf", containers.Trunc(endParentCommitment.LastLeaf.Bytes()),
		"parentStartHeight", startParentCommitment.Height,
		"parentEndHeight", endParentCommitment.Height,
	)
	log.Info("Identified single point of disagreement within a challenge level, now opening subchallenge", fields...)
	log.Info("Making subchallenge creation move on edge", fields...)

	manager := et.chain.SpecChallengeManager()
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
	addedLeafChallengeLevel := addedLeaf.GetChallengeLevel()
	fields = append(fields, "subchallengeType", addedLeafChallengeLevel)
	log.Info("Successfully created a subchallenge edge", fields...)

	if addVerifiedErr := et.chainWatcher.AddVerifiedHonestEdge(ctx, addedLeaf); addVerifiedErr != nil {
		// We simply log an error, as if this errored, it will be added later on by the chain watcher
		// scraping events from the chain, but this is a helpful optimization.
		log.Error("Could not add verified honest edge to chain watcher", "err", addVerifiedErr)
	}

	tracker, err := New(
		ctx,
		addedLeaf,
		et.chain,
		et.stateProvider,
		et.chainWatcher,
		et.challengeManager,
		et.associatedAssertionMetadata,
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
	log.Info("Identified single step of disagreement at the execution of a block, ready for one-step fraud proof", fields...)
	log.Info("Submitting one-step-proof to protocol", fields...)
	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get top level claim height")
	}
	pc, _ := et.edge.StartCommitment()

	challengeOriginHeights := make([]l2stateprovider.Height, len(originHeights.ChallengeOriginHeights))
	for index, height := range originHeights.ChallengeOriginHeights {
		challengeOriginHeights[index] = l2stateprovider.Height(height)
	}
	data, beforeStateInclusionProof, afterStateInclusionProof, err := et.stateProvider.OneStepProofData(
		ctx,
		et.associatedAssertionMetadata,
		challengeOriginHeights,
		l2stateprovider.Height(pc),
	)
	if err != nil {
		return errors.Wrapf(errBadOneStepProof, "could not get one step data: %v", err)
	}
	manager := et.chain.SpecChallengeManager()
	if err = manager.ConfirmEdgeByOneStepProof(
		ctx,
		et.edge.Id(),
		data,
		beforeStateInclusionProof,
		afterStateInclusionProof,
	); err != nil {
		return errors.Wrap(err, "could not confirm one step proof against protocol")
	}
	log.Info("Succeeded one-step-proof for edge and confirmed it as winner", fields...)
	return nil
}

func canOneStepProve(ctx context.Context, edge protocol.SpecEdge) (bool, error) {
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

func IsRootBlockChallengeEdge(edge protocol.ReadOnlyEdge) bool {
	return edge.ClaimId().IsSome() && edge.GetChallengeLevel() == protocol.NewBlockChallengeLevel()
}
