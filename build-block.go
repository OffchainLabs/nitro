package arbstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbos"
)

var chainId = big.NewInt(0xA4B12) // TODO

func BuildBlock(statedb *state.StateDB, lastBlockHeader *types.Header, chainContext core.ChainContext, inbox InboxBackend) *types.Block {
	var delayedMessagesRead uint64
	if lastBlockHeader != nil {
		delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
	}
	inboxMultiplexer := NewInboxMultiplexer(inbox, delayedMessagesRead)
	blockBuilder := arbos.NewBlockBuilder(statedb, lastBlockHeader, chainContext)
	for {
		message, shouldEndBlock, err := inboxMultiplexer.Peek()
		if err != nil {
			log.Warn("error parsing inbox message: %v", err)
			inboxMultiplexer.Advance()
			break
		}
		segment, err := arbos.IncomingMessageToSegment(message, chainId)
		if err != nil {
			log.Warn("error parsing incoming message: %v", err)
			inboxMultiplexer.Advance()
			break
		}
		// Always passes if the block is empty
		if !blockBuilder.ShouldAddMessage(segment) {
			break
		}
		inboxMultiplexer.Advance()
		blockBuilder.AddMessage(segment)
		if shouldEndBlock {
			break
		}
	}
	return blockBuilder.ConstructBlock(inboxMultiplexer.DelayedMessagesRead())
}
