package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

// txByLogFetcher is wrapper around ParentChainReader to implement TransactionByLog method
type txByLogFetcher struct {
	client ParentChainReader
}

func (f *txByLogFetcher) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	if log == nil {
		return nil, errors.New("transactionByLog got nil log value")
	}
	tx, _, err := f.client.TransactionByHash(ctx, log.TxHash)
	return tx, err
}

func (m *MessageExtractor) processNextBlock(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	// Process the next block in the parent chain and extracts messages.
	processAction, ok := current.SourceEvent.(processNextBlock)
	if !ok {
		return m.config.RetryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
	}
	preState := processAction.melState
	if preState.GetDelayedMessageBacklog() == nil { // Safety check since its relevant for native mode
		return m.config.RetryInterval, errors.New("detected nil DelayedMessageBacklog of melState, shouldnt be possible")
	}
	// If the current parent chain block is not safe/finalized we wait till it becomes safe/finalized as determined by the ReadMode
	if m.config.ReadMode != "latest" && preState.ParentChainBlockNumber+1 > m.lastBlockToRead.Load() {
		return m.config.RetryInterval, nil
	}
	parentChainBlock, err := m.logsAndHeadersPreFetcher.getHeaderByNumber(ctx, preState.ParentChainBlockNumber+1)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// If the block with the specified number is not found, it likely has not
			// been posted yet to the parent chain, so we can retry
			// without returning an error from the FSM.
			if !m.caughtUp && m.config.ReadMode == "latest" {
				if latestBlk, err := m.parentChainReader.HeaderByNumber(ctx, big.NewInt(rpc.LatestBlockNumber.Int64())); err != nil {
					log.Error("Error fetching LatestBlockNumber from parent chain to determine if mel has caught up", "err", err)
				} else if latestBlk.Number.Uint64()-preState.ParentChainBlockNumber <= 5 { // tolerance of catching up i.e parent chain might have progressed in the time between the above two function calls
					m.caughtUp = true
					close(m.caughtUpChan)
				}
			}
			return m.config.RetryInterval, nil
		} else {
			return m.config.RetryInterval, err
		}
	}
	if parentChainBlock.ParentHash != preState.ParentChainBlockHash {
		log.Info("MEL detected L1 reorg", "block", preState.ParentChainBlockNumber) // Log level is Info because L1 reorgs are a common occurrence
		return 0, m.fsm.Do(reorgToOldBlock{
			melState: preState,
		})
	}
	// Reorging of MEL states successfully completed, we can now rewind MEL validator
	if m.reorgEventsNotifier != nil && processAction.prevStepWasReorg {
		m.reorgEventsNotifier <- preState.ParentChainBlockNumber
	}
	// Conditionally prefetch headers and logs for upcoming block/s
	if err = m.logsAndHeadersPreFetcher.fetch(ctx, preState); err != nil {
		return m.config.RetryInterval, err
	}
	start := time.Now()
	postState, msgs, delayedMsgs, batchMetas, err := melextraction.ExtractMessages(
		ctx,
		preState,
		parentChainBlock,
		m.dataProviders,
		m.melDB,
		&txByLogFetcher{m.parentChainReader},
		m.logsAndHeadersPreFetcher,
		m.chainConfig,
	)
	if err != nil {
		extractionErrors.Inc(1)
		return m.config.RetryInterval, err
	}
	end := time.Since(start)
	// After processing every 100 parent chain blocks, print a status log
	if postState.ParentChainBlockNumber%100 == 0 {
		log.Info("Message extraction successful", "parentChainBlockNumber", postState.ParentChainBlockNumber, "msgCount", postState.MsgCount)
	}

	// Update metrics.
	latestBlockGauge.Update(int64(postState.ParentChainBlockNumber))
	latestMsgCountGauge.Update(int64(postState.MsgCount))
	latestDelayedSeenCountGauge.Update(int64(postState.DelayedMessagesSeen))
	latestDelayedReadCountGauge.Update(int64(postState.DelayedMessagesRead))
	msgsExtractedCounter.Inc(int64(len(msgs)))
	blockProcessTimeGauge.Update(end.Microseconds())

	// Begin the next FSM state immediately.
	return 0, m.fsm.Do(saveMessages{
		preStateMsgCount: preState.MsgCount,
		postState:        postState,
		messages:         msgs,
		delayedMessages:  delayedMsgs,
		batchMetas:       batchMetas,
	})
}
