//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbos"
)

func BuildBlock(statedb *state.StateDB, lastBlockHeader *types.Header, chainContext core.ChainContext, inbox InboxBackend) (*types.Block, error) {
	var delayedMessagesRead uint64
	if lastBlockHeader != nil {
		delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
	}
	inboxMultiplexer := NewInboxMultiplexer(inbox, delayedMessagesRead)
	blockBuilder := arbos.NewBlockBuilder(statedb, lastBlockHeader, chainContext)
	for {
		message, err := inboxMultiplexer.Peek()
		if err != nil {
			return nil, err
		}
		segment, err := arbos.IncomingMessageToSegment(message.Message, arbos.ChainConfig.ChainID)
		if err != nil {
			log.Warn("error parsing incoming message", "err", err)
			err = inboxMultiplexer.Advance()
			if err != nil {
				return nil, err
			}
			break
		}
		// Always passes if the block is empty
		if !blockBuilder.ShouldAddMessage(segment) {
			break
		}
		err = inboxMultiplexer.Advance()
		if err != nil {
			return nil, err
		}
		blockBuilder.AddMessage(segment)
		if message.MustEndBlock {
			break
		}
	}
	block, _, _ := blockBuilder.ConstructBlock(inboxMultiplexer.DelayedMessagesRead())
	return block, nil
}
