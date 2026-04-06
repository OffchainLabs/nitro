// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

func (m *MessageExtractor) initialize(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	// Start from the latest MEL state we have in the database.
	// State() already calls Validate(), so invariants are checked at load time.
	melState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return m.config.RetryInterval, err
	}
	if err := melState.RebuildDelayedMsgPreimages(m.melDB.FetchDelayedMessage); err != nil {
		return m.config.RetryInterval, fmt.Errorf("error rebuilding delayed msg preimages: %w", err)
	}
	// Start mel state is now ready. Check if the state's parent chain block hash exists in the parent chain
	startBlock, err := m.parentChainReader.HeaderByNumber(ctx, new(big.Int).SetUint64(melState.ParentChainBlockNumber))
	if err != nil {
		return m.config.RetryInterval, fmt.Errorf("failed to get start parent chain block: %d corresponding to head mel state from parent chain: %w", melState.ParentChainBlockNumber, err)
	}
	if startBlock == nil {
		return m.config.RetryInterval, fmt.Errorf("start parent chain block %d not found", melState.ParentChainBlockNumber)
	}
	// Initialize logsPreFetcher
	m.logsAndHeadersPreFetcher = newLogsAndHeadersFetcher(m.parentChainReader, m.config.BlocksToPrefetch)
	// We check if our head mel state's parentChainBlockHash matches the one on-chain, if it doesnt then we detected a reorg
	if melState.ParentChainBlockHash != startBlock.Hash() {
		log.Info("MEL detected L1 reorg at the start", "block", melState.ParentChainBlockNumber, "parentChainBlockHash", melState.ParentChainBlockHash, "onchainParentChainBlockHash", startBlock.Hash()) // Log level is Info because L1 reorgs are a common occurrence
		return 0, m.fsm.Do(reorgToOldBlock{
			melState: melState,
		})
	}
	// Begin the next FSM state immediately.
	return 0, m.fsm.Do(processNextBlock{
		melState: melState,
	})
}
