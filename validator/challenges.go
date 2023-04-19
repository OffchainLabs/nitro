package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (v *Validator) challengeAssertion(ctx context.Context, assertion protocol.Assertion) error {
	assertionPrevSeqNum, err := assertion.PrevSeqNum()
	if err != nil {
		return err
	}
	assertionPrev, err := v.chain.AssertionBySequenceNum(ctx, assertionPrevSeqNum)
	if err != nil {
		return err
	}
	assertionPrevHeight, err := assertionPrev.Height()
	if err != nil {
		return err
	}
	assertionPrevMsgCountSeen, err := assertionPrev.InboxMsgCountSeen()
	if err != nil {
		return err
	}
	// We then add a challenge vertex to the challenge.
	levelZeroEdge, err := v.addBlockChallengeLevelZeroEdge(ctx, assertionPrevSeqNum)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists on assertion with sequence num %d",
				assertionPrevSeqNum,
			)
			return nil
		}
		return err
	}

	// Start tracking the challenge.
	tracker, err := newEdgeTracker(
		ctx,
		&edgeTrackerConfig{
			timeRef:          v.timeRef,
			actEveryNSeconds: v.edgeTrackerWakeInterval,
			chain:            v.chain,
			stateManager:     v.stateManager,
			validatorName:    v.name,
			validatorAddress: v.address,
		},
		levelZeroEdge,
		assertionPrevHeight,
		assertionPrevMsgCountSeen,
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
	inboxMaxCount, err := prevAssertion.InboxMsgCountSeen()
	if err != nil {
		return nil, err
	}
	startCommit, err := v.stateManager.HistoryCommitmentUpToBatch(ctx, prevHeight, prevHeight, inboxMaxCount)
	if err != nil {
		return nil, err
	}
	prevStateHash, err := prevAssertion.StateHash()
	if err != nil {
		return nil, err
	}
	if startCommit.FirstLeaf != prevStateHash {
		return nil, fmt.Errorf("start state at height %v has hash %v locally but %v in assertion", prevHeight, startCommit.FirstLeaf, prevStateHash)
	}
	endCommit, err := v.stateManager.HistoryCommitmentUpToBatch(ctx, prevHeight, prevHeight+protocol.LayerZeroBlockEdgeHeight, inboxMaxCount)
	if err != nil {
		return nil, err
	}
	endStateHash, err := assertion.StateHash()
	if err != nil {
		return nil, err
	}
	if endCommit.LastLeaf != endStateHash {
		return nil, fmt.Errorf("end state has hash %v locally but %v in assertion", endCommit.LastLeaf, endStateHash)
	}
	startEndPrefixProof, err := v.stateManager.PrefixProofUpToBatch(ctx, prevHeight, prevHeight, prevHeight+protocol.LayerZeroBlockEdgeHeight, inboxMaxCount)
	if err != nil {
		return nil, err
	}
	manager, err := v.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	return manager.AddBlockChallengeLevelZeroEdge(ctx, assertion, startCommit, endCommit, startEndPrefixProof)
}
