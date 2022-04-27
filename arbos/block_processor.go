// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"

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

func createNewHeader(prevHeader *types.Header, l1info *L1Info, state *arbosState.ArbosState, chainConfig *params.ChainConfig) *types.Header {
	var lastBlockHash common.Hash
	blockNumber := big.NewInt(0)
	timestamp := uint64(0)
	coinbase := common.Address{}
	if l1info != nil {
		timestamp = l1info.l1Timestamp
		coinbase = l1info.poster
	}
	if prevHeader != nil {
		lastBlockHash = prevHeader.Hash()
		blockNumber.Add(prevHeader.Number, big.NewInt(1))
		if timestamp < prevHeader.Time {
			timestamp = prevHeader.Time
		}
	}

	timePassed := timestamp - prevHeader.Time
	baseFee := state.L2PricingState().UpdatePricingModel(prevHeader.BaseFee, timePassed, false, false)

	return &types.Header{
		ParentHash:  lastBlockHash,
		UncleHash:   types.EmptyUncleHash, // Post-merge Ethereum will require this to be types.EmptyUncleHash
		Coinbase:    coinbase,
		Root:        [32]byte{},    // Filled in later
		TxHash:      [32]byte{},    // Filled in later
		ReceiptHash: [32]byte{},    // Filled in later
		Bloom:       [256]byte{},   // Filled in later
		Difficulty:  big.NewInt(1), // Eventually, Ethereum plans to require this to be zero
		Number:      blockNumber,
		GasLimit:    l2pricing.GethBlockGasLimit,
		GasUsed:     0,
		Time:        timestamp,
		Extra:       []byte{},   // Unused; Post-merge Ethereum will limit the size of this to 32 bytes
		MixDigest:   [32]byte{}, // Post-merge Ethereum will require this to be zero
		Nonce:       [8]byte{},  // Filled in later; post-merge Ethereum will require this to be zero
		BaseFee:     baseFee,
	}
}

type SequencingHooks struct {
	TxErrors       []error
	RequireDataGas bool
	PreTxFilter    func(*arbosState.ArbosState, *types.Transaction, common.Address) error
	PostTxFilter   func(*arbosState.ArbosState, *types.Transaction, common.Address, uint64, *types.Receipt) error
}

func noopSequencingHooks() *SequencingHooks {
	return &SequencingHooks{
		[]error{},
		false,
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
	l1Header *L1IncomingMessageHeader,
	txes types.Transactions,
	delayedMessagesRead uint64,
	lastBlockHeader *types.Header,
	statedb *state.StateDB,
	chainContext core.ChainContext,
	chainConfig *params.ChainConfig,
	sequencingHooks *SequencingHooks,
) (*types.Block, types.Receipts) {

	state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		panic(err)
	}

	if statedb.GetUnexpectedBalanceDelta().BitLen() != 0 {
		panic("ProduceBlock called with dirty StateDB (non-zero unexpected balance delta)")
	}

	poster := l1Header.Poster

	l1Info := &L1Info{
		poster:        poster,
		l1BlockNumber: l1Header.BlockNumber,
		l1Timestamp:   l1Header.Timestamp,
	}

	header := createNewHeader(lastBlockHeader, l1Info, state, chainConfig)
	signer := types.MakeSigner(chainConfig, header.Number)
	gasLeft, _ := state.L2PricingState().PerBlockGasLimit()
	l1BlockNum := l1Info.l1BlockNumber

	// Prepend a tx before all others to touch up the state (update the L1 block num, pricing pools, etc)
	startTx := InternalTxStartBlock(chainConfig.ChainID, l1Header.L1BaseFee, l1BlockNum, header, lastBlockHeader)
	txes = append(types.Transactions{types.NewTx(startTx)}, txes...)

	complete := types.Transactions{}
	receipts := types.Receipts{}
	gasPrice := header.BaseFee
	time := header.Time
	expectedBalanceDelta := new(big.Int)
	redeems := types.Transactions{}
	userTxsCompleted := 0

	// We'll check that the block can fit each message, so this pool is set to not run out
	gethGas := core.GasPool(l2pricing.GethBlockGasLimit)

	for len(txes) > 0 || len(redeems) > 0 {
		// repeatedly process the next tx, doing redeems created along the way in FIFO order

		var tx *types.Transaction
		hooks := noopSequencingHooks()
		isUserTx := false
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
			switch tx := tx.GetInner().(type) {
			case *types.ArbitrumInternalTx:
				tx.TxIndex = uint64(len(receipts))
			default:
				hooks = sequencingHooks // the sequencer has the ability to drop this tx
				isUserTx = true
			}
		}

		var sender common.Address
		var dataGas uint64 = 0
		gasPool := gethGas
		receipt, scheduled, err := (func() (*types.Receipt, types.Transactions, error) {
			sender, err = signer.Sender(tx)
			if err != nil {
				return nil, nil, err
			}

			if err := hooks.PreTxFilter(state, tx, sender); err != nil {
				return nil, nil, err
			}

			if gasPrice.Sign() > 0 {
				dataGas = math.MaxUint64
				state.L1PricingState().AddPosterInfo(tx, sender, poster)
				posterCostInL2Gas := arbmath.BigDiv(tx.PosterCost, gasPrice)

				if posterCostInL2Gas.IsUint64() {
					dataGas = posterCostInL2Gas.Uint64()
				} else {
					log.Error("Could not get poster cost in L2 terms", tx.PosterCost, gasPrice)
				}
			}

			if dataGas > tx.Gas() {
				// this txn is going to be rejected later
				if hooks.RequireDataGas {
					return nil, nil, core.ErrIntrinsicGas
				}
				dataGas = 0
			}

			computeGas := tx.Gas() - dataGas
			if computeGas < params.TxGas {
				// ensure at least TxGas is left in the pool before trying a state transition
				computeGas = params.TxGas
			}

			if computeGas > gasLeft && isUserTx && userTxsCompleted > 0 {
				return nil, nil, core.ErrGasLimitReached
			}

			snap := statedb.Snapshot()
			statedb.Prepare(tx.Hash(), len(receipts)) // the number of successful state transitions

			receipt, result, err := core.ApplyTransaction(
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
				return nil, nil, err
			}

			return receipt, result.ScheduledTxes, hooks.PostTxFilter(state, tx, sender, dataGas, receipt)
		})()

		// append the err, even if it is nil
		hooks.TxErrors = append(hooks.TxErrors, err)

		if err != nil {
			// we'll still deduct a TxGas's worth from the block even if the tx was invalid
			log.Debug("error applying transaction", "tx", tx, "err", err)
			if gasLeft > params.TxGas && isUserTx {
				gasLeft -= params.TxGas
			} else {
				gasLeft = 0
			}
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

		// append any scheduled redeems
		redeems = append(redeems, scheduled...)

		for _, txLog := range receipt.Logs {
			if txLog.Address == ArbSysAddress && txLog.Topics[0] == L2ToL1TransactionEventID {
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

		if isUserTx {
			computeUsed := gasUsed - dataGas
			if computeUsed < params.TxGas {
				// a tx, even if invalid, must at least reduce the pool by TxGas
				computeUsed = params.TxGas
			}
			gasLeft -= computeUsed
		}

		complete = append(complete, tx)
		receipts = append(receipts, receipt)

		if isUserTx {
			userTxsCompleted++
		}
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

	FinalizeBlock(header, complete, statedb)
	header.Root = statedb.IntermediateRoot(true)

	block := types.NewBlock(header, complete, nil, receipts, trie.NewStackTrie(nil))

	if len(block.Transactions()) != len(receipts) {
		panic(fmt.Sprintf("Block has %d txes but %d receipts", len(block.Transactions()), len(receipts)))
	}

	balanceDelta := statedb.GetUnexpectedBalanceDelta()
	if !arbmath.BigEquals(balanceDelta, expectedBalanceDelta) {
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

func FinalizeBlock(header *types.Header, txs types.Transactions, statedb *state.StateDB) {
	if header != nil {
		state, _ := arbosState.OpenSystemArbosState(statedb, nil, true)

		// Add outbox info to the header for client-side proving
		acc := state.SendMerkleAccumulator()
		root, _ := acc.Root()
		size, _ := acc.Size()
		nextL1BlockNumber, _ := state.Blockhashes().NextBlockNumber()
		arbitrumHeader := types.HeaderInfo{
			SendRoot:      root,
			SendCount:     size,
			L1BlockNumber: nextL1BlockNumber,
		}
		arbitrumHeader.UpdateHeaderWithInfo(header)
	}
}
