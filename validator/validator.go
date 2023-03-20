package validator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const defaultCreateLeafInterval = time.Second * 5

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
	chalManager                            *challengeV2gen.ChallengeManagerImplFilterer
	backend                                bind.ContractBackend
	stateManager                           statemanager.Manager
	address                                common.Address
	name                                   string
	knownValidatorNames                    map[common.Address]string
	createdAssertions                      map[common.Hash]protocol.Assertion
	assertionsLock                         sync.RWMutex
	sequenceNumbersByParentStateCommitment map[common.Hash][]protocol.AssertionSequenceNumber
	assertions                             map[protocol.AssertionSequenceNumber]protocol.Assertion
	leavesLock                             sync.RWMutex
	createLeafInterval                     time.Duration
	chaosMonkeyProbability                 float64
	disableLeafCreation                    bool
	timeRef                                util.TimeReference
	challengeVertexWakeInterval            time.Duration
}

// WithChaosMonkeyProbability adds a probability a validator will take
// irrational or chaotic actions during challenges.
func WithChaosMonkeyProbability(p float64) Opt {
	return func(val *Validator) {
		val.chaosMonkeyProbability = p
	}
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

// WithKnownValidators provides a map of known validator names by address for
// cleaner and more understandable logging.
func WithKnownValidators(vals map[common.Address]string) Opt {
	return func(val *Validator) {
		val.knownValidatorNames = vals
	}
}

// WithCreateLeafEvery sets a parameter that tells the validator when to initiate leaf creation.
func WithCreateLeafEvery(d time.Duration) Opt {
	return func(val *Validator) {
		val.createLeafInterval = d
	}
}

// WithTimeReference adds a time reference interface to the validator.
func WithTimeReference(ref util.TimeReference) Opt {
	return func(val *Validator) {
		val.timeRef = ref
	}
}

// WithChallengeVertexWakeInterval specifies how often each challenge vertex goroutine will
// act on its responsibilites.
func WithChallengeVertexWakeInterval(d time.Duration) Opt {
	return func(val *Validator) {
		val.challengeVertexWakeInterval = d
	}
}

// WithDisableLeafCreation disables scheduled, background submission of assertions to the protocol in the validator.
// Useful for testing.
func WithDisableLeafCreation() Opt {
	return func(val *Validator) {
		val.disableLeafCreation = true
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
		createLeafInterval:                     defaultCreateLeafInterval,
		createdAssertions:                      make(map[common.Hash]protocol.Assertion),
		sequenceNumbersByParentStateCommitment: make(map[common.Hash][]protocol.AssertionSequenceNumber),
		assertions:                             make(map[protocol.AssertionSequenceNumber]protocol.Assertion),
		timeRef:                                util.NewRealTimeReference(),
		rollupAddr:                             rollupAddr,
		challengeVertexWakeInterval:            time.Millisecond * 100,
	}
	for _, o := range opts {
		o(v)
	}
	var chalManagerAddr common.Address
	var genesis protocol.Assertion
	if err := v.chain.Call(func(tx protocol.ActiveTx) error {
		genesisAssertion, err := v.chain.AssertionBySequenceNum(ctx, tx, 0)
		if err != nil {
			return err
		}
		chalManager, err := v.chain.CurrentChallengeManager(ctx, tx)
		if err != nil {
			return err
		}
		chalManagerAddr = chalManager.Address()
		genesis = genesisAssertion
		return nil
	}); err != nil {
		return nil, err
	}
	rollup, err := rollupgen.NewRollupCore(rollupAddr, backend)
	if err != nil {
		return nil, err
	}
	rollupFilterer, err := rollupgen.NewRollupCoreFilterer(rollupAddr, backend)
	if err != nil {
		return nil, err
	}
	chalManager, err := challengeV2gen.NewChallengeManagerImplFilterer(chalManagerAddr, backend)
	if err != nil {
		return nil, err
	}
	v.rollup = rollup
	v.rollupFilterer = rollupFilterer
	v.assertions[0] = genesis
	v.chalManagerAddr = chalManagerAddr
	v.chalManager = chalManager
	return v, nil
}

func (v *Validator) Start(ctx context.Context, tx protocol.ActiveTx) {
	go v.handleRollupEvents(ctx, tx)
	if !v.disableLeafCreation {
		go v.prepareLeafCreationPeriodically(ctx)
	}
	log.WithField(
		"address",
		v.address.Hex(),
	).Info("Started validator client")
}

func (v *Validator) prepareLeafCreationPeriodically(ctx context.Context) {
	ticker := time.NewTicker(v.createLeafInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			leaf, err := v.SubmitLeafCreation(ctx)
			if err != nil {
				log.WithError(err).Error("Could not submit leaf to protocol")
				continue
			}
			go v.confirmLeafAfterChallengePeriod(ctx, leaf)
		case <-ctx.Done():
			return
		}
	}
}

// TODO: Include leaf creation validity conditions which are more complex than this.
// For example, a validator must include messages from the inbox that were not included
// by the last validator in the last leaf's creation.
func (v *Validator) SubmitLeafCreation(ctx context.Context) (protocol.Assertion, error) {
	// Ensure that we only build on a valid parent from this validator's perspective.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	parentAssertionSeq, err := v.findLatestValidAssertion(ctx)
	if err != nil {
		return nil, err
	}
	var parentAssertion protocol.Assertion
	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		parentAssertion, err = v.chain.AssertionBySequenceNum(ctx, tx, parentAssertionSeq)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
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
	var leaf protocol.Assertion
	err = v.chain.Tx(func(tx protocol.ActiveTx) error {
		leaf, err = v.chain.CreateAssertion(
			ctx,
			tx,
			assertionToCreate.Height,
			parentAssertionSeq,
			assertionToCreate.PreState,
			assertionToCreate.PostState,
			assertionToCreate.InboxMaxCount,
		)
		if err != nil {
			return err
		}
		return nil
	})
	switch {
	case errors.Is(err, solimpl.ErrAlreadyExists):
		return nil, errors.Wrap(err, "assertion already exists, unable to create new leaf")
	case err != nil:
		return nil, err
	}
	parentAssertionStateHash, err := parentAssertion.StateHash()
	if err != nil {
		return nil, err
	}
	leafStateHash, err := leaf.StateHash()
	if err != nil {
		return nil, err
	}
	leafHeight, err := leaf.Height()
	if err != nil {
		return nil, err
	}
	logFields := logrus.Fields{
		"name":               v.name,
		"parentHeight":       fmt.Sprintf("%+v", parentAssertionHeight),
		"parentStateHash":    fmt.Sprintf("%#x", parentAssertionStateHash),
		"assertionHeight":    leafHeight,
		"assertionStateHash": fmt.Sprintf("%#x", leafStateHash),
	}
	log.WithFields(logFields).Info("Submitted assertion")

	// Keep track of the created assertion locally.
	// TODO: Get the event from the chain instead, by using logs from the receipt.
	v.assertionsLock.Lock()
	// TODO: Store a more minimal struct, with only what we need.
	v.assertions[leaf.SeqNum()] = leaf
	v.sequenceNumbersByParentStateCommitment[parentAssertionStateHash] = append(
		v.sequenceNumbersByParentStateCommitment[parentAssertionStateHash],
		leaf.SeqNum(),
	)
	v.assertionsLock.Unlock()

	v.leavesLock.Lock()
	v.createdAssertions[leafStateHash] = leaf
	v.leavesLock.Unlock()
	return leaf, nil
}

// Finds the latest valid assertion sequence num a validator should build their new leaves upon. This walks
// down from the number of assertions in the protocol down until it finds
// an assertion that we have a state commitment for.
func (v *Validator) findLatestValidAssertion(ctx context.Context) (protocol.AssertionSequenceNumber, error) {
	var numAssertions uint64
	var latestConfirmed protocol.AssertionSequenceNumber
	var err error
	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		numAssertions, err = v.chain.NumAssertions(ctx, tx)
		if err != nil {
			return err
		}
		latestConfirmedFetched, err2 := v.chain.LatestConfirmed(ctx, tx)
		if err2 != nil {
			return err2
		}
		latestConfirmed = latestConfirmedFetched.SeqNum()
		return nil
	}); err != nil {
		return 0, err
	}
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

// For a leaf created by a validator, we confirm the leaf has no rival after the challenge deadline has passed.
// This function is meant to be ran as a goroutine for each leaf created by the validator.
func (v *Validator) confirmLeafAfterChallengePeriod(ctx context.Context, leaf protocol.Assertion) {
	var chalPeriod time.Duration
	if err := v.chain.Call(func(tx protocol.ActiveTx) error {
		manager, err := v.chain.CurrentChallengeManager(ctx, tx)
		if err != nil {
			return err
		}
		challengePeriodLength, err2 := manager.ChallengePeriodSeconds(ctx, tx)
		if err2 != nil {
			return err2
		}
		chalPeriod = challengePeriodLength
		return nil
	}); err != nil {
		panic(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(chalPeriod))
	defer cancel()

	// TODO: Handle validator process dying here.
	<-ctx.Done()
	leafHeight, err := leaf.Height()
	if err != nil {
		panic(err)
	}
	logFields := logrus.Fields{
		"height":      leafHeight,
		"sequenceNum": leaf.SeqNum(),
	}
	if err := v.chain.Tx(func(tx protocol.ActiveTx) error {
		// TODO: Add fields.
		return v.chain.Confirm(ctx, tx, common.Hash{}, common.Hash{})
	}); err != nil {
		log.WithError(err).WithFields(logFields).Warn("Could not confirm that created leaf had no rival")
		return
	}
	log.WithFields(logFields).Info("Confirmed leaf passed challenge period successfully on-chain")
}

// Processes new leaf creation events from the protocol that were not initiated by self.
func (v *Validator) onLeafCreated(
	ctx context.Context,
	tx protocol.ActiveTx,
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
	var prev protocol.Assertion
	if err := v.chain.Call(func(tx protocol.ActiveTx) error {
		assertionPrevSeqNum, err := assertion.PrevSeqNum()
		if err != nil {
			return err
		}
		prevAssertion, err := v.chain.AssertionBySequenceNum(ctx, tx, assertionPrevSeqNum)
		if err != nil {
			return err
		}
		prev = prevAssertion
		return nil
	}); err != nil {
		return err
	}

	v.assertionsLock.Lock()
	key, err := prev.StateHash()
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

	return v.challengeAssertion(ctx, tx, assertion)
}

func isFromSelf(self, staker common.Address) bool {
	return self == staker
}
