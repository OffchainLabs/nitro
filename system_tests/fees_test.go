// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l1pricing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

func TestSequencerFeePaid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, _, l2client, _, _, _, stack := CreateTestNodeOnL1(t, ctx, true)
	defer stack.Close()

	callOpts := l2info.GetDefaultCallOpts("Owner", ctx)

	// get the network fee account
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), l2client)
	Require(t, err, "could not deploy ArbOwner contract")
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), l2client)
	Require(t, err, "could not deploy ArbOwner contract")
	networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
	Require(t, err, "could not get the network fee account")

	l1Estimate, err := arbGasInfo.GetL1BaseFeeEstimate(callOpts)
	Require(t, err)
	networkBefore := GetBalance(t, ctx, l2client, networkFeeAccount)

	l2info.GasPrice = GetBaseFee(t, l2client, ctx)
	tx, receipt := TransferBalance(t, "Faucet", "Faucet", big.NewInt(0), l2info, l2client, ctx)
	txSize := compressedTxSize(t, tx)

	networkAfter := GetBalance(t, ctx, l2client, networkFeeAccount)
	l1Charge := arbmath.BigMulByUint(l2info.GasPrice, receipt.GasUsedForL1)

	networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
	gasUsedForL2 := receipt.GasUsed - receipt.GasUsedForL1
	if !arbmath.BigEquals(networkRevenue, arbmath.BigMulByUint(tx.GasPrice(), gasUsedForL2)) {
		Fail(t, "network didn't receive expected payment")
	}

	l1GasBought := arbmath.BigDiv(l1Charge, l1Estimate).Uint64()
	l1GasActual := txSize * params.TxDataNonZeroGasEIP2028

	colors.PrintBlue("bytes ", l1GasBought/params.TxDataNonZeroGasEIP2028, txSize)

	if l1GasBought != l1GasActual {
		Fail(t, "the sequencer's future revenue does not match its costs", l1GasBought, l1GasActual)
	}
}

func TestSequencerPriceAdjusts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig := params.ArbitrumDevTestChainConfig()
	conf := arbnode.ConfigDefaultL1Test()
	conf.DelayedSequencer.FinalizeDistance = 1

	l2info, node, l2client, _, _, l1client, stack := CreateTestNodeOnL1WithConfig(t, ctx, true, conf, chainConfig)
	defer stack.Close()

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), l2client)
	Require(t, err)
	lastEstimate, err := arbGasInfo.GetL1BaseFeeEstimate(&bind.CallOpts{Context: ctx})
	Require(t, err)
	lastBatchCount, err := node.InboxTracker.GetBatchCount()
	Require(t, err)
	l1Header, err := l1client.HeaderByNumber(ctx, nil)
	Require(t, err)

	sequencerBalanceBefore := GetBalance(t, ctx, l2client, l1pricing.BatchPosterAddress)
	timesPriceAdjusted := 0

	colors.PrintBlue("Initial values")
	colors.PrintBlue("    L1 base fee ", l1Header.BaseFee)
	colors.PrintBlue("    L1 estimate ", lastEstimate)

	for i := 0; i < 128; i++ {
		tx, receipt := TransferBalance(t, "Owner", "Owner", common.Big1, l2info, l2client, ctx)
		header, err := l2client.HeaderByHash(ctx, receipt.BlockHash)
		Require(t, err)

		units := compressedTxSize(t, tx) * params.TxDataNonZeroGasEIP2028
		currEstimate := arbmath.BigDivByUint(arbmath.BigMulByUint(header.BaseFee, receipt.GasUsedForL1), units)

		if !arbmath.BigEquals(lastEstimate, currEstimate) {
			l1Header, err = l1client.HeaderByNumber(ctx, nil)
			Require(t, err)

			callOpts := &bind.CallOpts{Context: ctx, BlockNumber: receipt.BlockNumber}
			trueEstimate, err := arbGasInfo.GetL1BaseFeeEstimate(callOpts)
			Require(t, err)

			colors.PrintGrey("ArbOS updated its L1 estimate")
			colors.PrintGrey("    L1 base fee ", l1Header.BaseFee)
			colors.PrintGrey("    L1 estimate ", lastEstimate, " ➤ ", currEstimate, " = ", trueEstimate)

			oldDiff := arbmath.BigAbs(arbmath.BigSub(lastEstimate, l1Header.BaseFee))
			newDiff := arbmath.BigAbs(arbmath.BigSub(trueEstimate, l1Header.BaseFee))

			if arbmath.BigGreaterThan(newDiff, oldDiff) {
				Fail(t, "L1 gas price estimate should tend toward the basefee")
			}
			if !arbmath.BigEquals(trueEstimate, currEstimate) {
				Fail(t, "New L1 estimate does not match receipt")
			}
			if arbmath.BigEquals(trueEstimate, common.Big0) {
				Fail(t, "Estimate is zero", i)
			}
			lastEstimate = trueEstimate
			timesPriceAdjusted++
		}

		if i%16 == 0 {
			// see that the inbox advances

			for j := 16; j > 0; j-- {
				newBatchCount, err := node.InboxTracker.GetBatchCount()
				Require(t, err)
				if newBatchCount > lastBatchCount {
					colors.PrintGrey("posted new batch ", newBatchCount)
					lastBatchCount = newBatchCount
					break
				}
				if j == 1 {
					Fail(t, "batch count didn't update in time")
				}
				time.Sleep(time.Millisecond * 100)
			}
		}
	}

	sequencerBalanceAfter := GetBalance(t, ctx, l2client, l1pricing.BatchPosterAddress)
	colors.PrintMint("sequencer balance ", sequencerBalanceBefore, " ➤ ", sequencerBalanceAfter)
	colors.PrintMint("price changes     ", timesPriceAdjusted)

	if timesPriceAdjusted == 0 {
		Fail(t, "L1 gas price estimate never adjusted")
	}
	if !arbmath.BigGreaterThan(sequencerBalanceAfter, sequencerBalanceBefore) {
		Fail(t, "sequencer didn't get paid")
	}
}

func compressedTxSize(t *testing.T, tx *types.Transaction) uint64 {
	txBin, err := tx.MarshalBinary()
	Require(t, err)
	compressed, err := arbcompress.CompressFast(txBin)
	Require(t, err)
	return uint64(len(compressed))
}
