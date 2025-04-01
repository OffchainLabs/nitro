// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"time"

	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type SequencerTrigger struct {
	stopwaiter.StopWaiter

	execSequencer execution.ExecutionSequencer
}

func NewSequencerTrigger(
	execSequencer execution.ExecutionSequencer,
) *SequencerTrigger {
	return &SequencerTrigger{
		execSequencer: execSequencer,
	}
}

func (s *SequencerTrigger) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.CallIteratively(s.triggerSequencing)
}

func (s *SequencerTrigger) triggerSequencing(ctx context.Context) time.Duration {
	_, nextSequenceCall := s.execSequencer.Sequence(ctx)
	return nextSequenceCall
}
