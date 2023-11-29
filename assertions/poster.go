// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package assertions

import (
	"context"
	"fmt"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

func (m *Manager) postAssertionRoutine(ctx context.Context) {
	if m.challengeReader.Mode() < types.MakeMode {
		srvlog.Warn("Staker strategy not configured to stake on latest assertions")
		return
	}
	if _, err := m.PostAssertion(ctx); err != nil {
		if !errors.Is(err, solimpl.ErrAlreadyExists) {
			srvlog.Error("Could not submit latest assertion to L1", log.Ctx{"err": err})
		}
	}
	ticker := time.NewTicker(m.postInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := m.PostAssertion(ctx); err != nil {
				if !errors.Is(err, solimpl.ErrAlreadyExists) {
					srvlog.Error("Could not submit latest assertion to L1", log.Ctx{"err": err})
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// PostAssertion differs depending on whether or not the validator is currently staked.
func (m *Manager) PostAssertion(ctx context.Context) (protocol.Assertion, error) {
	// Ensure that we only build on a valid parent from this validator's perspective.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	parentAssertionSeq, err := m.findLatestValidAssertion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not find latest valid assertion")
	}
	parentAssertionCreationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, parentAssertionSeq)
	if err != nil {
		return nil, err
	}
	staked, err := m.chain.IsStaked(ctx)
	if err != nil {
		return nil, err
	}
	// If the validator is already staked, we post an assertion and move existing stake to it.
	if staked {
		assertion, postErr := m.PostAssertionBasedOnParent(
			ctx, parentAssertionCreationInfo, m.chain.StakeOnNewAssertion,
		)
		if postErr != nil {
			return nil, postErr
		}
		m.submittedAssertions.Insert(assertion.Id().Hash)
		return assertion, nil
	}
	// Otherwise, we post a new assertion and place a new stake on it.
	assertion, err := m.PostAssertionBasedOnParent(
		ctx, parentAssertionCreationInfo, m.chain.NewStakeOnNewAssertion,
	)
	if err != nil {
		return nil, err
	}
	m.submittedAssertions.Insert(assertion.Id().Hash)
	return assertion, nil
}

// Posts a new assertion onchain based on a parent assertion we agree with.
func (m *Manager) PostAssertionBasedOnParent(
	ctx context.Context,
	parentCreationInfo *protocol.AssertionCreatedInfo,
	submitFn func(
		ctx context.Context,
		parentCreationInfo *protocol.AssertionCreatedInfo,
		newState *protocol.ExecutionState,
	) (protocol.Assertion, error),
) (protocol.Assertion, error) {
	if !parentCreationInfo.InboxMaxCount.IsUint64() {
		return nil, errors.New("inbox max count not a uint64")
	}
	// The parent assertion tells us what the next posted assertion's batch should be.
	// We read this value and use it to compute the required execution state we must post.
	batchCount := parentCreationInfo.InboxMaxCount.Uint64()
	newState, err := m.stateManager.ExecutionStateAfterBatchCount(ctx, batchCount)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get execution state at message count %d", batchCount)
	}
	srvlog.Info(
		"Posting assertion with retrieved state", log.Ctx{
			"batchCount": batchCount,
			"newState":   fmt.Sprintf("%+v", newState),
		},
	)
	assertion, err := submitFn(
		ctx,
		parentCreationInfo,
		newState,
	)
	switch {
	case errors.Is(err, solimpl.ErrAlreadyExists):
		return nil, errors.Wrap(err, "assertion already exists, was unable to post")
	case err != nil:
		return nil, err
	}
	srvlog.Info("Submitted latest L2 state claim as an assertion to L1", log.Ctx{
		"validatorName":         m.validatorName,
		"layer2BlockHash":       containers.Trunc(newState.GlobalState.BlockHash[:]),
		"requiredInboxMaxCount": batchCount,
		"postedExecutionState":  fmt.Sprintf("%+v", newState),
	})

	return assertion, nil
}

// Finds the latest valid assertion sequence num a validator should build their new leaves upon.
// It retrieves the latest assertion hashes posted to the rollup contract since the last confirmed assertion block.
// This walks down the list of assertions in the protocol down until it finds
// the latest assertion that we have a state commitment for.
func (m *Manager) findLatestValidAssertion(ctx context.Context) (protocol.AssertionHash, error) {
	latestCreatedAssertionHashes, err := m.chain.LatestCreatedAssertionHashes(ctx)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	// Loop over latestCreatedAssertionHashes in reverse order to find the latest valid assertion.
	for i := len(latestCreatedAssertionHashes) - 1; i >= 0; i-- {
		var info *protocol.AssertionCreatedInfo
		info, err = m.chain.ReadAssertionCreationInfo(ctx, latestCreatedAssertionHashes[i])
		if err != nil {
			return protocol.AssertionHash{}, err
		}
		if err = m.stateManager.AgreesWithExecutionState(ctx, protocol.GoExecutionStateFromSolidity(info.AfterState)); err == nil {
			return latestCreatedAssertionHashes[i], nil
		}
	}
	latestConfirmed, err := m.chain.LatestConfirmed(ctx)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	return latestConfirmed.Id(), nil
}
