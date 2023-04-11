package statemanager

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/execution"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	prefixproofs "github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
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

// AssertionToCreate defines a struct that can provide local state data and historical
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
	PrefixProof(ctx context.Context, from, to uint64) ([]byte, error)
	BigStepPrefixProof(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight,
		lo,
		hi uint64,
	) ([]byte, error)
	SmallStepPrefixProof(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight,
		lo,
		hi uint64,
	) ([]byte, error)
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
func New(stateRoots []common.Hash, opts ...Opt) (*Simulated, error) {
	if len(stateRoots) == 0 {
		return nil, errors.New("no state roots provided")
	}
	s := &Simulated{stateRoots: stateRoots}
	for _, o := range opts {
		o(s)
	}
	return s, nil
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

// LatestAssertionCreationData gets the state commitment corresponding to the last, local state root the manager has
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

// BigStepCommitmentUpTo creates a history commitment up to a big step.
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
	leaves, err := s.intermediateLeavesFromEngineSteps(
		toBigStep,
		fromAssertionHeight,
		toAssertionHeight,
		protocol.BigStepChallengeEdge,
		s.bigStepDivergenceHeight,
		engine,
		engine.StateAfterBigSteps,
	)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.NewHistoryCommitment(toBigStep, leaves)
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

// SmallStepCommitmentUpTo creates a history commitment up to a program counter (step).
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
	leaves, err := s.intermediateLeavesFromEngineSteps(
		toPc,
		fromAssertionHeight,
		toAssertionHeight,
		protocol.SmallStepChallengeEdge,
		s.smallStepDivergenceHeight,
		engine,
		engine.StateAfterSmallSteps,
	)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.NewHistoryCommitment(toPc, leaves)
}

// Generates the intermediate machine hashes up to a certain step from a given engine.
func (s *Simulated) intermediateLeavesFromEngineSteps(
	toStep,
	fromAssertionHeight,
	toAssertionHeight uint64,
	chalType protocol.EdgeType,
	divergenceHeight uint64,
	engine execution.EngineAtBlock,
	stepperFn func(n uint64) (execution.IntermediateStateIterator, error),
) ([]common.Hash, error) {
	leaves := make([]common.Hash, 0)
	leaves = append(leaves, engine.FirstState())
	// Up to and including the specified step.
	for i := uint64(0); i < toStep; i++ {
		start, err := stepperFn(i)
		if err != nil {
			return nil, err
		}
		intermediateState, err := start.NextState()
		if err != nil {
			return nil, err
		}
		var hash common.Hash

		// For testing purposes, if we want to diverge from the honest
		// hashes starting at a specified hash.
		if divergenceHeight == 0 || i+1 < divergenceHeight {
			hash = intermediateState.Hash()
		} else {
			hash = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d:%d:%d:%d", i, fromAssertionHeight, toAssertionHeight, chalType)))
		}
		leaves = append(leaves, hash)
	}
	return leaves, nil
}

// BigStepPrefixProof for a big step subchallenge from assertion N to N+1 from a height lo to hi.
func (s *Simulated) BigStepPrefixProof(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	lo,
	hi uint64,
) ([]byte, error) {
	if fromAssertionHeight+1 != toAssertionHeight {
		return nil, fmt.Errorf(
			"fromAssertionHeight=%d is not 1 height apart from toAssertionHeight=%d",
			fromAssertionHeight,
			toAssertionHeight,
		)
	}
	engine, err := s.setupEngine(fromAssertionHeight, toAssertionHeight)
	if err != nil {
		return nil, err
	}
	if engine.NumOpcodes() < hi {
		return nil, err
	}
	return s.subchallengePrefixProof(
		engine,
		fromAssertionHeight,
		toAssertionHeight,
		protocol.BigStepChallengeEdge,
		s.bigStepDivergenceHeight,
		lo,
		hi,
		engine.StateAfterBigSteps,
	)
}

// SmallStepPrefixProof for a small step subchallenge from assertion N to N+1 from a height lo to hi.
func (s *Simulated) SmallStepPrefixProof(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	lo,
	hi uint64,
) ([]byte, error) {
	if fromAssertionHeight+1 != toAssertionHeight {
		return nil, fmt.Errorf(
			"fromAssertionHeight=%d is not 1 height apart from toAssertionHeight=%d",
			fromAssertionHeight,
			toAssertionHeight,
		)
	}
	engine, err := s.setupEngine(fromAssertionHeight, toAssertionHeight)
	if err != nil {
		return nil, err
	}
	if engine.NumOpcodes() < hi {
		return nil, err
	}
	return s.subchallengePrefixProof(
		engine,
		fromAssertionHeight,
		toAssertionHeight,
		protocol.SmallStepChallengeEdge,
		s.smallStepDivergenceHeight,
		lo,
		hi,
		engine.StateAfterSmallSteps,
	)
}

func (s *Simulated) subchallengePrefixProof(
	engine execution.EngineAtBlock,
	fromAssertionHeight,
	toAssertionHeight uint64,
	challengeType protocol.EdgeType,
	divergenceHeight uint64,
	lo,
	hi uint64,
	stepperFn func(n uint64) (execution.IntermediateStateIterator, error),
) ([]byte, error) {
	loSize := lo + 1
	hiSize := hi + 1
	prefixLeaves, err := s.intermediateLeavesFromEngineSteps(
		hiSize,
		fromAssertionHeight,
		toAssertionHeight,
		challengeType,
		divergenceHeight,
		engine,
		stepperFn,
	)
	if err != nil {
		return nil, err
	}
	prefixExpansion, err := prefixproofs.ExpansionFromLeaves(prefixLeaves[:loSize])
	if err != nil {
		return nil, err
	}
	prefixProof, err := prefixproofs.GeneratePrefixProof(
		loSize,
		prefixExpansion,
		prefixLeaves[loSize:hiSize],
		prefixproofs.RootFetcherFromExpansion,
	)
	if err != nil {
		return nil, err
	}
	_, numRead := prefixproofs.MerkleExpansionFromCompact(prefixProof, loSize)
	onlyProof := prefixProof[numRead:]
	return ProofArgs.Pack(&prefixExpansion, &onlyProof)
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
