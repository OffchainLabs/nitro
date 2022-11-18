package statemanager

import (
	"context"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
)

// Manager defines a struct that can provide local state data and historical
// Merkle commitments to L2 state for the validator.
type Manager interface {
	LatestHistoryCommitment(ctx context.Context) util.HistoryCommitment
	HasStateRoot(ctx context.Context, stateRoot common.Hash) bool
	StateCommitmentAtHeight(ctx context.Context, height uint64) (util.HistoryCommitment, error)
	SubscribeStateEvents(ctx context.Context, ch chan<- *L2StateEvent)
}

type L2StateEvent struct{}
