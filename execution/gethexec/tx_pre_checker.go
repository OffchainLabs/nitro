// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var (
	conditionalTxRejectedByTxPreCheckerCurrentStateCounter = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/currentstate/rejected", nil)
	conditionalTxAcceptedByTxPreCheckerCurrentStateCounter = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/currentstate/accepted", nil)
	conditionalTxRejectedByTxPreCheckerOldStateCounter     = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/oldstate/rejected", nil)
	conditionalTxAcceptedByTxPreCheckerOldStateCounter     = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/oldstate/accepted", nil)
	txPreCheckerAddressFilterRejectedCounter               = metrics.NewRegisteredCounter("arb/txprechecker/addressfilter/rejected", nil)
)

const TxPreCheckerStrictnessNone uint = 0
const TxPreCheckerStrictnessAlwaysCompatible uint = 10
const TxPreCheckerStrictnessLikelyCompatible uint = 20
const TxPreCheckerStrictnessFullValidation uint = 30

type TxPreCheckerConfig struct {
	Strictness             uint  `koanf:"strictness" reload:"hot"`
	RequiredStateAge       int64 `koanf:"required-state-age" reload:"hot"`
	RequiredStateMaxBlocks uint  `koanf:"required-state-max-blocks" reload:"hot"`
}

type TxPreCheckerConfigFetcher func() *TxPreCheckerConfig

var DefaultTxPreCheckerConfig = TxPreCheckerConfig{
	Strictness:             TxPreCheckerStrictnessLikelyCompatible,
	RequiredStateAge:       2,
	RequiredStateMaxBlocks: 4,
}

func TxPreCheckerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint(prefix+".strictness", DefaultTxPreCheckerConfig.Strictness, "how strict to be when checking txs before forwarding them. 0 = accept anything, "+
		"10 = should never reject anything that'd succeed, 20 = likely won't reject anything that'd succeed, "+
		"30 = full validation which may reject txs that would succeed")
	f.Int64(prefix+".required-state-age", DefaultTxPreCheckerConfig.RequiredStateAge, "how long ago should the storage conditions from eth_SendRawTransactionConditional be true, 0 = don't check old state")
	f.Uint(prefix+".required-state-max-blocks", DefaultTxPreCheckerConfig.RequiredStateMaxBlocks, "maximum number of blocks to look back while looking for the <required-state-age> seconds old state, 0 = don't limit the search")
}

type TxPreChecker struct {
	TransactionPublisher
	bc                 *core.BlockChain
	config             TxPreCheckerConfigFetcher
	expressLaneTracker *timeboost.ExpressLaneTracker
	addressChecker     state.AddressChecker
	eventFilter        *eventfilter.EventFilter
}

func NewTxPreChecker(
	publisher TransactionPublisher,
	bc *core.BlockChain,
	config TxPreCheckerConfigFetcher) *TxPreChecker {
	return &TxPreChecker{
		TransactionPublisher: publisher,
		bc:                   bc,
		config:               config,
	}
}

type NonceError struct {
	sender     common.Address
	txNonce    uint64
	stateNonce uint64
}

func (e NonceError) Error() string {
	if e.txNonce < e.stateNonce {
		return fmt.Sprintf("%v: address %v, tx: %d state: %d", core.ErrNonceTooLow, e.sender, e.txNonce, e.stateNonce)
	}
	if e.txNonce > e.stateNonce {
		return fmt.Sprintf("%v: address %v, tx: %d state: %d", core.ErrNonceTooHigh, e.sender, e.txNonce, e.stateNonce)
	}
	// This should be unreachable
	return fmt.Sprintf("invalid nonce error for address %v nonce %v", e.sender, e.txNonce)
}

func (e NonceError) Unwrap() error {
	if e.txNonce < e.stateNonce {
		return core.ErrNonceTooLow
	}
	if e.txNonce > e.stateNonce {
		return core.ErrNonceTooHigh
	}
	// This should be unreachable
	return nil
}

func MakeNonceError(sender common.Address, txNonce uint64, stateNonce uint64) error {
	if txNonce == stateNonce {
		return nil
	}
	return NonceError{
		sender:     sender,
		txNonce:    txNonce,
		stateNonce: stateNonce,
	}
}

func PreCheckTx(bc *core.BlockChain, chainConfig *params.ChainConfig, header *types.Header, statedb *state.StateDB, arbos *arbosState.ArbosState, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, config *TxPreCheckerConfig) error {
	if config.Strictness < TxPreCheckerStrictnessAlwaysCompatible {
		return nil
	}
	if tx.Gas() < params.TxGas {
		return core.ErrIntrinsicGas
	}
	if tx.Type() >= types.ArbitrumDepositTxType || tx.Type() == types.BlobTxType {
		// Should be unreachable for Arbitrum types due to UnmarshalBinary not accepting Arbitrum internal txs
		// and we want to disallow BlobTxType since Arbitrum doesn't support EIP-4844 txs yet.
		return types.ErrTxTypeNotSupported
	}
	sender, err := types.Sender(types.MakeSigner(chainConfig, header.Number, header.Time, arbos.ArbOSVersion()), tx)
	if err != nil {
		return err
	}
	baseFee := header.BaseFee
	if config.Strictness < TxPreCheckerStrictnessLikelyCompatible {
		baseFee, err = arbos.L2PricingState().MinBaseFeeWei()
		if err != nil {
			return err
		}
	}
	if arbmath.BigLessThan(tx.GasFeeCap(), baseFee) {
		return fmt.Errorf("%w: address %v, maxFeePerGas: %s baseFee: %s", core.ErrFeeCapTooLow, sender, tx.GasFeeCap(), header.BaseFee)
	}
	stateNonce := statedb.GetNonce(sender)
	if tx.Nonce() < stateNonce {
		return MakeNonceError(sender, tx.Nonce(), stateNonce)
	}
	extraInfo := types.DeserializeHeaderExtraInformation(header)
	intrinsic, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), tx.To() == nil, chainConfig.IsHomestead(header.Number), chainConfig.IsIstanbul(header.Number), chainConfig.IsShanghai(header.Number, header.Time, extraInfo.ArbOSFormatVersion))
	if err != nil {
		return err
	}
	if tx.Gas() < intrinsic {
		return core.ErrIntrinsicGas
	}
	if config.Strictness < TxPreCheckerStrictnessLikelyCompatible {
		return nil
	}
	if options != nil {
		if err := options.Check(extraInfo.L1BlockNumber, header.Time, statedb); err != nil {
			conditionalTxRejectedByTxPreCheckerCurrentStateCounter.Inc(1)
			return err
		}
		conditionalTxAcceptedByTxPreCheckerCurrentStateCounter.Inc(1)
		if config.RequiredStateAge > 0 {
			now := time.Now().Unix()
			oldHeader := header
			blocksTraversed := uint(0)
			// find a block that's old enough
			// #nosec G115
			for now-int64(oldHeader.Time) < config.RequiredStateAge &&
				(config.RequiredStateMaxBlocks <= 0 || blocksTraversed < config.RequiredStateMaxBlocks) &&
				oldHeader.Number.Uint64() > 0 {
				previousHeader := bc.GetHeader(oldHeader.ParentHash, oldHeader.Number.Uint64()-1)
				if previousHeader == nil {
					break
				}
				oldHeader = previousHeader
				blocksTraversed++
			}
			if !headerreader.HeadersEqual(oldHeader, header) {
				secondOldStatedb, err := bc.StateAt(oldHeader.Root)
				if err != nil {
					return fmt.Errorf("failed to get old state: %w", err)
				}
				oldExtraInfo := types.DeserializeHeaderExtraInformation(oldHeader)
				if err := options.Check(oldExtraInfo.L1BlockNumber, oldHeader.Time, secondOldStatedb); err != nil {
					conditionalTxRejectedByTxPreCheckerOldStateCounter.Inc(1)
					return arbitrum_types.WrapOptionsCheckError(err, "conditions check failed for old state")
				}
			}
			conditionalTxAcceptedByTxPreCheckerOldStateCounter.Inc(1)
		}
	}
	balance := statedb.GetBalance(sender)
	cost := tx.Cost()
	if arbmath.BigLessThan(balance.ToBig(), cost) {
		return fmt.Errorf("%w: address %v have %v want %v", core.ErrInsufficientFunds, sender, balance, cost)
	}
	if config.Strictness >= TxPreCheckerStrictnessFullValidation && tx.Nonce() > stateNonce {
		return MakeNonceError(sender, tx.Nonce(), stateNonce)
	}
	brotliCompressionLevel, err := arbos.BrotliCompressionLevel()
	if err != nil {
		return fmt.Errorf("failed to get brotli compression level: %w", err)
	}
	dataCost, _ := arbos.L1PricingState().GetPosterInfo(tx, l1pricing.BatchPosterAddress, brotliCompressionLevel)
	dataGas := arbmath.BigDiv(dataCost, header.BaseFee)
	if tx.Gas() < intrinsic+dataGas.Uint64() {
		return core.ErrIntrinsicGas
	}
	return nil
}

func (c *TxPreChecker) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	block := c.bc.CurrentBlock()
	statedb, err := c.bc.StateAt(block.Root)
	if err != nil {
		return err
	}
	arbos, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		return err
	}
	err = PreCheckTx(c.bc, c.bc.Config(), block, statedb, arbos, tx, options, c.config())
	if err != nil {
		return err
	}
	sender, err := types.Sender(types.MakeSigner(c.bc.Config(), block.Number, block.Time, arbos.ArbOSVersion()), tx)
	if err != nil {
		return err
	}
	if err := c.checkFilteredAddresses(tx, sender, block); err != nil {
		return err
	}
	return c.TransactionPublisher.PublishTransaction(ctx, tx, options)
}

func (c *TxPreChecker) PublishExpressLaneTransaction(ctx context.Context, msg *timeboost.ExpressLaneSubmission) error {
	if msg == nil || msg.Transaction == nil {
		return timeboost.ErrMalformedData
	}
	if c.expressLaneTracker == nil {
		log.Error("ExpressLaneTracker not properly initialized in TxPreChecker, rejecting transaction.", "msg", msg)
		return errors.New("express lane server misconfiguration")
	}
	err := c.expressLaneTracker.ValidateExpressLaneTx(msg)
	if err != nil {
		return err
	}

	block := c.bc.CurrentBlock()
	statedb, err := c.bc.StateAt(block.Root)
	if err != nil {
		return err
	}
	arbos, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		return err
	}
	err = PreCheckTx(c.bc, c.bc.Config(), block, statedb, arbos, msg.Transaction, msg.Options, c.config())
	if err != nil {
		return err
	}
	sender, err := types.Sender(types.MakeSigner(c.bc.Config(), block.Number, block.Time, arbos.ArbOSVersion()), msg.Transaction)
	if err != nil {
		return err
	}
	if err := c.checkFilteredAddresses(msg.Transaction, sender, block); err != nil {
		return err
	}
	return c.TransactionPublisher.PublishExpressLaneTransaction(ctx, msg)
}

func (c *TxPreChecker) PublishAuctionResolutionTransaction(ctx context.Context, tx *types.Transaction) error {
	block := c.bc.CurrentBlock()
	statedb, err := c.bc.StateAt(block.Root)
	if err != nil {
		return err
	}
	arbos, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		return err
	}
	err = PreCheckTx(c.bc, c.bc.Config(), block, statedb, arbos, tx, nil, c.config())
	if err != nil {
		return err
	}
	sender, err := types.Sender(types.MakeSigner(c.bc.Config(), block.Number, block.Time, arbos.ArbOSVersion()), tx)
	if err != nil {
		return err
	}
	if err := c.checkFilteredAddresses(tx, sender, block); err != nil {
		return err
	}
	return c.TransactionPublisher.PublishAuctionResolutionTransaction(ctx, tx)
}

func (c *TxPreChecker) SetExpressLaneTracker(tracker *timeboost.ExpressLaneTracker) {
	c.expressLaneTracker = tracker
}

func (c *TxPreChecker) SetAddressChecker(checker state.AddressChecker) {
	c.addressChecker = checker
}

func (c *TxPreChecker) SetEventFilter(filter *eventfilter.EventFilter) {
	c.eventFilter = filter
}

func dryRunCheckFilteredAddresses(
	statedb *state.StateDB, evm *vm.EVM, signer types.Signer, runCtx *core.MessageRunContext,
	header *types.Header, tx *types.Transaction, txIndex int, gasLimit uint64,
	resultFilter func(*core.ExecutionResult) error,
) error {
	msg, err := core.TransactionToMessage(tx, signer, header.BaseFee, runCtx)
	if err != nil {
		return err
	}
	msg.GasLimit = gasLimit
	// Zero gas prices and skip nonce/balance checks — we only care about
	// which addresses are touched.
	msg.GasPrice = new(big.Int)
	msg.GasFeeCap = new(big.Int)
	msg.GasTipCap = new(big.Int)
	msg.SkipNonceChecks = true
	msg.SkipTransactionChecks = true

	gasPool := core.GasPool(gasLimit)
	var usedGas uint64
	statedb.SetTxContext(tx.Hash(), txIndex)
	_, _, err = core.ApplyTransactionWithEVM(
		msg, &gasPool, statedb, header.Number, header.Hash(), header.Time,
		tx, &usedGas, evm, resultFilter,
	)
	if errors.Is(err, state.ErrArbTxFilter) {
		txPreCheckerAddressFilterRejectedCounter.Inc(1)
		return err
	}
	// Other execution errors are ignored since the pre-check is only concerned
	// with address filtering results, not with exact execution results.
	return nil
}

// touchScheduledRetryableAddresses touches From/To of scheduled ArbitrumRetryTx.
func touchScheduledRetryableAddresses(statedb *state.StateDB, scheduledTxes types.Transactions) {
	for _, scheduledTx := range scheduledTxes {
		if inner, ok := scheduledTx.GetInner().(*types.ArbitrumRetryTx); ok {
			statedb.TouchAddress(inner.From)
			if inner.To != nil {
				statedb.TouchAddress(*inner.To)
			}
		}
	}
}

func (c *TxPreChecker) checkFilteredAddresses(tx *types.Transaction, sender common.Address, header *types.Header) error {
	if c.addressChecker == nil {
		return nil
	}
	statedb, err := c.bc.StateAt(header.Root)
	if err != nil {
		return err
	}
	statedb.SetAddressChecker(c.addressChecker)

	blockContext := core.NewEVMBlockContext(header, c.bc, &header.Coinbase)
	signer := types.MakeSigner(c.bc.Config(), header.Number, header.Time, blockContext.ArbOSVersion)
	runCtx := core.NewMessageEthcallContext()
	// NoBaseFee skips the base fee comparison when gas fields are zeroed.
	evm := vm.NewEVM(blockContext, statedb, c.bc.Config(), vm.Config{NoBaseFee: true})

	var scheduledTxes types.Transactions
	err = dryRunCheckFilteredAddresses(statedb, evm, signer, runCtx, header, tx, 0, tx.Gas(),
		func(result *core.ExecutionResult) error {
			touchAddresses(statedb, c.eventFilter, tx, sender)
			touchScheduledRetryableAddresses(statedb, result.ScheduledTxes)
			if statedb.IsAddressFiltered() {
				return state.ErrArbTxFilter
			}
			scheduledTxes = result.ScheduledTxes
			return nil
		},
	)
	if err != nil {
		return err
	}

	// Process scheduled redeems in FIFO order including cascading redeems.
	// We replicate the loop from ProduceBlockAdvanced because we only need
	// EVM execution + address checking, without block-production overhead.
	txIndex := 1
	for len(scheduledTxes) > 0 {
		redeemTx := scheduledTxes[0]
		scheduledTxes = scheduledTxes[1:]
		redeemSender, err := types.Sender(signer, redeemTx)
		if err != nil {
			log.Warn("failed to recover redeem sender in address filter", "err", err, "txHash", redeemTx.Hash())
			continue
		}
		err = dryRunCheckFilteredAddresses(statedb, evm, signer, runCtx, header, redeemTx, txIndex, redeemTx.Gas(),
			func(result *core.ExecutionResult) error {
				touchAddresses(statedb, c.eventFilter, redeemTx, redeemSender)
				touchScheduledRetryableAddresses(statedb, result.ScheduledTxes)
				if statedb.IsAddressFiltered() {
					return state.ErrArbTxFilter
				}
				scheduledTxes = append(scheduledTxes, result.ScheduledTxes...)
				return nil
			},
		)
		if err != nil {
			return err
		}
		txIndex++
	}

	return nil
}
