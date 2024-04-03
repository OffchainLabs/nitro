// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestSequencerParallelNonces(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
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
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce++

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
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	builder.execConfig.Sequencer.NonceFailureCacheSize = 5
	builder.execConfig.Sequencer.NonceFailureCacheExpiry = time.Minute
	cleanup := builder.Build(t)
	defer cleanup()

	count := 15
	var completed uint64
	for i := 0; i < count; i++ {
		builder.L2Info.GetInfoWithPrivKey("Owner").Nonce++
		tx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, common.Big0, nil)
		go func() {
			err := builder.L2.Client.SendTransaction(ctx, tx)
			if err == nil {
				Fatal(t, "No error when nonce was too high")
			}
			atomic.AddUint64(&completed, 1)
		}()
	}

	for wait := 9; wait >= 0; wait-- {
		got := int(atomic.LoadUint64(&completed))
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
