package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Processes new challenge creation events from the protocol that were not initiated by other validators.
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
	// Checks if the challenge has to do with a vertex we created.
	v.leavesLock.RLock()
	leaf, ok := v.createdLeaves[ev.ParentStateCommitment.StateRoot]
	if !ok {
		v.leavesLock.RUnlock()
		// TODO: Act on the honest vertices even if this challenge does not have to do with us by
		// keeping track of associated challenge vertices' clocks and acting if the associated
		// staker we agree with is not performing their responsibilities on time. As an honest
		// validator, we should participate in confirming valid assertions.
		return nil
	}
	v.leavesLock.RUnlock()
	challengerName := "unknown-name"
	if !leaf.Staker.IsNone() {
		if name, ok := v.knownValidatorNames[leaf.Staker.Unwrap()]; ok {
			challengerName = name
		} else {
			challengerName = leaf.Staker.Unwrap().Hex()
		}
	}
	log.WithFields(logrus.Fields{
		"name":                 v.name,
		"challenger":           challengerName,
		"challengingStateRoot": fmt.Sprintf("%#x", leaf.StateCommitment.StateRoot),
		"challengingHeight":    leaf.StateCommitment.Height,
	}).Warn("Received challenge for a created leaf")

	historyCommit, err := v.stateManager.LatestHistoryCommitment(ctx)
	if err != nil {
		return err
	}

	// We then add a leaf to the challenge using a historical commitment at our latest height.
	var challenge *protocol.Challenge
	var challengeVertex *protocol.ChallengeVertex
	if err = v.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		parentAssertion, fetchErr := p.AssertionBySequenceNum(tx, ev.ParentSeqNum)
		if fetchErr != nil {
			return err
		}
		challenge, err = p.ChallengeByAssertionStateCommit(
			tx,
			protocol.AssertionStateCommitHash(parentAssertion.StateCommitment.Hash()),
		)
		if err != nil {
			return err
		}
		// TODO: Ideally, check if the leaf already exists before making the tx at all.
		challengeVertex, err = challenge.AddLeaf(tx, parentAssertion, historyCommit, v.address)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		if errors.Is(err, protocol.ErrVertexAlreadyExists) {
			log.Info(
				"Attempted to add a challenge leaf with history: height=%d, merkle=%#x but "+
					"it has already been created",
				historyCommit.Height,
				historyCommit.Merkle,
			)
			return nil
		}
		return errors.Wrapf(
			err,
			"could add challenge vertex to challenge with parent sequence number: %d",
			ev.ParentSeqNum,
		)
	}

	// TODO: Start tracking the challenge.
	_ = challengeVertex

	return nil
}

// Initiates a challenge on a leaf added to the assertion protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (v *Validator) challengeLeaf(ctx context.Context, ev *protocol.CreateLeafEvent) error {
	var parentAssertion *protocol.Assertion
	var currentAssertion *protocol.Assertion
	var err error
	if err = v.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		parentAssertion, err = p.AssertionBySequenceNum(tx, ev.PrevSeqNum)
		if err != nil {
			return err
		}
		currentAssertion, err = p.AssertionBySequenceNum(tx, ev.SeqNum)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	challenge, err := v.submitOrFetchProtocolChallenge(ctx, parentAssertion)
	if err != nil {
		return err
	}

	// We produce a historical commiment to add a leaf to the initiated challenge
	// by retrieving it from our local state manager.
	historyCommit, err := v.stateManager.LatestHistoryCommitment(ctx)
	if err != nil {
		return err
	}

	var challengeVertex *protocol.ChallengeVertex
	if err = v.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		challengeVertex, err = challenge.AddLeaf(tx, currentAssertion, historyCommit, v.address)
		if err != nil {
			return errors.Wrap(err, "cannot add leaf")
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "could not add leaf to challenge")
	}

	// TODO: Start tracking the challenge.
	_ = challengeVertex

	logFields := logrus.Fields{}
	logFields["name"] = v.name
	logFields["parentAssertionSeqNum"] = parentAssertion.SequenceNum
	logFields["parentAssertionStateRoot"] = fmt.Sprintf("%#x", parentAssertion.StateCommitment.StateRoot)
	logFields["challengeID"] = fmt.Sprintf("%#x", parentAssertion.StateCommitment.Hash())
	log.WithFields(logFields).Info("Successfully created challenge and added leaf, now tracking events")

	return nil
}

// Tries to submit a challenge to the protocol or retrieve it if it already exists.
// based on the parent assertion's state commitment hash.
func (v *Validator) submitOrFetchProtocolChallenge(
	ctx context.Context,
	parentAssertion *protocol.Assertion,
) (*protocol.Challenge, error) {
	var challenge *protocol.Challenge
	var err error
	err = v.chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		challenge, err = parentAssertion.CreateChallenge(tx, ctx, v.address)
		if err != nil {
			return errors.Wrap(err, "cannot make challenge")
		}
		return nil
	})
	switch {
	case errors.Is(err, protocol.ErrChallengeAlreadyExists):
		log.Info("Challenge on leaf already exists, reading existing challenge from protocol")
		if err = v.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
			challenge, err = p.ChallengeByAssertionStateCommit(tx, protocol.AssertionStateCommitHash(parentAssertion.StateCommitment.Hash()))
			if err != nil {
				return errors.Wrap(err, "cannot make challenge")
			}
			return nil
		}); err != nil {
			return nil, errors.Wrap(err, "could not get challenge by ID")
		}
	case err != nil:
		return nil, errors.Wrapf(
			err,
			"could not initiate challenge on assertion with seq num %d",
			parentAssertion.SequenceNum,
		)
	default:
	}
	if challenge == nil {
		return nil, errors.New("got nil challenge from protocol")
	}
	return challenge, nil
}
