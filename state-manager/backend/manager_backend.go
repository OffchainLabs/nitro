package statemanagerbackend

import (
	"context"
	"errors"

	"github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
)

// ManagerBackend defines a struct that can provide state data and historical
// Merkle commitments to L2 state for the validator from a backend.
type ManagerBackend interface {
	GetMerkleRoot(ctx context.Context, start uint64, end uint64) (common.Hash, error)
	GetStateRoot(ctx context.Context, height uint64) (common.Hash, error)
	GetLatestStateHeight(ctx context.Context) (uint64, error)
}

// SimulatedManagerBackend defines a very naive manager backend that is initialized from a list of predetermined
// state roots. It can produce state and Merkle roots from those roots.
type SimulatedManagerBackend struct {
	stateRoots []common.Hash
}

func NewSimulatedManagerBackend(stateRoots []common.Hash) *SimulatedManagerBackend {
	if len(stateRoots) == 0 {
		panic("must have state roots")
	}
	return &SimulatedManagerBackend{stateRoots}
}

// GetMerkleRoot gets merkle root from start to end state passed as arguments from our local list of state roots.
func (s *SimulatedManagerBackend) GetMerkleRoot(ctx context.Context, start uint64, end uint64) (common.Hash, error) {
	if start >= uint64(len(s.stateRoots)) || end >= uint64(len(s.stateRoots)) || start > end {
		return common.Hash{}, errors.New("commitment height out of range")
	}
	exp, err := prefixproofs.ExpansionFromLeaves(s.stateRoots[start : end+1])
	if err != nil {
		return common.Hash{}, err
	}
	return prefixproofs.Root(exp), nil
}

// GetStateRoot gets the state root at a specified height from our local list of state roots.
func (s *SimulatedManagerBackend) GetStateRoot(ctx context.Context, height uint64) (common.Hash, error) {
	if height >= uint64(len(s.stateRoots)) {
		return common.Hash{}, errors.New("commitment height out of range")
	}
	return s.stateRoots[height], nil
}

// GetLatestStateHeight gets the state height corresponding to the last, our local list of state root has.
func (s *SimulatedManagerBackend) GetLatestStateHeight(ctx context.Context) (uint64, error) {
	return uint64(len(s.stateRoots) - 1), nil
}
