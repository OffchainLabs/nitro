// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/merkletree"
)

func retryableSetup(t *testing.T, timestampWorkaround bool) (
	*arbnode.Node,
	info,
	info,
	*ethclient.Client,
	*ethclient.Client,
	*bridgegen.Inbox,
	func(*types.Receipt) *types.Transaction,
	context.Context,
	func(),
) {
	ctx, cancel := context.WithCancel(context.Background())
	nodeConfig := arbnode.ConfigDefaultL1Test()
	if timestampWorkaround {
		nodeConfig.BatchPoster.MaxDelay = 3 * retryables.RetryableLifetimeSeconds * time.Second
	}
	l2info, l2node, l2client, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nodeConfig, nil, nil)
	l2info.GenerateAccount("User2")
	l2info.GenerateAccount("Beneficiary")
	l2info.GenerateAccount("Burn")
	delayedInbox, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(l1client, l1info.GetAddress("Bridge"), 0)
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
	TransferBalance(t, "Faucet", "Burn", discard, l2info, l2client, ctx)

	teardown := func() {

		// check the integrity of the RPC
		blockNum, err := l2client.BlockNumber(ctx)
		Require(t, err, "failed to get L2 block number")
		for number := uint64(0); number < blockNum; number++ {
			block, err := l2client.BlockByNumber(ctx, arbmath.UintToBig(number))
			Require(t, err, "failed to get L2 block", number, "of", blockNum)
			if block.Number().Uint64() != number {
				Fatal(t, "block number mismatch", number, block.Number().Uint64())
			}
		}

		cancel()

		l2node.StopAndWait()
		requireClose(t, l1stack)
	}
	return l2node, l2info, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown
}

func TestRetryableNoExist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, node, l2client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, common.Hash{})
	if err.Error() != "execution reverted: error NoTicketWithID()" {
		Fatal(t, "didn't get expected NoTicketWithID error")
	}
}

func TestSubmitRetryableImmediateSuccess(t *testing.T) {
	t.Parallel()
	_, l2info, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, false)
	defer teardown()

	user2Address := l2info.GetAddress("User2")
	beneficiaryAddress := l2info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, l2client)
	Require(t, err, "failed to deploy NodeInterface")

	// estimate the gas needed to auto redeem the retryable
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

	// submit & auto redeem the retryable using the gas estimate
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

	l1Receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := EnsureTxSucceeded(ctx, l2client, lookupL2Tx(l1Receipt))
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t)
	}

	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if !arbmath.BigEquals(l2balance, big.NewInt(1e6)) {
		Fatal(t, "Unexpected balance:", l2balance)
	}
}

func TestSubmitRetryableFailThenRetry(t *testing.T) {
	t.Parallel()
	_, l2info, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, false)
	defer teardown()

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, simple := deploySimple(t, ctx, ownerTxOpts, l2client)
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
		simpleABI.Methods["incrementRedeem"].ID,
	)
	Require(t, err)

	l1Receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := EnsureTxSucceeded(ctx, l2client, lookupL2Tx(l1Receipt))
	Require(t, err)
	if len(receipt.Logs) != 2 {
		Fatal(t, len(receipt.Logs))
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// get receipt for the auto redeem, make sure it failed
	receipt, err = WaitForTx(ctx, l2client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fatal(t, receipt.GasUsed)
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
	_, l2info, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, false)
	defer teardown()
	infraFeeAddr, networkFeeAddr := setupFeeAddresses(t, ctx, l2client, l2info)
	elevateL2Basefee(t, ctx, l2client, l2info)

	usertxopts := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	l2info.GenerateAccount("Refund")
	l2info.GenerateAccount("Receive")
	faucetAddress := util.RemapL1Address(l1info.GetAddress("Faucet"))
	beneficiaryAddress := l2info.GetAddress("Beneficiary")
	feeRefundAddress := l2info.GetAddress("Refund")
	receiveAddress := l2info.GetAddress("Receive")

	colors.PrintBlue("Faucet      ", faucetAddress)
	colors.PrintBlue("Receive     ", receiveAddress)
	colors.PrintBlue("Beneficiary ", beneficiaryAddress)
	colors.PrintBlue("Fee Refund  ", feeRefundAddress)

	fundsBeforeSubmit, err := l2client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)

	infraBalanceBefore, err := l2client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceBefore, err := l2client.BalanceAt(ctx, networkFeeAddr, nil)
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

	l1Receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	submissionTxOuter := lookupL2Tx(l1Receipt)
	submissionReceipt, err := EnsureTxSucceeded(ctx, l2client, submissionTxOuter)
	Require(t, err)
	if len(submissionReceipt.Logs) != 2 {
		Fatal(t, "Unexpected number of logs:", len(submissionReceipt.Logs))
	}
	firstRetryTxId := submissionReceipt.Logs[1].Topics[2]
	// get receipt for the auto redeem
	redeemReceipt, err := WaitForTx(ctx, l2client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if redeemReceipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "first retry tx failed")
	}
	redeemBlock, err := l2client.HeaderByNumber(ctx, redeemReceipt.BlockNumber)
	Require(t, err)

	l2BaseFee := redeemBlock.BaseFee
	excessGasPrice := arbmath.BigSub(gasFeeCap, l2BaseFee)
	excessWei := arbmath.BigMulByUint(l2BaseFee, excessGasLimit)
	excessWei.Add(excessWei, arbmath.BigMul(excessGasPrice, retryableGas))

	fundsAfterSubmit, err := l2client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)
	beneficiaryFunds, err := l2client.BalanceAt(ctx, beneficiaryAddress, nil)
	Require(t, err)
	refundFunds, err := l2client.BalanceAt(ctx, feeRefundAddress, nil)
	Require(t, err)
	receiveFunds, err := l2client.BalanceAt(ctx, receiveAddress, nil)
	Require(t, err)

	infraBalanceAfter, err := l2client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfter, err := l2client.BalanceAt(ctx, networkFeeAddr, nil)
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

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), l2client)
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

func TestRetryableExpiryAndRevival(t *testing.T) {
	t.Parallel()
	testRetryableExpiryAndRevival(t, 1, 0)
	testRetryableExpiryAndRevival(t, 3, 0)
	testRetryableExpiryAndRevival(t, 4, 0)
	testRetryableExpiryAndRevival(t, 8, 0)
	testRetryableExpiryAndRevival(t, 9, 0)

	testRetryableExpiryAndRevival(t, 1, 1)
	testRetryableExpiryAndRevival(t, 3, 1)
	testRetryableExpiryAndRevival(t, 4, 1)
	testRetryableExpiryAndRevival(t, 8, 1)
	testRetryableExpiryAndRevival(t, 9, 1)

	testRetryableExpiryAndRevival(t, 1, 2)
	testRetryableExpiryAndRevival(t, 3, 2)
	testRetryableExpiryAndRevival(t, 4, 2)
	testRetryableExpiryAndRevival(t, 8, 2)
	testRetryableExpiryAndRevival(t, 9, 2)
}

func testRetryableExpiryAndRevival(t *testing.T, numRetryables int, whichRootSnapshot int) {
	l2node, l2info, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, true)
	defer teardown()
	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	simpleAddr, simple := deploySimple(t, ctx, ownerTxOpts, l2client)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)
	submissionCtx := &submissionContext{
		delayedInbox: delayedInbox,
		usertxopts:   &usertxopts,
		l1info:       l1info,
		l1client:     l1client,
		l2client:     l2client,
		lookupL2Tx:   lookupL2Tx,
	}
	var currentL1time uint64 // 0 means unitialized, will be fetched from latests l1 header
	var expectedSimpleCounter uint64

	var retriesData []*retryables.TestRetryableData
	for i := 0; i < numRetryables; i++ {
		retry := &retryables.TestRetryableData{
			Id:          common.Hash{}, // filled later in submitRetryable
			NumTries:    0,             // filled later in expiredRetryableFromLogs just before revival
			From:        util.RemapL1Address(usertxopts.From),
			To:          simpleAddr,
			CallValue:   common.Big0,
			Beneficiary: l2info.GetAddress("Beneficiary"),
			CallData:    simpleABI.Methods["incrementRedeem"].ID,
		}
		retriesData = append(retriesData, retry)
	}
	// submit retryables
	for _, retry := range retriesData {
		autoRedeemReceipt := submitRetryable(t, ctx, submissionCtx, retry)
		if autoRedeemReceipt.Status != types.ReceiptStatusFailed {
			Fatal(t, autoRedeemReceipt.GasUsed)
		}
	}
	// trigger expiry, 2 retryables at a time
	rootRotations := 0
	var rootHashes []common.Hash
	var treeSizes []uint64
	for i := 0; i < (len(retriesData)+1)/2; i++ {
		currentL1time = warpL1Time(t, ctx, l2node, l1client, l2info, currentL1time, retryables.RetryableLifetimeSeconds+1)

		rootHash, treeSize := lastExpiredRootSnapshot(t, l2client)
		rootHashes = append(rootHashes, rootHash)
		treeSizes = append(treeSizes, treeSize)
	}
	// make sure to make enough root rotations
	for i := len(rootHashes); i < whichRootSnapshot; i++ {
		currentL1time = warpL1Time(t, ctx, l2node, l1client, l2info, currentL1time, retryables.ExpiredSnapshotsRotationIntervalSeconds+1)
		rootHash, treeSize := lastExpiredRootSnapshot(t, l2client)
		rootHashes = append(rootHashes, rootHash)
		treeSizes = append(treeSizes, treeSize)
	}
	treeSize := accumulatorSizeFromLogs(t, ctx, l2client)
	rootHashes = append(rootHashes, common.Hash{})
	treeSizes = append(treeSizes, treeSize)

	// check if expiry happened
	for _, retry := range retriesData {
		redeemRetryableShouldFailNoTicket(t, ctx, l2client, &ownerTxOpts, retry)
	}
	// wait one block and re-check expiry
	waitForNextL2Block(t, ctx, l2client, l2info)
	for _, retry := range retriesData {
		redeemRetryableShouldFailNoTicket(t, ctx, l2client, &ownerTxOpts, retry)
	}
	t.Logf("rootHashes: %+v, treeSizes: %+v", rootHashes, treeSizes)
	// revive retryables
	for _, retry := range retriesData {
		reviveRetryableShouldSucceed(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, whichRootSnapshot, retry)
	}
	// repeated revival shouldn't be possible against any root
	for rootSnapshot := 0; rootSnapshot <= whichRootSnapshot; rootSnapshot++ {
		for _, retry := range retriesData {
			reviveRetryableShouldFail(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, rootSnapshot, retry)
		}
	}
	// check that they can be redeemed
	for _, retry := range retriesData {
		expectedSimpleCounter++
		redeemRetryableShouldSucceed(t, ctx, l2client, &ownerTxOpts, retry, simple, expectedSimpleCounter)
	}
	// repeated redeem shouldn't be possible
	for _, retry := range retriesData {
		redeemRetryableShouldFailNoTicket(t, ctx, l2client, &ownerTxOpts, retry)
	}
	// repeated revival shouldn't be possible against any root
	for rootSnapshot := 0; rootSnapshot <= whichRootSnapshot; rootSnapshot++ {
		for _, retry := range retriesData {
			reviveRetryableShouldFail(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, rootSnapshot, retry)
		}
	}
	// submit again the retryables
	for _, retry := range retriesData {
		autoRedeemReceipt := submitRetryable(t, ctx, submissionCtx, retry)
		if autoRedeemReceipt.Status != types.ReceiptStatusFailed {
			Fatal(t, autoRedeemReceipt.GasUsed)
		}
	}
	// trigger expiry, 2 retryables at a time
	rootHashes = []common.Hash{}
	treeSizes = []uint64{}
	for i := 0; i < (len(retriesData)+1)/2; i++ {
		currentL1time = warpL1Time(t, ctx, l2node, l1client, l2info, currentL1time, retryables.RetryableLifetimeSeconds+1)
		rootHash, treeSize := lastExpiredRootSnapshot(t, l2client)
		rootHashes = append(rootHashes, rootHash)
		treeSizes = append(treeSizes, treeSize)
	}
	// make sure to make enough root rotations
	for i := len(rootHashes); i < whichRootSnapshot; i++ {
		currentL1time = warpL1Time(t, ctx, l2node, l1client, l2info, currentL1time, retryables.ExpiredSnapshotsRotationIntervalSeconds+1)
		rootHash, treeSize := lastExpiredRootSnapshot(t, l2client)
		rootHashes = append(rootHashes, rootHash)
		treeSizes = append(treeSizes, treeSize)
	}
	treeSize = accumulatorSizeFromLogs(t, ctx, l2client)
	rootHashes = append(rootHashes, common.Hash{})
	treeSizes = append(treeSizes, treeSize)
	// check if expiry happened
	for _, retry := range retriesData {
		redeemRetryableShouldFailNoTicket(t, ctx, l2client, &ownerTxOpts, retry)
	}
	// revive retryables
	for _, retry := range retriesData {
		reviveRetryableShouldSucceed(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, whichRootSnapshot, retry)
	}
	// repeated revival shouldn't be possible against any root
	for rootSnapshot := 0; rootSnapshot <= whichRootSnapshot; rootSnapshot++ {
		for _, retry := range retriesData {
			reviveRetryableShouldFail(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, whichRootSnapshot, retry)
		}
	}
	// trigger 2nd expiry, 2 retryables at a time
	rootRotations = 0
	for i := 0; i < (len(retriesData)+1)/2; i++ {
		currentL1time = warpL1Time(t, ctx, l2node, l1client, l2info, currentL1time, retryables.RetryableLifetimeSeconds+1)
		rootRotations++
	}
	// make sure to make enough root rotations
	for i := rootRotations; i <= whichRootSnapshot; i++ {
		currentL1time = warpL1Time(t, ctx, l2node, l1client, l2info, currentL1time, retryables.ExpiredSnapshotsRotationIntervalSeconds+1)
		rootHash, treeSize := lastExpiredRootSnapshot(t, l2client)
		rootHashes = append(rootHashes, rootHash)
		treeSizes = append(treeSizes, treeSize)
	}
	treeSize = accumulatorSizeFromLogs(t, ctx, l2client)
	rootHashes = append(rootHashes, common.Hash{})
	treeSizes = append(treeSizes, treeSize)
	// check if 2nd expiry happened
	for _, retry := range retriesData {
		redeemRetryableShouldFailNoTicket(t, ctx, l2client, &ownerTxOpts, retry)
	}
	// revive retryables for the 2nd time
	for _, retry := range retriesData {
		reviveRetryableShouldSucceed(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, whichRootSnapshot, retry)
	}
	// repeated revival shouldn't be possible against any root
	for rootSnapshot := 0; rootSnapshot <= whichRootSnapshot; rootSnapshot++ {
		for _, retry := range retriesData {
			reviveRetryableShouldFail(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, rootSnapshot, retry)
		}
	}
	// check that they can be redeemed
	for _, retry := range retriesData {
		expectedSimpleCounter++
		redeemRetryableShouldSucceed(t, ctx, l2client, &ownerTxOpts, retry, simple, expectedSimpleCounter)
	}
	// repeated redeem shouldn't be possible
	for _, retry := range retriesData {
		redeemRetryableShouldFailNoTicket(t, ctx, l2client, &ownerTxOpts, retry)
	}
	// repeated revival shouldn't be possible against any root
	for rootSnapshot := 0; rootSnapshot <= whichRootSnapshot; rootSnapshot++ {
		for _, retry := range retriesData {
			reviveRetryableShouldFail(t, ctx, l2client, &ownerTxOpts, rootHashes, treeSizes, rootSnapshot, retry)
		}
	}
}

type submissionContext struct {
	delayedInbox *bridgegen.Inbox
	usertxopts   *bind.TransactOpts
	l1info       info
	l1client     *ethclient.Client
	l2client     *ethclient.Client
	lookupL2Tx   func(*types.Receipt) *types.Transaction
}

// sets retry.Id
// returns first retry tx hash
func submitRetryable(t *testing.T, ctx context.Context, subCtx *submissionContext, retry *retryables.TestRetryableData) *types.Receipt {
	t.Helper()
	l1tx, err := subCtx.delayedInbox.CreateRetryableTicket(
		subCtx.usertxopts,
		retry.To,
		retry.CallValue,
		big.NewInt(1e16),
		retry.Beneficiary,
		retry.Beneficiary,
		// send enough L2 gas for intrinsic but not compute
		big.NewInt(int64(params.TxGas+params.TxDataNonZeroGasEIP2028*4)),
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		retry.CallData,
	)
	Require(t, err, "CreateRetryableTicket failed")
	l1receipt, err := EnsureTxSucceeded(ctx, subCtx.l1client, l1tx)
	Require(t, err, "EnsureTxSucceeded failed")
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1receipt indicated failure")
	}
	waitForL1DelayBlocks(t, ctx, subCtx.l1client, subCtx.l1info)
	receipt, err := EnsureTxSucceeded(ctx, subCtx.l2client, subCtx.lookupL2Tx(l1receipt))
	Require(t, err, "EnsureTxSucceeded failed, waited for retryable submission tx")
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "unexpected receipt status, expected success")
	}
	if len(receipt.Logs) != 2 {
		Fatal(t, "unexpected length of submit retryable tx receipt logs, want: 2, have:", len(receipt.Logs))
	}
	retry.Id = receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]
	autoRedeemReceipt, err := WaitForTx(ctx, subCtx.l2client, firstRetryTxId, time.Second*5)
	Require(t, err, "WaitForTx failed, waited for first retry tx")
	return autoRedeemReceipt
}

func warpL1Time(t *testing.T, ctx context.Context, l2node *arbnode.Node, l1client *ethclient.Client, l2info info, currentL1time, advanceTime uint64) uint64 {
	t.Log("Warping L1 time...")
	l1LatestHeader, err := l1client.HeaderByNumber(ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
	Require(t, err)
	if currentL1time == 0 {
		currentL1time = l1LatestHeader.Time
	}
	newL1Timestamp := currentL1time + advanceTime
	timeWarpHeader := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: l1LatestHeader.Number.Uint64(),
		Timestamp:   newL1Timestamp,
		RequestId:   nil,
		L1BaseFee:   nil,
	}
	hooks := arbos.NoopSequencingHooks()
	_, err = l2node.Execution.ExecEngine.SequenceTransactions(timeWarpHeader, types.Transactions{
		l2info.PrepareTx("Faucet", "User2", 300000, big.NewInt(1), nil),
	}, hooks)
	Require(t, err)
	return newL1Timestamp
}

// updates retry.NumTries
// returns leaf hash and index in expired acc
func expiredRetryableFromLogs(t *testing.T, ctx context.Context, l2client *ethclient.Client, retry *retryables.TestRetryableData) (uint64, common.Hash) {
	t.Helper()
	arbRetryableAbi, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)
	retryableExpiredTopic := arbRetryableAbi.Events["RetryableExpired"].ID
	// find out expired retryable hash
	logs, err := l2client.FilterLogs(ctx, ethereum.FilterQuery{
		Addresses: []common.Address{
			types.ArbRetryableTxAddress,
		},
		Topics: [][]common.Hash{
			{retryableExpiredTopic},
			nil,
			nil,
			{retry.Id},
		},
	})
	Require(t, err, "FilterLogs failed")
	if len(logs) == 0 {
		Fatal(t, "found no logs for RetryableExpired event for ticket id:", retry.Id)
	}
	var maxNumTries uint64
	var leafHash common.Hash
	var leafIndex uint64
	for _, log := range logs {
		event := &precompilesgen.ArbRetryableTxRetryableExpired{}
		l := log
		err := util.ParseRetryableExpiredLog(event, &l)
		Require(t, err, "ParseRetryableExpiredLog failed for log:", log)
		if event.NumTries >= maxNumTries {
			retry.NumTries = event.NumTries
			leafHash = common.BytesToHash(event.Hash[:])
			leafIndex = event.Position.Uint64()
		}
	}
	return leafIndex, leafHash
}

func accumulatorSizeFromLogs(t *testing.T, ctx context.Context, l2client *ethclient.Client) uint64 {
	t.Helper()
	arbRetryableAbi, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)
	retryableExpiredTopic := arbRetryableAbi.Events["RetryableExpired"].ID
	logs, err := l2client.FilterLogs(ctx, ethereum.FilterQuery{
		Addresses: []common.Address{
			types.ArbRetryableTxAddress,
		},
		Topics: [][]common.Hash{
			{retryableExpiredTopic},
		},
	})
	Require(t, err, "FilterLogs failed")
	if len(logs) == 0 {
		Fatal(t, "found no logs for RetryableExpired event")
	}
	var maxLeaf uint64
	for _, log := range logs {
		event := &precompilesgen.ArbRetryableTxRetryableExpired{}
		l := log
		err := util.ParseRetryableExpiredLog(event, &l)
		Require(t, err, "ParseRetryableExpiredLog failed for log:", log)
		place := merkletree.NewLevelAndLeafFromPostion(event.Position)
		if place.Leaf > maxLeaf {
			maxLeaf = place.Leaf
		}
	}
	return maxLeaf + 1
}

func prepareQueryNeededNodesAndPartials(t *testing.T, ctx context.Context, l2client *ethclient.Client, treeSize, leafIndex uint64) ([]common.Hash, []merkletree.LevelAndLeaf, map[merkletree.LevelAndLeaf]common.Hash) {
	// find which nodes we'll want in our proof up to a partial
	balanced := treeSize == arbmath.NextPowerOf2(treeSize)/2
	treeLevels := int(arbmath.Log2ceil(treeSize)) // the # of levels in the tree
	proofLevels := treeLevels - 1                 // the # of levels where a hash is needed (all but root)
	walkLevels := treeLevels                      // the # of levels we need to consider when building walks
	if balanced {
		walkLevels -= 1 // skip the root
	}
	t.Log("Tree has", treeSize, "leaves and", treeLevels, "levels")
	t.Log("Balanced:", balanced)
	query := make([]common.Hash, 0)             // the nodes we'll query for
	nodes := make([]merkletree.LevelAndLeaf, 0) // the nodes needed (might not be found from query)
	which := uint64(1)                          // which bit to flip & set
	place := leafIndex                          // where we are in the tree
	t.Log("start place:", place)
	for level := 0; level < walkLevels; level++ {
		sibling := place ^ which
		position := merkletree.LevelAndLeaf{
			Level: uint64(level),
			Leaf:  sibling,
		}
		if sibling < treeSize {
			// the sibling must not be newer than the root
			query = append(query, common.BigToHash(position.ToBigInt()))
		}
		nodes = append(nodes, position)
		place |= which // set the bit so that we approach from the right
		which <<= 1    // advance to the next bit
	}
	// find all the partials
	partials := make(map[merkletree.LevelAndLeaf]common.Hash)
	if !balanced {
		power := uint64(1) << proofLevels
		total := uint64(0)
		for level := proofLevels; level >= 0; level-- {
			if (power & treeSize) > 0 { // the partials map to the binary representation of the tree size
				total += power    // The actual leaf for a given partial is the sum of the powers of 2
				leaf := total - 1 // preceding it. We subtract 1 since we count from 0
				partial := merkletree.LevelAndLeaf{
					Level: uint64(level),
					Leaf:  leaf,
				}
				query = append(query, common.BigToHash(partial.ToBigInt()))
				partials[partial] = common.Hash{}
			}
			power >>= 1
		}
	}
	t.Log("Query:", query)
	t.Log("Found", len(partials), "partials:", partials)
	return query, nodes, partials
}

func fetchMerkleNodesFromLogs(t *testing.T, ctx context.Context, l2client *ethclient.Client, query []common.Hash) []merkleAccumulator.MerkleTreeNodeEvent {
	t.Helper()
	logs := []types.Log{}
	arbRetryableAbi, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)
	retryableExpiredTopic := arbRetryableAbi.Events["RetryableExpired"].ID
	expiredMerkleUpdateTopic := arbRetryableAbi.Events["ExpiredMerkleUpdate"].ID
	if len(query) > 0 {
		// in one lookup, query geth for all the data we need to construct a proof
		logs, err = l2client.FilterLogs(ctx, ethereum.FilterQuery{
			Addresses: []common.Address{
				types.ArbRetryableTxAddress,
			},
			Topics: [][]common.Hash{
				{expiredMerkleUpdateTopic, retryableExpiredTopic},
				nil,
				query,
			},
		})
		Require(t, err, "couldn't get logs")
	}
	// leaf and internal nodes
	var merkleNodes []merkleAccumulator.MerkleTreeNodeEvent
	for _, log := range logs {
		var hash common.Hash
		var place merkletree.LevelAndLeaf
		l := log
		switch log.Topics[0] {
		case expiredMerkleUpdateTopic:
			event := &precompilesgen.ArbRetryableTxExpiredMerkleUpdate{}
			err := util.ParseExpiredMerkleUpdateLog(event, &l)
			Require(t, err)
			hash = common.BytesToHash(event.Hash[:])
			place = merkletree.NewLevelAndLeafFromPostion(event.Position)
		case retryableExpiredTopic:
			event := &precompilesgen.ArbRetryableTxRetryableExpired{}
			err := util.ParseRetryableExpiredLog(event, &l)
			Require(t, err)
			hash = common.BytesToHash(event.Hash[:])
			place = merkletree.NewLevelAndLeafFromPostion(event.Position)
		}
		merkleNodes = append(merkleNodes, merkleAccumulator.MerkleTreeNodeEvent{
			Level:     place.Level,
			NumLeaves: place.Leaf,
			Hash:      hash,
		})
	}
	return merkleNodes
}

func proofFromMerkleNodes(t *testing.T, treeSize, leafIndex uint64, leafHash common.Hash, nodes []merkletree.LevelAndLeaf, partials map[merkletree.LevelAndLeaf]common.Hash, merkleNodes []merkleAccumulator.MerkleTreeNodeEvent) (merkletree.MerkleProof, [][32]byte) {
	t.Helper()
	known := make(map[merkletree.LevelAndLeaf]common.Hash) // all values in the tree we know
	partialsByLevel := make(map[uint64]common.Hash)        // maps for each level the partial it may have
	var minPartialPlace *merkletree.LevelAndLeaf           // the lowest-level partial
	for _, merkleNode := range merkleNodes {
		place := merkletree.LevelAndLeaf{Level: merkleNode.Level, Leaf: merkleNode.NumLeaves}
		hash := merkleNode.Hash
		known[place] = hash
		if zero, ok := partials[place]; ok {
			if zero != (common.Hash{}) {
				Fatal(t, "Somehow got 2 partials for the same level\n\t1st:", zero, "\n\t2nd:", hash, "place:", place)
			}
			partials[place] = hash
			partialsByLevel[place.Level] = hash
			if minPartialPlace == nil || place.Level < minPartialPlace.Level {
				minPartialPlace = &place
			}
		}
	}
	for place, hash := range known {
		t.Log("known  ", place.Level, hash, "@", place)
	}
	t.Log(len(known), "values are known\n")
	for place, hash := range partials {
		t.Log("partial", place.Level, hash, "@", place)
	}
	balanced := treeSize == arbmath.NextPowerOf2(treeSize)/2
	t.Log("resolving frontiers\n", "minPartialPlace:", minPartialPlace)
	if !balanced {
		// This tree isn't balanced, so we'll need to use the partials to recover the missing info.
		// To do this, we'll walk the boundary of what's known, computing hashes along the way
		zero := common.Hash{}
		step := *minPartialPlace
		step.Leaf += 1 << step.Level // we start on the min partial's zero-hash sibling
		known[step] = zero
		t.Log("zeroing:", step)
		treeLevels := int(arbmath.Log2ceil(treeSize)) // the # of levels in the tree
		for step.Level < uint64(treeLevels) {
			curr, ok := known[step]
			if !ok {
				Fatal(t, "We should know the current node's value")
			}
			left := curr
			right := curr
			if _, ok := partialsByLevel[step.Level]; ok {
				// a partial on the frontier can only appear on the left
				// moving leftward for a level l skips 2^l leaves
				step.Leaf -= 1 << step.Level
				partial, ok := known[step]
				if !ok {
					Fatal(t, "There should be a partial here")
				}
				left = partial
			} else {
				// getting to the next partial means covering its mirror subtree, so we look right
				// moving rightward for a level l skips 2^l leaves
				step.Leaf += 1 << step.Level
				known[step] = zero
				right = zero
			}
			// move to the parent
			step.Level += 1
			step.Leaf |= 1 << (step.Level - 1)
			known[step] = crypto.Keccak256Hash(left.Bytes(), right.Bytes())
		}
		// if known[step] != rootHash {
		//	// a correct walk of the frontier should end with resolving the root
		//	t.Log("Walking up the tree didn't re-create the root", known[step], "vs", rootHash)
		//}
		for place, hash := range known {
			t.Log("known", place, hash)
		}
	}
	t.Log("Complete proof of leaf", leafIndex)
	t.Log("needed nodes:", nodes)
	proof := make([]common.Hash, len(nodes))
	for i, place := range nodes {
		hash, ok := known[place]
		if !ok {
			Fatal(t, "We're missing data for the node at position", place)
		}
		proof[i] = hash
		t.Log("node", place, hash)
	}
	rootHash := leafHash
	index := leafIndex
	for _, hashFromProof := range proof {
		if index&1 == 0 {
			rootHash = crypto.Keccak256Hash(rootHash.Bytes(), hashFromProof.Bytes())
		} else {
			rootHash = crypto.Keccak256Hash(hashFromProof.Bytes(), rootHash.Bytes())
		}
		index = index / 2
	}
	merkleProof := merkletree.MerkleProof{
		RootHash:  rootHash,
		LeafHash:  leafHash,
		LeafIndex: leafIndex,
		Proof:     proof,
	}
	t.Logf("the proof: %+v", merkleProof)
	if !merkleProof.IsCorrect() {
		Fatal(t, "internal test error - incorrect proof")
	}
	proofBytes := make([][32]byte, len(proof))
	for i, p := range proof {
		proofBytes[i] = *(*[32]byte)(p.Bytes())
	}
	return merkleProof, proofBytes
}

func proofForLeaf(t *testing.T, ctx context.Context, l2client *ethclient.Client, treeSize, leafIndex uint64, leafHash common.Hash) (merkletree.MerkleProof, [][32]byte) {
	t.Helper()
	query, nodes, partials := prepareQueryNeededNodesAndPartials(t, ctx, l2client, treeSize, leafIndex)
	merkleNodes := fetchMerkleNodesFromLogs(t, ctx, l2client, query)
	return proofFromMerkleNodes(t, treeSize, leafIndex, leafHash, nodes, partials, merkleNodes)
}

func reviveRetryable(t *testing.T, ctx context.Context, l2client *ethclient.Client, ownerTxOpts *bind.TransactOpts, rootHashes []common.Hash, treeSizes []uint64, snapshot int, retry *retryables.TestRetryableData) (*types.Transaction, error) {
	t.Helper()
	leafIndex, leafHash := expiredRetryableFromLogs(t, ctx, l2client, retry)
	i := len(rootHashes) - 1 - snapshot
	rootHash, treeSize := rootHashes[i], treeSizes[i]
	if treeSize <= leafIndex {
		// the leaf was added after the snapshot, prove against current root instead
		i = len(rootHashes) - 1
		rootHash, treeSize = rootHashes[i], treeSizes[i]
	}
	merkleProof, proofBytes := proofForLeaf(t, ctx, l2client, treeSize, leafIndex, leafHash)
	if (rootHash != common.Hash{}) {
		if !bytes.Equal(merkleProof.RootHash.Bytes(), rootHash.Bytes()) {
			Fatal(t, "Root hash mismatch, root hash from state:", rootHash, "root hash from proof:", merkleProof.RootHash, "tree size:", treeSize)
		}
	}
	t.Logf("rootHash: %+v, treeSize: %+v, merkleProof: %+v", rootHash, treeSize, merkleProof)
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	return arbRetryableTx.Revive(ownerTxOpts, retry.Id, retry.NumTries, retry.From, retry.To, retry.CallValue, retry.Beneficiary, retry.CallData, merkleProof.RootHash, merkleProof.LeafIndex, proofBytes)
}

func reviveRetryableShouldSucceed(t *testing.T, ctx context.Context, l2client *ethclient.Client, ownerTxOpts *bind.TransactOpts, rootHashes []common.Hash, treeSizes []uint64, snapshot int, retry *retryables.TestRetryableData) {
	t.Helper()
	tx, err := reviveRetryable(t, ctx, l2client, ownerTxOpts, rootHashes, treeSizes, snapshot, retry)
	Require(t, err, "Revive failed, retry data:", fmt.Sprintf("%+v", retry), "snapshot:", snapshot)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "receipt indicated Revive failure")
	}
}

func reviveRetryableShouldFail(t *testing.T, ctx context.Context, l2client *ethclient.Client, ownerTxOpts *bind.TransactOpts, rootHashes []common.Hash, treeSizes []uint64, snapshot int, retry *retryables.TestRetryableData) {
	t.Helper()
	_, err := reviveRetryable(t, ctx, l2client, ownerTxOpts, rootHashes, treeSizes, snapshot, retry)
	if err == nil || err.Error() != "execution reverted: error AlreadyRevived()" {
		Fatal(t, "didn't get expected AlreadyRevived error, err:", err)
	}
}

func redeemRetryableShouldFailNoTicket(t *testing.T, ctx context.Context, l2client *ethclient.Client, ownerTxOpts *bind.TransactOpts, retry *retryables.TestRetryableData) {
	t.Helper()
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	_, err = arbRetryableTx.Redeem(ownerTxOpts, retry.Id)
	if err == nil || err.Error() != "execution reverted: error NoTicketWithID()" {
		Fatal(t, "didn't get expected NoTicketWithID error, err:", err)
	}
}

func redeemRetryableShouldSucceed(t *testing.T, ctx context.Context, l2client *ethclient.Client, ownerTxOpts *bind.TransactOpts, retry *retryables.TestRetryableData, simple *mocksgen.Simple, expectedCounter uint64) {
	t.Helper()
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(ownerTxOpts, retry.Id)
	Require(t, err, "Redeem failed")
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err, "EnsureTxSucceeded failed")
	retryTxId := receipt.Logs[0].Topics[2]
	// check the receipt for the retry
	receipt, err = WaitForTx(ctx, l2client, retryTxId, time.Second*1)
	Require(t, err, "WaitForTx failed")
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, receipt.Status)
	}
	// verify that the increment happened, so we know the retry succeeded
	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err)

	if counter != expectedCounter {
		Fatal(t, "Unexpected counter, want:", expectedCounter, "have:", counter)
	}
}

func lastExpiredRootSnapshot(t *testing.T, l2client *ethclient.Client) (common.Hash, uint64) {
	t.Helper()
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	rootHash, treeSize, err := arbRetryableTx.GetLastExpiredRootSnapshot(&bind.CallOpts{})
	Require(t, err, "LastExpiredRootSnapshot failed")
	return rootHash, treeSize
}

func waitForL1DelayBlocks(t *testing.T, ctx context.Context, l1client *ethclient.Client, l1info info) {
	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}
}

func waitForNextL2Block(t *testing.T, ctx context.Context, l2client *ethclient.Client, l2info info) {
	header, err := l2client.HeaderByNumber(ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
	Require(t, err)
	startBlockNum := header.Number.Uint64()
	// sending l2 messages until new l2 block is created
	for header.Number.Uint64() == startBlockNum {
		SendWaitTestTransactions(t, ctx, l2client, []*types.Transaction{
			l2info.PrepareTx("Faucet", "User2", 300000, big.NewInt(1e12), nil),
		})
		header, err = l2client.HeaderByNumber(ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
		Require(t, err)
	}
}
func TestDepositETH(t *testing.T) {
	t.Parallel()
	_, _, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, false)
	defer teardown()

	faucetAddr := l1info.GetAddress("Faucet")

	oldBalance, err := l2client.BalanceAt(ctx, faucetAddr, nil)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", faucetAddr, err)
	}

	txOpts := l1info.GetDefaultTransactOpts("Faucet", ctx)
	txOpts.Value = big.NewInt(13)

	l1tx, err := delayedInbox.DepositEth0(&txOpts)
	if err != nil {
		t.Fatalf("DepositEth0() unexected error: %v", err)
	}

	l1Receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded() unexpected error: %v", err)
	}
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("Got transaction status: %v, want: %v", l1Receipt.Status, types.ReceiptStatusSuccessful)
	}
	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	l2Receipt, err := EnsureTxSucceeded(ctx, l2client, lookupL2Tx(l1Receipt))
	if err != nil {
		t.Fatalf("EnsureTxSucceeded unexpected error: %v", err)
	}
	newBalance, err := l2client.BalanceAt(ctx, faucetAddr, l2Receipt.BlockNumber)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", faucetAddr, err)
	}
	if got := new(big.Int); got.Sub(newBalance, oldBalance).Cmp(txOpts.Value) != 0 {
		t.Errorf("Got transferred: %v, want: %v", got, txOpts.Value)
	}
}

func TestArbitrumContractTx(t *testing.T) {
	_, l2info, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, false)
	defer teardown()
	faucetL2Addr := util.RemapL1Address(l1info.GetAddress("Faucet"))
	TransferBalanceTo(t, "Faucet", faucetL2Addr, big.NewInt(1e18), l2info, l2client, ctx)

	l2TxOpts := l2info.GetDefaultTransactOpts("Faucet", ctx)
	l2ContractAddr, _ := deploySimple(t, ctx, l2TxOpts, l2client)
	l2ContractABI, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	if err != nil {
		t.Fatalf("Error parsing contract ABI: %v", err)
	}
	data, err := l2ContractABI.Pack("checkCalls", true, true, false, false, false, false)
	if err != nil {
		t.Fatalf("Error packing method's call data: %v", err)
	}
	unsignedTx := types.NewTx(&types.ArbitrumContractTx{
		ChainId:   l2info.Signer.ChainID(),
		From:      faucetL2Addr,
		GasFeeCap: l2info.GasPrice.Mul(l2info.GasPrice, big.NewInt(2)),
		Gas:       1e6,
		To:        &l2ContractAddr,
		Value:     common.Big0,
		Data:      data,
	})
	txOpts := l1info.GetDefaultTransactOpts("Faucet", ctx)
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
	receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", l1tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L1 transaction: %v has failed", l1tx.Hash())
	}
	waitForL1DelayBlocks(t, ctx, l1client, l1info)
	receipt, err = EnsureTxSucceeded(ctx, l2client, lookupL2Tx(receipt))
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", unsignedTx.Hash(), err)
	}
}

func TestL1FundedUnsignedTransaction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	l2info, node, l2client, l1info, _, l1client, l1Stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1Stack)
	defer node.StopAndWait()

	faucetL2Addr := util.RemapL1Address(l1info.GetAddress("Faucet"))
	// Transfer balance to Faucet's corresponding L2 address, so that there is
	// enough balance on its' account for executing L2 transaction.
	TransferBalanceTo(t, "Faucet", faucetL2Addr, big.NewInt(1e18), l2info, l2client, ctx)

	l2TxOpts := l2info.GetDefaultTransactOpts("Faucet", ctx)
	contractAddr, _ := deploySimple(t, ctx, l2TxOpts, l2client)
	contractABI, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	if err != nil {
		t.Fatalf("Error parsing contract ABI: %v", err)
	}
	data, err := contractABI.Pack("checkCalls", true, true, false, false, false, false)
	if err != nil {
		t.Fatalf("Error packing method's call data: %v", err)
	}
	nonce, err := l2client.NonceAt(ctx, faucetL2Addr, nil)
	if err != nil {
		t.Fatalf("Error getting nonce at address: %v, error: %v", faucetL2Addr, err)
	}
	unsignedTx := types.NewTx(&types.ArbitrumUnsignedTx{
		ChainId:   l2info.Signer.ChainID(),
		From:      faucetL2Addr,
		Nonce:     nonce,
		GasFeeCap: l2info.GasPrice,
		Gas:       1e6,
		To:        &contractAddr,
		Value:     common.Big0,
		Data:      data,
	})

	delayedInbox, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	if err != nil {
		t.Fatalf("Error getting Go binding of L1 Inbox contract: %v", err)
	}

	txOpts := l1info.GetDefaultTransactOpts("Faucet", ctx)
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
	receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", l1tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L1 transaction: %v has failed", l1tx.Hash())
	}
	waitForL1DelayBlocks(t, ctx, l1client, l1info)
	receipt, err = EnsureTxSucceeded(ctx, l2client, unsignedTx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", unsignedTx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L2 transaction: %v has failed", receipt.TxHash)
	}
}

func TestRetryableSubmissionAndRedeemFees(t *testing.T) {
	_, l2info, l1info, l2client, l1client, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, false)
	defer teardown()
	infraFeeAddr, networkFeeAddr := setupFeeAddresses(t, ctx, l2client, l2info)

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	simpleAddr, simple := deploySimple(t, ctx, ownerTxOpts, l2client)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)

	elevateL2Basefee(t, ctx, l2client, l2info)

	infraBalanceBefore, err := l2client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceBefore, err := l2client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	beneficiaryAddress := l2info.GetAddress("Beneficiary")
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := common.Big0
	usertxoptsL1 := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit
	baseFee := GetBaseFee(t, l2client, ctx)
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
	l1Receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	submissionTxOuter := lookupL2Tx(l1Receipt)
	submissionReceipt, err := EnsureTxSucceeded(ctx, l2client, submissionTxOuter)
	Require(t, err)
	if len(submissionReceipt.Logs) != 2 {
		Fatal(t, len(submissionReceipt.Logs))
	}
	ticketId := submissionReceipt.Logs[0].Topics[1]
	firstRetryTxId := submissionReceipt.Logs[1].Topics[2]
	// get receipt for the auto redeem, make sure it failed
	autoRedeemReceipt, err := WaitForTx(ctx, l2client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if autoRedeemReceipt.Status != types.ReceiptStatusFailed {
		Fatal(t, "first retry tx shouldn't have succeeded")
	}

	infraBalanceAfterSubmission, err := l2client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfterSubmission, err := l2client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	usertxoptsL2 := l2info.GetDefaultTransactOpts("Faucet", ctx)
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(&usertxoptsL2, ticketId)
	Require(t, err)
	redeemReceipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	retryTxId := redeemReceipt.Logs[0].Topics[2]

	// check the receipt for the retry
	retryReceipt, err := WaitForTx(ctx, l2client, retryTxId, time.Second*1)
	Require(t, err)
	if retryReceipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "retry failed")
	}

	infraBalanceAfterRedeem, err := l2client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfterRedeem, err := l2client.BalanceAt(ctx, networkFeeAddr, nil)
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

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), l2client)
	Require(t, err)
	minimumBaseFee, err := arbGasInfo.GetMinimumGasPrice(&bind.CallOpts{Context: ctx})
	Require(t, err)
	submissionBaseFee := GetBaseFeeAt(t, l2client, ctx, submissionReceipt.BlockNumber)
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

	retryTxOuter, _, err := l2client.TransactionByHash(ctx, retryTxId)
	Require(t, err)
	retryTx, ok := retryTxOuter.GetInner().(*types.ArbitrumRetryTx)
	if !ok {
		Fatal(t, "inner tx isn't ArbitrumRetryTx")
	}
	redeemBaseFee := GetBaseFeeAt(t, l2client, ctx, redeemReceipt.BlockNumber)

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
func elevateL2Basefee(t *testing.T, ctx context.Context, l2client *ethclient.Client, l2info *BlockchainTestInfo) {
	baseFeeBefore := GetBaseFee(t, l2client, ctx)
	colors.PrintBlue("Elevating base fee...")
	arbostestabi, err := precompilesgen.ArbosTestMetaData.GetAbi()
	Require(t, err)
	_, err = precompilesgen.NewArbosTest(common.HexToAddress("0x69"), l2client)
	Require(t, err, "failed to deploy ArbosTest")

	burnAmount := arbnode.ConfigDefaultL1Test().RPC.RPCGasCap
	burnTarget := uint64(5 * l2pricing.InitialSpeedLimitPerSecondV6 * l2pricing.InitialBacklogTolerance)
	for i := uint64(0); i < (burnTarget+burnAmount)/burnAmount; i++ {
		burnArbGas := arbostestabi.Methods["burnArbGas"]
		data, err := burnArbGas.Inputs.Pack(arbmath.UintToBig(burnAmount - l2info.TransferGas))
		Require(t, err)
		input := append([]byte{}, burnArbGas.ID...)
		input = append(input, data...)
		to := common.HexToAddress("0x69")
		tx := l2info.PrepareTxTo("Faucet", &to, burnAmount, big.NewInt(0), input)
		Require(t, l2client.SendTransaction(ctx, tx))
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}
	baseFee := GetBaseFee(t, l2client, ctx)
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
