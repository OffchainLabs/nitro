package statemanager

import (
	"context"

	"errors"
	"fmt"
	"math/big"

	"math/rand"

	"github.com/OffchainLabs/challenge-protocol-v2/execution"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Defines the ABI encoding structure for submission of prefix proofs to the protocol contracts
var (
	b32Arr, _ = abi.NewType("bytes32[]", "", nil)
	// ProofArgs for submission to the protocol.
	ProofArgs = abi.Arguments{
		{Type: b32Arr, Name: "prefixExpansion"},
		{Type: b32Arr, Name: "prefixProof"},
	}
)

// Manager defines a struct that can provide local state data and historical
// Merkle commitments to L2 state for the validator.
type AssertionToCreate struct {
	PreState      *protocol.ExecutionState
	PostState     *protocol.ExecutionState
	InboxMaxCount *big.Int
	Height        uint64
}

type Manager interface {
	LatestAssertionCreationData(ctx context.Context, prevHeight uint64) (*AssertionToCreate, error)
	HasStateCommitment(ctx context.Context, commitment util.StateCommitment) bool
	HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error)
	PrefixProof(ctx context.Context, from, to uint64) ([]byte, error)
	BigStepLeafCommitment(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight uint64,
	) (util.HistoryCommitment, error)
	BigStepCommitmentUpTo(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight,
		toBigStep uint64,
	) (util.HistoryCommitment, error)
	SmallStepLeafCommitment(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight uint64,
	) (util.HistoryCommitment, error)
	SmallStepCommitmentUpTo(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight,
		toStep uint64,
	) (util.HistoryCommitment, error)
}

// Simulated defines a very naive state manager that is initialized from a list of predetermined
// state roots. It can produce state and history commitments from those roots.
type Simulated struct {
	stateRoots                []common.Hash
	executionStates           []*protocol.ExecutionState
	inboxMaxCounts            []*big.Int
	maxWavmOpcodes            uint64
	numOpcodesPerBigStep      uint64
	bigStepDivergenceHeight   uint64
	smallStepDivergenceHeight uint64
}

// New simulated manager from a list of predefined state roots, useful for tests and simulations.
func New(stateRoots []common.Hash) *Simulated {
	if len(stateRoots) == 0 {
		panic("must have state roots")
	}
	return &Simulated{stateRoots: stateRoots}
}

type Opt func(*Simulated)

func WithMaxWavmOpcodesPerBlock(maxOpcodes uint64) Opt {
	return func(s *Simulated) {
		s.maxWavmOpcodes = maxOpcodes
	}
}

func WithNumOpcodesPerBigStep(numOpcodes uint64) Opt {
	return func(s *Simulated) {
		s.numOpcodesPerBigStep = numOpcodes
	}
}

func WithBigStepStateDivergenceHeight(divergenceHeight uint64) Opt {
	return func(s *Simulated) {
		s.bigStepDivergenceHeight = divergenceHeight
	}
}

func WithSmallStepStateDivergenceHeight(divergenceHeight uint64) Opt {
	return func(s *Simulated) {
		s.smallStepDivergenceHeight = divergenceHeight
	}
}

// NewWithAssertionStates creates a simulated state manager from a list of predefined state roots for
// the top-level assertion chain, useful for tests and simulation purposes in block challenges.
// This also allows for specifying the honest states for big and small step subchallenges along
// with the point at which the state manager should diverge from the honest computation.
func NewWithAssertionStates(
	assertionChainExecutionStates []*protocol.ExecutionState,
	inboxMaxCounts []*big.Int,
	opts ...Opt,
) (*Simulated, error) {
	if len(assertionChainExecutionStates) == 0 {
		return nil, errors.New("must have execution states")
	}
	if len(assertionChainExecutionStates) != len(inboxMaxCounts) {
		return nil, fmt.Errorf(
			"number of exec states %d must match number of inbox max counts %d",
			len(assertionChainExecutionStates),
			len(inboxMaxCounts),
		)
	}
	stateRoots := make([]common.Hash, len(assertionChainExecutionStates))
	for i := 0; i < len(stateRoots); i++ {
		stateRoots[i] = protocol.ComputeStateHash(assertionChainExecutionStates[i], big.NewInt(2))
	}
	s := &Simulated{
		stateRoots:      stateRoots,
		executionStates: assertionChainExecutionStates,
		inboxMaxCounts:  inboxMaxCounts,
	}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

// LatestStateCommitment gets the state commitment corresponding to the last, local state root the manager has
// and a pre-state based on a height of the previous assertion the validator should build upon.
func (s *Simulated) LatestAssertionCreationData(
	ctx context.Context,
	prevHeight uint64,
) (*AssertionToCreate, error) {
	if len(s.executionStates) == 0 {
		return nil, errors.New("no local execution states")
	}
	if prevHeight >= uint64(len(s.stateRoots)) {
		return nil, fmt.Errorf(
			"prev height %d cannot be >= %d state roots",
			prevHeight,
			len(s.stateRoots),
		)
	}
	lastState := s.executionStates[len(s.executionStates)-1]
	return &AssertionToCreate{
		PreState:      s.executionStates[prevHeight],
		PostState:     lastState,
		InboxMaxCount: big.NewInt(1),
		Height:        uint64(len(s.stateRoots)) - 1,
	}, nil
}

// HasStateCommitment checks if a state commitment is found in our local list of state roots.
func (s *Simulated) HasStateCommitment(ctx context.Context, commitment util.StateCommitment) bool {
	if commitment.Height >= uint64(len(s.stateRoots)) {
		return false
	}
	return s.stateRoots[commitment.Height] == commitment.StateRoot
}

// HistoryCommitmentUpTo gets the history commitment for the merkle expansion up to a height.
func (s *Simulated) HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error) {
	// The size is the number of elements being committed to. For example, if the height is 7, there will
	// be 8 elements being committed to from [0, 7] inclusive.
	size := height + 1
	return util.NewHistoryCommitment(
		height,
		s.stateRoots[:size],
	)
}

// BigStepLeafCommitment produces a big step history commitment which includes
// a Merkleization of the N big-steps in between assertions A and B. This function
// is called when a validator is preparing a subchallenge on assertions A and B that
// are one-step away from each other. It will then load up the big steps
// between those two heights and produce a commitment.
func (s *Simulated) BigStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight uint64,
) (util.HistoryCommitment, error) {
	if fromAssertionHeight+1 != toAssertionHeight {
		return util.HistoryCommitment{}, fmt.Errorf(
			"from height %d is not one-step away from to height %d",
			fromAssertionHeight,
			toAssertionHeight,
		)
	}
	// Number of big steps between assertion heights A and B will be
	// fixed in this simulated state manager. It is simply the max number of opcodes
	// per block divided by the size of a big step.
	numBigSteps := s.maxWavmOpcodes / s.numOpcodesPerBigStep
	return s.BigStepCommitmentUpTo(
		ctx,
		fromAssertionHeight,
		toAssertionHeight,
		numBigSteps,
	)
}

func (s *Simulated) setupEngine(fromHeight, toHeight uint64) (*execution.Engine, error) {
	machineCfg := execution.DefaultMachineConfig()
	if s.maxWavmOpcodes > 0 {
		machineCfg.MaxInstructionsPerBlock = s.maxWavmOpcodes
	}
	if s.numOpcodesPerBigStep > 0 {
		machineCfg.BigStepSize = s.numOpcodesPerBigStep
	}
	return execution.NewExecutionEngine(
		machineCfg,
		s.stateRoots[fromHeight:toHeight+1],
	)
}

func (s *Simulated) BigStepCommitmentUpTo(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	toBigStep uint64,
) (util.HistoryCommitment, error) {
	engine, err := s.setupEngine(fromAssertionHeight, toAssertionHeight)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	if engine.NumBigSteps() < toBigStep {
		return util.HistoryCommitment{}, errors.New("not enough big steps")
	}
	return s.commitmentFromEngineSteps(
		toBigStep,
		s.bigStepDivergenceHeight,
		engine,
		engine.StateAfterBigSteps,
	)
}

// SmallStepLeafCommitment produces a small step history commitment which includes
// a Merkleization of the N WAVM opcodes in between big steps A and B. This function
// is called when a validator is preparing a subchallenge on big-steps A and B that
// are one-step away from each other. It will then load up the WAVM opcodes
// between those two values and produce a commitment.
func (s *Simulated) SmallStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight uint64,
) (util.HistoryCommitment, error) {
	if fromAssertionHeight+1 != toAssertionHeight {
		return util.HistoryCommitment{}, fmt.Errorf(
			"from height %d is not one-step away from to height %d",
			fromAssertionHeight,
			toAssertionHeight,
		)
	}
	return s.SmallStepCommitmentUpTo(
		ctx,
		fromAssertionHeight,
		toAssertionHeight,
		s.numOpcodesPerBigStep,
	)
}

func (s *Simulated) SmallStepCommitmentUpTo(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	toPc uint64,
) (util.HistoryCommitment, error) {
	engine, err := s.setupEngine(fromAssertionHeight, toAssertionHeight)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	if engine.NumOpcodes() < toPc {
		return util.HistoryCommitment{}, errors.New("not enough small steps")
	}
	return s.commitmentFromEngineSteps(
		toPc,
		s.smallStepDivergenceHeight,
		engine,
		engine.StateAfterSmallSteps,
	)
}

func (s *Simulated) commitmentFromEngineSteps(
	toStep,
	divergenceHeight uint64,
	engine execution.EngineAtBlock,
	stepperFn func(n uint64) (execution.IntermediateStateIterator, error),
) (util.HistoryCommitment, error) {
	leaves := make([]common.Hash, toStep+1)
	leaves[0] = engine.FirstState()

	// Up to and including the specified small step.
	for i := uint64(0); i <= toStep; i++ {
		start, err := stepperFn(i)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		intermediateState, err := start.NextState()
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		var hash common.Hash
		if divergenceHeight == 0 || i < divergenceHeight {
			hash = intermediateState.Hash()
		} else {
			junkRoot := make([]byte, 32)
			_, err := rand.Read(junkRoot)
			if err != nil {
				return util.HistoryCommitment{}, err
			}
			hash = crypto.Keccak256Hash(junkRoot)
		}
		leaves[i] = hash
	}
	return util.NewHistoryCommitment(toStep, leaves)
}

// PrefixProof generates a proof of a merkle expansion from genesis to a low point to a slice of state roots
// from a low point to a high point specified as arguments.
func (s *Simulated) PrefixProof(ctx context.Context, lo, hi uint64) ([]byte, error) {
	loSize := lo + 1
	hiSize := hi + 1
	prefixExpansion, err := prefixproofs.ExpansionFromLeaves(s.stateRoots[:loSize])
	if err != nil {
		return nil, err
	}
	prefixProof, err := prefixproofs.GeneratePrefixProof(
		loSize,
		prefixExpansion,
		s.stateRoots[loSize:hiSize],
		prefixproofs.RootFetcherFromExpansion,
	)
	if err != nil {
		return nil, err
	}
	_, numRead := prefixproofs.MerkleExpansionFromCompact(prefixProof, loSize)
	onlyProof := prefixProof[numRead:]
	return ProofArgs.Pack(&prefixExpansion, &onlyProof)
}
