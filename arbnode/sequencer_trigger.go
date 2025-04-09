// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
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
	s.txStreamer.insertionMutex.Lock()
	defer s.txStreamer.insertionMutex.Unlock()

	if err := s.txStreamer.ExpectChosenSequencer(); err != nil {
		if errors.Is(err, execution.ErrRetrySequencer) {
			log.Debug("Not active sequencer, retrying", "err", err)
		} else {
			log.Error("Error expecting chosen sequencer", "err", err)
		}
		return 100 * time.Millisecond
	}

	sequencedMsg, nextSequenceCall := s.execSequencer.Sequence(ctx)
	if sequencedMsg != nil {
		err := s.txStreamer.WriteSequencedMsg(sequencedMsg)
		if err != nil {
			log.Error("Error writing sequenced message", "err", err)
			return nextSequenceCall
		}

		err = s.execSequencer.AppendLastSequencedBlock(sequencedMsg.MsgResult.BlockHash)
		if err != nil {
			log.Error("Error appending last sequenced block", "err", err)
		}
	}
	return nextSequenceCall
}
