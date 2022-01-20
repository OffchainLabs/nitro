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

	"github.com/offchainlabs/arbstate/arbos/arbosState"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// set by the precompile module, to avoid a package dependence cycle
var ArbRetryableTxAddress common.Address
var RedeemScheduledEventID common.Hash

func createNewHeader(prevHeader *types.Header, l1info *L1Info, state *arbosState.ArbosState) *types.Header {
	baseFee, err := state.GasPriceWei()
	state.Restrict(err)

	var lastBlockHash common.Hash
	blockNumber := big.NewInt(0)
	timestamp := uint64(0)
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
		GasLimit:    arbosState.PerBlockGasLimit,
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
	chainConfig *params.ChainConfig,
) (*types.Block, types.Receipts) {

	txes, err := message.ParseL2Transactions(chainConfig.ChainID)
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

	state := arbosState.OpenSystemArbosState(statedb)
	_ = state.Blockhashes().RecordNewL1Block(l1Info.l1BlockNumber.Uint64(), lastBlockHeader.Hash())
	gasLeft, _ := state.CurrentPerBlockGasLimit()
	header := createNewHeader(lastBlockHeader, l1Info, state)
	signer := types.MakeSigner(chainConfig, header.Number)

	complete := types.Transactions{}
	receipts := types.Receipts{}
	gasPrice := header.BaseFee
	time := header.Time

	redeems := types.Transactions{}

	// We'll check that the block can fit each message, so this pool is set to not run out
	gethGas := core.GasPool(1 << 63)

	for len(txes) > 0 || len(redeems) > 0 {
		// repeatedly process the next tx, doing redeems created along the way in FIFO order
		retryableState := state.RetryableState()

		var tx *types.Transaction
		if len(redeems) > 0 {
			tx = redeems[0]
			redeems = redeems[1:]

			retry, ok := (tx.GetInner()).(*types.ArbitrumRetryTx)
			if !ok {
				panic("retryable tx is somehow not a retryable")
			}
			retryable, _ := retryableState.OpenRetryable(retry.TicketId, time)
			if retryable == nil {
				// retryable was already deleted, so just refund the gas
				retryGas := new(big.Int).SetUint64(retry.Gas)
				gasGiven := new(big.Int).Mul(retryGas, gasPrice)
				statedb.AddBalance(retry.RefundTo, gasGiven)
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
			posterCost, _ := pricing.PosterDataCost(sender, aggregator, tx.Data())
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

		snap := statedb.Snapshot()
		statedb.Prepare(tx.Hash(), len(receipts)) // the number of successful state transitions

		gasLeft -= computeGas
		gasPool := gethGas

		receipt, err := core.ApplyTransaction(
			chainConfig,
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
				retryable, _ := state.RetryableState().OpenRetryable(ticketId, time)

				from, _ := retryable.From()
				to, _ := retryable.To()
				value, _ := retryable.Callvalue()
				data, _ := retryable.Calldata()

				reedem := types.NewTx(&types.ArbitrumRetryTx{
					ArbitrumContractTx: types.ArbitrumContractTx{
						ChainId:   chainConfig.ChainID,
						RequestId: txLog.Topics[2],
						From:      from,
						GasPrice:  gasPrice,
						Gas:       common.BytesToHash(txLog.Data[32:64]).Big().Uint64(),
						To:        to,
						Value:     value,
						Data:      data,
					},
					TicketId: ticketId,
					RefundTo: common.BytesToAddress(txLog.Data[64:96]),
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

	state.UpgradeArbosVersionIfNecessary(header.Time)

	FinalizeBlock(header, complete, receipts, statedb)
	header.Root = statedb.IntermediateRoot(true)

	block := types.NewBlock(header, complete, nil, receipts, trie.NewStackTrie(nil))

	if len(block.Transactions()) != len(receipts) {
		panic(fmt.Sprintf("Block has %d txes but %d receipts", len(block.Transactions()), len(receipts)))
	}

	return block, receipts
}

type HeaderExtraInformation struct {
	SendRoot  common.Hash
	SendCount uint64
}

func (extra HeaderExtraInformation) Serialize() []byte {
	out := [32 + 8]byte{}
	copy(out[:], extra.SendRoot[:])
	binary.BigEndian.PutUint64(out[32:], extra.SendCount)
	return out[:]
}

func DeserializeHeaderExtraInformation(header *types.Header) (HeaderExtraInformation, error) {
	if header.Number.Sign() == 0 || len(header.Extra) == 0 {
		// The genesis block doesn't have an ArbOS encoded extra field
		return HeaderExtraInformation{}, nil
	}
	if len(header.Extra) != 40 {
		return HeaderExtraInformation{}, fmt.Errorf("unexpected header extra field length %v", len(header.Extra))
	}
	extra := HeaderExtraInformation{}
	copy(extra.SendRoot[:], header.Extra)
	extra.SendCount = binary.BigEndian.Uint64(header.Extra[32:])
	return extra, nil
}

func FinalizeBlock(header *types.Header, txs types.Transactions, receipts types.Receipts, statedb *state.StateDB) {
	if header != nil {
		state := arbosState.OpenSystemArbosState(statedb)
		state.SetLastTimestampSeen(header.Time)
		_ = state.RetryableState().TryToReapOneRetryable(header.Time)

		maxSafePrice := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
		state.SetMaxGasPriceWei(maxSafePrice)

		// Add outbox info to the header for client-side proving
		acc := state.SendMerkleAccumulator()
		root, _ := acc.Root()
		size, _ := acc.Size()
		header.Extra = HeaderExtraInformation{root, size}.Serialize()
	}
}
