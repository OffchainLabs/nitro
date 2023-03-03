package statemanager

import (
	"context"

	"errors"
	"fmt"
	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/execution"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
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
	PrefixProof(ctx context.Context, from, to uint64) ([]common.Hash, error)
	BigStepLeafCommitment(
		ctx context.Context,
		fromAssertionHeight,
		toAssertionHeight uint64,
	) (util.HistoryCommitment, error)
	BigStepCommitmentUpTo(
		ctx context.Context,
		toBigStep uint64,
	) (util.HistoryCommitment, error)
	SmallStepLeafCommitment(
		ctx context.Context,
		fromBigStep,
		toBigStep uint64,
	) (util.HistoryCommitment, error)
	SmallStepCommitmentUpTo(
		ctx context.Context,
		toStep uint64,
	) (util.HistoryCommitment, error)
}

// Simulated defines a very naive state manager that is initialized from a list of predetermined
// state roots. It can produce state and history commitments from those roots.
type Simulated struct {
	stateRoots      []common.Hash
	executionStates []*protocol.ExecutionState
	inboxMaxCounts  []*big.Int
}

// New simulated manager from a list of predefined state roots, useful for tests and simulations.
func New(stateRoots []common.Hash) *Simulated {
	if len(stateRoots) == 0 {
		panic("must have state roots")
	}
	return &Simulated{stateRoots: stateRoots}
}

// New simulated manager from a list of predefined state roots, useful for tests and simulations.
func NewWithExecutionStates(executionStates []*protocol.ExecutionState, inboxMaxCounts []*big.Int) *Simulated {
	if len(executionStates) == 0 {
		panic("must have execution states")
	}
	if len(executionStates) != len(inboxMaxCounts) {
		panic("number of exec states must match number of inbox max counts")
	}
	stateRoots := make([]common.Hash, len(executionStates))
	for i := 0; i < len(stateRoots); i++ {
		stateRoots[i] = protocol.ComputeStateHash(executionStates[i], big.NewInt(2))
	}
	return &Simulated{
		stateRoots:      stateRoots,
		executionStates: executionStates,
		inboxMaxCounts:  inboxMaxCounts,
	}
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
	return util.NewHistoryCommitment(
		height,
		s.stateRoots[:height+1],
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
	if toAssertionHeight != fromAssertionHeight+1 {
		return util.HistoryCommitment{}, fmt.Errorf(
			"from height %d is not one-step away from to height %d",
			fromAssertionHeight,
			toAssertionHeight,
		)
	}

	preState := s.stateRoots[fromAssertionHeight]
	postState := s.stateRoots[toAssertionHeight]
	engine, err := execution.NewExecutionEngine(toAssertionHeight, preState, postState, &execution.Config{
		FixedNumSteps: 1,
	})
	if err != nil {
		return util.HistoryCommitment{}, err
	}

	expansion := util.NewEmptyMerkleExpansion()

	// TODO: Advance by big-steps instead.
	var total int
	for i := uint64(0); i < engine.NumOpcodes(); i++ {
		start, err := engine.StateAfter(i)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		intermediateState, err := start.NextState()
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		expansion = expansion.AppendLeaf(intermediateState.Hash())
		total++
	}

	return util.HistoryCommitment{
		Height: engine.NumBigSteps(),
		Merkle: expansion.Root(),
	}, nil
}

func (s *Simulated) BigStepCommitmentUpTo(
	ctx context.Context,
	toBigStep uint64,
) (util.HistoryCommitment, error) {
	return util.HistoryCommitment{
		Height: toBigStep,
	}, nil
}

func (s *Simulated) SmallStepLeafCommitment(
	ctx context.Context,
	fromBigStep,
	toBigStep uint64,
) (util.HistoryCommitment, error) {
	if toBigStep != fromBigStep+1 {
		return util.HistoryCommitment{}, fmt.Errorf(
			"from height %d is not one-step away from to height %d",
			fromBigStep,
			toBigStep,
		)
	}
	// TODO: Not state roots but rather intermediate wavm roots in a big step challenge.
	// We should probably pass these into the function.
	preState := s.stateRoots[fromBigStep]
	postState := s.stateRoots[toBigStep]

	// TODO: Args should be block num instead of big step num, and pre/post states should
	// be top-level states in this case.
	engine, err := execution.NewExecutionEngine(toBigStep, preState, postState, &execution.Config{
		FixedNumSteps: 1,
	})
	if err != nil {
		return util.HistoryCommitment{}, err
	}

	expansion := util.NewEmptyMerkleExpansion()

	var total int
	for i := uint64(0); i < engine.NumOpcodes(); i++ {
		start, err := engine.StateAfter(i)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		intermediateState, err := start.NextState()
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		expansion = expansion.AppendLeaf(intermediateState.Hash())
		total++
	}

	return util.HistoryCommitment{
		Height: engine.NumOpcodes(),
		Merkle: expansion.Root(),
	}, nil
}

func (s *Simulated) SmallStepCommitmentUpTo(
	ctx context.Context,
	toStep uint64,
) (util.HistoryCommitment, error) {
	return util.HistoryCommitment{
		Height: toStep,
	}, nil
}

// PrefixProof generates a proof of a merkle expansion from genesis to a low point to a slice of state roots
// from a low point to a high point specified as arguments.
func (s *Simulated) PrefixProof(ctx context.Context, lo, hi uint64) ([]common.Hash, error) {
	exp := util.ExpansionFromLeaves(s.stateRoots[:lo])
	return util.GeneratePrefixProof(
		lo,
		exp,
		s.stateRoots[lo:hi+1],
	), nil
}
