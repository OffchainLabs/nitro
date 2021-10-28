package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

func TestDelayInbox(t *testing.T) {
	background := context.Background()
	l1backend, l1info := CreateTestL1(t)
	_, l2info := CreateTestL2(t)

	delayedBridge, err := arbnode.NewDelayedBridge(l1backend, l1info.GetAddress("Bridge"), 0)
	if err != nil {
		t.Fatal(err)
	}
	sequencerInbox, err := arbnode.NewSequencerInbox(l1backend, l1info.GetAddress("SequencerInbox"), 0)
	if err != nil {
		t.Fatal(err)
	}

	inboxDB := rawdb.NewMemoryDatabase()
	inboxReaderConfig := &arbnode.InboxReaderConfig{
		DelayBlocks: 0,
		CheckDelay:  time.Millisecond * 100,
	}
	inboxReader, err := arbnode.NewInboxReader(inboxDB, l1backend, big.NewInt(0), delayedBridge, sequencerInbox, inboxReaderConfig)
	if err != nil {
		t.Fatal(err)
	}

	inboxReader.Start(background)
	readerDB, err := arbnode.NewInboxReaderDb(inboxDB)
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

	l1backend.Commit()
	msgs, err := delayedBridge.GetMessageCount(background, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msgs != 0 {
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
	if msgs != 1 {
		t.Fatal("Unexpected message count before: ", msgs)
	}

	correctDelayedCount := func() bool {
		for i := 0; i < 5; i++ {
			readCount, err := readerDB.GetDelayedCount()
			if err != nil {
				t.Fatal(err)
			}
			if readCount == 1 {
				return true
			}
			time.Sleep(500 * time.Millisecond)
		}
		return false
	}()
	if !correctDelayedCount {
		t.Fatal("incorrect delayed count")
	}
}
