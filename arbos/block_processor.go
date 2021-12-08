//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"math/big"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var ChainConfig = &params.ChainConfig{
	ChainID:             big.NewInt(412345),
	HomesteadBlock:      big.NewInt(0),
	DAOForkBlock:        nil,
	DAOForkSupport:      true,
	EIP150Block:         big.NewInt(0),
	EIP150Hash:          common.Hash{},
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	IstanbulBlock:       big.NewInt(0),
	MuirGlacierBlock:    big.NewInt(0),
	BerlinBlock:         big.NewInt(0),
	LondonBlock:         big.NewInt(0),
	Arbitrum:            true,

	Clique: &params.CliqueConfig{
		Period: 0,
		Epoch:  0,
	},
}

type BlockBuilder struct {
	statedb         *state.StateDB
	lastBlockHeader *types.Header
	chainContext    core.ChainContext

	recordingStatedb      *state.StateDB
	recordingChainContext core.ChainContext
	recordingGasPool      core.GasPool
	recordingHeader       *types.Header
	recordingKeyValue     ethdb.KeyValueStore
	startPos              uint64 // recorded and not used

	// Setup based on first segment
	blockInfo *L1Info
	header    *types.Header
	gasPool   core.GasPool

	txes     types.Transactions
	receipts types.Receipts

	isDone bool
}

type BlockData struct {
	Txes   types.Transactions
	Header *types.Header
}

func NewRecordingBlockBuilder(lastBlockHeader *types.Header, statedb *state.StateDB, chainContext core.ChainContext, startPos uint64, recordingstateDb *state.StateDB, recordingChainContext core.ChainContext, recordingKeyValue ethdb.KeyValueStore) *BlockBuilder {
	return &BlockBuilder{
		statedb:               statedb,
		lastBlockHeader:       lastBlockHeader,
		chainContext:          chainContext,
		recordingStatedb:      recordingstateDb,
		recordingChainContext: recordingChainContext,
		recordingKeyValue:     recordingKeyValue,
		startPos:              startPos,
	}
}

func NewBlockBuilder(lastBlockHeader *types.Header, statedb *state.StateDB, chainContext core.ChainContext) *BlockBuilder {
	return NewRecordingBlockBuilder(lastBlockHeader, statedb, chainContext, 0, nil, nil, nil)
}

// Must always return true if the block is empty
func (b *BlockBuilder) CanAddMessage(segment MessageSegment) bool {
	if b.isDone {
		return false
	}
	if b.blockInfo == nil {
		return true
	}
	info := segment.L1Info
	// End current block without including segment
	// TODO: This would split up all delayed messages
	// If we distinguish between segments that might be aggregated from ones that definitely aren't
	// we could handle coinbases differently
	return info.l1Sender == b.blockInfo.l1Sender &&
		info.l1BlockNumber.Cmp(b.blockInfo.l1BlockNumber) <= 0 &&
		info.l1Timestamp.Cmp(b.blockInfo.l1Timestamp) <= 0
}

// Must always return true if the block is empty
func (b *BlockBuilder) ShouldAddMessage(segment MessageSegment) bool {
	if !b.CanAddMessage(segment) {
		return false
	}
	if b.blockInfo == nil {
		return true
	}
	newGasUsed := b.header.GasUsed
	for _, tx := range segment.Txes {
		oldGasUsed := newGasUsed
		newGasUsed += tx.Gas()
		if newGasUsed < oldGasUsed {
			newGasUsed = ^uint64(0)
		}
	}
	return newGasUsed <= PerBlockGasLimit
}

func createNewHeader(prevHeader *types.Header, l1info *L1Info) *types.Header {
	var lastBlockHash common.Hash
	blockNumber := big.NewInt(0)
	baseFee := big.NewInt(params.InitialBaseFee / 100)
	timestamp := uint64(time.Now().Unix())
	coinbase := common.Address{}
	if l1info != nil {
		timestamp = l1info.l1Timestamp.Uint64()
		coinbase = l1info.l1Sender
	}
	if prevHeader != nil {
		lastBlockHash = prevHeader.Hash()
		blockNumber.Add(prevHeader.Number, big.NewInt(1))
		baseFee = prevHeader.BaseFee
		if timestamp < prevHeader.Time {
			timestamp = prevHeader.Time
		}
	}
	return &types.Header{
		ParentHash:  lastBlockHash,
		UncleHash:   [32]byte{},
		Coinbase:    coinbase,
		Root:        [32]byte{},  // Filled in later
		TxHash:      [32]byte{},  // Filled in later
		ReceiptHash: [32]byte{},  // Filled in later
		Bloom:       [256]byte{}, // Filled in later
		Difficulty:  big.NewInt(1),
		Number:      blockNumber,
		GasLimit:    PerBlockGasLimit,
		GasUsed:     0,
		Time:        timestamp,
		Extra:       []byte{},   // Unused
		MixDigest:   [32]byte{}, // Unused
		Nonce:       [8]byte{},  // Filled in later
		BaseFee:     baseFee,    //TODO: parameter
	}
}

func (b *BlockBuilder) AddMessage(segment MessageSegment) {
	if !b.CanAddMessage(segment) {
		log.Warn("attempted to add incompatible message to block")
		return
	}
	if b.blockInfo == nil {
		l1Info := segment.L1Info
		b.blockInfo = &L1Info{
			l1Sender:      l1Info.l1Sender,
			l1BlockNumber: new(big.Int).Set(l1Info.l1BlockNumber),
			l1Timestamp:   new(big.Int).Set(l1Info.l1Timestamp),
		}

		b.header = createNewHeader(b.lastBlockHeader, b.blockInfo)
		b.gasPool = core.GasPool(b.header.GasLimit)
		if b.recordingStatedb != nil {
			b.recordingHeader = types.CopyHeader(b.header)
			b.recordingGasPool = b.gasPool
		}
	}

	for _, tx := range segment.Txes {
		if tx.Gas() > PerBlockGasLimit || tx.Gas() > b.gasPool.Gas() {
			// Ignore and transactions with higher than the max possible gas
			continue
		}
		snap := b.statedb.Snapshot()
		b.statedb.Prepare(tx.Hash(), len(b.txes))
		receipt, err := core.ApplyTransaction(
			ChainConfig,
			b.chainContext,
			&b.header.Coinbase,
			&b.gasPool,
			b.statedb,
			b.header,
			tx,
			&b.header.GasUsed,
			vm.Config{},
		)
		if err != nil {
			// Ignore this transaction if it's invalid under our more lenient state transaction function
			b.statedb.RevertToSnapshot(snap)
			continue
		}
		if b.recordingStatedb != nil {
			recReciept, err := core.ApplyTransaction(
				ChainConfig,
				b.recordingChainContext,
				&b.recordingHeader.Coinbase,
				&b.recordingGasPool,
				b.recordingStatedb,
				b.recordingHeader,
				tx,
				&b.recordingHeader.GasUsed,
				vm.Config{},
			)
			if (err != nil) || !reflect.DeepEqual(recReciept.Logs, receipt.Logs) || !reflect.DeepEqual(b.header, b.recordingHeader) {
				log.Error("recording transaction failed", "txhash", tx.Hash())
				b.recordingChainContext = nil
				b.recordingStatedb = nil
			}
		}
		b.txes = append(b.txes, tx)
		b.receipts = append(b.receipts, receipt)
	}
}

func (b *BlockBuilder) ConstructBlock(delayedMessagesRead uint64) (*types.Block, types.Receipts, *state.StateDB) {
	if b.header == nil {
		b.header = createNewHeader(b.lastBlockHeader, b.blockInfo)
	}

	binary.BigEndian.PutUint64(b.header.Nonce[:], delayedMessagesRead)
	b.header.Root = b.statedb.IntermediateRoot(true)

	// Touch up the block hashes in receipts
	tmpBlock := types.NewBlock(b.header, b.txes, nil, b.receipts, trie.NewStackTrie(nil))
	blockHash := tmpBlock.Hash()

	for _, receipt := range b.receipts {
		receipt.BlockHash = blockHash
		for _, txLog := range receipt.Logs {
			txLog.BlockHash = blockHash
		}
	}

	block := types.NewBlock(b.header, b.txes, nil, b.receipts, trie.NewStackTrie(nil))

	FinalizeBlock(b.header, b.txes, b.receipts, b.statedb)

	if b.recordingStatedb != nil {
		FinalizeBlock(b.recordingHeader, b.txes, b.receipts, b.recordingStatedb)
	}
	b.isDone = true
	// Reset the block builder for the next block
	receipts := b.receipts
	return block, receipts, b.statedb
}

func FinalizeBlock(header *types.Header, txs types.Transactions, receipts types.Receipts, statedb *state.StateDB) {
	if header != nil {
		state := OpenArbosState(statedb)
		state.SetLastTimestampSeen(header.Time)
		state.RetryableState().TryToReapOneRetryable(header.Time)

		// write send merkle accumulator hash into extra data field of the header
		header.Extra = state.SendMerkleAccumulator().Root().Bytes()
	}
}

func (b *BlockBuilder) StartPos() uint64 {
	return b.startPos
}

func (b *BlockBuilder) RecordingStateDB() *state.StateDB {
	return b.recordingStatedb
}

func (b *BlockBuilder) RecordingChainContext() core.ChainContext {
	return b.recordingChainContext
}

func (b *BlockBuilder) RecordingKeyValue() ethdb.KeyValueStore {
	return b.recordingKeyValue
}
