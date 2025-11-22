// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package assertions contains testing utilities for posting and scanning for
// assertions on chain, which are useful for simulating the responsibilities of
// Arbitrum Nitro and initiating challenges as needed using our challenge
// manager.
package assertions

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/bold/api/db"
	protocol "github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	l2stateprovider "github.com/offchainlabs/nitro/bold/layer2-state-provider"
	retry "github.com/offchainlabs/nitro/bold/runtime"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	evilAssertionCounter                  = metrics.NewRegisteredCounter("arb/validator/scanner/evil_assertion", nil)
	assertionConfirmedCounter             = metrics.GetOrRegisterCounter("arb/validator/scanner/assertion_confirmed", nil)
	errorConfirmingAssertionByTimeCounter = metrics.NewRegisteredCounter("arb/validator/scanner/error_confirming_assertion_by_time", nil)
	latestConfirmedAssertionGauge         = metrics.NewRegisteredGauge("arb/validator/scanner/latest_confirmed_assertion_block_number", nil)
	safeBlockDelayCounter                 = metrics.GetOrRegisterCounter("arb/validator/scanner/safe_block_delay", nil)
)

type timings struct {
	pollInterval   time.Duration
	confInterval   time.Duration
	postInterval   time.Duration
	avgBlockTime   time.Duration
	minGapToParent time.Duration
}

var defaultTimings = timings{
	pollInterval:   time.Minute,
	confInterval:   time.Second * 10,
	postInterval:   time.Hour,
	avgBlockTime:   time.Second * 12,
	minGapToParent: time.Minute * 15,
}

// The Manager struct is responsible for several tasks related to the assertion
// chain:
//
// 1. It continuously polls the assertion chain to check for posted, on-chain
// assertions starting from the latest confirmed assertion up to the newest one.
// 2. As the assertion chain advances, the Manager keeps polling to stay
// updated.
// 3. Upon observing each new assertion, the Manager evaluates whether it should
// challenge the assertion or not.
// 4. The Manager frequently posts new assertions to the assertion chain at
// specific intervals.
// 5. When posting assertions, it relies on the most recent execution state
// available in its local execution provider.
type Manager struct {
	stopwaiter.StopWaiter
	chain                       protocol.AssertionChain
	backend                     protocol.ChainBackend
	execProvider                l2stateprovider.ExecutionProvider
	times                       timings
	rollupAddr                  common.Address
	validatorName               string
	forksDetectedCount          uint64
	assertionsProcessedCount    uint64
	submittedRivalsCount        uint64
	submittedAssertions         *threadsafe.LruSet[protocol.AssertionHash]
	apiDB                       db.Database
	assertionChainData          *assertionChainData
	observedCanonicalAssertions chan protocol.AssertionHash
	isReadyToPost               bool
	disablePosting              bool
	startPostingSignal          chan struct{}
	enableFastConfirmation      bool
	mode                        types.Mode
	rivalHandler                types.RivalHandler
	delegatedStaking            bool
	autoDeposit                 bool
	autoAllowanceApproval       bool
	maxGetLogBlocks             uint64
	confirming                  *threadsafe.LruSet[protocol.AssertionHash]
	confirmQueueMutex           sync.Mutex
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

func WithFastConfirmation() Opt {
	return func(m *Manager) {
		m.enableFastConfirmation = true
	}
}

func WithDangerousReadyToPost() Opt {
	return func(m *Manager) {
		m.isReadyToPost = true
	}
}

func WithDelegatedStaking() Opt {
	return func(m *Manager) {
		m.delegatedStaking = true
	}
}

func WithoutAutoDeposit() Opt {
	return func(m *Manager) {
		m.autoDeposit = false
	}
}

func WithoutAutoAllowanceApproval() Opt {
	return func(m *Manager) {
		m.autoAllowanceApproval = false
	}
}

// WithAPIDB sets the database to use for the assertion manager.
func WithAPIDB(db db.Database) Opt {
	return func(m *Manager) {
		m.apiDB = db
	}
}

// WithPostingInterval overrides the default posting interval.
//
// This interval is the amount of time the assertsion manager will wait between
// attempts to post assertions, unless the previous assertion was an overflow
// asseartion. If the previous assertion was an overflow assertion, and the
// assertion manager has the data it needs to post an additional assertion,
// it will disregard the posting interval and post right away.
func WithPostingInterval(t time.Duration) Opt {
	return func(m *Manager) {
		m.times.postInterval = t
	}
}

// WithPollingInterval overrides the default polling interval.
//
// This interval is the amount of time the assertion manager will wait between
// atteampts to read new asseartions from the parent chain.
func WithPollingInterval(t time.Duration) Opt {
	return func(m *Manager) {
		m.times.pollInterval = t
	}
}

// WithConfirmationInterval overrides the default a confiramtion interval.
//
// This is the interval the assertion manager will wait between attempts to
// persist information about which assertions can be confirmed to the parent
// chain.
func WithConfirmationInterval(t time.Duration) Opt {
	return func(m *Manager) {
		m.times.confInterval = t
	}
}

// WithAverageBlockCreationTime overrides the default average block creation
// time.
//
// The average block cretion time is used by the assertion manager to emit
// warnings if the parent chain hasn't had any new blocks for considerably
// longer than this expected delay.
func WithAverageBlockCreationTime(t time.Duration) Opt {
	return func(m *Manager) {
		m.times.avgBlockTime = t
	}
}

// WithMaxGetLogBlocks overrides the default maximum number of blocks to get
// logs for in a single call.
func WithMaxGetLogBlocks(n uint64) Opt {
	return func(m *Manager) {
		m.maxGetLogBlocks = n
	}
}

// WithMinimumGapToParentAssertion overrides the default minimum gap (in duration)
// to parent assertion creation time.
//
// The minimum gap to parent assertion is used by the assertion manager to wait
// until this much amount of duration is passed since the parent assertion was created
// before posting a new assertion.
func WithMinimumGapToParentAssertion(t time.Duration) Opt {
	return func(m *Manager) {
		m.times.minGapToParent = t
	}
}

// NewManager creates a manager from the required dependencies.
func NewManager(
	chain protocol.AssertionChain,
	execProvider l2stateprovider.ExecutionProvider,
	validatorName string,
	mode types.Mode,
	opts ...Opt,
) (*Manager, error) {
	maxAssertions, err := safecast.ToInt(chain.MaxAssertionsPerChallengePeriod())
	if err != nil {
		return nil, errors.Wrap(err, "could not convert max assertions to int")
	}
	m := &Manager{
		chain:                    chain,
		apiDB:                    nil,
		backend:                  chain.Backend(),
		execProvider:             execProvider,
		rollupAddr:               chain.RollupAddress(),
		validatorName:            validatorName,
		times:                    defaultTimings,
		forksDetectedCount:       0,
		assertionsProcessedCount: 0,
		submittedAssertions:      threadsafe.NewLruSet(maxAssertions, threadsafe.LruSetWithMetric[protocol.AssertionHash]("submittedAssertions")),
		assertionChainData: &assertionChainData{
			latestAgreedAssertion: protocol.AssertionHash{},
			canonicalAssertions:   make(map[protocol.AssertionHash]*protocol.AssertionCreatedInfo),
		},
		observedCanonicalAssertions: make(chan protocol.AssertionHash, maxAssertions),
		isReadyToPost:               false,
		startPostingSignal:          make(chan struct{}),
		mode:                        mode,
		rivalHandler:                nil, // Must be set after construction if mode > DefensiveMode
		autoDeposit:                 true,
		autoAllowanceApproval:       true,
		maxGetLogBlocks:             1000,
		confirming:                  threadsafe.NewLruSet[protocol.AssertionHash](maxAssertions),
		confirmQueueMutex:           sync.Mutex{},
	}
	for _, o := range opts {
		o(m)
	}
	if m.times.pollInterval == 0 {
		return nil, errors.New("assertion polling interval must be greater than 0")
	}
	if m.times.confInterval == 0 {
		return nil, errors.New("assertion confirmation attempt interval must be greater than 0")
	}
	return m, nil
}

// SetRivalHandler sets the rival handler for the assertion manager.
func (m *Manager) SetRivalHandler(handler types.RivalHandler) {
	m.rivalHandler = handler
}

func (m *Manager) Start(ctx context.Context) {
	m.StopWaiter.Start(ctx, m)
	if m.mode != types.WatchTowerMode {
		if m.delegatedStaking {
			// Attempt to become a new staker onchain until successful.
			// This is only relevant for delegated stakers that will be funded
			// by another party.
			_, err := retry.UntilSucceeds(ctx, func() (bool, error) {
				if err2 := m.chain.NewStake(ctx); err2 != nil {
					return false, err2
				}
				return true, nil
			})
			if err != nil {
				log.Error("Could not become a delegated staker onchain", "err", err)
				return
			}
		}
		if m.autoDeposit {
			// Attempt to auto-deposit funds until successful into the stake token.
			_, err := retry.UntilSucceeds(ctx, func() (bool, error) {
				callOpts := m.chain.GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{Context: ctx})
				latestConfirmed, err2 := m.chain.LatestConfirmed(ctx, callOpts)
				if err2 != nil {
					return false, err2
				}
				latestConfirmedInfo, err2 := m.chain.ReadAssertionCreationInfo(ctx, latestConfirmed.Id())
				if err2 != nil {
					return false, err2
				}
				if err2 := m.chain.AutoDepositTokenForStaking(ctx, latestConfirmedInfo.RequiredStake); err2 != nil {
					return false, err2
				}
				return true, nil
			})
			if err != nil {
				log.Error("Could not auto-deposit funds to become a staker", "err", err)
				return
			}
		}
		if m.autoAllowanceApproval {
			// Attempt to auto-approve the stake token spending by the challenge manager
			// and rollup address until successful.
			_, err := retry.UntilSucceeds(ctx, func() (bool, error) {
				if err2 := m.chain.ApproveAllowances(ctx); err2 != nil {
					return false, err2
				}
				return true, nil
			})
			if err != nil {
				log.Error("Could not auto-approve allowances", "err", err)
				return
			}
		}
	}
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
			latestSafeBlock, err := m.backend.HeaderByNumber(ctx, big.NewInt(int64(rpc.SafeBlockNumber)))
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
			latestBlockTime, err := safecast.ToInt64(latestBlock.Time)
			if err != nil {
				log.Error("Error casting latest block time to int64", "err", err)
				continue
			}
			latestSafeBlockTime, err := safecast.ToInt64(latestSafeBlock.Time)
			if err != nil {
				log.Error("Error casting latest safe block time to int64", "err", err)
				continue
			}
			safeBlockDelayInSeconds := time.Unix(latestBlockTime, 0).Sub(time.Unix(latestSafeBlockTime, 0)).Seconds()
			if safeBlockDelayInSeconds > 1200 {
				log.Warn("Latest safe block is delayed by more that 20 minutes", "latestSafeBlock", latestSafeBlock.Number.Uint64(), "latestBlock", latestBlock.Number.Uint64())
				safeBlockDelayCounter.Inc(1)
			}
		}
	}
}

func (m *Manager) ExecutionStateAfterParent(ctx context.Context, parentInfo *protocol.AssertionCreatedInfo) (*protocol.ExecutionState, error) {
	goGlobalState := protocol.GoGlobalStateFromSolidity(parentInfo.AfterState.GlobalState)
	return m.execProvider.ExecutionStateAfterPreviousState(ctx, parentInfo.InboxMaxCount.Uint64(), goGlobalState)
}

func (m *Manager) ForksDetected() uint64 {
	return m.forksDetectedCount
}

func (m *Manager) AssertionsProcessed() uint64 {
	return m.assertionsProcessedCount
}

func (m *Manager) SubmittedRivals() uint64 {
	return m.submittedRivalsCount
}

func (m *Manager) AssertionsSubmittedInProcess() []protocol.AssertionHash {
	hashes := make([]protocol.AssertionHash, 0)
	m.submittedAssertions.ForEach(func(elem protocol.AssertionHash) {
		hashes = append(hashes, elem)
	})
	return hashes
}

func (m *Manager) LatestAgreedAssertion() protocol.AssertionHash {
	m.assertionChainData.RLock()
	defer m.assertionChainData.RUnlock()
	return m.assertionChainData.latestAgreedAssertion
}
