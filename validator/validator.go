package validator

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

type Opt = func(val *Validator)

// StateManager defines a struct that can provide local state data and historical
// Merkle commitments to L2 state for the validator.
type StateManager interface {
	LatestHistoryCommitment(ctx context.Context) util.HistoryCommitment
	HasStateRoot(ctx context.Context, stateRoot common.Hash) bool
	StateCommitmentAtHeight(ctx context.Context, height uint64) (util.HistoryCommitment, error)
	SubscribeStateEvents(ctx context.Context, ch chan<- *L2StateEvent)
}

type L2StateEvent struct{}

type Validator struct {
	protocol                    protocol.OnChainProtocol
	stateManager                StateManager
	assertionEvents             chan protocol.AssertionChainEvent
	l2StateUpdateEvents         chan *L2StateEvent
	address                     common.Address
	name                        string
	knownValidatorNames         map[common.Address]string
	createdLeaves               map[common.Hash]*protocol.Assertion
	assertionsLock              sync.RWMutex
	assertionsByParentStateRoot map[common.Hash][]*protocol.Assertion
	leavesLock                  sync.RWMutex
	createLeafInterval          time.Duration
	chaosMonkeyProbability      float64
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
	stateManager StateManager,
	opts ...Opt,
) (*Validator, error) {
	v := &Validator{
		protocol:                    onChainProtocol,
		stateManager:                stateManager,
		address:                     common.Address{},
		createLeafInterval:          5 * time.Second,
		assertionEvents:             make(chan protocol.AssertionChainEvent, 1),
		stateUpdateEvents:           make(chan *L2StateEvent, 1),
		createdLeaves:               make(map[common.Hash]*protocol.Assertion),
		assertionsByParentStateRoot: make(map[common.Hash][]*protocol.Assertion),
	}
	for _, o := range opts {
		o(v)
	}
	v.protocol.SubscribeChainEvents(ctx, v.assertionEvents)
	v.stateManager.SubscribeStateEvents(ctx, v.stateUpdateEvents)
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
			leaf := v.submitLeafCreation(ctx)
			if leaf == nil {
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

func (v *Validator) submitLeafCreation(ctx context.Context) *protocol.Assertion {
	// Ensure that we only build on a valid parent from this validator's perspective.
	// TODO: Instead of iterating over all assertions, validator should load up all created assertions since
	// the latest confirmed one in prod and update that list as faster way of looking up valid parents.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	parentAssertion, err := v.findLatestValidAssertion(ctx)
	if err != nil {
		log.WithError(err).Error("Could not find valid parent assertion to build leaf upon")
		return nil
	}

	// TODO: Fix! State commit and history commit are not the same thing.
	currentCommit := v.stateManager.LatestHistoryCommitment(ctx)
	stateCommit := protocol.StateCommitment{
		Height:    currentCommit.Height,
		StateRoot: currentCommit.Merkle,
	}
	logFields := logrus.Fields{
		"name":                       v.name,
		"latestValidParentHeight":    fmt.Sprintf("%+v", parentAssertion.StateCommitment.Height),
		"latestValidParentStateRoot": util.FormatHash(parentAssertion.StateCommitment.StateRoot),
		"leafHeight":                 currentCommit.Height,
		"leafCommitmentMerkle":       util.FormatHash(currentCommit.Merkle),
	}
	leaf, err := v.protocol.CreateLeaf(parentAssertion, stateCommit, v.address)
	switch {
	case errors.Is(err, protocol.ErrVertexAlreadyExists):
		log.WithFields(logFields).Debug("Vertex already exists, unable to create new leaf")
		return nil
	case errors.Is(err, protocol.ErrInvalid):
		log.WithFields(logFields).Debug("Tried to create a leaf with an older commitment")
		return nil
	case err != nil:
		log.WithError(err).Error("Could not create leaf")
		return nil
	}
	log.WithFields(logFields).Info("Submitted leaf creation")
	return leaf
}

// Finds the latest valid assertion a validator should build their new leaves upon. This starts from
// the latest confirmed assertion and makes it down the tree to the latest assertion that has a state
// root matching in the validator's database.
func (v *Validator) findLatestValidAssertion(ctx context.Context) (*protocol.Assertion, error) {
	latestValidParent := v.protocol.LatestConfirmed()
	for s := latestValidParent.SequenceNum; s < v.protocol.NumAssertions(); s++ {
		a, err := v.protocol.AssertionBySequenceNumber(s)
		if err != nil {
			return nil, err
		}
		if v.stateManager.HasStateRoot(ctx, a.StateCommitment.StateRoot) {
			latestValidParent = a
		}
	}
	return latestValidParent, nil
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
	// Detect if there is a fork, then decide if we want to challenge.
	// We check if the parent assertion has > 1 child.
	assertion, err := v.protocol.AssertionBySequenceNumber(seqNum)
	if err != nil {
		return err
	}
	v.assertionsLock.Lock()
	key := common.Hash{}
	if !assertion.Prev().IsEmpty() {
		parentAssertion := assertion.Prev().OpenKnownFull()
		key = parentAssertion.StateCommitment.StateRoot
	}
	v.assertionsByParentStateRoot[key] = append(
		v.assertionsByParentStateRoot[key],
		assertion,
	)
	v.assertionsLock.Unlock()
	hasForked := len(v.assertionsByParentStateRoot[key]) > 1
	if !hasForked {
		return nil
	}
	// We attempt to challenge the parent assertion if we detect a fork.
	if v.stateManager.HasStateRoot(ctx, assertion.StateCommitment.StateRoot) {
		return v.defendLeaf(ctx, assertion)
	}
	return v.challengeLeaf(ctx, assertion)
}

func (v *Validator) processChallengeStart(ctx context.Context, ev *protocol.StartChallengeEvent) error {
	// Checks if the challenge has to do with a vertex we created.
	challengedAssertion, err := v.protocol.AssertionBySequenceNumber(ev.ParentSeqNum)
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

// TODO: Defend a leaf if it is not created by us, but is a valid leaf from our perspective.
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
	log.WithFields(logFields).Info("New leaf matches local state")
	return nil
}

// Initiates a challenge on a created leaf.
func (v *Validator) challengeLeaf(ctx context.Context, as *protocol.Assertion) error {
	return errors.New("unimplemented")
}

func (v *Validator) isFromSelf(ev *protocol.CreateLeafEvent) bool {
	return v.address == ev.Staker
}

func (v *Validator) leafMatchesLocalState(localCommitment protocol.StateCommitment, ev *protocol.CreateLeafEvent) bool {
	return localCommitment.Hash() == ev.StateCommitment.Hash()
}
