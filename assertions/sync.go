package assertions

import (
	"context"
	"fmt"
	"time"

	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

func (m *Manager) syncAssertions(ctx context.Context) {
	latestConfirmed, err := retry.UntilSucceeds(ctx, func() (protocol.Assertion, error) {
		return m.chain.LatestConfirmed(ctx)
	})
	if err != nil {
		srvlog.Error("Could not get latest confirmed assertion", log.Ctx{"err": err})
		return
	}
	latestConfirmedInfo, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
		return m.chain.ReadAssertionCreationInfo(ctx, latestConfirmed.Id())
	})
	if err != nil {
		srvlog.Error("Could not get latest confirmed assertion", log.Ctx{"err": err})
		return
	}

	m.assertionChainData.Lock()
	m.assertionChainData.latestAgreedAssertion = latestConfirmed.Id()
	m.assertionChainData.canonicalAssertions[latestConfirmed.Id()] = latestConfirmedInfo
	if !m.disablePosting {
		m.startPostingSignal <- struct{}{}
		close(m.startPostingSignal)
	}
	m.assertionChainData.Unlock()

	fromBlock := latestConfirmed.CreatedAtBlock()

	filterer, err := retry.UntilSucceeds(ctx, func() (*rollupgen.RollupUserLogicFilterer, error) {
		return rollupgen.NewRollupUserLogicFilterer(m.rollupAddr, m.backend)
	})
	if err != nil {
		srvlog.Error("Could not get rollup user logic filterer", log.Ctx{"err": err})
		return
	}
	latestBlock, err := retry.UntilSucceeds(ctx, func() (*gethtypes.Header, error) {
		return m.backend.HeaderByNumber(ctx, util.GetSafeBlockNumber())
	})
	if err != nil {
		srvlog.Error("Could not get header by number", log.Ctx{"err": err})
		return
	}
	if !latestBlock.Number.IsUint64() {
		srvlog.Error("Latest block number was not a uint64")
		return
	}
	toBlock := latestBlock.Number.Uint64()
	if fromBlock != toBlock {
		filterOpts := &bind.FilterOpts{
			Start:   fromBlock,
			End:     &toBlock,
			Context: ctx,
		}
		_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
			innerErr := m.processAllAssertionsInRange(ctx, filterer, filterOpts)
			if innerErr != nil {
				srvlog.Error("Could not process assertions in range", log.Ctx{"err": innerErr})
				return false, innerErr
			}
			return true, nil
		})
		if err != nil {
			srvlog.Error("Could not check for assertion added event")
			return
		}
		fromBlock = toBlock
	}

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := m.backend.HeaderByNumber(ctx, util.GetSafeBlockNumber())
			if err != nil {
				srvlog.Error("Could not get header by number", log.Ctx{"err": err})
				continue
			}
			if !latestBlock.Number.IsUint64() {
				srvlog.Error("Latest block number was not a uint64")
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
				innerErr := m.processAllAssertionsInRange(ctx, filterer, filterOpts)
				if innerErr != nil {
					srvlog.Error("Could not process assertions in range", log.Ctx{"err": innerErr})
					return false, innerErr
				}
				return true, nil
			})
			if err != nil {
				srvlog.Error("Could not check for assertion added", log.Ctx{"err": err})
				return
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

// This function will scan for all assertion creation events to determine which
// ones are canonical and which ones must be challenged.
func (m *Manager) processAllAssertionsInRange(
	ctx context.Context,
	filterer *rollupgen.RollupUserLogicFilterer,
	filterOpts *bind.FilterOpts,
) error {
	it, err := filterer.FilterAssertionCreated(filterOpts, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err = it.Close(); err != nil {
			srvlog.Error("Could not close filter iterator", log.Ctx{"err": err})
		}
	}()

	// Extract all assertion creation events from the log filter iterator.
	assertions := make([]*protocol.AssertionCreatedInfo, 0)
	for it.Next() {
		if it.Error() != nil {
			return errors.Wrapf(
				err,
				"got iterator error when scanning assertion creations from block %d to %d",
				filterOpts.Start,
				*filterOpts.End,
			)
		}
		assertionOpt, err := retry.UntilSucceeds(ctx, func() (option.Option[*protocol.AssertionCreatedInfo], error) {
			item, innerErr := m.extractAssertionFromEvent(ctx, it.Event)
			if innerErr != nil {
				srvlog.Error("Could not extract assertion from event", log.Ctx{"err": innerErr})
				return option.None[*protocol.AssertionCreatedInfo](), innerErr
			}
			return item, nil
		})
		if err != nil {
			return err
		}
		if assertionOpt.IsSome() {
			assertions = append(assertions, assertionOpt.Unwrap())
		}
	}

	// Save all observed assertions to the database.
	for _, assertion := range assertions {
		if _, err := retry.UntilSucceeds(ctx, func() (bool, error) {
			if err := m.saveAssertionToDB(ctx, assertion); err != nil {
				srvlog.Error("Could not save assertion to DB", log.Ctx{"err": err})
				return false, err
			}
			return true, nil
		}); err != nil {
			return err
		}
	}

	m.assertionChainData.Lock()
	defer m.assertionChainData.Unlock()

	// Determine the canonical branch of all assertions.
	if _, err := retry.UntilSucceeds(ctx, func() (bool, error) {
		if innerErr := m.findCanonicalAssertionBranch(ctx, assertions); innerErr != nil {
			srvlog.Error("Could not find canonical assertion branch", log.Ctx{"err": innerErr})
			return false, innerErr
		}
		return true, nil
	}); err != nil {
		return err
	}

	// Now that we derived the canonical chain, we perform a pass over all assertions
	// to figure out which ones are invalid and therefore should be challenged.
	if _, err := retry.UntilSucceeds(ctx, func() (bool, error) {
		if innerErr := m.respondToAnyInvalidAssertions(ctx, assertions, m); innerErr != nil {
			srvlog.Error("Could not find canonical assertion branch", log.Ctx{"err": innerErr})
			return false, innerErr
		}
		return true, nil
	}); err != nil {
		return err
	}
	return nil
}

// Extracts a valid assertion creation from an event log. Returns none
// if the assertion is genesis or if the hash is the zero hash.
func (m *Manager) extractAssertionFromEvent(
	ctx context.Context,
	event *rollupgen.RollupUserLogicAssertionCreated,
) (option.Option[*protocol.AssertionCreatedInfo], error) {
	none := option.None[*protocol.AssertionCreatedInfo]()
	if event.AssertionHash == (common.Hash{}) {
		srvlog.Warn("Encountered an assertion with a zero hash", log.Ctx{
			"creationEvent": fmt.Sprintf("%+v", event),
		})
		return none, nil
	}
	assertionHash := protocol.AssertionHash{Hash: event.AssertionHash}
	creationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return none, errors.Wrapf(err, "could not read assertion creation info for %#x", assertionHash.Hash)
	}
	if creationInfo.ParentAssertionHash == (common.Hash{}) {
		return none, nil
	}
	return option.Some(creationInfo), nil
}

// Finds all canonical assertions from an ordered list by creation time.
// Starts by setting a cursor to the latest confirmed assertion, then finds all assertions parent == cursor.
// We then check which one we agree with.
// From there, checks all assertions that have that assertion as parent, etc.
// This function must hold the lock on m.assertionChainData.
func (m *Manager) findCanonicalAssertionBranch(
	ctx context.Context,
	assertions []*protocol.AssertionCreatedInfo,
) error {
	latestAgreedWithAssertion := m.assertionChainData.latestAgreedAssertion
	cursor := latestAgreedWithAssertion

	for _, assertion := range assertions {
		if assertion.ParentAssertionHash == cursor.Hash {
			agreedWithAssertion, err := retry.UntilSucceeds(ctx, func() (bool, error) {
				state := protocol.GoExecutionStateFromSolidity(assertion.AfterState)
				err := m.stateProvider.AgreesWithExecutionState(ctx, state)
				switch {
				case errors.Is(err, l2stateprovider.ErrNoExecutionState):
					return false, nil
				case errors.Is(err, l2stateprovider.ErrChainCatchingUp):
					// Otherwise, we return the error that we are still catching up to the
					// execution state claimed by the assertion, and this function will be retried
					// by the caller if wrapped in a retryable call.
					chainCatchingUpCounter.Inc(1)
					srvlog.Info("Chain still catching up to processed execution state "+
						"will reattempt processing when caught up", log.Ctx{"err": err})
					return false, l2stateprovider.ErrChainCatchingUp
				case err != nil:
					return false, err
				}
				return true, nil
			})
			if err != nil {
				return errors.New("could not check for assertion agreements")
			}
			if agreedWithAssertion {
				cursor = protocol.AssertionHash{Hash: assertion.AssertionHash}
				m.assertionChainData.latestAgreedAssertion = cursor
				m.assertionChainData.canonicalAssertions[cursor] = assertion
				m.observedCanonicalAssertions <- cursor
			}
		}
	}
	return nil
}

type rivalPosterArgs struct {
	canonicalParent  *protocol.AssertionCreatedInfo
	invalidAssertion *protocol.AssertionCreatedInfo
}

type rivalPoster interface {
	maybePostRivalAssertionAndChallenge(
		ctx context.Context,
		args rivalPosterArgs,
	) (*protocol.AssertionCreatedInfo, error)
}

// Finds all canonical assertions from a list. Starts by setting a cursor to the
// latest confirmed assertion, then finds all assertions parent == cursor.
// We then check which one we agree with.
// From there, checks all assertions that have that assertion as parent, etc.
// This function must hold the lock on m.assertionChainData.
func (m *Manager) respondToAnyInvalidAssertions(
	ctx context.Context,
	assertions []*protocol.AssertionCreatedInfo,
	rivalPoster rivalPoster,
) error {
	for _, assertion := range assertions {
		canonicalParent, hasCanonicalParent := m.assertionChainData.canonicalAssertions[protocol.AssertionHash{
			Hash: assertion.ParentAssertionHash,
		}]
		_, isCanonical := m.assertionChainData.canonicalAssertions[protocol.AssertionHash{
			Hash: assertion.AssertionHash,
		}]
		// If an assertion has a canonical parent but is not canonical itself,
		// then we should challenge the assertion if we are configured to do so,
		// or raise an alarm if we are only a watchtower validator.
		if hasCanonicalParent && !isCanonical {
			postedRival, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
				posted, innerErr := rivalPoster.maybePostRivalAssertionAndChallenge(ctx, rivalPosterArgs{
					canonicalParent:  canonicalParent,
					invalidAssertion: assertion,
				})
				if innerErr != nil {
					srvlog.Error("Could not post rival assertion and/or challenge", log.Ctx{"err": innerErr})
					return nil, innerErr
				}
				return posted, nil
			})
			if err != nil {
				return err
			}
			if postedRival != nil {
				postedAssertionHash := protocol.AssertionHash{Hash: postedRival.AssertionHash}
				if _, ok := m.assertionChainData.canonicalAssertions[postedAssertionHash]; !ok {
					m.assertionChainData.canonicalAssertions[postedAssertionHash] = postedRival
					m.submittedAssertions.Insert(postedAssertionHash.Hash)
					m.submittedRivalsCount++
					m.observedCanonicalAssertions <- postedAssertionHash
				}
			}
		}
	}
	return nil
}

// Attempts to post a rival assertion to a given assertion and then attempts to
// open a challenge on that fork in the chain if configured to do so.
func (m *Manager) maybePostRivalAssertionAndChallenge(
	ctx context.Context,
	args rivalPosterArgs,
) (*protocol.AssertionCreatedInfo, error) {
	if !args.invalidAssertion.InboxMaxCount.IsUint64() {
		return nil, errors.New("inbox max count not a uint64")
	}
	if args.canonicalParent.AssertionHash != args.invalidAssertion.ParentAssertionHash {
		return nil, errors.New("invalid assertion does not have correct canonical parent")
	}
	batchCount := args.invalidAssertion.InboxMaxCount.Uint64()
	logFields := log.Ctx{
		"validatorName":         m.validatorName,
		"canonicalParentHash":   args.invalidAssertion.ParentAssertionHash,
		"detectedAssertionHash": args.invalidAssertion.AssertionHash,
		"batchCount":            batchCount,
	}
	if !m.canPostRivalAssertion() {
		srvlog.Warn("Detected invalid assertion, but not configured to post a rival stake", logFields)
		evilAssertionCounter.Inc(1)
		return nil, nil
	}

	srvlog.Info("Disagreed with execution state from observed assertion", logFields)
	evilAssertionCounter.Inc(1)

	// Post what we believe is the correct rival assertion that follows the ancestor we agree with.
	correctRivalAssertion, err := m.maybePostRivalAssertion(ctx, args.canonicalParent)
	if err != nil {
		return nil, err
	}
	if correctRivalAssertion.IsNone() {
		srvlog.Warn(fmt.Sprintf("Expected to post a rival assertion to %#x, but did not post anything", args.invalidAssertion.AssertionHash))
		return nil, nil
	}
	if !m.canPostChallenge() {
		srvlog.Warn("Attempted to post rival assertion and stake, but not configured to initiate a challenge", logFields)
		return nil, nil
	}
	assertionHash := protocol.AssertionHash{Hash: correctRivalAssertion.Unwrap().AssertionHash}
	postedRival, err := m.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read assertion creation info for %#x", assertionHash.Hash)
	}

	if args.canonicalParent.ChallengeManager != m.challengeManagerAddr {
		srvlog.Warn("Posted rival assertion, but could not challenge as challenge manager address did not match, "+
			"start a new server with the right challenge manager address", log.Ctx{
			"correctAssertion":                  postedRival.AssertionHash,
			"evilAssertion":                     args.invalidAssertion.AssertionHash,
			"expectedChallengeManagerAddress":   args.canonicalParent.ChallengeManager,
			"configuredChallengeManagerAddress": m.challengeManagerAddr,
		})
		return nil, nil
	}

	// Generating a random integer between 0 and max delay second to wait before challenging.
	// This is to avoid all validators challenging at the same time.
	mds := 1 // default max delay seconds to 1 to avoid panic
	if m.challengeReader.MaxDelaySeconds() > 1 {
		mds = m.challengeReader.MaxDelaySeconds()
	}
	randSecs, err := randUint64(uint64(mds))
	if err != nil {
		return nil, err
	}
	srvlog.Info("Waiting before submitting challenge on assertion", log.Ctx{"delay": randSecs})
	time.Sleep(time.Duration(randSecs) * time.Second)
	correctClaimedAssertionHash := protocol.AssertionHash{
		Hash: correctRivalAssertion.Unwrap().AssertionHash,
	}
	challengeSubmitted, err := m.challengeCreator.ChallengeAssertion(ctx, correctClaimedAssertionHash)
	if err != nil {
		return nil, err
	}
	if challengeSubmitted {
		challengeSubmittedCounter.Inc(1)
		m.challengesSubmittedCount++
	}

	if err := m.logChallengeConfigs(ctx); err != nil {
		srvlog.Error("Could not log challenge configs", log.Ctx{"err": err})
	}
	return postedRival, nil
}

// Attempt to post a rival assertion based on the last agreed with ancestor
// of a given assertion.
//
// If this parent assertion already has a rival we agree with that arleady exists
// then this function will return that assertion.
func (m *Manager) maybePostRivalAssertion(
	ctx context.Context,
	canonicalParent *protocol.AssertionCreatedInfo,
) (option.Option[*protocol.AssertionCreatedInfo], error) {
	none := option.None[*protocol.AssertionCreatedInfo]()
	// Post what we believe is the correct assertion that follows the ancestor we agree with.
	staked, err := m.chain.IsStaked(ctx)
	if err != nil {
		return none, err
	}
	// If the validator is already staked, we post an assertion and move existing stake to it.
	var assertionOpt option.Option[*protocol.AssertionCreatedInfo]
	var postErr error
	if staked {
		assertionOpt, postErr = m.PostAssertionBasedOnParent(
			ctx, canonicalParent, m.chain.StakeOnNewAssertion,
		)
	} else {
		// Otherwise, we post a new assertion and place a new stake on it.
		assertionOpt, postErr = m.PostAssertionBasedOnParent(
			ctx, canonicalParent, m.chain.NewStakeOnNewAssertion,
		)
	}
	if postErr != nil {
		return none, postErr
	}
	if assertionOpt.IsSome() {
		if err2 := m.saveAssertionToDB(ctx, assertionOpt.Unwrap()); err2 != nil {
			return none, err2
		}
	}
	return assertionOpt, nil
}

func (m *Manager) saveAssertionToDB(ctx context.Context, creationInfo *protocol.AssertionCreatedInfo) error {
	if api.IsNil(m.apiDB) {
		return nil
	}
	beforeState := protocol.GoExecutionStateFromSolidity(creationInfo.BeforeState)
	afterState := protocol.GoExecutionStateFromSolidity(creationInfo.AfterState)
	assertionHash := protocol.AssertionHash{Hash: creationInfo.AssertionHash}
	status, err := m.chain.AssertionStatus(ctx, assertionHash)
	if err != nil {
		return err
	}
	assertion, err := m.chain.GetAssertion(ctx, assertionHash)
	if err != nil {
		return err
	}
	isFirstChild, err := assertion.IsFirstChild()
	if err != nil {
		return err
	}
	firstChildBlock, err := assertion.SecondChildCreationBlock()
	if err != nil {
		return err
	}
	secondChildBlock, err := assertion.SecondChildCreationBlock()
	if err != nil {
		return err
	}
	return m.apiDB.InsertAssertion(&api.JsonAssertion{
		Hash:                     assertionHash.Hash,
		ConfirmPeriodBlocks:      creationInfo.ConfirmPeriodBlocks,
		RequiredStake:            creationInfo.RequiredStake.String(),
		ParentAssertionHash:      creationInfo.ParentAssertionHash,
		InboxMaxCount:            creationInfo.InboxMaxCount.String(),
		AfterInboxBatchAcc:       creationInfo.AfterInboxBatchAcc,
		WasmModuleRoot:           creationInfo.WasmModuleRoot,
		ChallengeManager:         creationInfo.ChallengeManager,
		CreationBlock:            creationInfo.CreationBlock,
		TransactionHash:          creationInfo.TransactionHash,
		BeforeStateBlockHash:     beforeState.GlobalState.BlockHash,
		BeforeStateSendRoot:      beforeState.GlobalState.SendRoot,
		BeforeStateBatch:         beforeState.GlobalState.Batch,
		BeforeStatePosInBatch:    beforeState.GlobalState.PosInBatch,
		BeforeStateMachineStatus: beforeState.MachineStatus,
		AfterStateBlockHash:      afterState.GlobalState.BlockHash,
		AfterStateSendRoot:       afterState.GlobalState.SendRoot,
		AfterStateBatch:          afterState.GlobalState.Batch,
		AfterStatePosInBatch:     afterState.GlobalState.PosInBatch,
		AfterStateMachineStatus:  afterState.MachineStatus,
		FirstChildBlock:          &firstChildBlock,
		SecondChildBlock:         &secondChildBlock,
		IsFirstChild:             isFirstChild,
		Status:                   status.String(),
	})
}
