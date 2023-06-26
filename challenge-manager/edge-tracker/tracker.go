package edgetracker

import (
	"context"
	"fmt"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	challengetree "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/challenge-tree"
	"github.com/OffchainLabs/challenge-protocol-v2/containers"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/fsm"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/OffchainLabs/challenge-protocol-v2/math"
	commitments "github.com/OffchainLabs/challenge-protocol-v2/state-commitments/history"
	utilTime "github.com/OffchainLabs/challenge-protocol-v2/time"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var errBadOneStepProof = errors.New("bad one step proof data")

var log = logrus.WithField("prefix", "edge-tracker")

var (
	spawnedCounter       = metrics.NewRegisteredCounter("arb/validator/tracker/spawned", nil)
	bisectedCounter      = metrics.NewRegisteredCounter("arb/validator/tracker/bisected", nil)
	confirmedCounter     = metrics.NewRegisteredCounter("arb/validator/tracker/confirmed", nil)
	layerZeroLeafCounter = metrics.NewRegisteredCounter("arb/validator/tracker/layer_zero_leaves", nil)
)

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

type ChallengeTracker interface {
	IsTrackingEdge(protocol.EdgeId) bool
	MarkTrackedEdge(protocol.EdgeId)
}

type Opt func(et *Tracker)

func WithActInterval(d time.Duration) Opt {
	return func(et *Tracker) {
		et.actInterval = d
	}
}

func WithTimeReference(ref utilTime.Reference) Opt {
	return func(et *Tracker) {
		et.timeRef = ref
	}
}

func WithValidatorName(name string) Opt {
	return func(et *Tracker) {
		et.validatorName = name
	}
}

func WithValidatorAddress(addr common.Address) Opt {
	return func(et *Tracker) {
		et.validatorAddress = addr
	}
}

func WithFSMOpts(opts ...fsm.Opt[edgeTrackerAction, EdgeTrackerState]) Opt {
	return func(et *Tracker) {
		et.fsmOpts = opts
	}
}

type HeightConfig struct {
	StartBlockHeight           uint64
	TopLevelClaimEndBatchCount uint64
}

type Tracker struct {
	edge             protocol.SpecEdge
	fsm              *fsm.Fsm[edgeTrackerAction, EdgeTrackerState]
	fsmOpts          []fsm.Opt[edgeTrackerAction, EdgeTrackerState]
	actInterval      time.Duration
	timeRef          utilTime.Reference
	validatorName    string
	validatorAddress common.Address
	chain            protocol.Protocol
	stateProvider    l2stateprovider.Provider
	chainWatcher     ConfirmationMetadataChecker
	challengeManager ChallengeTracker
	heightConfig     HeightConfig
}

func New(
	edge protocol.SpecEdge,
	chain protocol.Protocol,
	stateProvider l2stateprovider.Provider,
	chainWatcher ConfirmationMetadataChecker,
	challengeManager ChallengeTracker,
	heightConfig HeightConfig,
	opts ...Opt,
) (*Tracker, error) {
	tr := &Tracker{
		edge:             edge,
		chain:            chain,
		stateProvider:    stateProvider,
		chainWatcher:     chainWatcher,
		challengeManager: challengeManager,
		heightConfig:     heightConfig,
		actInterval:      time.Second,
		timeRef:          utilTime.NewRealTimeReference(),
	}
	for _, o := range opts {
		o(tr)
	}
	fsm, err := newEdgeTrackerFsm(
		edgeStarted,
		tr.fsmOpts...,
	)
	if err != nil {
		return nil, err
	}
	tr.fsm = fsm
	return tr, nil
}

func (et *Tracker) TopLevelClaimEndBatchCount() uint64 {
	return et.heightConfig.TopLevelClaimEndBatchCount
}

func (et *Tracker) StartBlockHeight() uint64 {
	return et.heightConfig.StartBlockHeight
}

func (et *Tracker) Spawn(ctx context.Context) {
	// No-op if we are already tracking this edge in our challenge manager.
	if et.challengeManager.IsTrackingEdge(et.edge.Id()) {
		return
	}
	fields := et.uniqueTrackerLogFields()
	log.WithFields(fields).Info("Tracking edge")
	spawnedCounter.Inc(1)
	et.challengeManager.MarkTrackedEdge(et.edge.Id())
	t := et.timeRef.NewTicker(et.actInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			if et.shouldComplete() {
				log.WithFields(fields).Infof("Edge tracker received notice of a confirmation, exiting")
				spawnedCounter.Dec(1)
				return
			}
			if err := et.Act(ctx); err != nil {
				log.Error(err)
			}
		case <-ctx.Done():
			log.WithFields(fields).Debug("Edge tracker goroutine exiting")
			spawnedCounter.Dec(1)
			return
		}
	}
}

func (et *Tracker) CurrentState() EdgeTrackerState {
	return et.fsm.Current().State
}

func (et *Tracker) Act(ctx context.Context) error {
	fields := et.uniqueTrackerLogFields()
	current := et.fsm.Current()
	switch current.State {
	// Start state.
	case edgeStarted:
		canOsp, err := canOneStepProve(et.edge)
		if err != nil {
			log.WithFields(fields).WithError(err).Error("Could not check if edge can be one step proven")
			return et.fsm.Do(edgeBackToStart{})
		}
		if canOsp {
			return et.fsm.Do(edgeHandleOneStepProof{})
		}
		wasConfirmed, err := et.tryToConfirm(ctx)
		if err != nil {
			log.WithFields(fields).WithError(err).Debug("Could not confirm edge yet")
			return et.fsm.Do(edgeBackToStart{})
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
			log.WithFields(fields).WithError(err).Error("Could not check if edge has length one rival")
			return et.fsm.Do(edgeBackToStart{})
		}
		if atOneStepFork {
			return et.fsm.Do(edgeOpenSubchallengeLeaf{})
		}
		return et.fsm.Do(edgeBisect{})
	// Edge is at a one-step-proof in a small-step challenge.
	case edgeAtOneStepProof:
		if err := et.submitOneStepProof(ctx); err != nil {
			if errors.Is(err, errBadOneStepProof) {
				return et.fsm.Do(edgeConfirm{})
			}
			log.WithFields(fields).WithError(err).Error("Could not submit one step proof")
			return et.fsm.Do(edgeBackToStart{})
		}
		return et.fsm.Do(edgeConfirm{})
	// Edge tracker should add a subchallenge level zero leaf.
	case edgeAddingSubchallengeLeaf:
		if err := et.openSubchallengeLeaf(ctx); err != nil {
			log.WithFields(fields).WithError(err).Error("Could not open subchallenge leaf")
			return et.fsm.Do(edgeBackToStart{})
		}
		layerZeroLeafCounter.Inc(1)
		return et.fsm.Do(edgeAwaitConfirmation{})
	// Edge should bisect.
	case edgeBisecting:
		lowerChild, upperChild, err := et.bisect(ctx)
		if err != nil {
			log.WithError(err).WithFields(fields).Error("Could not bisect")
			return et.fsm.Do(edgeBackToStart{})
		}
		bisectedCounter.Inc(1)

		firstTracker, err := New(
			lowerChild,
			et.chain,
			et.stateProvider,
			et.chainWatcher,
			et.challengeManager,
			et.heightConfig,
			WithActInterval(et.actInterval),
			WithTimeReference(et.timeRef),
			WithValidatorAddress(et.validatorAddress),
			WithValidatorName(et.validatorName),
			WithFSMOpts(et.fsmOpts...),
		)
		if err != nil {
			log.WithError(err).WithFields(fields).Error("Could not create new edge tracker")
			return et.fsm.Do(edgeBackToStart{})
		}
		secondTracker, err := New(
			upperChild,
			et.chain,
			et.stateProvider,
			et.chainWatcher,
			et.challengeManager,
			et.heightConfig,
			WithActInterval(et.actInterval),
			WithTimeReference(et.timeRef),
			WithValidatorAddress(et.validatorAddress),
			WithValidatorName(et.validatorName),
			WithFSMOpts(et.fsmOpts...),
		)
		if err != nil {
			log.WithError(err).WithFields(fields).Error("Could not create new edge tracker")
			return et.fsm.Do(edgeBackToStart{})
		}
		go firstTracker.Spawn(ctx)
		go secondTracker.Spawn(ctx)
		return et.fsm.Do(edgeAwaitConfirmation{})
	case edgeConfirming:
		wasConfirmed, err := et.tryToConfirm(ctx)
		if err != nil {
			log.WithFields(fields).WithError(err).Debug("Could not confirm edge yet")
			return et.fsm.Do(edgeAwaitConfirmation{})
		}
		if !wasConfirmed {
			return et.fsm.Do(edgeAwaitConfirmation{})
		}
		return et.fsm.Do(edgeConfirm{})
	case edgeConfirmed:
		log.WithFields(fields).Info("Edge reached confirmed state")
		return et.fsm.Do(edgeConfirm{})
	default:
		return fmt.Errorf("invalid state: %s", current.State)
	}
}

func (et *Tracker) shouldComplete() bool {
	return et.fsm.Current().State == edgeConfirmed
}

func (et *Tracker) uniqueTrackerLogFields() logrus.Fields {
	startHeight, startCommit := et.edge.StartCommitment()
	endHeight, endCommit := et.edge.EndCommitment()
	id := et.edge.Id()
	return logrus.Fields{
		"id":            containers.Trunc(id[:]),
		"startHeight":   startHeight,
		"startCommit":   containers.Trunc(startCommit.Bytes()),
		"endHeight":     endHeight,
		"endCommit":     containers.Trunc(endCommit.Bytes()),
		"validatorName": et.validatorName,
		"challengeType": et.edge.GetType(),
	}
}

func (et *Tracker) childrenAreConfirmed(
	ctx context.Context,
	chalManager protocol.SpecChallengeManager,
) (bool, error) {
	lower, err := et.edge.LowerChild(ctx)
	if err != nil {
		return false, err
	}
	upper, err := et.edge.UpperChild(ctx)
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
	assertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not get prev assertion hash")
	}
	manager, err := et.chain.SpecChallengeManager(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not get challenge manager")
	}

	// Check if we can confirm by children.
	childrenConfirmed, err := et.childrenAreConfirmed(ctx, manager)
	if err != nil {
		return false, errors.Wrap(err, "could not check if children are confirmed")
	}
	if childrenConfirmed {
		if confirmErr := et.edge.ConfirmByChildren(ctx); confirmErr != nil {
			return false, errors.Wrap(confirmErr, "could not confirm by children")
		}
		log.WithFields(et.uniqueTrackerLogFields()).Info("Confirmed by children")
		confirmedCounter.Inc(1)
		return true, nil
	}

	// Check if we can confirm by claim.
	claimingEdge, ok := et.chainWatcher.ConfirmedEdgeWithClaimExists(
		assertionHash,
		protocol.ClaimId(et.edge.Id()),
	)
	if ok {
		if confirmClaimErr := et.edge.ConfirmByClaim(ctx, protocol.ClaimId(claimingEdge)); confirmClaimErr != nil {
			return false, errors.Wrap(confirmClaimErr, "could not confirm by claim")
		}
		log.WithFields(et.uniqueTrackerLogFields()).Info("Confirmed by claim")
		confirmedCounter.Inc(1)
		return true, nil
	}

	// Check if we can confirm by time.
	timer, ancestors, err := et.chainWatcher.ComputeHonestPathTimer(ctx, assertionHash, et.edge.Id())
	if err != nil {
		return false, errors.Wrap(err, "could not compute honest path timer")
	}
	chalPeriod, err := manager.ChallengePeriodBlocks(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not check the challenge period length")
	}
	if timer >= challengetree.PathTimer(chalPeriod) {
		if err := et.edge.ConfirmByTimer(ctx, ancestors); err != nil {
			return false, errors.Wrap(err, "could not confirm by timer")
		}
		log.WithFields(et.uniqueTrackerLogFields()).Info("Confirmed by time")
		confirmedCounter.Inc(1)
		return true, nil
	}
	return false, nil
}

// Determines the bisection point from parentHeight to toHeight and returns a history
// commitment with a prefix proof for the action based on the challenge type.
func (et *Tracker) determineBisectionHistoryWithProof(
	ctx context.Context,
) (commitments.History, []byte, error) {
	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.EndCommitment()
	bisectTo, err := math.Bisect(uint64(startHeight), uint64(endHeight))
	if err != nil {
		return commitments.History{}, nil, errors.Wrapf(err, "determining bisection point failed for %d and %d", startHeight, endHeight)
	}
	if et.edge.GetType() == protocol.BlockChallengeEdge {
		historyCommit, commitErr := et.stateProvider.HistoryCommitmentUpToBatch(ctx, et.heightConfig.StartBlockHeight, et.heightConfig.StartBlockHeight+bisectTo, et.heightConfig.TopLevelClaimEndBatchCount)
		if commitErr != nil {
			return commitments.History{}, nil, commitErr
		}
		proof, proofErr := et.stateProvider.PrefixProofUpToBatch(ctx, et.heightConfig.StartBlockHeight, bisectTo, uint64(endHeight), et.heightConfig.TopLevelClaimEndBatchCount)
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

	fromAssertionHeight := uint64(originHeights.BlockChallengeOriginHeight)
	toAssertionHeight := fromAssertionHeight + 1

	switch et.edge.GetType() {
	case protocol.BigStepChallengeEdge:
		historyCommit, commitErr = et.stateProvider.BigStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, bisectTo)
		proof, proofErr = et.stateProvider.BigStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, bisectTo, uint64(endHeight))
	case protocol.SmallStepChallengeEdge:
		fromBigStep := uint64(originHeights.BigStepChallengeOriginHeight)
		toBigStep := fromBigStep + 1

		historyCommit, commitErr = et.stateProvider.SmallStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, fromBigStep, toBigStep, bisectTo)
		proof, proofErr = et.stateProvider.SmallStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, fromBigStep, toBigStep, bisectTo, uint64(endHeight))
	default:
		return commitments.History{}, nil, fmt.Errorf("unsupported challenge type: %s", et.edge.GetType())
	}
	if commitErr != nil {
		return commitments.History{}, nil, errors.Wrap(commitErr, "could not produce history commitment")
	}
	if proofErr != nil {
		return commitments.History{}, nil, errors.Wrap(proofErr, "could not produce prefix proof")
	}
	return historyCommit, proof, nil
}

func (et *Tracker) bisect(ctx context.Context) (protocol.SpecEdge, protocol.SpecEdge, error) {
	historyCommit, proof, err := et.determineBisectionHistoryWithProof(ctx)
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
	log.WithFields(logrus.Fields{
		"name":               et.validatorName,
		"challengeType":      et.edge.GetType(),
		"bisectedFrom":       endHeight,
		"bisectedFromMerkle": containers.Trunc(endCommit.Bytes()),
		"bisectedTo":         bisectTo,
		"bisectedToMerkle":   containers.Trunc(historyCommit.Merkle.Bytes()),
	}).Info("Successfully bisected edge")
	return firstChild, secondChild, nil
}

func (et *Tracker) openSubchallengeLeaf(ctx context.Context) error {
	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get top level claim height")
	}

	fromAssertionHeight := uint64(originHeights.BlockChallengeOriginHeight)
	toAssertionHeight := fromAssertionHeight + 1

	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.EndCommitment()

	fields := logrus.Fields{
		"name":                et.validatorName,
		"edgeStartHeight":     startHeight,
		"edgeEndHeight":       endHeight,
		"fromAssertionHeight": fromAssertionHeight,
	}

	var startHistory commitments.History
	var endHistory commitments.History
	var startParentCommitment commitments.History
	var endParentCommitment commitments.History
	var startEndPrefixProof []byte
	switch et.edge.GetType() {
	case protocol.BlockChallengeEdge:
		fromBlock := fromAssertionHeight + et.heightConfig.StartBlockHeight
		toBlock := toAssertionHeight + et.heightConfig.StartBlockHeight
		startHistory, err = et.stateProvider.BigStepCommitmentUpTo(ctx, fromBlock, toBlock, 0)
		if err != nil {
			return err
		}
		endHistory, err = et.stateProvider.BigStepLeafCommitment(ctx, fromBlock, toBlock)
		if err != nil {
			return err
		}
		startParentCommitment, err = et.stateProvider.HistoryCommitmentUpToBatch(ctx, et.heightConfig.StartBlockHeight, fromBlock, et.heightConfig.TopLevelClaimEndBatchCount)
		if err != nil {
			return err
		}
		endParentCommitment, err = et.stateProvider.HistoryCommitmentUpToBatch(ctx, et.heightConfig.StartBlockHeight, toBlock, et.heightConfig.TopLevelClaimEndBatchCount)
		if err != nil {
			return err
		}
		startEndPrefixProof, err = et.stateProvider.BigStepPrefixProof(ctx, fromBlock, toBlock, 0, endHistory.Height)
		if err != nil {
			return err
		}
	case protocol.BigStepChallengeEdge:
		fromBlock := fromAssertionHeight + et.heightConfig.StartBlockHeight
		toBlock := toAssertionHeight + et.heightConfig.StartBlockHeight
		startHistory, err = et.stateProvider.SmallStepCommitmentUpTo(ctx, fromBlock, toBlock, uint64(startHeight), uint64(endHeight), 0)
		if err != nil {
			return err
		}
		endHistory, err = et.stateProvider.SmallStepLeafCommitment(ctx, fromBlock, toBlock, uint64(startHeight), uint64(endHeight))
		if err != nil {
			return err
		}
		startParentCommitment, err = et.stateProvider.BigStepCommitmentUpTo(ctx, fromBlock, toBlock, uint64(startHeight))
		if err != nil {
			return err
		}
		endParentCommitment, err = et.stateProvider.BigStepCommitmentUpTo(ctx, fromBlock, toBlock, uint64(endHeight))
		if err != nil {
			return err
		}
		startEndPrefixProof, err = et.stateProvider.SmallStepPrefixProof(ctx, fromBlock, toBlock, uint64(startHeight), uint64(endHeight), 0, endHistory.Height)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported subchallenge type for creating leaf commitment")
	}
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
	fields["subChallengeType"] = addedLeaf.GetType()
	log.WithFields(fields).Info("Created subchallenge edge")
	tracker, err := New(
		addedLeaf,
		et.chain,
		et.stateProvider,
		et.chainWatcher,
		et.challengeManager,
		et.heightConfig,
		WithActInterval(et.actInterval),
		WithTimeReference(et.timeRef),
		WithValidatorAddress(et.validatorAddress),
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
	log.WithFields(fields).Info("Submitting one-step-proof to protocol")
	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get top level claim height")
	}
	fromAssertionHeight := uint64(originHeights.BlockChallengeOriginHeight)
	toAssertionHeight := fromAssertionHeight + 1
	fromBigStep := uint64(originHeights.BigStepChallengeOriginHeight)
	toBigStep := fromBigStep + 1
	pc, _ := et.edge.StartCommitment()

	assertionHash, err := et.edge.AssertionHash(ctx)
	if err != nil {
		return err
	}
	parentAssertionCreationInfo, err := et.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return err
	}
	cfgSnapshot := &l2stateprovider.ConfigSnapshot{
		RequiredStake:           parentAssertionCreationInfo.RequiredStake,
		ChallengeManagerAddress: parentAssertionCreationInfo.ChallengeManager,
		ConfirmPeriodBlocks:     parentAssertionCreationInfo.ConfirmPeriodBlocks,
		WasmModuleRoot:          parentAssertionCreationInfo.WasmModuleRoot,
		InboxMaxCount:           parentAssertionCreationInfo.InboxMaxCount,
	}
	data, beforeStateInclusionProof, afterStateInclusionProof, err := et.stateProvider.OneStepProofData(
		ctx,
		cfgSnapshot,
		parentAssertionCreationInfo.AfterState,
		fromAssertionHeight,
		toAssertionHeight,
		fromBigStep,
		toBigStep,
		uint64(pc),
		uint64(pc)+1,
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
	log.WithFields(fields).Info("Succeeded one-step-proof for edge and confirmed it as winner")
	return nil
}

func canOneStepProve(edge protocol.SpecEdge) (bool, error) {
	start, _ := edge.StartCommitment()
	end, _ := edge.EndCommitment()
	// Can never happen in the protocol, but added as an additional defensive check.
	if start >= end {
		return false, fmt.Errorf("start height %d cannot be >= end height %d", start, end)
	}
	return end-start == 1 && edge.GetType() == protocol.SmallStepChallengeEdge, nil
}
