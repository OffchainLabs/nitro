// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestSequencerParallelNonces(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithDatabase(rawdb.DBPebble)
	builder.takeOwnership = false
	builder.execConfig.Sequencer.NonceFailureCacheExpiry = time.Minute
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("Destination")

	wg := sync.WaitGroup{}
	for thread := 0; thread < 10; thread++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				tx := builder.L2Info.PrepareTx("Owner", "Destination", builder.L2Info.TransferGas, common.Big1, nil)
				// Sleep a random amount of time up to 20 milliseconds
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(20)))
				t.Log("Submitting transaction with nonce", tx.Nonce())
				err := builder.L2.Client.SendTransaction(ctx, tx)
				Require(t, err)
				t.Log("Got response for transaction with nonce", tx.Nonce())
			}
		}()
	}
	wg.Wait()

	addr := builder.L2Info.GetAddress("Destination")
	balance, err := builder.L2.Client.BalanceAt(ctx, addr, nil)
	Require(t, err)
	if !arbmath.BigEquals(balance, big.NewInt(100)) {
		Fatal(t, "Unexpected user balance", balance)
	}
}

func TestSequencerNonceTooHigh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Add(1)

	before := time.Now()
	tx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, common.Big0, nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	if err == nil {
		Fatal(t, "No error when nonce was too high")
	}
	if !strings.Contains(err.Error(), core.ErrNonceTooHigh.Error()) {
		Fatal(t, "Unexpected transaction error", err)
	}
	elapsed := time.Since(before)
	if elapsed > 2*builder.execConfig.Sequencer.NonceFailureCacheExpiry {
		Fatal(t, "Sequencer took too long to respond with nonce too high")
	}
}

func TestSequencerNonceTooHighQueueFull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	builder.execConfig.Sequencer.NonceFailureCacheSize = 5
	builder.execConfig.Sequencer.NonceFailureCacheExpiry = time.Minute
	cleanup := builder.Build(t)
	defer cleanup()

	count := 15
	var completed atomic.Uint64
	for i := 0; i < count; i++ {
		builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Add(1)
		tx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, common.Big0, nil)
		go func() {
			err := builder.L2.Client.SendTransaction(ctx, tx)
			if err == nil {
				Fatal(t, "No error when nonce was too high")
			}
			completed.Add(1)
		}()
	}

	for wait := 9; wait >= 0; wait-- {
		// #nosec G115
		got := int(completed.Load())
		expected := count - builder.execConfig.Sequencer.NonceFailureCacheSize
		if got == expected {
			break
		}
		if wait == 0 || got > expected {
			Fatal(t, "Wrong number of transaction responses; got", got, "but expected", expected)
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func TestSequencerNonceHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithDatabase(rawdb.DBPebble)
	builder.execConfig.Sequencer.MaxBlockSpeed = time.Second
	builder.execConfig.Sequencer.NonceFailureCacheExpiry = 4 * time.Second
	cleanup := builder.Build(t)
	defer cleanup()

	userAccount := "User"
	builder.L2Info.GenerateAccount(userAccount)
	userAccount2 := "User2"
	builder.L2Info.GenerateAccount(userAccount2)
	userAccount3 := "User3"
	builder.L2Info.GenerateAccount(userAccount3)
	val := big.NewInt(1e18)
	builder.L2.TransferBalance(t, "Owner", "User", val, builder.L2Info)
	builder.L2.TransferBalance(t, "Owner", "User2", val, builder.L2Info)
	userBal, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(userAccount), nil)
	Require(t, err)
	if userBal.Cmp(val) != 0 {
		t.Fatal("balance mismatch")
	}
	userBal2, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(userAccount2), nil)
	Require(t, err)
	if userBal2.Cmp(val) != 0 {
		t.Fatal("balance mismatch")
	}

	size := 40000
	data := make([]byte, size)
	_, err = rand.Read(data)
	Require(t, err)

	var largeTxs types.Transactions
	builder.L2Info.GetInfoWithPrivKey(userAccount).Nonce.Store(0)
	largeTxs = append(largeTxs, builder.L2Info.PrepareTx(userAccount, userAccount3, 70000000, big.NewInt(1e8), data))
	builder.L2Info.GetInfoWithPrivKey(userAccount).Nonce.Store(0)
	largeTxs = append(largeTxs, builder.L2Info.PrepareTx(userAccount, userAccount3, 70000000, big.NewInt(1e9), data))
	builder.L2Info.GetInfoWithPrivKey(userAccount).Nonce.Store(0)
	txFirst := builder.L2Info.PrepareTx(userAccount, userAccount3, 7000000, big.NewInt(1e8), data)
	builder.L2Info.GetInfoWithPrivKey(userAccount).Nonce.Store(1)
	largeTxs = append(largeTxs, builder.L2Info.PrepareTx(userAccount, userAccount3, 35000000, big.NewInt(1e5), data))
	builder.L2Info.GetInfoWithPrivKey(userAccount).Nonce.Store(1)
	largeTxs = append(largeTxs, builder.L2Info.PrepareTx(userAccount, userAccount3, 35000000, big.NewInt(1e5+1), data))
	builder.L2Info.GetInfoWithPrivKey(userAccount).Nonce.Store(1)
	largeTxs = append(largeTxs, builder.L2Info.PrepareTx(userAccount, userAccount3, 35000000, big.NewInt(1e5+2), data))

	var allTxs types.Transactions
	allTxs = append(allTxs, txFirst)
	allTxs = append(allTxs, largeTxs...)
	for i := 0; i < 5; i++ {
		allTxs = append(allTxs, builder.L2Info.PrepareTx("Owner", userAccount3, 7000000, big.NewInt(1e8), nil))
	}
	var wg sync.WaitGroup
	wg.Add(len(allTxs))
	for i, tx := range allTxs {
		go func(w *sync.WaitGroup, txParallel *types.Transaction) {
			time.Sleep(time.Duration(i * 10 * int(time.Millisecond)))
			_ = builder.L2.Client.SendTransaction(ctx, txParallel)
			w.Done()
		}(&wg, tx)
	}
	wg.Wait()
	var blockNumsOfAcceptedTxs []uint64
	for _, tx := range allTxs {
		receipt, err := builder.L2.Client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			blockNumsOfAcceptedTxs = append(blockNumsOfAcceptedTxs, receipt.BlockNumber.Uint64())
		}
	}
	if len(blockNumsOfAcceptedTxs) != 7 {
		t.Fatalf("unexpected number of block nums in blockNumsOfAcceptedTxs. Have: %d, Want: 7", len(blockNumsOfAcceptedTxs))
	}
	if blockNumsOfAcceptedTxs[0] != blockNumsOfAcceptedTxs[1] {
		t.Fatal("first and second valid txs shouldnt have been sequenced in two different blocks")
	}
	if blockNumsOfAcceptedTxs[2] != blockNumsOfAcceptedTxs[0] ||
		blockNumsOfAcceptedTxs[3] != blockNumsOfAcceptedTxs[0] ||
		blockNumsOfAcceptedTxs[4] != blockNumsOfAcceptedTxs[0] ||
		blockNumsOfAcceptedTxs[5] != blockNumsOfAcceptedTxs[0] ||
		blockNumsOfAcceptedTxs[6] != blockNumsOfAcceptedTxs[0] {
		t.Fatal("all the following valid txs should have been sequenced in the same block")
	}
}
