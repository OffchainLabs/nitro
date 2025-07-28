package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/arbnode"
)

func TestEspressoBatcherMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := createL1AndL2Node(ctx, t, true, false)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	seqInboxAddr := builder.addresses.SequencerInbox

	monitor := arbnode.NewBatcherAddrMonitor(
		[]common.Address{},
		rawdb.NewMemoryDatabase(),
		builder.L2.ConsensusNode.L1Reader,
		seqInboxAddr,
		builder.L2.ConsensusNode.DeployInfo.DeployedAt,
		builder.L2.ConsensusNode.DeployInfo.DeployedAt,
	)
	err = monitor.Start(ctx)
	Require(t, err)

	abi, err := bridgegen.SequencerInboxMetaData.GetAbi()
	Require(t, err)
	batchPosterAddr := builder.L2Info.GetAddress("Faucet")
	data, err := abi.Pack("setIsBatchPoster", batchPosterAddr, true)
	Require(t, err)
	tx := builder.L1Info.PrepareTxTo("RollupOwner", &seqInboxAddr, 100000, big.NewInt(0), data)
	err = builder.L1.Client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err := EnsureTxSucceededWithTimeout(ctx, builder.L1.Client, tx, time.Second*10)
	Require(t, err)
	log.Info("tx receipt", "receipt", receipt.BlockNumber)

	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	time.Sleep(time.Second * 5)

	events := monitor.GetEvents()
	if len(events) != 1 {
		t.Fatal("expected 1 valid address, got", events)
	}
	if events[0].Addr != batchPosterAddr {
		t.Fatal("expected valid address to be", batchPosterAddr, "got", events[0].Addr)
	}
	if events[0].IsBatcher != true {
		t.Fatal("expected valid address to be batcher, got", events[0].IsBatcher)
	}

	newAddr := common.Address{}
	data2, err := abi.Pack("setIsBatchPoster", newAddr, false)
	Require(t, err)
	tx2 := builder.L1Info.PrepareTxTo("RollupOwner", &seqInboxAddr, 100000, big.NewInt(0), data2)
	err = builder.L1.Client.SendTransaction(ctx, tx2)
	Require(t, err)
	_, err = EnsureTxSucceededWithTimeout(ctx, builder.L1.Client, tx2, time.Second*10)
	Require(t, err)
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	time.Sleep(time.Second * 5)

	events2 := monitor.GetEvents()
	if len(events2) != 2 {
		t.Fatal("expected 2 valid addresses, got", events2)
	}
	if events2[1].Addr != newAddr {
		t.Fatal("expected valid address to be", newAddr, "got", events2[1].Addr)
	}
	if events2[1].IsBatcher != false {
		t.Fatal("expected valid address to not be batcher, got", events2[1].IsBatcher)
	}

}
