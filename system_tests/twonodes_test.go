//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/arbstate/arbnode"
)

func Create2ndNode(t *testing.T, ctx context.Context, first *arbnode.Node, l1stack *node.Node) (*ethclient.Client, *arbnode.Node) {
	l1rpcClient, err := l1stack.Attach()
	if err != nil {
		t.Fatal(err)
	}
	l1client := ethclient.NewClient(l1rpcClient)
	l2stack, err := arbnode.CreateDefaultStack()
	if err != nil {
		t.Fatal(err)
	}
	l2chainDb, l2blockchain, err := arbnode.CreateDefaultBlockChain(l2stack, l2Genesys)
	if err != nil {
		t.Fatal(err)
	}
	nodeConf := arbnode.NodeConfigL1Test
	nodeConf.BatchPoster = false
	node, err := arbnode.CreateNode(l2stack, l2chainDb, &nodeConf, l2blockchain, l1client, first.DeployInfo, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = node.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	l2client := ClientForArbBackend(t, node.Backend)
	return l2client, node
}

func TestTwoNodesSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, node1, l1info, _, l1stack := CreateTestNodeOnL1(t, ctx, true)
	defer l1stack.Close()
	defer node1.Stop()

	l2clientB, node2 := Create2ndNode(t, ctx, node1, l1stack)
	defer node2.Stop()

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", 30000, big.NewInt(1e12), nil)

	err := l2info.Client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbnode.EnsureTxSucceeded(ctx, l2info.Client, tx)
	if err != nil {
		t.Fatal(err)
	}

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1info.Client, []*types.Transaction{
			l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = arbnode.WaitForTx(ctx, l2clientB, tx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
