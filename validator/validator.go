package validator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	statemanager "github.com/OffchainLabs/new-rollup-exploration/state-manager"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

type Opt = func(val *Validator)

type Validator struct {
	protocol                          protocol.OnChainProtocol
	stateManager                      statemanager.Manager
	assertionEvents                   chan protocol.AssertionChainEvent
	l2StateUpdateEvents               chan *statemanager.L2StateEvent
	address                           common.Address
	name                              string
	knownValidatorNames               map[common.Address]string
	createdLeaves                     map[common.Hash]*protocol.Assertion
	assertionsLock                    sync.RWMutex
	assertionsByParentStateCommitment map[common.Hash][]*protocol.Assertion
	assertions                        map[uint64]*protocol.Assertion
	leavesLock                        sync.RWMutex
	createLeafInterval                time.Duration
	chaosMonkeyProbability            float64
}

func WithChaosMonkeyProbability(p float64) Opt {
	return func(val *Validator) {
		val.chaosMonkeyProbability = p
	}
}

func WithName(name string) Opt {
	return func(val *Validator) {
		val.name = name
	}
}

func WithAddress(addr common.Address) Opt {
	return func(val *Validator) {
		val.address = addr
	}
}

func WithKnownValidators(vals map[common.Address]string) Opt {
	return func(val *Validator) {
		val.knownValidatorNames = vals
	}
}

func WithCreateLeafEvery(d time.Duration) Opt {
	return func(val *Validator) {
		val.createLeafInterval = d
	}
}

func New(
	ctx context.Context,
	onChainProtocol protocol.OnChainProtocol,
	stateManager statemanager.Manager,
	opts ...Opt,
) (*Validator, error) {
	v := &Validator{
		protocol:                          onChainProtocol,
		stateManager:                      stateManager,
		address:                           common.Address{},
		createLeafInterval:                5 * time.Second,
		assertionEvents:                   make(chan protocol.AssertionChainEvent, 1),
		l2StateUpdateEvents:               make(chan *statemanager.L2StateEvent, 1),
		createdLeaves:                     make(map[common.Hash]*protocol.Assertion),
		assertionsByParentStateCommitment: make(map[common.Hash][]*protocol.Assertion),
		assertions:                        make(map[uint64]*protocol.Assertion),
	}
	for _, o := range opts {
		o(v)
	}
	v.protocol.SubscribeChainEvents(ctx, v.assertionEvents)
	v.stateManager.SubscribeStateEvents(ctx, v.l2StateUpdateEvents)
	return v, nil
}

func (v *Validator) Start(ctx context.Context) {
	go v.listenForAssertionEvents(ctx)
	go v.prepareLeafCreationPeriodically(ctx)
}

// TODO: Simulate posting leaf events with some jitter delay, validators will have
// latency in posting created leaves to the protocol.
func (v *Validator) prepareLeafCreationPeriodically(ctx context.Context) {
	ticker := time.NewTicker(v.createLeafInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			randDuration := rand.Int31n(2000) // 2000 ms simulating latency in submitting leaf creation.
			time.Sleep(time.Millisecond * time.Duration(randDuration))
			leaf, err := v.submitLeafCreation(ctx)
			if err != nil {
				log.WithError(err).Error("Could not submit leaf to protocol")
				continue
			}
			v.leavesLock.Lock()
			v.createdLeaves[leaf.StateCommitment.StateRoot] = leaf
			v.leavesLock.Unlock()
			go v.confirmLeafAfterChallengePeriod(leaf)
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
				// TODO: Ignore all events from self, not just CreateLeafEvent.
				if v.isFromSelf(ev) {
					return
				}
				go func() {
					if err := v.processLeafCreation(ctx, ev.SeqNum, ev.StateCommitment); err != nil {
						log.WithError(err).Error("Could not process leaf creation event")
					}
				}()
			case *protocol.StartChallengeEvent:
				go func() {
					if err := v.processChallengeStart(ctx, ev); err != nil {
						log.WithError(err).Error("Could not process challenge start event")
					}
				}()
			case *protocol.ConfirmEvent:
				log.WithField(
					"sequenceNum", ev.SeqNum,
				).Info("Leaf with sequence number confirmed on-chain")
			default:
				log.WithField("ev", fmt.Sprintf("%+v", ev)).Error("Not a recognized chain event")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (v *Validator) submitLeafCreation(ctx context.Context) (*protocol.Assertion, error) {
	// Ensure that we only build on a valid parent from this validator's perspective.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	parentAssertion := v.findLatestValidAssertion(ctx)
	currentCommit, err := v.stateManager.LatestStateCommitment(ctx)
	if err != nil {
		return nil, err
	}
	stateCommit := protocol.StateCommitment{
		Height:    currentCommit.Height,
		StateRoot: currentCommit.Merkle,
	}
	leaf, err := v.protocol.CreateLeaf(parentAssertion, stateCommit, v.address)
	switch {
	case errors.Is(err, protocol.ErrVertexAlreadyExists):
		return nil, errors.Wrap(err, "vertex already exists, unable to create new leaf")
	case errors.Is(err, protocol.ErrInvalid):
		return nil, errors.Wrap(err, "not allowed to create new leaf")
	case err != nil:
		return nil, err
	}
	logFields := logrus.Fields{
		"name":                       v.name,
		"latestValidParentHeight":    fmt.Sprintf("%+v", parentAssertion.StateCommitment.Height),
		"latestValidParentStateRoot": fmt.Sprintf("%#x", parentAssertion.StateCommitment.StateRoot),
		"leafHeight":                 currentCommit.Height,
		"leafCommitmentMerkle":       fmt.Sprintf("%#x", currentCommit.Merkle),
	}
	log.WithFields(logFields).Info("Submitted leaf creation")
	return leaf, nil
}

// Finds the latest valid assertion a validator should build their new leaves upon. This starts from
// the latest confirmed assertion and makes it down the tree to the latest assertion that has a state
// commitment matching in the validator's database.
func (v *Validator) findLatestValidAssertion(ctx context.Context) *protocol.Assertion {
	latestValidParent := v.protocol.LatestConfirmed()
	numAssertions := v.protocol.NumAssertions()
	v.assertionsLock.RLock()
	defer v.assertionsLock.RUnlock()
	for s := latestValidParent.SequenceNum; s < numAssertions; s++ {
		a, ok := v.assertions[s]
		if !ok {
			continue
		}
		if v.stateManager.HasStateCommitment(ctx, a.StateCommitment) {
			latestValidParent = a
		}
	}
	return latestValidParent
}

// For a leaf created by a validator, we confirm the leaf has no rival after the challenge deadline has passed.
// This function is meant to be ran as a goroutine for each leaf created by the validator.
func (v *Validator) confirmLeafAfterChallengePeriod(leaf *protocol.Assertion) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(v.protocol.ChallengePeriodLength()))
	defer cancel()
	<-ctx.Done()
	logFields := logrus.Fields{
		"height":      leaf.StateCommitment.Height,
		"sequenceNum": leaf.SequenceNum,
	}
	if err := leaf.ConfirmNoRival(); err != nil {
		log.WithError(err).WithFields(logFields).Warn("Could not confirm that created leaf had no rival")
		return
	}
	log.WithFields(logFields).Info("Confirmed leaf passed challenge period successfully on-chain")
}

func (v *Validator) processLeafCreation(ctx context.Context, seqNum uint64, stateCommit protocol.StateCommitment) error {
	log.WithFields(logrus.Fields{
		"name":      v.name,
		"stateRoot": fmt.Sprintf("%#x", stateCommit.StateRoot),
		"height":    stateCommit.Height,
	}).Info("New leaf appended to protocol")
	// Detect if there is a fork, then decide if we want to challenge.
	// We check if the parent assertion has > 1 child.
	assertion, err := v.protocol.AssertionBySequenceNumber(ctx, seqNum)
	if err != nil {
		return err
	}
	v.assertionsLock.Lock()
	// Keep track of the created assertion locally.
	v.assertions[seqNum] = assertion

	// Keep track of assertions by parent state root to more easily detect forks.
	key := common.Hash{}
	if !assertion.Prev.IsEmpty() {
		parentAssertion := assertion.Prev.OpenKnownFull()
		key = parentAssertion.StateCommitment.Hash()
	}
	v.assertionsByParentStateCommitment[key] = append(
		v.assertionsByParentStateCommitment[key],
		assertion,
	)
	hasForked := len(v.assertionsByParentStateCommitment[key]) > 1
	v.assertionsLock.Unlock()

	// If this leaf's creation has not triggered fork, we have nothing else to do.
	if !hasForked {
		log.Info("No fork detected in assertion tree upon leaf creation")
		return nil
	}
	// If there is a fork, we challenge if we disagree with its state commitment. Otherwise,
	// we will defend challenge moves that agree with our local state.
	if v.stateManager.HasStateCommitment(ctx, assertion.StateCommitment) {
		return v.defendLeaf(ctx, assertion)
	}
	return v.challengeLeaf(ctx, assertion)
}

func (v *Validator) processChallengeStart(ctx context.Context, ev *protocol.StartChallengeEvent) error {
	// Checks if the challenge has to do with a vertex we created.
	challengedAssertion, err := v.protocol.AssertionBySequenceNumber(ctx, ev.ParentSeqNum)
	if err != nil {
		return err
	}
	v.leavesLock.RLock()
	defer v.leavesLock.RUnlock()
	leaf, ok := v.createdLeaves[challengedAssertion.StateCommitment.StateRoot]
	if !ok {
		// TODO: Act on the honest vertices even if this challenge does not have to do with us by
		// keeping track of associated challenge vertices' clocks and acting if the associated
		// staker we agree with is not performing their responsibilities on time. As an honest
		// validator, we should participate in confirming valid assertions.
		return nil
	}
	challengerName := "unknown-name"
	if !leaf.Staker.IsEmpty() {
		if name, ok := v.knownValidatorNames[leaf.Staker.OpenKnownFull()]; ok {
			challengerName = name
		} else {
			challengerName = leaf.Staker.OpenKnownFull().Hex()
		}
	}
	log.WithFields(logrus.Fields{
		"name":                 v.name,
		"challenger":           challengerName,
		"challengingStateRoot": fmt.Sprintf("%#x", leaf.StateCommitment.StateRoot),
		"challengingHeight":    leaf.StateCommitment.Height,
	}).Warn("Received challenge for a created leaf!")
	return nil
}

// Prepares to defend a leaf that matches our local history and is part of a fork
// in the assertions tree. This leaf may be challenged and the local validator should
// be ready to perform proper challenge moves on the assertion if no one else is making them.
func (v *Validator) defendLeaf(ctx context.Context, as *protocol.Assertion) error {
	logFields := logrus.Fields{}
	if !as.Staker.IsEmpty() {
		if name, ok := v.knownValidatorNames[as.Staker.OpenKnownFull()]; ok {
			logFields["createdBy"] = name
		}
	}
	logFields["name"] = v.name
	logFields["height"] = as.StateCommitment.Height
	logFields["stateRoot"] = fmt.Sprintf("%#x", as.StateCommitment.StateRoot)
	log.WithFields(logFields).Info(
		"New leaf created by another validator matching local state has " +
			"forked the protocol, preparing to defend",
	)
	return nil
}

// Initiates a challenge on a created leaf.
func (v *Validator) challengeLeaf(ctx context.Context, as *protocol.Assertion) error {
	logFields := logrus.Fields{}
	logFields["name"] = v.name
	logFields["height"] = as.StateCommitment.Height
	logFields["stateRoot"] = fmt.Sprintf("%#x", as.StateCommitment.StateRoot)
	log.WithFields(logFields).Info("Initiating challenge on leaf validator disagrees with")
	return nil
}

func (v *Validator) isFromSelf(ev *protocol.CreateLeafEvent) bool {
	return v.address == ev.Staker
}
