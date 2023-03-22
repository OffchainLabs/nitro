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

var (
	ErrPrevNone = errors.New("vertex parent is none")
)

type vertexTrackerConfig struct {
	actEveryNSeconds      time.Duration
	timeRef               util.TimeReference
	challengePeriodLength time.Duration
	challengeCreationTime time.Time
	chain                 protocol.Protocol
	stateManager          statemanager.Manager
	validatorName         string
	validatorAddress      common.Address
}

type vertexTracker struct {
	cfg       *vertexTrackerConfig
	challenge protocol.Challenge
	vertex    protocol.ChallengeVertex
	fsm       *util.Fsm[vertexTrackerAction, vertexTrackerState]
}

func newVertexTracker(
	cfg *vertexTrackerConfig,
	challenge protocol.Challenge,
	vertex protocol.ChallengeVertex,
	fsmOpts ...util.FsmOpt[vertexTrackerAction, vertexTrackerState],
) (*vertexTracker, error) {
	fsm, err := newVertexTrackerFsm(trackerStarted, util.WithTrackedTransitions[vertexTrackerAction, vertexTrackerState]())
	if err != nil {
		return nil, err
	}
	return &vertexTracker{
		cfg:       cfg,
		challenge: challenge,
		vertex:    vertex,
		fsm:       fsm,
	}, nil
}

func (v *vertexTracker) spawn(ctx context.Context) {
	commitment := v.vertex.HistoryCommitment()
	fields := logrus.Fields{
		"height":        commitment.Height,
		"merkle":        util.Trunc(commitment.Merkle[:]),
		"validatorName": v.cfg.validatorName,
		"challengeType": v.challenge.GetType(),
		"address":       util.Trunc(v.cfg.validatorAddress.Bytes()),
	}
	log.WithFields(fields).Info("Tracking challenge vertex")

	t := v.cfg.timeRef.NewTicker(v.cfg.actEveryNSeconds)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			// Check if the associated vertex or challenge are confirmed,
			// or if a rival vertex exists that has been confirmed before acting.
			shouldComplete, err := v.trackerShouldComplete(ctx)
			if err != nil {
				log.WithError(err).WithFields(fields).Error("Could not check if vertex tracker should complete")
				continue
			}
			if shouldComplete {
				log.WithFields(fields).Debug("Vertex tracker received notice of a confirmation, exiting")
				return
			}
			if err := v.act(ctx); err != nil {
				log.Error(err)
			}
		case <-ctx.Done():
			log.WithFields(fields).Debug("Challenge goroutine exiting")
			return
		}
	}
}

func (vt *vertexTracker) trackerShouldComplete(ctx context.Context) (bool, error) {
	challengeCompleted, err := vt.challenge.Completed(ctx)
	if err != nil {
		return false, nil
	}
	siblingConfirmed, err := vt.vertex.HasConfirmedSibling(ctx)
	if err != nil {
		return false, nil
	}
	return challengeCompleted || siblingConfirmed, nil
}

func (vt *vertexTracker) act(ctx context.Context) error {
	current := vt.fsm.Current()
	switch current.State {
	case trackerStarted:
		prevVertex, err := vt.prevVertex(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get prev")
		}
		atOneStepFork, err := vt.checkOneStepFork(ctx, prevVertex)
		if err != nil {
			return errors.Wrap(err, "could not check one step fork")
		}
		isPresumptive, err := vt.isPresumptive(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check presumptive")
		}
		if atOneStepFork {
			return vt.fsm.Do(actOneStepFork{
				forkPointVertex: prevVertex,
			})
		}
		if isPresumptive {
			return vt.fsm.Do(markPresumptive{})
		}
		return vt.fsm.Do(bisect{})
	case trackerAtOneStepFork:
		event, ok := current.SourceEvent.(actOneStepFork)
		if !ok {
			return fmt.Errorf("bad source event: %s", event)
		}
		forkPointVertexHistoryCommitment := event.forkPointVertex.HistoryCommitment()
		log.WithFields(logrus.Fields{
			"name": vt.cfg.validatorName,
		}).Infof(
			"Reached one-step-fork at %d and commitment %s",
			forkPointVertexHistoryCommitment.Height,
			util.Trunc(forkPointVertexHistoryCommitment.Merkle.Bytes()),
		)
		challengeType := vt.challenge.GetType()
		if challengeType == protocol.SmallStepChallenge {
			return vt.fsm.Do(actOneStepProof{})
		}
		return vt.fsm.Do(openSubchallenge{
			challengeForkVertex: event.forkPointVertex,
		})
	case trackerAtOneStepProof:
		log.WithFields(logrus.Fields{
			"name": vt.cfg.validatorName,
		}).Info("Checking one-step-proof against protocol")
		return vt.fsm.Do(actOneStepProof{})
	case trackerOpeningSubchallenge:
		event, ok := current.SourceEvent.(openSubchallenge)
		if !ok {
			return fmt.Errorf("bad source event: %s", event)
		}
		subChallenge, err := vt.openSubchallenge(ctx, event.challengeForkVertex)
		if err != nil {
			return err
		}
		return vt.fsm.Do(openSubchallengeLeaf{
			subChallenge:    subChallenge,
			forkPointVertex: event.challengeForkVertex,
		})
	case trackerAddingSubchallengeLeaf:
		event, ok := current.SourceEvent.(openSubchallengeLeaf)
		if !ok {
			return fmt.Errorf("bad source event: %s", event)
		}
		if err := vt.openSubchallengeLeaf(
			ctx, event.forkPointVertex, event.subChallenge,
		); err != nil {
			return errors.Wrap(err, "could not open subchallenge leaf")
		}
		return vt.fsm.Do(awaitSubchallengeResolution{})
	case trackerBisecting:
		bisectedTo, err := vt.bisect(ctx, vt.vertex)
		if err != nil {
			if errors.Is(err, solimpl.ErrAlreadyExists) {
				return vt.fsm.Do(merge{})
			}
			log.WithError(err).WithFields(logrus.Fields{
				"height":        vt.vertex.HistoryCommitment().Height,
				"merkle":        util.Trunc(vt.vertex.HistoryCommitment().Merkle.Bytes()),
				"validatorName": vt.cfg.validatorName,
				"challengeType": vt.challenge.GetType(),
				"address":       util.Trunc(vt.cfg.validatorAddress.Bytes()),
			}).Error("could not bisect")
			return vt.fsm.Do(backToStart{})
		}
		tracker, err := newVertexTracker(
			vt.cfg,
			vt.challenge,
			bisectedTo,
		)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"height":        vt.vertex.HistoryCommitment().Height,
				"merkle":        util.Trunc(vt.vertex.HistoryCommitment().Merkle.Bytes()),
				"validatorName": vt.cfg.validatorName,
				"challengeType": vt.challenge.GetType(),
				"address":       util.Trunc(vt.cfg.validatorAddress.Bytes()),
			}).Error("could not create new vertex tracker")
			return vt.fsm.Do(backToStart{})
		}
		go tracker.spawn(ctx)
		return vt.fsm.Do(backToStart{})
	case trackerMerging:
		mergedTo, err := vt.mergeToExistingVertex(ctx)
		if err != nil {
			return errors.Wrap(err, "could not merge")
		}
		tracker, err := newVertexTracker(
			vt.cfg,
			vt.challenge,
			mergedTo,
		)
		if err != nil {
			return errors.Wrap(err, "could not create new vertex tracker")
		}
		go tracker.spawn(ctx)
		return vt.fsm.Do(backToStart{})
	case trackerConfirming:
		// TODO: Implement.
		return vt.fsm.Do(confirmWinner{})
	case trackerPresumptive:
		// Terminal state does nothing. The vertex tracker will end next time it acts.
		isPs, err := vt.isPresumptive(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check if presumptive")
		}
		if !isPs {
			return vt.fsm.Do(backToStart{})
		}
		return vt.fsm.Do(markPresumptive{})
	case trackerAwaitingSubchallengeResolution:
		// Terminal state does nothing. The vertex tracker will end next time it acts.
		return vt.fsm.Do(awaitSubchallengeResolution{})
	default:
		return fmt.Errorf("invalid state: %s", current.State)
	}
}

func (vt *vertexTracker) isPresumptive(ctx context.Context) (bool, error) {
	isPresumptive, err := vt.vertex.IsPresumptiveSuccessor(ctx)
	if err != nil {
		return false, err
	}
	return isPresumptive, nil
}

func (vt *vertexTracker) checkOneStepFork(ctx context.Context, prevVertex protocol.ChallengeVertex) (bool, error) {
	commitment := vt.vertex.HistoryCommitment()
	prevCommitment := prevVertex.HistoryCommitment()
	if commitment.Height != prevCommitment.Height+1 {
		return false, nil
	}
	oneStepFork, err := prevVertex.ChildrenAreAtOneStepFork(ctx)
	if err != nil {
		return false, err
	}
	return oneStepFork, nil
}

func (vt *vertexTracker) prevVertex(ctx context.Context) (protocol.ChallengeVertex, error) {
	prevV, err := vt.vertex.Prev(ctx)
	if err != nil {
		return nil, err
	}
	if prevV.IsNone() {
		return nil, errors.Wrapf(ErrPrevNone, "vertex with id: %#x", vt.vertex.Id())
	}
	return prevV.Unwrap(), nil
}

// Merges to a vertex that already exists in the protocol by fetching its history commit
// from our state manager and then performing a merge transaction in the chain. Then,
// this method returns the vertex it merged to.
func (v *vertexTracker) mergeToExistingVertex(ctx context.Context) (protocol.ChallengeVertex, error) {
	prev, err := v.vertex.Prev(ctx)
	if err != nil {
		return nil, err
	}
	if prev.IsNone() {
		return nil, errors.New("no prev vertex found")
	}
	prevCommitment := prev.Unwrap().HistoryCommitment()
	commitment := v.vertex.HistoryCommitment()
	parentHeight := prevCommitment.Height
	toHeight := commitment.Height

	mergeHistory, prefixProof, err := v.determineBisectionHistoryWithProof(
		ctx,
		parentHeight,
		toHeight,
	)
	if err != nil {
		return nil, err
	}
	return v.merge(ctx, mergeHistory, prefixProof)
}

// Opens a subchallenge on a parent vertex. This function determines the type of subchallenge
// that should be opened, and then the tracker attempts to submit a subchallenge creation
// on-chain and return its value. If the subchallenge already exists, it will instead fetch
// the challenge and return its value.
func (v *vertexTracker) openSubchallenge(
	ctx context.Context,
	prevVertex protocol.ChallengeVertex,
) (protocol.Challenge, error) {
	if v.challenge.GetType() == protocol.SmallStepChallenge {
		return nil, errors.New("cannot create subchallenge on small step challenge")
	}
	manager, err := v.cfg.chain.CurrentChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	var subChalToCreate protocol.ChallengeType
	switch v.challenge.GetType() {
	case protocol.BlockChallenge:
		subChalToCreate = protocol.BigStepChallenge
	case protocol.BigStepChallenge:
		subChalToCreate = protocol.SmallStepChallenge
	default:
		return nil, errors.New("unsupported challenge type to create")
	}
	subChal, err := prevVertex.CreateSubChallenge(ctx)
	if err != nil {
		switch {
		case errors.Is(err, solimpl.ErrAlreadyExists):
			subChalHash, err := manager.CalculateChallengeHash(ctx, prevVertex.Id(), subChalToCreate)
			if err != nil {
				return nil, err
			}
			fetchedSubChal, err := manager.GetChallenge(ctx, subChalHash)
			if err != nil {
				return nil, err
			}
			if fetchedSubChal.IsNone() {
				return nil, fmt.Errorf("no subchallenge found on-chain for id %#x", subChalHash)
			}
			subChal = fetchedSubChal.Unwrap()
		default:
			return nil, errors.Wrap(err, "subchallenge creation failed")
		}
	}
	log.WithFields(logrus.Fields{
		"name":   v.cfg.validatorName,
		"height": prevVertex.HistoryCommitment().Height,
		"merkle": util.Trunc(prevVertex.HistoryCommitment().Merkle.Bytes()),
	}).Infof("Opened %s subchallenge", subChal.GetType())
	return subChal, nil
}

func (vt *vertexTracker) openSubchallengeLeaf(
	ctx context.Context,
	prevVertex protocol.ChallengeVertex,
	subChallenge protocol.Challenge,
) error {
	fromVertexHeight := prevVertex.HistoryCommitment().Height
	toVertexHeight := vt.vertex.HistoryCommitment().Height

	topLevelClaimVertex, err := subChallenge.TopLevelClaimVertex(ctx)
	if err != nil {
		return err
	}

	fromAssertionHeight := topLevelClaimVertex.HistoryCommitment().Height
	toAssertionHeight := fromAssertionHeight + 1

	var history util.HistoryCommitment
	switch subChallenge.GetType() {
	case protocol.BigStepChallenge:
		log.WithFields(logrus.Fields{
			"name":                vt.cfg.validatorName,
			"fromVertexHeight":    fromVertexHeight,
			"toVertexHeight":      toVertexHeight,
			"fromAssertionHeight": fromAssertionHeight,
			"toAssertionHeight":   toAssertionHeight,
		}).Info("Big step leaf commit")
		history, err = vt.cfg.stateManager.BigStepLeafCommitment(ctx, fromAssertionHeight, toAssertionHeight)
	case protocol.SmallStepChallenge:
		log.WithFields(logrus.Fields{
			"name":                vt.cfg.validatorName,
			"fromVertexHeight":    fromVertexHeight,
			"toVertexHeight":      toVertexHeight,
			"fromAssertionHeight": fromAssertionHeight,
			"toAssertionHeight":   toAssertionHeight,
		}).Info("Small step leaf commit")
		history, err = vt.cfg.stateManager.SmallStepLeafCommitment(ctx, fromAssertionHeight, toAssertionHeight)
	default:
		return errors.New("unsupported subchallenge type for creating leaf commitment")
	}
	if err != nil {
		return err
	}
	addedLeaf, err := subChallenge.AddSubChallengeLeaf(ctx, vt.vertex, history)
	if err != nil {
		return err
	}
	log.WithFields(logrus.Fields{
		"name":                      vt.cfg.validatorName,
		"upperLevelForkPoint":       prevVertex.HistoryCommitment().Height,
		"upperLevelForkPointMerkle": util.Trunc(prevVertex.HistoryCommitment().Merkle.Bytes()),
		"height":                    addedLeaf.HistoryCommitment().Height,
		"leafFirstState":            util.Trunc(history.FirstLeaf.Bytes()),
		"leafCommitment":            util.Trunc(addedLeaf.HistoryCommitment().Merkle.Bytes()),
		"subChallengeType":          subChallenge.GetType(),
	}).Info("Added subchallenge leaf, now tracking its vertex")
	tracker, err := newVertexTracker(
		vt.cfg,
		subChallenge,
		addedLeaf,
	)
	if err != nil {
		return err
	}
	go tracker.spawn(ctx)
	return nil
}

// TODO: Unused - need to refactor into something more manageable.
// TODO: Refactor as this function does too much. A vertex tracker should only be responsible
// for confirming its own vertex, not subchallenge vertices.
// nolint:unused
func (v *vertexTracker) confirmed(ctx context.Context) (bool, error) {
	// Can't confirm if the vertex is not in correct state.
	status, err := v.vertex.Status(ctx)
	if err != nil {
		return false, err
	}
	if status != protocol.AssertionPending {
		return false, nil
	}

	// Can't confirm if parent isn't confirmed, exit early.
	prev, err := v.vertex.Prev(ctx)
	if err != nil {
		return false, err
	}
	if prev.IsNone() {
		return false, errors.New("no prev vertex")
	}
	prevStatus, err := prev.Unwrap().Status(ctx)
	if err != nil {
		return false, err
	}
	// TODO: Vertex status different from assertion status.
	if prevStatus != protocol.AssertionConfirmed {
		return false, nil
	}

	// Can confirm if vertex's parent has a sub-challenge, and the sub-challenge has reported vertex as its winner.
	subChallenge, err := prev.Unwrap().GetSubChallenge(ctx)
	if err != nil {
		return false, err
	}
	if !subChallenge.IsNone() {
		var subChallengeWinnerVertex util.Option[protocol.ChallengeVertex]
		subChallengeWinnerVertex, err = subChallenge.Unwrap().WinnerVertex(ctx)
		if err != nil {
			return false, err
		}
		if !subChallengeWinnerVertex.IsNone() {
			winner := subChallengeWinnerVertex.Unwrap()
			if winner == v.vertex {
				if confirmErr := v.vertex.ConfirmForSubChallengeWin(ctx); confirmErr != nil {
					return false, confirmErr
				}
				return true, nil
			}
			return false, nil
		}
	}

	// Can confirm if vertex's presumptive successor timer is greater than one challenge period.
	psTimer, err := v.vertex.PsTimer(ctx)
	if err != nil {
		return false, err
	}
	if time.Duration(psTimer)*time.Second > v.cfg.challengePeriodLength {
		if confirmErr := v.vertex.ConfirmForPsTimer(ctx); confirmErr != nil {
			return false, err
		}
		return true, nil
	}

	// Can confirm if the challenge’s end time has been reached, and vertex is the presumptive successor of parent.
	if v.cfg.timeRef.Get().After(v.cfg.challengeCreationTime.Add(2 * v.cfg.challengePeriodLength)) {
		if confirmErr := v.vertex.ConfirmForChallengeDeadline(ctx); confirmErr != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
