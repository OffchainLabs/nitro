package arbos

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

func GetExtraSegmentToBeNextBlock() *MessageSegment {
	return nil
}

type BlockBuilder struct {
	statedb *state.StateDB
	lastBlockHeader *types.Header

	blockInfo *L1Info
	txes types.Transactions
}

type BlockData struct {
	Txes   types.Transactions
	Header *types.Header
}

func NewBlockBuilder(statedb *state.StateDB, lastBlockHeader *types.Header) *BlockBuilder {
	return &BlockBuilder{
		statedb:         statedb,
		lastBlockHeader: lastBlockHeader,
	}
}

func (b *BlockBuilder) AddSegment(segment *MessageSegment) *BlockData  {
	if b.blockInfo == nil {
		b.blockInfo = &segment.L1Info
	} else if segment.L1Info.l1Sender != b.blockInfo.l1Sender ||
		segment.L1Info.l1BlockNumber.Cmp(b.blockInfo.l1BlockNumber) > 0 ||
		segment.L1Info.l1Timestamp.Cmp(b.blockInfo.l1Timestamp) > 0{
		// End current block without including segment
		// TODO: This would split up all delayed messages
		// If we distinguish between segments that might be aggregated from ones that definitely aren't
		// we could handle coinbases differently
		return b.BuildBlockData()
	}
	b.txes = append(b.txes, segment.txes...)
	return nil
}

func (b *BlockBuilder) BuildBlockData() *BlockData {
	var lastBlockHash common.Hash
	timestamp := b.blockInfo.l1Timestamp.Uint64()
	blockNumber := big.NewInt(0)
	if b.lastBlockHeader != nil {
		lastBlockHash = b.lastBlockHeader.Hash()
		blockNumber.Add(b.lastBlockHeader.Number, big.NewInt(1))
		if timestamp < b.lastBlockHeader.Time {
			timestamp = b.lastBlockHeader.Time
		}
	}

	gasLimit := uint64(1e10) // TODO

	header := &types.Header{
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
		Nonce:       [8]byte{},  // Unused
		BaseFee:     new(big.Int),
	}
	return &BlockData{
		Txes:   b.txes,
		Header: header,
	}
}

func FinalizeBlock(header *types.Header, txs types.Transactions, receipts types.Receipts) {

}
