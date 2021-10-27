package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

func TestDelayInbox(t *testing.T) {
	_, l2info, l1backend, l1info := CreateTestBackendWithBalance(t)

	delayedBridge, err := arbnode.NewDelayedBridge(l1backend, l1info.GetAddress("Bridge"), 0)
	if err != nil {
		t.Fatal(err)
	}

	l2info.GenerateAccount("User2")

	accesses := types.AccessList{types.AccessTuple{
		Address:     l2info.GetAddress("User2"),
		StorageKeys: []common.Hash{{0}},
	}}

	l2addr := l2info.GetAddress("User2")
	txdata := &types.DynamicFeeTx{
		ChainID:    arbos.ChainConfig.ChainID,
		Nonce:      0,
		To:         &l2addr,
		Gas:        30000,
		GasFeeCap:  big.NewInt(5e+09),
		GasTipCap:  big.NewInt(2),
		Value:      big.NewInt(1e12),
		AccessList: accesses,
		Data:       []byte{},
	}
	tx := l2info.SignTxAs("Owner", txdata)

	background := context.Background()
	l1backend.Commit()
	msgs, err := delayedBridge.GetMessageCount(background, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msgs.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Unexpected message count before: ", msgs)
	}

	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1backend)
	if err != nil {
		t.Fatal(err)
	}
	usertxopts := l1info.GetDefaultTransactOpts("User")
	txbytes, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	_, err = delayedInboxContract.SendL2Message(&usertxopts, txbytes)
	if err != nil {
		t.Fatal(err)
	}
	l1backend.Commit()
	msgs, err = delayedBridge.GetMessageCount(background, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msgs.Cmp(big.NewInt(1)) != 0 {
		t.Fatal("Unexpected message count before: ", msgs)
	}

}
