// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	flag "github.com/spf13/pflag"
)

var (
	conditionalTxRejectedByTxPreCheckerCurrentStateCounter = metrics.NewRegisteredCounter("arb/txprechecker/condtionaltx/currentstate/rejected", nil)
	conditionalTxAcceptedByTxPreCheckerCurrentStateCounter = metrics.NewRegisteredCounter("arb/txprechecker/condtionaltx/currentstate/accepted", nil)
	conditionalTxRejectedByTxPreCheckerOldStateCounter     = metrics.NewRegisteredCounter("arb/txprechecker/condtionaltx/oldstate/rejected", nil)
	conditionalTxAcceptedByTxPreCheckerOldStateCounter     = metrics.NewRegisteredCounter("arb/txprechecker/condtionaltx/oldstate/accepted", nil)
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
	Strictness:             TxPreCheckerStrictnessNone,
	RequiredStateAge:       2,
	RequiredStateMaxBlocks: 4,
}

func TxPreCheckerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint(prefix+".strictness", DefaultTxPreCheckerConfig.Strictness, "how strict to be when checking txs before forwarding them. 0 = accept anything, "+
		"10 = should never reject anything that'd succeed, 20 = likely won't reject anything that'd succeed, "+
		"30 = full validation which may reject txs that would succeed")
	f.Int64(prefix+".required-state-age", DefaultTxPreCheckerConfig.RequiredStateAge, "how long ago should the storage conditions from eth_SendRawTransactionConditional be true, 0 = don't check old state")
	f.Uint(prefix+".required-state-max-blocks", DefaultTxPreCheckerConfig.RequiredStateMaxBlocks, "maximum number of blocks to look back while looking for the <required-state-age> seconds old state, 0 = don't limit the search")
}

type TxPreChecker struct {
	TransactionPublisher
	bc     *core.BlockChain
	config TxPreCheckerConfigFetcher
}

func NewTxPreChecker(publisher TransactionPublisher, bc *core.BlockChain, config TxPreCheckerConfigFetcher) *TxPreChecker {
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
	sender, err := types.Sender(types.MakeSigner(chainConfig, header.Number), tx)
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
	intrinsic, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.To() == nil, chainConfig.IsHomestead(header.Number), chainConfig.IsIstanbul(header.Number), chainConfig.IsShanghai(header.Time, extraInfo.ArbOSFormatVersion))
	if err != nil {
		return err
	}
	if tx.Gas() < intrinsic {
		return core.ErrIntrinsicGas
	}
	if config.Strictness < TxPreCheckerStrictnessLikelyCompatible {
		return nil
	}
	balance := statedb.GetBalance(sender)
	cost := tx.Cost()
	if arbmath.BigLessThan(balance, cost) {
		return fmt.Errorf("%w: address %v have %v want %v", core.ErrInsufficientFunds, sender, balance, cost)
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
			if headerreader.HeadersEqual(oldHeader, header) {
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
	if config.Strictness >= TxPreCheckerStrictnessFullValidation && tx.Nonce() > stateNonce {
		return MakeNonceError(sender, tx.Nonce(), stateNonce)
	}
	dataCost, _ := arbos.L1PricingState().GetPosterInfo(tx, l1pricing.BatchPosterAddress)
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
	return c.TransactionPublisher.PublishTransaction(ctx, tx, options)
}
