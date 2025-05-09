package mel

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	extractionfunction "github.com/offchainlabs/nitro/arbnode/message-extraction/extraction-function"
	"github.com/offchainlabs/nitro/staker/bold"
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
		// Fetch the initial state for the FSM from a state fetcher and
		// read the first parent chain block we should process.
		// TODO: Start from the latest MEL state we have in the database if it exists
		// instead of the latest confirmed assertion.
		client := m.l1Reader.Client()
		rollup, err := rollupgen.NewRollupUserLogic(m.addrs.Rollup, client)
		if err != nil {
			return time.Second, err
		}
		confirmedAssertionHash, err := rollup.LatestConfirmed(m.callOpts(ctx))
		if err != nil {
			return time.Second, err
		}
		latestConfirmedAssertion, err := bold.ReadBoldAssertionCreationInfo(
			ctx,
			rollup,
			client,
			m.addrs.Rollup,
			confirmedAssertionHash,
		)
		if err != nil {
			return time.Second, err
		}
		startBlock, err := client.HeaderByNumber(
			ctx,
			new(big.Int).SetUint64(latestConfirmedAssertion.CreationL1Block),
		)
		if err != nil {
			return time.Second, err
		}
		melState, err := m.stateFetcher.GetState(
			ctx,
			startBlock.Hash(),
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

		// TODO: Check the latest block number to see if it exists, otherwise, we just have to
		// repeat this FSM state until the new block exists.
		parentChainBlock, err := m.l1Reader.Client().BlockByNumber(
			ctx,
			new(big.Int).SetUint64(preState.ParentChainBlockNumber+1),
		)
		if err != nil {
			// TODO: Additionally return a duration from this function so that a stop waiter
			// can know how long before it retries. This gives us more control of how often
			// we want to retry a state in the FSM.
			if strings.Contains(err.Error(), "not found") {
				return time.Second, m.fsm.Do(processNextBlock{
					melState: preState,
				})
			}
			return time.Second, err
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
