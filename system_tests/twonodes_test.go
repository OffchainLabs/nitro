//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
)

func testTwoNodesSimple(t *testing.T, dasMode das.DataAvailabilityMode) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l1NodeConfigA := arbnode.NodeConfigL1Test
	l1NodeConfigA.DataAvailabilityMode = dasMode
	var dbPath string
	var err error
	defer os.RemoveAll(dbPath)
	if dasMode == das.LocalDataAvailability {
		dbPath, err = ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		l1NodeConfigA.DataAvailabilityConfig.LocalDiskDataDir = dbPath
	}
	l2info, nodeA, l2clientA, l1info, _, l1client, l1stack := CreateTestNodeOnL1WithConfig(t, ctx, true, &l1NodeConfigA)
	defer l1stack.Close()

	l1NodeConfigB := arbnode.NodeConfigL1Test
	l1NodeConfigB.BatchPoster = false
	l1NodeConfigB.BlockValidator = false
	l1NodeConfigB.DataAvailabilityMode = dasMode
	l1NodeConfigB.DataAvailabilityConfig.LocalDiskDataDir = dbPath
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, &l1NodeConfigB)

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)

	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = arbutil.EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = arbutil.WaitForTx(ctx, l2clientB, tx.Hash(), time.Second*5)
	Require(t, err)

	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}

	nodeA.StopAndWait()
	nodeB.StopAndWait()
}

func TestTwoNodesSimple(t *testing.T) {
	testTwoNodesSimple(t, das.OnchainDataAvailability)
}

func TestTwoNodesSimpleLocalDAS(t *testing.T) {
	testTwoNodesSimple(t, das.LocalDataAvailability)
}
