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

	l2info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, client)
	Require(t, err)

	logChan := make(chan types.Log, 128)
	subscription, err := client.SubscribeFilterLogs(ctx, ethereum.FilterQuery{}, logChan)
	Require(t, err)
	defer subscription.Unsubscribe()

	tx, err := arbSys.WithdrawEth(&auth, common.Address{})
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	if len(receipt.Logs) != 1 {
		Fail(t, "Unexpected number of logs", len(receipt.Logs))
	}

	var receiptLog types.Log = *receipt.Logs[0]
	var subscriptionLog types.Log
	timer := time.NewTimer(time.Second * 5)
	defer timer.Stop()
	select {
	case <-timer.C:
		Fail(t, "Hit timeout waiting for log from subscription")
	case subscriptionLog = <-logChan:
	}
	if !reflect.DeepEqual(receiptLog, subscriptionLog) {
		Fail(t, "Receipt log", receiptLog, "is different than subscription log", subscriptionLog)
	}
	_, err = client.BlockByHash(ctx, subscriptionLog.BlockHash)
	Require(t, err)
}
