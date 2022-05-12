// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/das"
)

func testBlockValidatorSimple(t *testing.T, dasModeString string, expensiveTx bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l1NodeConfigA := arbnode.ConfigDefaultL1Test()
	l1NodeConfigA.DataAvailability.ModeImpl = dasModeString
	chainConfig := params.ArbitrumDevTestChainConfig()
	var dbPath string
	var err error

	if dasModeString == das.LocalDiskDataAvailabilityString {
		dbPath, err = ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)
		dasConfig := das.LocalDiskDASConfig{
			KeyDir:            dbPath,
			DataDir:           dbPath,
			AllowGenerateKeys: true,
		}
		l1NodeConfigA.DataAvailability.LocalDiskDASConfig = dasConfig
		chainConfig = params.ArbitrumDevTestDASChainConfig()
	}
	_, err = l1NodeConfigA.DataAvailability.Mode()

	Require(t, err)

	l2info, nodeA, l2client, l1info, _, l1client, l1stack := CreateTestNodeOnL1WithConfig(t, ctx, true, l1NodeConfigA, chainConfig)

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
	l1NodeConfigB.BlockValidator.Enable = true
	l1NodeConfigB.DataAvailability.ModeImpl = dasModeString
	dasConfig := das.LocalDiskDASConfig{
		KeyDir:            dbPath,
		DataDir:           dbPath,
		AllowGenerateKeys: true,
	}
	l1NodeConfigB.DataAvailability.LocalDiskDASConfig = dasConfig
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, &l2info.ArbInitData, l1NodeConfigB)

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, big.NewInt(1e12), nil)

	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	if expensiveTx {
		contractData := []byte{0x5b} // JUMPDEST
		for i := 0; i < 20; i++ {
			contractData = append(contractData, 0x60, 0x00, 0x60, 0x00, 0x52) // PUSH1 0 MSTORE
		}
		contractData = append(contractData, 0x60, 0x00, 0x56) // JUMP
		ownerInfo := l2info.GetInfoWithPrivKey("Owner")
		tx = l2info.SignTxAs("Owner", &types.DynamicFeeTx{
			To:        nil,
			Gas:       l2info.TransferGas*2 + l2pricing.InitialPerBlockGasLimit,
			GasFeeCap: new(big.Int).Set(l2info.GasPrice),
			Value:     common.Big0,
			Nonce:     ownerInfo.Nonce,
			Data:      contractData,
		})
		ownerInfo.Nonce++
		err = l2client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
		Require(t, err)
	}

	delayedTx := l2info.PrepareTx("Owner", "User2", 30002, big.NewInt(1e12), nil)
	SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, l1info, "User", 100000),
	})

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 500)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = WaitForTx(ctx, l2clientB, delayedTx.Hash(), time.Second*5)
	Require(t, err)

	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(2e12)) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}

	lastBlockHeader, err := l2clientB.HeaderByNumber(ctx, nil)
	Require(t, err)
	testDeadLine, _ := t.Deadline()
	nodeA.StopAndWait()
	if !nodeB.BlockValidator.WaitForBlock(lastBlockHeader.Number.Uint64(), time.Until(testDeadLine)-time.Second*10) {
		Fail(t, "did not validate all blocks")
	}
	nodeB.StopAndWait()
}

func TestBlockValidatorSimple(t *testing.T) {
	testBlockValidatorSimple(t, das.OnchainDataAvailabilityString, false)
}

func TestBlockValidatorSimpleLocalDAS(t *testing.T) {
	testBlockValidatorSimple(t, das.LocalDiskDataAvailabilityString, false)
}
