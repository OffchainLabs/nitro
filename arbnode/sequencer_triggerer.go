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

type SequencerTriggerer struct {
	stopwaiter.StopWaiter

	execSequencer execution.ExecutionSequencer
	txStreamer    *TransactionStreamer
}

func NewSequencerTriggerer(
	execSequencer execution.ExecutionSequencer,
	txStreamer *TransactionStreamer,
) *SequencerTriggerer {
	return &SequencerTriggerer{
		execSequencer: execSequencer,
		txStreamer:    txStreamer,
	}
}

func (s *SequencerTriggerer) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.CallIteratively(s.triggerSequencing)
}

func (s *SequencerTriggerer) triggerSequencing(ctx context.Context) time.Duration {
	startSequencingTime := time.Now()

	s.txStreamer.insertionMutex.Lock()
	defer s.txStreamer.insertionMutex.Unlock()

	if err := s.txStreamer.ExpectChosenSequencer(); err != nil {
		log.Debug("Not active sequencer, retrying", "err", err)
		return 50 * time.Millisecond
	}

	sequencedMsg, timeToWaitUntilNextSequencing := s.execSequencer.StartSequencing(ctx)

	var errWhileSequencing error
	defer s.execSequencer.EndSequencing(ctx, errWhileSequencing)

	if sequencedMsg != nil {
		errWhileSequencing = s.txStreamer.WriteSequencedMsg(sequencedMsg)
		if errWhileSequencing != nil {
			log.Error("Error writing sequenced message", "err", errWhileSequencing)
			return 0
		}

		errWhileSequencing = s.execSequencer.AppendLastSequencedBlock(sequencedMsg.MsgResult.BlockHash)
		if errWhileSequencing != nil {
			log.Error("Error appending last sequenced block", "err", errWhileSequencing)
			return 0
		}
	}
	return time.Until(startSequencingTime.Add(timeToWaitUntilNextSequencing))
}
