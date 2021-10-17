package arbos

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var perBlockGasLimit uint64 = 20000000

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

	// Setup based on first segment
	blockInfo *L1Info
	header    *types.Header
	gasPool   core.GasPool

	txes         types.Transactions
	receipts     types.Receipts
	pendingBlock *types.Block
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
		pendingBlock:    nil,
	}
}

// AddSegment returns true if Segment is done
func (b *BlockBuilder) AddSegment(segment *MessageSegment) bool {
	if b.pendingBlock != nil {
		return false
	}
	startIndex := uint64(0)
	if b.blockInfo == nil {
		b.blockInfo = &segment.L1Info
		var lastBlockHash common.Hash
		timestamp := b.blockInfo.l1Timestamp.Uint64()
		blockNumber := big.NewInt(0)
		if b.lastBlockHeader != nil {
			lastBlockHash = b.lastBlockHeader.Hash()
			blockNumber.Add(b.lastBlockHeader.Number, big.NewInt(1))
			if timestamp < b.lastBlockHeader.Time {
				timestamp = b.lastBlockHeader.Time
			}
			startIndex = binary.BigEndian.Uint64(b.lastBlockHeader.Nonce[:])
		}

		gasLimit := perBlockGasLimit // TODO

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
			Nonce:       [8]byte{},  // Unused
			BaseFee:     new(big.Int),
		}
		b.gasPool = core.GasPool(b.header.GasLimit)
	} else if segment.L1Info.l1Sender != b.blockInfo.l1Sender ||
		segment.L1Info.l1BlockNumber.Cmp(b.blockInfo.l1BlockNumber) > 0 ||
		segment.L1Info.l1Timestamp.Cmp(b.blockInfo.l1Timestamp) > 0 {
		// End current block without including segment
		// TODO: This would split up all delayed messages
		// If we distinguish between segments that might be aggregated from ones that definitely aren't
		// we could handle coinbases differently
		b.pendingBlock = b.ConstructBlock(0)
		return false
	}

	for i, tx := range segment.txes[startIndex:] {
		if tx.Gas() > perBlockGasLimit {
			// Ignore and transactions with higher than the max possible gas
			continue
		}
		if tx.Gas() > b.gasPool.Gas() {
			b.pendingBlock = b.ConstructBlock(startIndex + uint64(i))
			return false
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
	return true
}

func (b *BlockBuilder) PendingBlock() (*types.Block, *state.StateDB, types.Receipts) {
	if b.pendingBlock == nil {
		return nil, nil, types.Receipts{}
	}
	return b.pendingBlock, b.statedb, b.receipts
}

func (b *BlockBuilder) ConstructBlock(nextIndexToRead uint64) *types.Block {
	binary.BigEndian.PutUint64(b.header.Nonce[:], nextIndexToRead)
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

	FinalizeBlock(b.header, b.txes, b.receipts)

	return types.NewBlock(b.header, b.txes, nil, b.receipts, trie.NewStackTrie(nil))
}

func FinalizeBlock(header *types.Header, txs types.Transactions, receipts types.Receipts) {

}
