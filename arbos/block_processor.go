//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
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

	Clique: &params.CliqueConfig{
		Period: 0,
		Epoch:  0,
	},
}

type BlockBuilder struct {
	statedb         *state.StateDB
	lastBlockHeader *types.Header
	chainContext    core.ChainContext

	// Setup based on first storage
	blockInfo *L1Info
	header    *types.Header
	gasPool   core.GasPool

	txes     types.Transactions
	receipts types.Receipts
}

type BlockData struct {
	Txes   types.Transactions
	Header *types.Header
}

func NewBlockBuilder(statedb *state.StateDB, lastBlockHeader *types.Header, chainContext core.ChainContext) *BlockBuilder {
	return &BlockBuilder{
		statedb:         statedb,
		lastBlockHeader: lastBlockHeader,
		chainContext:    chainContext,
	}
}

// Must always return true if the block is empty
func (b *BlockBuilder) CanAddMessage(segment MessageSegment) bool {
	if b.blockInfo == nil {
		return true
	}
	info := segment.L1Info
	// End current block without including storage
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

func (b *BlockBuilder) AddMessage(segment MessageSegment) {
	if !b.CanAddMessage(segment) {
		log.Warn("attempted to add incompatible message to block")
		return
	}
	if b.blockInfo == nil {
		l1Info := segment.L1Info
		l1Sender := l1Info.l1Sender
		timestamp := l1Info.l1Timestamp.Uint64()
		l1BlockNumber := l1Info.l1BlockNumber.Uint64()
		var lastBlockHash common.Hash
		blockNumber := big.NewInt(0)
		if b.lastBlockHeader != nil {
			lastBlockHash = b.lastBlockHeader.Hash()
			blockNumber.Add(b.lastBlockHeader.Number, big.NewInt(1))
			if timestamp < b.lastBlockHeader.Time {
				timestamp = b.lastBlockHeader.Time
			}
			// TODO ensure l1BlockNumber is non-decreasing
		}
		b.blockInfo = &L1Info{
			l1Sender:      l1Sender,
			l1BlockNumber: new(big.Int).SetUint64(l1BlockNumber),
			l1Timestamp:   new(big.Int).SetUint64(timestamp),
		}

		gasLimit := PerBlockGasLimit

		b.header = &types.Header{
			ParentHash:  lastBlockHash,
			UncleHash:   [32]byte{},
			Coinbase:    b.blockInfo.l1Sender,
			Root:        [32]byte{},  // Filled in later
			TxHash:      [32]byte{},  // Filled in later
			ReceiptHash: [32]byte{},  // Filled in later
			Bloom:       [256]byte{}, // Filled in later
			Difficulty:  big.NewInt(1),
			Number:      blockNumber,
			GasLimit:    gasLimit,
			GasUsed:     0, // Filled in later
			Time:        timestamp,
			Extra:       []byte{},   // Unused
			MixDigest:   [32]byte{}, // Unused
			Nonce:       [8]byte{},  // Filled in later
			BaseFee:     new(big.Int),
		}
		b.gasPool = core.GasPool(b.header.GasLimit)
	}

	for _, tx := range segment.Txes {
		if tx.Gas() > PerBlockGasLimit || tx.Gas() > b.gasPool.Gas() {
			// Ignore and transactions with higher than the max possible gas
			continue
		}
		snap := b.statedb.Snapshot()
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
		b.txes = append(b.txes, tx)
		b.receipts = append(b.receipts, receipt)
	}
}

func (b *BlockBuilder) ConstructBlock(delayedMessagesRead uint64) *types.Block {
	if b.header == nil {
		var lastBlockHash common.Hash
		blockNumber := big.NewInt(0)
		if b.lastBlockHeader != nil {
			lastBlockHash = b.lastBlockHeader.Hash()
			blockNumber.Add(b.lastBlockHeader.Number, big.NewInt(1))
		}
		b.header = &types.Header{
			ParentHash:  lastBlockHash,
			UncleHash:   [32]byte{},
			Coinbase:    b.blockInfo.l1Sender,
			Root:        [32]byte{},  // Filled in later
			TxHash:      [32]byte{},  // Filled in later
			ReceiptHash: [32]byte{},  // Filled in later
			Bloom:       [256]byte{}, // Filled in later
			Difficulty:  big.NewInt(1),
			Number:      blockNumber,
			GasLimit:    PerBlockGasLimit,
			GasUsed:     0,
			Time:        b.lastBlockHeader.Time,
			Extra:       []byte{},   // Unused
			MixDigest:   [32]byte{}, // Unused
			Nonce:       [8]byte{},  // Filled in later
			BaseFee:     new(big.Int),
		}
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

	FinalizeBlock(b.header, b.txes, b.receipts, b.statedb, b.chainContext)

	return types.NewBlock(b.header, b.txes, nil, b.receipts, trie.NewStackTrie(nil))
}

func FinalizeBlock(
	header *types.Header,
	txs types.Transactions,
	receipts types.Receipts,
	statedb *state.StateDB,
	chainContext core.ChainContext, // should be nil if there is no previous block
) {
	arbosState := OpenArbosState(statedb)
	arbosState.TryToReapOneRetryable()
}
