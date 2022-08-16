// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/das"
)

func testTwoNodesSimple(t *testing.T, dasModeStr string) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig, l1NodeConfigA, dasConfig, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeStr)
	dasServerStack, lifecycleManager, err := arbnode.SetUpDataAvailability(ctx, dasConfig, nil, nil)
	Require(t, err)
	rpcLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	restLis, err := net.Listen("tcp", "localhost:0")
	Require(t, err)
	l1NodeConfigA.DataAvailability = das.DefaultDataAvailabilityConfig
	if dasModeStr != "onchain" {
		_, err = das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, dasServerStack)
		Require(t, err)
		_, err = das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, dasServerStack)
		Require(t, err)

		beConfigA := das.BackendConfig{
			URL:                 "http://" + rpcLis.Addr().String(),
			PubKeyBase64Encoded: blsPubToBase64(dasSignerKey),
			SignerMask:          1,
		}
		l1NodeConfigA.DataAvailability.AggregatorConfig = aggConfigForBackend(t, beConfigA)
		l1NodeConfigA.DataAvailability.Enable = true
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig = das.DefaultRestfulClientAggregatorConfig
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Enable = true
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Urls = []string{"http://" + restLis.Addr().String()}
		l1NodeConfigA.DataAvailability.L1NodeURL = "none"
	}

	l2info, nodeA, l2clientA, l2stackA, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, l1NodeConfigA, chainConfig)
	defer requireClose(t, l1stack)
	defer requireClose(t, l2stackA)

	authorizeDASKeyset(t, ctx, dasSignerKey, l1info, l1client)
	l1NodeConfigBDataAvailability := l1NodeConfigA.DataAvailability
	l1NodeConfigBDataAvailability.AggregatorConfig.Enable = false
	l2clientB, _, l2stackB := Create2ndNode(t, ctx, nodeA, l1stack, &l2info.ArbInitData, &l1NodeConfigBDataAvailability)
	defer requireClose(t, l2stackB)

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)

	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = WaitForTx(ctx, l2clientB, tx.Hash(), time.Second*5)
	Require(t, err)

	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)

	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}

	lifecycleManager.StopAndWaitUntil(time.Second)
}

func TestTwoNodesSimple(t *testing.T) {
	testTwoNodesSimple(t, "onchain")
}

func TestTwoNodesSimpleLocalDAS(t *testing.T) {
	testTwoNodesSimple(t, "files")
}
