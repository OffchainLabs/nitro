package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate"
	"github.com/offchainlabs/arbstate/wavmio"
)

func getBlockHeaderByHash(hash common.Hash) *types.Header {
	enc := wavmio.ResolvePreImage(hash)
	header := &types.Header{}
	err := rlp.DecodeBytes(enc, &header)
	if err != nil {
		panic(fmt.Sprintf("Error parsing resolved block header: %v", err))
	}
	return header
}

type WavmBlockRetriever struct {
	earliestResolvedHeader uint64
	knownBlockHeaders      map[uint64]*types.Header
}

func NewWavmBlockRetriever(lastBlockHash common.Hash) (*WavmBlockRetriever, *types.Header) {
	knownBlockHeaders := make(map[uint64]*types.Header)
	var earliestResolvedHeader uint64
	var lastBlockHeader *types.Header
	if lastBlockHash != (common.Hash{}) {
		lastBlockHeader = getBlockHeaderByHash(lastBlockHash)
		num := lastBlockHeader.Number.Uint64()
		knownBlockHeaders[num] = lastBlockHeader
		earliestResolvedHeader = num
	}
	return &WavmBlockRetriever{
		earliestResolvedHeader: earliestResolvedHeader,
		knownBlockHeaders:      knownBlockHeaders,
	}, lastBlockHeader
}

func (r *WavmBlockRetriever) GetBlockHash(num uint64) common.Hash {
	if num == 0 {
		return common.Hash{}
	}
	for ; r.earliestResolvedHeader > num; r.earliestResolvedHeader-- {
		lastHeader := r.knownBlockHeaders[r.earliestResolvedHeader]
		newHeader := getBlockHeaderByHash(lastHeader.ParentHash)
		r.knownBlockHeaders[newHeader.Number.Uint64()] = newHeader
	}
	return r.knownBlockHeaders[num].Hash()
}

func main() {
	rawdb := rawdb.NewDatabase(PreimageDb{})
	db := state.NewDatabase(rawdb)
	lastBlockHash := wavmio.GetLastBlockHash()

	inboxMessageBytes := wavmio.ReadInboxMessage()
	inboxMessageSegments, err := arbstate.SplitInboxMessageIntoSegments(inboxMessageBytes)
	if err != nil {
		fmt.Printf("Error splitting inbox message into segments: %v\n", err)
		wavmio.AdvanceInboxMessage()
		return
	}

	positionWithinMessage := wavmio.GetPositionWithinMessage()
	if positionWithinMessage+1 >= uint64(len(inboxMessageSegments)) {
		// This is the last segment in the message
		wavmio.AdvanceInboxMessage()
		wavmio.SetPositionWithinMessage(0)
		if positionWithinMessage >= uint64(len(inboxMessageSegments)) {
			// The message is empty
			return
		}
	} else {
		// There's remaining segment(s) in this message
		wavmio.SetPositionWithinMessage(positionWithinMessage + 1)
	}

	msg, err := arbstate.DecodeMessageSegment(inboxMessageSegments[positionWithinMessage])
	if err != nil {
		fmt.Printf("Error decoding message segment: %v\n", err)
		return
	}

	fmt.Printf("Previous block hash: %v\n", lastBlockHash)
	retriever, lastBlockHeader := NewWavmBlockRetriever(lastBlockHash)
	var lastBlockStateRoot common.Hash
	if lastBlockHeader != nil {
		lastBlockStateRoot = lastBlockHeader.Root
	}

	fmt.Printf("Previous block state root: %v\n", lastBlockStateRoot)
	statedb, err := state.New(lastBlockStateRoot, db, nil)
	if err != nil {
		panic(fmt.Sprintf("Error opening state db: %v", err))
	}

	fmt.Printf("Sender address: %v\n", msg.From.String())
	senderBalance := statedb.GetBalance(msg.From)
	fmt.Printf("Sender balance: %v\n", senderBalance.String())

	newBlockHeader, err := arbstate.Process(statedb, lastBlockHeader, retriever, msg)
	if err == nil {
		fmt.Printf("New state root: %v\n", newBlockHeader.Root)
		newBlockHash := newBlockHeader.Hash()
		fmt.Printf("New block hash: %v\n", newBlockHash)

		wavmio.SetLastBlockHash(newBlockHash)
	} else {
		fmt.Printf("Error processing message: %v\n", err)
	}

	senderBalance = statedb.GetBalance(msg.From)
	fmt.Printf("New sender balance: %v\n", senderBalance.String())
}
