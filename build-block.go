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
	inboxReader := NewInboxReader(inbox, delayedMessagesRead)
	blockBuilder := arbos.NewBlockBuilder(statedb, lastBlockHeader, chainContext)
	for {
		message, shouldEndBlock, err := inboxReader.Peek()
		if err != nil {
			log.Warn("error parsing inbox message: %v", err)
			break
		}
		segments, err := arbos.ExtractL1MessageSegments(message, chainId)
		if !blockBuilder.ShouldAddMessage(segments) {
			break
		}
		inboxReader.Advance()
		blockBuilder.AddMessage(segments)
		if shouldEndBlock {
			break
		}
	}
	return blockBuilder.ConstructBlock(inboxReader.DelayedMessagesRead())
}
