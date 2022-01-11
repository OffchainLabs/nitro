//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"fmt"
	"os"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbstate"
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

type WavmChainContext struct{}

func (c WavmChainContext) Engine() consensus.Engine {
	return arbos.Engine{}
}

func (c WavmChainContext) GetHeader(hash common.Hash, num uint64) *types.Header {
	header := getBlockHeaderByHash(hash)
	if !header.Number.IsUint64() || header.Number.Uint64() != num {
		panic(fmt.Sprintf("Retrieved wrong block number for header hash %v -- requested %v but got %v", hash, num, header.Number.String()))
	}
	return header
}

type WavmInbox struct{}

func (i WavmInbox) PeekSequencerInbox() ([]byte, error) {
	pos := wavmio.GetInboxPosition()
	res := wavmio.ReadInboxMessage(pos)
	log.Info("PeekSequencerInbox", "pos", pos, "res[:8]", res[:8])
	return res, nil
}

func (i WavmInbox) GetSequencerInboxPosition() uint64 {
	pos := wavmio.GetInboxPosition()
	log.Info("GetSequencerInboxPosition", "pos", pos)
	return pos
}

func (i WavmInbox) AdvanceSequencerInbox() {
	log.Info("AdvanceSequencerInbox")
	wavmio.AdvanceInboxMessage()
}

func (i WavmInbox) GetPositionWithinMessage() uint64 {
	pos := wavmio.GetPositionWithinMessage()
	log.Info("GetPositionWithinMessage", "pos", pos)
	return pos
}

func (i WavmInbox) SetPositionWithinMessage(pos uint64) {
	log.Info("SetPositionWithinMessage", "pos", pos)
	wavmio.SetPositionWithinMessage(pos)
}

func (i WavmInbox) ReadDelayedInbox(seqNum uint64) ([]byte, error) {
	log.Info("ReadDelayedMsg", "seqNum", seqNum)
	return wavmio.ReadDelayedInboxMessage(seqNum), nil
}

func BuildBlock(
	statedb *state.StateDB,
	lastBlockHeader *types.Header,
	chainContext core.ChainContext,
	chainConfig *params.ChainConfig,
	inbox arbstate.InboxBackend,
) (*types.Block, error) {
	var delayedMessagesRead uint64
	if lastBlockHeader != nil {
		delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
	}
	inboxMultiplexer := arbstate.NewInboxMultiplexer(inbox, delayedMessagesRead)

	message, err := inboxMultiplexer.Pop()
	if err != nil {
		return nil, err
	}

	delayedMessagesRead = inboxMultiplexer.DelayedMessagesRead()
	l1Message := message.Message

	block, _ := arbos.ProduceBlock(
		l1Message, delayedMessagesRead, lastBlockHeader, statedb, chainContext, chainConfig,
	)
	return block, nil
}

func main() {
	wavmio.StubInit()

	// We initialize the elliptic curve before calling into wavmio.
	// This allows the validator to cache the elliptic curve initialization.
	btcec.S256()

	raw := rawdb.NewDatabase(PreimageDb{})
	db := state.NewDatabase(raw)
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlError)
	log.Root().SetHandler(glogger)

	lastBlockHash := wavmio.GetLastBlockHash()

	var lastBlockHeader *types.Header
	var lastBlockStateRoot common.Hash
	if lastBlockHash != (common.Hash{}) {
		lastBlockHeader = getBlockHeaderByHash(lastBlockHash)
		lastBlockStateRoot = lastBlockHeader.Root
	}

	log.Info("Initial State", "lastBlockHash", lastBlockHash, "lastBlockStateRoot", lastBlockStateRoot)
	statedb, err := state.New(lastBlockStateRoot, db, nil)
	if err != nil {
		panic(fmt.Sprintf("Error opening state db: %v", err.Error()))
	}

	chainConfig := params.ArbitrumOneChainConfig()
	chainContext := WavmChainContext{}
	newBlock, err := BuildBlock(statedb, lastBlockHeader, chainContext, chainConfig, WavmInbox{})
	if err != nil {
		panic(fmt.Sprintf("Error building block: %v", err.Error()))
	}

	newBlockHash := newBlock.Hash()

	log.Info("Final State", "newBlockHash", newBlockHash, "StateRoot", newBlock.Root())

	wavmio.SetLastBlockHash(newBlockHash)
	arbstate := arbosState.OpenArbosState(statedb)
	wavmio.SetSendRoot(arbstate.SendMerkleAccumulator().Root())

	wavmio.StubFinal()
}
