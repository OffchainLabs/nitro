// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/statetransfer"
)

func BuildBlock(
	statedb *state.StateDB,
	lastBlockHeader *types.Header,
	chainContext core.ChainContext,
	chainConfig *params.ChainConfig,
	inbox arbstate.InboxBackend,
	seqBatch []byte,
) (*types.Block, error) {
	var delayedMessagesRead uint64
	if lastBlockHeader != nil {
		delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
	}
	inboxMultiplexer := arbstate.NewInboxMultiplexer(inbox, delayedMessagesRead, nil, arbstate.KeysetValidate)

	ctx := context.Background()
	message, err := inboxMultiplexer.Pop(ctx)
	if err != nil {
		return nil, err
	}

	delayedMessagesRead = inboxMultiplexer.DelayedMessagesRead()
	l1Message := message.Message

	batchFetcher := func(uint64) ([]byte, error) {
		return seqBatch, nil
	}
	block, _, err := arbos.ProduceBlock(
		l1Message, delayedMessagesRead, lastBlockHeader, statedb, chainContext, chainConfig, batchFetcher,
	)
	return block, err
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

func (b *inboxBackend) ReadDelayedInbox(seqNum uint64) (*arbostypes.L1IncomingMessage, error) {
	if seqNum >= uint64(len(b.delayedMessages)) {
		return nil, errors.New("delayed inbox message out of bounds")
	}
	msg, err := arbostypes.ParseIncomingL1Message(bytes.NewReader(b.delayedMessages[seqNum]), nil)
	if err != nil {
		// The bridge won't generate an invalid L1 message,
		// so here we substitute it with a less invalid one for fuzzing.
		msg = &arbostypes.TestIncomingMessageWithRequestId
	}
	return msg, nil
}

// A chain context with no information
type noopChainContext struct{}

func (c noopChainContext) Engine() consensus.Engine {
	return nil
}

func (c noopChainContext) GetHeader(common.Hash, uint64) *types.Header {
	return nil
}

func FuzzStateTransition(f *testing.F) {
	f.Fuzz(func(t *testing.T, compressSeqMsg bool, seqMsg []byte, delayedMsg []byte) {
		chainDb := rawdb.NewMemoryDatabase()
		chainConfig := params.ArbitrumRollupGoerliTestnetChainConfig()
		serializedChainConfig, err := json.Marshal(chainConfig)
		if err != nil {
			panic(err)
		}
		stateRoot, err := arbosState.InitializeArbosInDatabase(
			chainDb,
			statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{}),
			chainConfig,
			serializedChainConfig,
			0,
			0,
		)
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
		delayedMessages := [][]byte{delayedMsg}
		seqBatch := make([]byte, 40)
		binary.BigEndian.PutUint64(seqBatch[8:16], ^uint64(0))
		binary.BigEndian.PutUint64(seqBatch[24:32], ^uint64(0))
		binary.BigEndian.PutUint64(seqBatch[32:40], uint64(len(delayedMessages)))
		if compressSeqMsg {
			seqBatch = append(seqBatch, arbstate.BrotliMessageHeaderByte)
			seqMsgCompressed, err := arbcompress.CompressFast(seqMsg)
			if err != nil {
				panic(fmt.Sprintf("failed to compress sequencer message: %v", err))
			}
			seqBatch = append(seqBatch, seqMsgCompressed...)
		} else {
			seqBatch = append(seqBatch, seqMsg...)
		}
		inbox := &inboxBackend{
			batchSeqNum:           0,
			batches:               [][]byte{seqBatch},
			positionWithinMessage: 0,
			delayedMessages:       delayedMessages,
		}
		_, err = BuildBlock(statedb, genesis, noopChainContext{}, params.ArbitrumOneChainConfig(), inbox, seqBatch)
		if err != nil {
			// With the fixed header it shouldn't be possible to read a delayed message,
			// and no other type of error should be possible.
			panic(err)
		}
	})
}
