// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

func (m *MessageExtractor) saveMessages(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	saveAction, ok := current.SourceEvent.(saveMessages)
	if !ok {
		return m.config.RetryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
	}
	// Push messages to the transaction streamer first. This is a separate DB
	// so it cannot be made atomic with the MEL DB writes below. If we crash
	// after push but before the MEL write, MEL will reprocess and push again.
	// PushMessages is idempotent for identical messages; re-pushing after a crash
	// is safe. See TransactionStreamer.AddMessagesAndEndBatch for details.
	if err := m.msgConsumer.PushMessages(ctx, saveAction.preStateMsgCount, saveAction.messages); err != nil {
		return m.config.RetryInterval, fmt.Errorf("saveMessages: pushing messages to consumer (firstMsg=%d, count=%d): %w", saveAction.preStateMsgCount, len(saveAction.messages), err)
	}
	// Atomically write batch metadata, delayed messages, and the new head
	// MEL state in a single database batch.
	if err := m.melDB.SaveProcessedBlock(saveAction.postState, saveAction.batchMetas, saveAction.delayedMessages); err != nil {
		log.Error("SaveProcessedBlock failed after messages were already pushed to streamer; MEL will retry and re-push on recovery",
			"block", saveAction.postState.ParentChainBlockNumber, "msgCount", len(saveAction.messages), "err", err)
		return m.config.RetryInterval, fmt.Errorf("saveMessages: persisting processed block %d: %w", saveAction.postState.ParentChainBlockNumber, err)
	}
	return 0, m.fsm.Do(processNextBlock{
		melState: saveAction.postState,
	})
}
