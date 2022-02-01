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
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"

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
var ArbSysAddress common.Address
var RedeemScheduledEventID common.Hash
var L2ToL1TransactionEventID common.Hash
var EmitReedeemScheduledEvent func(*vm.EVM, uint64, uint64, [32]byte, [32]byte, common.Address) error
var EmitTicketCreatedEvent func(*vm.EVM, [32]byte) error

func createNewHeader(prevHeader *types.Header, l1info *L1Info, state *arbosState.ArbosState) *types.Header {
	l2Pricing := state.L2PricingState()
	baseFee, err := l2Pricing.GasPriceWei()
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
		GasLimit:    l2pricing.L2GasLimit,
		GasUsed:     0,
		Time:        timestamp,
		Extra:       []byte{},   // Unused
		MixDigest:   [32]byte{}, // Unused
		Nonce:       [8]byte{},  // Filled in later
		BaseFee:     baseFee,
	}
}

type SequencingHooks struct {
	TxErrors     []error
	PreTxFilter  func(*arbosState.ArbosState, *types.Transaction, common.Address) error
	PostTxFilter func(*arbosState.ArbosState, *types.Transaction, common.Address, uint64, *types.Receipt) error
}

func noopSequencingHooks() *SequencingHooks {
	return &SequencingHooks{
		[]error{},
		func(*arbosState.ArbosState, *types.Transaction, common.Address) error {
			return nil
		},
		func(*arbosState.ArbosState, *types.Transaction, common.Address, uint64, *types.Receipt) error {
			return nil
		},
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

	hooks := noopSequencingHooks()
	return ProduceBlockAdvanced(
		message.Header, txes, delayedMessagesRead, lastBlockHeader, statedb, chainContext, chainConfig, hooks,
	)
}

// A bit more flexible than ProduceBlock for use in the sequencer.
func ProduceBlockAdvanced(
	messageHeader *L1IncomingMessageHeader,
	txes types.Transactions,
	delayedMessagesRead uint64,
	lastBlockHeader *types.Header,
	statedb *state.StateDB,
	chainContext core.ChainContext,
	chainConfig *params.ChainConfig,
	sequencingHooks *SequencingHooks,
) (*types.Block, types.Receipts) {

	state, err := arbosState.OpenSystemArbosState(statedb, true)
	if err != nil {
		panic(err)
	}

	if statedb.GetTotalBalanceDelta().BitLen() != 0 {
		panic("ProduceBlock called with dirty StateDB (non-zero total balance delta)")
	}

	poster := messageHeader.Poster

	l1Info := &L1Info{
		poster:        poster,
		l1BlockNumber: messageHeader.BlockNumber.Big(),
		l1Timestamp:   messageHeader.Timestamp.Big(),
	}

	gasLeft, _ := state.L2PricingState().PerBlockGasLimit()
	header := createNewHeader(lastBlockHeader, l1Info, state)
	signer := types.MakeSigner(chainConfig, header.Number)
	nextL1BlockNumber, _ := state.Blockhashes().NextBlockNumber()
	if l1Info.l1BlockNumber.Uint64() >= nextL1BlockNumber {
		// Make an ArbitrumInternalTx the first tx to update the L1 block number
		// Note: 0 is the TxIndex. If this transaction is ever not the first, that needs updated.
		tx := InternalTxUpdateL1BlockNumber(chainConfig.ChainID, l1Info.l1BlockNumber, header.Number, 0)
		txes = append([]*types.Transaction{types.NewTx(tx)}, txes...)
	}

	complete := types.Transactions{}
	receipts := types.Receipts{}
	gasPrice := header.BaseFee
	time := header.Time
	expectedBalanceDelta := new(big.Int)
	redeems := types.Transactions{}

	// We'll check that the block can fit each message, so this pool is set to not run out
	gethGas := core.GasPool(1 << 63)

	for len(txes) > 0 || len(redeems) > 0 {
		// repeatedly process the next tx, doing redeems created along the way in FIFO order

		var tx *types.Transaction
		hooks := noopSequencingHooks()
		if len(redeems) > 0 {
			tx = redeems[0]
			redeems = redeems[1:]

			retry, ok := (tx.GetInner()).(*types.ArbitrumRetryTx)
			if !ok {
				panic("retryable tx is somehow not a retryable")
			}
			retryable, _ := state.RetryableState().OpenRetryable(retry.TicketId, time)
			if retryable == nil {
				// retryable was already deleted
				continue
			}
		} else {
			tx = txes[0]
			txes = txes[1:]
			if tx.Type() != types.ArbitrumInternalTxType {
				// the sequencer has the ability to drop this tx
				hooks = sequencingHooks
			}
		}

		var sender common.Address
		var dataGas uint64 = 0
		gasPool := gethGas
		receipt, err := (func() (*types.Receipt, error) {
			sender, err = signer.Sender(tx)
			if err != nil {
				return nil, err
			}

			if err := hooks.PreTxFilter(state, tx, sender); err != nil {
				return nil, err
			}

			aggregator := &poster
			if util.DoesTxTypeAlias(tx.Type()) {
				aggregator = nil
			}
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
				return nil, core.ErrGasLimitReached
			}

			snap := statedb.Snapshot()
			statedb.Prepare(tx.Hash(), len(receipts)) // the number of successful state transitions

			gasLeft -= computeGas

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
				// Ignore this transaction if it's invalid under the state transition function
				statedb.RevertToSnapshot(snap)
				return nil, err
			}

			return receipt, hooks.PostTxFilter(state, tx, sender, dataGas, receipt)
		})()

		// append the err, even if it is nil
		hooks.TxErrors = append(hooks.TxErrors, err)

		if err != nil {
			log.Debug("error applying transaction", "tx", tx, "err", err)
			continue
		}

		// Update expectedTotalBalanceDelta (also done in logs loop)
		switch txInner := tx.GetInner().(type) {
		case *types.ArbitrumDepositTx:
			// L1->L2 deposits add eth to the system
			expectedBalanceDelta.Add(expectedBalanceDelta, txInner.Value)
		case *types.ArbitrumSubmitRetryableTx:
			// Retryable submission can include a deposit which adds eth to the system
			expectedBalanceDelta.Add(expectedBalanceDelta, txInner.DepositValue)
		}

		if gasPool > gethGas {
			delta := strconv.FormatUint(gasPool.Gas()-gethGas.Gas(), 10)
			panic("ApplyTransaction() gave back " + delta + " gas")
		}

		gasUsed := gethGas.Gas() - gasPool.Gas()
		gethGas = gasPool

		if gasUsed < dataGas {
			delta := strconv.FormatUint(dataGas-gasUsed, 10)
			panic("ApplyTransaction() used " + delta + " less gas than it should have")
		}

		if gasUsed > tx.Gas() {
			delta := strconv.FormatUint(gasUsed-tx.Gas(), 10)
			panic("ApplyTransaction() used " + delta + " more gas than it should have")
		}

		for _, txLog := range receipt.Logs {
			if txLog.Address == ArbRetryableTxAddress && txLog.Topics[0] == RedeemScheduledEventID {
				event := &precompilesgen.ArbRetryableTxRedeemScheduled{}
				err := util.ParseRedeemScheduledLog(event, txLog)
				if err != nil {
					log.Error("Failed to parse RedeemScheduled log", "err", err)
				} else {
					retryable, _ := state.RetryableState().OpenRetryable(event.TicketId, time)
					redeem, _ := retryable.MakeTx(
						chainConfig.ChainID,
						event.SequenceNum,
						gasPrice,
						event.DonatedGas,
						event.TicketId,
						event.GasDonor,
					)
					redeems = append(redeems, types.NewTx(redeem))
				}
			} else if txLog.Address == ArbSysAddress && txLog.Topics[0] == L2ToL1TransactionEventID {
				// L2->L1 withdrawals remove eth from the system
				event := &precompilesgen.ArbSysL2ToL1Transaction{}
				err := util.ParseL2ToL1TransactionLog(event, txLog)
				if err != nil {
					log.Error("Failed to parse L2ToL1Transaction log", "err", err)
				} else {
					expectedBalanceDelta.Sub(expectedBalanceDelta, event.Callvalue)
				}
			}
		}

		complete = append(complete, tx)
		receipts = append(receipts, receipt)
		gasLeft -= gasUsed - dataGas
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

	balanceDelta := statedb.GetTotalBalanceDelta()
	if balanceDelta.Cmp(expectedBalanceDelta) != 0 {
		// Panic if funds have been minted or debug mode is enabled (i.e. this is a test)
		if balanceDelta.Cmp(expectedBalanceDelta) > 0 || chainConfig.DebugMode() {
			panic(fmt.Sprintf("Unexpected total balance delta %v (expected %v)", balanceDelta, expectedBalanceDelta))
		} else {
			// This is a real chain and funds were burnt, not minted, so only log an error and don't panic
			log.Error("Unexpected total balance delta", "delta", balanceDelta, "expected", expectedBalanceDelta)
		}
	}

	return block, receipts
}

type ArbitrumHeaderInfo struct {
	SendRoot  common.Hash
	SendCount uint64
}

func (info ArbitrumHeaderInfo) Extra() []byte {
	return info.SendRoot[:]
}

func (info ArbitrumHeaderInfo) MixDigest() [32]byte {
	mixDigest := common.Hash{}
	binary.BigEndian.PutUint64(mixDigest[:8], info.SendCount)
	return mixDigest
}

func DeserializeHeaderExtraInformation(header *types.Header) (ArbitrumHeaderInfo, error) {
	if header.Number.Sign() == 0 || len(header.Extra) == 0 {
		// The genesis block doesn't have an ArbOS encoded extra field
		return ArbitrumHeaderInfo{}, nil
	}
	if len(header.Extra) != 32 {
		return ArbitrumHeaderInfo{}, fmt.Errorf("unexpected header extra field length %v", len(header.Extra))
	}
	extra := ArbitrumHeaderInfo{}
	copy(extra.SendRoot[:], header.Extra)
	extra.SendCount = binary.BigEndian.Uint64(header.MixDigest[:8])
	return extra, nil
}

func FinalizeBlock(header *types.Header, txs types.Transactions, receipts types.Receipts, statedb *state.StateDB) {
	if header != nil {
		state, err := arbosState.OpenSystemArbosState(statedb, false)
		if err != nil {
			panic(err)
		}
		state.SetLastTimestampSeen(header.Time)
		_ = state.RetryableState().TryToReapOneRetryable(header.Time)

		maxSafePrice := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
		state.L2PricingState().SetMaxGasPriceWei(maxSafePrice)

		// Add outbox info to the header for client-side proving
		acc := state.SendMerkleAccumulator()
		root, _ := acc.Root()
		size, _ := acc.Size()
		arbitrumHeader := ArbitrumHeaderInfo{root, size}
		header.Extra = arbitrumHeader.Extra()
		header.MixDigest = arbitrumHeader.MixDigest()

		state.UpgradeArbosVersionIfNecessary(header.Time)
	}
}
