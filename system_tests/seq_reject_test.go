// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

func TestSequencerRejection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest()
	feedErrChan := make(chan error, 10)
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true)
	defer nodeA.StopAndWait()

	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)

	_, nodeB, client2 := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false)
	defer nodeB.StopAndWait()

	auth := l2info1.GetDefaultTransactOpts("Owner", ctx)
	simpleAddr, _ := deploySimple(t, ctx, auth, client1)
	simpleAbi, err := mocksgen.SimpleMetaData.GetAbi()
	Require(t, err)
	noopId := simpleAbi.Methods["noop"].ID
	revertId := simpleAbi.Methods["pleaseRevert"].ID

	// Generate the accounts before hand to avoid races
	for user := 0; user < 9; user++ {
		name := fmt.Sprintf("User%v", user)
		l2info1.GenerateAccount(name)
	}

	wg := sync.WaitGroup{}
	var stopBackground int32
	for user := 0; user < 9; user++ {
		user := user
		name := fmt.Sprintf("User%v", user)
		tx := l2info1.PrepareTx("Owner", name, l2info1.TransferGas, big.NewInt(params.Ether), nil)

		err := client1.SendTransaction(ctx, tx)
		Require(t, err)

		_, err = EnsureTxSucceeded(ctx, client1, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, client2, tx)
		Require(t, err)

		wg.Add(1)
		go func() {
			defer wg.Done()
			info := l2info1.GetInfoWithPrivKey(name)
			txData := &types.DynamicFeeTx{
				To:        &simpleAddr,
				Gas:       l2info1.TransferGas + 10000,
				GasFeeCap: arbmath.BigMulByUint(l2info1.GasPrice, 100),
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
				tx = l2info1.SignTxAs(name, txData)
				err = client1.SendTransaction(ctx, tx)
				if err != nil && (expectedErr == "" || !strings.Contains(err.Error(), expectedErr)) {
					Require(t, err, "failed to send tx for user", user)
				}
			}
		}()
	}

	for i := 100; i >= 0; i-- {
		block, err := client1.BlockNumber(ctx)
		Require(t, err)
		if block >= 200 {
			break
		}
		if i == 0 {
			Fail(t, "failed to reach block 200, only reached block", block)
		}
		select {
		case err := <-feedErrChan:
			Fail(t, "error: ", err)
		case <-time.After(time.Millisecond * 100):
		}
	}

	atomic.StoreInt32(&stopBackground, 1)
	wg.Wait()

	header1, err := client1.HeaderByNumber(ctx, nil)
	Require(t, err)

	for i := 100; i >= 0; i-- {
		header2, err := client2.HeaderByNumber(ctx, header1.Number)
		if err != nil {
			select {
			case err := <-feedErrChan:
				Fail(t, "error: ", err)
			case <-time.After(time.Millisecond * 100):
			}
			if i == 0 {
				client2Block, _ := client2.BlockNumber(ctx)
				Fail(t, "client2 failed to reach client1 block ", header1.Number, ", only reached block", client2Block)
			}
			continue
		}
		if header1.Hash() == header2.Hash() {
			colors.PrintMint("client headers are equal")
			break
		} else {
			colors.PrintBlue("header 1:", header1)
			colors.PrintBlue("header 2:", header2)
			Fail(t, "header 1 and header 2 have different hashes")
		}
	}
}
