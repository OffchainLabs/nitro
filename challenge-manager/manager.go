// Package challengemanager includes the main entrypoint for setting up a BOLD
// challenge manager instance and challenging assertions onchain.
//
// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package challengemanager

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/OffchainLabs/bold/api"
	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	watcher "github.com/OffchainLabs/bold/challenge-manager/chain-watcher"
	edgetracker "github.com/OffchainLabs/bold/challenge-manager/edge-tracker"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers/threadsafe"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	utilTime "github.com/OffchainLabs/bold/time"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	srvlog = log.New("service", "challenge-manager")
)

func init() {
	srvlog.SetHandler(log.StreamHandler(os.Stdout, log.LogfmtFormat()))
}

type Opt = func(val *Manager)

// Manager defines an offchain, challenge manager, which will be
// an active participant in interacting with the on-chain contracts.
type Manager struct {
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
	edgeTrackerWakeInterval     time.Duration
	chainWatcherInterval        time.Duration
	watcher                     *watcher.Watcher
	trackedEdgeIds              *threadsafe.Set[protocol.EdgeId]
	batchIndexForAssertionCache *threadsafe.Map[protocol.AssertionHash, edgetracker.AssociatedAssertionMetadata]
	assertionManager            *assertions.Manager
	assertionPostingInterval    time.Duration
	assertionScanningInterval   time.Duration
	assertionConfirmingInterval time.Duration
	averageTimeForBlockCreation time.Duration
	mode                        types.Mode
	maxDelaySeconds             int

	challengedAssertions *threadsafe.Set[protocol.AssertionHash]
	// API
	apiAddr string
	api     *api.Server
}

// WithName is a human-readable identifier for this challenge manager for logging purposes.
func WithName(name string) Opt {
	return func(val *Manager) {
		val.name = name
	}
}

// WithAddress gives a staker address to the validator.
func WithAddress(addr common.Address) Opt {
	return func(val *Manager) {
		val.address = addr
	}
}

// WithEdgeTrackerWakeInterval specifies how often each edge tracker goroutine will
// act on its responsibilities.
func WithEdgeTrackerWakeInterval(d time.Duration) Opt {
	return func(val *Manager) {
		val.edgeTrackerWakeInterval = d
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

// WithMode specifies the mode of the challenge manager.
func WithMode(m types.Mode) Opt {
	return func(val *Manager) {
		val.mode = m
	}
}

// WithAPIEnabled specifies whether or not to enable the API and the address to listen on.
func WithAPIEnabled(addr string) Opt {
	return func(val *Manager) {
		val.apiAddr = addr
	}
}

func WithRPCClient(client *rpc.Client) Opt {
	return func(val *Manager) {
		val.client = client
	}
}

// New sets up a challenge manager instance provided a protocol, state manager, and additional options.
func New(
	ctx context.Context,
	chain protocol.Protocol,
	backend bind.ContractBackend,
	stateManager l2stateprovider.Provider,
	rollupAddr common.Address,
	opts ...Opt,
) (*Manager, error) {

	m := &Manager{
		backend:                     backend,
		chain:                       chain,
		stateManager:                stateManager,
		address:                     common.Address{},
		timeRef:                     utilTime.NewRealTimeReference(),
		rollupAddr:                  rollupAddr,
		chainWatcherInterval:        time.Millisecond * 500,
		trackedEdgeIds:              threadsafe.NewSet[protocol.EdgeId](),
		batchIndexForAssertionCache: threadsafe.NewMap[protocol.AssertionHash, edgetracker.AssociatedAssertionMetadata](),
		assertionPostingInterval:    time.Hour,
		assertionScanningInterval:   time.Minute,
		assertionConfirmingInterval: time.Second * 10,
		averageTimeForBlockCreation: time.Millisecond * 500,
		challengedAssertions:        threadsafe.NewSet[protocol.AssertionHash](),
	}
	for _, o := range opts {
		o(m)
	}

	if m.edgeTrackerWakeInterval == 0 {
		// Generating a random integer between 0 and 60 second to wake up the edge tracker.
		// This is to avoid all edge trackers waking up at the same time across participants.
		n, err := rand.Int(rand.Reader, new(big.Int).SetUint64(60))
		if err != nil {
			return nil, err
		}
		m.edgeTrackerWakeInterval = time.Second * time.Duration(n.Uint64())
	}

	chalManager, err := m.chain.SpecChallengeManager(ctx)
	if err != nil {
		return nil, err
	}
	chalManagerAddr := chalManager.Address()

	rollup, err := rollupgen.NewRollupCore(rollupAddr, backend)
	if err != nil {
		return nil, err
	}
	rollupFilterer, err := rollupgen.NewRollupCoreFilterer(rollupAddr, backend)
	if err != nil {
		return nil, err
	}
	chalManagerFilterer, err := challengeV2gen.NewEdgeChallengeManagerFilterer(chalManagerAddr, backend)
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
	watcher, err := watcher.New(m.chain, m, m.stateManager, backend, m.chainWatcherInterval, numBigStepLevels, m.name)
	if err != nil {
		return nil, err
	}
	m.watcher = watcher
	assertionManager, err := assertions.NewManager(
		m.chain,
		m.stateManager,
		m.backend,
		m,
		m.rollupAddr,
		m.name,
		m.assertionScanningInterval,
		m.assertionConfirmingInterval,
		m.stateManager,
		m.assertionPostingInterval,
		m.averageTimeForBlockCreation,
	)
	if err != nil {
		return nil, err
	}
	m.assertionManager = assertionManager

	if m.apiAddr != "" && m.client == nil {
		return nil, errors.New("go-ethereum RPC client required to enable API service")
	}

	if m.apiAddr != "" {
		a, err := api.NewServer(&api.Config{
			Address:            m.apiAddr,
			EdgesProvider:      m.watcher,
			AssertionsProvider: m.chain,
		})
		if err != nil {
			return nil, err
		}
		m.api = a
	}

	return m, nil
}

// IsTrackingEdge returns true if we are currently tracking a specified edge id as an edge tracker goroutine.
func (m *Manager) IsTrackingEdge(edgeId protocol.EdgeId) bool {
	return m.trackedEdgeIds.Has(edgeId)
}

// MarkTrackedEdge marks an edge id as being tracked by our challenge manager.
func (m *Manager) MarkTrackedEdge(edgeId protocol.EdgeId) {
	m.trackedEdgeIds.Insert(edgeId)
}

// Mode returns the mode of the challenge manager.
func (m *Manager) Mode() types.Mode {
	return m.mode
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
	go trk.Spawn(ctx)
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
			FromBatch:      l2stateprovider.Batch(fromBatch),
			ToBatch:        l2stateprovider.Batch(toBatch),
			WasmModuleRoot: prevCreationInfo.WasmModuleRoot,
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
			edgetracker.WithActInterval(m.edgeTrackerWakeInterval),
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

func (m *Manager) Start(ctx context.Context) {
	srvlog.Info("Started challenge manager", log.Ctx{
		"validatorAddress": m.address.Hex(),
	})

	// Start the assertion manager.
	go m.assertionManager.Start(ctx)

	// Watcher tower and resolve modes don't monitor challenges.
	if m.mode == types.WatchTowerMode || m.mode == types.ResolveMode {
		return
	}

	// Start watching for ongoing chain events in the background.
	go m.watcher.Start(ctx)

	if m.api != nil {
		go func() {
			if err := m.api.Start(); err != nil {
				srvlog.Error("Failed to start API server", log.Ctx{
					"address": m.apiAddr,
					"err":     err,
				})
			}
		}()
	}
}
