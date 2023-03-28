package validator

import (
	"context"
	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

func (et *edgeTracker) act(ctx context.Context) error {
	current := et.fsm.Current()
	switch current.State {
	// Start state.
	case edgeStarted:
		// TODO: Add a conditional to check if we can confirm.
		atOneStepFork, err := et.challenge.EdgeIsOneStepForkSource(et.edge)
		if err != nil {
			return errors.Wrap(err, "could not check one step fork")
		}
		if atOneStepFork {
			return et.fsm.Do(edgeHandleOneStepFork{})
		}
		isPresumptive, err := et.edge.IsPresumptive(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check presumptive")
		}
		if isPresumptive {
			return et.fsm.Do(edgeMarkPresumptive{})
		}
		return et.fsm.Do(edgeBisect{})
	// Edge is the source of a one-step-fork.
	case edgeAtOneStepFork:
		event, ok := current.SourceEvent.(edgeHandleOneStepFork)
		if !ok {
			return fmt.Errorf("bad source event: %s", event)
		}
		startHeight, startCommit := et.edge.StartCommitment()
		log.WithFields(logrus.Fields{
			"name": et.cfg.validatorName,
		}).Infof(
			"Reached one-step-fork at %d and commitment %s",
			startHeight,
			util.Trunc(startCommit.Bytes()),
		)
		challengeType := et.challenge.GetType()
		if challengeType == protocol.SmallStepChallenge {
			return et.fsm.Do(edgeHandleOneStepProof{})
		}
		return et.fsm.Do(edgeOpenSubchallenge{})
	// Edge is at a one-step-proof in a small-step challenge.
	case edgeAtOneStepProof:
		log.WithFields(logrus.Fields{
			"name": et.cfg.validatorName,
		}).Info("Checking one-step-proof against protocol")
		return et.fsm.Do(edgeHandleOneStepProof{})
	// Edge tracker should open a subchallenge.
	case edgeOpeningSubchallenge:
		subChallenge, err := et.openSubchallenge(ctx)
		if err != nil {
			return err
		}
		return et.fsm.Do(edgeOpenSubchallengeLeaf{
			subChallenge: subChallenge,
		})
	// Edge tracker should add a subchallenge leaf.
	case edgeAddingSubchallengeLeaf:
		event, ok := current.SourceEvent.(edgeOpenSubchallengeLeaf)
		if !ok {
			return fmt.Errorf("bad source event: %s", event)
		}
		if err := et.openSubchallengeLeaf(
			ctx, event.subChallenge,
		); err != nil {
			return errors.Wrap(err, "could not open subchallenge leaf")
		}
		return et.fsm.Do(edgeAwaitSubchallengeResolution{})
	// Edge should bisect.
	case edgeBisecting:
		firstChild, secondChild, err := et.bisect(ctx)
		if err != nil {
			if errors.Is(err, solimpl.ErrAlreadyExists) {
				return et.fsm.Do(edgeBackToStart{})
			}
			// log.WithError(err).WithFields(logrus.Fields{
			// 	"height":        vt.vertex.HistoryCommitment().Height,
			// 	"merkle":        util.Trunc(vt.vertex.HistoryCommitment().Merkle.Bytes()),
			// 	"validatorName": vt.cfg.validatorName,
			// 	"challengeType": vt.challenge.GetType(),
			// 	"address":       util.Trunc(vt.cfg.validatorAddress.Bytes()),
			// }).Error("could not bisect")
			return et.fsm.Do(edgeBackToStart{})
		}
		firstTracker, err := newEdgeTracker(
			et.cfg,
			et.challenge,
			firstChild,
		)
		if err != nil {
			// log.WithError(err).WithFields(logrus.Fields{
			// 	"height":        vt.vertex.HistoryCommitment().Height,
			// 	"merkle":        util.Trunc(vt.vertex.HistoryCommitment().Merkle.Bytes()),
			// 	"validatorName": vt.cfg.validatorName,
			// 	"challengeType": vt.challenge.GetType(),
			// 	"address":       util.Trunc(vt.cfg.validatorAddress.Bytes()),
			// }).Error("could not create new vertex tracker")
			return et.fsm.Do(edgeBackToStart{})
		}
		go firstTracker.spawn(ctx)
		secondTracker, err := newEdgeTracker(
			et.cfg,
			et.challenge,
			secondChild,
		)
		if err != nil {
			// log.WithError(err).WithFields(logrus.Fields{
			// 	"height":        vt.vertex.HistoryCommitment().Height,
			// 	"merkle":        util.Trunc(vt.vertex.HistoryCommitment().Merkle.Bytes()),
			// 	"validatorName": vt.cfg.validatorName,
			// 	"challengeType": vt.challenge.GetType(),
			// 	"address":       util.Trunc(vt.cfg.validatorAddress.Bytes()),
			// }).Error("could not create new vertex tracker")
			return et.fsm.Do(edgeBackToStart{})
		}
		go secondTracker.spawn(ctx)
		return et.fsm.Do(edgeBackToStart{})
	// Edge is presumptive, should do nothing until it loses ps status.
	case edgePresumptive:
		isPs, err := et.edge.IsPresumptive(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check if presumptive")
		}
		if !isPs {
			return et.fsm.Do(edgeLosePresumptive{})
		}
		return et.fsm.Do(edgeMarkPresumptive{})
	// TODO: Handle this case.
	case edgePresumptiveLost:
		return et.fsm.Do(edgeLosePresumptive{})
	case edgeAwaitingSubchallenge:
		return et.fsm.Do(edgeAwaitSubchallengeResolution{})
	default:
		return fmt.Errorf("invalid state: %s", current.State)
	}
}

// Determines the bisection point from parentHeight to toHeight and returns a history
// commitment with a prefix proof for the action based on the challenge type.
func (et *edgeTracker) determineBisectionHistoryWithProof(
	ctx context.Context,
) (util.HistoryCommitment, []byte, error) {
	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.TargetCommitment()
	bisectTo, err := util.BisectionPoint(uint64(startHeight), uint64(endHeight))
	if err != nil {
		return util.HistoryCommitment{}, nil, errors.Wrapf(err, "determining bisection point failed for %d and %d", startHeight, endHeight)
	}

	if et.challenge.GetType() == protocol.BlockChallenge {
		historyCommit, commitErr := et.cfg.stateManager.HistoryCommitmentUpTo(ctx, bisectTo)
		if commitErr != nil {
			return util.HistoryCommitment{}, nil, commitErr
		}
		proof, proofErr := et.cfg.stateManager.PrefixProof(ctx, bisectTo, uint64(endHeight))
		if proofErr != nil {
			return util.HistoryCommitment{}, nil, proofErr
		}
		return historyCommit, proof, nil
	}
	topLevelHeight, _, err := et.challenge.TopLevelClaimCommitment(ctx)
	if err != nil {
		return util.HistoryCommitment{}, nil, err
	}

	fromAssertionHeight := uint64(topLevelHeight)
	toAssertionHeight := fromAssertionHeight + 1

	var historyCommit util.HistoryCommitment
	var commitErr error
	var proof []byte
	var proofErr error
	switch et.challenge.GetType() {
	case protocol.BigStepChallenge:
		historyCommit, commitErr = et.cfg.stateManager.BigStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, bisectTo)
		proof, proofErr = et.cfg.stateManager.BigStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, bisectTo, uint64(endHeight))
	case protocol.SmallStepChallenge:
		historyCommit, commitErr = et.cfg.stateManager.SmallStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, bisectTo)
		proof, proofErr = et.cfg.stateManager.SmallStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, bisectTo, uint64(endHeight))
	default:
		return util.HistoryCommitment{}, nil, fmt.Errorf("unsupported challenge type: %s", et.challenge.GetType())
	}
	if commitErr != nil {
		return util.HistoryCommitment{}, nil, commitErr
	}
	if proofErr != nil {
		return util.HistoryCommitment{}, nil, proofErr
	}
	return historyCommit, proof, nil
}

func (et *edgeTracker) bisect(ctx context.Context) (protocol.SpecEdge, protocol.SpecEdge, error) {
	historyCommit, proof, err := et.determineBisectionHistoryWithProof(ctx)
	if err != nil {
		return nil, nil, err
	}
	bisectTo := historyCommit.Height
	firstChild, secondChild, err := et.edge.Bisect(ctx, historyCommit, proof)
	if err != nil {
		return nil, nil, errors.Wrapf(
			err,
			"%s could not bisect to height=%d,commit=%s from height=%d,commit=%s",
			et.cfg.validatorName,
			bisectTo,
			util.Trunc(historyCommit.Merkle.Bytes()),
			// TODO: Change.
			5,
			"hi",
			// validatorChallengeVertexHistoryCommitment.Height,
			// util.Trunc(validatorChallengeVertexHistoryCommitment.Merkle.Bytes()),
		)
	}
	isPs, err := firstChild.IsPresumptive(ctx)
	if err != nil {
		return nil, nil, err
	}
	log.WithFields(logrus.Fields{
		"name":          et.cfg.validatorName,
		"challengeType": et.challenge.GetType(),
		"isPs":          isPs,
		// "bisectedFrom":       validatorChallengeVertexHistoryCommitment.Height,
		// "bisectedFromMerkle": util.Trunc(validatorChallengeVertexHistoryCommitment.Merkle.Bytes()),
		// "bisectedTo":         bisectedVertexCommitment.Height,
		// "bisectedToMerkle":   util.Trunc(bisectedVertexCommitment.Merkle[:]),
	}).Info("Successfully bisected edge")
	return firstChild, secondChild, nil
}

func (et *edgeTracker) openSubchallenge(ctx context.Context) (protocol.SpecChallenge, error) {
	if et.challenge.GetType() == protocol.SmallStepChallenge {
		return nil, errors.New("cannot create subchallenge on small step challenge")
	}
	manager, err := et.cfg.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	var subChalToCreate protocol.ChallengeType
	switch et.challenge.GetType() {
	case protocol.BlockChallenge:
		subChalToCreate = protocol.BigStepChallenge
	case protocol.BigStepChallenge:
		subChalToCreate = protocol.SmallStepChallenge
	default:
		return nil, errors.New("unsupported challenge type to create")
	}
	subChal, err := et.edge.CreateSubChallenge(ctx)
	if err != nil {
		switch {
		case errors.Is(err, solimpl.ErrAlreadyExists):
			subChalHash, calcErr := manager.CalculateChallengeHash(ctx, et.edge.Id(), subChalToCreate)
			if calcErr != nil {
				return nil, calcErr
			}
			fetchedSubChal, fetcherErr := manager.GetChallenge(ctx, subChalHash)
			if fetcherErr != nil {
				return nil, fetcherErr
			}
			if fetchedSubChal.IsNone() {
				return nil, fmt.Errorf("no subchallenge found on-chain for id %#x", subChalHash)
			}
			subChal = fetchedSubChal.Unwrap()
		default:
			return nil, errors.Wrap(err, "subchallenge creation failed")
		}
	}
	startHeight, startCommit := et.edge.StartCommitment()
	log.WithFields(logrus.Fields{
		"name":        et.cfg.validatorName,
		"startHeight": startHeight,
		"startCommit": util.Trunc(startCommit.Bytes()),
	}).Infof("Opened %s subchallenge", subChal.GetType())
	return subChal, nil
}

func (et *edgeTracker) openSubchallengeLeaf(
	ctx context.Context,
	subchallenge protocol.SpecChallenge,
) error {
	topLevelHeight, _, err := subchallenge.TopLevelClaimCommitment(ctx)
	if err != nil {
		return err
	}

	fromAssertionHeight := topLevelHeight
	toAssertionHeight := fromAssertionHeight + 1

	startHeight, _ := et.edge.StartCommitment()
	endHeight, _ := et.edge.TargetCommitment()

	fields := logrus.Fields{
		"name":                et.cfg.validatorName,
		"edgeStartHeight":     startHeight,
		"edgeEndHeight":       endHeight,
		"fromAssertionHeight": fromAssertionHeight,
		"toAssertionHeight":   toAssertionHeight,
	}

	var history util.HistoryCommitment
	switch subchallenge.GetType() {
	case protocol.BigStepChallenge:
		log.WithFields(fields).Info("Big step leaf commit")
		history, err = et.cfg.stateManager.BigStepLeafCommitment(ctx, uint64(fromAssertionHeight), uint64(toAssertionHeight))
	case protocol.SmallStepChallenge:
		log.WithFields(fields).Info("Small step leaf commit")
		history, err = et.cfg.stateManager.SmallStepLeafCommitment(ctx, uint64(fromAssertionHeight), uint64(toAssertionHeight))
	default:
		return errors.New("unsupported subchallenge type for creating leaf commitment")
	}
	if err != nil {
		return err
	}
	addedLeaf, err := subchallenge.AddSubChallengeLevelZeroEdge(ctx, et.edge, history)
	if err != nil {
		return err
	}
	fields["leafFirstState"] = util.Trunc(history.FirstLeaf.Bytes())
	fields["leafHeight"] = history.Height
	fields["leafCommitment"] = util.Trunc(history.Merkle.Bytes())
	fields["subChallengeType"] = subchallenge.GetType()
	log.WithFields(fields).Info("Added subchallenge leaf, now tracking its vertex")
	tracker, err := newEdgeTracker(
		et.cfg,
		subchallenge,
		addedLeaf,
	)
	if err != nil {
		return err
	}
	go tracker.spawn(ctx)
	return nil
}

type edgeTrackerConfig struct {
	actEveryNSeconds      time.Duration
	timeRef               util.TimeReference
	challengePeriodLength time.Duration
	chain                 protocol.Protocol
	stateManager          statemanager.Manager
	validatorName         string
	validatorAddress      common.Address
}

type edgeTracker struct {
	cfg       *edgeTrackerConfig
	challenge protocol.SpecChallenge
	edge      protocol.SpecEdge
	fsm       *util.Fsm[edgeTrackerAction, edgeTrackerState]
}

func newEdgeTracker(
	cfg *edgeTrackerConfig,
	challenge protocol.SpecChallenge,
	edge protocol.SpecEdge,
	fsmOpts ...util.FsmOpt[edgeTrackerAction, edgeTrackerState],
) (*edgeTracker, error) {
	fsm, err := newEdgeTrackerFsm(
		edgeStarted,
		util.WithTrackedTransitions[edgeTrackerAction, edgeTrackerState](),
	)
	if err != nil {
		return nil, err
	}
	return &edgeTracker{
		cfg:       cfg,
		challenge: challenge,
		edge:      edge,
		fsm:       fsm,
	}, nil
}

func (et *edgeTracker) spawn(ctx context.Context) {
	startHeight, startCommit := et.edge.StartCommitment()
	endHeight, endCommit := et.edge.TargetCommitment()
	fields := logrus.Fields{
		"start":         startHeight,
		"end":           endHeight,
		"startMerkle":   util.Trunc(startCommit.Bytes()),
		"endMerkle":     util.Trunc(endCommit.Bytes()),
		"validatorName": et.cfg.validatorName,
		"challengeType": et.challenge.GetType(),
		"address":       util.Trunc(et.cfg.validatorAddress.Bytes()),
	}
	log.WithFields(fields).Info("Tracking edge vertex")

	t := et.cfg.timeRef.NewTicker(et.cfg.actEveryNSeconds)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			// Check if the associated edge or challenge are confirmed,
			// or if a rival edge exists that has been confirmed before acting.
			shouldComplete, err := et.shouldComplete(ctx)
			if err != nil {
				log.WithError(err).WithFields(fields).Error("Could not check if edge tracker should complete")
				continue
			}
			if shouldComplete {
				log.WithFields(fields).Debug("Edge tracker received notice of a confirmation, exiting")
				return
			}
			if err := et.act(ctx); err != nil {
				log.Error(err)
			}
		case <-ctx.Done():
			log.WithFields(fields).Debug("Edge tracker goroutine exiting")
			return
		}
	}
}

func (et *edgeTracker) shouldComplete(ctx context.Context) (bool, error) {
	return false, nil
}
