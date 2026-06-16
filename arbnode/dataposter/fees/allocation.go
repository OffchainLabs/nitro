// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package fees

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var big4 = big.NewInt(4)

// We split the transaction weight into three groups:
// - The first weight point gets 1/2 of the balance.
// - The first half of the weight gets 1/3 of the balance split among them.
// - The remaining weight get the remaining 1/6 of the balance split among them.
// This helps ensure batch posting is reliable under a variety of fee conditions.
// With noop storage, we don't try to replace-by-fee, so we don't need to worry about this.
func allocateBalance(cfg *config.DataPosterConfig, usingNoOpStorage bool, balance *big.Int, nonce uint64, softConfNonce uint64, weight uint64, maxMempoolWeight uint64) *big.Int {
	balanceForTx := new(big.Int).Set(balance)

	if !cfg.AllocateMempoolBalance || usingNoOpStorage {
		return balanceForTx
	}

	balancePerWeight := new(big.Int).Div(balanceForTx, common.Big2)
	balanceForTx = big.NewInt(0)
	weightRemaining := weight
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
	return balanceForTx
}

// Parameters for splitting between blob and non-blob consts
type costSplitter struct {
	numBlobs          uint64
	lastTx            *types.Transaction
	minRbfIncrease    arbmath.Bips
	currentBlobFee    *big.Int
	currentNonBlobFee *big.Int
	blobGasUsed       uint64
}

func newCostSplitter(numBlobs uint64, lastTx *types.Transaction, minRbfIncrease arbmath.Bips, currentBlobFee *big.Int, currentNonBlobFee *big.Int) *costSplitter {
	blobGasUsed := params.BlobTxBlobGasPerBlob * numBlobs
	return &costSplitter{
		numBlobs,
		lastTx,
		minRbfIncrease,
		currentBlobFee,
		currentNonBlobFee,
		blobGasUsed,
	}
}

// Divide the targetMaxCost into blob and non-blob costs.
func (cs *costSplitter) splitCost(gasLimit uint64, targetMaxCost *big.Int) BlobSplit[*big.Int] {
	currentBlobCost := arbmath.BigMulByUint(cs.currentBlobFee, cs.blobGasUsed)
	currentNonBlobCost := arbmath.BigMulByUint(cs.currentNonBlobFee, gasLimit)
	newBlobFeeCap := arbmath.BigMul(targetMaxCost, cs.currentBlobFee)
	newBlobFeeCap.Div(newBlobFeeCap, arbmath.BigAdd(currentBlobCost, currentNonBlobCost))
	if cs.lastTx != nil && cs.lastTx.BlobGasFeeCap() != nil {
		newBlobFeeCap = arbmath.BigMax(newBlobFeeCap, arbmath.BigMulByBips(cs.lastTx.BlobGasFeeCap(), cs.minRbfIncrease))
	}
	targetBlobCost := arbmath.BigMulByUint(newBlobFeeCap, cs.blobGasUsed)
	targetNonBlobCost := arbmath.BigSub(targetMaxCost, targetBlobCost)
	newBaseFeeCap := arbmath.BigDivByUint(targetNonBlobCost, gasLimit)
	if cs.lastTx != nil && cs.numBlobs > 0 && cs.lastTx.GasFeeCap().Sign() > 0 && arbmath.BigDivToBips(newBaseFeeCap, cs.lastTx.GasFeeCap()) < cs.minRbfIncrease {
		// Increase the non-blob fee cap to the minimum rbf increase
		newBaseFeeCap = arbmath.BigMulByBips(cs.lastTx.GasFeeCap(), cs.minRbfIncrease)
		newNonBlobCost := arbmath.BigMulByUint(newBaseFeeCap, gasLimit)
		// Increasing the non-blob fee cap requires lowering the blob fee cap to compensate
		baseFeeCostIncrease := arbmath.BigSub(newNonBlobCost, targetNonBlobCost)
		newBlobCost := arbmath.BigSub(targetBlobCost, baseFeeCostIncrease)
		newBlobFeeCap = arbmath.BigDivByUint(newBlobCost, cs.blobGasUsed)
	}
	return BlobSplit[*big.Int]{NonBlob: newBaseFeeCap, Blob: newBlobFeeCap}
}

// Limit the fee caps to be no greater than max(MaxFeeBidMultipleBips, minRbf)
func (cs *costSplitter) applyCap(config *config.DataPosterConfig, costs *BlobSplit[*big.Int]) {
	maxNonBlobFee := arbmath.BigMulByUBips(cs.currentNonBlobFee, config.MaxFeeBidMultipleBips)
	if cs.lastTx != nil {
		maxNonBlobFee = arbmath.BigMax(maxNonBlobFee, arbmath.BigMulByBips(cs.lastTx.GasFeeCap(), cs.minRbfIncrease))
	}
	maxBlobFee := arbmath.BigMulByUBips(cs.currentBlobFee, config.MaxFeeBidMultipleBips)
	if cs.lastTx != nil && cs.lastTx.BlobGasFeeCap() != nil {
		maxBlobFee = arbmath.BigMax(maxBlobFee, arbmath.BigMulByBips(cs.lastTx.BlobGasFeeCap(), cs.minRbfIncrease))
	}
	costs.NonBlob = arbmath.BigMin(costs.NonBlob, maxNonBlobFee)
	costs.Blob = arbmath.BigMin(costs.Blob, maxBlobFee)
}
