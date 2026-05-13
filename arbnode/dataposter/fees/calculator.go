// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package fees

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/dataposter/metrics"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/floatmath"
)

var MinRbfIncrease = BlobSplit[arbmath.Bips] {
	Blob: arbmath.OneInBips * 2,
	NonBlob: arbmath.OneInBips * 11 / 10,
}

type feeCalculatorOpts struct {
	minTipCapGwei float64
	maxTipCapGwei float64
	minRbfIncrease arbmath.Bips
}

type feeCalculator interface {
	opts() *feeCalculatorOpts
	currentBlobFee(ctx context.Context) (*big.Int, error)
}

type blobFeeCalculator struct {
	o feeCalculatorOpts
	dp dataPoster
	latestHeader *types.Header
}

func newBlobFeeCalculator(dp dataPoster, latestHeader *types.Header) feeCalculator {
	config := dp.Config()

	return &blobFeeCalculator {
		dp: dp,
		latestHeader: latestHeader,

		o: feeCalculatorOpts {
			minTipCapGwei: config.MinBlobTxTipCapGwei,
			maxTipCapGwei: config.MaxBlobTxTipCapGwei,
			minRbfIncrease: MinRbfIncrease.Blob,
		},
	}
}

func (c *blobFeeCalculator) opts() *feeCalculatorOpts {
	return &c.o
}

func (c *blobFeeCalculator) currentBlobFee(ctx context.Context) (*big.Int, error) {
	if c.latestHeader.ExcessBlobGas == nil || c.latestHeader.BlobGasUsed == nil {
		return nil, fmt.Errorf(
			"latest parent chain block %v missing ExcessBlobGas or BlobGasUsed but blobs were specified in data poster transaction "+
				"(either the parent chain node is not synced or the EIP-4844 was improperly activated)",
			c.latestHeader.Number,
		)
	}

	currentBlobFee, err := c.dp.ParentChain().BlobFeePerByte(ctx, c.latestHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob base fee: %w", err)
	}
	return currentBlobFee, nil
}

type nonBlobFeeCalculator struct {
	o feeCalculatorOpts
}

func newNonBlobFeeCalculator(dp dataPoster) feeCalculator {
	config := dp.Config()

	return &nonBlobFeeCalculator {
		o: feeCalculatorOpts {
			minTipCapGwei: config.MinTipCapGwei,
			maxTipCapGwei: config.MaxTipCapGwei,
			minRbfIncrease: MinRbfIncrease.NonBlob,
		},
	}
}

func (c *nonBlobFeeCalculator) opts() *feeCalculatorOpts {
	return &c.o
}

func (*nonBlobFeeCalculator) currentBlobFee(ctx context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}

func EvalMaxFeeCapExpr(dp dataPoster, backlogOfBatches uint64, elapsed time.Duration) (*big.Int, error) {
	config := dp.Config()
	parameters := map[string]any{
		"BacklogOfBatches":      float64(backlogOfBatches),
		"UrgencyGWei":           config.UrgencyGwei,
		"ElapsedTime":           float64(elapsed),
		"ElapsedTimeBase":       float64(config.ElapsedTimeBase),
		"ElapsedTimeImportance": config.ElapsedTimeImportance,
		"TargetPriceGWei":       config.TargetPriceGwei,
	}
	result, err := dp.MaxFeeCapExpression().Evaluate(parameters)
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

func FeeAndTipCaps(ctx context.Context, dp dataPoster, s *state.LockedInternalState, nonce uint64, gasLimit uint64, numBlobs uint64, lastTx *types.Transaction, dataCreatedAt time.Time, dataPosterBacklog uint64, latestHeader *types.Header) (*Caps, error) {
	config := dp.Config()
	dataPosterBacklog += dp.ExtraBacklog()

	if latestHeader.BaseFee == nil {
		return nil, fmt.Errorf("latest parent chain block %v missing BaseFee (either the parent chain does not have EIP-1559 or the parent chain node is not synced)", latestHeader.Number)
	}

	calc := (&BlobSplit[feeCalculator] {
		Blob: newBlobFeeCalculator(dp, latestHeader),
		NonBlob: newNonBlobFeeCalculator(dp),
	}).SelectIfBlobs(numBlobs > 0)
	opts := calc.opts()

	currentBlobFee, err := calc.currentBlobFee(ctx)
	if err != nil {
		return nil, err
	}

	softConfBlock := arbmath.BigSubByUint(latestHeader.Number, config.NonceRbfSoftConfs)
	softConfNonce, err := dp.Client().NonceAt(ctx, dp.Sender(), softConfBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest nonce %v blocks ago (block %v): %w", config.NonceRbfSoftConfs, softConfBlock, err)
	}
	// #nosec G115
	datapostermetrics.LatestSoftConfirmedNonceGauge.Update(int64(softConfNonce))

	suggestedTip, err := dp.Client().SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}

	newTipCap := suggestedTip
	newTipCap = arbmath.BigMax(newTipCap, floatmath.FloatToBig(opts.minTipCapGwei*params.GWei))
	newTipCap = arbmath.BigMin(newTipCap, floatmath.FloatToBig(opts.maxTipCapGwei*params.GWei))

	// Compute the max fee with normalized gas so that blob txs aren't priced differently.
	// Later, split the total cost bid into blob and non-blob fee caps.
	elapsed := time.Since(dataCreatedAt)
	maxNormalizedFeeCap, err := EvalMaxFeeCapExpr(dp, dataPosterBacklog, elapsed)
	if err != nil {
		return nil, err
	}
	normalizedGas := gasLimit + numBlobs*blobs.BlobEncodableData*params.TxDataNonZeroGasEIP2028
	targetMaxCost := arbmath.BigMulByUint(maxNormalizedFeeCap, normalizedGas)

	maxMempoolWeight := arbmath.MinInt(config.MaxMempoolWeight, config.MaxMempoolTransactions)

	latestBalance := s.Balance
	weight := arbmath.MaxInt(1, numBlobs)
	balanceForTx := allocateBalance(dp, s, nonce, softConfNonce, weight, maxMempoolWeight)

	if arbmath.BigGreaterThan(targetMaxCost, balanceForTx) {
		log.Warn(
			"lack of L1 balance prevents posting transaction with desired fee cap",
			"balance", latestBalance,
			"weight", weight,
			"maxMempoolWeight", maxMempoolWeight,
			"balanceForTransaction", balanceForTx,
			"gasLimit", gasLimit,
			"targetMaxCost", targetMaxCost,
			"nonce", nonce,
			"softConfNonce", softConfNonce,
		)
		targetMaxCost = balanceForTx
	}

	if lastTx != nil {
		// Replace by fee rules require that the tip cap is increased
		newTipCap = arbmath.BigMax(newTipCap, arbmath.BigMulByBips(lastTx.GasTipCap(), calc.opts().minRbfIncrease))
	}

	currentNonBlobFee := arbmath.BigAdd(latestHeader.BaseFee, newTipCap)
	cs := newCostSplitter(numBlobs, lastTx, calc.opts().minRbfIncrease, currentBlobFee, currentNonBlobFee)
	costs := cs.splitCost(gasLimit, targetMaxCost)

	if config.MaxFeeBidMultipleBips > 0 {
		cs.applyCap(config, &costs)
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
		"dataPosterBacklog", dataPosterBacklog,
		"nonce", nonce,
		"isReplacing", lastTx != nil,
		"balanceForTx", balanceForTx,
		"currentBaseFee", latestHeader.BaseFee,
		"newBasefeeCap", costs.NonBlob,
		"suggestedTip", suggestedTip,
		"newTipCap", newTipCap,
		"currentBlobFee", currentBlobFee,
		"newBlobFeeCap", costs.Blob,
	}

	log.Debug("calculated data poster fee and tip caps", logFields...)

	if costs.NonBlob.Sign() < 0 || newTipCap.Sign() < 0 || costs.Blob.Sign() < 0 {
		msg := "can't meet data poster fee cap obligations with current target max cost"
		log.Info(msg, logFields...)
		if lastTx != nil {
			// wait until we have a higher target max cost to replace by fee
			return &Caps {
				Fee: BlobSplit[*big.Int] {
					NonBlob: lastTx.GasFeeCap(),
					Blob: lastTx.BlobGasFeeCap(),
				},
				Tip: lastTx.GasTipCap(),
			}, nil
		} else {
			return nil, errors.New(msg)
		}
	}

	if lastTx != nil && (arbmath.BigLessThan(costs.NonBlob, currentNonBlobFee) || (numBlobs > 0 && arbmath.BigLessThan(costs.Blob, currentBlobFee))) {
		// Make sure our replacement by fee can meet the current parent chain fee demands.
		// Without this check, we'd blindly increase each fee component by the min rbf amount each time,
		// without looking at which component(s) actually need increased.
		// E.g. instead of 2x basefee and 2x blobfee, we might actually want to 4x basefee and 2x blobfee.
		// This check lets us hold off on the rbf until we are actually meet the current fee requirements,
		// which lets us move in a particular direction (biasing towards either basefee or blobfee).
		log.Info("can't meet current parent chain fees with current target max cost", logFields...)
		// wait until we have a higher target max cost to replace by fee
		return &Caps {
			Fee: BlobSplit[*big.Int] {
				NonBlob: lastTx.GasFeeCap(),
				Blob: lastTx.BlobGasFeeCap(),
			},
			Tip: lastTx.GasTipCap(),
		}, nil
	}

	// Ensure we bid at least 1 wei to prevent division by zero
	if costs.NonBlob.Sign() == 0 {
		costs.NonBlob = big.NewInt(1)
	}
	if costs.Blob.Sign() == 0 {
		costs.Blob = big.NewInt(1)
	}

	return &Caps {
		Fee: costs,
		Tip: newTipCap,
	}, nil
}
