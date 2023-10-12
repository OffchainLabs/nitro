// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package challengemanager

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/bold/containers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// ChallengeAssertion initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (m *Manager) ChallengeAssertion(ctx context.Context, id protocol.AssertionHash) error {
	assertion, err := m.chain.GetAssertion(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get assertion to challenge with id %#x", id)
	}

	// We then add a level zero edge to initiate a challenge.
	levelZeroEdge, edgeTrackerAssertionInfo, err := m.addBlockChallengeLevelZeroEdge(ctx, assertion)
	if err != nil {
		return fmt.Errorf("could not add block challenge level zero edge %v: %w", m.name, err)
	}
	if verifiedErr := m.watcher.AddVerifiedHonestEdge(ctx, levelZeroEdge); verifiedErr != nil {
		fields := log.Ctx{
			"edgeId": levelZeroEdge.Id(),
			"err":    verifiedErr,
		}
		srvlog.Error("could not add verified honest edge to chain watcher", fields)
	}
	// Start tracking the challenge.
	tracker, err := edgetracker.New(
		ctx,
		levelZeroEdge,
		m.chain,
		m.stateManager,
		m.watcher,
		m,
		edgeTrackerAssertionInfo,
		edgetracker.WithActInterval(m.edgeTrackerWakeInterval),
		edgetracker.WithTimeReference(m.timeRef),
		edgetracker.WithValidatorName(m.name),
	)
	if err != nil {
		return err
	}
	go tracker.Spawn(ctx)

	srvlog.Info("Successfully created level zero edge for block challenge", log.Ctx{
		"name":          m.name,
		"assertionHash": containers.Trunc(id.Bytes()),
		"fromBatch":     edgeTrackerAssertionInfo.FromBatch,
		"toBatch":       edgeTrackerAssertionInfo.ToBatch,
	})
	return nil
}

func (m *Manager) addBlockChallengeLevelZeroEdge(
	ctx context.Context,
	assertion protocol.Assertion,
) (protocol.VerifiedHonestEdge, *edgetracker.AssociatedAssertionMetadata, error) {
	creationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, assertion.Id())
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get assertion creation info")
	}
	prevCreationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: creationInfo.ParentAssertionHash})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get assertion creation info")
	}
	fromBatch := l2stateprovider.Batch(protocol.GoGlobalStateFromSolidity(creationInfo.BeforeState.GlobalState).Batch)
	toBatch := l2stateprovider.Batch(protocol.GoGlobalStateFromSolidity(creationInfo.AfterState.GlobalState).Batch)

	startCommit, err := m.stateManager.HistoryCommitment(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot:              prevCreationInfo.WasmModuleRoot,
			FromBatch:                   fromBatch,
			ToBatch:                     toBatch,
			UpperChallengeOriginHeights: []l2stateprovider.Height{},
			FromHeight:                  0,
			UpToHeight:                  option.Some(l2stateprovider.Height(0)),
		},
	)
	if err != nil {
		return nil, nil, err
	}
	manager, err := m.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, nil, err
	}
	layerZeroHeights, err := manager.LayerZeroHeights(ctx)
	if err != nil {
		return nil, nil, err
	}
	req := &l2stateprovider.HistoryCommitmentRequest{
		WasmModuleRoot:              prevCreationInfo.WasmModuleRoot,
		FromBatch:                   fromBatch,
		ToBatch:                     toBatch,
		UpperChallengeOriginHeights: []l2stateprovider.Height{},
		FromHeight:                  0,
		UpToHeight:                  option.Some(l2stateprovider.Height(layerZeroHeights.BlockChallengeHeight)),
	}
	endCommit, err := m.stateManager.HistoryCommitment(
		ctx,
		req,
	)
	if err != nil {
		return nil, nil, err
	}
	startEndPrefixProof, err := m.stateManager.PrefixProof(
		ctx,
		req,
		l2stateprovider.Height(0),
	)
	if err != nil {
		return nil, nil, err
	}
	edge, err := manager.AddBlockChallengeLevelZeroEdge(ctx, assertion, startCommit, endCommit, startEndPrefixProof)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not post block challenge root edge")
	}
	return edge, &edgetracker.AssociatedAssertionMetadata{
		FromBatch:      fromBatch,
		ToBatch:        toBatch,
		WasmModuleRoot: prevCreationInfo.WasmModuleRoot,
	}, nil
}
