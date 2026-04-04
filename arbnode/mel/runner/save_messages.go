// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"fmt"
	"time"

	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

func (m *MessageExtractor) saveMessages(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	saveAction, ok := current.SourceEvent.(saveMessages)
	if !ok {
		return m.config.RetryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
	}
	// Push messages to the transaction streamer first. This is a separate DB
	// so it cannot be made atomic with the MEL DB writes below. If we crash
	// after push but before the MEL write, MEL will reprocess and push again;
	// PushMessages delegates to AddMessagesAndEndBatch(messagesAreConfirmed=true),
	// which calls countDuplicateMessages: exact byte-equal messages are skipped,
	// and messages differing only in batch gas cost fields are reconciled without
	// triggering a reorg. This idempotency guarantee is critical during migration
	// when computeMigrationStartBlock may set the start block behind the streamer's
	// current MessageCount.
	if err := m.msgConsumer.PushMessages(ctx, saveAction.preStateMsgCount, saveAction.messages); err != nil {
		return m.config.RetryInterval, fmt.Errorf("saveMessages: pushing messages to consumer (firstMsg=%d, count=%d): %w", saveAction.preStateMsgCount, len(saveAction.messages), err)
	}
	// Atomically write batch metadata, delayed messages, and the new head
	// MEL state in a single database batch.
	if err := m.melDB.SaveProcessedBlock(saveAction.postState, saveAction.batchMetas, saveAction.delayedMessages); err != nil {
		return m.config.RetryInterval, fmt.Errorf("saveMessages: persisting processed block %d: %w", saveAction.postState.ParentChainBlockNumber, err)
	}
	return 0, m.fsm.Do(processNextBlock{
		melState: saveAction.postState,
	})
}
