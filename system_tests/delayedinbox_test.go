// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

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

func TestDelayInboxSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	delayedTx := builder.L2Info.PrepareTx("Owner", "User2", 50001, big.NewInt(1e6), nil)
	builder.L1.SendSignedTx(t, builder.L2.Client, delayedTx, builder.L1Info)

	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		Fatal(t, "Unexpected balance:", l2balance)
	}
}

func TestDelayedMessagesBatchSequencedInOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("Sender")
	builder.L2.TransferBalance(t, "Owner", "Sender", big.NewInt(1e18), builder.L2Info)

	const numMsgs = 5
	transferEach := big.NewInt(1e12)
	recipients := make([]string, numMsgs)
	txHashes := make([]common.Hash, numMsgs)
	for i := 0; i < numMsgs; i++ {
		name := fmt.Sprintf("DelayedRecipient%d", i)
		recipients[i] = name
		builder.L2Info.GenerateAccount(name)
		tx := builder.L2Info.PrepareTx("Sender", name, builder.L2Info.TransferGas, transferEach, nil)
		txHashes[i], _ = sendDelayedTx(t, ctx, builder, tx)
	}

	// A single L1 advance enqueues all pending delayed messages together as a batch.
	advanceL1ForDelayed(t, ctx, builder)

	// All delayed messages should be sequenced, each in its own block, in submission
	// order (later messages land in strictly higher block numbers).
	prevBlock := uint64(0)
	for i := 0; i < numMsgs; i++ {
		receipt, err := WaitForTx(ctx, builder.L2.Client, txHashes[i], 15*time.Second)
		require.NoError(t, err, "delayed message %d should be sequenced", i)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		curBlock := receipt.BlockNumber.Uint64()
		require.Greater(t, curBlock, prevBlock,
			"delayed messages should be sequenced in submission order, each in its own block")
		prevBlock = curBlock

		bal, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(recipients[i]), nil)
		require.NoError(t, err)
		require.Equal(t, transferEach, bal, "recipient %d should receive the delayed transfer", i)
	}
}
