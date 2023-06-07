package validator

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (v *Manager) challengeAssertion(ctx context.Context, id protocol.AssertionId) error {
	assertion, err := v.chain.GetAssertion(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get assertion to challenge with id %#x", id)
	}

	// We then add a level zero edge to initiate a challenge.
	levelZeroEdge, err := v.addBlockChallengeLevelZeroEdge(ctx, assertion)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists on assertion with sequence num %d",
				id,
			)
			return nil
		}
		return fmt.Errorf("failed to created block challenge layer zero edge: %w", err)
	}

	prevCreationInfo, err := v.chain.ReadAssertionCreationInfo(ctx, id)
	if err != nil {
		return err
	}
	assertionPrevHeight, ok := v.stateManager.ExecutionStateBlockHeight(ctx, protocol.GoExecutionStateFromSolidity(prevCreationInfo.AfterState))
	if !ok {
		return fmt.Errorf("missing previous assertion %v after execution %+v in local state manager", id, prevCreationInfo.AfterState)
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
		prevCreationInfo.InboxMaxCount.Uint64(),
	)
	if err != nil {
		return err
	}
	go tracker.spawn(ctx)

	logFields := logrus.Fields{}
	logFields["name"] = v.name
	logFields["assertionId"] = id
	log.WithFields(logFields).Info("Successfully created level zero edge for block challenge, now tracking")
	return nil
}

func (v *Manager) addBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
) (protocol.SpecEdge, error) {
	prevId := assertion.PrevId()
	prevCreationInfo, err := v.chain.ReadAssertionCreationInfo(ctx, prevId)
	if err != nil {
		return nil, errors.Wrap(err, "could not get assertion creation info")
	}
	startCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, 0)
	if err != nil {
		return nil, err
	}
	endCommit, err := v.stateManager.HistoryCommitmentUpToBatch(
		ctx,
		0,
		protocol.LevelZeroBlockEdgeHeight,
		prevCreationInfo.InboxMaxCount.Uint64(),
	)
	if err != nil {
		return nil, err
	}
	startEndPrefixProof, err := v.stateManager.PrefixProofUpToBatch(
		ctx,
		0,
		0,
		protocol.LevelZeroBlockEdgeHeight,
		prevCreationInfo.InboxMaxCount.Uint64(),
	)
	if err != nil {
		return nil, err
	}
	manager, err := v.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	return manager.AddBlockChallengeLevelZeroEdge(ctx, assertion, startCommit, endCommit, startEndPrefixProof)
}
