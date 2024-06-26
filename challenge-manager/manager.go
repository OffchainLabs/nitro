// Package challengemanager includes the main entrypoint for setting up a BOLD
// challenge manager instance and challenging assertions onchain.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package challengemanager

import (
	"context"
	"fmt"
	"time"

	apibackend "github.com/OffchainLabs/bold/api/backend"
	"github.com/OffchainLabs/bold/api/db"
	"github.com/OffchainLabs/bold/api/server"
	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	watcher "github.com/OffchainLabs/bold/challenge-manager/chain-watcher"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers/events"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	utilTime "github.com/OffchainLabs/bold/time"
	"github.com/OffchainLabs/bold/util/stopwaiter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Opt = func(val *Manager)

// Manager defines an offchain, challenge manager, which will be
// an active participant in interacting with the on-chain contracts.
type Manager struct {
	stopwaiter.StopWaiter
	chain                       protocol.Protocol
	chalManagerAddr             common.Address
	rollupAddr                  common.Address
	rollup                      *rollupgen.RollupCore
	rollupFilterer              *rollupgen.RollupCoreFilterer
	chalManager                 *challengeV2gen.EdgeChallengeManagerFilterer
	backend                     bind.ContractBackend
	client                      *rpc.Client
	stateManager                l2stateprovider.Provider
	address                     common.Address
	name                        string
	timeRef                     utilTime.Reference
	chainWatcherInterval        time.Duration
	watcher                     *watcher.Watcher
	trackedEdgeIds              *threadsafe.Map[protocol.EdgeId, *edgetracker.Tracker]
	batchIndexForAssertionCache *threadsafe.LruMap[protocol.AssertionHash, edgetracker.AssociatedAssertionMetadata]
	newBlockNotifier            *events.Producer[*gethtypes.Header]
	notifyOnNumberOfBlocks      uint64
	// Optional list of challenges to track, keyed by challenged parent assertion hash. If nil,
	// all challenges will be tracked.
	trackChallengeParentAssertionHashes []protocol.AssertionHash
	assertionManager                    *assertions.Manager
	assertionPostingInterval            time.Duration
	assertionScanningInterval           time.Duration
	assertionConfirmingInterval         time.Duration
	averageTimeForBlockCreation         time.Duration
	mode                                types.Mode
	maxDelaySeconds                     int
	claimedAssertionsInChallenge        *threadsafe.LruSet[protocol.AssertionHash]
	enableFastConfirmation              bool
	headBlockSubscriptions              bool
	// API
	apiAddr   string
	apiDBPath string
	api       *server.Server
	apiDB     db.Database
}

// WithName is a human-readable identifier for this challenge manager for logging purposes.
func WithName(name string) Opt {
	return func(val *Manager) {
		val.name = name
	}
}

// WithFastConfirmation enables fast confirmation of challenges.
func WithFastConfirmation() Opt {
	return func(val *Manager) {
		val.enableFastConfirmation = true
	}
}

// WithAddress gives a staker address to the validator.
func WithAddress(addr common.Address) Opt {
	return func(val *Manager) {
		val.address = addr
	}
}

func WithAssertionPostingInterval(d time.Duration) Opt {
	return func(val *Manager) {
		val.assertionPostingInterval = d
	}
}

func WithAssertionScanningInterval(d time.Duration) Opt {
	return func(val *Manager) {
		val.assertionScanningInterval = d
	}
}

func WithAssertionConfirmingInterval(d time.Duration) Opt {
	return func(val *Manager) {
		val.assertionConfirmingInterval = d
	}
}

func WithAvgBlockCreationTime(d time.Duration) Opt {
	return func(val *Manager) {
		val.averageTimeForBlockCreation = d
	}
}

// Edges tick on every block received from the parent chain of the rollup, by default. Alternatively,
// they can be configured to tick every N blocks.
func WithTickEdgesOnNumberOfBlocks(n uint64) Opt {
	return func(val *Manager) {
		val.notifyOnNumberOfBlocks = n
	}
}

// WithMode specifies the mode of the challenge manager.
func WithMode(m types.Mode) Opt {
	return func(val *Manager) {
		val.mode = m
	}
}

// WithAPIEnabled specifies whether or not to enable the API and the address to listen on.
func WithAPIEnabled(addr string, dbPath string) Opt {
	return func(val *Manager) {
		val.apiAddr = addr
		val.apiDBPath = dbPath
	}
}

func WithRPCClient(client *rpc.Client) Opt {
	return func(val *Manager) {
		val.client = client
	}
}

func WithTrackChallengeParentAssertionHashes(trackChallengeParentAssertionHashes []string) Opt {
	return func(val *Manager) {
		val.trackChallengeParentAssertionHashes = make([]protocol.AssertionHash, len(trackChallengeParentAssertionHashes))
		for i, hash := range trackChallengeParentAssertionHashes {
			val.trackChallengeParentAssertionHashes[i] = protocol.AssertionHash{Hash: common.HexToHash(hash)}
		}
	}
}

func WithHeadBlockSubscriptions() Opt {
	return func(val *Manager) {
		val.headBlockSubscriptions = true
	}
}

// New sets up a challenge manager instance provided a protocol, state manager, and additional options.
func New(
	ctx context.Context,
	chain protocol.Protocol,
	stateManager l2stateprovider.Provider,
	rollupAddr common.Address,
	opts ...Opt,
) (*Manager, error) {

	m := &Manager{
		backend:                      chain.Backend(),
		chain:                        chain,
		stateManager:                 stateManager,
		address:                      common.Address{},
		timeRef:                      utilTime.NewRealTimeReference(),
		rollupAddr:                   rollupAddr,
		chainWatcherInterval:         time.Millisecond * 500,
		trackedEdgeIds:               threadsafe.NewMap[protocol.EdgeId, *edgetracker.Tracker](threadsafe.MapWithMetric[protocol.EdgeId, *edgetracker.Tracker]("trackedEdgeIds")),
		batchIndexForAssertionCache:  threadsafe.NewLruMap[protocol.AssertionHash, edgetracker.AssociatedAssertionMetadata](1000, threadsafe.LruMapWithMetric[protocol.AssertionHash, edgetracker.AssociatedAssertionMetadata]("batchIndexForAssertionCache")),
		notifyOnNumberOfBlocks:       1,
		newBlockNotifier:             events.NewProducer[*gethtypes.Header](),
		assertionPostingInterval:     time.Hour,
		assertionScanningInterval:    time.Minute,
		assertionConfirmingInterval:  time.Second * 10,
		averageTimeForBlockCreation:  time.Second * 12,
		headBlockSubscriptions:       false,
		claimedAssertionsInChallenge: threadsafe.NewLruSet[protocol.AssertionHash](1000, threadsafe.LruSetWithMetric[protocol.AssertionHash]("claimedAssertionsInChallenge")),
	}
	for _, o := range opts {
		o(m)
	}
	chalManager, err := m.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	chalManagerAddr := chalManager.Address()

	rollup, err := rollupgen.NewRollupCore(rollupAddr, m.backend)
	if err != nil {
		return nil, err
	}
	rollupFilterer, err := rollupgen.NewRollupCoreFilterer(rollupAddr, m.backend)
	if err != nil {
		return nil, err
	}
	chalManagerFilterer, err := challengeV2gen.NewEdgeChallengeManagerFilterer(chalManagerAddr, m.backend)
	if err != nil {
		return nil, err
	}
	numBigStepLevels, err := chalManager.NumBigSteps(ctx)
	if err != nil {
		return nil, err
	}
	m.rollup = rollup
	m.rollupFilterer = rollupFilterer
	m.chalManagerAddr = chalManagerAddr
	m.chalManager = chalManagerFilterer

	if m.apiDBPath != "" {
		apiDB, err2 := db.NewDatabase(m.apiDBPath)
		if err2 != nil {
			return nil, err2
		}
		m.apiDB = apiDB
	}

	watcher, err := watcher.New(m.chain, m, m.stateManager, m.backend, m.chainWatcherInterval, numBigStepLevels, m.name, m.apiDB, m.assertionConfirmingInterval, m.averageTimeForBlockCreation, m.trackChallengeParentAssertionHashes)
	if err != nil {
		return nil, err
	}
	m.watcher = watcher

	if m.apiAddr != "" {
		bknd := apibackend.NewBackend(m.apiDB, m.chain, m.watcher, m)
		srv, err2 := server.New(m.apiAddr, bknd)
		if err2 != nil {
			return nil, err2
		}
		m.api = srv
	}
	assertionsOpts := make([]assertions.Opt, 0)
	if m.enableFastConfirmation {
		assertionsOpts = append(assertionsOpts, assertions.WithFastConfirmation())
	}
	assertionManager, err := assertions.NewManager(
		m.chain,
		m.stateManager,
		m.backend,
		m,
		m.rollupAddr,
		m.chalManagerAddr,
		m.name,
		m.assertionScanningInterval,
		m.assertionConfirmingInterval,
		m.stateManager,
		m.assertionPostingInterval,
		m.averageTimeForBlockCreation,
		m.apiDB,
		assertionsOpts...,
	)
	if err != nil {
		return nil, err
	}
	m.assertionManager = assertionManager
	return m, nil
}

func (m *Manager) GetEdgeTracker(edgeId protocol.EdgeId) option.Option[*edgetracker.Tracker] {
	if m.IsTrackingEdge(edgeId) {
		return option.Some(m.trackedEdgeIds.Get(edgeId))
	}
	return option.None[*edgetracker.Tracker]()
}

// IsTrackingEdge returns true if we are currently tracking a specified edge id as an edge tracker goroutine.
func (m *Manager) IsTrackingEdge(edgeId protocol.EdgeId) bool {
	return m.trackedEdgeIds.Has(edgeId)
}

func (m *Manager) Database() db.Database {
	return m.apiDB
}

func (m *Manager) ChallengeManagerAddress() common.Address {
	return m.chalManagerAddr
}

func (m *Manager) BlockTimes() time.Duration {
	return m.averageTimeForBlockCreation
}

// MarkTrackedEdge marks an edge id as being tracked by our challenge manager.
func (m *Manager) MarkTrackedEdge(edgeId protocol.EdgeId, tracker *edgetracker.Tracker) {
	m.trackedEdgeIds.Put(edgeId, tracker)
}

func (m *Manager) RemovedTrackedEdge(edgeId protocol.EdgeId) {
	m.trackedEdgeIds.Delete(edgeId)
}

// Mode returns the mode of the challenge manager.
func (m *Manager) Mode() types.Mode {
	return m.mode
}

// IsChallengedAssertion checks if an assertion with a given hash has a challenge.
func (m *Manager) IsClaimedByChallenge(assertionHash protocol.AssertionHash) bool {
	return m.claimedAssertionsInChallenge.Has(assertionHash)
}

// MaxDelaySeconds returns the maximum number of seconds that the challenge manager will wait open a challenge.
func (m *Manager) MaxDelaySeconds() int {
	return m.maxDelaySeconds
}

// TrackEdge spawns an edge tracker for an edge if it is not currently being tracked.
func (m *Manager) TrackEdge(ctx context.Context, edge protocol.SpecEdge) error {
	if m.trackedEdgeIds.Has(edge.Id()) {
		return nil
	}
	trk, err := m.getTrackerForEdge(ctx, edge)
	if err != nil {
		return err
	}
	m.LaunchThread(trk.Spawn)
	return nil
}

// Gets an edge tracker for an edge by retrieving its associated assertion creation info.
func (m *Manager) getTrackerForEdge(ctx context.Context, edge protocol.SpecEdge) (*edgetracker.Tracker, error) {
	// Retry until you get the previous assertion Hash.
	assertionHash, err := retry.UntilSucceeds(ctx, func() (protocol.AssertionHash, error) {
		return edge.AssertionHash(ctx)
	})
	if err != nil {
		return nil, err
	}
	blockChallengeRootEdge, err := m.watcher.HonestBlockChallengeRootEdge(ctx, assertionHash)
	if err != nil {
		return nil, err
	}
	if blockChallengeRootEdge.ClaimId().IsNone() {
		return nil, fmt.Errorf(
			"block challenge root edge %#x did not have a claim id for challenged assertion %#x",
			blockChallengeRootEdge.Id(),
			assertionHash,
		)
	}
	claimedAssertionId := blockChallengeRootEdge.ClaimId().Unwrap()

	// Smart caching to avoid querying the same assertion number and creation info multiple times.
	// Edges in the same challenge should have the same creation info.
	cachedHeightAndInboxMsgCount, ok := m.batchIndexForAssertionCache.TryGet(protocol.AssertionHash{Hash: common.Hash(claimedAssertionId)})
	var edgeTrackerAssertionInfo edgetracker.AssociatedAssertionMetadata
	if !ok {
		assertionCreationInfo, creationErr := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
			return m.chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: common.Hash(claimedAssertionId)})
		})
		if creationErr != nil {
			return nil, creationErr
		}
		prevCreationInfo, prevCreationErr := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
			return m.chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: assertionCreationInfo.ParentAssertionHash})
		})
		if prevCreationErr != nil {
			return nil, prevCreationErr
		}
		fromBatch := protocol.GoGlobalStateFromSolidity(assertionCreationInfo.BeforeState.GlobalState).Batch
		toBatch := protocol.GoGlobalStateFromSolidity(assertionCreationInfo.AfterState.GlobalState).Batch
		edgeTrackerAssertionInfo = edgetracker.AssociatedAssertionMetadata{
			FromBatch:            l2stateprovider.Batch(fromBatch),
			ToBatch:              l2stateprovider.Batch(toBatch),
			WasmModuleRoot:       prevCreationInfo.WasmModuleRoot,
			ClaimedAssertionHash: common.Hash(claimedAssertionId),
		}
		m.batchIndexForAssertionCache.Put(protocol.AssertionHash{Hash: common.Hash(claimedAssertionId)}, edgeTrackerAssertionInfo)
	} else {
		edgeTrackerAssertionInfo = cachedHeightAndInboxMsgCount
	}
	return retry.UntilSucceeds(ctx, func() (*edgetracker.Tracker, error) {
		return edgetracker.New(
			ctx,
			edge,
			m.chain,
			m.stateManager,
			m.watcher,
			m,
			&edgeTrackerAssertionInfo,
			edgetracker.WithTimeReference(m.timeRef),
			edgetracker.WithValidatorName(m.name),
		)
	})
}
func (m *Manager) Watcher() *watcher.Watcher {
	return m.watcher
}

func (m *Manager) ChallengeManager() *challengeV2gen.EdgeChallengeManagerFilterer {
	return m.chalManager
}

func (m *Manager) NewBlockSubscriber() *events.Producer[*gethtypes.Header] {
	return m.newBlockNotifier
}

func (m *Manager) Start(ctx context.Context) {
	m.StopWaiter.Start(ctx, m)
	log.Info("Started challenge manager",
		"validatorAddress", m.address.Hex(),
	)

	// Start the assertion manager.
	m.LaunchThread(m.assertionManager.Start)

	// Watcher tower and resolve modes don't monitor challenges.
	if m.mode == types.WatchTowerMode || m.mode == types.ResolveMode {
		return
	}

	// Start watching for parent chain block events in the background.
	m.LaunchThread(m.listenForBlockEvents)

	// Start watching for ongoing chain events in the background.
	m.LaunchThread(m.watcher.Start)

	if m.api != nil {
		m.LaunchThread(func(ctx context.Context) {
			if err := m.api.Start(ctx); err != nil {
				log.Error("Could not start API server",
					"address", m.apiAddr,
					"err", err,
				)
			}
		})
	}
}

func (m *Manager) StopAndWait() {
	m.StopWaiter.StopAndWait()
	m.assertionManager.StopAndWait()
	m.watcher.StopAndWait()
	m.api.StopAndWait()
}

func (m *Manager) listenForBlockEvents(ctx context.Context) {
	// If the chain watcher has not yet scraped and caught up all BOLD
	// events up to the latest head, then we fire "block notification" events
	// every second. This will help the tracked edges act fast if we are
	// just starting up the validator or catching up to a challenge.
	m.fastTickWhileCatchingUp(ctx)

	// Then, once the watcher has reached the latest head, we
	// fire off a block notifications events normally.
	if m.headBlockSubscriptions {
		m.tickOnHeadBlockSubscriptions(ctx)
	} else {
		m.tickAtInterval(ctx)
	}
}

func (m *Manager) tickOnHeadBlockSubscriptions(ctx context.Context) {
	ch := make(chan *gethtypes.Header, 100)
	sub, err := m.chain.Backend().SubscribeNewHead(ctx, ch)
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()
	numBlocksReceived := uint64(0)
	for {
		select {
		case header := <-ch:
			numBlocksReceived += 1
			// Only broadcast every N blocks received. This is important for Orbit chains
			// that have parent chains with very fast block times, such as Arbitrum One, as broadcasting
			// every 250ms would otherwise be too frequent.
			if numBlocksReceived%m.notifyOnNumberOfBlocks == 0 {
				m.newBlockNotifier.Broadcast(ctx, header)
			}
		case <-sub.Err():
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) tickAtInterval(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 12)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			m.newBlockNotifier.Broadcast(ctx, nil)
		}
	}
}

func (m *Manager) fastTickWhileCatchingUp(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			m.newBlockNotifier.Broadcast(ctx, nil)
			if m.watcher.IsSynced() {
				return
			}
		}
	}
}
