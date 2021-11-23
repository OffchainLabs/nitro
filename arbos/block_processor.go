//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"math/big"
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
	gasPool   core.GasPool

	txes     types.Transactions
	receipts types.Receipts

	queuedRetries []retryables.QueuedRetry
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
	}

	arbosState := OpenArbosState(b.statedb)
	nextTxNum := 0
	for len(b.queuedRetries) > 0 || nextTxNum < len(segment.Txes) {
		var tx *types.Transaction
		if len(b.queuedRetries) > 0 {
			retry := b.queuedRetries[0]
			b.queuedRetries = b.queuedRetries[1:]
			tx = arbosState.RetryableState().MakeRetryTx(retry, b.header.Time, ChainConfig.ChainID, b.header.BaseFee)
			if tx == nil {
				// retryable was already deleted, so just refund the gas
				b.statedb.AddBalance(retry.RefundTo, new(big.Int).Mul(retry.Gas, b.header.BaseFee))
				continue
			}
			// add the retry tx's gas back into the gas pool
			b.gasPool.AddGas(retry.Gas.Uint64())
			arbosState.AddToGasPools(retry.Gas.Int64())
		} else {
			tx = segment.Txes[nextTxNum]
			nextTxNum++
		}
		if tx.Gas() > PerBlockGasLimit || tx.Gas() > b.gasPool.Gas() {
			// Ignore any transactions with higher than the max possible gas
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
		if tx.Type() == types.ArbitrumSubmitRetryableTxType && receipt.Status == 0 {
			arbTx, ok := tx.GetInner().(*types.ArbitrumSubmitRetryableTx)
			if !ok {
				panic("Tx of type ArbitrumSubmitRetryableTxType is somehow not an ")
			}
			arbosState.RetryableState().CreateRetryable(
				b.header.Time,
				tx.Hash(),
				b.header.Time+retryables.RetryableLifetimeSeconds,
				arbTx.From,
				tx.To(),
				tx.Value(),
				arbTx.Beneficiary,
				tx.Data(),
			)
		}
		retryTx, isRetry := tx.GetInner().(*types.ArbitrumRetryTx)
		if isRetry && receipt.Status == 1 {
			arbosState.RetryableState().DeleteRetryable(retryTx.TicketId)
			unusedGas := tx.Gas() - receipt.GasUsed
			if unusedGas > 0 {
				_ = b.gasPool.SubGas(unusedGas) // deliberately ignore error
				arbosState.AddToGasPools(-int64(unusedGas))
			}
		}
		b.txes = append(b.txes, tx)
		b.receipts = append(b.receipts, receipt)

		for _, txLog := range receipt.Logs {
			if txLog.Address == ArbRetryableTxAddress && txLog.Topics[0] == RedeemScheduledEventID {
				retry := retryables.QueuedRetry{
					TicketId: txLog.Topics[1],
					RetryId:  txLog.Topics[2],
					SeqNum:   common.BytesToHash(txLog.Data[:32]).Big().Uint64(),
					Gas:      common.BytesToHash(txLog.Data[32:64]).Big(),
					RefundTo: common.BytesToAddress(txLog.Data[64:96]),
				}
				b.queuedRetries = append(b.queuedRetries, retry)
			}
		}
	}
}

var ( // set by the precompile module, to avoid a package dependence cycle
	ArbRetryableTxAddress  common.Address
	RedeemScheduledEventID common.Hash
)

func (b *BlockBuilder) ConstructBlock(delayedMessagesRead uint64) (*types.Block, types.Receipts, *state.StateDB) {
	if b.header == nil {
		b.header = createNewHeader(b.lastBlockHeader, b.blockInfo)
	}

	binary.BigEndian.PutUint64(b.header.Nonce[:], delayedMessagesRead)

	FinalizeBlock(b.header, b.txes, b.receipts, b.statedb)
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
