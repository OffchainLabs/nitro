package mel

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"

	extractionfunction "github.com/offchainlabs/nitro/arbnode/message-extraction/extraction-function"
)

func (m *MessageExtractor) ReceiptForTransactionIndex(
	ctx context.Context,
	parentChainBlock *types.Block,
	txIndex uint,
) (*types.Receipt, error) {
	tx, err := m.l1Reader.Client().TransactionInBlock(ctx, parentChainBlock.Hash(), txIndex)
	if err != nil {
		return nil, err
	}
	return m.l1Reader.Client().TransactionReceipt(ctx, tx.Hash())
}

func (m *MessageExtractor) CurrentFSMState() FSMState {
	return m.fsm.Current().State
}

func (m *MessageExtractor) Act(ctx context.Context) (time.Duration, error) {
	current := m.fsm.Current()
	switch current.State {
	case Start:
		// TODO: Start from the latest MEL state we have in the database if it exists as the first step.
		// Check if the specified start block hash exists in the parent chain.
		if _, err := m.l1Reader.Client().HeaderByHash(
			ctx,
			m.startParentChainBlockHash,
		); err != nil {
			return time.Second, fmt.Errorf(
				"failed to get start block by hash %s from parent chain: %w",
				m.startParentChainBlockHash,
				err,
			)
		}
		// Fetch the initial state for MEL from a state fetcher interface by parent chain block hash.
		melState, err := m.stateFetcher.GetState(
			ctx,
			m.startParentChainBlockHash,
		)
		if err != nil {
			return time.Second, err
		}
		return 0, m.fsm.Do(processNextBlock{
			melState: melState,
		})
	case ProcessingNextBlock:
		// Process the next block in the parent chain and extracts messages.
		processAction, ok := current.SourceEvent.(processNextBlock)
		if !ok {
			return time.Second, fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		preState := processAction.melState

		parentChainBlock, err := m.l1Reader.Client().BlockByNumber(
			ctx,
			new(big.Int).SetUint64(preState.ParentChainBlockNumber+1),
		)
		if err != nil {
			if err == ethereum.NotFound {
				// If the block with the specified number is not found, it likely has not
				// been posted yet to the parent chain, so we can retry after a short delay
				// without returning an error from the FSM.
				return time.Second, nil
			} else {
				return time.Second, err
			}
		}
		postState, msgs, delayedMsgs, err := extractionfunction.ExtractMessages(
			ctx,
			preState,
			parentChainBlock,
			m.dataProviders,
			m.melDB,
			m,
		)
		if err != nil {
			return time.Second, err
		}
		return 0, m.fsm.Do(saveMessages{
			postState:       postState,
			messages:        msgs,
			delayedMessages: delayedMsgs,
		})
	case SavingMessages:
		// Persists messages and a processed MEL state to the database.
		saveAction, ok := current.SourceEvent.(saveMessages)
		if !ok {
			return time.Second, fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		// TODO: Make these database writes atomic, so if one fails, nothing
		// gets persisted and we retry.
		if err := m.melDB.SaveDelayedMessages(ctx, saveAction.postState, saveAction.delayedMessages); err != nil {
			return time.Second, err
		}
		if err := m.melDB.SaveState(ctx, saveAction.postState, saveAction.messages); err != nil {
			return time.Second, err
		}
		return 0, m.fsm.Do(processNextBlock{
			melState: saveAction.postState,
		})
	default:
		return time.Second, fmt.Errorf("invalid state: %s", current.State)
	}
}
