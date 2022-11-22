package statemanager

import (
	"context"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/util"
)

// Manager defines a struct that can provide local state data and historical
// Merkle commitments to L2 state for the validator.
type Manager interface {
	HasStateCommitment(ctx context.Context, commitment protocol.StateCommitment) bool
	StateCommitmentAtHeight(ctx context.Context, height uint64) (util.HistoryCommitment, error)
	LatestStateCommitment(ctx context.Context) (util.HistoryCommitment, error)
	SubscribeStateEvents(ctx context.Context, ch chan<- *L2StateEvent)
}

type L2StateEvent struct{}
