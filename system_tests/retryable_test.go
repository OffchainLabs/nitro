// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/retryables"
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
	*NodeBuilder,
	*bridgegen.Inbox,
	func(*types.Receipt) *types.Transaction,
	context.Context,
	func(),
) {
	ctx, cancel := context.WithCancel(context.Background())
	testNode := NewNodeBuilder(ctx).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)

	testNode.L2Info.GenerateAccount("User2")
	testNode.L2Info.GenerateAccount("Beneficiary")
	testNode.L2Info.GenerateAccount("Burn")

	delayedInbox, err := bridgegen.NewInbox(testNode.L1Info.GetAddress("Inbox"), testNode.L1Client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(testNode.L1Client, testNode.L1Info.GetAddress("Bridge"), 0)
	Require(t, err)

	lookupL2Tx := func(l1Receipt *types.Receipt) *types.Transaction {
		messages, err := delayedBridge.LookupMessagesInRange(ctx, l1Receipt.BlockNumber, l1Receipt.BlockNumber, nil)
		Require(t, err)
		if len(messages) == 0 {
			Fatal(t, "didn't find message for submission")
		}
		var submissionTxs []*types.Transaction
		msgTypes := map[uint8]bool{
			arbostypes.L1MessageType_SubmitRetryable: true,
			arbostypes.L1MessageType_EthDeposit:      true,
			arbostypes.L1MessageType_L2Message:       true,
		}
		txTypes := map[uint8]bool{
			types.ArbitrumSubmitRetryableTxType: true,
			types.ArbitrumDepositTxType:         true,
			types.ArbitrumContractTxType:        true,
		}
		for _, message := range messages {
			if !msgTypes[message.Message.Header.Kind] {
				continue
			}
			txs, err := arbos.ParseL2Transactions(message.Message, params.ArbitrumDevTestChainConfig().ChainID, nil)
			Require(t, err)
			for _, tx := range txs {
				if txTypes[tx.Type()] {
					submissionTxs = append(submissionTxs, tx)
				}
			}
		}
		if len(submissionTxs) != 1 {
			Fatal(t, "expected 1 tx from submission, found", len(submissionTxs))
		}
		return submissionTxs[0]
	}

	// burn some gas so that the faucet's Callvalue + Balance never exceeds a uint256
	discard := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	testNode.TransferBalanceViaL2(t, "Faucet", "Burn", discard)

	teardown := func() {

		// check the integrity of the RPC
		blockNum, err := testNode.L2Client.BlockNumber(ctx)
		Require(t, err, "failed to get L2 block number")
		for number := uint64(0); number < blockNum; number++ {
			block, err := testNode.L2Client.BlockByNumber(ctx, arbmath.UintToBig(number))
			Require(t, err, "failed to get L2 block", number, "of", blockNum)
			if block.Number().Uint64() != number {
				Fatal(t, "block number mismatch", number, block.Number().Uint64())
			}
		}

		cancel()

		testNode.L2Node.StopAndWait()
		requireClose(t, testNode.L1Stack)
	}
	return testNode, delayedInbox, lookupL2Tx, ctx, teardown
}

func TestRetryableNoExist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testNode := NewNodeBuilder(ctx).SetNodeConfig(arbnode.ConfigDefaultL2Test()).CreateTestNodeOnL2Only(t, true)
	defer testNode.L2Node.StopAndWait()

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), testNode.L2Client)
	Require(t, err)
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, common.Hash{})
	if err.Error() != "execution reverted: error NoTicketWithID()" {
		Fatal(t, "didn't get expected NoTicketWithID error")
	}
}

func TestSubmitRetryableImmediateSuccess(t *testing.T) {
	t.Parallel()
	testNode, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	user2Address := testNode.L2Info.GetAddress("User2")
	beneficiaryAddress := testNode.L2Info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, testNode.L2Client)
	Require(t, err, "failed to deploy NodeInterface")

	// estimate the gas needed to auto redeem the retryable
	usertxoptsL2 := testNode.L2Info.GetDefaultTransactOpts("Faucet", ctx)
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

	// submit & auto redeem the retryable using the gas estimate
	usertxoptsL1 := testNode.L1Info.GetDefaultTransactOpts("Faucet", ctx)
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

	l1Receipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, testNode.L1Client, testNode.L1Info)

	receipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, lookupL2Tx(l1Receipt))
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t)
	}

	l2balance, err := testNode.L2Client.BalanceAt(ctx, testNode.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if !arbmath.BigEquals(l2balance, big.NewInt(1e6)) {
		Fatal(t, "Unexpected balance:", l2balance)
	}
}

func TestSubmitRetryableFailThenRetry(t *testing.T) {
	t.Parallel()
	testNode, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	ownerTxOpts := testNode.L2Info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := testNode.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, simple := testNode.DeploySimple(t, ownerTxOpts)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := testNode.L2Info.GetAddress("Beneficiary")
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
		simpleABI.Methods["incrementRedeem"].ID,
	)
	Require(t, err)

	l1Receipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, testNode.L1Client, testNode.L1Info)

	receipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, lookupL2Tx(l1Receipt))
	Require(t, err)
	if len(receipt.Logs) != 2 {
		Fatal(t, len(receipt.Logs))
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// get receipt for the auto redeem, make sure it failed
	receipt, err = WaitForTx(ctx, testNode.L2Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fatal(t, receipt.GasUsed)
	}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), testNode.L2Client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(&ownerTxOpts, ticketId)
	Require(t, err)
	receipt, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)

	retryTxId := receipt.Logs[0].Topics[2]

	// check the receipt for the retry
	receipt, err = WaitForTx(ctx, testNode.L2Client, retryTxId, time.Second*1)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, receipt.Status)
	}

	// verify that the increment happened, so we know the retry succeeded
	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err)

	if counter != 1 {
		Fatal(t, "Unexpected counter:", counter)
	}

	if len(receipt.Logs) != 1 {
		Fatal(t, "Unexpected log count:", len(receipt.Logs))
	}
	parsed, err := simple.ParseRedeemedEvent(*receipt.Logs[0])
	Require(t, err)
	aliasedSender := util.RemapL1Address(usertxopts.From)
	if parsed.Caller != aliasedSender {
		Fatal(t, "Unexpected caller", parsed.Caller, "expected", aliasedSender)
	}
	if parsed.Redeemer != ownerTxOpts.From {
		Fatal(t, "Unexpected redeemer", parsed.Redeemer, "expected", ownerTxOpts.From)
	}
}

func TestSubmissionGasCosts(t *testing.T) {
	t.Parallel()
	testNode, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()
	infraFeeAddr, networkFeeAddr := setupFeeAddresses(t, ctx, testNode.L2Client, testNode.L2Info)
	elevateL2Basefee(t, ctx, testNode)

	usertxopts := testNode.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	testNode.L2Info.GenerateAccount("Refund")
	testNode.L2Info.GenerateAccount("Receive")
	faucetAddress := util.RemapL1Address(testNode.L1Info.GetAddress("Faucet"))
	beneficiaryAddress := testNode.L2Info.GetAddress("Beneficiary")
	feeRefundAddress := testNode.L2Info.GetAddress("Refund")
	receiveAddress := testNode.L2Info.GetAddress("Receive")

	colors.PrintBlue("Faucet      ", faucetAddress)
	colors.PrintBlue("Receive     ", receiveAddress)
	colors.PrintBlue("Beneficiary ", beneficiaryAddress)
	colors.PrintBlue("Fee Refund  ", feeRefundAddress)

	fundsBeforeSubmit, err := testNode.L2Client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)

	infraBalanceBefore, err := testNode.L2Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceBefore, err := testNode.L2Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	usefulGas := params.TxGas
	excessGasLimit := uint64(808)

	maxSubmissionFee := big.NewInt(1e14)
	retryableGas := arbmath.UintToBig(usefulGas + excessGasLimit) // will only burn the intrinsic cost
	retryableL2CallValue := big.NewInt(1e4)
	retryableCallData := []byte{}
	gasFeeCap := big.NewInt(l2pricing.InitialBaseFeeWei * 2)
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxopts,
		receiveAddress,
		retryableL2CallValue,
		maxSubmissionFee,
		feeRefundAddress,
		beneficiaryAddress,
		retryableGas,
		gasFeeCap,
		retryableCallData,
	)
	Require(t, err)

	l1Receipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, testNode.L1Client, testNode.L1Info)

	submissionTxOuter := lookupL2Tx(l1Receipt)
	submissionReceipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, submissionTxOuter)
	Require(t, err)
	if len(submissionReceipt.Logs) != 2 {
		Fatal(t, "Unexpected number of logs:", len(submissionReceipt.Logs))
	}
	firstRetryTxId := submissionReceipt.Logs[1].Topics[2]
	// get receipt for the auto redeem
	redeemReceipt, err := WaitForTx(ctx, testNode.L2Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if redeemReceipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "first retry tx failed")
	}
	redeemBlock, err := testNode.L2Client.HeaderByNumber(ctx, redeemReceipt.BlockNumber)
	Require(t, err)

	l2BaseFee := redeemBlock.BaseFee
	excessGasPrice := arbmath.BigSub(gasFeeCap, l2BaseFee)
	excessWei := arbmath.BigMulByUint(l2BaseFee, excessGasLimit)
	excessWei.Add(excessWei, arbmath.BigMul(excessGasPrice, retryableGas))

	fundsAfterSubmit, err := testNode.L2Client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)
	beneficiaryFunds, err := testNode.L2Client.BalanceAt(ctx, beneficiaryAddress, nil)
	Require(t, err)
	refundFunds, err := testNode.L2Client.BalanceAt(ctx, feeRefundAddress, nil)
	Require(t, err)
	receiveFunds, err := testNode.L2Client.BalanceAt(ctx, receiveAddress, nil)
	Require(t, err)

	infraBalanceAfter, err := testNode.L2Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfter, err := testNode.L2Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	colors.PrintBlue("CallGas    ", retryableGas)
	colors.PrintMint("Gas cost   ", arbmath.BigMul(retryableGas, l2BaseFee))
	colors.PrintBlue("Payment    ", usertxopts.Value)

	colors.PrintMint("Faucet before ", fundsAfterSubmit)
	colors.PrintMint("Faucet after  ", fundsAfterSubmit)

	// the retryable should pay the receiver the supplied callvalue
	colors.PrintMint("Receive       ", receiveFunds)
	colors.PrintBlue("L2 Call Value ", retryableL2CallValue)
	if !arbmath.BigEquals(receiveFunds, retryableL2CallValue) {
		Fatal(t, "Recipient didn't receive the right funds")
	}

	// the beneficiary should receive nothing
	colors.PrintMint("Beneficiary   ", beneficiaryFunds)
	if beneficiaryFunds.Sign() != 0 {
		Fatal(t, "The beneficiary shouldn't have received funds")
	}

	// the fee refund address should recieve the excess gas
	colors.PrintBlue("Base Fee         ", l2BaseFee)
	colors.PrintBlue("Excess Gas Price ", excessGasPrice)
	colors.PrintBlue("Excess Gas       ", excessGasLimit)
	colors.PrintBlue("Excess Wei       ", excessWei)
	colors.PrintMint("Fee Refund       ", refundFunds)
	if !arbmath.BigEquals(refundFunds, arbmath.BigAdd(excessWei, maxSubmissionFee)) {
		Fatal(t, "The Fee Refund Address didn't receive the right funds")
	}

	// the faucet must pay for both the gas used and the call value supplied
	expectedGasChange := arbmath.BigMul(gasFeeCap, retryableGas)
	expectedGasChange = arbmath.BigSub(expectedGasChange, usertxopts.Value) // the user is credited this
	expectedGasChange = arbmath.BigAdd(expectedGasChange, maxSubmissionFee)
	expectedGasChange = arbmath.BigAdd(expectedGasChange, retryableL2CallValue)

	if !arbmath.BigEquals(fundsBeforeSubmit, arbmath.BigAdd(fundsAfterSubmit, expectedGasChange)) {
		diff := arbmath.BigSub(fundsBeforeSubmit, fundsAfterSubmit)
		colors.PrintRed("Expected ", expectedGasChange)
		colors.PrintRed("Observed ", diff)
		colors.PrintRed("Off by   ", arbmath.BigSub(expectedGasChange, diff))
		Fatal(t, "Supplied gas was improperly deducted\n", fundsBeforeSubmit, "\n", fundsAfterSubmit)
	}

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), testNode.L2Client)
	Require(t, err)
	minimumBaseFee, err := arbGasInfo.GetMinimumGasPrice(&bind.CallOpts{Context: ctx})
	Require(t, err)

	expectedFee := arbmath.BigMulByUint(l2BaseFee, usefulGas)
	expectedInfraFee := arbmath.BigMulByUint(minimumBaseFee, usefulGas)
	expectedNetworkFee := arbmath.BigSub(expectedFee, expectedInfraFee)

	infraFee := arbmath.BigSub(infraBalanceAfter, infraBalanceBefore)
	networkFee := arbmath.BigSub(networkBalanceAfter, networkBalanceBefore)
	fee := arbmath.BigAdd(infraFee, networkFee)

	colors.PrintMint("paid infra fee:      ", infraFee)
	colors.PrintMint("paid network fee:    ", networkFee)
	colors.PrintMint("paid fee:            ", fee)

	if !arbmath.BigEquals(infraFee, expectedInfraFee) {
		Fatal(t, "Unexpected infra fee paid, want:", expectedInfraFee, "have:", infraFee)
	}
	if !arbmath.BigEquals(networkFee, expectedNetworkFee) {
		Fatal(t, "Unexpected network fee paid, want:", expectedNetworkFee, "have:", networkFee)
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

func TestDepositETH(t *testing.T) {
	t.Parallel()
	testNode, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	faucetAddr := testNode.L1Info.GetAddress("Faucet")

	oldBalance, err := testNode.L2Client.BalanceAt(ctx, faucetAddr, nil)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", faucetAddr, err)
	}

	txOpts := testNode.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	txOpts.Value = big.NewInt(13)

	l1tx, err := delayedInbox.DepositEth0(&txOpts)
	if err != nil {
		t.Fatalf("DepositEth0() unexected error: %v", err)
	}

	l1Receipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded() unexpected error: %v", err)
	}
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("Got transaction status: %v, want: %v", l1Receipt.Status, types.ReceiptStatusSuccessful)
	}
	waitForL1DelayBlocks(t, ctx, testNode.L1Client, testNode.L1Info)

	l2Receipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, lookupL2Tx(l1Receipt))
	if err != nil {
		t.Fatalf("EnsureTxSucceeded unexpected error: %v", err)
	}
	newBalance, err := testNode.L2Client.BalanceAt(ctx, faucetAddr, l2Receipt.BlockNumber)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", faucetAddr, err)
	}
	if got := new(big.Int); got.Sub(newBalance, oldBalance).Cmp(txOpts.Value) != 0 {
		t.Errorf("Got transferred: %v, want: %v", got, txOpts.Value)
	}
}

func TestArbitrumContractTx(t *testing.T) {
	testNode, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()
	faucetL2Addr := util.RemapL1Address(testNode.L1Info.GetAddress("Faucet"))
	testNode.TransferBalanceToViaL2(t, "Faucet", faucetL2Addr, big.NewInt(1e18))

	l2TxOpts := testNode.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	l2ContractAddr, _ := testNode.DeploySimple(t, l2TxOpts)
	l2ContractABI, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	if err != nil {
		t.Fatalf("Error parsing contract ABI: %v", err)
	}
	data, err := l2ContractABI.Pack("checkCalls", true, true, false, false, false, false)
	if err != nil {
		t.Fatalf("Error packing method's call data: %v", err)
	}
	unsignedTx := types.NewTx(&types.ArbitrumContractTx{
		ChainId:   testNode.L2Info.Signer.ChainID(),
		From:      faucetL2Addr,
		GasFeeCap: testNode.L2Info.GasPrice.Mul(testNode.L2Info.GasPrice, big.NewInt(2)),
		Gas:       1e6,
		To:        &l2ContractAddr,
		Value:     common.Big0,
		Data:      data,
	})
	txOpts := testNode.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1tx, err := delayedInbox.SendContractTransaction(
		&txOpts,
		arbmath.UintToBig(unsignedTx.Gas()),
		unsignedTx.GasFeeCap(),
		*unsignedTx.To(),
		unsignedTx.Value(),
		unsignedTx.Data(),
	)
	if err != nil {
		t.Fatalf("Error sending unsigned transaction: %v", err)
	}
	receipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", l1tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L1 transaction: %v has failed", l1tx.Hash())
	}
	waitForL1DelayBlocks(t, ctx, testNode.L1Client, testNode.L1Info)
	receipt, err = EnsureTxSucceeded(ctx, testNode.L2Client, lookupL2Tx(receipt))
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", unsignedTx.Hash(), err)
	}
}

func TestL1FundedUnsignedTransaction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	testNode := NewNodeBuilder(ctx).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNode.L1Stack)
	defer testNode.L2Node.StopAndWait()

	faucetL2Addr := util.RemapL1Address(testNode.L1Info.GetAddress("Faucet"))
	// Transfer balance to Faucet's corresponding L2 address, so that there is
	// enough balance on its' account for executing L2 transaction.
	testNode.TransferBalanceToViaL2(t, "Faucet", faucetL2Addr, big.NewInt(1e18))

	l2TxOpts := testNode.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	contractAddr, _ := testNode.DeploySimple(t, l2TxOpts)
	contractABI, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	if err != nil {
		t.Fatalf("Error parsing contract ABI: %v", err)
	}
	data, err := contractABI.Pack("checkCalls", true, true, false, false, false, false)
	if err != nil {
		t.Fatalf("Error packing method's call data: %v", err)
	}
	nonce, err := testNode.L2Client.NonceAt(ctx, faucetL2Addr, nil)
	if err != nil {
		t.Fatalf("Error getting nonce at address: %v, error: %v", faucetL2Addr, err)
	}
	unsignedTx := types.NewTx(&types.ArbitrumUnsignedTx{
		ChainId:   testNode.L2Info.Signer.ChainID(),
		From:      faucetL2Addr,
		Nonce:     nonce,
		GasFeeCap: testNode.L2Info.GasPrice,
		Gas:       1e6,
		To:        &contractAddr,
		Value:     common.Big0,
		Data:      data,
	})

	delayedInbox, err := bridgegen.NewInbox(testNode.L1Info.GetAddress("Inbox"), testNode.L1Client)
	if err != nil {
		t.Fatalf("Error getting Go binding of L1 Inbox contract: %v", err)
	}

	txOpts := testNode.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	l1tx, err := delayedInbox.SendUnsignedTransaction(
		&txOpts,
		arbmath.UintToBig(unsignedTx.Gas()),
		unsignedTx.GasFeeCap(),
		arbmath.UintToBig(unsignedTx.Nonce()),
		*unsignedTx.To(),
		unsignedTx.Value(),
		unsignedTx.Data(),
	)
	if err != nil {
		t.Fatalf("Error sending unsigned transaction: %v", err)
	}
	receipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", l1tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L1 transaction: %v has failed", l1tx.Hash())
	}
	waitForL1DelayBlocks(t, ctx, testNode.L1Client, testNode.L1Info)
	receipt, err = EnsureTxSucceeded(ctx, testNode.L2Client, unsignedTx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", unsignedTx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L2 transaction: %v has failed", receipt.TxHash)
	}
}

func TestRetryableSubmissionAndRedeemFees(t *testing.T) {
	testNode, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()
	infraFeeAddr, networkFeeAddr := setupFeeAddresses(t, ctx, testNode.L2Client, testNode.L2Info)

	ownerTxOpts := testNode.L2Info.GetDefaultTransactOpts("Owner", ctx)
	simpleAddr, simple := testNode.DeploySimple(t, ownerTxOpts)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)

	elevateL2Basefee(t, ctx, testNode)

	infraBalanceBefore, err := testNode.L2Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceBefore, err := testNode.L2Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	beneficiaryAddress := testNode.L2Info.GetAddress("Beneficiary")
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := common.Big0
	usertxoptsL1 := testNode.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit
	baseFee := testNode.GetBaseFeeAtViaL2(t, nil)
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxoptsL1,
		simpleAddr,
		callValue,
		big.NewInt(1e16),
		beneficiaryAddress,
		beneficiaryAddress,
		// send enough L2 gas for intrinsic but not compute
		big.NewInt(int64(params.TxGas+params.TxDataNonZeroGasEIP2028*4)),
		big.NewInt(baseFee.Int64()*2),
		simpleABI.Methods["incrementRedeem"].ID,
	)
	Require(t, err)
	l1Receipt, err := EnsureTxSucceeded(ctx, testNode.L1Client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, testNode.L1Client, testNode.L1Info)

	submissionTxOuter := lookupL2Tx(l1Receipt)
	submissionReceipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, submissionTxOuter)
	Require(t, err)
	if len(submissionReceipt.Logs) != 2 {
		Fatal(t, len(submissionReceipt.Logs))
	}
	ticketId := submissionReceipt.Logs[0].Topics[1]
	firstRetryTxId := submissionReceipt.Logs[1].Topics[2]
	// get receipt for the auto redeem, make sure it failed
	autoRedeemReceipt, err := WaitForTx(ctx, testNode.L2Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if autoRedeemReceipt.Status != types.ReceiptStatusFailed {
		Fatal(t, "first retry tx shouldn't have succeeded")
	}

	infraBalanceAfterSubmission, err := testNode.L2Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfterSubmission, err := testNode.L2Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	usertxoptsL2 := testNode.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), testNode.L2Client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(&usertxoptsL2, ticketId)
	Require(t, err)
	redeemReceipt, err := EnsureTxSucceeded(ctx, testNode.L2Client, tx)
	Require(t, err)
	retryTxId := redeemReceipt.Logs[0].Topics[2]

	// check the receipt for the retry
	retryReceipt, err := WaitForTx(ctx, testNode.L2Client, retryTxId, time.Second*1)
	Require(t, err)
	if retryReceipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "retry failed")
	}

	infraBalanceAfterRedeem, err := testNode.L2Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfterRedeem, err := testNode.L2Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	// verify that the increment happened, so we know the retry succeeded
	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err)

	if counter != 1 {
		Fatal(t, "Unexpected counter:", counter)
	}

	if len(retryReceipt.Logs) != 1 {
		Fatal(t, "Unexpected log count:", len(retryReceipt.Logs))
	}
	parsed, err := simple.ParseRedeemedEvent(*retryReceipt.Logs[0])
	Require(t, err)
	aliasedSender := util.RemapL1Address(usertxoptsL1.From)
	if parsed.Caller != aliasedSender {
		Fatal(t, "Unexpected caller", parsed.Caller, "expected", aliasedSender)
	}
	if parsed.Redeemer != usertxoptsL2.From {
		Fatal(t, "Unexpected redeemer", parsed.Redeemer, "expected", usertxoptsL2.From)
	}

	infraSubmissionFee := arbmath.BigSub(infraBalanceAfterSubmission, infraBalanceBefore)
	networkSubmissionFee := arbmath.BigSub(networkBalanceAfterSubmission, networkBalanceBefore)
	infraRedeemFee := arbmath.BigSub(infraBalanceAfterRedeem, infraBalanceAfterSubmission)
	networkRedeemFee := arbmath.BigSub(networkBalanceAfterRedeem, networkBalanceAfterSubmission)

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), testNode.L2Client)
	Require(t, err)
	minimumBaseFee, err := arbGasInfo.GetMinimumGasPrice(&bind.CallOpts{Context: ctx})
	Require(t, err)
	submissionBaseFee := testNode.GetBaseFeeAtViaL2(t, submissionReceipt.BlockNumber)
	submissionTx, ok := submissionTxOuter.GetInner().(*types.ArbitrumSubmitRetryableTx)
	if !ok {
		Fatal(t, "inner tx isn't ArbitrumSubmitRetryableTx")
	}
	// submission + auto redeemed retry expected fees
	retryableSubmissionFee := retryables.RetryableSubmissionFee(len(submissionTx.RetryData), submissionTx.L1BaseFee)
	expectedSubmissionFee := arbmath.BigMulByUint(submissionBaseFee, autoRedeemReceipt.GasUsed)
	expectedInfraSubmissionFee := arbmath.BigMulByUint(minimumBaseFee, autoRedeemReceipt.GasUsed)
	expectedNetworkSubmissionFee := arbmath.BigAdd(
		arbmath.BigSub(expectedSubmissionFee, expectedInfraSubmissionFee),
		retryableSubmissionFee,
	)

	retryTxOuter, _, err := testNode.L2Client.TransactionByHash(ctx, retryTxId)
	Require(t, err)
	retryTx, ok := retryTxOuter.GetInner().(*types.ArbitrumRetryTx)
	if !ok {
		Fatal(t, "inner tx isn't ArbitrumRetryTx")
	}
	redeemBaseFee := testNode.GetBaseFeeAtViaL2(t, redeemReceipt.BlockNumber)

	t.Log("redeem base fee:", redeemBaseFee)
	// redeem & retry expected fees
	redeemGasUsed := redeemReceipt.GasUsed - redeemReceipt.GasUsedForL1 - retryTx.Gas + retryReceipt.GasUsed
	expectedRedeemFee := arbmath.BigMulByUint(redeemBaseFee, redeemGasUsed)
	expectedInfraRedeemFee := arbmath.BigMulByUint(minimumBaseFee, redeemGasUsed)
	expectedNetworkRedeemFee := arbmath.BigSub(expectedRedeemFee, expectedInfraRedeemFee)

	t.Log("submission gas:         ", submissionReceipt.GasUsed)
	t.Log("auto redeemed retry gas:", autoRedeemReceipt.GasUsed)
	t.Log("redeem gas:             ", redeemReceipt.GasUsed)
	t.Log("retry gas:              ", retryReceipt.GasUsed)
	colors.PrintMint("submission and auto redeemed retry - paid infra fee:        ", infraSubmissionFee)
	colors.PrintBlue("submission and auto redeemed retry - expected infra fee:    ", expectedInfraSubmissionFee)
	colors.PrintMint("submission and auto redeemed retry - paid network fee:      ", networkSubmissionFee)
	colors.PrintBlue("submission and auto redeemed retry - expected network fee:  ", expectedNetworkSubmissionFee)
	colors.PrintMint("redeem and retry - paid infra fee:            ", infraRedeemFee)
	colors.PrintBlue("redeem and retry - expected infra fee:        ", expectedInfraRedeemFee)
	colors.PrintMint("redeem and retry - paid network fee:          ", networkRedeemFee)
	colors.PrintBlue("redeem and retry - expected network fee:      ", expectedNetworkRedeemFee)
	if !arbmath.BigEquals(infraSubmissionFee, expectedInfraSubmissionFee) {
		Fatal(t, "Unexpected infra fee paid by submission and auto redeem, want:", expectedInfraSubmissionFee, "have:", infraSubmissionFee)
	}
	if !arbmath.BigEquals(networkSubmissionFee, expectedNetworkSubmissionFee) {
		Fatal(t, "Unexpected network fee paid by submission and auto redeem, want:", expectedNetworkSubmissionFee, "have:", networkSubmissionFee)
	}
	if !arbmath.BigEquals(infraRedeemFee, expectedInfraRedeemFee) {
		Fatal(t, "Unexpected infra fee paid by redeem and retry, want:", expectedInfraRedeemFee, "have:", infraRedeemFee)
	}
	if !arbmath.BigEquals(networkRedeemFee, expectedNetworkRedeemFee) {
		Fatal(t, "Unexpected network fee paid by redeem and retry, want:", expectedNetworkRedeemFee, "have:", networkRedeemFee)
	}
}

// elevateL2Basefee by burning gas exceeding speed limit
func elevateL2Basefee(t *testing.T, ctx context.Context, testNode *NodeBuilder) {
	baseFeeBefore := testNode.GetBaseFeeAtViaL2(t, nil)
	colors.PrintBlue("Elevating base fee...")
	arbostestabi, err := precompilesgen.ArbosTestMetaData.GetAbi()
	Require(t, err)
	_, err = precompilesgen.NewArbosTest(common.HexToAddress("0x69"), testNode.L2Client)
	Require(t, err, "failed to deploy ArbosTest")

	burnAmount := arbnode.ConfigDefaultL1Test().RPC.RPCGasCap
	burnTarget := uint64(5 * l2pricing.InitialSpeedLimitPerSecondV6 * l2pricing.InitialBacklogTolerance)
	for i := uint64(0); i < (burnTarget+burnAmount)/burnAmount; i++ {
		burnArbGas := arbostestabi.Methods["burnArbGas"]
		data, err := burnArbGas.Inputs.Pack(arbmath.UintToBig(burnAmount - testNode.L2Info.TransferGas))
		Require(t, err)
		input := append([]byte{}, burnArbGas.ID...)
		input = append(input, data...)
		to := common.HexToAddress("0x69")
		tx := testNode.L2Info.PrepareTxTo("Faucet", &to, burnAmount, big.NewInt(0), input)
		Require(t, testNode.L2Client.SendTransaction(ctx, tx))
		_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
		Require(t, err)
	}
	baseFee := testNode.GetBaseFeeAtViaL2(t, nil)
	colors.PrintBlue("New base fee: ", baseFee, " diff:", baseFee.Uint64()-baseFeeBefore.Uint64())
}

func setupFeeAddresses(t *testing.T, ctx context.Context, l2client *ethclient.Client, l2info *BlockchainTestInfo) (common.Address, common.Address) {
	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	ownerCallOpts := l2info.GetDefaultCallOpts("Owner", ctx)
	// make "Owner" a chain owner
	arbdebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), l2client)
	Require(t, err, "failed to deploy ArbDebug")
	tx, err := arbdebug.BecomeChainOwner(&ownerTxOpts)
	Require(t, err, "failed to deploy ArbDebug")
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	arbowner, err := precompilesgen.NewArbOwner(common.HexToAddress("70"), l2client)
	Require(t, err)
	arbownerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("6b"), l2client)
	Require(t, err)
	l2info.GenerateAccount("InfraFee")
	l2info.GenerateAccount("NetworkFee")
	networkFeeAddr := l2info.GetAddress("NetworkFee")
	infraFeeAddr := l2info.GetAddress("InfraFee")
	tx, err = arbowner.SetNetworkFeeAccount(&ownerTxOpts, networkFeeAddr)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	networkFeeAccount, err := arbownerPublic.GetNetworkFeeAccount(ownerCallOpts)
	Require(t, err)
	tx, err = arbowner.SetInfraFeeAccount(&ownerTxOpts, infraFeeAddr)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	infraFeeAccount, err := arbownerPublic.GetInfraFeeAccount(ownerCallOpts)
	Require(t, err)
	t.Log("Infra fee account: ", infraFeeAccount)
	t.Log("Network fee account: ", networkFeeAccount)
	return infraFeeAddr, networkFeeAddr
}
