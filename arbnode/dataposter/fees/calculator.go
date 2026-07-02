// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package fees

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/Knetic/govaluate"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/floatmath"
)

var MinRbfIncrease = BlobSplit[arbmath.Bips]{
	Blob:    arbmath.OneInBips * 2,
	NonBlob: arbmath.OneInBips * 11 / 10,
}

// FeeCalcOpts contains all inputs needed for fee calculation.
// No RPC calls are made inside the fee calculation — the caller
// is responsible for fetching SoftConfNonce, SuggestedTip, and
// CurrentBlobFee before constructing this struct.
type FeeCalcOpts struct {
	// DataPoster environment
	Config              *config.DataPosterConfig
	Balance             *big.Int
	SoftConfNonce       uint64
	SuggestedTip        *big.Int
	CurrentBlobFee      *big.Int // zero if no blobs
	ExtraBacklog        uint64
	MaxFeeCapExpression *govaluate.EvaluableExpression
	UsingNoOpStorage    bool

	// Per-transaction
	Nonce             uint64
	GasLimit          uint64
	NumBlobs          uint64
	LastTx            *types.Transaction // nil for new tx, set for RBF
	DataCreatedAt     time.Time
	DataPosterBacklog uint64
	LatestHeader      *types.Header
}

func EvalMaxFeeCapExpr(expression *govaluate.EvaluableExpression, cfg *config.DataPosterConfig, backlogOfBatches uint64, elapsed time.Duration) (*big.Int, error) {
	parameters := map[string]any{
		"BacklogOfBatches":      float64(backlogOfBatches),
		"UrgencyGWei":           cfg.UrgencyGwei,
		"ElapsedTime":           float64(elapsed),
		"ElapsedTimeBase":       float64(cfg.ElapsedTimeBase),
		"ElapsedTimeImportance": cfg.ElapsedTimeImportance,
		"TargetPriceGWei":       cfg.TargetPriceGwei,
	}
	result, err := expression.Evaluate(parameters)
	if err != nil {
		return nil, fmt.Errorf("error evaluating maxFeeCapExpression: %w", err)
	}
	resultFloat, ok := result.(float64)
	if !ok {
		// This shouldn't be possible because we only pass in float64s as arguments
		return nil, fmt.Errorf("maxFeeCapExpression evaluated to non-float64: %v", result)
	}
	// 1e9 gwei gas price is practically speaking an infinite gas price, so we cap it there.
	// This also allows the formula to return positive infinity safely.
	resultFloat = math.Min(resultFloat, 1e9)
	resultBig := floatmath.FloatToBig(resultFloat * params.GWei)
	if resultBig == nil {
		return nil, fmt.Errorf("maxFeeCapExpression evaluated to float64 not convertible to integer: %v", resultFloat)
	}
	if resultBig.Sign() < 0 {
		return nil, fmt.Errorf("maxFeeCapExpression evaluated < 0: %v", resultFloat)
	}
	return resultBig, nil
}

func FeeAndTipCaps(opts *FeeCalcOpts) (*Caps, error) {
	cfg := opts.Config
	backlog := opts.DataPosterBacklog + opts.ExtraBacklog

	if opts.LatestHeader.BaseFee == nil {
		return nil, fmt.Errorf("latest parent chain block %v missing BaseFee (either the parent chain does not have EIP-1559 or the parent chain node is not synced)", opts.LatestHeader.Number)
	}

	hasBlobs := opts.NumBlobs > 0
	minTipCapGwei := cfg.MinTipCapGwei
	maxTipCapGwei := cfg.MaxTipCapGwei
	minRbfIncrease := MinRbfIncrease.NonBlob
	if hasBlobs {
		minTipCapGwei = cfg.MinBlobTxTipCapGwei
		maxTipCapGwei = cfg.MaxBlobTxTipCapGwei
		minRbfIncrease = MinRbfIncrease.Blob
	}

	currentBlobFee := opts.CurrentBlobFee
	if currentBlobFee == nil {
		currentBlobFee = big.NewInt(0)
	}

	newTipCap := new(big.Int).Set(opts.SuggestedTip)
	newTipCap = arbmath.BigMax(newTipCap, floatmath.FloatToBig(minTipCapGwei*params.GWei))
	newTipCap = arbmath.BigMin(newTipCap, floatmath.FloatToBig(maxTipCapGwei*params.GWei))

	// Compute the max fee with normalized gas so that blob txs aren't priced differently.
	// Later, split the total cost bid into blob and non-blob fee caps.
	elapsed := time.Since(opts.DataCreatedAt)
	maxNormalizedFeeCap, err := EvalMaxFeeCapExpr(opts.MaxFeeCapExpression, cfg, backlog, elapsed)
	if err != nil {
		return nil, err
	}
	normalizedGas := opts.GasLimit + opts.NumBlobs*blobs.BlobEncodableData*params.TxDataNonZeroGasEIP2028
	targetMaxCost := arbmath.BigMulByUint(maxNormalizedFeeCap, normalizedGas)

	maxMempoolWeight := arbmath.MinInt(cfg.MaxMempoolWeight, cfg.MaxMempoolTransactions)

	weight := arbmath.MaxInt(1, opts.NumBlobs)
	balanceForTx := allocateBalance(cfg, opts.UsingNoOpStorage, opts.Balance, opts.Nonce, opts.SoftConfNonce, weight, maxMempoolWeight)

	if arbmath.BigGreaterThan(targetMaxCost, balanceForTx) {
		log.Warn(
			"lack of L1 balance prevents posting transaction with desired fee cap",
			"balance", opts.Balance,
			"weight", weight,
			"maxMempoolWeight", maxMempoolWeight,
			"balanceForTransaction", balanceForTx,
			"gasLimit", opts.GasLimit,
			"targetMaxCost", targetMaxCost,
			"nonce", opts.Nonce,
			"softConfNonce", opts.SoftConfNonce,
		)
		targetMaxCost = balanceForTx
	}

	if opts.LastTx != nil {
		// Replace by fee rules require that the tip cap is increased
		newTipCap = arbmath.BigMax(newTipCap, arbmath.BigMulByBips(opts.LastTx.GasTipCap(), minRbfIncrease))
	}

	currentNonBlobFee := arbmath.BigAdd(opts.LatestHeader.BaseFee, newTipCap)
	cs := newCostSplitter(opts.NumBlobs, opts.LastTx, minRbfIncrease, currentBlobFee, currentNonBlobFee)
	costs := cs.splitCost(opts.GasLimit, targetMaxCost)

	if cfg.MaxFeeBidMultipleBips > 0 {
		cs.applyCap(cfg, &costs)
	}

	if arbmath.BigGreaterThan(newTipCap, costs.NonBlob) {
		log.Info(
			"reducing new tip cap to new basefee cap",
			"proposedTipCap", newTipCap,
			"newBasefeeCap", costs.NonBlob,
		)
		newTipCap = new(big.Int).Set(costs.NonBlob)
	}

	logFields := []any{
		"targetMaxCost", targetMaxCost,
		"elapsed", elapsed,
		"dataPosterBacklog", backlog,
		"nonce", opts.Nonce,
		"isReplacing", opts.LastTx != nil,
		"balanceForTx", balanceForTx,
		"currentBaseFee", opts.LatestHeader.BaseFee,
		"newBasefeeCap", costs.NonBlob,
		"suggestedTip", opts.SuggestedTip,
		"newTipCap", newTipCap,
		"currentBlobFee", currentBlobFee,
		"newBlobFeeCap", costs.Blob,
	}

	log.Debug("calculated data poster fee and tip caps", logFields...)

	if costs.NonBlob.Sign() < 0 || newTipCap.Sign() < 0 || costs.Blob.Sign() < 0 {
		msg := "can't meet data poster fee cap obligations with current target max cost"
		log.Info(msg, logFields...)
		if opts.LastTx != nil {
			// wait until we have a higher target max cost to replace by fee
			return &Caps{
				Fee: BlobSplit[*big.Int]{
					NonBlob: opts.LastTx.GasFeeCap(),
					Blob:    opts.LastTx.BlobGasFeeCap(),
				},
				Tip: opts.LastTx.GasTipCap(),
			}, nil
		} else {
			return nil, errors.New(msg)
		}
	}

	if opts.LastTx != nil && (arbmath.BigLessThan(costs.NonBlob, currentNonBlobFee) || (hasBlobs && arbmath.BigLessThan(costs.Blob, currentBlobFee))) {
		// Make sure our replacement by fee can meet the current parent chain fee demands.
		// Without this check, we'd blindly increase each fee component by the min rbf amount each time,
		// without looking at which component(s) actually need increased.
		// E.g. instead of 2x basefee and 2x blobfee, we might actually want to 4x basefee and 2x blobfee.
		// This check lets us hold off on the rbf until we are actually meet the current fee requirements,
		// which lets us move in a particular direction (biasing towards either basefee or blobfee).
		log.Info("can't meet current parent chain fees with current target max cost", logFields...)
		// wait until we have a higher target max cost to replace by fee
		return &Caps{
			Fee: BlobSplit[*big.Int]{
				NonBlob: opts.LastTx.GasFeeCap(),
				Blob:    opts.LastTx.BlobGasFeeCap(),
			},
			Tip: opts.LastTx.GasTipCap(),
		}, nil
	}

	// Ensure we bid at least 1 wei to prevent division by zero
	if costs.NonBlob.Sign() == 0 {
		costs.NonBlob = big.NewInt(1)
	}
	if costs.Blob.Sign() == 0 {
		costs.Blob = big.NewInt(1)
	}

	return &Caps{
		Fee: costs,
		Tip: newTipCap,
	}, nil
}
