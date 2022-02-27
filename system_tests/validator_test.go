//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/das"
)

func testValidatorSimple(t *testing.T, dasMode arbnode.DataAvailabilityMode) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l1NodeConfigA := arbnode.NodeConfigL1Test
	l1NodeConfigA.DataAvailabilityMode = dasMode
	l2info, node1, l2client, l1info, _, l1client, l1stack := CreateTestNodeOnL1WithConfig(t, ctx, true, &l1NodeConfigA)
	defer l1stack.Close()

	l1NodeConfigB := arbnode.NodeConfigL1Test
	l1NodeConfigB.BatchPoster = false
	l1NodeConfigB.BlockValidator = true
	l1NodeConfigB.DataAvailabilityMode = dasMode
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, node1, l1stack, &l2info.ArbInitData, &l1NodeConfigB)

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)

	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = arbnode.EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
		WrapL2ForDelayed(t, l2info.PrepareTx("Owner", "User2", 30002, big.NewInt(1e12), nil), l1info, "User", 100000),
	})

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 500)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	// this is needed to stop the 1000000 balance error in CI (BUG)
	time.Sleep(time.Millisecond * 500)

	_, err = arbnode.WaitForTx(ctx, l2clientB, tx.Hash(), time.Second*5)
	Require(t, err)

	// BUG: need to sleep to avoid (Unexpected balance: 1000000000000)
	time.Sleep(time.Millisecond * 100)

	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(2e12)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}
	lastBlockHeader, err := l2clientB.HeaderByNumber(ctx, nil)
	Require(t, err)
	testDeadLine, _ := t.Deadline()
	if !nodeB.BlockValidator.WaitForBlock(lastBlockHeader.Number.Uint64(), time.Until(testDeadLine)-time.Second*10) {
		Fail(t, "did not validate all blocks")
	}

}

func TestValidatorSimple(t *testing.T) {
	testValidatorSimple(t, arbnode.OnchainDataAvailability)
}

func TestValidatorSimpleLocalDAS(t *testing.T) {
	defer das.CleanupSingletonTestingDAS()
	testValidatorSimple(t, arbnode.LocalDataAvailability)
}
