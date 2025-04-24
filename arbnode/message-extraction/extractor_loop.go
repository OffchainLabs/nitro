package mel

import (
	"context"
	"fmt"
	"math/big"

	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/bold"
)

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
		melState := processAction.melState

		// TODO: Check the latest block number to see if it exists, otherwise, we just have to
		// repeat this FSM state until the new block exists.
		parentChainBlock, err := m.l1Reader.Client().BlockByNumber(
			ctx,
			new(big.Int).SetUint64(melState.ParentChainBlockNumber),
		)
		if err != nil {
			return err
		}
		postState, msgs, err := m.extractMessages(
			ctx,
			melState,
			parentChainBlock,
		)
		if err != nil {
			return err
		}
		return m.fsm.Do(saveMessages{
			messages: msgs,
			melState: postState,
		})
	case SavingMessages:
		// Persists messages and a processed MEL state to the database.
		saveAction, ok := current.SourceEvent.(saveMessages)
		if !ok {
			return fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		if err := m.melDB.SaveState(ctx, saveAction.melState, saveAction.messages); err != nil {
			return err
		}
		return m.fsm.Do(processNextBlock{
			melState: saveAction.melState,
		})
	default:
		return fmt.Errorf("invalid state: %s", current.State)
	}
}
