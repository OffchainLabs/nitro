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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/burn"
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

	var delayedMessagesRead uint64
	if lastBlockHeader != nil {
		delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
	}
	inboxMultiplexer := arbstate.NewInboxMultiplexer(WavmInbox{}, delayedMessagesRead)
	messageWithMetadata, err := inboxMultiplexer.Pop()
	if err != nil {
		panic(fmt.Sprintf("Error reading from inbox multiplexer: %v", err.Error()))
	}
	delayedMessagesRead = inboxMultiplexer.DelayedMessagesRead()
	message := messageWithMetadata.Message

	var newBlock *types.Block
	if lastBlockStateRoot != (common.Hash{}) {
		// ArbOS has already been initialized.
		// Load the chain config and then produce a block normally.

		initialArbosState, err := arbosState.OpenSystemArbosState(statedb, true)
		if err != nil {
			panic(fmt.Sprintf("Error opening initial ArbOS state: %v", err.Error()))
		}
		chainId, err := initialArbosState.ChainId()
		if err != nil {
			panic(fmt.Sprintf("Error getting chain ID from initial ArbOS state: %v", err.Error()))
		}
		chainConfig, err := arbos.GetChainConfig(chainId)
		if err != nil {
			panic(err)
		}

		chainContext := WavmChainContext{}
		newBlock, _ = arbos.ProduceBlock(message, delayedMessagesRead, lastBlockHeader, statedb, chainContext, chainConfig)

	} else {
		// Initialize ArbOS with this init message and create the genesis block.

		chainId, err := message.ParseInitMessage()
		if err != nil {
			panic(err)
		}
		chainConfig, err := arbos.GetChainConfig(chainId)
		if err != nil {
			panic(err)
		}
		_, err = arbosState.InitializeArbosState(statedb, burn.NewSystemBurner(false), chainConfig)
		if err != nil {
			panic(fmt.Sprintf("Error initializing ArbOS: %v", err.Error()))
		}

		newBlock = arbosState.MakeGenesisBlock(common.Hash{}, 0, 0, statedb.IntermediateRoot(true))

	}

	newBlockHash := newBlock.Hash()

	log.Info("Final State", "newBlockHash", newBlockHash, "StateRoot", newBlock.Root())

	extraInfo, err := types.DeserializeHeaderExtraInformation(newBlock.Header())
	if err != nil {
		panic(fmt.Sprintf("Error deserializing header extra info: %v", err))
	}
	wavmio.SetLastBlockHash(newBlockHash)
	wavmio.SetSendRoot(extraInfo.SendRoot)

	wavmio.StubFinal()
}
