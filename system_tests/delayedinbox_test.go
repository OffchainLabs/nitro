// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var inboxABI abi.ABI

func init() {
	var err error
	inboxABI, err = abi.JSON(strings.NewReader(bridgegen.InboxABI))
	if err != nil {
		panic(err)
	}
}

func WrapL2ForDelayed(t *testing.T, l2Tx *types.Transaction, l1info *BlockchainTestInfo, delayedSender string, gas uint64) *types.Transaction {
	txbytes, err := l2Tx.MarshalBinary()
	Require(t, err)
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	delayedInboxTxData, err := inboxABI.Pack("sendL2Message", txwrapped)
	Require(t, err)
	return l1info.PrepareTx(delayedSender, "Inbox", gas, big.NewInt(0), delayedInboxTxData)
}

func testDelayInboxSimple(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// Build replica with specified execution mode
	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaParams := &SecondNodeParams{
		nodeConfig:             replicaConfig,
		useExecutionClientOnly: true,
		executionClientMode:    executionClientMode,
	}
	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, replicaParams)
	defer replicaCleanup()
	replicaClient := replicaTestClient.Client

	// Wait for replica to initialize
	time.Sleep(time.Second * 2)

	builder.L2Info.GenerateAccount("User2")

	// Prepare and send delayed transaction on primary
	delayedTx := builder.L2Info.PrepareTx("Owner", "User2", 50001, big.NewInt(1e6), nil)
	builder.L1.SendSignedTx(t, builder.L2.Client, delayedTx, builder.L1Info)

	// Get current block from primary
	block, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	// Wait for replica to catch up to primary
	for i := 0; i < 60; i++ {
		replicaBlock, err := replicaClient.BlockNumber(ctx)
		Require(t, err)
		if replicaBlock >= block {
			break
		}
		time.Sleep(time.Second)
	}

	// Verify replica caught up
	replicaBlock, err := replicaClient.BlockNumber(ctx)
	Require(t, err)
	if replicaBlock < block {
		t.Fatalf("Replica failed to sync: primary at block %d, replica at block %d", block, replicaBlock)
	}

	// Verify balance on primary
	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		Fatal(t, "Unexpected balance on primary:", l2balance)
	}

	// Verify balance on replica
	replicaBalance, err := replicaClient.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if replicaBalance.Cmp(big.NewInt(1e6)) != 0 {
		Fatal(t, "Unexpected balance on replica:", replicaBalance)
	}
}

func TestDelayInboxSimpleInternal(t *testing.T) {
	testDelayInboxSimple(t, ExecutionClientModeInternal)
}

func TestDelayInboxSimpleExternal(t *testing.T) {
	testDelayInboxSimple(t, ExecutionClientModeExternal)
}

func TestDelayInboxSimpleComparison(t *testing.T) {
	testDelayInboxSimple(t, ExecutionClientModeComparison)
}
