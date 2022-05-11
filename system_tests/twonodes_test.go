// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/das"
)

func testTwoNodesSimple(t *testing.T, dasModeStr string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l1NodeConfigA := arbnode.ConfigDefaultL1Test()
	l1NodeConfigA.DataAvailability.ModeImpl = dasModeStr
	chainConfig := params.ArbitrumDevTestChainConfig()
	var dbPath string
	var err error
	if dasModeStr == "local-disk" {
		dbPath, err = ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)
		chainConfig = params.ArbitrumDevTestDASChainConfig()
		dasConfig := das.LocalDiskDASConfig{
			KeyDir:            dbPath,
			DataDir:           dbPath,
			AllowGenerateKeys: true,
		}
		l1NodeConfigA.DataAvailability.LocalDiskDASConfig = dasConfig
	}
	_, err = l1NodeConfigA.DataAvailability.Mode()
	Require(t, err)
	l2info, nodeA, l2clientA, l1info, _, l1client, l1stack := CreateTestNodeOnL1WithConfig(t, ctx, true, l1NodeConfigA, chainConfig)
	defer l1stack.Close()

	usingDas := nodeA.DataAvailService
	if usingDas != nil {
		keysetBytes, err := usingDas.CurrentKeysetBytes(ctx)
		Require(t, err)
		abiBytes := []byte{0xd1, 0xce, 0x8d, 0xa8}
		var buf [32]byte
		buf[31] = 0x20
		abiBytes = append(abiBytes, buf[:]...)
		buf[30] = byte(len(keysetBytes) / 256)
		buf[31] = byte(len(keysetBytes) % 256)
		abiBytes = append(abiBytes, buf[:]...)
		abiBytes = append(abiBytes, keysetBytes...)
		for len(abiBytes)%32 != 4 {
			abiBytes = append(abiBytes, byte(0))
		}
		tx := l1info.PrepareTx("RollupOwner", "SequencerInbox", 2000000, big.NewInt(0), abiBytes)
		err = l1client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l1client, tx)
		Require(t, err)
	}

	l1NodeConfigB := arbnode.ConfigDefaultL1Test()
	l1NodeConfigB.BatchPoster.Enable = false
	l1NodeConfigB.BlockValidator.Enable = false
	l1NodeConfigB.DataAvailability.ModeImpl = dasModeStr
	dasConfig := das.LocalDiskDASConfig{
		KeyDir:            dbPath,
		DataDir:           dbPath,
		AllowGenerateKeys: true,
	}
	l1NodeConfigB.DataAvailability.LocalDiskDASConfig = dasConfig
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, l1NodeConfigB)

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

	nodeA.StopAndWait()
	nodeB.StopAndWait()
}

func TestTwoNodesSimple(t *testing.T) {
	testTwoNodesSimple(t, "onchain")
}

func TestTwoNodesSimpleLocalDAS(t *testing.T) {
	testTwoNodesSimple(t, "local-disk")
}
