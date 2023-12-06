// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestLogSubscription(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)

	logChan := make(chan types.Log, 128)
	subscription, err := builder.L2.Client.SubscribeFilterLogs(ctx, ethereum.FilterQuery{}, logChan)
	Require(t, err)
	defer subscription.Unsubscribe()

	tx, err := arbSys.WithdrawEth(&auth, common.Address{})
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	if len(receipt.Logs) != 1 {
		Fatal(t, "Unexpected number of logs", len(receipt.Logs))
	}

	var receiptLog types.Log = *receipt.Logs[0]
	var subscriptionLog types.Log
	timer := time.NewTimer(time.Second * 5)
	defer timer.Stop()
	select {
	case <-timer.C:
		Fatal(t, "Hit timeout waiting for log from subscription")
	case subscriptionLog = <-logChan:
	}
	if !reflect.DeepEqual(receiptLog, subscriptionLog) {
		Fatal(t, "Receipt log", receiptLog, "is different than subscription log", subscriptionLog)
	}
	_, err = builder.L2.Client.BlockByHash(ctx, subscriptionLog.BlockHash)
	Require(t, err)
}
