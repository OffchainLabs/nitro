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
	msgsPushedCounter.Inc(int64(len(saveAction.messages)))
	if err := m.melDB.SaveState(saveAction.postState); err != nil {
		log.Error("Error saving latest state as head state to db", "err", err)
		return m.config.RetryInterval, err
	}
	return 0, m.fsm.Do(processNextBlock{
		melState: saveAction.postState,
	})
}
