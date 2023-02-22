package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
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
		ev.ParentStateHash,
	)
	if err != nil {
		return err
	}

	// We then add a challenge vertex to the challenge.
	challengeVertex, err := v.addChallengeVertex(ctx, challenge)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists with state hash %#x",
				ev.ParentStateHash,
			)
			return nil
		}
		return err
	}

	challengerName := "unknown-name"
	staker := challengeVertex.MiniStaker()
	if name, ok := v.knownValidatorNames[staker]; ok {
		challengerName = name
	}
	parentStateCommitment, err := challenge.ParentStateCommitment(ctx, tx)
	if err != nil {
		return err
	}
	log.WithFields(logrus.Fields{
		"name":                 v.name,
		"challenger":           challengerName,
		"challengingStateRoot": fmt.Sprintf("%#x", parentStateCommitment.StateRoot),
		"challengingHeight":    parentStateCommitment.Height,
	}).Warn("Received challenge for a created leaf, added own leaf with history commitment")

	// Start tracking the challenge.
	go newVertexTracker(v.timeRef, v.challengeVertexWakeInterval, challenge, challengeVertex, v.chain, v.stateManager, v.name, v.address).track(ctx)

	return nil
}

// Initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (v *Validator) challengeAssertion(ctx context.Context, ev *protocol.CreateLeafEvent) error {
	var challenge protocol.Challenge
	var err error
	challenge, err = v.submitProtocolChallenge(ctx, ev.PrevSeqNum)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			existingChallenge, fetchErr := v.fetchProtocolChallenge(ctx, ev.PrevSeqNum, ev.PrevStateHash)
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
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists with height %d and state hash %#x",
				ev.Height,
				ev.StateHash,
			)
			return nil
		}
		return err
	}

	// Start tracking the challenge.
	go newVertexTracker(v.timeRef, v.challengeVertexWakeInterval, challenge, challengeVertex, v.chain, v.stateManager, v.name, v.address).track(ctx)

	logFields := logrus.Fields{}
	logFields["name"] = v.name
	logFields["parentAssertionSeqNum"] = ev.PrevSeqNum
	logFields["parentAssertionStateRoot"] = fmt.Sprintf("%#x", ev.PrevStateHash)
	// TODO: Compute challenge ID properly.
	//logFields["challengeID"] = fmt.Sprintf("%#x", ev.PrevStateCommitment.Hash())
	log.WithFields(logFields).Info("Successfully created challenge and added leaf, now tracking events")

	return nil
}

func (v *Validator) verifyAddLeafConditions(ctx context.Context, a protocol.Assertion, c protocol.Challenge) error {
	return v.chain.Call(func(tx protocol.ActiveTx) error {
		prev, err := v.chain.AssertionBySequenceNum(ctx, tx, a.PrevSeqNum())
		if err != nil {
			return err
		}
		rootAssertion, err := c.RootAssertion(ctx, tx)
		if err != nil {
			return err
		}
		if prev != rootAssertion {
			return errors.New("challenge and assertion parent mismatch")
		}
		completed, err := c.Completed(ctx, tx)
		if err != nil {
			return nil
		}
		if completed {
			return errors.New("challenge has been completed")
		}
		rootVertex, err := c.RootVertex(ctx, tx)
		if err != nil {
			return err
		}
		eligibleForNewSuccessor, err := rootVertex.EligibleForNewSuccessor(ctx, tx)
		if err != nil {
			return err
		}
		if !eligibleForNewSuccessor {
			return errors.New("root vertex is not eligible for new successor")
		}
		return nil
	})
}

func (v *Validator) addChallengeVertex(
	ctx context.Context,
	challenge protocol.Challenge,
) (protocol.ChallengeVertex, error) {
	latestValidAssertionSeq := v.findLatestValidAssertion(ctx)

	var assertion protocol.Assertion
	var err error
	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		assertion, err = v.chain.AssertionBySequenceNum(tx, latestValidAssertionSeq)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	historyCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, assertion.Height())
	if err != nil {
		return nil, err
	}

	var challengeVertex protocol.ChallengeVertex
	if err = v.chain.Tx(func(tx protocol.ActiveTx) error {
		challengeVertex, err = challenge.AddLeaf(ctx, tx, assertion, historyCommit, v.address)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		parentStateCommitment, _ := challenge.ParentStateCommitment(ctx, tx)
		return nil, errors.Wrapf(
			err,
			"could add challenge vertex to challenge with parent state commitment: height=%d, stateRoot=%#x",
			parentStateCommitment.Height,
			parentStateCommitment.StateRoot,
		)
	}
	return challengeVertex, nil
}

func (v *Validator) submitProtocolChallenge(
	ctx context.Context,
	parentAssertionSeqNum protocol.AssertionSequenceNumber,
) (protocol.Challenge, error) {
	var challenge protocol.Challenge
	var err error
	if err = v.chain.Tx(func(tx protocol.ActiveTx) error {
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
	parentStateHash common.Hash,
) (protocol.Challenge, error) {
	var err error
	var challenge util.Option[protocol.Challenge]
	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		manager, err := v.chain.CurrentChallengeManager(ctx, tx)
		if err != nil {
			return err
		}
		challenge, err = manager.GetChallenge(
			ctx,
			tx,
			// TODO: Compute challenge hash.
			protocol.ChallengeHash(parentAssertionCommit.Hash()),
		)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "could not get challenge from protocol")
	}
	if challenge.IsNone() {
		return nil, errors.New("got nil challenge from protocol")
	}
	return challenge.Unwrap(), nil
}
