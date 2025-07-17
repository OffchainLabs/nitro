// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMaintenance(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.nodeConfig.Maintenance.RunInterval = time.Second * 5
	builder.nodeConfig.Maintenance.Enable = true
	cleanup := builder.Build(t)
	defer cleanup()

	numberOfTransfers := 10
	for i := 2; i < 3+numberOfTransfers; i++ {
		account := fmt.Sprintf("User%d", i)
		builder.L2Info.GenerateAccount(account)

		tx := builder.L2Info.PrepareTx("Owner", account, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	finished := false
	for range 200 {
		if logHandler.WasLogged("Execution is not running maintenance anymore, maintenance completed successfully") {
			finished = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !finished {
		t.Fatal("Maintenance did not complete successfully from Consensus perspective")
	}

	if !logHandler.WasLogged("Flushed trie db through maintenance completed successfully") {
		t.Fatal("Expected log message not found")
	}

	// checks that balances are correct after maintenance
	for i := 2; i < 3+numberOfTransfers; i++ {
		account := fmt.Sprintf("User%d", i)
		balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(account), nil)
		Require(t, err)
		if balance.Cmp(big.NewInt(int64(1e12))) != 0 {
			t.Fatal("Unexpected balance:", balance, "for account:", account)
		}
	}
}
