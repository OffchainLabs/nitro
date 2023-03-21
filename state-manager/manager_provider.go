package statemanager

import (
	"context"
	"math/bits"

	statemanagerbackend "github.com/OffchainLabs/challenge-protocol-v2/state-manager/backend"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
)

// ManagerProvider defines a state manager that is initialized from a ManagerBackend.
// It can produce state and history commitments from backend.
type ManagerProvider struct {
	ManagerBackend statemanagerbackend.ManagerBackend
}

// NewManagerProvider from a list of predefined state roots, useful for tests and simulations.
func NewManagerProvider(stateRoots []common.Hash) *ManagerProvider {
	if len(stateRoots) == 0 {
		panic("must have state roots")
	}
	return &ManagerProvider{statemanagerbackend.NewSimulatedManagerBackend(stateRoots)}
}

// HasStateCommitment checks if a state commitment is found in manager backend.
func (m *ManagerProvider) HasStateCommitment(ctx context.Context, commitment util.StateCommitment) bool {
	root, err := m.ManagerBackend.GetStateRoot(ctx, commitment.Height)
	if err != nil {
		return false
	}
	return root == commitment.StateRoot
}

// StateCommitmentAtHeight gets the state commitment at a specified height from manager backend.
func (m *ManagerProvider) StateCommitmentAtHeight(ctx context.Context, height uint64) (util.StateCommitment, error) {
	root, err := m.ManagerBackend.GetStateRoot(ctx, height)
	if err != nil {
		return util.StateCommitment{}, err
	}
	return util.StateCommitment{
		Height:    height,
		StateRoot: root,
	}, nil
}

// LatestStateCommitment gets the state commitment corresponding to the last, manager backend has.
func (m *ManagerProvider) LatestStateCommitment(ctx context.Context) (util.StateCommitment, error) {
	height, err := m.ManagerBackend.GetLatestStateHeight(ctx)
	if err != nil {
		return util.StateCommitment{}, err
	}
	root, err := m.ManagerBackend.GetStateRoot(ctx, height)
	if err != nil {
		return util.StateCommitment{}, err
	}
	return util.StateCommitment{
		Height:    height,
		StateRoot: root,
	}, nil
}

// HistoryCommitmentUpTo gets the history commitment for the merkle expansion up to a height.
func (m *ManagerProvider) HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error) {
	root, err := m.ManagerBackend.GetMerkleRoot(ctx, 0, height-1)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.HistoryCommitment{
		Height: height,
		Merkle: root,
	}, nil
}

// PrefixProof generates a proof of a merkle expansion from genesis to a low point to a slice of state roots
// from a low point to a high point specified as arguments.
func (m *ManagerProvider) PrefixProof(ctx context.Context, lo, hi uint64) ([]common.Hash, error) {
	highBit := 63 - bits.LeadingZeros64(lo)
	proofStart := uint64(0)
	var exp prefixproofs.MerkleExpansion
	for i := highBit; i >= 0; i-- {
		if (lo & (1 << i)) > 0 {
			root, err := m.ManagerBackend.GetMerkleRoot(ctx, proofStart, proofStart+(1<<i)-1)
			if err != nil {
				return nil, err
			}
			exp = append([]common.Hash{root}, exp...)
			proofStart = proofStart + (1 << i)
		}
	}
	return prefixproofs.GeneratePrefixProof(
		lo,
		exp,
		nil,
		func(_ []common.Hash, toHeight uint64) (common.Hash, error) {
			return m.ManagerBackend.GetMerkleRoot(ctx, 0, hi)
		},
	)
}

// HasHistoryCommitment checks if a history commitment matches our merkle expansion for the specified height.
func (m *ManagerProvider) HasHistoryCommitment(ctx context.Context, commitment util.HistoryCommitment) bool {
	root, err := m.ManagerBackend.GetMerkleRoot(ctx, 0, commitment.Height-1)
	if err != nil {
		return false
	}
	return root == commitment.Merkle
}

// LatestHistoryCommitment gets the history commitment up to and including the last, manager backend has.
func (m *ManagerProvider) LatestHistoryCommitment(ctx context.Context) (util.HistoryCommitment, error) {
	height, err := m.ManagerBackend.GetLatestStateHeight(ctx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	root, err := m.ManagerBackend.GetMerkleRoot(ctx, 0, height-1)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return util.HistoryCommitment{
		Height: height,
		Merkle: root,
	}, nil
}
