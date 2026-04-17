// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// set by the precompile module, to avoid a package dependence cycle
var ArbRetryableTxAddress common.Address
var ArbSysAddress common.Address
var InternalTxStartBlockMethodID [4]byte
var InternalTxBatchPostingReportMethodID [4]byte
var InternalTxBatchPostingReportV2MethodID [4]byte
var RedeemScheduledEventID common.Hash
var L2ToL1TransactionEventID common.Hash
var L2ToL1TxEventID common.Hash
var EmitReedeemScheduledEvent func(*vm.EVM, uint64, uint64, [32]byte, [32]byte, common.Address, *big.Int, *big.Int) error
var EmitTicketCreatedEvent func(*vm.EVM, [32]byte) error

// ErrFilteredCascadingRedeem is returned via TxFailed when a redeem's
// inner execution touches a filtered address, requiring the entire tx group
// (originating user tx + all its redeems) to be reverted. All fields are
// captured before the group rollback so TxFailed can build a fully populated
// FilteredTxReport without late-filling.
type ErrFilteredCascadingRedeem struct {
	OriginatingTx     *types.Transaction
	FilteredAddresses []filter.FilteredAddressRecord
	BlockNumber       uint64
	ParentBlockHash   common.Hash
	PositionInBlock   int // receipt index of the originating user tx
}

func (e *ErrFilteredCascadingRedeem) Error() string {
	return fmt.Sprintf("cascading redeem filtered (originating tx: %s)", e.OriginatingTx.Hash().Hex())
}

// A helper struct that implements String() by marshalling to JSON.
// This is useful for logging because it's lazy, so if the log level is too high to print the transaction,
// it doesn't waste compute marshalling the transaction when the result wouldn't be used.
type printTxAsJson struct {
	tx *types.Transaction
}

func (p printTxAsJson) String() string {
	json, err := p.tx.MarshalJSON()
	if err != nil {
		return fmt.Sprintf("[error marshalling tx: %v]", err)
	}
	return string(json)
}

// blockBuildState holds all mutable state that accumulates during the tx
// processing loop in ProduceBlockAdvanced. Grouping it here ensures that
// the group-checkpoint and rollback logic stays in sync with the state it
// manages. If you add a field, check whether saveGroupCheckpoint and
// rollbackToGroupCheckpoint need updating.
//
// lint:require-exhaustive-initialization
type blockBuildState struct {
	statedb              *state.StateDB
	arbState             *arbosState.ArbosState
	blockGasLeft         uint64
	expectedBalanceDelta *big.Int
	userTxsProcessed     int
	complete             types.Transactions
	receipts             types.Receipts
	redeems              types.Transactions
	activeGroupCP        *groupCheckpoint
}

// lint:require-exhaustive-initialization
type groupCheckpoint struct {
	backup               *state.StateDB
	snap                 int
	headerGasUsed        uint64
	blockGasLeft         uint64
	expectedBalanceDelta *big.Int
	userTxsProcessed     int
	completeLen          int
	receiptsLen          int
	userTxHash           common.Hash
}

// saveGroupCheckpoint snapshots the loop state so the entire tx group can be
// rolled back if a descendant redeem is filtered. header is passed separately
// because only GasUsed is checkpointed; the rest of the header is immutable
// during the loop.
func (s *blockBuildState) saveGroupCheckpoint(header *types.Header, snap int, userTxHash common.Hash) error {
	if len(s.redeems) != 0 {
		return errors.New("saveGroupCheckpoint called with pending redeems")
	}
	s.activeGroupCP = &groupCheckpoint{
		backup:               s.statedb.Copy(),
		snap:                 snap,
		headerGasUsed:        header.GasUsed,
		blockGasLeft:         s.blockGasLeft,
		expectedBalanceDelta: new(big.Int).Set(s.expectedBalanceDelta),
		userTxsProcessed:     s.userTxsProcessed,
		completeLen:          len(s.complete),
		receiptsLen:          len(s.receipts),
		userTxHash:           userTxHash,
	}
	return nil
}

// rollbackToGroupCheckpoint restores loop state to the saved checkpoint,
// undoing the user tx and all its redeems. header is needed to restore
// GasUsed, which lives outside blockBuildState.
func (s *blockBuildState) rollbackToGroupCheckpoint(header *types.Header) error {
	cp := s.activeGroupCP
	cp.backup.RevertToSnapshot(cp.snap)
	s.statedb = cp.backup
	header.GasUsed = cp.headerGasUsed
	s.blockGasLeft = cp.blockGasLeft
	s.expectedBalanceDelta.Set(cp.expectedBalanceDelta)
	s.userTxsProcessed = cp.userTxsProcessed
	s.redeems = s.redeems[:0]
	s.complete = s.complete[:cp.completeLen]
	s.receipts = s.receipts[:cp.receiptsLen]
	var err error
	s.arbState, err = arbosState.OpenSystemArbosState(s.statedb, nil, true)
	if err != nil {
		return err
	}
	s.activeGroupCP = nil
	return nil
}

func (s *blockBuildState) clearGroupCheckpoint() {
	s.activeGroupCP = nil
}

type L1Info struct {
	poster        common.Address
	l1BlockNumber uint64
	l1Timestamp   uint64
}

func (info *L1Info) Equals(o *L1Info) bool {
	return info.poster == o.poster && info.l1BlockNumber == o.l1BlockNumber && info.l1Timestamp == o.l1Timestamp
}

func (info *L1Info) L1BlockNumber() uint64 {
	return info.l1BlockNumber
}

func createNewHeader(prevHeader *types.Header, l1info *L1Info, baseFee *big.Int, chainConfig *params.ChainConfig) *types.Header {
	var lastBlockHash common.Hash
	blockNumber := big.NewInt(0)
	timestamp := uint64(0)
	coinbase := common.Address{}
	if l1info != nil {
		timestamp = l1info.l1Timestamp
		coinbase = l1info.poster
	}
	extra := common.Hash{}.Bytes()
	mixDigest := common.Hash{}
	if prevHeader != nil {
		lastBlockHash = prevHeader.Hash()
		blockNumber.Add(prevHeader.Number, big.NewInt(1))
		if timestamp < prevHeader.Time {
			timestamp = prevHeader.Time
		}
		copy(extra, prevHeader.Extra)
		mixDigest = prevHeader.MixDigest
	}
	header := &types.Header{
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
		Extra:       extra,     // used by NewEVMBlockContext
		MixDigest:   mixDigest, // used by NewEVMBlockContext
		Nonce:       [8]byte{}, // Filled in later; post-merge Ethereum will require this to be zero
		BaseFee:     baseFee,
	}
	return header
}

type ConditionalOptionsForTx []*arbitrum_types.ConditionalOptions

type SequencingHooks interface {
	// NextTxToSequence returns the next tx to include, or nil when done.
	NextTxToSequence() (*types.Transaction, *arbitrum_types.ConditionalOptions, error)
	// CanDiscardTx returns whether failed txs can be excluded from the block.
	// This is a static property of the implementing type (true for sequencer, false for replay).
	CanDiscardTx() bool
	// SupportsGroupRollback returns whether the hooks support checkpointing and
	// rolling back a group of transactions (user tx + its scheduled redeems).
	SupportsGroupRollback() bool
	// PreTxFilter rejects a tx before execution. positionInBlock is len(receipts) at call time.
	PreTxFilter(*params.ChainConfig, *types.Header, *state.StateDB, *arbosState.ArbosState, *types.Transaction, *arbitrum_types.ConditionalOptions, common.Address, *L1Info, int) error
	// PostTxFilter rejects a tx after execution. positionInBlock is len(receipts) at call time.
	PostTxFilter(*types.Header, *state.StateDB, *arbosState.ArbosState, *types.Transaction, common.Address, uint64, *core.ExecutionResult, int) error
	// BlockFilter rejects an entire block after all txs have been applied.
	BlockFilter(*types.Header, *state.StateDB, types.Transactions, types.Receipts) error
	// TxSucceeded records that the last user tx from NextTxToSequence executed successfully.
	TxSucceeded()
	// TxFailed records an error for the last user tx from NextTxToSequence.
	TxFailed(error)
}

type NoopSequencingHooks struct {
	txs               types.Transactions
	scheduledTxsCount int
}

func (n *NoopSequencingHooks) NextTxToSequence() (*types.Transaction, *arbitrum_types.ConditionalOptions, error) {
	// This is not supposed to happen, if so we have a bug
	if n.scheduledTxsCount > len(n.txs) {
		return nil, nil, errors.New("noopTxScheduler: requested too many transactions")
	}
	if n.scheduledTxsCount == len(n.txs) {
		return nil, nil, nil
	}
	n.scheduledTxsCount += 1
	return n.txs[n.scheduledTxsCount-1], nil, nil
}

func (n *NoopSequencingHooks) CanDiscardTx() bool { return false }

func (n *NoopSequencingHooks) PreTxFilter(config *params.ChainConfig, header *types.Header, db *state.StateDB, a *arbosState.ArbosState, transaction *types.Transaction, options *arbitrum_types.ConditionalOptions, address common.Address, info *L1Info, positionInBlock int) error {
	return nil
}

func (n *NoopSequencingHooks) PostTxFilter(header *types.Header, db *state.StateDB, a *arbosState.ArbosState, transaction *types.Transaction, address common.Address, u uint64, result *core.ExecutionResult, positionInBlock int) error {
	return nil
}

func (n *NoopSequencingHooks) BlockFilter(header *types.Header, db *state.StateDB, transactions types.Transactions, receipts types.Receipts) error {
	return nil
}

func (n *NoopSequencingHooks) TxSucceeded() {}

func (n *NoopSequencingHooks) TxFailed(error) {}

func (n *NoopSequencingHooks) SupportsGroupRollback() bool { return false }

func NewNoopSequencingHooks(txes types.Transactions) *NoopSequencingHooks {
	return &NoopSequencingHooks{txs: txes}
}

func ProduceBlock(
	message *arbostypes.L1IncomingMessage,
	delayedMessagesRead uint64,
	lastBlockHeader *types.Header,
	statedb *state.StateDB,
	chainContext core.ChainContext,
	isMsgForPrefetch bool,
	runCtx *core.MessageRunContext,
	exposeMultiGas bool,
) (*types.Block, *state.StateDB, types.Receipts, error) {
	chainConfig := chainContext.Config()
	lastArbosVersion := types.DeserializeHeaderExtraInformation(lastBlockHeader).ArbOSFormatVersion
	txes, err := ParseL2Transactions(message, chainConfig.ChainID, lastArbosVersion)
	if err != nil {
		log.Warn("error parsing incoming message", "err", err)
		txes = types.Transactions{}
	}
	hooks := NewNoopSequencingHooks(txes)

	return ProduceBlockAdvanced(
		message.Header, delayedMessagesRead, lastBlockHeader, statedb, chainContext, hooks, isMsgForPrefetch, runCtx, exposeMultiGas,
	)
}

// A bit more flexible than ProduceBlock for use in the sequencer.
func ProduceBlockAdvanced(
	l1Header *arbostypes.L1IncomingMessageHeader,
	delayedMessagesRead uint64,
	lastBlockHeader *types.Header,
	statedb *state.StateDB,
	chainContext core.ChainContext,
	sequencingHooks SequencingHooks,
	isMsgForPrefetch bool,
	runCtx *core.MessageRunContext,
	exposeMultiGas bool,
) (*types.Block, *state.StateDB, types.Receipts, error) {

	arbState, err := arbosState.OpenSystemArbosState(statedb, nil, false)
	if err != nil {
		return nil, nil, nil, err
	}

	if statedb.GetUnexpectedBalanceDelta().BitLen() != 0 {
		return nil, nil, nil, errors.New("ProduceBlock called with dirty StateDB (non-zero unexpected balance delta)")
	}

	poster := l1Header.Poster

	l1Info := &L1Info{
		poster:        poster,
		l1BlockNumber: l1Header.BlockNumber,
		l1Timestamp:   l1Header.Timestamp,
	}

	chainConfig := chainContext.Config()

	l2Pricing := arbState.L2PricingState()
	err = l2Pricing.CommitMultiGasFees()
	if err != nil {
		return nil, nil, nil, err
	}
	baseFee, err := l2Pricing.BaseFeeWei()
	if err != nil {
		return nil, nil, nil, err
	}

	header := createNewHeader(lastBlockHeader, l1Info, baseFee, chainConfig)
	// Note: blockGasLeft will diverge from the actual gas left during execution in the event of invalid txs,
	// but it's only used as block-local representation limiting the amount of work done in a block.
	blockGasLeft, _ := arbState.L2PricingState().PerBlockGasLimit()
	l1BlockNum := l1Info.l1BlockNumber

	// Prepend a tx before all others to touch up the state (update the L1 block num, pricing pools, etc)
	startTx := InternalTxStartBlock(chainConfig.ChainID, l1Header.L1BaseFee, l1BlockNum, header, lastBlockHeader)

	basefee := header.BaseFee
	time := header.Time

	// We'll check that the block can fit each message, so this pool is set to not run out
	gethGas := core.GasPool(l2pricing.GethBlockGasLimit)

	firstTx := types.NewTx(startTx)

	buildState := &blockBuildState{
		statedb:              statedb,
		arbState:             arbState,
		blockGasLeft:         blockGasLeft,
		expectedBalanceDelta: new(big.Int),
		userTxsProcessed:     0,
		complete:             nil,
		receipts:             nil,
		redeems:              nil,
		activeGroupCP:        nil,
	}

	for {
		// repeatedly process the next tx, doing redeems created along the way in FIFO order

		var tx *types.Transaction
		var options *arbitrum_types.ConditionalOptions
		isUserTx := false
		if firstTx != nil {
			tx = firstTx
			firstTx = nil
		} else if len(buildState.redeems) > 0 {
			tx = buildState.redeems[0]
			buildState.redeems = buildState.redeems[1:]

			retry, ok := (tx.GetInner()).(*types.ArbitrumRetryTx)
			if !ok {
				return nil, nil, nil, errors.New("retryable tx is somehow not a retryable")
			}
			retryable, _ := buildState.arbState.RetryableState().OpenRetryable(retry.TicketId, time)
			if retryable == nil {
				// retryable was already deleted
				continue
			}
		} else {
			// Previous group (if any) completed successfully
			if buildState.activeGroupCP != nil {
				sequencingHooks.TxSucceeded()
			}
			buildState.clearGroupCheckpoint()
			var conditionalOptions *arbitrum_types.ConditionalOptions
			tx, conditionalOptions, err = sequencingHooks.NextTxToSequence()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("error fetching next transaction to sequence, userTxsProcessed: %d, err: %w", buildState.userTxsProcessed, err)
			}
			if tx == nil {
				break
			}
			if tx.Type() != types.ArbitrumInternalTxType {
				isUserTx = true
				options = conditionalOptions
			}
		}

		startRefund := buildState.statedb.GetRefund()
		if startRefund != 0 {
			return nil, nil, nil, fmt.Errorf("at beginning of tx statedb has non-zero refund %v", startRefund)
		}

		var sender common.Address
		var dataGas uint64 = 0
		preTxHeaderGasUsed := header.GasUsed
		arbosVersion := buildState.arbState.ArbOSVersion()
		signer := types.MakeSigner(chainConfig, header.Number, header.Time, arbosVersion)
		receipt, result, err := (func() (*types.Receipt, *core.ExecutionResult, error) {
			// If we've done too much work in this block, discard the tx as early as possible
			if buildState.blockGasLeft < params.TxGas && isUserTx {
				return nil, nil, core.ErrGasLimitReached
			}

			sender, err = types.Sender(signer, tx)
			if err != nil {
				return nil, nil, err
			}

			// Writes to statedb object should be avoided to prevent invalid state from permeating as statedb snapshot is not taken
			if isUserTx {
				if err = sequencingHooks.PreTxFilter(chainConfig, header, buildState.statedb, buildState.arbState, tx, options, sender, l1Info, len(buildState.receipts)); err != nil {
					return nil, nil, err
				}
			}

			// Additional pre-transaction validity check
			// Writes to statedb object should be avoided to prevent invalid state from permeating as statedb snapshot is not taken
			if err = extraPreTxFilter(chainConfig, header, buildState.statedb, buildState.arbState, tx, options, sender, l1Info); err != nil {
				return nil, nil, err
			}

			if basefee.Sign() > 0 {
				dataGas = math.MaxUint64
				brotliCompressionLevel, err := buildState.arbState.BrotliCompressionLevel()
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get brotli compression level: %w", err)
				}
				posterCost, _ := buildState.arbState.L1PricingState().GetPosterInfo(tx, poster, brotliCompressionLevel)
				posterCostInL2Gas := arbmath.BigDiv(posterCost, basefee)

				if posterCostInL2Gas.IsUint64() {
					dataGas = posterCostInL2Gas.Uint64()
				} else {
					log.Error("Could not get poster cost in L2 terms", "posterCost", posterCost, "basefee", basefee)
				}
			}

			if dataGas > tx.Gas() {
				// this txn is going to be rejected later
				dataGas = tx.Gas()
			}

			computeGas := tx.Gas() - dataGas

			if computeGas < params.TxGas {
				if isUserTx && sequencingHooks.CanDiscardTx() {
					return nil, nil, core.ErrIntrinsicGas
				}
				// ensure at least TxGas is left in the pool before trying a state transition
				computeGas = params.TxGas
			}

			// arbos<50: reject tx if they have available computeGas over block-gas-limit
			// in arbos>=50, per-block-gas is limited to L2PricingState().PerBlockGasLimit() + L2PricingState().PerTxGasLimit()
			if arbosVersion < params.ArbosVersion_50 && computeGas > buildState.blockGasLeft && isUserTx && buildState.userTxsProcessed > 0 {
				return nil, nil, core.ErrGasLimitReached
			}

			snap := buildState.statedb.Snapshot()
			buildState.statedb.SetTxContext(tx.Hash(), len(buildState.receipts)) // the number of successful state transitions

			gasPool := gethGas
			blockContext := core.NewEVMBlockContext(header, chainContext, &header.Coinbase)
			evm := vm.NewEVM(blockContext, buildState.statedb, chainConfig, vm.Config{ExposeMultiGas: exposeMultiGas})
			receipt, result, err := core.ApplyTransactionWithResultFilter(
				evm,
				&gasPool,
				buildState.statedb,
				header,
				tx,
				&header.GasUsed,
				runCtx,
				func(result *core.ExecutionResult) error {
					if err := sequencingHooks.PostTxFilter(header, buildState.statedb, buildState.arbState, tx, sender, dataGas, result, len(buildState.receipts)); err != nil {
						return err
					}
					// Additional post-transaction validity check
					if err = extraPostTxFilter(chainConfig, header, buildState.statedb, buildState.arbState, tx, options, sender, l1Info, result); err != nil {
						return err
					}
					if isUserTx && len(result.ScheduledTxes) > 0 && sequencingHooks.SupportsGroupRollback() {
						if err := buildState.saveGroupCheckpoint(header, snap, tx.Hash()); err != nil {
							return err
						}
					}
					return nil
				},
			)
			if err != nil {
				// Ignore this transaction if it's invalid under the state transition function
				buildState.statedb.RevertToSnapshot(snap)
				buildState.statedb.ClearTxFilter()
				return nil, nil, err
			}

			return receipt, result, nil
		})()

		if err != nil {
			// If a redeem was rejected by the address filter and we have an
			// active group checkpoint, roll back the entire group (user tx + all
			// redeems) to the pre-group state.
			if !isUserTx && buildState.activeGroupCP != nil && errors.Is(err, state.ErrArbTxFilter) {
				// Capture everything before rollback — addressCheckerStateß
				cp := buildState.activeGroupCP
				_, filteredAddresses := buildState.statedb.IsAddressFiltered()
				originatingTx := buildState.complete[cp.completeLen]
				if err := buildState.rollbackToGroupCheckpoint(header); err != nil {
					return nil, nil, nil, err
				}
				sequencingHooks.TxFailed(&ErrFilteredCascadingRedeem{
					OriginatingTx:     originatingTx,
					FilteredAddresses: filteredAddresses,
					BlockNumber:       header.Number.Uint64(),
					ParentBlockHash:   header.ParentHash,
					PositionInBlock:   cp.receiptsLen,
				})
				continue
			}
			if isUserTx {
				sequencingHooks.TxFailed(err)
			}
			logLevel := log.Debug
			if chainConfig.DebugMode() {
				logLevel = log.Warn
			}
			if !isMsgForPrefetch {
				logLevel("error applying transaction", "tx", printTxAsJson{tx}, "err", err)
			}
			if !(isUserTx && sequencingHooks.CanDiscardTx()) {
				// we'll still deduct a TxGas's worth from the block-local rate limiter even if the tx was invalid
				buildState.blockGasLeft = arbmath.SaturatingUSub(buildState.blockGasLeft, params.TxGas)
				if isUserTx {
					buildState.userTxsProcessed++
				}
			}
			continue
		}

		if tx.Type() == types.ArbitrumInternalTxType {
			// ArbOS might have upgraded to a new version, so we need to refresh our state
			buildState.arbState, err = arbosState.OpenSystemArbosState(buildState.statedb, nil, true)
			if err != nil {
				return nil, nil, nil, err
			}
			// Update the ArbOS version in the header (if it changed)
			extraInfo := types.DeserializeHeaderExtraInformation(header)
			extraInfo.ArbOSFormatVersion = buildState.arbState.ArbOSVersion()
			extraInfo.UpdateHeaderWithInfo(header)
		}

		if tx.Type() == types.ArbitrumInternalTxType && result.Err != nil {
			return nil, nil, nil, fmt.Errorf("failed to apply internal transaction: %w", result.Err)
		}

		if preTxHeaderGasUsed > header.GasUsed {
			return nil, nil, nil, fmt.Errorf("ApplyTransaction() used -%v gas", preTxHeaderGasUsed-header.GasUsed)
		}
		txGasUsed := header.GasUsed - preTxHeaderGasUsed

		arbosVer := types.DeserializeHeaderExtraInformation(header).ArbOSFormatVersion
		if arbosVer >= params.ArbosVersion_FixRedeemGas {
			// subtract gas burned for future use
			for _, scheduledTx := range result.ScheduledTxes {
				switch inner := scheduledTx.GetInner().(type) {
				case *types.ArbitrumRetryTx:
					txGasUsed = arbmath.SaturatingUSub(txGasUsed, inner.Gas)
				default:
					log.Warn("Unexpected type of scheduled tx", "type", scheduledTx.Type())
				}
			}
		}

		// Update expectedTotalBalanceDelta (also done in logs loop)
		switch txInner := tx.GetInner().(type) {
		case *types.ArbitrumDepositTx:
			// L1->L2 deposits add eth to the system
			buildState.expectedBalanceDelta.Add(buildState.expectedBalanceDelta, txInner.Value)
		case *types.ArbitrumSubmitRetryableTx:
			// Retryable submission can include a deposit which adds eth to the system
			buildState.expectedBalanceDelta.Add(buildState.expectedBalanceDelta, txInner.DepositValue)
		}

		// Use the actual poster gas from the receipt (which accounts for tip collection)
		// rather than the pre-tx estimate which always uses basefee.
		posterGasUsed := dataGas
		if arbosVersion >= params.ArbosVersion_60 {
			posterGasUsed = receipt.GasUsedForL1
		}
		computeUsed := txGasUsed - posterGasUsed
		if txGasUsed < posterGasUsed {
			log.Error("ApplyTransaction() used less gas than it should have", "delta", posterGasUsed-txGasUsed)
			computeUsed = params.TxGas
		} else if computeUsed < params.TxGas {
			computeUsed = params.TxGas
		}

		if txGasUsed > tx.Gas() {
			return nil, nil, nil, fmt.Errorf("ApplyTransaction() used %v more gas than it should have", txGasUsed-tx.Gas())
		}

		// append any scheduled redeems
		buildState.redeems = append(buildState.redeems, result.ScheduledTxes...)

		for _, txLog := range receipt.Logs {
			if txLog.Address == ArbSysAddress {
				// L2ToL1TransactionEventID is deprecated in upgrade 4, but it should to safe to make this code handle
				// both events ignoring the version.
				// TODO: Remove L2ToL1Transaction handling on next chain reset
				// L2->L1 withdrawals remove eth from the system
				switch txLog.Topics[0] {
				case L2ToL1TransactionEventID:
					event, err := util.ParseL2ToL1TransactionLog(txLog)
					if err != nil {
						log.Error("Failed to parse L2ToL1Transaction log", "err", err)
					} else {
						buildState.expectedBalanceDelta.Sub(buildState.expectedBalanceDelta, event.Callvalue)
					}
				case L2ToL1TxEventID:
					event, err := util.ParseL2ToL1TxLog(txLog)
					if err != nil {
						log.Error("Failed to parse L2ToL1Tx log", "err", err)
					} else {
						buildState.expectedBalanceDelta.Sub(buildState.expectedBalanceDelta, event.Callvalue)
					}
				}
			}
		}

		buildState.blockGasLeft = arbmath.SaturatingUSub(buildState.blockGasLeft, computeUsed)

		buildState.complete = append(buildState.complete, tx)
		buildState.receipts = append(buildState.receipts, receipt)

		if isUserTx {
			if buildState.activeGroupCP == nil {
				sequencingHooks.TxSucceeded()
			}
			buildState.userTxsProcessed++
		} else if buildState.activeGroupCP != nil && len(buildState.redeems) == 0 {
			buildState.activeGroupCP = nil
			sequencingHooks.TxSucceeded()
		}
	}

	if buildState.statedb.IsTxFiltered() {
		return nil, nil, nil, state.ErrArbTxFilter
	}

	if err = sequencingHooks.BlockFilter(header, buildState.statedb, buildState.complete, buildState.receipts); err != nil {
		return nil, nil, nil, err
	}

	binary.BigEndian.PutUint64(header.Nonce[:], delayedMessagesRead)

	FinalizeBlock(header, buildState.complete, buildState.statedb, chainConfig)

	// Touch up the block hashes in receipts
	tmpBlock := types.NewBlock(header, &types.Body{Transactions: buildState.complete}, buildState.receipts, trie.NewStackTrie(nil))
	blockHash := tmpBlock.Hash()

	for _, receipt := range buildState.receipts {
		receipt.BlockHash = blockHash
		for _, txLog := range receipt.Logs {
			txLog.BlockHash = blockHash
		}
	}

	block := types.NewBlock(header, &types.Body{Transactions: buildState.complete}, buildState.receipts, trie.NewStackTrie(nil))

	if len(block.Transactions()) != len(buildState.receipts) {
		return nil, nil, nil, fmt.Errorf("block has %d txes but %d receipts", len(block.Transactions()), len(buildState.receipts))
	}

	balanceDelta := buildState.statedb.GetUnexpectedBalanceDelta()
	if !arbmath.BigEquals(balanceDelta, buildState.expectedBalanceDelta) {
		// Fail if funds have been minted or debug mode is enabled (i.e. this is a test)
		if balanceDelta.Cmp(buildState.expectedBalanceDelta) > 0 || chainConfig.DebugMode() {
			return nil, nil, nil, fmt.Errorf("unexpected total balance delta %v (expected %v)", balanceDelta, buildState.expectedBalanceDelta)
		}
		// This is a real chain and funds were burnt, not minted, so only log an error and don't panic
		log.Error("Unexpected total balance delta", "delta", balanceDelta, "expected", buildState.expectedBalanceDelta)
	}

	return block, buildState.statedb, buildState.receipts, nil
}

// Also sets header.Root
func FinalizeBlock(header *types.Header, txs types.Transactions, statedb vm.StateDB, chainConfig *params.ChainConfig) {
	if header != nil {
		if header.Number.Uint64() < chainConfig.ArbitrumChainParams.GenesisBlockNum {
			panic("cannot finalize blocks before genesis")
		}

		var sendRoot common.Hash
		var sendCount uint64
		var nextL1BlockNumber uint64
		var arbosVersion uint64
		collectTips := false

		if header.Number.Uint64() == chainConfig.ArbitrumChainParams.GenesisBlockNum {
			arbosVersion = chainConfig.ArbitrumChainParams.InitialArbOSVersion
		} else {
			state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
			if err != nil {
				newErr := fmt.Errorf("%w while opening arbos state. Block: %d root: %v", err, header.Number, header.Root)
				panic(newErr)
			}
			collectTips, err = state.CollectTips()
			if err != nil {
				newErr := fmt.Errorf("%w while reading collect tips setting. Block: %d root: %v", err, header.Number, header.Root)
				panic(newErr)
			}
			// Delayed-message blocks never collect tips, regardless of the chain-wide setting.
			// All transactions in a block share the same Coinbase, so this is a block-level property.
			if collectTips && header.Coinbase != l1pricing.BatchPosterAddress {
				collectTips = false
			}
			// Add outbox info to the header for client-side proving
			acc := state.SendMerkleAccumulator()
			sendRoot, _ = acc.Root()
			sendCount, _ = acc.Size()
			nextL1BlockNumber, _ = state.Blockhashes().L1BlockNumber()
			arbosVersion = state.ArbOSVersion()
		}
		arbitrumHeader := types.HeaderInfo{
			SendRoot:           sendRoot,
			SendCount:          sendCount,
			L1BlockNumber:      nextL1BlockNumber,
			ArbOSFormatVersion: arbosVersion,
			CollectTips:        collectTips,
		}
		arbitrumHeader.UpdateHeaderWithInfo(header)
		header.Root = statedb.IntermediateRoot(true)
	}
}
