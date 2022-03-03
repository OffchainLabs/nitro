//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package replay_fuzz

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/statetransfer"
)

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

// A simple mock inbox multiplexer backend
type inboxBackend struct {
	batchSeqNum           uint64
	batches               [][]byte
	positionWithinMessage uint64
	delayedMessages       [][]byte
}

func (b *inboxBackend) PeekSequencerInbox() ([]byte, error) {
	if len(b.batches) == 0 {
		return nil, errors.New("read past end of specified sequencer batches")
	}
	return b.batches[0], nil
}

func (b *inboxBackend) GetSequencerInboxPosition() uint64 {
	return b.batchSeqNum
}

func (b *inboxBackend) AdvanceSequencerInbox() {
	b.batchSeqNum++
	if len(b.batches) > 0 {
		b.batches = b.batches[1:]
	}
}

func (b *inboxBackend) GetPositionWithinMessage() uint64 {
	return b.positionWithinMessage
}

func (b *inboxBackend) SetPositionWithinMessage(pos uint64) {
	b.positionWithinMessage = pos
}

func (b *inboxBackend) ReadDelayedInbox(seqNum uint64) ([]byte, error) {
	if seqNum >= uint64(len(b.delayedMessages)) {
		return nil, errors.New("delayed inbox message out of bounds")
	}
	return b.delayedMessages[seqNum], nil
}

// A chain context with no information
type noopChainContext struct{}

func (c noopChainContext) Engine() consensus.Engine {
	return nil
}

func (c noopChainContext) GetHeader(common.Hash, uint64) *types.Header {
	return nil
}

func Fuzz(input []byte) int {
	chainDb := rawdb.NewMemoryDatabase()
	stateRoot, err := arbosState.InitializeArbosInDatabase(chainDb, statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{}), params.ArbitrumTestnetChainConfig())
	if err != nil {
		panic(err)
	}
	statedb, err := state.New(stateRoot, state.NewDatabase(chainDb), nil)
	if err != nil {
		panic(err)
	}
	genesis := &types.Header{
		Number:     new(big.Int),
		Nonce:      types.EncodeNonce(0),
		Time:       0,
		ParentHash: common.Hash{},
		Extra:      []byte("Arbitrum"),
		GasLimit:   l2pricing.GethBlockGasLimit,
		GasUsed:    0,
		BaseFee:    big.NewInt(l2pricing.InitialBaseFeeWei),
		Difficulty: big.NewInt(1),
		MixDigest:  common.Hash{},
		Coinbase:   common.Address{},
		Root:       stateRoot,
	}

	// Append a header to the input (this part is authenticated by L1).
	// The first 32 bytes encode timestamp and L1 block number bounds.
	// For simplicity, those are all set to 0.
	// The next 8 bytes encode the after delayed message count.
	delayedMessages := [][]byte{input}
	seqBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(seqBatch[32:], uint64(len(delayedMessages)))
	seqBatch = append(seqBatch, input...)
	inbox := &inboxBackend{
		batchSeqNum:           0,
		batches:               [][]byte{seqBatch},
		positionWithinMessage: 0,
		delayedMessages:       delayedMessages,
	}
	_, err = BuildBlock(statedb, genesis, noopChainContext{}, params.ArbitrumOneChainConfig(), inbox)
	if err != nil {
		// With the fixed header it shouldn't be possible to read a delayed message,
		// and no other type of error should be possible.
		panic(err)
	}

	return 0
}
