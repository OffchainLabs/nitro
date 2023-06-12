package validator

import (
	"context"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	watcher "github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/chain-watcher"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	utilTime "github.com/OffchainLabs/challenge-protocol-v2/time"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

type Opt = func(val *Manager)

// Manager defines an offchain, challenge manager, which will be
// an active participant in interacting with the on-chain contracts.
type Manager struct {
	chain                     protocol.Protocol
	chalManagerAddr           common.Address
	rollupAddr                common.Address
	rollup                    *rollupgen.RollupCore
	rollupFilterer            *rollupgen.RollupCoreFilterer
	chalManager               *challengeV2gen.EdgeChallengeManagerFilterer
	backend                   bind.ContractBackend
	stateManager              l2stateprovider.Provider
	address                   common.Address
	name                      string
	postAssertionsInterval    time.Duration
	timeRef                   utilTime.Reference
	edgeTrackerWakeInterval   time.Duration
	newAssertionCheckInterval time.Duration
	initialSyncCompleted      chan struct{}
	chainWatcherInterval      time.Duration
	watcher                   *watcher.Watcher
}

// WithName is a human-readable identifier for this validator client for logging purposes.
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

// WithTimeReference adds a time reference interface to the validator.

// WithPostAssertionsInterval specifies how often the validator should try to post assertions.
func WithPostAssertionsInterval(d time.Duration) Opt {
	return func(val *Manager) {
		val.postAssertionsInterval = d
	}
}

// WithEdgeTrackerWakeInterval specifies how often each edge tracker goroutine will
// act on its responsibilities.
func WithEdgeTrackerWakeInterval(d time.Duration) Opt {
	return func(val *Manager) {
		val.edgeTrackerWakeInterval = d
	}
}

// WithNewAssertionCheckInterval specifies how often handle assertions goroutine will
// act on its responsibilities.
func WithNewAssertionCheckInterval(d time.Duration) Opt {
	return func(val *Manager) {
		val.newAssertionCheckInterval = d
	}
}

// WithChainWatcherInterval specifies how often the chain watcher will scan for edge events.

// New sets up a validator client instances provided a protocol, state manager,
// and additional options.
func New(
	ctx context.Context,
	chain protocol.Protocol,
	backend bind.ContractBackend,
	stateManager l2stateprovider.Provider,
	rollupAddr common.Address,
	opts ...Opt,
) (*Manager, error) {
	v := &Manager{
		backend:                   backend,
		chain:                     chain,
		stateManager:              stateManager,
		address:                   common.Address{},
		timeRef:                   utilTime.NewRealTimeReference(),
		rollupAddr:                rollupAddr,
		edgeTrackerWakeInterval:   time.Millisecond * 100,
		newAssertionCheckInterval: time.Second,
		postAssertionsInterval:    time.Second * 5,
		chainWatcherInterval:      time.Second * 5,
		initialSyncCompleted:      make(chan struct{}),
	}
	for _, o := range opts {
		o(v)
	}
	chalManager, err := v.chain.SpecChallengeManager(ctx)
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
	v.rollup = rollup
	v.rollupFilterer = rollupFilterer
	v.chalManagerAddr = chalManagerAddr
	v.chalManager = chalManagerFilterer
	v.watcher = watcher.New(v.chain, v.stateManager, backend, v.chainWatcherInterval, v.name)
	return v, nil
}

func (v *Manager) Start(ctx context.Context) {
	log.WithField(
		"address",
		v.address.Hex(),
	).Info("Started validator client")

	// Start watching for ongoing chain events in the background.
	go v.watcher.Watch(ctx, v.initialSyncCompleted)

	// Then, block the main thread and wait until the chain event watcher has synced up with
	// all edges from the chain since the latest confirmed assertion up to the latest block number.
	if err := v.syncEdges(ctx); err != nil {
		log.WithError(err).Fatal("Could not sync with onchain edges")
	}
}

// waitForSync waits for a notificataion that initial sync of onchain edges is complete.
func (v *Manager) waitForSync(ctx context.Context) error {
	select {
	case <-v.initialSyncCompleted:
		return nil
	case <-ctx.Done():
		return errors.New("context closed, exiting goroutine")
	}
}
