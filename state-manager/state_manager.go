package statemanager

import (
	"context"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
)

type Manager interface {
	LatestHistoryCommitment(ctx context.Context) util.HistoryCommitment
	HasStateRoot(ctx context.Context, stateRoot common.Hash) bool
	HistoryCommitmentAtHeight(ctx context.Context, height uint64) (util.HistoryCommitment, error)
	SubscribeStateEvents(ctx context.Context, ch chan<- *StateAdvancedEvent)
}

type StateAdvancedEvent struct {
	HistoryCommitment *util.HistoryCommitment
}

type Simulated struct{}

func (s *Simulated) SubscribeStateEvents(ctx context.Context, ch chan<- *StateAdvancedEvent) {
	panic("unimplemented")
}

func (s *Simulated) HasStateRoot(ctx context.Context, stateRoot common.Hash) bool {
	panic("unimplemented")
}

// LatestHistoryCommitment --
func (s *Simulated) LatestHistoryCommitment(_ context.Context) util.HistoryCommitment {
	panic("unimplemented")
}

// HistoryCommitmentAtHeight --
func (s *Simulated) HistoryCommitmentAtHeight(_ context.Context, height uint64) (util.HistoryCommitment, error) {
	panic("unimplemented")
}
