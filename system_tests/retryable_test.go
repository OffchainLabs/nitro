//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

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
	"github.com/offchainlabs/arbstate/arbnode"
	arbos_util "github.com/offchainlabs/arbstate/arbos/util"

	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/solgen/go/mocksgen"
	"github.com/offchainlabs/arbstate/solgen/go/node_interfacegen"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
	"github.com/offchainlabs/arbstate/util"
	"github.com/offchainlabs/arbstate/util/colors"
)

func retryableSetup(t *testing.T) (
	*BlockchainTestInfo,
	*BlockchainTestInfo,
	*ethclient.Client,
	*ethclient.Client,
	*bridgegen.Inbox,
	*bridgegen.InboxFilterer,
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
	inboxFilterer, err := bridgegen.NewInboxFilterer(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)

	// burn some gas so that the faucet's Callvalue + Balance never exceeds a uint256
	discard := util.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	TransferBalance(t, "Faucet", "Burn", discard, l2info, l2client, ctx)

	teardown := func() {
		cancel()
		stack.Close()
	}
	return l2info, l1info, l2client, l1client, delayedInbox, inboxFilterer, ctx, teardown
}

func TestSubmitRetryableImmediateSuccess(t *testing.T) {
	l2info, l1info, l2client, l1client, delayedInbox, inboxFilterer, ctx, teardown := retryableSetup(t)
	defer teardown()

	user2Address := l2info.GetAddress("User2")
	beneficiaryAddress := l2info.GetAddress("Beneficiary")

	deposit := util.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(common.HexToAddress("0xc8"), l2client)
	Require(t, err, "failed to deploy NodeInterface")
	_ = nodeInterface

	// estimate the gas needed to auto-redeem the retryable
	usertxoptsL2 := l2info.GetDefaultTransactOpts("Faucet")
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
	usertxoptsL1 := l1info.GetDefaultTransactOpts("Faucet")
	usertxoptsL1.Value = deposit
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxoptsL1,
		user2Address,
		callValue,
		big.NewInt(1e6),
		beneficiaryAddress,
		beneficiaryAddress,
		util.UintToBig(estimate),
		big.NewInt(params.InitialBaseFee*2),
		[]byte{0x32, 0x42, 0x32, 0x88},
	)
	Require(t, err)

	l1receipt, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	var l2TxId *common.Hash
	for _, log := range l1receipt.Logs {
		msg, _ := inboxFilterer.ParseInboxMessageDelivered(*log)
		if msg != nil {
			id := common.BigToHash(msg.MessageNum)
			l2TxId = &id
		}
	}
	if l2TxId == nil {
		Fail(t)
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := arbnode.WaitForTx(ctx, l2client, *l2TxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t)
	}

	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if !util.BigEquals(l2balance, big.NewInt(1e6)) {
		Fail(t, "Unexpected balance:", l2balance)
	}
}

func TestSubmitRetryableFailThenRetry(t *testing.T) {
	l2info, l1info, l2client, l1client, delayedInbox, inboxFilterer, ctx, teardown := retryableSetup(t)
	defer teardown()

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner")
	usertxopts := l1info.GetDefaultTransactOpts("Faucet")
	usertxopts.Value = util.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	simpleAddr, _, simple, err := mocksgen.DeploySimple(&ownerTxOpts, l2client)
	Require(t, err)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)

	beneficiaryAddress := l2info.GetAddress("Beneficiary")
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxopts,
		simpleAddr,
		common.Big0,
		big.NewInt(1e6),
		beneficiaryAddress,
		beneficiaryAddress,
		// send enough L2 gas for intrinsic but not compute
		big.NewInt(int64(params.TxGas+params.TxDataNonZeroGasEIP2028*4)),
		big.NewInt(params.InitialBaseFee*2),
		simpleABI.Methods["increment"].ID,
	)
	Require(t, err)

	l1receipt, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	var l2TxId *common.Hash
	for _, log := range l1receipt.Logs {
		msg, _ := inboxFilterer.ParseInboxMessageDelivered(*log)
		if msg != nil {
			id := common.BigToHash(msg.MessageNum)
			l2TxId = &id
		}
	}
	if l2TxId == nil {
		Fail(t)
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)

	receipt, err := arbnode.WaitForTx(ctx, l2client, *l2TxId, time.Second*5)
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
	receipt, err = arbnode.WaitForTx(ctx, l2client, firstRetryTxId, time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fail(t, receipt.GasUsed)
	}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), l2client)
	Require(t, err)
	tx, err := arbRetryableTx.Redeem(&ownerTxOpts, ticketId)
	Require(t, err)
	receipt, err = arbnode.EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	retryTxId := receipt.Logs[0].Topics[2]

	// check the receipt for the retry
	receipt, err = arbnode.WaitForTx(ctx, l2client, retryTxId, time.Second*1)
	Require(t, err)
	if receipt.Status != 1 {
		Fail(t)
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

	usertxopts := l1info.GetDefaultTransactOpts("Faucet")
	usertxopts.Value = util.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	l2info.GenerateAccount("Recieve")
	faucetAddress := arbos_util.RemapL1Address(l1info.GetAddress("Faucet"))
	beneficiaryAddress := l2info.GetAddress("Beneficiary")
	receiveAddress := l2info.GetAddress("Recieve")

	colors.PrintBlue("Faucet      ", faucetAddress)
	colors.PrintBlue("Receive     ", receiveAddress)
	colors.PrintBlue("Beneficiary ", beneficiaryAddress)

	fundsBeforeSubmit, err := l2client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)

	usefulGas := params.TxGas
	excessGas := uint64(808)

	retryableGas := new(big.Int).SetUint64(usefulGas + excessGas) // will only burn the intrinsic cost
	retryableL2CallValue := big.NewInt(1e4)
	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxopts,
		receiveAddress,
		retryableL2CallValue,
		big.NewInt(1e6),
		beneficiaryAddress,
		beneficiaryAddress,
		retryableGas,
		big.NewInt(params.InitialBaseFee*2),
		[]byte{},
	)
	Require(t, err)

	l1receipt, err := arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)
	if l1receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, "l1receipt indicated failure")
	}

	waitForL1DelayBlocks(t, ctx, l1client, l1info)
	l2GasPrice := GetBaseFee(t, l2client, ctx)
	excessWei := util.BigMulByUint(l2GasPrice, excessGas)

	fundsAfterSubmit, err := l2client.BalanceAt(ctx, faucetAddress, nil)
	Require(t, err)
	beneficiaryFunds, err := l2client.BalanceAt(ctx, beneficiaryAddress, nil)
	Require(t, err)
	receiveFunds, err := l2client.BalanceAt(ctx, receiveAddress, nil)
	Require(t, err)

	colors.PrintMint("Faucet before ", fundsBeforeSubmit)
	colors.PrintMint("Faucet after  ", fundsAfterSubmit)

	// the retryable should pay the receiver the supplied callvalue
	colors.PrintMint("Receive       ", receiveFunds)
	colors.PrintBlue("L2 Call Value ", retryableL2CallValue)
	if !util.BigEquals(receiveFunds, retryableL2CallValue) {
		Fail(t, "Recipient didn't receive the right funds")
	}

	// the beneficiary should recieve the excess gas
	colors.PrintBlue("Base Fee      ", l2GasPrice)
	colors.PrintBlue("Excess Gas    ", excessGas)
	colors.PrintBlue("Excess Wei    ", excessWei)
	colors.PrintMint("Beneficiary   ", beneficiaryFunds)
	if !util.BigEquals(beneficiaryFunds, excessWei) {
		Fail(t, "Beneficiary didn't receive the right funds")
	}

	// the faucet must pay for both the gas used and the call value supplied
	expectedGasChange := util.BigMul(l2GasPrice, retryableGas)
	expectedGasChange = util.BigSub(expectedGasChange, usertxopts.Value) // the user is credited this
	expectedGasChange = util.BigAdd(expectedGasChange, retryableL2CallValue)

	colors.PrintBlue("CallGas    ", retryableGas)
	colors.PrintMint("Gas cost   ", util.BigMul(retryableGas, l2GasPrice))
	colors.PrintBlue("Payment    ", usertxopts.Value)

	if !util.BigEquals(fundsBeforeSubmit, util.BigAdd(fundsAfterSubmit, expectedGasChange)) {
		diff := util.BigSub(fundsBeforeSubmit, fundsAfterSubmit)
		colors.PrintRed("Expected ", expectedGasChange)
		colors.PrintRed("Observed ", diff)
		colors.PrintRed("Off by   ", util.BigSub(expectedGasChange, diff))
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
