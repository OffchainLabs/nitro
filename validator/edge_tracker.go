package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (et *edgeTracker) uniqueTrackerLogFields() logrus.Fields {
	startHeight, startCommit := et.edge.StartCommitment()
	endHeight, endCommit := et.edge.EndCommitment()
	return logrus.Fields{
		"startHeight":   startHeight,
		"startCommit":   util.Trunc(startCommit.Bytes()),
		"endHeight":     endHeight,
		"endCommit":     util.Trunc(endCommit.Bytes()),
		"validatorName": et.cfg.validatorName,
		"challengeType": et.edge.GetType(),
		"address":       util.Trunc(et.cfg.validatorAddress.Bytes()),
	}
}

func (et *edgeTracker) act(ctx context.Context) error {
	fields := et.uniqueTrackerLogFields()
	current := et.fsm.Current()
	switch current.State {
	// Start state.
	case edgeStarted:
		hasRival, err := et.edge.HasRival(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check presumptive")
		}
		if !hasRival {
			return et.fsm.Do(edgeMarkPresumptive{})
		}
		// TODO: Add a conditional to check if we can confirm.
		atOneStepFork, err := et.edge.HasLengthOneRival(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check if edge is at one step fork")
		}
		if atOneStepFork {
			return et.fsm.Do(edgeHandleOneStepFork{})
		}
		return et.fsm.Do(edgeBisect{})
	// Edge is the source of a one-step-fork.
	case edgeAtOneStepFork:
		event, ok := current.SourceEvent.(edgeHandleOneStepFork)
		if !ok {
			return fmt.Errorf("bad source event: %s", event)
		}
		startHeight, startCommit := et.edge.StartCommitment()
		log.WithFields(fields).Infof(
			"Reached one-step-fork at start height %d and start history commitment %s",
			startHeight,
			util.Trunc(startCommit.Bytes()),
		)
		if et.edge.GetType() == protocol.SmallStepChallengeEdge {
			return et.fsm.Do(edgeHandleOneStepProof{})
		}
		return et.fsm.Do(edgeOpenSubchallengeLeaf{})
	// Edge is at a one-step-proof in a small-step challenge.
	case edgeAtOneStepProof:
		log.WithFields(fields).Info("Checking one-step-proof against protocol")
		return et.fsm.Do(edgeHandleOneStepProof{})
	// Edge tracker should add a subchallenge level zero leaf.
	case edgeAddingSubchallengeLeaf:
		event, ok := current.SourceEvent.(edgeOpenSubchallengeLeaf)
		if !ok {
			return fmt.Errorf("bad source event: %s", event)
		}
		if err := et.openSubchallengeLeaf(ctx); err != nil {
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
			log.WithError(err).WithFields(fields).Error("Could not bisect")
			return et.fsm.Do(edgeBackToStart{})
		}
		firstTracker, err := newEdgeTracker(
			et.cfg,
			firstChild,
		)
		if err != nil {
			log.WithError(err).WithFields(fields).Error("Could not create new vertex tracker")
			return et.fsm.Do(edgeBackToStart{})
		}
		secondTracker, err := newEdgeTracker(
			et.cfg,
			secondChild,
		)
		if err != nil {
			log.WithError(err).WithFields(fields).Error("Could not create new vertex tracker")
			return et.fsm.Do(edgeBackToStart{})
		}
		go firstTracker.spawn(ctx)
		go secondTracker.spawn(ctx)
		return et.fsm.Do(edgeAwaitSubchallengeResolution{})
	// Edge is presumptive, should do nothing until it loses ps status.
	case edgePresumptive:
		hasRival, err := et.edge.HasRival(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check if presumptive")
		}
		if hasRival {
			return et.fsm.Do(edgeBackToStart{})
		}
		return et.fsm.Do(edgeMarkPresumptive{})
	case edgeAwaitingSubchallenge:
		// TODO: Perhaps we need an intermediate stage that tries to
		// take confirmation actions if it can.
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
	endHeight, _ := et.edge.EndCommitment()
	bisectTo, err := util.BisectionPoint(uint64(startHeight), uint64(endHeight))
	if err != nil {
		return util.HistoryCommitment{}, nil, errors.Wrapf(err, "determining bisection point failed for %d and %d", startHeight, endHeight)
	}
	if et.edge.GetType() == protocol.BlockChallengeEdge {
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
	var historyCommit util.HistoryCommitment
	var commitErr error
	var proof []byte
	var proofErr error
	switch et.edge.GetType() {
	case protocol.BigStepChallengeEdge:
		originHeights, err := et.edge.TopLevelClaimHeight(ctx)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}

		fromAssertionHeight := uint64(originHeights.BlockChallengeOriginHeight)
		toAssertionHeight := fromAssertionHeight + 1

		historyCommit, commitErr = et.cfg.stateManager.BigStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, bisectTo)
		proof, proofErr = et.cfg.stateManager.BigStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, bisectTo, uint64(endHeight))
	case protocol.SmallStepChallengeEdge:
		originHeights, err := et.edge.TopLevelClaimHeight(ctx)
		if err != nil {
			return util.HistoryCommitment{}, nil, err
		}

		fromAssertionHeight := uint64(originHeights.BlockChallengeOriginHeight)
		toAssertionHeight := fromAssertionHeight + 1
		fromBigStep := uint64(originHeights.BigStepChallengeOriginHeight)
		toBigStep := fromBigStep + 1

		historyCommit, commitErr = et.cfg.stateManager.SmallStepCommitmentUpTo(ctx, fromAssertionHeight, toAssertionHeight, fromBigStep, toBigStep, bisectTo)
		proof, proofErr = et.cfg.stateManager.SmallStepPrefixProof(ctx, fromAssertionHeight, toAssertionHeight, fromBigStep, toBigStep, bisectTo, uint64(endHeight))
	default:
		return util.HistoryCommitment{}, nil, fmt.Errorf("unsupported challenge type: %s", et.edge.GetType())
	}
	if commitErr != nil {
		return util.HistoryCommitment{}, nil, errors.Wrap(commitErr, "could not produce history commitment")
	}
	if proofErr != nil {
		return util.HistoryCommitment{}, nil, errors.Wrap(proofErr, "could not produce prefix proof")
	}
	return historyCommit, proof, nil
}

func (et *edgeTracker) bisect(ctx context.Context) (protocol.SpecEdge, protocol.SpecEdge, error) {
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
			et.cfg.validatorName,
			bisectTo,
			util.Trunc(historyCommit.Merkle.Bytes()),
			endHeight,
			util.Trunc(endCommit.Bytes()),
		)
	}
	log.WithFields(logrus.Fields{
		"name":               et.cfg.validatorName,
		"challengeType":      et.edge.GetType(),
		"bisectedFrom":       endHeight,
		"bisectedFromMerkle": util.Trunc(endCommit.Bytes()),
		"bisectedTo":         bisectTo,
		"bisectedToMerkle":   util.Trunc(historyCommit.Merkle.Bytes()),
	}).Info("Successfully bisected edge")
	return firstChild, secondChild, nil
}

func (et *edgeTracker) openSubchallengeLeaf(ctx context.Context) error {
	originHeights, err := et.edge.TopLevelClaimHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get top level claim height")
	}

	fromAssertionHeight := uint64(originHeights.BlockChallengeOriginHeight)
	toAssertionHeight := fromAssertionHeight + 1

	startHeight, startCommit := et.edge.StartCommitment()
	endHeight, _ := et.edge.EndCommitment()

	fields := logrus.Fields{
		"name":                et.cfg.validatorName,
		"edgeStartHeight":     startHeight,
		"edgeEndHeight":       endHeight,
		"fromAssertionHeight": fromAssertionHeight,
		"toAssertionHeight":   toAssertionHeight,
	}

	var history util.HistoryCommitment
	switch et.edge.GetType() {
	case protocol.BlockChallengeEdge:
		log.WithFields(fields).Info("Big step leaf commit")
		history, err = et.cfg.stateManager.BigStepLeafCommitment(ctx, uint64(fromAssertionHeight), uint64(toAssertionHeight))
	case protocol.BigStepChallengeEdge:
		log.WithFields(fields).Info("Small step leaf commit")
		history, err = et.cfg.stateManager.SmallStepLeafCommitment(ctx, uint64(fromAssertionHeight), uint64(toAssertionHeight), uint64(startHeight), uint64(endHeight))
	default:
		return errors.New("unsupported subchallenge type for creating leaf commitment")
	}
	if err != nil {
		return err
	}
	manager, err := et.cfg.chain.SpecChallengeManager(ctx)
	if err != nil {
		return err
	}
	addedLeaf, err := manager.AddSubChallengeLevelZeroEdge(
		ctx,
		et.edge,
		util.HistoryCommitment{
			Height: uint64(startHeight),
			Merkle: startCommit,
		},
		history,
	)
	if err != nil {
		return err
	}
	fields["leafFirstState"] = util.Trunc(history.FirstLeaf.Bytes())
	fields["leafHeight"] = history.Height
	fields["leafCommitment"] = util.Trunc(history.Merkle.Bytes())
	fields["subChallengeType"] = addedLeaf.GetType()
	log.WithFields(fields).Info("Added subchallenge leaf, now tracking its vertex")
	tracker, err := newEdgeTracker(
		et.cfg,
		addedLeaf,
	)
	if err != nil {
		return err
	}
	go tracker.spawn(ctx)
	return nil
}

type edgeTrackerConfig struct {
	actEveryNSeconds time.Duration
	timeRef          util.TimeReference
	chain            protocol.Protocol
	stateManager     statemanager.Manager
	validatorName    string
	validatorAddress common.Address
}

type edgeTracker struct {
	cfg  *edgeTrackerConfig
	edge protocol.SpecEdge
	fsm  *util.Fsm[edgeTrackerAction, edgeTrackerState]
}

func newEdgeTracker(
	cfg *edgeTrackerConfig,
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
		cfg:  cfg,
		edge: edge,
		fsm:  fsm,
	}, nil
}

func (et *edgeTracker) spawn(ctx context.Context) {
	fields := et.uniqueTrackerLogFields()
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

// TODO(RJ): Implement
func (et *edgeTracker) shouldComplete(ctx context.Context) (bool, error) {
	return false, nil
}
