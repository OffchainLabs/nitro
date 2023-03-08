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
	ErrConfirmed          = errors.New("Vertex has been confirmed")
	ErrSiblingConfirmed   = errors.New("Vertex sibling has been confirmed")
	ErrPrevNone           = errors.New("Vertex parent is none")
	ErrChallengeCompleted = errors.New("Challenge has been completed")
)

type vertexTracker struct {
	actEveryNSeconds      time.Duration
	timeRef               util.TimeReference
	challenge             protocol.Challenge
	challengePeriodLength time.Duration
	challengeCreationTime time.Time
	vertex                protocol.ChallengeVertex
	chain                 protocol.Protocol
	stateManager          statemanager.Manager
	awaitingOneStepFork   bool
	validatorName         string
	validatorAddress      common.Address
	fsm                   *util.Fsm[vertexTrackerAction, vertexTrackerState]
}

func newVertexTracker(
	timeRef util.TimeReference,
	actEveryNSeconds time.Duration,
	challenge protocol.Challenge,
	vertex protocol.ChallengeVertex,
	chain protocol.Protocol,
	stateManager statemanager.Manager,
	validatorName string,
	validatorAddress common.Address,
) *vertexTracker {
	return &vertexTracker{
		timeRef:          timeRef,
		actEveryNSeconds: actEveryNSeconds,
		challenge:        challenge,
		vertex:           vertex,
		chain:            chain,
		stateManager:     stateManager,
		validatorName:    validatorName,
		validatorAddress: validatorAddress,
	}
}

func (v *vertexTracker) track(ctx context.Context) {
	commitment := v.vertex.HistoryCommitment()
	miniStakerAddr := v.vertex.MiniStaker()
	log.WithFields(logrus.Fields{
		"height":     commitment.Height,
		"merkle":     fmt.Sprintf("%#x", commitment.Merkle),
		"miniStaker": miniStakerAddr,
	}).Info("Tracking challenge vertex")

	t := v.timeRef.NewTicker(v.actEveryNSeconds)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			if err := v.actOnBlockChallenge(ctx); err != nil {
				switch {
				case errors.Is(err, ErrConfirmed):
					return
				case errors.Is(err, ErrSiblingConfirmed):
					return
				case errors.Is(err, ErrChallengeCompleted):
					return
				case errors.Is(err, ErrPrevNone):
					return
				default:
					log.Error(err)
				}
			}
		case <-ctx.Done():
			log.WithFields(logrus.Fields{
				"height": commitment.Height,
				"merkle": fmt.Sprintf("%#x", commitment.Merkle),
			}).Debug("Challenge goroutine exiting")
			return
		}
	}
}

// TODO: Add a condition that determines when the vertex is at a one-step-fork is resolved (can check some data from parent)
func (v *vertexTracker) actOnBlockChallenge(ctx context.Context) error {
	if v.awaitingOneStepFork {
		return nil
	}
	// Refresh the vertex by reading it again from the protocol as some of its fields may have changed.
	vertex, err := v.fetchVertexByHistoryCommit(ctx, v.vertex.Id())
	if err != nil {
		return errors.Wrap(err, "could not refresh vertex from protocol")
	}
	v.vertex = vertex
	var challengeCompleted bool
	var siblingConfirmed bool
	var prev protocol.ChallengeVertex
	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		prevV, err2 := v.vertex.Prev(ctx, tx)
		if err2 != nil {
			return err2
		}
		if prevV.IsNone() {
			return ErrPrevNone
		}
		prev = prevV.Unwrap()
		status := v.vertex.Status()
		if status == protocol.AssertionConfirmed {
			return ErrConfirmed
		}
		challengeCompleted, err = v.challenge.Completed(ctx, tx)
		if err != nil {
			return nil
		}
		siblingConfirmed, err = v.vertex.HasConfirmedSibling(ctx, tx)
		if err != nil {
			return nil
		}
		return nil
	}); err != nil {
		return err
	}
	if challengeCompleted {
		return ErrChallengeCompleted
	}
	if siblingConfirmed {
		return ErrSiblingConfirmed
	}

	confirmed, err := v.confirmed(ctx)
	if err != nil {
		//log.WithError(err).Error("Could not check if vertex is confirmed")
		//return err
		_ = err
	}
	if confirmed {
		return ErrConfirmed
	}

	// We check if we are one-step away from the parent, in which case we then
	// await the resolution of any one-step fork if needed, or confirm once time passes.
	commitment := v.vertex.HistoryCommitment()
	prevCommitment := prev.HistoryCommitment()
	var isPresumptive bool

	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		if commitment.Height == prevCommitment.Height+1 {
			// Check if in a one-step fork.
			atOneStepFork, fetchErr := prev.ChildrenAreAtOneStepFork(ctx, tx)
			if fetchErr != nil {
				return fetchErr
			}
			if atOneStepFork {
				log.WithField("name", v.validatorName).Infof(
					"Reached one-step-fork at %d %#x, now tracking subchallenge resolution",
					prevCommitment.Height, prevCommitment.Merkle,
				)
				v.awaitingOneStepFork = true
				// TODO: Add subchallenge resolution.
				return nil
			}
		}
		isPresumptive, err = v.vertex.IsPresumptiveSuccessor(ctx, tx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	if v.awaitingOneStepFork {
		return nil
	}

	// If presumptive, there is no action to take.
	if isPresumptive {
		return nil
	}

	log.WithFields(logrus.Fields{
		"height": commitment.Height,
		"merkle": fmt.Sprintf("%#x", commitment.Merkle),
	}).Debugf("Challenge vertex goroutine acting")

	// Determine if we should bisect or merge (how do we determine if we should merge?)
	// Naive idea: if we get vertex already exists during a bisection, then we should attempt a merge move.
	bisectedVertex, err := v.bisect(ctx, vertex)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			mergedTo, mergeErr := v.mergeToExistingVertex(ctx)
			if mergeErr != nil {
				return mergeErr
			}
			// Yield tracking of the vertex we merged to in a new goroutine.
			go newVertexTracker(v.timeRef, v.actEveryNSeconds, v.challenge, mergedTo, v.chain, v.stateManager, v.validatorName, v.validatorAddress).track(ctx)
			return nil
		}
		return err
	}

	// Yield tracking of the bisected vertex to a new goroutine.
	go newVertexTracker(v.timeRef, v.actEveryNSeconds, v.challenge, bisectedVertex, v.chain, v.stateManager, v.validatorName, v.validatorAddress).track(ctx)

	return nil
}

// Obtains a challenge vertex we should perform move into given its corresponding challenge ID
// and the history commitment of the vertex itself from the chain.
func (v *vertexTracker) fetchVertexByHistoryCommit(ctx context.Context, hash protocol.VertexHash) (protocol.ChallengeVertex, error) {
	var mergingTo util.Option[protocol.ChallengeVertex]
	var err error
	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		manager, err2 := v.chain.CurrentChallengeManager(ctx, tx)
		if err2 != nil {
			return err2
		}
		mergingTo, err = manager.GetVertex(ctx, tx, hash)
		if err != nil {
			return err
		}
		return nil

	}); err != nil {
		return nil, err
	}
	if mergingTo.IsNone() {
		return nil, errors.New("fetched nil challenge vertex from protocol")
	}
	return mergingTo.Unwrap(), nil
}

// Merges to a vertex that already exists in the protocol by fetching its history commit
// from our state manager and then performing a merge transaction in the chain. Then,
// this method returns the vertex it merged to.
func (v *vertexTracker) mergeToExistingVertex(ctx context.Context) (protocol.ChallengeVertex, error) {
	var prev protocol.ChallengeVertex
	var mergingInto protocol.ChallengeVertex
	var parentCommit util.StateCommitment
	if err := v.chain.Call(func(tx protocol.ActiveTx) error {
		prevV, err := v.vertex.Prev(ctx, tx)
		if err != nil {
			return err
		}
		if prevV.IsNone() {
			return errors.New("no prev vertex found")
		}
		prev = prevV.Unwrap()
		parentStateCommitment, err := v.challenge.ParentStateCommitment(ctx, tx)
		if err != nil {
			return err
		}
		prevCommitment := prev.HistoryCommitment()
		commitment := v.vertex.HistoryCommitment()
		parentHeight := prevCommitment.Height
		toHeight := commitment.Height

		mergingToHistory, err := v.determineBisectionPointWithHistory(
			ctx,
			parentHeight,
			toHeight,
		)
		if err != nil {
			return err
		}
		manager, err := v.chain.CurrentChallengeManager(ctx, tx)
		if err != nil {
			return err
		}
		vertexId, err := manager.CalculateChallengeVertexId(ctx, tx, v.challenge.Id(), mergingToHistory)
		if err != nil {
			return err
		}
		vertex, err := manager.GetVertex(ctx, tx, vertexId)
		if err != nil {
			return err
		}
		if vertex.IsNone() {
			return errors.New("no vertex found to merge into")
		}
		mergingInto = vertex.Unwrap()
		parentCommit = parentStateCommitment
		return nil
	}); err != nil {
		return nil, err
	}
	mergingFrom := v.vertex
	mergedTo, err := v.merge(ctx, protocol.ChallengeHash(parentCommit.Hash()), mergingInto, mergingFrom)
	if err != nil {
		return nil, err
	}
	return mergedTo, nil
}

func (v *vertexTracker) confirmed(ctx context.Context) (bool, error) {
	// Can't confirm if the vertex is not in correct state.
	status := v.vertex.Status()
	if status != protocol.AssertionPending {
		return false, nil
	}

	var gotConfirmed bool

	if err := v.chain.Tx(func(tx protocol.ActiveTx) error {
		// Can't confirm if parent isn't confirmed, exit early.
		prev, err := v.vertex.Prev(ctx, tx)
		if err != nil {
			return err
		}
		if prev.IsNone() {
			return errors.New("no prev vertex")
		}
		prevStatus := prev.Unwrap().Status()
		// TODO: Vertex status different from assertion status.
		if prevStatus != protocol.AssertionConfirmed {
			return nil
		}

		// Can confirm if vertex's parent has a sub-challenge, and the sub-challenge has reported vertex as its winner.
		subChallenge, err := prev.Unwrap().GetSubChallenge(ctx, tx)
		if err != nil {
			return err
		}
		if !subChallenge.IsNone() {
			var subChallengeWinnerVertex util.Option[protocol.ChallengeVertex]
			subChallengeWinnerVertex, err = subChallenge.Unwrap().WinnerVertex(ctx, tx)
			if err != nil {
				return err
			}
			if !subChallengeWinnerVertex.IsNone() {
				winner := subChallengeWinnerVertex.Unwrap()
				if winner == v.vertex {
					if confirmErr := v.vertex.ConfirmForSubChallengeWin(ctx, tx); confirmErr != nil {
						return confirmErr
					}
					gotConfirmed = true
				}
				return nil
			}
		}

		// Can confirm if vertex's presumptive successor timer is greater than one challenge period.
		psTimer, err := v.vertex.PsTimer(ctx, tx)
		if err != nil {
			return err
		}
		if time.Duration(psTimer)*time.Second > v.challengePeriodLength {
			if confirmErr := v.vertex.ConfirmForPsTimer(ctx, tx); confirmErr != nil {
				return err
			}
			gotConfirmed = true
			return nil
		}

		// Can confirm if the challenge’s end time has been reached, and vertex is the presumptive successor of parent.
		if v.timeRef.Get().After(v.challengeCreationTime.Add(2 * v.challengePeriodLength)) {
			if confirmErr := v.vertex.ConfirmForChallengeDeadline(ctx, tx); confirmErr != nil {
				return err
			}
			gotConfirmed = true
		}
		return nil
	}); err != nil {
		return false, err
	}
	return gotConfirmed, nil
}
