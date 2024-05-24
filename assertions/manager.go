// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

// Package assertions contains testing utilities for posting and scanning for
// assertions on chain, which are useful for simulating the responsibilities
// of Arbitrum Nitro and initiating challenges as needed using our challenge manager.
package assertions

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/OffchainLabs/bold/util/stopwaiter"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/OffchainLabs/bold/api/db"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	evilAssertionCounter                  = metrics.NewRegisteredCounter("arb/validator/scanner/evil_assertion", nil)
	challengeSubmittedCounter             = metrics.NewRegisteredCounter("arb/validator/scanner/challenge_submitted", nil)
	assertionConfirmedCounter             = metrics.GetOrRegisterCounter("arb/validator/scanner/assertion_confirmed", nil)
	errorConfirmingAssertionByTimeCounter = metrics.NewRegisteredCounter("arb/validator/scanner/error_confirming_assertion_by_time", nil)
	latestConfirmedAssertionGauge         = metrics.NewRegisteredGauge("arb/validator/scanner/latest_confirmed_assertion_block_number", nil)
	evilAssertionConfirmedCounter         = metrics.GetOrRegisterCounter("arb/validator/scanner/evil_assertion_confirmed", nil)
	safeBlockDelayCounter                 = metrics.GetOrRegisterCounter("arb/validator/scanner/safe_block_delay", nil)
)

// The Manager struct is responsible for several tasks related to the assertion chain:
// 1. It continuously polls the assertion chain to check for posted, on-chain assertions starting from the latest confirmed assertion up to the newest one.
// 2. As the assertion chain advances, the Manager keeps polling to stay updated.
// 3. Upon observing each new assertion, the Manager evaluates whether it should challenge the assertion or not.
// 4. The Manager frequently posts new assertions to the assertion chain at specific intervals.
// 5. When posting assertions, it relies on the most recent execution state available in its local state manager.
type Manager struct {
	stopwaiter.StopWaiter
	chain                       protocol.AssertionChain
	backend                     bind.ContractBackend
	challengeCreator            types.ChallengeCreator
	challengeReader             types.ChallengeReader
	stateProvider               l2stateprovider.ExecutionProvider
	pollInterval                time.Duration
	confirmationAttemptInterval time.Duration
	averageTimeForBlockCreation time.Duration
	rollupAddr                  common.Address
	challengeManagerAddr        common.Address
	validatorName               string
	forksDetectedCount          uint64
	challengesSubmittedCount    uint64
	assertionsProcessedCount    uint64
	submittedRivalsCount        uint64
	postInterval                time.Duration
	submittedAssertions         *threadsafe.LruSet[common.Hash]
	apiDB                       db.Database
	assertionChainData          *assertionChainData
	observedCanonicalAssertions chan protocol.AssertionHash
	isReadyToPost               bool
	disablePosting              bool
	startPostingSignal          chan struct{}
	layerZeroHeightsCache       *protocol.LayerZeroHeights
	layerZeroHeightsCacheLock   sync.RWMutex
}

type assertionChainData struct {
	sync.RWMutex
	latestAgreedAssertion protocol.AssertionHash
	canonicalAssertions   map[protocol.AssertionHash]*protocol.AssertionCreatedInfo
}

type Opt func(*Manager)

func WithPostingDisabled() Opt {
	return func(m *Manager) {
		m.disablePosting = true
	}
}

func WithDangerousReadyToPost() Opt {
	return func(m *Manager) {
		m.isReadyToPost = true
	}
}

// NewManager creates a manager from the required dependencies.
func NewManager(
	chain protocol.AssertionChain,
	stateProvider l2stateprovider.Provider,
	backend bind.ContractBackend,
	challengeManager types.ChallengeManager,
	rollupAddr common.Address,
	challengeManagerAddr common.Address,
	validatorName string,
	pollInterval,
	assertionConfirmationAttemptInterval time.Duration,
	stateManager l2stateprovider.ExecutionProvider,
	postInterval time.Duration,
	averageTimeForBlockCreation time.Duration,
	apiDB db.Database,
	opts ...Opt,
) (*Manager, error) {
	if pollInterval == 0 {
		return nil, errors.New("assertion scanning interval must be greater than 0")
	}
	if assertionConfirmationAttemptInterval == 0 {
		return nil, errors.New("assertion confirmation attempt interval must be greater than 0")
	}
	m := &Manager{
		chain:                       chain,
		apiDB:                       apiDB,
		backend:                     backend,
		stateProvider:               stateProvider,
		challengeCreator:            challengeManager,
		challengeReader:             challengeManager,
		rollupAddr:                  rollupAddr,
		challengeManagerAddr:        challengeManagerAddr,
		validatorName:               validatorName,
		pollInterval:                pollInterval,
		confirmationAttemptInterval: assertionConfirmationAttemptInterval,
		forksDetectedCount:          0,
		challengesSubmittedCount:    0,
		assertionsProcessedCount:    0,
		postInterval:                postInterval,
		submittedAssertions:         threadsafe.NewLruSet[common.Hash](1000, threadsafe.LruSetWithMetric[common.Hash]("submittedAssertions")),
		averageTimeForBlockCreation: averageTimeForBlockCreation,
		assertionChainData: &assertionChainData{
			latestAgreedAssertion: protocol.AssertionHash{},
			canonicalAssertions:   make(map[protocol.AssertionHash]*protocol.AssertionCreatedInfo),
		},
		observedCanonicalAssertions: make(chan protocol.AssertionHash, 1000),
		isReadyToPost:               false,
		startPostingSignal:          make(chan struct{}),
	}
	for _, o := range opts {
		o(m)
	}
	return m, nil
}

func (m *Manager) Start(ctx context.Context) {
	m.StopWaiter.Start(ctx, m)
	if !m.disablePosting {
		m.LaunchThread(m.postAssertionRoutine)
	}
	m.LaunchThread(m.updateLatestConfirmedMetrics)
	m.LaunchThread(m.syncAssertions)
	m.LaunchThread(m.queueCanonicalAssertionsForConfirmation)
	m.LaunchThread(m.checkLatestDesiredBlock)
}

func (m *Manager) checkLatestDesiredBlock(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
			latestSafeBlock, err := m.backend.HeaderByNumber(ctx, m.chain.GetDesiredRpcHeadBlockNumber())
			if err != nil {
				log.Error("Error getting latest safe block", "err", err)
				continue
			}
			if !latestSafeBlock.Number.IsUint64() {
				log.Error("Latest safe block number not a uint64")
				continue
			}

			latestBlock, err := m.backend.HeaderByNumber(ctx, nil)
			if err != nil {
				log.Error("Error getting latest block", "err", err)
				continue
			}
			if !latestBlock.Number.IsUint64() {
				log.Error("Latest block number not a uint64")
				continue
			}
			safeBlockDelayInSeconds := (latestBlock.Number.Uint64() - latestSafeBlock.Number.Uint64()) * uint64(m.averageTimeForBlockCreation.Seconds())
			if safeBlockDelayInSeconds > 1200 {
				log.Warn("Latest safe block is delayed by more that 20 minutes", "latestSafeBlock", latestSafeBlock.Number.Uint64(), "latestBlock", latestBlock.Number.Uint64())
				safeBlockDelayCounter.Inc(1)
			}
		}
	}
}

func (m *Manager) LayerZeroHeights(ctx context.Context) (*protocol.LayerZeroHeights, error) {
	m.layerZeroHeightsCacheLock.RLock()
	cachedValue := m.layerZeroHeightsCache
	m.layerZeroHeightsCacheLock.RUnlock()
	if cachedValue != nil {
		return cachedValue, nil
	}

	m.layerZeroHeightsCacheLock.Lock()
	defer m.layerZeroHeightsCacheLock.Unlock()
	cm, err := m.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	layerZeroHeights, err := cm.LayerZeroHeights(ctx)
	if err != nil {
		return nil, err
	}
	m.layerZeroHeightsCache = layerZeroHeights
	return layerZeroHeights, nil
}

func (m *Manager) ExecutionStateAfterParent(ctx context.Context, parentInfo *protocol.AssertionCreatedInfo) (*protocol.ExecutionState, error) {
	layerZeroHeights, err := m.LayerZeroHeights(ctx)
	if err != nil {
		return nil, err
	}
	if layerZeroHeights.BlockChallengeHeight == 0 {
		return nil, errors.New("block challenge height is zero")
	}
	goGlobalState := protocol.GoGlobalStateFromSolidity(parentInfo.AfterState.GlobalState)
	return m.stateProvider.ExecutionStateAfterPreviousState(ctx, parentInfo.InboxMaxCount.Uint64(), &goGlobalState, layerZeroHeights.BlockChallengeHeight-1)
}

func (m *Manager) ForksDetected() uint64 {
	return m.forksDetectedCount
}

func (m *Manager) ChallengesSubmitted() uint64 {
	return m.challengesSubmittedCount
}

func (m *Manager) AssertionsProcessed() uint64 {
	return m.assertionsProcessedCount
}

func (m *Manager) SubmittedRivals() uint64 {
	return m.submittedRivalsCount
}

func (m *Manager) AssertionsSubmittedInProcess() []common.Hash {
	hashes := make([]common.Hash, 0)
	m.submittedAssertions.ForEach(func(elem common.Hash) {
		hashes = append(hashes, elem)
	})
	return hashes
}

func (m *Manager) logChallengeConfigs(ctx context.Context) error {
	cm, err := m.chain.SpecChallengeManager(ctx)
	if err != nil {
		return err
	}
	bigStepNum, err := cm.NumBigSteps(ctx)
	if err != nil {
		return err
	}
	challengePeriodBlocks, err := cm.ChallengePeriodBlocks(ctx)
	if err != nil {
		return err
	}
	layerZeroHeights, err := m.LayerZeroHeights(ctx)
	if err != nil {
		return err
	}
	log.Info("Opening challenge with the following configuration",
		"address", cm.Address(),
		"bigStepNumber", bigStepNum,
		"challengePeriodBlocks", challengePeriodBlocks,
		"layerZeroHeights", layerZeroHeights,
	)
	return nil
}

// Returns true if the manager can respond to an assertion with a challenge.
func (m *Manager) canPostRivalAssertion() bool {
	return m.challengeReader.Mode() >= types.DefensiveMode
}

func (m *Manager) canPostChallenge() bool {
	return m.challengeReader.Mode() > types.DefensiveMode
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
