// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
)

func TestBloom(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.RPC.BloomBitsBlocks = 256
	builder.execConfig.RPC.BloomConfirms = 1
	builder.takeOwnership = false
	cleanup := builder.Build(t)

	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	ownerTxOpts.Context = ctx
	_, simple := builder.L2.DeploySimple(t, ownerTxOpts)
	simpleABI, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)

	countsNum := 800
	eventsNum := 20
	nullEventsNum := 50

	eventCounts := make(map[uint64]struct{})
	nullEventCounts := make(map[uint64]struct{})

	for i := 0; i < eventsNum; i++ {
		// #nosec G115
		count := uint64(rand.Int() % countsNum)
		eventCounts[count] = struct{}{}
	}

	for i := 0; i < nullEventsNum; i++ {
		// #nosec G115
		count := uint64(rand.Int() % countsNum)
		nullEventCounts[count] = struct{}{}
	}

	for i := 0; i <= countsNum; i++ {
		var tx *types.Transaction
		var err error
		// #nosec G115
		_, sendNullEvent := nullEventCounts[uint64(i)]
		if sendNullEvent {
			tx, err = simple.EmitNullEvent(&ownerTxOpts)
			Require(t, err)
			_, err = builder.L2.EnsureTxSucceeded(tx)
			Require(t, err)
		}

		// #nosec G115
		_, sendEvent := eventCounts[uint64(i)]
		if sendEvent {
			tx, err = simple.IncrementEmit(&ownerTxOpts)
		} else {
			tx, err = simple.Increment(&ownerTxOpts)
		}
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		if i%100 == 0 {
			t.Log("counts: ", i, "/", countsNum)
		}
	}
	for {
		sectionSize, sectionNum := builder.L2.ExecNode.Backend.APIBackend().BloomStatus()
		if sectionSize != 256 {
			Fatal(t, "unexpected section size: ", sectionSize)
		}
		// #nosec G115
		t.Log("sections: ", sectionNum, "/", uint64(countsNum)/sectionSize)
		// #nosec G115
		if sectionSize*(sectionNum+1) > uint64(countsNum) && sectionNum > 1 {
			break
		}
		<-time.After(time.Second)
	}
	lastHeader, err := builder.L2.Client.HeaderByNumber(ctx, nil)
	Require(t, err)
	nullEventQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		ToBlock:   lastHeader.Number,
		Topics:    [][]common.Hash{{simpleABI.Events["NullEvent"].ID}},
	}
	logs, err := builder.L2.Client.FilterLogs(ctx, nullEventQuery)
	Require(t, err)
	if len(logs) != len(nullEventCounts) {
		Fatal(t, "expected ", len(nullEventCounts), " logs, got ", len(logs))
	}

	// Test that calling filters.FilterLogs is equivalent to calling ethclient's FilterLogs(ctx, filterQuery)
	allLogs := getAllLogs(t, ctx, builder.L2)
	haveLogs := filters.FilterLogs(allLogs, nil, nil, nil, [][]common.Hash{{simpleABI.Events["NullEvent"].ID}})
	var haveLogs2 []types.Log
	for _, log := range haveLogs {
		haveLogs2 = append(haveLogs2, *log)
	}
	if !reflect.DeepEqual(logs, haveLogs2) {
		t.Fatal("ethclient FilterLogs rpc call and filters.FilterLogs function call result mismatch")
	}

	incrementEventQuery := ethereum.FilterQuery{
		Topics: [][]common.Hash{{simpleABI.Events["CounterEvent"].ID}},
	}
	logs, err = builder.L2.Client.FilterLogs(ctx, incrementEventQuery)
	Require(t, err)
	if len(logs) != len(eventCounts) {
		Fatal(t, "expected ", len(eventCounts), " logs, got ", len(logs))
	}
	for _, log := range logs {
		parsedLog, err := simple.ParseCounterEvent(log)
		Require(t, err)
		_, expected := eventCounts[parsedLog.Count-1]
		if !expected {
			Fatal(t, "unxpected count in logs: ", parsedLog.Count)
		}
	}

	haveLogs = filters.FilterLogs(allLogs, nil, nil, nil, [][]common.Hash{{simpleABI.Events["CounterEvent"].ID}})
	haveLogs2 = []types.Log{}
	for _, log := range haveLogs {
		haveLogs2 = append(haveLogs2, *log)
	}
	if !reflect.DeepEqual(logs, haveLogs2) {
		t.Fatal("ethclient FilterLogs rpc call and filters.FilterLogs function call result mismatch")
	}
}

func getAllLogs(t *testing.T, ctx context.Context, l2 *TestClient) []*types.Log {
	t.Helper()

	lastBlockNum, err := l2.Client.BlockNumber(ctx)
	Require(t, err)
	var logs []*types.Log
	for i := uint64(0); i < lastBlockNum; i++ {
		// #nosec G115
		receipts, err := l2.Client.BlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(i)))
		Require(t, err)
		for _, receipt := range receipts {
			logs = append(logs, receipt.Logs...)
		}
	}
	return logs
}
