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
	targetBlock := currentDirtyState.ParentChainBlockNumber - 1
	if initialBlockNum, ok := m.melDB.InitialBlockNum(); ok && targetBlock < initialBlockNum {
		return m.config.RetryInterval, fmt.Errorf("reorg walked back to block %d which is below the MEL migration boundary %d; manual intervention required", targetBlock, initialBlockNum)
	}
	previousState, err := m.melDB.State(targetBlock)
	if err != nil {
		return m.config.RetryInterval, fmt.Errorf("reorg: failed to load MEL state for parent block %d: %w", targetBlock, err)
	}
	if err := m.melDB.RewriteHeadBlockNum(targetBlock); err != nil {
		return m.config.RetryInterval, fmt.Errorf("reorg: failed to rewrite head block num to %d: %w", targetBlock, err)
	}
	m.logsAndHeadersPreFetcher.reset()
	return 0, m.fsm.Do(processNextBlock{
		prevStepWasReorg: true,
		melState:         previousState,
	})
}
