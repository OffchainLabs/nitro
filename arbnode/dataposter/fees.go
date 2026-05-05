// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dataposter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/floatmath"
)

const minNonBlobRbfIncrease = arbmath.OneInBips * 11 / 10
const minBlobRbfIncrease = arbmath.OneInBips * 2

// evalMaxFeeCapExpr uses MaxFeeCapFormula from config to calculate the expression's result by plugging in appropriate parameter values
// backlogOfBatches should already include extraBacklog
func (p *DataPoster) evalMaxFeeCapExpr(backlogOfBatches uint64, elapsed time.Duration) (*big.Int, error) {
	config := p.config()
	parameters := map[string]any{
		"BacklogOfBatches":      float64(backlogOfBatches),
		"UrgencyGWei":           config.UrgencyGwei,
		"ElapsedTime":           float64(elapsed),
		"ElapsedTimeBase":       float64(config.ElapsedTimeBase),
		"ElapsedTimeImportance": config.ElapsedTimeImportance,
		"TargetPriceGWei":       config.TargetPriceGwei,
	}
	result, err := p.maxFeeCapExpression.Evaluate(parameters)
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

var big4 = big.NewInt(4)

// The dataPosterBacklog argument should *not* include extraBacklog (it's added in this function)
func (p *DataPoster) feeAndTipCaps(ctx context.Context, s *state.LockedInternalState, nonce uint64, gasLimit uint64, numBlobs uint64, lastTx *types.Transaction, dataCreatedAt time.Time, dataPosterBacklog uint64, latestHeader *types.Header) (*big.Int, *big.Int, *big.Int, error) {
	config := p.config()
	dataPosterBacklog += p.extraBacklog()

	if latestHeader.BaseFee == nil {
		return nil, nil, nil, fmt.Errorf("latest parent chain block %v missing BaseFee (either the parent chain does not have EIP-1559 or the parent chain node is not synced)", latestHeader.Number)
	}
	currentBlobFee := big.NewInt(0)
	if numBlobs > 0 {
		if latestHeader.ExcessBlobGas != nil && latestHeader.BlobGasUsed != nil {
			var err error
			currentBlobFee, err = p.parentChain.BlobFeePerByte(ctx, latestHeader)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to get blob base fee: %w", err)
			}
		} else {
			return nil, nil, nil, fmt.Errorf(
				"latest parent chain block %v missing ExcessBlobGas or BlobGasUsed but blobs were specified in data poster transaction "+
					"(either the parent chain node is not synced or the EIP-4844 was improperly activated)",
				latestHeader.Number,
			)
		}
	}
	softConfBlock := arbmath.BigSubByUint(latestHeader.Number, config.NonceRbfSoftConfs)
	softConfNonce, err := p.client.NonceAt(ctx, p.Sender(), softConfBlock)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get latest nonce %v blocks ago (block %v): %w", config.NonceRbfSoftConfs, softConfBlock, err)
	}
	// #nosec G115
	latestSoftConfirmedNonceGauge.Update(int64(softConfNonce))

	suggestedTip, err := p.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	minTipCapGwei, maxTipCapGwei, minRbfIncrease := config.MinTipCapGwei, config.MaxTipCapGwei, minNonBlobRbfIncrease
	if numBlobs > 0 {
		minTipCapGwei, maxTipCapGwei, minRbfIncrease = config.MinBlobTxTipCapGwei, config.MaxBlobTxTipCapGwei, minBlobRbfIncrease
	}
	newTipCap := suggestedTip
	newTipCap = arbmath.BigMax(newTipCap, floatmath.FloatToBig(minTipCapGwei*params.GWei))
	newTipCap = arbmath.BigMin(newTipCap, floatmath.FloatToBig(maxTipCapGwei*params.GWei))

	// Compute the max fee with normalized gas so that blob txs aren't priced differently.
	// Later, split the total cost bid into blob and non-blob fee caps.
	elapsed := time.Since(dataCreatedAt)
	maxNormalizedFeeCap, err := p.evalMaxFeeCapExpr(dataPosterBacklog, elapsed)
	if err != nil {
		return nil, nil, nil, err
	}
	normalizedGas := gasLimit + numBlobs*blobs.BlobEncodableData*params.TxDataNonZeroGasEIP2028
	targetMaxCost := arbmath.BigMulByUint(maxNormalizedFeeCap, normalizedGas)

	maxMempoolWeight := arbmath.MinInt(config.MaxMempoolWeight, config.MaxMempoolTransactions)

	latestBalance := s.Balance
	balanceForTx := new(big.Int).Set(latestBalance)
	weight := arbmath.MaxInt(1, numBlobs)
	weightRemaining := weight

	if config.AllocateMempoolBalance && !p.usingNoOpStorage {
		// We split the transaction weight into three groups:
		// - The first weight point gets 1/2 of the balance.
		// - The first half of the weight gets 1/3 of the balance split among them.
		// - The remaining weight get the remaining 1/6 of the balance split among them.
		// This helps ensure batch posting is reliable under a variety of fee conditions.
		// With noop storage, we don't try to replace-by-fee, so we don't need to worry about this.
		balancePerWeight := new(big.Int).Div(balanceForTx, common.Big2)
		balanceForTx = big.NewInt(0)
		if nonce == softConfNonce || maxMempoolWeight == 1 {
			balanceForTx.Add(balanceForTx, balancePerWeight)
			weightRemaining -= 1
		}
		if weightRemaining > 0 {
			// Compared to dividing the remaining transactions by balance equally,
			// the first half of transactions should get a 4/3 weight,
			// and the remaining half should get a 2/3 weight.
			// This makes sure the average weight is 1, and the first half of transactions
			// have twice the weight of the second half of transactions.
			// The +1 and -1 here are to account for the first transaction being handled separately.
			if nonce > softConfNonce && nonce < softConfNonce+1+(maxMempoolWeight-1)/2 {
				balancePerWeight.Mul(balancePerWeight, big4)
			} else {
				balancePerWeight.Mul(balancePerWeight, common.Big2)
			}
			balancePerWeight.Div(balancePerWeight, common.Big3)
			// After weighting, split the balance between each of the transactions
			// other than the first tx which already got half.
			// balanceForTx /= config.MaxMempoolTransactions-1
			balancePerWeight.Div(balancePerWeight, arbmath.UintToBig(maxMempoolWeight-1))
			balanceForTx.Add(balanceForTx, arbmath.BigMulByUint(balancePerWeight, weight))
		}
	}

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
		newTipCap = arbmath.BigMax(newTipCap, arbmath.BigMulByBips(lastTx.GasTipCap(), minRbfIncrease))
	}

	// Divide the targetMaxCost into blob and non-blob costs.
	currentNonBlobFee := arbmath.BigAdd(latestHeader.BaseFee, newTipCap)
	blobGasUsed := params.BlobTxBlobGasPerBlob * numBlobs
	currentBlobCost := arbmath.BigMulByUint(currentBlobFee, blobGasUsed)
	currentNonBlobCost := arbmath.BigMulByUint(currentNonBlobFee, gasLimit)
	newBlobFeeCap := arbmath.BigMul(targetMaxCost, currentBlobFee)
	newBlobFeeCap.Div(newBlobFeeCap, arbmath.BigAdd(currentBlobCost, currentNonBlobCost))
	if lastTx != nil && lastTx.BlobGasFeeCap() != nil {
		newBlobFeeCap = arbmath.BigMax(newBlobFeeCap, arbmath.BigMulByBips(lastTx.BlobGasFeeCap(), minRbfIncrease))
	}
	targetBlobCost := arbmath.BigMulByUint(newBlobFeeCap, blobGasUsed)
	targetNonBlobCost := arbmath.BigSub(targetMaxCost, targetBlobCost)
	newBaseFeeCap := arbmath.BigDivByUint(targetNonBlobCost, gasLimit)
	if lastTx != nil && numBlobs > 0 && lastTx.GasFeeCap().Sign() > 0 && arbmath.BigDivToBips(newBaseFeeCap, lastTx.GasFeeCap()) < minRbfIncrease {
		// Increase the non-blob fee cap to the minimum rbf increase
		newBaseFeeCap = arbmath.BigMulByBips(lastTx.GasFeeCap(), minRbfIncrease)
		newNonBlobCost := arbmath.BigMulByUint(newBaseFeeCap, gasLimit)
		// Increasing the non-blob fee cap requires lowering the blob fee cap to compensate
		baseFeeCostIncrease := arbmath.BigSub(newNonBlobCost, targetNonBlobCost)
		newBlobCost := arbmath.BigSub(targetBlobCost, baseFeeCostIncrease)
		newBlobFeeCap = arbmath.BigDivByUint(newBlobCost, blobGasUsed)
	}

	if config.MaxFeeBidMultipleBips > 0 {
		// Limit the fee caps to be no greater than max(MaxFeeBidMultipleBips, minRbf)
		maxNonBlobFee := arbmath.BigMulByUBips(currentNonBlobFee, config.MaxFeeBidMultipleBips)
		if lastTx != nil {
			maxNonBlobFee = arbmath.BigMax(maxNonBlobFee, arbmath.BigMulByBips(lastTx.GasFeeCap(), minRbfIncrease))
		}
		maxBlobFee := arbmath.BigMulByUBips(currentBlobFee, config.MaxFeeBidMultipleBips)
		if lastTx != nil && lastTx.BlobGasFeeCap() != nil {
			maxBlobFee = arbmath.BigMax(maxBlobFee, arbmath.BigMulByBips(lastTx.BlobGasFeeCap(), minRbfIncrease))
		}
		newBaseFeeCap = arbmath.BigMin(newBaseFeeCap, maxNonBlobFee)
		newBlobFeeCap = arbmath.BigMin(newBlobFeeCap, maxBlobFee)
	}

	if arbmath.BigGreaterThan(newTipCap, newBaseFeeCap) {
		log.Info(
			"reducing new tip cap to new basefee cap",
			"proposedTipCap", newTipCap,
			"newBasefeeCap", newBaseFeeCap,
		)
		newTipCap = new(big.Int).Set(newBaseFeeCap)
	}

	logFields := []any{
		"targetMaxCost", targetMaxCost,
		"elapsed", elapsed,
		"dataPosterBacklog", dataPosterBacklog,
		"nonce", nonce,
		"isReplacing", lastTx != nil,
		"balanceForTx", balanceForTx,
		"currentBaseFee", latestHeader.BaseFee,
		"newBasefeeCap", newBaseFeeCap,
		"suggestedTip", suggestedTip,
		"newTipCap", newTipCap,
		"currentBlobFee", currentBlobFee,
		"newBlobFeeCap", newBlobFeeCap,
	}

	log.Debug("calculated data poster fee and tip caps", logFields...)

	if newBaseFeeCap.Sign() < 0 || newTipCap.Sign() < 0 || newBlobFeeCap.Sign() < 0 {
		msg := "can't meet data poster fee cap obligations with current target max cost"
		log.Info(msg, logFields...)
		if lastTx != nil {
			// wait until we have a higher target max cost to replace by fee
			return lastTx.GasFeeCap(), lastTx.GasTipCap(), lastTx.BlobGasFeeCap(), nil
		} else {
			return nil, nil, nil, errors.New(msg)
		}
	}

	if lastTx != nil && (arbmath.BigLessThan(newBaseFeeCap, currentNonBlobFee) || (numBlobs > 0 && arbmath.BigLessThan(newBlobFeeCap, currentBlobFee))) {
		// Make sure our replacement by fee can meet the current parent chain fee demands.
		// Without this check, we'd blindly increase each fee component by the min rbf amount each time,
		// without looking at which component(s) actually need increased.
		// E.g. instead of 2x basefee and 2x blobfee, we might actually want to 4x basefee and 2x blobfee.
		// This check lets us hold off on the rbf until we are actually meet the current fee requirements,
		// which lets us move in a particular direction (biasing towards either basefee or blobfee).
		log.Info("can't meet current parent chain fees with current target max cost", logFields...)
		// wait until we have a higher target max cost to replace by fee
		return lastTx.GasFeeCap(), lastTx.GasTipCap(), lastTx.BlobGasFeeCap(), nil
	}

	// Ensure we bid at least 1 wei to prevent division by zero
	if newBaseFeeCap.Sign() == 0 {
		newBaseFeeCap = big.NewInt(1)
	}
	if newBlobFeeCap.Sign() == 0 {
		newBlobFeeCap = big.NewInt(1)
	}

	return newBaseFeeCap, newTipCap, newBlobFeeCap, nil
}

func updateTxDataGasCaps(data types.TxData, newFeeCap, newTipCap, newBlobFeeCap *big.Int) error {
	switch data := data.(type) {
	case *types.DynamicFeeTx:
		data.GasFeeCap = newFeeCap
		data.GasTipCap = newTipCap
		return nil
	case *types.BlobTx:
		var overflow bool
		data.GasFeeCap, overflow = uint256.FromBig(newFeeCap)
		if overflow {
			return fmt.Errorf("blob tx fee cap %v exceeds uint256", newFeeCap)
		}
		data.GasTipCap, overflow = uint256.FromBig(newTipCap)
		if overflow {
			return fmt.Errorf("blob tx tip cap %v exceeds uint256", newTipCap)
		}
		data.BlobFeeCap, overflow = uint256.FromBig(newBlobFeeCap)
		if overflow {
			return fmt.Errorf("blob tx blob fee cap %v exceeds uint256", newBlobFeeCap)
		}
		return nil
	default:
		return fmt.Errorf("unexpected transaction data type %T", data)
	}
}

func updateGasCaps(tx *types.Transaction, newFeeCap, newTipCap, newBlobFeeCap *big.Int) (*types.Transaction, error) {
	data := tx.GetInner()
	err := updateTxDataGasCaps(data, newFeeCap, newTipCap, newBlobFeeCap)
	if err != nil {
		return nil, err
	}
	return types.NewTx(data), nil
}
