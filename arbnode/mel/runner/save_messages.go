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
	// Persists messages and a processed MEL state to the database.
	saveAction, ok := current.SourceEvent.(saveMessages)
	if !ok {
		return m.config.RetryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
	}
	if err := m.melDB.SaveBatchMetas(saveAction.postState, saveAction.batchMetas); err != nil {
		return m.config.RetryInterval, err
	}
	if err := m.melDB.SaveDelayedMessages(saveAction.postState, saveAction.delayedMessages); err != nil {
		return m.config.RetryInterval, err
	}
	if err := m.msgConsumer.PushMessages(ctx, saveAction.preStateMsgCount, saveAction.messages); err != nil {
		return m.config.RetryInterval, err
	}
	if err := m.melDB.SaveState(saveAction.postState); err != nil {
		log.Error("Error saving messages from MessageExtractor to MessageConsumer", "err", err)
		return m.config.RetryInterval, err
	}
	if saveAction.postState.ParentChainBlockNumber%1000 == 0 {
		if err := saveAction.postState.RebuildDelayedMsgPreimages(m.melDB.FetchDelayedMessage); err != nil {
			return m.config.RetryInterval, fmt.Errorf("error rebuilding delayed msg preimages for cleanup: %w", err)
		}
	}
	return 0, m.fsm.Do(processNextBlock{
		melState: saveAction.postState,
	})
}
