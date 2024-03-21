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
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pkg/errors"
)

var (
	assertionPostedCounter       = metrics.NewRegisteredCounter("arb/validator/poster/assertion_posted", nil)
	errorPostingAssertionCounter = metrics.NewRegisteredCounter("arb/validator/poster/error_posting_assertion", nil)
	chainCatchingUpCounter       = metrics.NewRegisteredCounter("arb/validator/poster/chain_catching_up", nil)
)

func (m *Manager) postAssertionRoutine(ctx context.Context) {
	if m.challengeReader.Mode() < types.MakeMode {
		srvlog.Warn("Staker strategy not configured to stake on latest assertions")
		return
	}
	if _, err := m.PostAssertion(ctx); err != nil {
		if !errors.Is(err, solimpl.ErrAlreadyExists) {
			srvlog.Error("Could not submit latest assertion to L1", log.Ctx{"err": err})
			errorPostingAssertionCounter.Inc(1)
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
					errorPostingAssertionCounter.Inc(1)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) awaitPostingSignal(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.startPostingSignal:
			m.isReadyToPost = true
			return
		}
	}
}

// PostAssertion differs depending on whether or not the validator is currently staked.
func (m *Manager) PostAssertion(ctx context.Context) (option.Option[*protocol.AssertionCreatedInfo], error) {
	if !m.isReadyToPost {
		m.awaitPostingSignal(ctx)
	}
	// Ensure that we only build on a valid parent from this validator's perspective.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	m.assertionChainData.Lock()
	parentAssertionCreationInfo, ok := m.assertionChainData.canonicalAssertions[m.assertionChainData.latestAgreedAssertion]
	m.assertionChainData.Unlock()
	none := option.None[*protocol.AssertionCreatedInfo]()
	if !ok {
		return none, fmt.Errorf(
			"latest agreed assertion %#x not part of canonical mapping, something is wrong",
			m.assertionChainData.latestAgreedAssertion.Hash,
		)
	}
	staked, err := m.chain.IsStaked(ctx)
	if err != nil {
		return none, err
	}
	// If the validator is already staked, we post an assertion and move existing stake to it.
	var assertionOpt option.Option[*protocol.AssertionCreatedInfo]
	var postErr error
	if staked {
		assertionOpt, postErr = m.PostAssertionBasedOnParent(
			ctx, parentAssertionCreationInfo, m.chain.StakeOnNewAssertion,
		)
	} else {
		// Otherwise, we post a new assertion and place a new stake on it.
		assertionOpt, postErr = m.PostAssertionBasedOnParent(
			ctx, parentAssertionCreationInfo, m.chain.NewStakeOnNewAssertion,
		)
	}
	if postErr != nil {
		return none, postErr
	}
	if assertionOpt.IsSome() {
		m.submittedAssertions.Insert(assertionOpt.Unwrap().AssertionHash)
	}
	return assertionOpt, nil
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
) (option.Option[*protocol.AssertionCreatedInfo], error) {
	none := option.None[*protocol.AssertionCreatedInfo]()
	if !parentCreationInfo.InboxMaxCount.IsUint64() {
		return none, errors.New("inbox max count not a uint64")
	}
	// The parent assertion tells us what the next posted assertion's batch should be.
	// We read this value and use it to compute the required execution state we must post.
	batchCount := parentCreationInfo.InboxMaxCount.Uint64()
	newState, err := m.stateManager.ExecutionStateAfterBatchCount(ctx, batchCount)
	if err != nil {
		if errors.Is(err, l2stateprovider.ErrChainCatchingUp) {
			chainCatchingUpCounter.Inc(1)
			srvlog.Info(
				"No available batch to post as assertion, waiting for more batches", log.Ctx{
					"batchCount": batchCount,
				},
			)
			return none, nil
		}
		return none, errors.Wrapf(err, "could not get execution state at batch count %d", batchCount)
	}
	srvlog.Info(
		"Posting assertion with retrieved state", log.Ctx{
			"batchCount":    batchCount,
			"validatorName": m.validatorName,
		},
	)
	assertion, err := submitFn(
		ctx,
		parentCreationInfo,
		newState,
	)
	if err != nil {
		return none, err
	}
	srvlog.Info("Submitted latest L2 state claim as an assertion to L1", log.Ctx{
		"validatorName":         m.validatorName,
		"requiredInboxMaxCount": batchCount,
		"postedExecutionState":  fmt.Sprintf("%+v", newState),
	})
	assertionPostedCounter.Inc(1)
	creationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, assertion.Id())
	if err != nil {
		return none, err
	}
	m.observedCanonicalAssertions <- assertion.Id()
	return option.Some(creationInfo), nil
}
