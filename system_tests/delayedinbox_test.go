// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
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
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testNode := NewNodeBuilder(ctx).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNode.L1Stack)
	defer testNode.L2Node.StopAndWait()

	testNode.L2Info.GenerateAccount("User2")

	delayedTx := testNode.L2Info.PrepareTx("Owner", "User2", 50001, big.NewInt(1e6), nil)
	testNode.SendSignedTxViaL1(t, delayedTx)

	l2balance, err := testNode.L2Client.BalanceAt(ctx, testNode.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		Fatal(t, "Unexpected balance:", l2balance)
	}
}
