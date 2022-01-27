//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	arbos_util "github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
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

	usertxopts := l1info.GetDefaultTransactOpts("Faucet")
	usertxopts.Value = util.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	user2Address := l2info.GetAddress("User2")
	beneficiaryAddress := l2info.GetAddress("Beneficiary")

	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxopts,
		user2Address,
		big.NewInt(1e6),
		big.NewInt(1e6),
		beneficiaryAddress,
		beneficiaryAddress,
		big.NewInt(50001),
		big.NewInt(params.InitialBaseFee*2),
		[]byte{},
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

	usertxopts := l1info.GetDefaultTransactOpts("Faucet")
	usertxopts.Value = util.BigMul(big.NewInt(1e12), big.NewInt(1e12))

	user2Address := l2info.GetAddress("User2")
	beneficiaryAddress := l2info.GetAddress("Beneficiary")

	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxopts,
		user2Address,
		big.NewInt(1e6),
		big.NewInt(1e6),
		beneficiaryAddress,
		beneficiaryAddress,
		big.NewInt(int64(params.TxGas)+1), // send inadequate L2 gas
		big.NewInt(params.InitialBaseFee*2),
		[]byte{0x00},
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

	// send tx to redeem the retryable
	arbRetryableTxAbi, err := precompilesgen.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)

	arbRetryableAddress := common.BigToAddress(big.NewInt(0x6e))
	txData := &types.DynamicFeeTx{
		To:        &arbRetryableAddress,
		Gas:       10000001,
		GasFeeCap: big.NewInt(params.InitialBaseFee * 2),
		Value:     big.NewInt(0),
		Nonce:     0,
		Data:      append(arbRetryableTxAbi.Methods["redeem"].ID, ticketId.Bytes()...),
	}
	tx := l2info.SignTxAs("Owner", txData)
	txbytes, err := tx.MarshalBinary()
	Require(t, err)

	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	usertxopts = l1info.GetDefaultTransactOpts("Faucet")
	l1tx, err = delayedInbox.SendL2Message(&usertxopts, txwrapped)
	Require(t, err)

	_, err = arbnode.EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)

	// wait for redeem transaction to complete successfully
	waitForL1DelayBlocks(t, ctx, l1client, l1info)
	receipt, err = arbnode.WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fail(t, *receipt)
	}
	retryTxId := receipt.Logs[0].Topics[2]

	// verify that balance transfer happened, so we know the retry succeeded
	l2balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if !util.BigEquals(l2balance, big.NewInt(1e6)) {
		Fail(t, "Unexpected balance:", l2balance)
	}

	// check the receipt for the retry
	receipt, err = arbnode.WaitForTx(ctx, l2client, retryTxId, time.Second*1)
	Require(t, err)
	if receipt.Status != 1 {
		Fail(t)
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
