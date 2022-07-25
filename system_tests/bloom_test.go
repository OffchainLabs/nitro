// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func TestBloom(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeconfig := arbnode.ConfigDefaultL2Test()
	nodeconfig.RPC.BloomBitsBlocks = 256
	nodeconfig.RPC.BloomConfirms = 1
	l2info, node, client, stack := CreateTestL2WithConfig(t, ctx, nil, nodeconfig, false)
	defer requireClose(t, stack)

	l2info.GenerateAccount("User2")

	ownerTxOpts := l2info.GetDefaultTransactOpts("Owner", ctx)
	ownerTxOpts.Context = ctx
	_, tx, simple, err := mocksgen.DeploySimple(&ownerTxOpts, client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)

	countsNum := 800
	eventsNum := 20
	nullEventsNum := 50

	eventCounts := make(map[uint64]struct{})
	nullEventCounts := make(map[uint64]struct{})

	for i := 0; i < eventsNum; i++ {
		count := uint64(rand.Int() % countsNum)
		eventCounts[count] = struct{}{}
	}

	for i := 0; i < nullEventsNum; i++ {
		count := uint64(rand.Int() % countsNum)
		nullEventCounts[count] = struct{}{}
	}

	for i := 0; i <= countsNum; i++ {
		var tx *types.Transaction
		var err error
		_, sendNullEvent := nullEventCounts[uint64(i)]
		if sendNullEvent {
			tx, err = simple.EmitNullEvent(&ownerTxOpts)
			Require(t, err)
			_, err = EnsureTxSucceeded(ctx, client, tx)
			Require(t, err)
		}

		_, sendEvent := eventCounts[uint64(i)]
		if sendEvent {
			tx, err = simple.IncrementEmit(&ownerTxOpts)
		} else {
			tx, err = simple.Increment(&ownerTxOpts)
		}
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if i%100 == 0 {
			t.Log("counts: ", i, "/", countsNum)
		}
	}

	for {
		sectionSize, sectionNum := node.Backend.APIBackend().BloomStatus()
		if sectionSize != 256 {
			Fail(t, "unexpected section size: ", sectionSize)
		}
		t.Log("sections: ", sectionNum, "/", uint64(countsNum)/sectionSize)
		if sectionSize*(sectionNum+1) > uint64(countsNum) && sectionNum > 1 {
			break
		}
		<-time.After(time.Second)
	}
	lastHeader, err := client.HeaderByNumber(ctx, nil)
	Require(t, err)
	nullEventQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		ToBlock:   lastHeader.Number,
		Topics:    [][]common.Hash{{simpleABI.Events["NullEvent"].ID}},
	}
	logs, err := client.FilterLogs(ctx, nullEventQuery)
	Require(t, err)
	if len(logs) != len(nullEventCounts) {
		Fail(t, "expected ", len(nullEventCounts), " logs, got ", len(logs))
	}
	incrementEventQuery := ethereum.FilterQuery{
		Topics: [][]common.Hash{{simpleABI.Events["CounterEvent"].ID}},
	}
	logs, err = client.FilterLogs(ctx, incrementEventQuery)
	Require(t, err)
	if len(logs) != len(eventCounts) {
		Fail(t, "expected ", len(eventCounts), " logs, got ", len(logs))
	}
	for _, log := range logs {
		parsedLog, err := simple.ParseCounterEvent(log)
		Require(t, err)
		_, expected := eventCounts[parsedLog.Count-1]
		if !expected {
			Fail(t, "unxpected count in logs: ", parsedLog.Count)
		}
	}
}
