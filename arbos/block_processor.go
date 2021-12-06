//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"math/big"
	"strconv"
	"time"

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

	// Setup based on first segment
	blockInfo *L1Info
	header    *types.Header

	txes     types.Transactions
	receipts types.Receipts
	gasLeft  uint64
	gasLimit uint64
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
		gasLeft:         PerBlockGasLimit,
		gasLimit:        PerBlockGasLimit,
	}
}

// Must always return true if the block is empty
func (b *BlockBuilder) CanAddMessage(segment MessageSegment) bool {
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

	gasLeft := b.gasLeft
	for _, tx := range segment.Txes {
		if gasLeft < tx.Gas() {
			return false
		}
		gasLeft -= tx.Gas()
	}
	return true
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
	}

	signer := types.MakeSigner(ChainConfig, b.header.Number)

	for _, tx := range segment.Txes {

		sender, err := signer.Sender(tx)
		if err != nil {
			continue
		}

		aggregator := &segment.L1Info.l1Sender

		if !isAggregated(*aggregator, sender) {
			aggregator = nil
		}

		pricing := OpenArbosState(b.statedb).L1PricingState()
		dataGas := pricing.GetL1Charges(sender, aggregator, tx.Data()).Uint64()

		if dataGas > tx.Gas() {
			// this txn is going to be rejected later
			dataGas = 0
		}

		computeGas := tx.Gas() - dataGas

		if computeGas > b.gasLeft {
			continue
		}

		b.gasLeft -= computeGas

		snap := b.statedb.Snapshot()
		b.statedb.Prepare(tx.Hash(), len(b.txes))

		// We've checked that the block can fit this message, so we'll use a pool that won't run out
		gethGas := core.GasPool(1 << 63)
		gasPool := gethGas

		receipt, err := core.ApplyTransaction(
			ChainConfig,
			b.chainContext,
			&b.header.Coinbase,
			&gasPool,
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

		if gasPool > gethGas {
			delta := strconv.FormatUint(gasPool.Gas()-gethGas.Gas(), 10)
			panic("ApplyTransaction() gave back " + delta + " gas")
		}

		gasUsed := gethGas.Gas() - gasPool.Gas()

		if gasUsed > computeGas {
			delta := strconv.FormatUint(gasUsed-computeGas, 10)
			panic("ApplyTransaction() used " + delta + " more gas than it should have")
		}

		b.txes = append(b.txes, tx)
		b.receipts = append(b.receipts, receipt)
		b.gasLeft -= gasUsed
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

	// Reset the block builder for the next block
	receipts := b.receipts
	*b = *NewBlockBuilder(b.statedb, block.Header(), b.chainContext)
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
