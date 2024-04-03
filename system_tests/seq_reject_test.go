// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

func TestSequencerRejection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feedErrChan := make(chan error, 10)
	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	port := builderSeq.L2.ConsensusNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builderSeq.L2Info.GetDefaultTransactOpts("Owner", ctx)
	simpleAddr, _ := builderSeq.L2.DeploySimple(t, auth)
	simpleAbi, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)
	noopId := simpleAbi.Methods["noop"].ID
	revertId := simpleAbi.Methods["pleaseRevert"].ID

	// Generate the accounts before hand to avoid races
	for user := 0; user < 9; user++ {
		name := fmt.Sprintf("User%v", user)
		builderSeq.L2Info.GenerateAccount(name)
	}

	wg := sync.WaitGroup{}
	var stopBackground int32
	for user := 0; user < 9; user++ {
		user := user
		name := fmt.Sprintf("User%v", user)
		tx := builderSeq.L2Info.PrepareTx("Owner", name, builderSeq.L2Info.TransferGas, big.NewInt(params.Ether), nil)

		err := builderSeq.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)

		_, err = builderSeq.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)

		wg.Add(1)
		go func() {
			defer wg.Done()
			info := builderSeq.L2Info.GetInfoWithPrivKey(name)
			txData := &types.DynamicFeeTx{
				To:        &simpleAddr,
				Gas:       builderSeq.L2Info.TransferGas + 10000,
				GasFeeCap: arbmath.BigMulByUint(builderSeq.L2Info.GasPrice, 100),
				Value:     common.Big0,
			}
			for atomic.LoadInt32(&stopBackground) == 0 {
				txData.Nonce = info.Nonce
				var expectedErr string
				if user%3 == 0 {
					txData.Data = noopId
					info.Nonce += 1
				} else if user%3 == 1 {
					txData.Data = revertId
					expectedErr = "execution reverted: SOLIDITY_REVERTING"
				} else {
					txData.Nonce = 1 << 32
					expectedErr = "nonce too high"
				}
				tx = builderSeq.L2Info.SignTxAs(name, txData)
				err = builderSeq.L2.Client.SendTransaction(ctx, tx)
				if err != nil && (expectedErr == "" || !strings.Contains(err.Error(), expectedErr)) {
					Require(t, err, "failed to send tx for user", user)
				}
			}
		}()
	}

	for i := 100; i >= 0; i-- {
		block, err := builderSeq.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if block >= 200 {
			break
		}
		if i == 0 {
			Fatal(t, "failed to reach block 200, only reached block", block)
		}
		select {
		case err := <-feedErrChan:
			Fatal(t, "error: ", err)
		case <-time.After(time.Millisecond * 100):
		}
	}

	atomic.StoreInt32(&stopBackground, 1)
	wg.Wait()

	header1, err := builderSeq.L2.Client.HeaderByNumber(ctx, nil)
	Require(t, err)

	for i := 100; i >= 0; i-- {
		header2, err := builder.L2.Client.HeaderByNumber(ctx, header1.Number)
		if err != nil {
			select {
			case err := <-feedErrChan:
				Fatal(t, "error: ", err)
			case <-time.After(time.Millisecond * 100):
			}
			if i == 0 {
				client2Block, _ := builder.L2.Client.BlockNumber(ctx)
				Fatal(t, "client2 failed to reach client1 block ", header1.Number, ", only reached block", client2Block)
			}
			continue
		}
		if header1.Hash() == header2.Hash() {
			colors.PrintMint("client headers are equal")
			break
		} else {
			colors.PrintBlue("header 1:", header1)
			colors.PrintBlue("header 2:", header2)
			Fatal(t, "header 1 and header 2 have different hashes")
		}
	}
}
