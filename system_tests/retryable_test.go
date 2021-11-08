//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/util"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
)

func TestRedeemNonExistentRetryable(t *testing.T) {
	arbstate.RequireHookedGeth()
	rand.Seed(time.Now().UTC().UnixNano())

	arbRetryableAddress := common.HexToAddress("0x6e")

	backend, l2info := CreateTestL2(t)
	client := ClientForArbBackend(t, backend)
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(arbRetryableAddress, client)
	if err != nil {
		t.Fatal(err)
	}
	ownerOps := l2info.GetDefaultTransactOpts("Owner")

	ctx := context.Background()

	tx, err := arbRetryableTx.Redeem(&ownerOps, [32]byte{})
	failOnError(t, err, "Error executing redeem")

	time.Sleep(4 * time.Millisecond) // allow some time for the receipt to show up
	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	failOnError(t, err, "Error getting receipt")
	if receipt.Status != 0 {
		t.Fatal("redeem of non-existent retryable reported success")
	}
}

func TestSubmitRetryable(t *testing.T) {
	arbstate.RequireHookedGeth()
	rand.Seed(time.Now().UTC().UnixNano())

	arbRetryableAddress := common.HexToAddress("0x6e")

	backend, l2info := CreateTestL2(t)
	client := ClientForArbBackend(t, backend)

	ownerOps := l2info.GetDefaultTransactOpts("Owner")

	ctx := context.Background()

	chainId, err := client.ChainID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requestId := common.BytesToHash([]byte{13})
	retryableTx := types.ArbitrumSubmitRetryableTx{
		ChainId:     chainId,
		RequestId:   requestId,
		From:        ownerOps.From,
		GasPrice:    ownerOps.GasPrice,
		Gas:         ownerOps.GasLimit,
		To:          &arbRetryableAddress,
		Value:       util.BigZero,
		Beneficiary: ownerOps.From,
		Data:        []byte{0x81, 0xe6, 0xe0, 0x83},
	}
	tx := types.NewTx(&retryableTx)

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(4 * time.Millisecond) // allow some time for the receipt to show up
	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	failOnError(t, err, "Error getting receipt")
	if receipt.Status != 0 {
		t.Fatal("Submitted retryable tx failed")
	}

	reqId := receipt.TxHash

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(arbRetryableAddress, client)
	if err != nil {
		t.Fatal(err)
	}

	callOpts := bind.CallOpts{
		Pending: false,
		From: ownerOps.From,
		BlockNumber: nil,
		Context: ctx,
	}
	_, err = arbRetryableTx.GetTimeout(&callOpts, reqId)
	if err == nil {
		t.Fatal("unexpected success of GetTimeout for retryable that shouldn't exist")
	}
	if err.Error() != "ticketId not found" {
		t.Fatal(err)
	}

	tx, err = arbRetryableTx.Redeem(&ownerOps, reqId)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Millisecond) // allow some time for the receipt to show up
	receipt, err = client.TransactionReceipt(ctx, tx.Hash())
	failOnError(t, err, "Error getting receipt")
	if receipt.Status != 0 {
		t.Fatal("was able to redeem a retryable that should not have existed")
	}
}
