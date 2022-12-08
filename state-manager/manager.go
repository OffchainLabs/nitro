package statemanager

import (
	"context"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
)

// Manager defines a struct that can provide local state data and historical
// Merkle commitments to L2 state for the validator.
type Manager interface {
	HasStateCommitment(ctx context.Context, commitment protocol.StateCommitment) bool
	StateCommitmentAtHeight(ctx context.Context, height uint64) (protocol.StateCommitment, error)
	LatestStateCommitment(ctx context.Context) (protocol.StateCommitment, error)
}
