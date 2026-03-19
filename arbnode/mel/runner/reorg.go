// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

func (m *MessageExtractor) reorg(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	reorgAction, ok := current.SourceEvent.(reorgToOldBlock)
	if !ok {
		return m.config.RetryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
	}
	currentDirtyState := reorgAction.melState
	if currentDirtyState.ParentChainBlockNumber == 0 {
		return m.config.RetryInterval, errors.New("invalid reorging stage, ParentChainBlockNumber of current mel state has reached 0")
	}
	previousState, err := m.melDB.State(currentDirtyState.ParentChainBlockNumber - 1)
	if err != nil {
		return m.config.RetryInterval, err
	}
	if m.outboxTracker != nil {
		m.outboxTracker.TrimRight(currentDirtyState.ParentChainBlockNumber)
	}
	m.logsAndHeadersPreFetcher.reset()
	return 0, m.fsm.Do(processNextBlock{
		prevStepWasReorg: true,
		melState:         previousState,
	})
}
