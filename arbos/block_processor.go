//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/binary"
	"fmt"
	"math"
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
	"github.com/offchainlabs/arbstate/util"
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

// set by the precompile module, to avoid a package dependence cycle
var ArbRetryableTxAddress common.Address
var RedeemScheduledEventID common.Hash

func createNewHeader(prevHeader *types.Header, l1info *L1Info, statedb *state.StateDB) *types.Header {
	var lastBlockHash common.Hash
	blockNumber := big.NewInt(0)
	baseFee := OpenArbosState(statedb).GasPriceWei()
	timestamp := uint64(time.Now().Unix())
	coinbase := common.Address{}
	if l1info != nil {
		timestamp = l1info.l1Timestamp.Uint64()
		coinbase = l1info.poster
	}
	if prevHeader != nil {
		lastBlockHash = prevHeader.Hash()
		blockNumber.Add(prevHeader.Number, big.NewInt(1))
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
		BaseFee:     baseFee,
	}
}

func ProduceBlock(
	message *L1IncomingMessage,
	delayedMessagesRead uint64,
	lastBlockHeader *types.Header,
	statedb *state.StateDB,
	chainContext core.ChainContext,
) (*types.Block, types.Receipts) {

	txes, err := message.ParseL2Transactions(ChainConfig.ChainID)
	if err != nil {
		log.Warn("error parsing incoming message", "err", err)
		txes = types.Transactions{}
	}

	poster := message.Header.Poster

	l1Info := &L1Info{
		poster:        poster,
		l1BlockNumber: message.Header.BlockNumber.Big(),
		l1Timestamp:   message.Header.Timestamp.Big(),
	}

	state := OpenArbosState(statedb)
	gasLeft := state.CurrentPerBlockGasLimit()

	if lastBlockHeader == nil {
		state.timestamp.Set(uint64(time.Now().Unix()))
	}

	header := createNewHeader(lastBlockHeader, l1Info, statedb)
	signer := types.MakeSigner(ChainConfig, header.Number)

	complete := types.Transactions{}
	receipts := types.Receipts{}
	gasPrice := header.BaseFee
	time := header.Time

	redeems := types.Transactions{}

	// We'll check that the block can fit each message, so this pool is set to not run out
	gethGas := core.GasPool(1 << 63)

	for len(txes) > 0 || len(redeems) > 0 {

		state := OpenArbosState(statedb)
		retryableState := state.RetryableState()

		var tx *types.Transaction
		if len(redeems) > 0 {
			tx = redeems[0]
			redeems = redeems[1:]

			retry, ok := (tx.GetInner()).(*types.ArbitrumRetryTx)
			if !ok {
				panic("retryable tx is somehow not a retryable")
			}
			retryable := retryableState.OpenRetryable(retry.TicketId, time)
			if retryable == nil {
				// retryable was already deleted, so just refund the gas
				retryGas := new(big.Int).SetUint64(retry.Gas)
				gasGiven := new(big.Int).Mul(retryGas, gasPrice)
				refund := util.BigMulByFrac(gasGiven, 31, 32)
				statedb.AddBalance(retry.RefundTo, refund)
				continue
			}
		} else {
			tx = txes[0]
			txes = txes[1:]
		}

		sender, err := signer.Sender(tx)
		if err != nil {
			continue
		}

		aggregator := &poster

		if !isAggregated(*aggregator, sender) {
			aggregator = nil
		}

		var dataGas uint64 = 0
		if gasPrice.Sign() > 0 {
			dataGas = math.MaxUint64
			pricing := state.L1PricingState()
			posterCost := pricing.PosterDataCost(sender, aggregator, tx.Data())
			posterCostInL2Gas := new(big.Int).Div(posterCost, gasPrice)
			if posterCostInL2Gas.IsUint64() {
				dataGas = posterCostInL2Gas.Uint64()
			} else {
				log.Error("Could not get poster cost in L2 terms", posterCost, gasPrice)
			}
		}

		if dataGas > tx.Gas() {
			// this txn is going to be rejected later
			dataGas = 0
		}

		computeGas := tx.Gas() - dataGas

		if computeGas > gasLeft {
			continue
		}

		gasLeft -= computeGas

		snap := statedb.Snapshot()
		statedb.Prepare(tx.Hash(), len(txes))

		// We've checked that the block can fit this message, so we'll use a pool that won't run out
		gasPool := gethGas

		receipt, err := core.ApplyTransaction(
			ChainConfig,
			chainContext,
			&header.Coinbase,
			&gasPool,
			statedb,
			header,
			tx,
			&header.GasUsed,
			vm.Config{},
		)
		if err != nil {
			// Ignore this transaction if it's invalid under our more lenient state transaction function
			statedb.RevertToSnapshot(snap)
			continue
		}

		if gasPool > gethGas {
			delta := strconv.FormatUint(gasPool.Gas()-gethGas.Gas(), 10)
			panic("ApplyTransaction() gave back " + delta + " gas")
		}

		gasUsed := gethGas.Gas() - gasPool.Gas()
		gethGas = gasPool

		if gasUsed > computeGas {
			delta := strconv.FormatUint(gasUsed-computeGas, 10)
			panic("ApplyTransaction() used " + delta + " more gas than it should have")
		}

		for _, txLog := range receipt.Logs {
			if txLog.Address == ArbRetryableTxAddress && txLog.Topics[0] == RedeemScheduledEventID {

				ticketId := txLog.Topics[1]

				retryableState = OpenArbosState(statedb).RetryableState()
				retryable := retryableState.OpenRetryable(ticketId, time)

				reedem := types.NewTx(&types.ArbitrumRetryTx{
					ArbitrumContractTx: types.ArbitrumContractTx{
						ChainId:   ChainConfig.ChainID,
						RequestId: txLog.Topics[2],
						From:      retryable.From(),
						GasPrice:  gasPrice,
						Gas:       common.BytesToHash(txLog.Data[8:40]).Big().Uint64(),
						To:        retryable.To(),
						Value:     retryable.Callvalue(),
						Data:      retryable.Calldata(),
					},
					TicketId: ticketId,
					RefundTo: common.BytesToAddress(txLog.Data[40:72]),
				})

				redeems = append(redeems, reedem)
			}
		}

		complete = append(complete, tx)
		receipts = append(receipts, receipt)
		gasLeft -= gasUsed
	}

	binary.BigEndian.PutUint64(header.Nonce[:], delayedMessagesRead)
	header.Root = statedb.IntermediateRoot(true)

	// Touch up the block hashes in receipts
	tmpBlock := types.NewBlock(header, complete, nil, receipts, trie.NewStackTrie(nil))
	blockHash := tmpBlock.Hash()

	for _, receipt := range receipts {
		receipt.BlockHash = blockHash
		for _, txLog := range receipt.Logs {
			txLog.BlockHash = blockHash
		}
	}

	FinalizeBlock(header, complete, receipts, statedb)
	header.Root = statedb.IntermediateRoot(true)

	block := types.NewBlock(header, complete, nil, receipts, trie.NewStackTrie(nil))

	if len(block.Transactions()) != len(receipts) {
		panic(fmt.Sprintf("Block has %d txes but %d receipts", len(block.Transactions()), len(receipts)))
	}

	return block, receipts
}

func FinalizeBlock(header *types.Header, txs types.Transactions, receipts types.Receipts, statedb *state.StateDB) {
	if header != nil {
		state := OpenArbosState(statedb)
		state.SetLastTimestampSeen(header.Time)
		state.RetryableState().TryToReapOneRetryable(header.Time)

		maxSafePrice := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
		state.SetMaxGasPriceWei(maxSafePrice)

		// write send merkle accumulator hash into extra data field of the header
		header.Extra = state.SendMerkleAccumulator().Root().Bytes()
	}
}
