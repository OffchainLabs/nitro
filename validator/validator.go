package validator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

type Opt = func(val *Validator)

// Validator defines a validator client instances in the assertion protocol, which will be
// an active participant in interacting with the on-chain contracts.
type Validator struct {
	chain                                  protocol.Protocol
	chalManagerAddr                        common.Address
	rollupAddr                             common.Address
	rollup                                 *rollupgen.RollupCore
	rollupFilterer                         *rollupgen.RollupCoreFilterer
	chalManager                            *challengeV2gen.EdgeChallengeManagerFilterer
	backend                                bind.ContractBackend
	stateManager                           statemanager.Manager
	address                                common.Address
	name                                   string
	createdAssertions                      map[common.Hash]protocol.Assertion
	assertionsLock                         sync.RWMutex
	sequenceNumbersByParentStateCommitment map[common.Hash][]protocol.AssertionSequenceNumber
	assertions                             map[protocol.AssertionSequenceNumber]protocol.Assertion
	postAssertionsInterval                 time.Duration
	timeRef                                util.TimeReference
	edgeTrackerWakeInterval                time.Duration
	newAssertionCheckInterval              time.Duration
}

// WithName is a human-readable identifier for this validator client for logging purposes.
func WithName(name string) Opt {
	return func(val *Validator) {
		val.name = name
	}
}

// WithAddress gives a staker address to the validator.
func WithAddress(addr common.Address) Opt {
	return func(val *Validator) {
		val.address = addr
	}
}

// WithTimeReference adds a time reference interface to the validator.
func WithTimeReference(ref util.TimeReference) Opt {
	return func(val *Validator) {
		val.timeRef = ref
	}
}

// WithPostAssertionsInterval specifies how often the validator should try to post assertions.
func WithPostAssertionsInterval(d time.Duration) Opt {
	return func(val *Validator) {
		val.postAssertionsInterval = d
	}
}

// WithEdgeTrackerWakeInterval specifies how often each edge tracker goroutine will
// act on its responsibilities.
func WithEdgeTrackerWakeInterval(d time.Duration) Opt {
	return func(val *Validator) {
		val.edgeTrackerWakeInterval = d
	}
}

// WithNewAssertionCheckInterval specifies how often handle assertions goroutine will
// act on its responsibilities.
func WithNewAssertionCheckInterval(d time.Duration) Opt {
	return func(val *Validator) {
		val.newAssertionCheckInterval = d
	}
}

// New sets up a validator client instances provided a protocol, state manager,
// and additional options.
func New(
	ctx context.Context,
	chain protocol.Protocol,
	backend bind.ContractBackend,
	stateManager statemanager.Manager,
	rollupAddr common.Address,
	opts ...Opt,
) (*Validator, error) {
	v := &Validator{
		backend:                                backend,
		chain:                                  chain,
		stateManager:                           stateManager,
		address:                                common.Address{},
		postAssertionsInterval:                 time.Second * 5,
		createdAssertions:                      make(map[common.Hash]protocol.Assertion),
		sequenceNumbersByParentStateCommitment: make(map[common.Hash][]protocol.AssertionSequenceNumber),
		assertions:                             make(map[protocol.AssertionSequenceNumber]protocol.Assertion),
		timeRef:                                util.NewRealTimeReference(),
		rollupAddr:                             rollupAddr,
		edgeTrackerWakeInterval:                time.Millisecond * 100,
		newAssertionCheckInterval:              time.Second,
	}
	for _, o := range opts {
		o(v)
	}
	genesisAssertion, err := v.chain.AssertionBySequenceNum(ctx, 1)
	if err != nil {
		return nil, err
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
	v.assertions[0] = genesisAssertion
	v.chalManagerAddr = chalManagerAddr
	v.chalManager = chalManagerFilterer
	return v, nil
}

func (v *Validator) Start(ctx context.Context) {
	go v.pollForAssertions(ctx)
	go v.postAssertionsPeriodically(ctx)
	log.WithField(
		"address",
		v.address.Hex(),
	).Info("Started validator client")
}

func (v *Validator) postAssertionsPeriodically(ctx context.Context) {
	if _, err := v.postLatestAssertion(ctx); err != nil {
		log.WithError(err).Error("Could not submit latest assertion to L1")
	}
	ticker := time.NewTicker(v.postAssertionsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := v.postLatestAssertion(ctx); err != nil {
				log.WithError(err).Error("Could not submit latest assertion to L1")
			}
		case <-ctx.Done():
			return
		}
	}
}

// Posts the latest claim of the Node's L2 state as an assertion to the L1 protocol smart contracts.
// TODO: Include leaf creation validity conditions which are more complex than this.
// For example, a validator must include messages from the inbox that were not included
// by the last validator in the last leaf's creation.
func (v *Validator) postLatestAssertion(ctx context.Context) (protocol.Assertion, error) {
	// Ensure that we only build on a valid parent from this validator's perspective.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	parentAssertionSeq, err := v.findLatestValidAssertion(ctx)
	if err != nil {
		return nil, err
	}
	parentAssertion, err := v.chain.AssertionBySequenceNum(ctx, parentAssertionSeq)
	if err != nil {
		return nil, err
	}
	parentAssertionHeight, err := parentAssertion.Height()
	if err != nil {
		return nil, err
	}
	assertionToCreate, err := v.stateManager.LatestAssertionCreationData(ctx, parentAssertionHeight)
	if err != nil {
		return nil, err
	}
	assertion, err := v.chain.CreateAssertion(
		ctx,
		assertionToCreate.Height,
		parentAssertionSeq,
		assertionToCreate.PreState,
		assertionToCreate.PostState,
		assertionToCreate.InboxMaxCount,
	)
	switch {
	case errors.Is(err, solimpl.ErrAlreadyExists):
		return nil, errors.Wrap(err, "assertion already exists, was unable to post")
	case err != nil:
		return nil, err
	}
	parentAssertionStateHash, err := parentAssertion.StateHash()
	if err != nil {
		return nil, err
	}
	assertionState, err := assertion.StateHash()
	if err != nil {
		return nil, err
	}
	assertionHeight, err := assertion.Height()
	if err != nil {
		return nil, err
	}
	logFields := logrus.Fields{
		"name":               v.name,
		"parentHeight":       parentAssertionHeight,
		"parentStateHash":    util.Trunc(parentAssertionStateHash.Bytes()),
		"assertionHeight":    assertionHeight,
		"assertionStateHash": util.Trunc(assertionState.Bytes()),
	}
	log.WithFields(logFields).Info("Submitted latest L2 state claim as an assertion to L1")

	// Keep track of the created assertion locally.
	v.assertionsLock.Lock()
	v.assertions[assertion.SeqNum()] = assertion
	v.sequenceNumbersByParentStateCommitment[parentAssertionStateHash] = append(
		v.sequenceNumbersByParentStateCommitment[parentAssertionStateHash],
		assertion.SeqNum(),
	)
	v.assertionsLock.Unlock()
	return assertion, nil
}

// Finds the latest valid assertion sequence num a validator should build their new leaves upon. This walks
// down from the number of assertions in the protocol down until it finds
// an assertion that we have a state commitment for.
func (v *Validator) findLatestValidAssertion(ctx context.Context) (protocol.AssertionSequenceNumber, error) {
	numAssertions, err := v.chain.NumAssertions(ctx)
	if err != nil {
		return 0, err
	}
	latestConfirmedFetched, err := v.chain.LatestConfirmed(ctx)
	if err != nil {
		return 0, err
	}
	latestConfirmed := latestConfirmedFetched.SeqNum()
	v.assertionsLock.RLock()
	defer v.assertionsLock.RUnlock()
	for s := protocol.AssertionSequenceNumber(numAssertions); s > latestConfirmed; s-- {
		a, ok := v.assertions[s]
		if !ok {
			continue
		}
		height, err := a.Height()
		if err != nil {
			return 0, err
		}
		stateHash, err := a.StateHash()
		if err != nil {
			return 0, err
		}
		if v.stateManager.HasStateCommitment(ctx, util.StateCommitment{
			Height:    height,
			StateRoot: stateHash,
		}) {
			return a.SeqNum(), nil
		}
	}
	return latestConfirmed, nil
}

// Processes new leaf creation events from the protocol that were not initiated by self.
func (v *Validator) onLeafCreated(
	ctx context.Context,
	assertion protocol.Assertion,
) error {
	assertionStateHash, err := assertion.StateHash()
	if err != nil {
		return err
	}
	assertionHeight, err := assertion.Height()
	if err != nil {
		return err
	}
	log.WithFields(logrus.Fields{
		"name":      v.name,
		"stateHash": fmt.Sprintf("%#x", assertionStateHash),
		"height":    assertionHeight,
	}).Info("New assertion appended to protocol")
	// Detect if there is a fork, then decide if we want to challenge.
	// We check if the parent assertion has > 1 child.
	v.assertionsLock.Lock()
	// Keep track of the created assertion locally.
	v.assertions[assertion.SeqNum()] = assertion
	v.assertionsLock.Unlock()

	// Keep track of assertions by parent state root to more easily detect forks.
	assertionPrevSeqNum, err := assertion.PrevSeqNum()
	if err != nil {
		return err
	}
	prevAssertion, err := v.chain.AssertionBySequenceNum(ctx, assertionPrevSeqNum)
	if err != nil {
		return err
	}

	v.assertionsLock.Lock()
	key, err := prevAssertion.StateHash()
	if err != nil {
		return err
	}
	v.sequenceNumbersByParentStateCommitment[key] = append(
		v.sequenceNumbersByParentStateCommitment[key],
		assertion.SeqNum(),
	)
	hasForked := len(v.sequenceNumbersByParentStateCommitment[key]) > 1
	v.assertionsLock.Unlock()

	// If this leaf's creation has not triggered fork, we have nothing else to do.
	if !hasForked {
		log.Info("No fork detected in assertion tree upon leaf creation")
		return nil
	}

	return v.challengeAssertion(ctx, assertion)
}
