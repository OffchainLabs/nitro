package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Processes new challenge creation events from the protocol that were not created by self.
// This will fetch the challenge, its parent assertion, and create a challenge leaf that is
// relevant towards resolving the challenge. We then spawn a challenge tracker in the background.
func (v *Validator) onChallengeStarted(
	ctx context.Context, ev *protocol.StartChallengeEvent,
) error {
	if ev == nil {
		return nil
	}
	// Ignore challenges initiated by self.
	if isFromSelf(v.address, ev.Validator) {
		return nil
	}

	challenge, err := v.fetchProtocolChallenge(
		ctx,
		ev.ParentSeqNum,
		ev.ParentStateCommitment,
	)
	if err != nil {
		return err
	}

	// We then add a challenge vertex to the challenge.
	challengeVertex, err := v.addChallengeVertex(ctx, challenge)
	if err != nil {
		if errors.Is(err, protocol.ErrVertexAlreadyExists) {
			log.Infof(
				"Attempted to add a challenge leaf that already exists to challenge with "+
					"parent state commit: height=%d, stateRoot=%#x",
				challenge.ParentStateCommitment().Height,
				challenge.ParentStateCommitment().StateRoot,
			)
			return nil
		}
		return err
	}

	challengerName := "unknown-name"
	staker := challengeVertex.GetValidator()
	if name, ok := v.knownValidatorNames[staker]; ok {
		challengerName = name
	}
	log.WithFields(logrus.Fields{
		"name":                 v.name,
		"challenger":           challengerName,
		"challengingStateRoot": fmt.Sprintf("%#x", challenge.ParentStateCommitment().StateRoot),
		"challengingHeight":    challenge.ParentStateCommitment().Height,
	}).Warn("Received challenge for a created leaf, added own leaf with history commitment")

	// Start tracking the challenge.
	go newVertexTracker(v.timeRef, v.challengeVertexWakeInterval, challenge, challengeVertex, v.chain, v.stateManager, v.name, v.address).track(ctx)

	return nil
}

// Initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (v *Validator) challengeAssertion(ctx context.Context, ev *protocol.CreateLeafEvent) error {
	var challenge protocol.ChallengeInterface
	var err error
	challenge, err = v.submitProtocolChallenge(ctx, ev.PrevSeqNum)
	if err != nil {
		if errors.Is(err, protocol.ErrChallengeAlreadyExists) {
			existingChallenge, fetchErr := v.fetchProtocolChallenge(ctx, ev.PrevSeqNum, ev.PrevStateCommitment)
			if fetchErr != nil {
				return fetchErr
			}
			challenge = existingChallenge
		} else {
			return err
		}
	}

	// We then add a challenge vertex to the challenge.
	challengeVertex, err := v.addChallengeVertex(ctx, challenge)
	if err != nil {
		return err
	}
	if errors.Is(err, protocol.ErrVertexAlreadyExists) {
		log.Infof(
			"Attempted to add a challenge leaf that already exists to challenge with "+
				"parent state commit: height=%d, stateRoot=%#x",
			challenge.ParentStateCommitment().Height,
			challenge.ParentStateCommitment().StateRoot,
		)
		return nil
	}

	// Start tracking the challenge.
	go newVertexTracker(v.timeRef, v.challengeVertexWakeInterval, challenge, challengeVertex, v.chain, v.stateManager, v.name, v.address).track(ctx)

	logFields := logrus.Fields{}
	logFields["name"] = v.name
	logFields["parentAssertionSeqNum"] = ev.PrevSeqNum
	logFields["parentAssertionStateRoot"] = fmt.Sprintf("%#x", ev.PrevStateCommitment.StateRoot)
	logFields["challengeID"] = fmt.Sprintf("%#x", ev.PrevStateCommitment.Hash())
	log.WithFields(logFields).Info("Successfully created challenge and added leaf, now tracking events")

	return nil
}

func (v *Validator) verifyAddLeafConditions(a *protocol.Assertion, c protocol.ChallengeInterface) error {
	if a.Prev.IsNone() {
		return errors.Wrap(protocol.ErrInvalidOp, "Can not add leaf on root assertion")
	}
	if a.Prev.Unwrap() != c.RootAssertion() {
		return errors.Wrap(protocol.ErrInvalidOp, "Challenge and assertion parent mismatch")
	}
	if err := v.chain.Call(func(tx *protocol.ActiveTx) error {
		if c.Completed(tx) {
			return errors.New("Challenge has been completed")
		}
		return nil
	}); err != nil {
		return errors.Wrap(protocol.ErrInvalidOp, err.Error())
	}
	if !c.RootVertex().EligibleForNewSuccessor() {
		return errors.Wrap(protocol.ErrInvalidOp, "Root vertex is not eligible for new successor")
	}
	return nil
}

func (v *Validator) addChallengeVertex(
	ctx context.Context,
	challenge protocol.ChallengeInterface,
) (protocol.ChallengeVertexInterface, error) {
	latestValidAssertionSeq := v.findLatestValidAssertion(ctx)

	var assertion *protocol.Assertion
	var err error
	if err = v.chain.Call(func(tx *protocol.ActiveTx) error {
		assertion, err = v.chain.AssertionBySequenceNum(tx, latestValidAssertionSeq)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	historyCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, assertion.StateCommitment.Height)
	if err != nil {
		return nil, err
	}

	var challengeVertex protocol.ChallengeVertexInterface
	if err = v.chain.Tx(func(tx *protocol.ActiveTx) error {
		challengeVertex, err = challenge.AddLeaf(tx, assertion, historyCommit, v.address)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, errors.Wrapf(
			err,
			"could add challenge vertex to challenge with parent state commitment: height=%d, stateRoot=%#x",
			challenge.ParentStateCommitment().Height,
			challenge.ParentStateCommitment().StateRoot,
		)
	}
	return challengeVertex, nil
}

func (v *Validator) submitProtocolChallenge(
	ctx context.Context,
	parentAssertionSeqNum protocol.AssertionSequenceNumber,
) (protocol.ChallengeInterface, error) {
	var challenge protocol.ChallengeInterface
	var err error
	if err = v.chain.Tx(func(tx *protocol.ActiveTx) error {
		parentAssertion, readErr := v.chain.AssertionBySequenceNum(tx, parentAssertionSeqNum)
		if readErr != nil {
			return readErr
		}
		challenge, err = parentAssertion.CreateChallenge(tx, ctx, v.address)
		if err != nil {
			return errors.Wrap(err, "could not submit challenge")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return challenge, nil
}

// Tries to retrieve a challenge from the protocol on-chain
// based on the parent assertion's state commitment hash.
func (v *Validator) fetchProtocolChallenge(
	ctx context.Context,
	parentAssertionSeqNum protocol.AssertionSequenceNumber,
	parentAssertionCommit util.StateCommitment,
) (*protocol.Challenge, error) {
	var err error
	var challenge *protocol.Challenge
	if err = v.chain.Call(func(tx *protocol.ActiveTx) error {
		challenge, err = v.chain.ChallengeByCommitHash(
			tx,
			protocol.ChallengeCommitHash(parentAssertionCommit.Hash()),
		)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "could not get challenge from protocol")
	}
	if challenge == nil {
		return nil, errors.New("got nil challenge from protocol")
	}
	return challenge, nil
}
