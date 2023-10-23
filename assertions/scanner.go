// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

// Package assertions contains testing utilities for posting and scanning for
// assertions on chain, which are useful for simulating the responsibilities
// of Arbitrum Nitro and initiating challenges as needed using our challenge manager.
package assertions

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	srvlog = log.New("service", "assertions")
)

func init() {
	srvlog.SetHandler(log.StreamHandler(os.Stdout, log.LogfmtFormat()))
}

// The Manager struct is responsible for several tasks related to the assertion chain:
// 1. It continuously polls the assertion chain to check for posted, on-chain assertions starting from the latest confirmed assertion up to the newest one.
// 2. As the assertion chain advances, the Manager keeps polling to stay updated.
// 3. Upon observing each new assertion, the Manager evaluates whether it should challenge the assertion or not.
// 4. The Manager frequently posts new assertions to the assertion chain at specific intervals.
// 5. When posting assertions, it relies on the most recent execution state available in its local state manager.
type Manager struct {
	chain                       protocol.AssertionChain
	backend                     bind.ContractBackend
	challengeCreator            types.ChallengeCreator
	challengeReader             types.ChallengeReader
	stateProvider               l2stateprovider.ExecutionStateAgreementChecker
	pollInterval                time.Duration
	confirmationAttemptInterval time.Duration
	rollupAddr                  common.Address
	validatorName               string
	forksDetectedCount          uint64
	challengesSubmittedCount    uint64
	assertionsProcessedCount    uint64
	stateManager                l2stateprovider.ExecutionProvider
	postInterval                time.Duration
	submittedAssertions         *threadsafe.Set[common.Hash]
}

// NewManager creates a manager from the required dependencies.
func NewManager(
	chain protocol.AssertionChain,
	stateProvider l2stateprovider.Provider,
	backend bind.ContractBackend,
	challengeManager types.ChallengeManager,
	rollupAddr common.Address,
	validatorName string,
	pollInterval,
	assertionConfirmationAttemptInterval time.Duration,
	stateManager l2stateprovider.ExecutionProvider,
	postInterval time.Duration,
) (*Manager, error) {
	if pollInterval == 0 {
		return nil, errors.New("assertion scanning interval must be greater than 0")
	}
	if assertionConfirmationAttemptInterval == 0 {
		return nil, errors.New("assertion confirmation attempt interval must be greater than 0")
	}
	return &Manager{
		chain:                       chain,
		backend:                     backend,
		stateProvider:               stateProvider,
		challengeCreator:            challengeManager,
		challengeReader:             challengeManager,
		rollupAddr:                  rollupAddr,
		validatorName:               validatorName,
		pollInterval:                pollInterval,
		confirmationAttemptInterval: assertionConfirmationAttemptInterval,
		forksDetectedCount:          0,
		challengesSubmittedCount:    0,
		assertionsProcessedCount:    0,
		stateManager:                stateManager,
		postInterval:                postInterval,
		submittedAssertions:         threadsafe.NewSet[common.Hash](),
	}, nil
}

// The Start function begins two main tasks:
// 1. It initiates scanning of the assertion chain for newly created assertions, starting from the latest confirmed assertion. This scanning is done via polling.
// 2. Concurrently, it also starts a routine that is responsible for posting new assertions to the assertion chain.
func (s *Manager) Start(ctx context.Context) {
	go s.postAssertionRoutine(ctx)

	latestConfirmed, err := s.chain.LatestConfirmed(ctx)
	if err != nil {
		srvlog.Error("Could not get latest confirmed assertion", log.Ctx{"err": err})
		return
	}
	fromBlock, err := latestConfirmed.CreatedAtBlock()
	if err != nil {
		srvlog.Error("Could not get creation block", log.Ctx{"err": err})
		return
	}

	filterer, err := retry.UntilSucceeds(ctx, func() (*rollupgen.RollupUserLogicFilterer, error) {
		return rollupgen.NewRollupUserLogicFilterer(s.rollupAddr, s.backend)
	})
	if err != nil {
		srvlog.Error("Could not get rollup user logic filterer", log.Ctx{"err": err})
		return
	}
	filterOpts := &bind.FilterOpts{
		Start:   fromBlock,
		End:     nil,
		Context: ctx,
	}
	_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
		return true, s.checkForAssertionAdded(ctx, filterer, filterOpts)
	})
	if err != nil {
		srvlog.Error("Could not check for assertion added event")
		return
	}

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := s.backend.HeaderByNumber(ctx, nil)
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
				return true, s.checkForAssertionAdded(ctx, filterer, filterOpts)
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

func (s *Manager) ForksDetected() uint64 {
	return s.forksDetectedCount
}

func (s *Manager) ChallengesSubmitted() uint64 {
	return s.challengesSubmittedCount
}

func (s *Manager) AssertionsProcessed() uint64 {
	return s.assertionsProcessedCount
}

func (s *Manager) checkForAssertionAdded(
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
	for it.Next() {
		if it.Error() != nil {
			return errors.Wrapf(
				err,
				"got iterator error when scanning assertion creations from block %d to %d",
				filterOpts.Start,
				*filterOpts.End,
			)
		}
		assertionHash := protocol.AssertionHash{Hash: it.Event.AssertionHash}

		// Try to confirm the assertion in the background.
		go s.keepTryingAssertionConfirmation(ctx, assertionHash)

		// Try to process the assertion creation event in the background
		// to not block the processing of other incoming events.
		go func() {
			_, processErr := retry.UntilSucceeds(ctx, func() (bool, error) {
				return true, s.ProcessAssertionCreationEvent(ctx, assertionHash)
			}, retry.WithInterval(time.Minute))
			if processErr != nil {
				srvlog.Error(
					"Could not process assertion creation after retries",
					log.Ctx{"err": processErr},
				)
			}
		}()
	}
	return nil
}

// ProcessAssertionCreationEvent by checking if we agree with its claimed state.
// If we do not, we attempt to post a rival assertion along the fork and initiate a challenge
// if we are configured to do so. If we have not yet caught up to the claimed state,
// this function will then return an error.
func (m *Manager) ProcessAssertionCreationEvent(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
) error {
	// Ignore assertions we have submitted ourselves.
	if m.submittedAssertions.Has(assertionHash.Hash) {
		return nil
	}
	if assertionHash.Hash == (common.Hash{}) {
		return nil // Assertions cannot have a zero hash, not even genesis.
	}
	creationInfo, err := m.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return errors.Wrapf(err, "could not read assertion creation info for %#x", assertionHash.Hash)
	}
	if creationInfo.ParentAssertionHash == (common.Hash{}) {
		return nil // Skip processing genesis, as it has a parent assertion hash of 0x0.
	}

	// Check if we agree with the assertion's claimed state.
	claimedState := protocol.GoExecutionStateFromSolidity(creationInfo.AfterState)
	err = m.stateProvider.AgreesWithExecutionState(ctx, claimedState)
	switch {
	case errors.Is(err, l2stateprovider.ErrNoExecutionState):
		// If we disagree with the execution state, we should try to post the rival
		// assertion that we believe is correct and initiate a challenge if possible.
		if postRivalErr := m.postRivalAssertionAndChallenge(ctx, creationInfo); postRivalErr != nil {
			return postRivalErr
		}
		m.assertionsProcessedCount++
		return nil
	case errors.Is(err, l2stateprovider.ErrChainCatchingUp):
		// Otherwise, we return the error that we are still catching up to the
		// execution state claimed by the assertion, and this function will be retried
		// by the caller if wrapped in a retryable call.
		return fmt.Errorf(
			"chain still catching up to processed execution state - "+
				"will reattempt assertion processing when caught up: %w",
			l2stateprovider.ErrChainCatchingUp,
		)
	case err != nil:
		return err
	}
	// If no error, this means we agree with the claimed assertion state
	// so there is no action to take.
	machineFinishedHash := crypto.Keccak256Hash([]byte("Machine finished:"), claimedState.GlobalState.Hash().Bytes())
	srvlog.Info("Agreed with incoming assertion", log.Ctx{
		"validatorName":       m.validatorName,
		"claimedState":        fmt.Sprintf("%+v", claimedState),
		"machineFinishedHash": machineFinishedHash,
		"assertionHash":       assertionHash,
	})
	m.assertionsProcessedCount++
	return nil
}

// Attempts to post a rival assertion to a given assertion and then attempts to
// open a challenge on that fork in the chain if configured to do so.
func (m *Manager) postRivalAssertionAndChallenge(
	ctx context.Context,
	creationInfo *protocol.AssertionCreatedInfo,
) error {
	if !creationInfo.InboxMaxCount.IsUint64() {
		return errors.New("inbox max count not a uint64")
	}
	batchCount := creationInfo.InboxMaxCount.Uint64()
	claimedState := protocol.GoExecutionStateFromSolidity(creationInfo.AfterState)
	logFields := log.Ctx{
		"validatorName":         m.validatorName,
		"parentAssertionHash":   creationInfo.ParentAssertionHash,
		"detectedAssertionHash": creationInfo.AssertionHash,
		"batchCount":            batchCount,
		"claimedExecutionState": fmt.Sprintf("%+v", claimedState),
	}
	if !m.canRespondToAssertion() {
		srvlog.Warn("Detected invalid assertion, but not configured to challenge", logFields)
		return nil
	}

	srvlog.Info("Disagreed with execution state from observed assertion", logFields)

	// Post what we believe is the correct rival assertion that follows the ancestor we agree with.
	correctRivalAssertion, err := m.maybePostRivalAssertion(ctx, creationInfo)
	if err != nil {
		return err
	}
	correctClaimedAssertionHash := correctRivalAssertion.Id()

	// Generating a random integer between 0 and max delay second to wait before challenging.
	// This is to avoid all validators challenging at the same time.
	// TODO: Abstract into a smaller function.
	mds := 1 // default max delay seconds to 1 to avoid panic
	if m.challengeReader.MaxDelaySeconds() > 1 {
		mds = m.challengeReader.MaxDelaySeconds()
	}
	randSecs, err := randUint64(uint64(mds))
	if err != nil {
		return err
	}
	srvlog.Info("Waiting before submitting challenge on assertion", log.Ctx{"delay": randSecs})
	time.Sleep(time.Duration(randSecs) * time.Second)

	if err := m.challengeCreator.ChallengeAssertion(ctx, correctClaimedAssertionHash); err != nil {
		return err
	}
	m.challengesSubmittedCount++
	return nil

}

// Attempt to post a rival assertion based on the last agreed with ancestor
// of a given assertion.
//
// If this parent assertion already has a rival we agree with that arleady exists
// then this function will return that assertion.
func (m *Manager) maybePostRivalAssertion(
	ctx context.Context, creationInfo *protocol.AssertionCreatedInfo,
) (protocol.Assertion, error) {
	latestAgreedWithAncestor, err := m.findLastAgreedWithAncestor(ctx, creationInfo)
	if err != nil {
		return nil, err
	}
	// Post what we believe is the correct assertion that follows the ancestor we agree with.
	staked, err := m.chain.IsStaked(ctx)
	if err != nil {
		return nil, err
	}
	// If the validator is already staked, we post an assertion and move existing stake to it.
	if staked {
		assertion, postErr := m.PostAssertionBasedOnParent(
			ctx, latestAgreedWithAncestor, m.chain.StakeOnNewAssertion,
		)
		if postErr != nil {
			return nil, postErr
		}
		m.submittedAssertions.Insert(assertion.Id().Hash)
		return assertion, nil
	}
	// Otherwise, we post a new assertion and place a new stake on it.
	assertion, err := m.PostAssertionBasedOnParent(
		ctx, latestAgreedWithAncestor, m.chain.NewStakeOnNewAssertion,
	)
	if err != nil {
		return nil, err
	}
	m.submittedAssertions.Insert(assertion.Id().Hash)
	return assertion, nil
}

// Look back until we find the ancestor we agree with for the given assertion.
func (m *Manager) findLastAgreedWithAncestor(
	ctx context.Context, assertionCreationInfo *protocol.AssertionCreatedInfo,
) (*protocol.AssertionCreatedInfo, error) {
	latestConfirmed, err := m.chain.LatestConfirmed(ctx)
	if err != nil {
		return nil, err
	}
	latestConfirmedInfo, err := m.chain.ReadAssertionCreationInfo(ctx, latestConfirmed.Id())
	if err != nil {
		return nil, err
	}
	agreedWithAncestor := latestConfirmed.Id().Hash
	cursor := assertionCreationInfo.ParentAssertionHash
	for cursor != agreedWithAncestor {
		// Get the cursor's creation info.
		parentCreationInfo, err := m.chain.ReadAssertionCreationInfo(
			ctx, protocol.AssertionHash{Hash: cursor},
		)
		if err != nil {
			return nil, err
		}
		parentExecState := protocol.GoExecutionStateFromSolidity(parentCreationInfo.AfterState)
		if err = m.stateProvider.AgreesWithExecutionState(ctx, parentExecState); err != nil {
			if errors.Is(err, l2stateprovider.ErrNoExecutionState) {
				// Disagreed with parent. This means we should look at the
				// grandparent and continue our loop.
				cursor = parentCreationInfo.ParentAssertionHash
				continue
			}
			return nil, err
		}
		// No error means we agree with this parent. We can break the loop.
		return parentCreationInfo, nil
	}
	return latestConfirmedInfo, nil
}

func (s *Manager) keepTryingAssertionConfirmation(ctx context.Context, assertionHash protocol.AssertionHash) {
	ticker := time.NewTicker(s.confirmationAttemptInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			status, err := s.chain.AssertionStatus(ctx, assertionHash)
			if err != nil {
				srvlog.Error("Could not get assertion by hash", log.Ctx{"err": err, "assertionHash": assertionHash.Hash})
				continue
			}
			if status == protocol.NoAssertion {
				srvlog.Error("No assertion found by hash", log.Ctx{"err": err, "assertionHash": assertionHash.Hash})
				continue
			}
			if status == protocol.AssertionConfirmed {
				srvlog.Info("Assertion confirmed", log.Ctx{"assertionHash": assertionHash.Hash})
				return
			}
			err = s.chain.ConfirmAssertionByTime(ctx, assertionHash)
			if err != nil {
				continue
			}
			srvlog.Info("Assertion confirmed", log.Ctx{"assertionHash": assertionHash.Hash})
			return
		case <-ctx.Done():
			return
		}
	}
}

// Returns true if the manager can respond to an assertion with a challenge.
func (m *Manager) canRespondToAssertion() bool {
	return m.challengeReader.Mode() == types.DefensiveMode || m.challengeReader.Mode() == types.MakeMode
}

func randUint64(max uint64) (uint64, error) {
	n, err := rand.Int(rand.Reader, new(big.Int).SetUint64(max))
	if err != nil {
		return 0, err
	}
	if !n.IsUint64() {
		return 0, errors.New("not a uint64")
	}
	return n.Uint64(), nil
}
