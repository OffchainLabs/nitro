// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/l1pricing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
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

	l1Estimate, err := arbGasInfo.GetL1GasPriceEstimate(callOpts)
	Require(t, err)
	networkBefore := GetBalance(t, ctx, l2client, networkFeeAccount)
	seqBefore := GetBalance(t, ctx, l2client, l1pricing.SequencerAddress)

	l2info.GasPrice = GetBaseFee(t, l2client, ctx)
	tx, receipt := TransferBalance(t, "Faucet", "Faucet", big.NewInt(0), l2info, l2client, ctx)

	networkAfter := GetBalance(t, ctx, l2client, networkFeeAccount)
	seqAfter := GetBalance(t, ctx, l2client, l1pricing.SequencerAddress)

	networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
	seqRevenue := arbmath.BigSub(seqAfter, seqBefore)

	gasUsedForL2 := receipt.GasUsed - receipt.GasUsedForL1

	if !arbmath.BigEquals(networkRevenue, arbmath.BigMulByUint(tx.GasPrice(), gasUsedForL2)) {
		Fail(t, "network didn't receive expected payment")
	}

	paidBytes := arbmath.BigDiv(seqRevenue, l1Estimate).Uint64() / params.TxDataNonZeroGasEIP2028

	txBin, err := tx.MarshalBinary()
	Require(t, err)
	compressed, err := arbcompress.CompressFast(txBin)
	Require(t, err)

	_ = paidBytes
	_ = compressed
	// if uint64(len(compressed)) != paidBytes {
	//	t.Fatal("unexpected number of bytes paid for")
	//}
}
