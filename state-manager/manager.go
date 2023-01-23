package statemanager

import (
	"context"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
)

// Manager defines a struct that can provide local state data and historical
// Merkle commitments to L2 state for the validator.
type Manager interface {
	HasStateCommitment(ctx context.Context, commitment protocol.StateCommitment) bool
	StateCommitmentAtHeight(ctx context.Context, height uint64) (protocol.StateCommitment, error)
	LatestStateCommitment(ctx context.Context) (protocol.StateCommitment, error)
	HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error)
	PrefixProof(ctx context.Context, from, to uint64) ([]common.Hash, error)
	HasHistoryCommitment(ctx context.Context, commitment util.HistoryCommitment) bool
	LatestHistoryCommitment(ctx context.Context) (util.HistoryCommitment, error)
}

// Simulated defines a very naive state manager that is initialized from a list of predetermined
// state roots. It can produce state and history commitments from those roots.
type Simulated struct {
	stateRoots []common.Hash
}

// New simulated manager from a list of predefined state roots, useful for tests and simulations.
func New(stateRoots []common.Hash) *Simulated {
	if len(stateRoots) == 0 {
		panic("must have state roots")
	}
	return &Simulated{stateRoots}
}

// HasStateCommitment checks if a state commitment is found in our local list of state roots.
func (s *Simulated) HasStateCommitment(ctx context.Context, commitment protocol.StateCommitment) bool {
	if commitment.Height >= uint64(len(s.stateRoots)) {
		return false
	}
	return s.stateRoots[commitment.Height] == commitment.StateRoot
}

// StateCommitmentAtHeight gets the state commitment at a specified height from our local list of state roots.
func (s *Simulated) StateCommitmentAtHeight(ctx context.Context, height uint64) (protocol.StateCommitment, error) {
	if height >= uint64(len(s.stateRoots)) {
		panic("commitment height out of range")
	}
	return protocol.StateCommitment{
		Height:    height,
		StateRoot: s.stateRoots[height],
	}, nil
}

// LatestStateCommitment gets the state commitment corresponding to the last, local state root the manager has.
func (s *Simulated) LatestStateCommitment(ctx context.Context) (protocol.StateCommitment, error) {
	return protocol.StateCommitment{
		Height:    uint64(len(s.stateRoots)) - 1,
		StateRoot: s.stateRoots[len(s.stateRoots)-1],
	}, nil
}

// HistoryCommitmentUpTo gets the history commitment for the merkle expansion up to a height.
func (s *Simulated) HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error) {
	exp := util.ExpansionFromLeaves(s.stateRoots[:height])
	return util.HistoryCommitment{
		Height: height,
		Merkle: exp.Root(),
	}, nil
}

// PrefixProof generates a proof of a merkle expansion from genesis to a low point to a slice of state roots
// from a low point to a high point specified as arguments.
func (s *Simulated) PrefixProof(ctx context.Context, lo, hi uint64) ([]common.Hash, error) {
	exp := util.ExpansionFromLeaves(s.stateRoots[:lo])
	return util.GeneratePrefixProof(
		lo,
		exp,
		s.stateRoots[lo:hi],
	), nil
}

// HasHistoryCommitment checks if a history commitment matches our merkle expansion for the specified height.
func (s *Simulated) HasHistoryCommitment(ctx context.Context, commitment util.HistoryCommitment) bool {
	if commitment.Height >= uint64(len(s.stateRoots)) {
		return false
	}
	merkle := util.ExpansionFromLeaves(s.stateRoots[:commitment.Height]).Root()
	return merkle == commitment.Merkle
}

// LatestHistoryCommitment gets the history commitment up to and including the last, local state root the manager has.
func (s *Simulated) LatestHistoryCommitment(ctx context.Context) (util.HistoryCommitment, error) {
	height := uint64(len(s.stateRoots)) - 1
	return util.HistoryCommitment{
		Height: height,
		Merkle: util.ExpansionFromLeaves(s.stateRoots[:height]).Root(),
	}, nil
}
