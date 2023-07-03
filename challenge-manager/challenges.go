// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

package challengemanager

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	edgetracker "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/challenge-protocol-v2/containers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ChallengeManager defines an offchain, challenge manager, which will be
// an active participant in interacting with the on-chain contracts.
type ChallengeManager interface {
	ChallengeCreator
	ChallengeModeReader
}

// ChallengeCreator defines a struct which can initiate a challenge on an assertion hash
// by creating a level zero, block challenge edge onchain.
type ChallengeCreator interface {
	ChallengeAssertion(ctx context.Context, id protocol.AssertionHash) error
}

// ChallengeModeReader defines a struct which can read the challenge mode of a challenge manager.
type ChallengeModeReader interface {
	Mode() Mode
}

// Initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (m *Manager) ChallengeAssertion(ctx context.Context, id protocol.AssertionHash) error {
	assertion, err := m.chain.GetAssertion(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get assertion to challenge with id %#x", id)
	}

	// We then add a level zero edge to initiate a challenge.
	levelZeroEdge, creationInfo, err := m.addBlockChallengeLevelZeroEdge(ctx, assertion)
	if err != nil {
		return fmt.Errorf("could not add block challenge level zero edge %v: %w", m.name, err)
	}
	if !creationInfo.InboxMaxCount.IsUint64() {
		return errors.New("assertion creation info inbox max count was not a uint64")
	}
	// Start tracking the challenge.
	tracker, err := edgetracker.New(
		ctx,
		levelZeroEdge,
		m.chain,
		m.stateManager,
		m.watcher,
		m,
		edgetracker.HeightConfig{
			StartBlockHeight:           0,
			TopLevelClaimEndBatchCount: creationInfo.InboxMaxCount.Uint64(),
		},
		edgetracker.WithActInterval(m.edgeTrackerWakeInterval),
		edgetracker.WithTimeReference(m.timeRef),
		edgetracker.WithValidatorName(m.name),
		edgetracker.WithValidatorAddress(m.address),
	)
	if err != nil {
		return err
	}
	go tracker.Spawn(ctx)

	logFields := logrus.Fields{}
	logFields["name"] = m.name
	logFields["assertionHash"] = containers.Trunc(id[:])
	log.WithFields(logFields).Info("Successfully created level zero edge for block challenge")
	return nil
}

func (m *Manager) addBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
) (protocol.SpecEdge, *protocol.AssertionCreatedInfo, error) {
	creationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, assertion.Id())
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get assertion creation info")
	}
	if !creationInfo.InboxMaxCount.IsUint64() {
		return nil, nil, errors.New("creation info inbox max count was not a uint64")
	}
	startCommit, err := m.stateManager.HistoryCommitmentUpTo(ctx, 0)
	if err != nil {
		return nil, nil, err
	}
	manager, err := m.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, nil, err
	}
	levelZeroBlockEdgeHeight, err := manager.LevelZeroBlockEdgeHeight(ctx)
	if err != nil {
		return nil, nil, err
	}
	endCommit, err := m.stateManager.HistoryCommitmentUpToBatch(
		ctx,
		0,
		levelZeroBlockEdgeHeight,
		creationInfo.InboxMaxCount.Uint64(),
	)
	if err != nil {
		return nil, nil, err
	}
	startEndPrefixProof, err := m.stateManager.PrefixProofUpToBatch(
		ctx,
		0,
		0,
		levelZeroBlockEdgeHeight,
		creationInfo.InboxMaxCount.Uint64(),
	)
	if err != nil {
		return nil, nil, err
	}
	edge, err := manager.AddBlockChallengeLevelZeroEdge(ctx, assertion, startCommit, endCommit, startEndPrefixProof)
	if err != nil {
		return nil, nil, err
	}
	return edge, creationInfo, nil
}
