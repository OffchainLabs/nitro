package validator

import (
	"context"
	"time"

	"fmt"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type vertexTracker struct {
	actEveryNSeconds    time.Duration
	timeRef             util.TimeReference
	challengeCommitHash protocol.CommitHash
	challenge           *protocol.Challenge
	vertex              *protocol.ChallengeVertex
	validator           *Validator
	awaitingOneStepFork bool
}

func newVertexTracker(
	timeRef util.TimeReference,
	actEveryNSeconds time.Duration,
	challenge *protocol.Challenge,
	vertex *protocol.ChallengeVertex,
	validator *Validator,
) *vertexTracker {
	commitHash := protocol.CommitHash(challenge.ParentStateCommitment().Hash())
	return &vertexTracker{
		timeRef:             timeRef,
		actEveryNSeconds:    actEveryNSeconds,
		challengeCommitHash: commitHash,
		challenge:           challenge,
		vertex:              vertex,
		validator:           validator,
	}
}

func (v *vertexTracker) track(ctx context.Context) {
	log.WithFields(logrus.Fields{
		"height":    v.vertex.Commitment.Height,
		"merkle":    fmt.Sprintf("%#x", v.vertex.Commitment.Merkle),
		"validator": v.vertex.Validator,
	}).Info("Tracking challenge vertex")
	t := v.timeRef.NewTicker(v.actEveryNSeconds)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			if err := v.actOnBlockChallenge(ctx); err != nil {
				log.Error(err)
			}
		case <-ctx.Done():
			log.WithFields(logrus.Fields{
				"height": v.vertex.Commitment.Height,
				"merkle": fmt.Sprintf("%#x", v.vertex.Commitment.Merkle),
			}).Debug("Challenge goroutine exiting")
			return
		}
	}
}

// TODO: Add a condition that exits the whole vertex tracker (close the goroutine) once the vertex:
// (a) is confirmed, or
// (b) another vertex with a height >= ours is confirmed in the protocol.
// TODO: Add a condition that determines when the vertex is at a one step fork is resolved (can check some data from parent)
// TODO: Add a condition that checks if we should take a confirmation action.
func (v *vertexTracker) actOnBlockChallenge(ctx context.Context) error {
	if v.awaitingOneStepFork {
		return nil
	}
	vertex, err := v.refreshVertex()
	if err != nil {
		return err
	}
	v.vertex = vertex

	// We check if we are one-step away from the parent, in which case we then
	// await the resolution of any one-step fork if needed, or confirm once time passes.
	// TODO: Add a condition that confirms the vertex after a certain amount of time
	// and the parent has been confirmed.
	if v.vertex.Commitment.Height == v.vertex.Prev.Commitment.Height+1 {
		// Check if in a one-step fork.
		atOneStepFork, fetchErr := v.isAtOneStepFork()
		if fetchErr != nil {
			return fetchErr
		}
		if atOneStepFork {
			log.WithField("name", v.validator.name).Infof(
				"Reached one-step-fork at %d %#x, now tracking subchallenge resolution",
				v.vertex.Prev.Commitment.Height, v.vertex.Prev.Commitment.Merkle,
			)
			v.awaitingOneStepFork = true
			return nil
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
		"height": v.vertex.Commitment.Height,
		"merkle": fmt.Sprintf("%#x", v.vertex.Commitment.Merkle),
	}).Debugf("Challenge vertex goroutine acting")

	// Determine if we should bisect or merge (how do we determine if we should merge?)
	// Naive idea: if we get vertex already exists during a bisection, then we should attempt a merge move.
	bisectedVertex, err := v.validator.bisect(ctx, vertex)
	if err != nil {
		if errors.Is(err, protocol.ErrVertexAlreadyExists) {
			parentHeight := v.vertex.Prev.Commitment.Height
			toHeight := v.vertex.Commitment.Height
			mergingToHistory, err2 := v.validator.determineBisectionPointWithHistory(
				ctx,
				parentHeight,
				toHeight,
			)
			if err2 != nil {
				return err2
			}
			mergingInto, err3 := v.fetchVertexByHistoryCommit(
				v.validator,
				v.challengeCommitHash,
				mergingToHistory,
			)
			if err3 != nil {
				return err3
			}
			mergingFrom := v.vertex
			mergedVertex, err4 := v.validator.merge(ctx, v.challengeCommitHash, mergingInto, mergingFrom)
			if err4 != nil {
				return err4
			}
			// Yield tracking of the vertex we merged to in a new goroutine.
			go newVertexTracker(
				v.timeRef,
				v.actEveryNSeconds,
				v.challenge,
				mergedVertex,
				v.validator,
			).track(ctx)
			return nil
		}
		return err
	}

	// Yield tracking of the bisected vertex to a new goroutine.
	go newVertexTracker(
		v.timeRef,
		v.actEveryNSeconds,
		v.challenge,
		bisectedVertex,
		v.validator,
	).track(ctx)

	return nil
}

// Retrieves the latest vertex from the protocol to refresh its status.
func (v *vertexTracker) refreshVertex() (*protocol.ChallengeVertex, error) {
	var vertex *protocol.ChallengeVertex
	var err error
	if err = v.validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		vertex, err = p.ChallengeVertexBySequenceNum(tx, v.challengeCommitHash, v.vertex.SequenceNum)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return vertex, nil
}

// Checks if the vertex is at a one-step-fork.
func (v *vertexTracker) isAtOneStepFork() (bool, error) {
	var atOneStepFork bool
	var err error
	if err = v.validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		atOneStepFork, err = p.IsAtOneStepFork(
			tx,
			v.challengeCommitHash,
			v.vertex.Commitment,
			v.vertex.Prev.Commitment,
		)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return atOneStepFork, err
	}
	return atOneStepFork, nil
}

// Obtains a challenge vertex we should perform move into given its corresponding challenge ID
// and the history commitment of the vertex itself from the chain.
func (v *vertexTracker) fetchVertexByHistoryCommit(
	validator *Validator,
	vertexChallengeID protocol.CommitHash,
	historyCommit util.HistoryCommitment,
) (*protocol.ChallengeVertex, error) {
	var mergingTo *protocol.ChallengeVertex
	var err error
	if err = validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		mergingTo, err = p.ChallengeVertexByHistoryCommit(tx, vertexChallengeID, historyCommit)
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
