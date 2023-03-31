package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Processes new challenge creation events from the protocol that were not created by self.
// This will fetch the challenge, its parent assertion, and create a challenge leaf that is
// relevant towards resolving the challenge. We then spawn a challenge tracker in the background.
func (v *Validator) onChallengeStarted(
	ctx context.Context, ev protocol.Challenge,
) error {
	challengedAssertion, err := ev.RootAssertion(ctx)
	if err != nil {
		return err
	}
	levelZeroEdge, err := v.addBlockChallengeLevelZeroEdge(ctx, challengedAssertion.SeqNum())
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists on challenge with id %#x",
				ev.Id(),
			)
			return nil
		}
		return err
	}

	challengerName := "unknown-name"
	staker := levelZeroEdge.MiniStaker()
	if name, ok := v.knownValidatorNames[staker.Unwrap()]; ok {
		challengerName = name
	}
	log.WithFields(logrus.Fields{
		"name":                 v.name,
		"challenger":           challengerName,
		"challengingAssertion": fmt.Sprintf("%d", challengedAssertion.SeqNum()),
	}).Warn("Received challenge for a created leaf, added own leaf with history commitment")

	// Start tracking the challenge.
	tracker, err := newEdgeTracker(
		&edgeTrackerConfig{
			timeRef:          v.timeRef,
			actEveryNSeconds: v.challengeVertexWakeInterval,
			chain:            v.chain,
			stateManager:     v.stateManager,
			validatorName:    v.name,
			validatorAddress: v.address,
		},
		levelZeroEdge,
	)
	if err != nil {
		return err
	}
	go tracker.spawn(ctx)
	return nil
}

// Initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (v *Validator) challengeAssertion(ctx context.Context, assertion protocol.Assertion) error {
	var challenge protocol.Challenge
	var err error
	assertionPrevSeqNum, err := assertion.PrevSeqNum()
	if err != nil {
		return err
	}
	// We then add a challenge vertex to the challenge.
	levelZeroEdge, err := v.addBlockChallengeLevelZeroEdge(ctx, assertionPrevSeqNum)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists with challenge hash %#x",
				challenge.Id(),
			)
			return nil
		}
		return err
	}

	// Start tracking the challenge.
	tracker, err := newEdgeTracker(
		&edgeTrackerConfig{
			timeRef:          v.timeRef,
			actEveryNSeconds: v.challengeVertexWakeInterval,
			chain:            v.chain,
			stateManager:     v.stateManager,
			validatorName:    v.name,
			validatorAddress: v.address,
		},
		levelZeroEdge,
	)
	if err != nil {
		return err
	}
	go tracker.spawn(ctx)

	logFields := logrus.Fields{}
	logFields["name"] = v.name
	logFields["parentAssertionSeqNum"], err = assertion.PrevSeqNum()
	if err != nil {
		return err
	}
	log.WithFields(logFields).Info("Successfully created level zero edge for block challenge, now tracking")
	return nil
}

func (v *Validator) addBlockChallengeLevelZeroEdge(
	ctx context.Context,
	prevAssertionSeqNum protocol.AssertionSequenceNumber,
) (protocol.SpecEdge, error) {
	prevAssertion, err := v.chain.AssertionBySequenceNum(ctx, prevAssertionSeqNum)
	if err != nil {
		return nil, err
	}
	latestValidAssertionSeq, err := v.findLatestValidAssertion(ctx)
	if err != nil {
		return nil, err
	}
	assertion, err := v.chain.AssertionBySequenceNum(ctx, latestValidAssertionSeq)
	if err != nil {
		return nil, err
	}
	prevHeight, err := prevAssertion.Height()
	if err != nil {
		return nil, err
	}
	assertionHeight, err := assertion.Height()
	if err != nil {
		return nil, err
	}
	startCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, prevHeight)
	if err != nil {
		return nil, err
	}
	endCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, assertionHeight)
	if err != nil {
		return nil, err
	}
	manager, err := v.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	return manager.AddBlockChallengeLevelZeroEdge(ctx, assertion, startCommit, endCommit)
}
