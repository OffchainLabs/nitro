// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/util"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

func retryableSetup(t *testing.T) (
	*BlockchainTestInfo,
	*BlockchainTestInfo,
	*ethclient.Client,
	*ethclient.Client,
	*bridgegen.Inbox,
	func(*types.Receipt) common.Hash,
	context.Context,
	func(),
) {
	ctx, cancel := context.WithCancel(context.Background())
	l2info, _, l2client, l1info, _, l1client, stack := CreateTestNodeOnL1(t, ctx, true)
	l2info.GenerateAccount("User2")
	l2info.GenerateAccount("Beneficiary")
	l2info.GenerateAccount("Burn")

	delayedInbox, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(l1client, l1info.GetAddress("Bridge"), 0)
	Require(t, err)

	lookupSubmitRetryableL2TxHash := func(l1Receipt *types.Receipt) common.Hash {
		messages, err := delayedBridge.LookupMessagesInRange(ctx, l1Receipt.BlockNumber, l1Receipt.BlockNumber)
		Require(t, err)
		if len(messages) != 1 {
			Fail(t, "expected 1 message from retryable submission, found", len(messages))
		}
		txs, err := messages[0].Message.ParseL2Transactions(params.ArbitrumDevTestChainConfig().ChainID)
		Require(t, err)
		if len(txs) != 1 {
			Fail(t, "expected 1 tx from retryable submission, found", len(txs))
		}

		return txs[0].Hash()
	}

	// burn some gas so that the faucet's Callvalue + Balance never exceeds a uint256
	discard := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	TransferBalance(t, "Faucet", "Burn", discard, l2info, l2client, ctx)

	teardown := func() {

		// check the integrity of the RPC
		blockNum, err := l2client.BlockNumber(ctx)
		Require(t, err, "failed to get L2 block number")
		for number := uint64(0); number < blockNum; number++ {
			block, err := l2client.BlockByNumber(ctx, arbmath.UintToBig(number))
			Require(t, err, "failed to get L2 block", number, "of", blockNum)
			if block.Number().Uint64() != number {
				Fail(t, "block number mismatch", number, block.Number().Uint64())
			}
		}

		cancel()
		stack.Close()
	}
	return l2info, l1info, l2client, l1client, delayedInbox, lookupSubmitRetryableL2TxHash, ctx, teardown
}

func TestRetryableNoExist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, l2client := CreateTestL2(t, ctx)

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, common.Hash{})
	if err.Error() != "error NoTicketWithID()" {
		Fail(t, "didn't get expected NoTicketWithID error")
	}
}

func TestSubmitRetryableImmediateSuccess(t *testing.T) {
	l2info, l1info, l2client, l1client, delayedInbox, lookupSubmitRetryableL2TxHash, ctx, teardown := retryableSetup(t)
	defer teardown()

	user2Address := l2info.GetAddress("User2")
	beneficiaryAddress := l2info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, l2client)
	Require(t, err, "failed to deploy NodeInterface")

	// estimate the gas needed to auto-redeem the retryable
	usertxoptsL2 := l2info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL2.NoSend = true
	usertxoptsL2.GasMargin = 0
	tx, err := nodeInterface.EstimateRetryableTicket(
		&usertxoptsL2,
		usertxoptsL2.From,
		deposit,
		user2Address,
		callValue,
		beneficiaryAddress,
		beneficiaryAddress,
		[]byte{0x32, 0x42, 0x32, 0x88}, // increase the cost to beyond that of params.TxGas
	)
	Require(t, err, "failed to estimate retryable submission")
	estimate := tx.Gas()
	colors.PrintBlue("estimate: ", estimate)

	// submit & auto-redeem the retryable using the gas estimate
	usertxoptsL1 := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxoptsL1,
		user2Address,
		callValue,
		big.NewInt(1e16),
		beneficiaryAddress,
		beneficiaryAddress,
		arbmath.UintToBig(estimate),
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		[]byte{0x32, 0x42, 0x32, 0x88},
	)
	Require(t, err)

	l1receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := WaitForTx(ctx, l2client, lookupSubmitRetryableL2TxHash(l1receipt), time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t)
	}

	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if !arbmath.BigEquals(l2balance, big.NewInt(1e6)) {
		Fail(t, "Unexpected balance:", l2balance)
	}
}

func TestSubmitRetryableFailThenRetry(t *testing.T) {
	l2info, l1info, l2client, l1client, delayedInbox, lookupSubmitRetryableL2TxHash, ctx, teardown := retryableSetup(t)
	defer teardown()

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, _, simple, err := mocksgen.DeploySimple(&ownerTxOpts, l2client)
	Require(t, err)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := l2info.GetAddress("Beneficiary")
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxopts,
		simpleAddr,
		common.Big0,
		big.NewInt(1e16),
		beneficiaryAddress,
		beneficiaryAddress,
		// send enough L2 gas for intrinsic but not compute
		big.NewInt(int64(params.TxGas+params.TxDataNonZeroGasEIP2028*4)),
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		simpleABI.Methods["increment"].ID,
	)
	Require(t, err)

	l1receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := WaitForTx(ctx, l2client, lookupSubmitRetryableL2TxHash(l1receipt), time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t)
	}
	if len(receipt.Logs) != 2 {
		Fail(t, len(receipt.Logs))
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// get receipt for the auto-redeem, make sure it failed
	receipt, err = WaitForTx(ctx, l2client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fail(t, receipt.GasUsed)
	}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(&ownerTxOpts, ticketId)
	Require(t, err)
	receipt, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	retryTxId := receipt.Logs[0].Topics[2]

	// check the receipt for the retry
	receipt, err = WaitForTx(ctx, l2client, retryTxId, time.Second*1)
	Require(t, err)
	if receipt.Status != 1 {
		Fail(t, receipt.Status)
	}

	// verify that the increment happened, so we know the retry succeeded
	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err)

	if counter != 1 {
		Fail(t, "Unexpected counter:", counter)
	}
}

func TestSubmissionGasCosts(t *testing.T) {
	l2info, l1info, l2client, l1client, delayedInbox, _, ctx, teardown := retryableSetup(t)
	defer teardown()

	usertxopts := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	l2info.GenerateAccount("Refund")
	l2info.GenerateAccount("Recieve")
	faucetAddress := util.RemapL1Address(l1info.GetAddress("Faucet"))
	beneficiaryAddress := l2info.GetAddress("Beneficiary")
	feeRefundAddress := l2info.GetAddress("Refund")
	receiveAddress := l2info.GetAddress("Recieve")

	colors.PrintBlue("Faucet      ", faucetAddress)
	colors.PrintBlue("Receive     ", receiveAddress)
	colors.PrintBlue("Beneficiary ", beneficiaryAddress)
	colors.PrintBlue("Fee Refund  ", feeRefundAddress)

	fundsBeforeSubmit, err := l2client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)

	usefulGas := params.TxGas

	maxSubmissionFee := big.NewInt(1e13)
	retryableGas := arbmath.UintToBig(usefulGas + uint64(808)) // will only burn the intrinsic cost
	retryableL2CallValue := big.NewInt(1e4)
	retryableCallData := []byte{}
	excessDeposit := arbmath.BigSub(
		arbmath.BigSub(
			usertxopts.Value,
			retryableL2CallValue,
		),
		maxSubmissionFee,
	)
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxopts,
		receiveAddress,
		retryableL2CallValue,
		maxSubmissionFee,
		feeRefundAddress,
		beneficiaryAddress,
		retryableGas,
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		retryableCallData,
	)
	Require(t, err)

	l1receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)
	l2BaseFee := GetBaseFee(t, l2client, ctx)
	usedWei := arbmath.BigMulByUint(l2BaseFee, usefulGas)

	l1HeaderAfterSubmit, err := l1client.HeaderByHash(ctx, l1receipt.BlockHash)
	Require(t, err)
	l1BaseFee := l1HeaderAfterSubmit.BaseFee
	submitFee := arbmath.BigMulByUint(l1BaseFee, uint64(1400+6*len(retryableCallData)))
	submissionFeeRefund := arbmath.BigSub(maxSubmissionFee, submitFee)

	fundsAfterSubmit, err := l2client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)
	beneficiaryFunds, err := l2client.BalanceAt(ctx, beneficiaryAddress, nil)
	Require(t, err)
	refundFunds, err := l2client.BalanceAt(ctx, feeRefundAddress, nil)
	Require(t, err)
	receiveFunds, err := l2client.BalanceAt(ctx, receiveAddress, nil)
	Require(t, err)

	colors.PrintBlue("CallGas    ", retryableGas)
	colors.PrintMint("Gas cost   ", arbmath.BigMul(retryableGas, l2BaseFee))
	colors.PrintBlue("Payment    ", usertxopts.Value)

	colors.PrintMint("Faucet before ", fundsBeforeSubmit)
	colors.PrintMint("Faucet after  ", fundsAfterSubmit)

	// the retryable should pay the receiver the supplied callvalue
	colors.PrintMint("Receive       ", receiveFunds)
	colors.PrintBlue("L2 Call Value ", retryableL2CallValue)
	if !arbmath.BigEquals(receiveFunds, retryableL2CallValue) {
		Fail(t, "Recipient didn't receive the right funds")
	}

	// the beneficiary should receive nothing
	colors.PrintMint("Beneficiary   ", beneficiaryFunds)
	if beneficiaryFunds.Sign() != 0 {
		Fail(t, "The beneficiary shouldn't have received funds")
	}

	// the fee refund address should recieve the excess gas
	colors.PrintBlue("Base Fee      ", l2BaseFee)
	colors.PrintBlue("Useful Gas    ", usefulGas)
	colors.PrintBlue("Used Wei      ", usedWei)
	colors.PrintBlue("Excess Deposit", excessDeposit)
	colors.PrintMint("Fee Refund    ", refundFunds)
	if !arbmath.BigEquals(refundFunds, arbmath.BigAdd(arbmath.BigSub(excessDeposit, usedWei), submissionFeeRefund)) {
		Fail(t, "The Fee Refund Address didn't receive the right funds")
	}

	// the retryable should be funded entirely from L1
	// any excess deposit less gas should be sent to fee refund address
	expectedGasChange := big.NewInt(0)

	if !arbmath.BigEquals(fundsBeforeSubmit, arbmath.BigAdd(fundsAfterSubmit, expectedGasChange)) {
		diff := arbmath.BigSub(fundsBeforeSubmit, fundsAfterSubmit)
		colors.PrintRed("Expected ", expectedGasChange)
		colors.PrintRed("Observed ", diff)
		colors.PrintRed("Off by   ", arbmath.BigSub(expectedGasChange, diff))
		Fail(t, "Supplied gas was improperly deducted\n", fundsBeforeSubmit, "\n", fundsAfterSubmit)
	}
}

func waitForL1DelayBlocks(t *testing.T, ctx context.Context, l1client *ethclient.Client, l1info *BlockchainTestInfo) {
	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}
}
