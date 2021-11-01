package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

func TestDelayInbox(t *testing.T) {
	background := context.Background()
	l2backend, l2info := CreateTestL2(t)
	l1backend, l1info := CreateTestL1(t, l2backend)
	l2client := ClientForArbBackend(t, l2backend)

	l2info.GenerateAccount("User2")

	delayedTx := l2info.PrepareTx("Owner", "User2", 50001, big.NewInt(1e6), nil)

	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1backend)
	if err != nil {
		t.Fatal(err)
	}
	usertxopts := l1info.GetDefaultTransactOpts("User")
	txbytes, err := delayedTx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	l1tx, err := delayedInboxContract.SendL2Message(&usertxopts, txwrapped)
	if err != nil {
		t.Fatal(err)
	}
	_, err = arbnode.EnsureTxSucceeded(l1backend, l1tx)
	if err != nil {
		t.Fatal(err)
	}

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, l1backend, []*types.Transaction{
			l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = arbnode.WaitForTx(l2client, delayedTx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	l2balance, err := l2client.BalanceAt(background, l2info.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(1e6)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
