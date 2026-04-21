// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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

	"github.com/offchainlabs/nitro/bold/containers"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/protocol/sol"
	"github.com/offchainlabs/nitro/bold/state"
	"github.com/offchainlabs/nitro/util"
	util_containers "github.com/offchainlabs/nitro/util/containers"
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

	exceedsMaxMempoolSizeEphemeralErrorHandler := util.NewEphemeralErrorHandler(10*time.Minute, "posting this transaction will exceed max mempool size", 0)
	gasEstimationEphemeralErrorHandler := util.NewEphemeralErrorHandler(10*time.Minute, "gas estimation errored for tx with hash", 0)

	log.Info("Ready to post")
	ticker := time.NewTicker(m.times.postInterval)
	defer ticker.Stop()
	for {
		_, err := m.PostAssertion(ctx)
		if err != nil {
			switch {
			case errors.Is(err, sol.ErrAlreadyExists):
			case errors.Is(err, sol.ErrBatchNotYetFound):
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

// advanceChainPointer reads the creation info for the given assertion and
// updates the local chain tracking state so subsequent posting attempts
// build on top of it.
func (m *Manager) advanceChainPointer(ctx context.Context, assertionId protocol.AssertionHash) error {
	creationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, assertionId)
	if err != nil {
		return fmt.Errorf("could not read creation info for assertion %#x: %w", assertionId.Hash, err)
	}
	m.assertionChainData.Lock()
	m.assertionChainData.latestAgreedAssertion = assertionId
	m.assertionChainData.canonicalAssertions[assertionId] = creationInfo
	m.assertionChainData.Unlock()
	m.submittedAssertions.Insert(assertionId)
	return nil
}

// PostAssertion differs depending on whether or not the validator is currently staked.
// It advances through any assertions that already exist onchain before attempting
// to post a genuinely new one, ensuring the chain tracking stays up to date.
func (m *Manager) PostAssertion(ctx context.Context) (util_containers.Option[protocol.Assertion], error) {
	if !m.isReadyToPost {
		m.awaitPostingSignal(ctx)
	}
	none := util_containers.None[protocol.Assertion]()

	staked, err := m.chain.IsStaked(ctx)
	if err != nil {
		return none, err
	}

	for {
		if ctx.Err() != nil {
			return none, ctx.Err()
		}
		// Ensure that we only build on a valid parent from this validator's perspective.
		m.assertionChainData.RLock()
		parentAssertionCreationInfo, ok := m.assertionChainData.canonicalAssertions[m.assertionChainData.latestAgreedAssertion]
		m.assertionChainData.RUnlock()
		if !ok {
			return none, fmt.Errorf(
				"latest agreed assertion %#x not part of canonical mapping, something is wrong",
				m.assertionChainData.latestAgreedAssertion.Hash,
			)
		}

		// If the validator is already staked, we post an assertion and move existing stake to it.
		var assertionOpt util_containers.Option[protocol.Assertion]
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
			if errors.Is(postErr, sol.ErrAlreadyExists) && assertionOpt.IsSome() {
				// The assertion we tried to post already exists onchain.
				// Advance our local chain pointer and loop to try the next assertion.
				existingId := assertionOpt.Unwrap().Id()
				if err := m.advanceChainPointer(ctx, existingId); err != nil {
					return none, err
				}
				m.sendToConfirmationQueue(existingId, "PostAssertion-catchup")
				log.Info("Assertion already exists onchain, advancing chain tracking",
					"assertionHash", existingId,
					"validatorName", m.validatorName,
				)
				continue
			}
			return none, postErr
		}

		// Successfully posted a new assertion. Advance our local chain pointer
		// so the next posting attempt uses this assertion as the parent.
		// Note: sendToConfirmationQueue is already called inside PostAssertionBasedOnParent
		// for newly posted assertions, so we don't call it again here.
		if assertionOpt.IsSome() {
			if err := m.advanceChainPointer(ctx, assertionOpt.Unwrap().Id()); err != nil {
				return none, err
			}
		}
		return assertionOpt, nil
	}
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
) (util_containers.Option[protocol.Assertion], error) {
	none := util_containers.None[protocol.Assertion]()
	if !parentCreationInfo.InboxMaxCount.IsUint64() {
		return none, errors.New("inbox max count not a uint64")
	}
	// The parent assertion tells us what the next posted assertion's batch should be.
	// We read this value and use it to compute the required execution state we must post.
	batchCount := parentCreationInfo.InboxMaxCount.Uint64()
	parentBlockHash := protocol.GoGlobalStateFromSolidity(parentCreationInfo.AfterState.GlobalState).BlockHash
	newState, err := m.ExecutionStateAfterParent(ctx, parentCreationInfo)
	if err != nil {
		if errors.Is(err, state.ErrChainCatchingUp) {
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
		if errors.Is(err, sol.ErrAlreadyExists) {
			// The assertion already exists on-chain. Return it with the error
			// so the caller can advance the chain pointer.
			return util_containers.Some(assertion), err
		}
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
	return util_containers.Some(assertion), nil
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
