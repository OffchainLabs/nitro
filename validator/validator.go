package validator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/big"
)

const defaultCreateLeafInterval = time.Second * 5

var log = logrus.WithField("prefix", "validator")

type Opt = func(val *Validator)

// Validator defines a validator client instances in the assertion protocol, which will be
// an active participant in interacting with the on-chain contracts.
type Validator struct {
	chain                                  protocol.Protocol
	stateManager                           statemanager.Manager
	assertionEvents                        chan protocol.AssertionChainEvent
	address                                common.Address
	name                                   string
	knownValidatorNames                    map[common.Address]string
	createdLeaves                          map[protocol.AssertionHash]protocol.Assertion
	assertionsLock                         sync.RWMutex
	sequenceNumbersByParentStateCommitment map[common.Hash][]protocol.AssertionSequenceNumber
	assertions                             map[protocol.AssertionSequenceNumber]*protocol.CreateLeafEvent
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
	stateManager statemanager.Manager,
	opts ...Opt,
) (*Validator, error) {
	v := &Validator{
		chain:                                  chain,
		stateManager:                           stateManager,
		address:                                common.Address{},
		createLeafInterval:                     defaultCreateLeafInterval,
		assertionEvents:                        make(chan protocol.AssertionChainEvent, 1),
		createdLeaves:                          make(map[protocol.AssertionHash]protocol.Assertion),
		sequenceNumbersByParentStateCommitment: make(map[common.Hash][]protocol.AssertionSequenceNumber),
		assertions:                             make(map[protocol.AssertionSequenceNumber]*protocol.CreateLeafEvent),
		timeRef:                                util.NewRealTimeReference(),
		challengeVertexWakeInterval:            time.Millisecond * 100,
	}
	for _, o := range opts {
		o(v)
	}
	v.assertions[0] = &protocol.CreateLeafEvent{
		PrevSeqNum:    0,
		SeqNum:        0,
		PrevStateHash: common.Hash{},
		PrevHeight:    0,
		Height:        0,
		StateHash:     common.Hash{},
		Validator:     common.Address{},
	}
	//v.chain.SubscribeChainEvents(ctx, v.assertionEvents)
	return v, nil
}

func (v *Validator) Start(ctx context.Context) {
	go v.listenForAssertionEvents(ctx)
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

func (v *Validator) listenForAssertionEvents(ctx context.Context) {
	for {
		select {
		case genericEvent := <-v.assertionEvents:
			switch ev := genericEvent.(type) {
			case *protocol.CreateLeafEvent:
				go func() {
					if err := v.onLeafCreated(ctx, ev); err != nil {
						log.WithError(err).Error("Could not process leaf creation event")
					}
				}()
			case *protocol.StartChallengeEvent:
				go func() {
					if err := v.onChallengeStarted(ctx, ev); err != nil {
						log.WithError(err).Error("Could not process challenge start event")
					}
				}()
			case *protocol.ConfirmEvent:
				log.WithField(
					"sequenceNum", ev.SeqNum,
				).Info("Leaf with sequence number confirmed on-chain")
			case *protocol.SetBalanceEvent:
			default:
				log.WithField("ev", fmt.Sprintf("%+v", ev)).Error("Not a recognized chain event")
			}
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
	// TODO: This should return the pre, post-state and a height.
	assertionToCreate, err := v.stateManager.LatestAssertionCreationData(ctx)
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
			big.NewInt(0), // TODO: Get from inbox
		)
		if err != nil {
			return err
		}
		return nil
	})
	switch {
	case errors.Is(err, solimpl.ErrAlreadyExists):
		return nil, errors.Wrap(err, "vertex already exists, unable to create new leaf")
	case err != nil:
		return nil, err
	}
	logFields := logrus.Fields{
		"name":                         v.name,
		"latestValidParentHeight":      fmt.Sprintf("%+v", parentAssertion.Height()),
		"latestValidParentStateRoot":   fmt.Sprintf("%#x", parentAssertion.StateHash()),
		"leafHeight":                   assertionToCreate.Height,
		"leafCommitmentBlockStateHash": fmt.Sprintf("%#x", assertionToCreate.PostState.BlockStateHash()),
	}
	log.WithFields(logFields).Info("Submitted leaf creation")

	// Keep track of the created assertion locally.
	v.assertionsLock.Lock()
	v.assertions[leaf.SeqNum()] = &protocol.CreateLeafEvent{
		PrevSeqNum:    protocol.AssertionSequenceNumber(leaf.PrevSeqNum()),
		SeqNum:        protocol.AssertionSequenceNumber(leaf.SeqNum()),
		PrevHeight:    parentAssertion.Height(),
		PrevStateHash: parentAssertion.StateHash(),
		Height:        leaf.Height(),
		StateHash:     leaf.StateHash(),
		Validator:     v.address,
	}
	key := parentAssertion.StateHash()
	v.sequenceNumbersByParentStateCommitment[key] = append(
		v.sequenceNumbersByParentStateCommitment[key],
		leaf.SeqNum(),
	)
	v.assertionsLock.Unlock()

	v.leavesLock.Lock()
	v.createdLeaves[protocol.AssertionHash(leaf.StateHash())] = leaf
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
		latestConfirmedFetched, err := v.chain.LatestConfirmed(ctx, tx)
		if err != nil {
			return err
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
		if v.stateManager.HasStateCommitment(ctx, util.StateCommitment{
			Height:    a.Height,
			StateRoot: a.StateHash,
		}) {
			return a.SeqNum, nil
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
	logFields := logrus.Fields{
		"height":      leaf.Height(),
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
func (v *Validator) onLeafCreated(ctx context.Context, ev *protocol.CreateLeafEvent) error {
	if ev == nil {
		return nil
	}
	if isFromSelf(v.address, ev.Validator) {
		return nil
	}
	seqNum := ev.SeqNum

	log.WithFields(logrus.Fields{
		"name":      v.name,
		"stateRoot": fmt.Sprintf("%#x", ev.StateHash),
		"height":    ev.Height,
	}).Info("New leaf appended to protocol")
	// Detect if there is a fork, then decide if we want to challenge.
	// We check if the parent assertion has > 1 child.
	v.assertionsLock.Lock()
	// Keep track of the created assertion locally.
	v.assertions[seqNum] = ev

	// Keep track of assertions by parent state root to more easily detect forks.
	key := ev.PrevStateHash
	v.sequenceNumbersByParentStateCommitment[key] = append(
		v.sequenceNumbersByParentStateCommitment[key],
		ev.SeqNum,
	)
	hasForked := len(v.sequenceNumbersByParentStateCommitment[key]) > 1
	v.assertionsLock.Unlock()

	// If this leaf's creation has not triggered fork, we have nothing else to do.
	if !hasForked {
		log.Info("No fork detected in assertion tree upon leaf creation")
		return nil
	}

	return v.challengeAssertion(ctx, ev)
}

func isFromSelf(self, staker common.Address) bool {
	return self == staker
}
