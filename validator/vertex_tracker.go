package validator

import (
	"context"
	"fmt"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/ethereum/go-ethereum/common"
	"time"

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
	chain                 protocol.ChainReadWriter
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
	chain protocol.ChainReadWriter,
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
	log.WithFields(logrus.Fields{
		"height":    v.vertex.GetCommitment().Height,
		"merkle":    fmt.Sprintf("%#x", v.vertex.GetCommitment().Merkle),
		"validator": v.vertex.GetValidator(),
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
				"height": v.vertex.GetCommitment().Height,
				"merkle": fmt.Sprintf("%#x", v.vertex.GetCommitment().Merkle),
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
	vertex, err := v.fetchVertexByHistoryCommit(protocol.VertexCommitHash(v.vertex.GetCommitment().Hash()))
	if err != nil {
		return errors.Wrap(err, "could not refresh vertex from protocol")
	}
	v.vertex = vertex
	if v.vertex.GetPrev().IsNone() {
		return ErrPrevNone
	}
	if v.vertex.GetStatus() == protocol.ConfirmedAssertionState {
		return ErrConfirmed
	}
	var challengeCompleted bool
	var siblingConfirmed bool
	if err = v.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		challengeCompleted = v.challenge.Completed(tx)
		siblingConfirmed = v.challenge.HasConfirmedSibling(tx, v.vertex)
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

	confirmed, err := v.confirmed()
	if err != nil {
		log.WithError(err).Error("Could not check if vertex is confirmed")
	} else if confirmed {
		return ErrConfirmed
	}

	// We check if we are one-step away from the parent, in which case we then
	// await the resolution of any one-step fork if needed, or confirm once time passes.
	if v.vertex.GetCommitment().Height == v.vertex.GetPrev().Unwrap().GetCommitment().Height+1 {
		// Check if in a one-step fork.
		atOneStepFork, fetchErr := v.isAtOneStepFork()
		if fetchErr != nil {
			return fetchErr
		}
		if atOneStepFork {
			log.WithField("name", v.validatorName).Infof(
				"Reached one-step-fork at %d %#x, now tracking subchallenge resolution",
				v.vertex.GetPrev().Unwrap().GetCommitment().Height, v.vertex.GetPrev().Unwrap().GetCommitment().Merkle,
			)
			v.awaitingOneStepFork = true
			// TODO: Add subchallenge resolution.
		}
		return nil
	}

	// If presumptive, there is no action to take.
	isPresumptive := v.vertex.IsPresumptiveSuccessor()
	if isPresumptive {
		return nil
	}

	log.WithFields(logrus.Fields{
		"height": v.vertex.GetCommitment().Height,
		"merkle": fmt.Sprintf("%#x", v.vertex.GetCommitment().Merkle),
	}).Debugf("Challenge vertex goroutine acting")

	// Determine if we should bisect or merge (how do we determine if we should merge?)
	// Naive idea: if we get vertex already exists during a bisection, then we should attempt a merge move.
	bisectedVertex, err := v.bisect(ctx, vertex)
	if err != nil {
		if errors.Is(err, protocol.ErrVertexAlreadyExists) {
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

// Checks if the vertex is at a one-step-fork.
func (v *vertexTracker) isAtOneStepFork() (bool, error) {
	var atOneStepFork bool
	var err error
	if err = v.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		atOneStepFork, err = p.IsAtOneStepFork(
			tx,
			protocol.ChallengeCommitHash(v.challenge.ParentStateCommitment().Hash()),
			v.vertex.GetCommitment(),
			v.vertex.GetPrev().Unwrap().GetCommitment(),
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
func (v *vertexTracker) fetchVertexByHistoryCommit(hash protocol.VertexCommitHash) (protocol.ChallengeVertexInterface, error) {
	var mergingTo protocol.ChallengeVertexInterface
	var err error
	if err = v.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		mergingTo, err = p.ChallengeVertexByCommitHash(tx, protocol.ChallengeCommitHash(v.challenge.ParentStateCommitment().Hash()), hash)
		if err != nil {
			return err
		}
		return nil

	}); err != nil {
		return nil, err
	}
	if mergingTo == nil {
		return nil, errors.New("fetched nil challenge vertex from protocol")
	}
	return mergingTo, nil
}

// Merges to a vertex that already exists in the protocol by fetching its history commit
// from our state manager and then performing a merge transaction in the chain. Then,
// this method returns the vertex it merged to.
func (v *vertexTracker) mergeToExistingVertex(ctx context.Context) (protocol.ChallengeVertexInterface, error) {
	parentHeight := v.vertex.GetPrev().Unwrap().GetCommitment().Height
	toHeight := v.vertex.GetCommitment().Height
	mergingToHistory, err := v.determineBisectionPointWithHistory(
		ctx,
		parentHeight,
		toHeight,
	)
	if err != nil {
		return nil, err
	}
	mergingInto, err := v.fetchVertexByHistoryCommit(protocol.VertexCommitHash(mergingToHistory.Hash()))
	if err != nil {
		return nil, err
	}
	mergingFrom := v.vertex
	mergedTo, err := v.merge(ctx, protocol.ChallengeCommitHash(v.challenge.ParentStateCommitment().Hash()), mergingInto, mergingFrom)
	if err != nil {
		return nil, err
	}
	return mergedTo, nil
}

func (v *vertexTracker) confirmed() (bool, error) {
	// Can't confirm if the vertex is not in correct state.
	if v.vertex.GetStatus() != protocol.PendingAssertionState {
		return false, nil
	}
	// Can't confirm if parent isn't confirmed, exit early.
	if v.vertex.GetPrev().Unwrap().GetStatus() != protocol.ConfirmedAssertionState {
		return false, nil
	}

	// Can confirm if vertex's parent has a sub-challenge, and the sub-challenge has reported vertex as its winner.
	subChallenge := v.vertex.GetPrev().Unwrap().GetSubChallenge()
	if !subChallenge.IsNone() && !subChallenge.Unwrap().GetWinnerVertex().IsNone() {
		winner := subChallenge.Unwrap().GetWinnerVertex().Unwrap()
		if winner == v.vertex {
			if err := v.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
				return v.vertex.ConfirmForSubChallengeWin(tx)
			}); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	}

	// Can confirm if vertex's presumptive successor timer is greater than one challenge period.
	if v.vertex.GetPsTimer().Get() > v.challengePeriodLength {
		if err := v.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
			return v.vertex.ConfirmForPsTimer(tx)
		}); err != nil {
			return false, err
		}
		return true, nil
	}

	// Can confirm if the challenge’s end time has been reached, and vertex is the presumptive successor of parent.
	if v.timeRef.Get().After(v.challengeCreationTime.Add(2 * v.challengePeriodLength)) {
		if err := v.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
			return v.vertex.ConfirmForChallengeDeadline(tx)
		}); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
