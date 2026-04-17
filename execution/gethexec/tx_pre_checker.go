// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/arbitrum/retryables"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/gasestimator"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var (
	conditionalTxRejectedByTxPreCheckerCurrentStateCounter = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/currentstate/rejected", nil)
	conditionalTxAcceptedByTxPreCheckerCurrentStateCounter = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/currentstate/accepted", nil)
	conditionalTxRejectedByTxPreCheckerOldStateCounter     = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/oldstate/rejected", nil)
	conditionalTxAcceptedByTxPreCheckerOldStateCounter     = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/oldstate/accepted", nil)
)

const TxPreCheckerStrictnessNone uint = 0
const TxPreCheckerStrictnessAlwaysCompatible uint = 10
const TxPreCheckerStrictnessLikelyCompatible uint = 20
const TxPreCheckerStrictnessFullValidation uint = 30

const filteredTxReportDeliveryTimeout = 5 * time.Second

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
	bc                       *core.BlockChain
	config                   TxPreCheckerConfigFetcher
	expressLaneTracker       *timeboost.ExpressLaneTracker
	backend                  core.NodeInterfaceBackendAPI
	filteringReportRPCClient *FilteringReportRPCClient
}

func NewTxPreChecker(
	publisher TransactionPublisher,
	bc *core.BlockChain,
	config TxPreCheckerConfigFetcher,
	filteringReportRPCClient *FilteringReportRPCClient) *TxPreChecker {
	return &TxPreChecker{
		TransactionPublisher:     publisher,
		bc:                       bc,
		config:                   config,
		filteringReportRPCClient: filteringReportRPCClient,
	}
}

func (c *TxPreChecker) SetAPIBackend(backend core.NodeInterfaceBackendAPI) {
	c.backend = backend
}

func (c *TxPreChecker) SetFilteringReportRPCClient(client *FilteringReportRPCClient) {
	c.filteringReportRPCClient = client
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
	if err := c.checkFilteredAddresses(ctx, tx, block); err != nil {
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
	if err := c.checkFilteredAddresses(ctx, msg.Transaction, block); err != nil {
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
	if err := c.checkFilteredAddresses(ctx, tx, block); err != nil {
		return err
	}
	return c.TransactionPublisher.PublishAuctionResolutionTransaction(ctx, tx)
}

func (c *TxPreChecker) SetExpressLaneTracker(tracker *timeboost.ExpressLaneTracker) {
	c.expressLaneTracker = tracker
}

func (c *TxPreChecker) checkFilteredAddresses(ctx context.Context, tx *types.Transaction, header *types.Header) error {
	if c.backend == nil || c.backend.TxFilter() == nil || c.config().Strictness < TxPreCheckerStrictnessAlwaysCompatible {
		return nil
	}
	statedb, err := c.bc.StateAt(header.Root)
	if err != nil {
		return err
	}

	blockContext := core.NewEVMBlockContext(header, c.bc, &header.Coinbase)
	signer := types.MakeSigner(c.bc.Config(), header.Number, header.Time, blockContext.ArbOSVersion)
	msg, err := core.TransactionToMessage(tx, signer, header.BaseFee, core.NewMessageGasEstimationContext())
	if err != nil {
		return err
	}
	msg.SkipNonceChecks = true

	_, filteredAddresses, err := gasestimator.Run(ctx, msg, &gasestimator.Options{
		Config:           c.bc.Config(),
		Chain:            c.bc,
		Header:           header,
		State:            statedb,
		Backend:          c.backend,
		RunScheduledTxes: retryables.RunScheduledTxes,
	})
	if errors.Is(err, state.ErrArbTxFilter) {
		if reportErr := c.reportFilteredTx(tx, header, filteredAddresses); reportErr != nil {
			log.Error("failed to build filtered tx report", "txHash", tx.Hash(), "err", reportErr)
		}
		return err
	}
	// Other execution errors are ignored since the pre-check is only concerned
	// with address filtering results, not with exact execution results.
	return nil
}

func (c *TxPreChecker) reportFilteredTx(tx *types.Transaction, header *types.Header, filteredAddresses []filter.FilteredAddressRecord) error {
	if c.filteringReportRPCClient == nil {
		return nil
	}
	txRLP, err := tx.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal filtered tx: %w", err)
	}
	report := addressfilter.FilteredTxReport{
		ID:                uuid.Must(uuid.NewV7()).String(),
		TxHash:            tx.Hash(),
		TxRLP:             txRLP,
		FilteredAddresses: filteredAddresses,
		BlockNumber:       header.Number.Uint64(),
		ParentBlockHash:   header.ParentHash,
		PositionInBlock:   0,
		FilteredAt:        time.Now().UTC(),
		IsDelayed:         false,
		DelayedReportData: nil,
	}
	promise := c.filteringReportRPCClient.ReportFilteredTransactions([]addressfilter.FilteredTxReport{report})
	go func(txHash common.Hash) {
		ctx, cancel := context.WithTimeout(context.Background(), filteredTxReportDeliveryTimeout)
		defer cancel()
		if _, err := promise.Await(ctx); err != nil {
			log.Error("failed to deliver filtered tx report", "txHash", txHash, "err", err)
		}
	}(tx.Hash())
	return nil
}
