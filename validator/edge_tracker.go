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
		bisectedTo, err := et.bisect(ctx)
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
		tracker, err := newEdgeTracker(
			et.cfg,
			et.challenge,
			bisectedTo,
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
		go tracker.spawn(ctx)
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

func (et *edgeTracker) bisect(ctx context.Context) (protocol.SpecEdge, error) {
	return nil, nil
}

func (et *edgeTracker) openSubchallenge(ctx context.Context) (protocol.SpecChallenge, error) {
	return nil, nil
}

func (et *edgeTracker) openSubchallengeLeaf(
	ctx context.Context,
	subchallenge protocol.SpecChallenge,
) error {
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
