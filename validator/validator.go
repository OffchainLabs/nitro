package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	utilTime "github.com/OffchainLabs/challenge-protocol-v2/util/time"
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
	chain                     protocol.Protocol
	chalManagerAddr           common.Address
	rollupAddr                common.Address
	rollup                    *rollupgen.RollupCore
	rollupFilterer            *rollupgen.RollupCoreFilterer
	chalManager               *challengeV2gen.EdgeChallengeManagerFilterer
	backend                   bind.ContractBackend
	stateManager              statemanager.Manager
	address                   common.Address
	name                      string
	postAssertionsInterval    time.Duration
	timeRef                   utilTime.Reference
	edgeTrackerWakeInterval   time.Duration
	newAssertionCheckInterval time.Duration
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
func WithTimeReference(ref utilTime.Reference) Opt {
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
		backend:                   backend,
		chain:                     chain,
		stateManager:              stateManager,
		address:                   common.Address{},
		timeRef:                   utilTime.NewRealTimeReference(),
		rollupAddr:                rollupAddr,
		edgeTrackerWakeInterval:   time.Millisecond * 100,
		newAssertionCheckInterval: time.Second,
		postAssertionsInterval:    time.Second * 5,
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
	parentAssertionCreationInfo, err := v.chain.ReadAssertionCreationInfo(ctx, parentAssertionSeq)
	if err != nil {
		return nil, err
	}
	// TODO: this should really only go up to the prevInboxMaxCount batch state
	newState, err := v.stateManager.LatestExecutionState(ctx)
	if err != nil {
		return nil, err
	}
	assertion, err := v.chain.CreateAssertion(
		ctx,
		protocol.GoExecutionStateFromSolidity(parentAssertionCreationInfo.AfterState),
		newState,
	)
	switch {
	case errors.Is(err, solimpl.ErrAlreadyExists):
		return nil, errors.Wrap(err, "assertion already exists, was unable to post")
	case err != nil:
		return nil, err
	}
	logFields := logrus.Fields{
		"name": v.name,
	}
	log.WithFields(logFields).Info("Submitted latest L2 state claim as an assertion to L1")

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
	bestAssertion := latestConfirmed
	for s := latestConfirmed + 1; s < protocol.AssertionSequenceNumber(numAssertions); s++ {
		a, err := v.chain.AssertionBySequenceNum(ctx, s)
		if err != nil {
			return 0, err
		}
		parent, err := a.PrevSeqNum()
		if err != nil {
			return 0, err
		}
		if parent != bestAssertion {
			continue
		}
		info, err := v.chain.ReadAssertionCreationInfo(ctx, s)
		if err != nil {
			return 0, err
		}
		_, hasState := v.stateManager.ExecutionStateBlockHeight(ctx, protocol.GoExecutionStateFromSolidity(info.AfterState))
		if hasState {
			bestAssertion = s
		}
	}
	return bestAssertion, nil
}

// validChildFromParent returns the assertion number of a child of the parent assertion number.
// The assertion must be valid and exists in state manager by `ExecutionStateBlockHeight` validation.
// It returns the first assertion number that is a child of the parent assertion number. This assumes there's no two children of the same parent.
// If no such assertion exists, an error gets returned.
func (v *Validator) validChildFromParent(ctx context.Context, parentAssertionNumber protocol.AssertionSequenceNumber) (protocol.AssertionSequenceNumber, error) {
	numAssertions, err := v.chain.NumAssertions(ctx)
	if err != nil {
		return 0, err
	}

	for s := parentAssertionNumber + 1; s < protocol.AssertionSequenceNumber(numAssertions); s++ {
		a, err := v.chain.AssertionBySequenceNum(ctx, s)
		if err != nil {
			return 0, err
		}
		n, err := a.PrevSeqNum()
		if err != nil {
			return 0, err
		}
		if n != parentAssertionNumber {
			continue
		}
		info, err := v.chain.ReadAssertionCreationInfo(ctx, s)
		if err != nil {
			return 0, err
		}
		_, hasState := v.stateManager.ExecutionStateBlockHeight(ctx, protocol.GoExecutionStateFromSolidity(info.AfterState))
		if hasState {
			return a.SeqNum(), nil
		}
	}
	return 0, fmt.Errorf("no valid assertion found from parent %v", parentAssertionNumber)
}

// Processes new leaf creation events from the protocol that were not initiated by self.
func (v *Validator) onLeafCreated(
	ctx context.Context,
	assertion protocol.Assertion,
) error {
	log.WithFields(logrus.Fields{
		"name": v.name,
	}).Info("New assertion appended to protocol")

	isFirstChild, err := assertion.IsFirstChild()
	if err != nil {
		return err
	}

	// If this leaf is the first child, we have nothing else to do.
	if isFirstChild {
		log.Info("No fork detected in assertion tree upon leaf creation")
		return nil
	}

	psn, err := assertion.PrevSeqNum()
	if err != nil {
		return err
	}

	return v.challengeAssertion(ctx, psn)
}
