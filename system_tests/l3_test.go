// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
)

func TestSimpleL3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.MessageExtraction.Enable = true
	builder.l3Config.nodeConfig.MessageExtraction.Enable = true
	cleanupL1AndL2 := builder.Build(t)
	defer cleanupL1AndL2()

	cleanupL3FirstNode := builder.BuildL3OnL2(t)
	defer cleanupL3FirstNode()
	firstNodeTestClient := builder.L3

	secondNodeNodeConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	secondNodeNodeConfig.MessageExtraction.Enable = true
	secondNodeTestClient, cleanupL3SecondNode := builder.Build2ndNodeOnL3(t, &SecondNodeParams{nodeConfig: secondNodeNodeConfig})
	defer cleanupL3SecondNode()

	accountName := "User2"
	builder.L3Info.GenerateAccount(accountName)
	tx := builder.L3Info.PrepareTx("Owner", accountName, builder.L3Info.TransferGas, big.NewInt(1e12), nil)

	err := firstNodeTestClient.Client.SendTransaction(ctx, tx)
	Require(t, err)

	// Checks that first node has the correct balance
	_, err = firstNodeTestClient.EnsureTxSucceeded(tx)
	Require(t, err)
	l2balance, err := firstNodeTestClient.Client.BalanceAt(ctx, builder.L3Info.GetAddress(accountName), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	// Checks that second node has the correct balance
	_, err = WaitForTx(ctx, secondNodeTestClient.Client, tx.Hash(), time.Second*15)
	Require(t, err)
	l2balance, err = secondNodeTestClient.Client.BalanceAt(ctx, builder.L3Info.GetAddress(accountName), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	headState1, err := firstNodeTestClient.ConsensusNode.MessageExtractor.GetHeadState(ctx)
	Require(t, err)
	headState2, err := secondNodeTestClient.ConsensusNode.MessageExtractor.GetHeadState(ctx)
	Require(t, err)
	if headState1.MsgCount <= 1 {
		t.Fatal("Expected more than 1 message in head MEL state of first L3 node")
	}
	if headState1.Hash() != headState2.Hash() {
		t.Fatal("Head MEL states of both L3 nodes do not match")
	}
}
