// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/gasestimator"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func retryableSetup(t *testing.T, modifyNodeConfig ...func(*NodeBuilder)) (
	*NodeBuilder,
	*bridgegen.Inbox,
	func(*types.Receipt) *types.Transaction,
	context.Context,
	func(),
) {
	ctx, cancel := context.WithCancel(context.Background())
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	for _, f := range modifyNodeConfig {
		f(builder)
	}

	// retryableSetup is being called by tests that validate blocks.
	// For now validation only works with HashScheme set.
	builder.RequireScheme(t, rawdb.HashScheme)
	builder.nodeConfig.BlockValidator.Enable = false
	builder.nodeConfig.Staker.Enable = true
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.ParentChainReader.Enable = true
	builder.nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute

	valConf := valnode.TestValidationConfig
	valConf.UseJit = true
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(builder.nodeConfig, valStack)

	builder.execConfig.Sequencer.MaxRevertGasReject = 0

	builder.Build(t)

	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("Beneficiary")
	builder.L2Info.GenerateAccount("Burn")

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	Require(t, err)
	delayedBridge, err := arbnode.NewDelayedBridge(builder.L1.Client, builder.L1Info.GetAddress("Bridge"), 0)
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
			txs, err := arbos.ParseL2Transactions(message.Message, chaininfo.ArbitrumDevTestChainConfig().ChainID, params.MaxDebugArbosVersionSupported)
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
	builder.L2.TransferBalance(t, "Faucet", "Burn", discard, builder.L2Info)

	teardown := func() {

		// check the integrity of the RPC
		blockNum, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err, "failed to get L2 block number")
		for number := uint64(0); number < blockNum; number++ {
			block, err := builder.L2.Client.BlockByNumber(ctx, arbmath.UintToBig(number))
			Require(t, err, "failed to get L2 block", number, "of", blockNum)
			if block.Number().Uint64() != number {
				Fatal(t, "block number mismatch", number, block.Number().Uint64())
			}
		}

		cancel()

		builder.L2.ConsensusNode.StopAndWait()
		requireClose(t, builder.L1.Stack)
	}
	return builder, delayedInbox, lookupL2Tx, ctx, teardown
}

func TestRetryableNoExist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, common.Hash{})
	// The first error is server side. The second error is client side ABI decoding.
	if err.Error() != "execution reverted: error NoTicketWithID(): NoTicketWithID()" {
		Fatal(t, "didn't get expected NoTicketWithID error")
	}
}

func TestEstimateRetryableTicketWithNoFundsAndZeroGasPrice(t *testing.T) {
	builder, _, _, ctx, teardown := retryableSetup(t)
	defer teardown()

	user2Address := builder.L2Info.GetAddress("User2")
	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err, "failed to deploy NodeInterface")

	builder.L2Info.GenerateAccount("zerofunds")
	usertxoptsL2 := builder.L2Info.GetDefaultTransactOpts("zerofunds", ctx)
	usertxoptsL2.NoSend = true
	usertxoptsL2.GasMargin = 0
	usertxoptsL2.GasPrice = big.NewInt(0)
	_, err = nodeInterface.EstimateRetryableTicket(
		&usertxoptsL2,
		usertxoptsL2.From,
		deposit,
		user2Address,
		callValue,
		beneficiaryAddress,
		beneficiaryAddress,
		[]byte{},
	)
	Require(t, err, "failed to estimate retryable submission")
}

func TestSubmitRetryableImmediateSuccess(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	user2Address := builder.L2Info.GetAddress("User2")
	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err, "failed to deploy NodeInterface")

	// estimate the gas needed to auto redeem the retryable
	usertxoptsL2 := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
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
	expectedEstimate := params.TxGas + params.TxDataNonZeroGasEIP2028*4
	if float64(estimate) > float64(expectedEstimate)*(1+gasestimator.EstimateGasErrorRatio) {
		t.Errorf("estimated retryable ticket at %v gas but expected %v, with error margin of %v",
			estimate,
			expectedEstimate,
			gasestimator.EstimateGasErrorRatio,
		)
	}

	// submit & auto redeem the retryable using the gas estimate
	usertxoptsL1 := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
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

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t)
	}

	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if !arbmath.BigEquals(l2balance, callValue) {
		Fatal(t, "Unexpected balance:", l2balance)
	}
	testFlatCallTracer(t, ctx, builder.L2.Client.Client())
}

func testSubmitRetryableEmptyEscrow(t *testing.T, arbosVersion uint64) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, func(builder *NodeBuilder) {
		builder.WithArbOSVersion(arbosVersion)
		builder.WithDatabase(rawdb.DBPebble)
	})
	defer teardown()

	user2Address := builder.L2Info.GetAddress("User2")
	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := common.Big0

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err, "failed to deploy NodeInterface")

	// estimate the gas needed to auto redeem the retryable
	usertxoptsL2 := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
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
	usertxoptsL1 := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
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

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	l2Tx := lookupL2Tx(l1Receipt)
	receipt, err := builder.L2.EnsureTxSucceeded(l2Tx)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t)
	}

	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)

	if !arbmath.BigEquals(l2balance, callValue) {
		Fatal(t, "Unexpected balance:", l2balance)
	}

	escrowAccount := retryables.RetryableEscrowAddress(l2Tx.Hash())
	state, err := builder.L2.ExecNode.ArbInterface.BlockChain().State()
	Require(t, err)
	escrowExists := state.Exist(escrowAccount)
	if escrowExists != (arbosVersion < params.ArbosVersion_30) {
		Fatal(t, "Escrow account existence", escrowExists, "doesn't correspond to ArbOS version", arbosVersion)
	}
}

func TestSubmitRetryableEmptyEscrowArbOS20(t *testing.T) {
	testSubmitRetryableEmptyEscrow(t, 20)
}

func TestSubmitRetryableEmptyEscrowArbOS30(t *testing.T) {
	testSubmitRetryableEmptyEscrow(t, 30)
}

func TestSubmitRetryableFailThenRetry(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, simple := builder.L2.DeploySimple(t, ownerTxOpts)
	simpleABI, err := localgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
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

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
	Require(t, err)
	if len(receipt.Logs) != 2 {
		Fatal(t, len(receipt.Logs))
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// get receipt for the auto redeem, make sure it failed
	receipt, err = WaitForTx(ctx, builder.L2.Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fatal(t, receipt.GasUsed)
	}

	l2FaucetTxOpts := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	l2FaucetTxOpts.GasLimit = 0 // gas estimation
	l2FaucetTxOpts.Value = big.NewInt(2)
	l2FaucetTxOpts.NoSend = true
	expectedErr := fmt.Errorf("retryable with ticketId: %v not found", ticketId)
	_, err = simple.RedeemAllAndCreateAddresses(&l2FaucetTxOpts, [][32]byte{ticketId, ticketId}, []common.Address{testhelpers.RandomAddress(), testhelpers.RandomAddress()})
	if err == nil {
		t.Fatal("expected non-nil error for gas estimation of duplicate retryable redeems")
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("unexpected error for gas estimation of duplicate retryable redeems. Want: %v, Got: %v", expectedErr, err)
	}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(&ownerTxOpts, ticketId)
	Require(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	redemptionL2Gas := receipt.GasUsed - receipt.GasUsedForL1
	var maxRedemptionL2Gas uint64 = 1_000_000
	if redemptionL2Gas > maxRedemptionL2Gas {
		t.Errorf("manual retryable redemption used %v gas, more than expected max %v gas", redemptionL2Gas, maxRedemptionL2Gas)
	}

	retryTxId := receipt.Logs[0].Topics[2]

	// check the receipt for the retry
	receipt, err = WaitForTx(ctx, builder.L2.Client, retryTxId, time.Second*1)
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
	testFlatCallTracer(t, ctx, builder.L2.Client.Client())
}

// insertRetriables inserts n retryable transactions into the delayed
// inbox and returns the receipts for the retryables.
//
// Each retryable transaction sends 10 wei to the benificiary.
func insertRetriables(
	t *testing.T,
	n uint64,
	inbox *bridgegen.Inbox,
	bld *NodeBuilder,
	userTxOpts bind.TransactOpts,
	receiveAddress common.Address,
	retryableL2CallValue *big.Int,
	maxSubmissionFee *big.Int,
	feeRefundAddress common.Address,
	beneficiaryAddress common.Address,
	retryableGas *big.Int,
	gasFeeCap *big.Int,
	retryableCallData []byte,
) []*types.Receipt {
	t.Helper()
	receipts := make([]*types.Receipt, n)
	for i := uint64(0); i < n; i++ {
		tx, err := inbox.CreateRetryableTicket(
			&userTxOpts,
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

		l1Receipt, err := bld.L1.EnsureTxSucceeded(tx)
		Require(t, err)
		if l1Receipt.Status != types.ReceiptStatusSuccessful {
			Fatal(t, fmt.Sprintf("l1Receipt %d indicated failure", i))
		}

		waitForL1DelayBlocks(t, bld)

		receipts[i] = l1Receipt
	}
	return receipts
}

func TestSubmitManyRetryableFailThenRetry(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, func(b *NodeBuilder) {
		b.WithDatabase(rawdb.DBPebble)
	})
	defer teardown()
	infraFeeAddr, networkFeeAddr := setupFeeAddresses(t, ctx, builder)
	elevateL2Basefee(t, ctx, builder)

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	userTxOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	userTxOpts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	builder.L2Info.GenerateAccount("Refund")
	builder.L2Info.GenerateAccount("Receive")
	l1FaucetAddress := builder.L1Info.GetAddress("Faucet")
	faucetAddress := util.RemapL1Address(builder.L1Info.GetAddress("Faucet"))
	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
	feeRefundAddress := builder.L2Info.GetAddress("Refund")
	receiveAddress := builder.L2Info.GetAddress("Receive")

	colors.PrintMint("L1 Faucet   ", l1FaucetAddress)
	colors.PrintBlue("Faucet      ", faucetAddress)
	colors.PrintBlue("Receive     ", receiveAddress)
	colors.PrintBlue("Beneficiary ", beneficiaryAddress)
	colors.PrintBlue("Fee Refund  ", feeRefundAddress)

	fundsBeforeSubmit, err := builder.L2.Client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)

	infraBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	// usefulGas := params.TxGas
	excessGasLimit := uint64(808)

	maxSubmissionFee := big.NewInt(1e14)
	retryableGas := arbmath.UintToBig(100) // Don't send enough gas
	retryableL2CallValue := big.NewInt(1e4)
	retryableCallData := []byte{}
	gasFeeCap := big.NewInt(l2pricing.InitialBaseFeeWei * 2)

	simpleAddress, simple := builder.L2.DeploySimple(t, ownerTxOpts)
	colors.PrintBlue("Simple      ", simpleAddress)

	rCnt := uint64(50)
	reciepts := insertRetriables(
		t,
		rCnt,
		delayedInbox,
		builder,
		userTxOpts,
		receiveAddress,
		retryableL2CallValue,
		maxSubmissionFee,
		feeRefundAddress,
		beneficiaryAddress,
		retryableGas,
		gasFeeCap,
		retryableCallData)

	ticketIds := make([][32]byte, rCnt)
	for idx, l1Receipt := range reciepts {
		receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
		Require(t, err)
		if len(receipt.Logs) != 1 {
			Fatal(t, len(receipt.Logs))
		}
		ticketId := receipt.Logs[0].Topics[1]
		ticketIds[idx] = ticketId
	}

	l2FaucetTxOpts := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	l2FaucetTxOpts.GasLimit = uint64(1e8)
	l2FaucetAddress := builder.L2Info.GetAddress("Faucet")
	colors.PrintBlue("L2 Faucet   ", l2FaucetAddress)
	var addressesToCreate []common.Address
	for range 50 {
		addressesToCreate = append(addressesToCreate, testhelpers.RandomAddress())
	}
	l2FaucetTxOpts.Value = big.NewInt(int64(len(addressesToCreate)))
	tx, err := simple.RedeemAllAndCreateAddresses(&l2FaucetTxOpts, ticketIds, addressesToCreate)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	redemptionL2Gas := receipt.GasUsed - receipt.GasUsedForL1
	var maxRedemptionL2Gas uint64 = 1_000_000_000
	if redemptionL2Gas > maxRedemptionL2Gas {
		t.Errorf("manual retryable redemption used %v gas, more than expected max %v gas", redemptionL2Gas, maxRedemptionL2Gas)
	}

	// verify that the increment happened, so we know the retry succeeded
	counter, err := simple.Counter(&bind.CallOpts{})
	Require(t, err)

	if counter != rCnt {
		Fatal(t, "Unexpected counter:", counter)
	}

	redeemBlock, err := builder.L2.Client.HeaderByNumber(ctx, receipt.BlockNumber)
	Require(t, err)

	l2BaseFee := redeemBlock.BaseFee
	excessGasPrice := arbmath.BigSub(gasFeeCap, l2BaseFee)
	excessWei := arbmath.BigMulByUint(l2BaseFee, excessGasLimit)
	excessWei.Add(excessWei, arbmath.BigMul(excessGasPrice, retryableGas))

	fundsAfterSubmit, err := builder.L2.Client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)
	beneficiaryFunds, err := builder.L2.Client.BalanceAt(ctx, beneficiaryAddress, nil)
	Require(t, err)
	refundFunds, err := builder.L2.Client.BalanceAt(ctx, feeRefundAddress, nil)
	Require(t, err)
	receiveFunds, err := builder.L2.Client.BalanceAt(ctx, receiveAddress, nil)
	Require(t, err)

	infraBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, networkFeeAddr, nil)
	Require(t, err)

	colors.PrintBlue("CallGas    ", retryableGas)
	colors.PrintMint("Gas cost   ", arbmath.BigMul(retryableGas, l2BaseFee))
	colors.PrintBlue("Payment    ", userTxOpts.Value)

	colors.PrintMint("Faucet before ", fundsBeforeSubmit)
	colors.PrintMint("Faucet after  ", fundsAfterSubmit)

	// the retryable should pay the receiver the supplied callvalue
	colors.PrintMint("Receive       ", receiveFunds)
	colors.PrintBlue("L2 Call Value ", retryableL2CallValue)

	bigCount := big.NewInt(int64(rCnt))
	if !arbmath.BigEquals(receiveFunds, arbmath.BigMul(retryableL2CallValue, bigCount)) {
		Fatal(t, "Recipient didn't receive the right funds")
	}

	// the beneficiary should receive nothing
	colors.PrintMint("Beneficiary   ", beneficiaryFunds)
	if beneficiaryFunds.Sign() != 0 {
		Fatal(t, "The beneficiary shouldn't have received funds")
	}

	// the fee refund address should receive the excess gas
	colors.PrintBlue("Base Fee         ", l2BaseFee)
	colors.PrintBlue("Excess Gas Price ", excessGasPrice)
	colors.PrintBlue("Excess Gas       ", excessGasLimit)
	colors.PrintBlue("Excess Wei       ", excessWei)
	colors.PrintMint("Fee Refund       ", refundFunds)

	infraFee := arbmath.BigSub(infraBalanceAfter, infraBalanceBefore)
	networkFee := arbmath.BigSub(networkBalanceAfter, networkBalanceBefore)
	fee := arbmath.BigAdd(infraFee, networkFee)

	colors.PrintMint("paid infra fee:      ", infraFee)
	colors.PrintMint("paid network fee:    ", networkFee)
	colors.PrintMint("paid fee:            ", fee)

	validateBlockRange(t, []uint64{receipt.BlockNumber.Uint64()}, true, builder)
}

func TestGetLifetime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)

	lifetime, err := arbRetryableTx.GetLifetime(callOpts)
	Require(t, err)
	if lifetime.Cmp(big.NewInt(retryables.RetryableLifetimeSeconds)) != 0 {
		t.Fatal("Expected to be ", retryables.RetryableLifetimeSeconds, " but got ", lifetime)
	}
}

func warpL1Time(t *testing.T, builder *NodeBuilder, ctx context.Context, currentL1time, advanceTime uint64) uint64 {
	t.Log("Warping L1 time...")
	l1LatestHeader, err := builder.L1.Client.HeaderByNumber(ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
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
	tx := builder.L2Info.PrepareTx("Faucet", "User2", 300000, big.NewInt(1), nil)
	hooks := gethexec.MakeZeroTxSizeSequencingHooksForTesting(types.Transactions{tx}, nil, nil, nil)
	_, err = builder.L2.ExecNode.ExecEngine.SequenceTransactions(timeWarpHeader, hooks, nil)
	Require(t, err)
	return newL1Timestamp
}

func TestRetryableExpiry(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, _ := builder.L2.DeploySimple(t, ownerTxOpts)
	simpleABI, err := localgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
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

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
	Require(t, err)
	if len(receipt.Logs) != 2 {
		Fatal(t, len(receipt.Logs))
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// make sure it failed
	receipt, err = WaitForTx(ctx, builder.L2.Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fatal(t, receipt.GasUsed)
	}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)

	// check that the ticket exists
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	Require(t, err)

	_ = warpL1Time(t, builder, ctx, 0, retryables.RetryableLifetimeSeconds)

	// check that the ticket no longer exists
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	if (err == nil) || (err.Error() != "execution reverted: error NoTicketWithID(): NoTicketWithID()") {
		Fatal(t, "didn't get expected NoTicketWithID error")
	}
}

func TestKeepaliveAndRetryableExpiry(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, _ := builder.L2.DeploySimple(t, ownerTxOpts)
	simpleABI, err := localgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
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

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
	Require(t, err)
	if len(receipt.Logs) != 2 {
		Fatal(t, len(receipt.Logs))
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// make sure it failed
	receipt, err = WaitForTx(ctx, builder.L2.Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fatal(t, receipt.GasUsed)
	}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)

	// checks that the ticket exists and gets current timeout
	timeoutBeforeKeepalive, err := arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	Require(t, err)

	// checks beneficiary
	retrievedBeneficiaryAddress, err := arbRetryableTx.GetBeneficiary(&bind.CallOpts{}, ticketId)
	Require(t, err)
	if retrievedBeneficiaryAddress != beneficiaryAddress {
		Fatal(t, "expected beneficiary to be", beneficiaryAddress, "but got", retrievedBeneficiaryAddress)
	}

	// checks that keepalive increases the timeout as expected
	_, err = arbRetryableTx.Keepalive(&ownerTxOpts, ticketId)
	Require(t, err)
	timeoutAfterKeepalive, err := arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	Require(t, err)
	expectedTimeoutAfterKeepAlive := arbmath.BigAdd(timeoutBeforeKeepalive, big.NewInt(retryables.RetryableLifetimeSeconds))
	if timeoutAfterKeepalive.Cmp(expectedTimeoutAfterKeepAlive) != 0 {
		Fatal(t, "expected timeout after keepalive to be", expectedTimeoutAfterKeepAlive, "but got", timeoutAfterKeepalive)
	}

	currentL1time := warpL1Time(t, builder, ctx, 0, retryables.RetryableLifetimeSeconds)

	// check that the ticket still exists
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	Require(t, err)

	_ = warpL1Time(t, builder, ctx, currentL1time, retryables.RetryableLifetimeSeconds)

	// check that the ticket no longer exists
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	if (err == nil) || (err.Error() != "execution reverted: error NoTicketWithID(): NoTicketWithID()") {
		Fatal(t, "didn't get expected NoTicketWithID error")
	}
}

func TestKeepaliveAndCancelRetryable(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	usertxopts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, _ := builder.L2.DeploySimple(t, ownerTxOpts)
	simpleABI, err := localgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
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

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
	Require(t, err)
	if len(receipt.Logs) != 2 {
		Fatal(t, len(receipt.Logs))
	}
	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]

	// make sure it failed
	receipt, err = WaitForTx(ctx, builder.L2.Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fatal(t, receipt.GasUsed)
	}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)

	// checks that the ticket exists and gets current timeout
	timeoutBeforeKeepalive, err := arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	Require(t, err)

	// checks beneficiary
	retrievedBeneficiaryAddress, err := arbRetryableTx.GetBeneficiary(&bind.CallOpts{}, ticketId)
	Require(t, err)
	if retrievedBeneficiaryAddress != beneficiaryAddress {
		Fatal(t, "expected beneficiary to be", beneficiaryAddress, "but got", retrievedBeneficiaryAddress)
	}

	// checks that keepalive increases the timeout as expected
	_, err = arbRetryableTx.Keepalive(&ownerTxOpts, ticketId)
	Require(t, err)
	timeoutAfterKeepalive, err := arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	Require(t, err)
	expectedTimeoutAfterKeepAlive := arbmath.BigAdd(timeoutBeforeKeepalive, big.NewInt(retryables.RetryableLifetimeSeconds))
	if timeoutAfterKeepalive.Cmp(expectedTimeoutAfterKeepAlive) != 0 {
		Fatal(t, "expected timeout after keepalive to be", expectedTimeoutAfterKeepAlive, "but got", timeoutAfterKeepalive)
	}

	// cancel the ticket
	beneficiaryTxOpts := builder.L2Info.GetDefaultTransactOpts("Beneficiary", ctx)
	tx, err := arbRetryableTx.Cancel(&beneficiaryTxOpts, ticketId)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// checks that the ticket no longer exists
	_, err = arbRetryableTx.GetTimeout(&bind.CallOpts{}, ticketId)
	if (err == nil) || (err.Error() != "execution reverted: error NoTicketWithID(): NoTicketWithID()") {
		Fatal(t, "didn't get expected NoTicketWithID error")
	}
}

func TestSubmissionGasCosts(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()
	infraFeeAddr, networkFeeAddr := setupFeeAddresses(t, ctx, builder)
	elevateL2Basefee(t, ctx, builder)

	usertxopts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxopts.Value = arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	builder.L2Info.GenerateAccount("Refund")
	builder.L2Info.GenerateAccount("Receive")
	faucetAddress := util.RemapL1Address(builder.L1Info.GetAddress("Faucet"))
	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
	feeRefundAddress := builder.L2Info.GetAddress("Refund")
	receiveAddress := builder.L2Info.GetAddress("Receive")

	colors.PrintBlue("Faucet      ", faucetAddress)
	colors.PrintBlue("Receive     ", receiveAddress)
	colors.PrintBlue("Beneficiary ", beneficiaryAddress)
	colors.PrintBlue("Fee Refund  ", feeRefundAddress)

	fundsBeforeSubmit, err := builder.L2.Client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)

	infraBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceBefore, err := builder.L2.Client.BalanceAt(ctx, networkFeeAddr, nil)
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

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	submissionTxOuter := lookupL2Tx(l1Receipt)
	submissionReceipt, err := builder.L2.EnsureTxSucceeded(submissionTxOuter)
	Require(t, err)
	if len(submissionReceipt.Logs) != 2 {
		Fatal(t, "Unexpected number of logs:", len(submissionReceipt.Logs))
	}
	firstRetryTxId := submissionReceipt.Logs[1].Topics[2]
	// get receipt for the auto redeem
	redeemReceipt, err := WaitForTx(ctx, builder.L2.Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if redeemReceipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "first retry tx failed")
	}
	redeemBlock, err := builder.L2.Client.HeaderByNumber(ctx, redeemReceipt.BlockNumber)
	Require(t, err)

	l2BaseFee := redeemBlock.BaseFee
	excessGasPrice := arbmath.BigSub(gasFeeCap, l2BaseFee)
	excessWei := arbmath.BigMulByUint(l2BaseFee, excessGasLimit)
	excessWei.Add(excessWei, arbmath.BigMul(excessGasPrice, retryableGas))

	fundsAfterSubmit, err := builder.L2.Client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)
	beneficiaryFunds, err := builder.L2.Client.BalanceAt(ctx, beneficiaryAddress, nil)
	Require(t, err)
	refundFunds, err := builder.L2.Client.BalanceAt(ctx, feeRefundAddress, nil)
	Require(t, err)
	receiveFunds, err := builder.L2.Client.BalanceAt(ctx, receiveAddress, nil)
	Require(t, err)

	infraBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, infraFeeAddr, nil)
	Require(t, err)
	networkBalanceAfter, err := builder.L2.Client.BalanceAt(ctx, networkFeeAddr, nil)
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
		Fatal(t, "Recipient didn't receive the right funds")
	}

	// the beneficiary should receive nothing
	colors.PrintMint("Beneficiary   ", beneficiaryFunds)
	if beneficiaryFunds.Sign() != 0 {
		Fatal(t, "The beneficiary shouldn't have received funds")
	}

	// the fee refund address should receive the excess gas
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

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), builder.L2.Client)
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

func waitForL1DelayBlocks(t *testing.T, builder *NodeBuilder) {
	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}
}

func TestDepositETH(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	faucetAddr := builder.L1Info.GetAddress("Faucet")

	oldBalance, err := builder.L2.Client.BalanceAt(ctx, faucetAddr, nil)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", faucetAddr, err)
	}

	txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	txOpts.Value = big.NewInt(13)

	l1tx, err := delayedInbox.DepositEth439370b1(&txOpts)
	if err != nil {
		t.Fatalf("DepositEth0() unexected error: %v", err)
	}

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded() unexpected error: %v", err)
	}
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("Got transaction status: %v, want: %v", l1Receipt.Status, types.ReceiptStatusSuccessful)
	}
	waitForL1DelayBlocks(t, builder)

	l2Receipt, err := builder.L2.EnsureTxSucceeded(lookupL2Tx(l1Receipt))
	if err != nil {
		t.Fatalf("EnsureTxSucceeded unexpected error: %v", err)
	}
	newBalance, err := builder.L2.Client.BalanceAt(ctx, faucetAddr, l2Receipt.BlockNumber)
	if err != nil {
		t.Fatalf("BalanceAt(%v) unexpected error: %v", faucetAddr, err)
	}
	if got := new(big.Int); got.Sub(newBalance, oldBalance).Cmp(txOpts.Value) != 0 {
		t.Errorf("Got transferred: %v, want: %v", got, txOpts.Value)
	}
	testFlatCallTracer(t, ctx, builder.L2.Client.Client())
}

func TestArbitrumContractTx(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()
	faucetL2Addr := util.RemapL1Address(builder.L1Info.GetAddress("Faucet"))
	builder.L2.TransferBalanceTo(t, "Faucet", faucetL2Addr, big.NewInt(1e18), builder.L2Info)

	l2TxOpts := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	l2ContractAddr, _ := builder.L2.DeploySimple(t, l2TxOpts)
	l2ContractABI, err := abi.JSON(strings.NewReader(localgen.SimpleABI))
	if err != nil {
		t.Fatalf("Error parsing contract ABI: %v", err)
	}
	data, err := l2ContractABI.Pack("checkCalls", true, true, false, false, false, false)
	if err != nil {
		t.Fatalf("Error packing method's call data: %v", err)
	}
	unsignedTx := types.NewTx(&types.ArbitrumContractTx{
		ChainId:   builder.L2Info.Signer.ChainID(),
		From:      faucetL2Addr,
		GasFeeCap: builder.L2Info.GasPrice.Mul(builder.L2Info.GasPrice, big.NewInt(2)),
		Gas:       1e6,
		To:        &l2ContractAddr,
		Value:     common.Big0,
		Data:      data,
	})
	txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
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
	receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", l1tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L1 transaction: %v has failed", l1tx.Hash())
	}
	waitForL1DelayBlocks(t, builder)
	_, err = builder.L2.EnsureTxSucceeded(lookupL2Tx(receipt))
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", unsignedTx.Hash(), err)
	}
	testFlatCallTracer(t, ctx, builder.L2.Client.Client())
}

func TestL1FundedUnsignedTransaction(t *testing.T) {
	ctx := context.Background()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	faucetL2Addr := util.RemapL1Address(builder.L1Info.GetAddress("Faucet"))
	// Transfer balance to Faucet's corresponding L2 address, so that there is
	// enough balance on its' account for executing L2 transaction.
	builder.L2.TransferBalanceTo(t, "Faucet", faucetL2Addr, big.NewInt(1e18), builder.L2Info)

	l2TxOpts := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	contractAddr, _ := builder.L2.DeploySimple(t, l2TxOpts)
	contractABI, err := abi.JSON(strings.NewReader(localgen.SimpleABI))
	if err != nil {
		t.Fatalf("Error parsing contract ABI: %v", err)
	}
	data, err := contractABI.Pack("checkCalls", true, true, false, false, false, false)
	if err != nil {
		t.Fatalf("Error packing method's call data: %v", err)
	}
	nonce, err := builder.L2.Client.NonceAt(ctx, faucetL2Addr, nil)
	if err != nil {
		t.Fatalf("Error getting nonce at address: %v, error: %v", faucetL2Addr, err)
	}
	unsignedTx := types.NewTx(&types.ArbitrumUnsignedTx{
		ChainId:   builder.L2Info.Signer.ChainID(),
		From:      faucetL2Addr,
		Nonce:     nonce,
		GasFeeCap: builder.L2Info.GasPrice,
		Gas:       1e6,
		To:        &contractAddr,
		Value:     common.Big0,
		Data:      data,
	})

	delayedInbox, err := bridgegen.NewInbox(builder.L1Info.GetAddress("Inbox"), builder.L1.Client)
	if err != nil {
		t.Fatalf("Error getting Go binding of L1 Inbox contract: %v", err)
	}

	txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
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
	receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", l1tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L1 transaction: %v has failed", l1tx.Hash())
	}
	waitForL1DelayBlocks(t, builder)
	receipt, err = builder.L2.EnsureTxSucceeded(unsignedTx)
	if err != nil {
		t.Fatalf("EnsureTxSucceeded(%v) unexpected error: %v", unsignedTx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("L2 transaction: %v has failed", receipt.TxHash)
	}
	testFlatCallTracer(t, ctx, builder.L2.Client.Client())
}

func TestRetryableSubmissionAndRedeemFees(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, func(b *NodeBuilder) {
		b.WithDatabase(rawdb.DBPebble)
	})
	defer teardown()
	infraFeeAddr, networkFeeAddr := setupFeeAddresses(t, ctx, builder)

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	simpleAddr, simple := builder.L2.DeploySimple(t, ownerTxOpts)
	simpleABI, err := localgen.SimpleMetaData.GetAbi()
	Require(t, err)

	elevateL2Basefee(t, ctx, builder)

	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := common.Big0
	usertxoptsL1 := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit
	baseFee := builder.L2.GetBaseFee(t)
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
	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	elevateL2Basefee(t, ctx, builder)

	waitForL1DelayBlocks(t, builder)

	elevateL2Basefee(t, ctx, builder)

	submissionTxOuter := lookupL2Tx(l1Receipt)
	submissionReceipt, err := builder.L2.EnsureTxSucceeded(submissionTxOuter)
	Require(t, err)
	if len(submissionReceipt.Logs) != 2 {
		Fatal(t, len(submissionReceipt.Logs))
	}
	ticketId := submissionReceipt.Logs[0].Topics[1]
	firstRetryTxId := submissionReceipt.Logs[1].Topics[2]
	// get receipt for the auto redeem, make sure it failed
	autoRedeemReceipt, err := WaitForTx(ctx, builder.L2.Client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if autoRedeemReceipt.Status != types.ReceiptStatusFailed {
		Fatal(t, "first retry tx shouldn't have succeeded")
	}

	usertxoptsL2 := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(&usertxoptsL2, ticketId)
	Require(t, err)
	redeemReceipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	retryTxId := redeemReceipt.Logs[0].Topics[2]

	// check the receipt for the retry
	retryReceipt, err := WaitForTx(ctx, builder.L2.Client, retryTxId, time.Second*1)
	Require(t, err)
	if retryReceipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "retry failed")
	}

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

	infraSubmissionFee, err := builder.L2.BalanceDifferenceAtBlock(infraFeeAddr, submissionReceipt.BlockNumber)
	Require(t, err)
	networkSubmissionFee, err := builder.L2.BalanceDifferenceAtBlock(networkFeeAddr, submissionReceipt.BlockNumber)
	Require(t, err)
	infraRedeemFee, err := builder.L2.BalanceDifferenceAtBlock(infraFeeAddr, retryReceipt.BlockNumber)
	Require(t, err)
	networkRedeemFee, err := builder.L2.BalanceDifferenceAtBlock(networkFeeAddr, retryReceipt.BlockNumber)
	Require(t, err)

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), builder.L2.Client)
	Require(t, err)
	minimumBaseFee, err := arbGasInfo.GetMinimumGasPrice(&bind.CallOpts{Context: ctx})
	Require(t, err)
	submissionBaseFee := builder.L2.GetBaseFeeAt(t, submissionReceipt.BlockNumber)
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

	retryTxOuter, _, err := builder.L2.Client.TransactionByHash(ctx, retryTxId)
	Require(t, err)
	retryTx, ok := retryTxOuter.GetInner().(*types.ArbitrumRetryTx)
	if !ok {
		Fatal(t, "inner tx isn't ArbitrumRetryTx")
	}
	redeemBaseFee := builder.L2.GetBaseFeeAt(t, redeemReceipt.BlockNumber)

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
	validateBlocks(t, 1, true, builder)
}

func TestRetryableRedeemBlockGasUsage(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()
	l2client := builder.L2.Client
	l2info := builder.L2Info
	l1client := builder.L1.Client
	l1info := builder.L1Info

	_, err := precompilesgen.NewArbosTest(common.HexToAddress("0x69"), l2client)
	Require(t, err, "failed to deploy ArbosTest")
	_, err = precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	simpleAddr, _ := deploySimple(t, ctx, ownerTxOpts, l2client)
	simpleABI, err := localgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := l2info.GetAddress("Beneficiary")
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := common.Big0
	usertxoptsL1 := l1info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxoptsL1,
		simpleAddr,
		callValue,
		big.NewInt(1e16),
		beneficiaryAddress,
		beneficiaryAddress,
		// send enough L2 gas for intrinsic but not compute
		big.NewInt(int64(params.TxGas+params.TxDataNonZeroGasEIP2028*4)),
		big.NewInt(int64(l2pricing.InitialBaseFeeWei)*2),
		simpleABI.Methods["incrementRedeem"].ID,
	)
	Require(t, err)
	l1Receipt, err := EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}
	waitForL1DelayBlocks(t, builder)
	submissionReceipt, err := EnsureTxSucceeded(ctx, l2client, lookupL2Tx(l1Receipt))
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

	redeemTx := func() *types.Transaction {
		arbRetryableTxAbi, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
		Require(t, err)
		redeem := arbRetryableTxAbi.Methods["redeem"]
		input, err := redeem.Inputs.Pack(ticketId)
		Require(t, err)
		data := append([]byte{}, redeem.ID...)
		data = append(data, input...)
		to := common.HexToAddress("6e")
		gas := uint64(l2pricing.InitialPerBlockGasLimitV6)
		return l2info.PrepareTxTo("Faucet", &to, gas, big.NewInt(0), data)
	}()
	burnTx := func() *types.Transaction {
		burnAmount := uint64(20 * 1e6)
		arbosTestAbi, err := precompilesgen.ArbosTestMetaData.GetAbi()
		Require(t, err)
		burnArbGas := arbosTestAbi.Methods["burnArbGas"]
		input, err := burnArbGas.Inputs.Pack(arbmath.UintToBig(burnAmount - l2info.TransferGas))
		Require(t, err)
		data := append([]byte{}, burnArbGas.ID...)
		data = append(data, input...)
		to := common.HexToAddress("0x69")
		return l2info.PrepareTxTo("Faucet", &to, burnAmount, big.NewInt(0), data)
	}()
	receipts := SendSignedTxesInBatchViaL1(t, ctx, l1info, l1client, l2client, types.Transactions{redeemTx, burnTx})
	redeemReceipt, burnReceipt := receipts[0], receipts[1]
	if len(redeemReceipt.Logs) != 1 {
		Fatal(t, "Unexpected log count:", len(redeemReceipt.Logs))
	}
	retryTxId := redeemReceipt.Logs[0].Topics[2]

	// check the receipt for the retry
	retryReceipt, err := WaitForTx(ctx, l2client, retryTxId, time.Second*1)
	Require(t, err)
	if retryReceipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "retry failed")
	}
	t.Log("submission  - block:", submissionReceipt.BlockNumber, "txInd:", submissionReceipt.TransactionIndex)
	t.Log("auto redeem - block:", autoRedeemReceipt.BlockNumber, "txInd:", autoRedeemReceipt.TransactionIndex)
	t.Log("redeem      - block:", redeemReceipt.BlockNumber, "txInd:", redeemReceipt.TransactionIndex)
	t.Log("retry       - block:", retryReceipt.BlockNumber, "txInd:", retryReceipt.TransactionIndex)
	t.Log("burn        - block:", burnReceipt.BlockNumber, "txInd:", burnReceipt.TransactionIndex)
	if !arbmath.BigEquals(burnReceipt.BlockNumber, redeemReceipt.BlockNumber) {
		Fatal(t, "Failed to fit a tx to the same block as redeem and retry")
	}
}

// elevateL2Basefee by burning gas exceeding speed limit
func elevateL2Basefee(t *testing.T, ctx context.Context, builder *NodeBuilder) {
	baseFeeBefore := builder.L2.GetBaseFee(t)
	colors.PrintBlue("Elevating base fee...")
	arbosTestAbi, err := precompilesgen.ArbosTestMetaData.GetAbi()
	Require(t, err)
	_, err = precompilesgen.NewArbosTest(common.HexToAddress("0x69"), builder.L2.Client)
	Require(t, err, "failed to deploy ArbosTest")

	burnAmount := gethexec.ConfigDefault.RPC.RPCGasCap
	burnTarget := uint64(5 * l2pricing.InitialSpeedLimitPerSecondV6 * l2pricing.InitialBacklogTolerance)
	for i := uint64(0); i < (burnTarget+burnAmount)/burnAmount; i++ {
		burnArbGas := arbosTestAbi.Methods["burnArbGas"]
		input, err := burnArbGas.Inputs.Pack(arbmath.UintToBig(burnAmount - builder.L2Info.TransferGas))
		Require(t, err)
		data := append([]byte{}, burnArbGas.ID...)
		data = append(data, input...)
		to := common.HexToAddress("0x69")
		tx := builder.L2Info.PrepareTxTo("Faucet", &to, burnAmount, big.NewInt(0), data)
		Require(t, builder.L2.Client.SendTransaction(ctx, tx))
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	baseFee := builder.L2.GetBaseFee(t)
	colors.PrintBlue("New base fee: ", baseFee, " diff:", baseFee.Uint64()-baseFeeBefore.Uint64())
}

func setupFeeAddresses(t *testing.T, ctx context.Context, builder *NodeBuilder) (common.Address, common.Address) {
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	ownerCallOpts := builder.L2Info.GetDefaultCallOpts("Owner", ctx)
	// make "Owner" a chain owner
	arbdebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
	Require(t, err, "failed to deploy ArbDebug")
	tx, err := arbdebug.BecomeChainOwner(&ownerTxOpts)
	Require(t, err, "failed to deploy ArbDebug")
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbowner, err := precompilesgen.NewArbOwner(common.HexToAddress("70"), builder.L2.Client)
	Require(t, err)
	arbownerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("6b"), builder.L2.Client)
	Require(t, err)
	builder.L2Info.GenerateAccount("InfraFee")
	builder.L2Info.GenerateAccount("NetworkFee")
	networkFeeAddr := builder.L2Info.GetAddress("NetworkFee")
	infraFeeAddr := builder.L2Info.GetAddress("InfraFee")
	tx, err = arbowner.SetNetworkFeeAccount(&ownerTxOpts, networkFeeAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	networkFeeAccount, err := arbownerPublic.GetNetworkFeeAccount(ownerCallOpts)
	Require(t, err)
	tx, err = arbowner.SetInfraFeeAccount(&ownerTxOpts, infraFeeAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	infraFeeAccount, err := arbownerPublic.GetInfraFeeAccount(ownerCallOpts)
	Require(t, err)
	t.Log("Infra fee account: ", infraFeeAccount)
	t.Log("Network fee account: ", networkFeeAccount)
	return infraFeeAddr, networkFeeAddr
}

func testFlatCallTracer(t *testing.T, ctx context.Context, client rpc.ClientInterface) {
	var blockNumber hexutil.Uint64
	err := client.CallContext(ctx, &blockNumber, "eth_blockNumber")
	Require(t, err)
	// #nosec G115
	for i := int64(1); i < int64(blockNumber); i++ {
		flatCallTracer := "flatCallTracer"
		var result interface{}
		err = client.CallContext(ctx, result, "debug_traceBlockByNumber", rpc.BlockNumber(i).String(), &tracers.TraceConfig{Tracer: &flatCallTracer})
		Require(t, err)
	}
}
