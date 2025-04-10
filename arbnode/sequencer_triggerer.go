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
	s.txStreamer.insertionMutex.Lock()
	defer s.txStreamer.insertionMutex.Unlock()

	if err := s.txStreamer.ExpectChosenSequencer(); err != nil {
		log.Debug("Not active sequencer, retrying", "err", err)
		return 50 * time.Millisecond
	}

	sequencedMsg, nextSequenceCall := s.execSequencer.Sequence(ctx)
	if sequencedMsg != nil {
		err := s.txStreamer.WriteSequencedMsg(sequencedMsg)
		if errors.Is(err, execution.ErrRetrySequencer) {
			log.Error("Error writing sequenced message, re-adding transactions from sequenced msg", "err", err)
			err = s.execSequencer.ReAddTransactionsFromLastCreatedBlock(ctx, sequencedMsg)
			if err != nil {
				log.Error("Error re-adding transactions from last created block", "err", err)
				return 0
			}
			return 0
		} else if err != nil {
			log.Error("Error writing sequenced message", "err", err)
			return nextSequenceCall
		}

		err = s.execSequencer.AppendLastSequencedBlock(sequencedMsg.MsgResult.BlockHash)
		if err != nil {
			log.Error("Error appending last sequenced block", "err", err)
			return nextSequenceCall
		}
		nextSequenceCall, err = s.execSequencer.ProcessHooksFromLastCreatedBlock(ctx, sequencedMsg.MsgResult.BlockHash)
		if err != nil {
			log.Error("Error processing hooks from last created block", "err", err)
			return 0
		}

	}
	return nextSequenceCall
}
