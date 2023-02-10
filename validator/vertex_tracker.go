package validator

import (
	"context"
	"fmt"
	"reflect"
	"time"

	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/ethereum/go-ethereum/common"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
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
	challenge             protocol.ChallengeInterface
	challengePeriodLength time.Duration
	challengeCreationTime time.Time
	vertex                protocol.ChallengeVertexInterface
	chain                 protocol.OnChainProtocol
	stateManager          statemanager.Manager
	awaitingOneStepFork   bool
	validatorName         string
	validatorAddress      common.Address
}

func newVertexTracker(
	timeRef util.TimeReference,
	actEveryNSeconds time.Duration,
	challenge protocol.ChallengeInterface,
	vertex protocol.ChallengeVertexInterface,
	chain protocol.OnChainProtocol,
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

func (v *vertexTracker) track(ctx context.Context, tx *protocol.ActiveTx) {
	commitment, err := v.vertex.GetCommitment(ctx, tx)
	if err != nil {
		return
	}
	validator, err := v.vertex.GetValidator(ctx, tx)
	if err != nil {
		return
	}
	log.WithFields(logrus.Fields{
		"height":    commitment.Height,
		"merkle":    fmt.Sprintf("%#x", commitment.Merkle),
		"validator": validator,
	}).Info("Tracking challenge vertex")

	t := v.timeRef.NewTicker(v.actEveryNSeconds)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			if err := v.actOnBlockChallenge(ctx, tx); err != nil {
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
func (v *vertexTracker) actOnBlockChallenge(ctx context.Context, tx *protocol.ActiveTx) error {
	if v.awaitingOneStepFork {
		return nil
	}
	// Refresh the vertex by reading it again from the protocol as some of its fields may have changed.
	commitment, err := v.vertex.GetCommitment(ctx, tx)
	if err != nil {
		return err
	}
	vertex, err := v.fetchVertexByHistoryCommit(ctx, protocol.VertexCommitHash(commitment.Hash()))
	if err != nil {
		return errors.Wrap(err, "could not refresh vertex from protocol")
	}
	v.vertex = vertex
	prev, err := v.vertex.GetPrev(ctx, tx)
	if err != nil {
		return err
	}
	if prev.IsNone() {
		return ErrPrevNone
	}
	status, err := v.vertex.GetStatus(ctx, tx)
	if err != nil {
		return err
	}
	if status == protocol.ConfirmedAssertionState {
		return ErrConfirmed
	}
	var challengeCompleted bool
	var siblingConfirmed bool
	if err = v.chain.Call(func(tx *protocol.ActiveTx) error {
		challengeCompleted, err = v.challenge.Completed(ctx, tx)
		if err != nil {
			return nil
		}
		siblingConfirmed, err = v.challenge.HasConfirmedSibling(ctx, tx, v.vertex)
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

	confirmed, err := v.confirmed(ctx, tx)
	if err != nil {
		log.WithError(err).Error("Could not check if vertex is confirmed")
	} else if confirmed {
		return ErrConfirmed
	}

	// We check if we are one-step away from the parent, in which case we then
	// await the resolution of any one-step fork if needed, or confirm once time passes.
	prevCommitment, err := prev.Unwrap().GetCommitment(ctx, tx)
	if err != nil {
		return err
	}
	if commitment.Height == prevCommitment.Height+1 {
		// Check if in a one-step fork.
		atOneStepFork, fetchErr := v.isAtOneStepFork(ctx, tx)
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
		}
		return nil
	}

	// If presumptive, there is no action to take.
	isPresumptive, err := v.vertex.IsPresumptiveSuccessor(ctx, tx)
	if err != nil {
		return err
	}
	if isPresumptive {
		return nil
	}

	log.WithFields(logrus.Fields{
		"height": commitment.Height,
		"merkle": fmt.Sprintf("%#x", commitment.Merkle),
	}).Debugf("Challenge vertex goroutine acting")

	// Determine if we should bisect or merge (how do we determine if we should merge?)
	// Naive idea: if we get vertex already exists during a bisection, then we should attempt a merge move.
	bisectedVertex, err := v.bisect(ctx, tx, vertex)
	if err != nil {
		if errors.Is(err, protocol.ErrVertexAlreadyExists) {
			mergedTo, mergeErr := v.mergeToExistingVertex(ctx, tx)
			if mergeErr != nil {
				return mergeErr
			}
			// Yield tracking of the vertex we merged to in a new goroutine.
			go newVertexTracker(v.timeRef, v.actEveryNSeconds, v.challenge, mergedTo, v.chain, v.stateManager, v.validatorName, v.validatorAddress).track(ctx, tx)
			return nil
		}
		return err
	}

	// Yield tracking of the bisected vertex to a new goroutine.
	go newVertexTracker(v.timeRef, v.actEveryNSeconds, v.challenge, bisectedVertex, v.chain, v.stateManager, v.validatorName, v.validatorAddress).track(ctx, tx)

	return nil
}

// Checks if the vertex is at a one-step-fork.
func (v *vertexTracker) isAtOneStepFork(ctx context.Context, tx *protocol.ActiveTx) (bool, error) {
	var atOneStepFork bool
	var err error
	commitment, err := v.vertex.GetCommitment(ctx, tx)
	if err != nil {
		return false, err
	}
	prev, err := v.vertex.GetPrev(ctx, tx)
	if err != nil {
		return false, err
	}
	prevCommitment, err := prev.Unwrap().GetCommitment(ctx, tx)
	if err != nil {
		return false, err
	}
	challengeParentStateCommitment, err := v.challenge.ParentStateCommitment(ctx, tx)
	if err != nil {
		return false, err
	}
	if err = v.chain.Call(func(tx *protocol.ActiveTx) error {
		atOneStepFork, err = v.chain.IsAtOneStepFork(
			ctx,
			tx,
			protocol.ChallengeCommitHash(challengeParentStateCommitment.Hash()),
			commitment,
			prevCommitment,
		)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false, err
	}
	return atOneStepFork, nil
}

// Obtains a challenge vertex we should perform move into given its corresponding challenge ID
// and the history commitment of the vertex itself from the chain.
func (v *vertexTracker) fetchVertexByHistoryCommit(ctx context.Context, hash protocol.VertexCommitHash) (protocol.ChallengeVertexInterface, error) {
	var mergingTo protocol.ChallengeVertexInterface
	var err error
	if err = v.chain.Call(func(tx *protocol.ActiveTx) error {
		var parentStateCommitment util.StateCommitment
		parentStateCommitment, err = v.challenge.ParentStateCommitment(ctx, tx)
		if err != nil {
			return err
		}
		mergingTo, err = v.chain.ChallengeVertexByCommitHash(tx, protocol.ChallengeCommitHash(parentStateCommitment.Hash()), hash)
		if err != nil {
			return err
		}
		return nil

	}); err != nil {
		return nil, err
	}
	if mergingTo == nil || (reflect.ValueOf(mergingTo).Kind() == reflect.Ptr && reflect.ValueOf(mergingTo).IsNil()) {
		return nil, errors.New("fetched nil challenge vertex from protocol")
	}
	return mergingTo, nil
}

// Merges to a vertex that already exists in the protocol by fetching its history commit
// from our state manager and then performing a merge transaction in the chain. Then,
// this method returns the vertex it merged to.
func (v *vertexTracker) mergeToExistingVertex(ctx context.Context, tx *protocol.ActiveTx) (protocol.ChallengeVertexInterface, error) {
	prev, err := v.vertex.GetPrev(ctx, tx)
	if err != nil {
		return nil, err
	}
	prevCommitment, err := prev.Unwrap().GetCommitment(ctx, tx)
	if err != nil {
		return nil, err
	}
	commitment, err := v.vertex.GetCommitment(ctx, tx)
	if err != nil {
		return nil, err
	}
	parentHeight := prevCommitment.Height
	toHeight := commitment.Height
	mergingToHistory, err := v.determineBisectionPointWithHistory(
		ctx,
		parentHeight,
		toHeight,
	)
	if err != nil {
		return nil, err
	}
	mergingInto, err := v.fetchVertexByHistoryCommit(ctx, protocol.VertexCommitHash(mergingToHistory.Hash()))
	if err != nil {
		return nil, err
	}
	mergingFrom := v.vertex
	parentStateCommitment, err := v.challenge.ParentStateCommitment(ctx, tx)
	if err != nil {
		return nil, err
	}
	mergedTo, err := v.merge(ctx, tx, protocol.ChallengeCommitHash(parentStateCommitment.Hash()), mergingInto, mergingFrom)
	if err != nil {
		return nil, err
	}
	return mergedTo, nil
}

func (v *vertexTracker) confirmed(ctx context.Context, tx *protocol.ActiveTx) (bool, error) {
	// Can't confirm if the vertex is not in correct state.
	status, err := v.vertex.GetStatus(ctx, tx)
	if err != nil {
		return false, err
	}
	if status != protocol.PendingAssertionState {
		return false, nil
	}
	// Can't confirm if parent isn't confirmed, exit early.
	prev, err := v.vertex.GetPrev(ctx, tx)
	if err != nil {
		return false, err
	}
	prevStatus, err := prev.Unwrap().GetStatus(ctx, tx)
	if err != nil {
		return false, err
	}
	if prevStatus != protocol.ConfirmedAssertionState {
		return false, nil
	}

	// Can confirm if vertex's parent has a sub-challenge, and the sub-challenge has reported vertex as its winner.
	subChallenge, err := prev.Unwrap().GetSubChallenge(ctx, tx)
	if err != nil {
		return false, err
	}
	if !subChallenge.IsNone() {
		var subChallengeWinnerVertex util.Option[protocol.ChallengeVertexInterface]
		subChallengeWinnerVertex, err = subChallenge.Unwrap().GetWinnerVertex(ctx, tx)
		if err != nil {
			return false, err
		}
		if !subChallengeWinnerVertex.IsNone() {
			winner := subChallengeWinnerVertex.Unwrap()
			if winner == v.vertex {
				if err = v.chain.Tx(func(tx *protocol.ActiveTx) error {
					return v.vertex.ConfirmForSubChallengeWin(ctx, tx)
				}); err != nil {
					return false, err
				}
				return true, nil
			}
			return false, nil
		}
	}

	// Can confirm if vertex's presumptive successor timer is greater than one challenge period.
	psTimer, err := v.vertex.GetPsTimer(ctx, tx)
	if err != nil {
		return false, err
	}
	if psTimer.Get() > v.challengePeriodLength {
		if err := v.chain.Tx(func(tx *protocol.ActiveTx) error {
			return v.vertex.ConfirmForPsTimer(ctx, tx)
		}); err != nil {
			return false, err
		}
		return true, nil
	}

	// Can confirm if the challenge’s end time has been reached, and vertex is the presumptive successor of parent.
	if v.timeRef.Get().After(v.challengeCreationTime.Add(2 * v.challengePeriodLength)) {
		if err := v.chain.Tx(func(tx *protocol.ActiveTx) error {
			return v.vertex.ConfirmForChallengeDeadline(ctx, tx)
		}); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
