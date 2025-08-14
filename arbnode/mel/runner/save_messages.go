package melrunner

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/bold/containers/fsm"
)

func (m *MessageExtractor) saveMessages(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	// Persists messages and a processed MEL state to the database.
	saveAction, ok := current.SourceEvent.(saveMessages)
	if !ok {
		return m.retryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
	}
	saveAction.postState.GetDelayedMessageBacklog().CommitDirties()
	if err := m.melDB.SaveBatchMetas(ctx, saveAction.postState, saveAction.batchMetas); err != nil {
		return m.retryInterval, err
	}
	if err := m.melDB.SaveDelayedMessages(ctx, saveAction.postState, saveAction.delayedMessages); err != nil {
		return m.retryInterval, err
	}
	if err := m.msgConsumer.PushMessages(ctx, saveAction.preStateMsgCount, saveAction.messages); err != nil {
		return m.retryInterval, err
	}
	if err := m.melDB.SaveState(ctx, saveAction.postState); err != nil {
		log.Error("Error saving messages from MessageExtractor to MessageConsumer", "err", err)
		return m.retryInterval, err
	}
	return 0, m.fsm.Do(processNextBlock{
		melState: saveAction.postState,
	})
}
