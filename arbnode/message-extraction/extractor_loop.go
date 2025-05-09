package mel

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	extractionfunction "github.com/offchainlabs/nitro/arbnode/message-extraction/extraction-function"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/staker/bold"
)

var batchDeliveredID common.Hash
var messageDeliveredID common.Hash
var inboxMessageDeliveredID common.Hash
var inboxMessageFromOriginID common.Hash
var seqInboxABI *abi.ABI
var iBridgeABI *abi.ABI
var iInboxABI *abi.ABI
var iDelayedMessageProviderABI *abi.ABI

func init() {
	var err error
	sequencerBridgeABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	batchDeliveredID = sequencerBridgeABI.Events["SequencerBatchDelivered"].ID
	parsedIBridgeABI, err := bridgegen.IBridgeMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iBridgeABI = parsedIBridgeABI
	parsedIMessageProviderABI, err := bridgegen.IDelayedMessageProviderMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iDelayedMessageProviderABI = parsedIMessageProviderABI
	messageDeliveredID = parsedIBridgeABI.Events["MessageDelivered"].ID
	inboxMessageDeliveredID = parsedIMessageProviderABI.Events["InboxMessageDelivered"].ID
	inboxMessageFromOriginID = parsedIMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID
	seqInboxABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	parsedIInboxABI, err := bridgegen.IInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iInboxABI = parsedIInboxABI
}

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

func (m *MessageExtractor) wasBlockReorgedOut(ctx context.Context, parentChainBlockNumber uint64, parentChainBlockHash common.Hash) (bool, error) {
	header, err := m.l1Reader.Client().HeaderByNumber(ctx, new(big.Int).SetUint64(parentChainBlockNumber))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return true, nil
		}
		return false, err
	}
	return header.Hash() != parentChainBlockHash, nil
}

func (m *MessageExtractor) Act(ctx context.Context) error {
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
			return err
		}
		confirmedAssertionHash, err := rollup.LatestConfirmed(m.callOpts(ctx))
		if err != nil {
			return err
		}
		latestConfirmedAssertion, err := bold.ReadBoldAssertionCreationInfo(
			ctx,
			rollup,
			client,
			m.addrs.Rollup,
			confirmedAssertionHash,
		)
		if err != nil {
			return err
		}
		startBlock, err := client.HeaderByNumber(
			ctx,
			new(big.Int).SetUint64(latestConfirmedAssertion.CreationL1Block),
		)
		if err != nil {
			return err
		}
		melState, err := m.stateFetcher.GetState(
			ctx,
			startBlock.Hash(),
		)
		if err != nil {
			return err
		}
		return m.fsm.Do(processNextBlock{
			melState: melState,
		})
	case ReorgingToOldBlock:
		reorgAction, ok := current.SourceEvent.(reorgingToOldBlock)
		if !ok {
			return fmt.Errorf("invalid action: %T", current.SourceEvent)
		}

		// First find the block to reorg to
		// TODO: we need access to melstate via db here
		currentDirtyState := reorgAction.melState
		for {
			previousState, err := m.stateFetcher.GetState(
				ctx,
				currentDirtyState.ParentChainPreviousBlockHash,
			)
			if err != nil { // save rewind progress in case of errors
				return m.fsm.Do(reorgingToOldBlock{
					melState: currentDirtyState,
				})
			}

			// Check if parent mel state was reorged
			wasReorged, err := m.wasBlockReorgedOut(ctx, previousState.ParentChainBlockNumber, previousState.ParentChainBlockHash)
			if err != nil {
				return m.fsm.Do(reorgingToOldBlock{
					melState: currentDirtyState,
				})
			}

			// Clear dirty melState
			// TODO: batch clearing stale keys from db- mapping to melStates, delayed messages, sequencer messages etc... with ParentChainBlockNumber as key suffix. Ideally txStreamer would do this
			// m.melDB.DeleteStartingAt(currentDirtyState.ParentChainBlockNumber)
			if err := m.melDB.DeleteState(ctx, currentDirtyState.ParentChainBlockHash); err != nil {
				return m.fsm.Do(reorgingToOldBlock{
					melState: currentDirtyState,
				})
			}
			if wasReorged {
				currentDirtyState = previousState
				continue
			}

			// Found the block to reorg to
			return m.fsm.Do(processNextBlock{
				melState: previousState,
			})
		}
	case ProcessingNextBlock:
		// Process the next block in the parent chain and extracts messages.
		processAction, ok := current.SourceEvent.(processNextBlock)
		if !ok {
			return fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		preState := processAction.melState

		// Check for reorg
		wasReorged, err := m.wasBlockReorgedOut(ctx, preState.ParentChainBlockNumber, preState.ParentChainBlockHash)
		if err != nil {
			return err
		}
		if wasReorged {
			// Found a reorg, we move to ReorgingToOldBlock state to handle it
			return m.fsm.Do(reorgingToOldBlock{
				melState: preState,
			})
		}

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
				return m.fsm.Do(processNextBlock{
					melState: preState,
				})
			}
			return err
		}
		postState, msgs, delayedMsgs, err := extractionfunction.ExtractMessages(
			ctx,
			preState,
			parentChainBlock,
			m.dataProviders,
			m.melDB,
			m,
			&extractionfunction.BatchLookupParams{
				BatchDeliveredEventID: batchDeliveredID,
				SequencerInboxABI:     seqInboxABI,
			},
			&extractionfunction.DelayedMessageLookupParams{
				MessageDeliveredID:         messageDeliveredID,
				InboxMessageDeliveredID:    inboxMessageDeliveredID,
				InboxMessageFromOriginID:   inboxMessageFromOriginID,
				IDelayedMessageProviderABI: iDelayedMessageProviderABI,
				IBridgeABI:                 iBridgeABI,
				IInboxABI:                  iInboxABI,
			},
		)
		if err != nil {
			return err
		}
		return m.fsm.Do(saveMessages{
			postState:       postState,
			messages:        msgs,
			delayedMessages: delayedMsgs,
		})
	case SavingMessages:
		// Persists messages and a processed MEL state to the database.
		saveAction, ok := current.SourceEvent.(saveMessages)
		if !ok {
			return fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		// TODO: Make these database writes atomic, so if one fails, nothing
		// gets persisted and we retry.
		if err := m.melDB.SaveDelayedMessages(ctx, saveAction.postState, saveAction.delayedMessages); err != nil {
			return err
		}
		if err := m.melDB.SaveState(ctx, saveAction.postState, saveAction.messages); err != nil {
			return err
		}
		return m.fsm.Do(processNextBlock{
			melState: saveAction.postState,
		})
	default:
		return fmt.Errorf("invalid state: %s", current.State)
	}
}
