package melrunner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

func (m *MessageExtractor) initialize(ctx context.Context, current *fsm.CurrentState[action, FSMState]) (time.Duration, error) {
	// Start from the latest MEL state we have in the database
	melState, err := m.melDB.GetHeadMelState(ctx)
	if err != nil {
		return m.config.RetryInterval, err
	}
	// Initialize delayedMessageBacklog and add it to the melState
	delayedMessageBacklog, err := mel.NewDelayedMessageBacklog(m.GetContext(), m.config.DelayedMessageBacklogCapacity, m.GetFinalizedDelayedMessagesRead)
	if err != nil {
		return m.config.RetryInterval, err
	}
	if err = InitializeDelayedMessageBacklog(ctx, delayedMessageBacklog, m.melDB, melState, m.GetFinalizedDelayedMessagesRead); err != nil {
		return m.config.RetryInterval, err
	}
	delayedMessageBacklog.CommitDirties()
	melState.SetDelayedMessageBacklog(delayedMessageBacklog)
	// Start mel state is now ready. Check if the state's parent chain block hash exists in the parent chain
	startBlock, err := m.parentChainReader.HeaderByNumber(ctx, new(big.Int).SetUint64(melState.ParentChainBlockNumber))
	if err != nil {
		return m.config.RetryInterval, fmt.Errorf("failed to get start parent chain block: %d corresponding to head mel state from parent chain: %w", melState.ParentChainBlockNumber, err)
	}
	// We check if our head mel state's parentChainBlockHash matches the one on-chain, if it doesnt then we detected a reorg
	if melState.ParentChainBlockHash != startBlock.Hash() {
		log.Info("MEL detected L1 reorg at the start", "block", melState.ParentChainBlockNumber, "parentChainBlockHash", melState.ParentChainBlockHash, "onchainParentChainBlockHash", startBlock.Hash()) // Log level is Info because L1 reorgs are a common occurrence
		return 0, m.fsm.Do(reorgToOldBlock{
			melState: melState,
		})
	}
	// Initialize logsPreFetcher
	m.logsPreFetcher = newLogsFetcher(m.parentChainReader, m.config.BlocksToPrefetch)
	// Begin the next FSM state immediately.
	return 0, m.fsm.Do(processNextBlock{
		melState: melState,
	})
}
