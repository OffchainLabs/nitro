// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type SequencerTrigger struct {
	stopwaiter.StopWaiter

	execSequencer execution.ExecutionSequencer
	txStreamer    *TransactionStreamer
}

func NewSequencerTrigger(
	execSequencer execution.ExecutionSequencer,
	txStreamer *TransactionStreamer,
) *SequencerTrigger {
	return &SequencerTrigger{
		execSequencer: execSequencer,
		txStreamer:    txStreamer,
	}
}

func (s *SequencerTrigger) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.CallIteratively(s.triggerSequencing)
}

func (s *SequencerTrigger) triggerSequencing(ctx context.Context) time.Duration {
	sequencedMsg, nextSequenceCall := s.execSequencer.Sequence(ctx)
	if sequencedMsg != nil {
		err := s.txStreamer.WriteSequencedMsg(sequencedMsg)
		if err != nil {
			log.Error("Error writing sequenced message", "err", err)
		}
	}
	return nextSequenceCall
}
