// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package assertions

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
	"github.com/offchainlabs/nitro/bold/containers"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/logs/ephemeral"
)

var (
	assertionPostedCounter       = metrics.NewRegisteredCounter("arb/validator/poster/assertion_posted", nil)
	errorPostingAssertionCounter = metrics.NewRegisteredCounter("arb/validator/poster/error_posting_assertion", nil)
	chainCatchingUpCounter       = metrics.NewRegisteredCounter("arb/validator/poster/chain_catching_up", nil)
)

func (m *Manager) postAssertionRoutine(ctx context.Context) {
	if !m.mode.SupportsStaking() {
		log.Warn("Staker strategy not configured to stake on latest assertions")
		return
	}

	exceedsMaxMempoolSizeEphemeralErrorHandler := ephemeral.NewEphemeralErrorHandler(10*time.Minute, "posting this transaction will exceed max mempool size", 0)
	gasEstimationEphemeralErrorHandler := ephemeral.NewEphemeralErrorHandler(10*time.Minute, "gas estimation errored for tx with hash", 0)

	log.Info("Ready to post")
	ticker := time.NewTicker(m.times.postInterval)
	defer ticker.Stop()
	for {
		_, err := m.PostAssertion(ctx)
		if err != nil {
			switch {
			case errors.Is(err, solimpl.ErrAlreadyExists):
			case errors.Is(err, solimpl.ErrBatchNotYetFound):
				log.Info("Waiting for more batches to post assertions about them onchain")
			default:
				logLevel := log.Error
				logLevel = exceedsMaxMempoolSizeEphemeralErrorHandler.LogLevel(err, logLevel)
				logLevel = gasEstimationEphemeralErrorHandler.LogLevel(err, logLevel)

				logLevel("Could not submit latest assertion", "err", err, "validatorName", m.validatorName)
				errorPostingAssertionCounter.Inc(1)

				if ctx.Err() != nil {
					return
				}
				continue // We retry again in case of a non ctx error
			}
		} else {
			exceedsMaxMempoolSizeEphemeralErrorHandler.Reset()
			gasEstimationEphemeralErrorHandler.Reset()
		}

		select {
		case <-ticker.C:
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
func (m *Manager) PostAssertion(ctx context.Context) (option.Option[protocol.Assertion], error) {
	if !m.isReadyToPost {
		m.awaitPostingSignal(ctx)
	}
	// Ensure that we only build on a valid parent from this validator's perspective.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	m.assertionChainData.Lock()
	parentAssertionCreationInfo, ok := m.assertionChainData.canonicalAssertions[m.assertionChainData.latestAgreedAssertion]
	m.assertionChainData.Unlock()
	none := option.None[protocol.Assertion]()
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
	var assertionOpt option.Option[protocol.Assertion]
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
		m.submittedAssertions.Insert(assertionOpt.Unwrap().Id())
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
) (option.Option[protocol.Assertion], error) {
	none := option.None[protocol.Assertion]()
	if !parentCreationInfo.InboxMaxCount.IsUint64() {
		return none, errors.New("inbox max count not a uint64")
	}
	// The parent assertion tells us what the next posted assertion's batch should be.
	// We read this value and use it to compute the required execution state we must post.
	batchCount := parentCreationInfo.InboxMaxCount.Uint64()
	parentBlockHash := protocol.GoGlobalStateFromSolidity(parentCreationInfo.AfterState.GlobalState).BlockHash
	newState, err := m.ExecutionStateAfterParent(ctx, parentCreationInfo)
	if err != nil {
		if errors.Is(err, l2stateprovider.ErrChainCatchingUp) {
			chainCatchingUpCounter.Inc(1)
			log.Info(
				"Waiting for more batches to post next assertion",
				"latestStakedAssertionBatchCount", batchCount,
				"latestStakedAssertionBlockHash", containers.Trunc(parentBlockHash[:]),
			)
			// If the chain is catching up, we wait for a bit and try again.
			time.Sleep(m.times.avgBlockTime / 10)
			return none, nil
		}
		return none, errors.Wrapf(err, "could not get execution state at batch count %d with parent block hash %v", batchCount, parentBlockHash)
	}

	// If the assertion is not an overflow assertion i.e !(newState.GlobalState.Batch < batchCount) derived from
	// contracts check for overflow assertion => assertion.afterState.globalState.u64Vals[0] < assertion.beforeStateData.configData.nextInboxPosition)
	// then should check if we need to wait for the minimum number of blocks between assertions and a minimum time since parent assertion creation.
	// Overflow ones are not subject to this check onchain.
	isOverflowAssertion := newState.MachineStatus != protocol.MachineStatusErrored && newState.GlobalState.Batch < batchCount
	if !isOverflowAssertion {
		if err = m.waitToPostIfNeeded(ctx, parentCreationInfo); err != nil {
			return none, err
		}
	}

	log.Info(
		"Posting assertion for batch we agree with",
		"requiredInboxMaxCount", batchCount,
		"validatorName", m.validatorName,
	)
	assertion, err := submitFn(
		ctx,
		parentCreationInfo,
		newState,
	)
	if err != nil {
		return none, err
	}
	assertionPostedCounter.Inc(1)
	log.Info("Successfully submitted assertion",
		"validatorName", m.validatorName,
		"requiredInboxMaxCount", batchCount,
		"postedExecutionState", fmt.Sprintf("%+v", newState),
		"assertionHash", assertion.Id(),
	)

	m.sendToConfirmationQueue(assertion.Id(), "PostAssertionBasedOnParent")
	return option.Some(assertion), nil
}

func (m *Manager) waitToPostIfNeeded(
	ctx context.Context,
	parentCreationInfo *protocol.AssertionCreatedInfo,
) error {
	if m.times.minGapToParent != 0 {
		parentCreationBlock, err := m.backend.HeaderByNumber(ctx, new(big.Int).SetUint64(parentCreationInfo.CreationParentBlock))
		if err != nil {
			return fmt.Errorf("error getting parent assertion creation block header: %w", err)
		}
		parentCreationTime, err := safecast.ToInt64(parentCreationBlock.Time)
		if err != nil {
			return fmt.Errorf("error casting parent assertion creation time to int64: %w", err)
		}
		targetTime := time.Unix(parentCreationTime, 0).Add(m.times.minGapToParent)
		time.Sleep(time.Until(targetTime))
	}
	minPeriodBlocks := m.chain.MinAssertionPeriodBlocks()
	for {
		latestL1BlockNumber, err := m.chain.DesiredL1HeaderU64(ctx)
		if err != nil {
			return err
		}
		blocksSinceLast := uint64(0)
		if parentCreationInfo.CreationL1Block < latestL1BlockNumber {
			blocksSinceLast = latestL1BlockNumber - parentCreationInfo.CreationL1Block
		}
		if blocksSinceLast >= minPeriodBlocks {
			return nil
		}
		// If we cannot post just yet, we can wait.
		log.Info(
			fmt.Sprintf("Need to wait %d blocks before posting next assertion. Current block number: %d",
				minPeriodBlocks-blocksSinceLast,
				latestL1BlockNumber,
			),
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.times.avgBlockTime):
		}
	}
}
