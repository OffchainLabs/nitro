package mel

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/arbnode"
	extractionfunction "github.com/offchainlabs/nitro/arbnode/message-extraction/extraction-function"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/staker/bold"
)

var batchDeliveredID common.Hash
var messageDeliveredID common.Hash
var inboxMessageDeliveredID common.Hash
var inboxMessageFromOriginID common.Hash
var l2MessageFromOriginCallABI abi.Method

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
	parsedIMessageProviderABI, err := bridgegen.IDelayedMessageProviderMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	messageDeliveredID = parsedIBridgeABI.Events["MessageDelivered"].ID
	inboxMessageDeliveredID = parsedIMessageProviderABI.Events["InboxMessageDelivered"].ID
	inboxMessageFromOriginID = parsedIMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID

	parsedIInboxABI, err := bridgegen.IInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	l2MessageFromOriginCallABI = parsedIInboxABI.Methods["sendL2MessageFromOrigin"]
}

type BatchSerializer interface {
	Serialize(
		ctx context.Context,
		batch *arbnode.SequencerInboxBatch,
	) ([]byte, error)
}

func (m *MessageExtractor) Serialize(ctx context.Context, batch *arbnode.SequencerInboxBatch) ([]byte, error) {
	return batch.Serialize(ctx, m.l1Reader.Client())
}

func (m *MessageExtractor) CurrentFSMState() FSMState {
	return m.fsm.Current().State
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
	case ProcessingNextBlock:
		// Process the next block in the parent chain and extracts messages.
		processAction, ok := current.SourceEvent.(processNextBlock)
		if !ok {
			return fmt.Errorf("invalid action: %T", current.SourceEvent)
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
			m.delayedBridge,
			m.melDB,
			m.sequencerInboxBindings,
			m.delayedBridgeBindings,
			m.l1Reader.Client(),
			m,
			batchDeliveredID,
			messageDeliveredID,
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
