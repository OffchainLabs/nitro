// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// race detection makes things slow and miss timeouts
//go:build !race
// +build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
)

func testBlockValidatorSimple(t *testing.T, dasModeString string, simpletxloops int, expensiveTx bool, arbitrator bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig, l1NodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeString)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	l2info, nodeA, l2client, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, l1NodeConfigA, chainConfig, nil)
	defer requireClose(t, l1stack)
	defer nodeA.StopAndWait()

	authorizeDASKeyset(t, ctx, dasSignerKey, l1info, l1client)

	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.BlockValidator.Enable = true
	validatorConfig.DataAvailability = l1NodeConfigA.DataAvailability
	validatorConfig.DataAvailability.AggregatorConfig.Enable = false
	AddDefaultValNode(t, ctx, validatorConfig, !arbitrator)
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, l1info, &l2info.ArbInitData, validatorConfig, nil)
	defer nodeB.StopAndWait()
	l2info.GenerateAccount("User2")

	perTransfer := big.NewInt(1e12)

	for i := 0; i < simpletxloops; i++ {

		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, perTransfer, nil)

		err := l2client.SendTransaction(ctx, tx)
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
			tx := l2info.SignTxAs("Owner", &types.DynamicFeeTx{
				To:        nil,
				Gas:       l2info.TransferGas*2 + l2pricing.InitialPerBlockGasLimitV6,
				GasFeeCap: new(big.Int).Set(l2info.GasPrice),
				Value:     common.Big0,
				Nonce:     ownerInfo.Nonce,
				Data:      contractData,
			})
			ownerInfo.Nonce++
			err := l2client.SendTransaction(ctx, tx)
			Require(t, err)
			_, err = EnsureTxSucceededWithTimeout(ctx, l2client, tx, time.Second*5)
			Require(t, err)
		}

	}

	delayedTx := l2info.PrepareTx("Owner", "User2", 30002, perTransfer, nil)
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

	_, err := WaitForTx(ctx, l2clientB, delayedTx.Hash(), time.Second*5)
	Require(t, err)

	l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
	Require(t, err)
	expectedBalance := new(big.Int).Mul(perTransfer, big.NewInt(int64(simpletxloops+1)))
	if l2balance.Cmp(expectedBalance) != 0 {
		Fail(t, "Unexpected balance:", l2balance)
	}

	lastBlock, err := l2clientB.BlockByNumber(ctx, nil)
	Require(t, err)
	for {
		usefulBlock := false
		for _, tx := range lastBlock.Transactions() {
			if tx.Type() != types.ArbitrumInternalTxType {
				usefulBlock = true
				break
			}
		}
		if usefulBlock {
			break
		}
		lastBlock, err = l2clientB.BlockByHash(ctx, lastBlock.ParentHash())
		Require(t, err)
	}
	t.Log("waiting for block: ", lastBlock.NumberU64())
	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !nodeB.BlockValidator.WaitForBlock(ctx, lastBlock.NumberU64(), timeout) {
		Fail(t, "did not validate all blocks")
	}
	finalRefCount := nodeB.BlockValidator.RecordDBReferenceCount()
	lastBlockNow, err := l2clientB.BlockByNumber(ctx, nil)
	Require(t, err)
	// up to 3 extra references: awaiting validation, recently valid, lastValidatedHeader
	largestRefCount := lastBlockNow.NumberU64() - lastBlock.NumberU64() + 3
	if finalRefCount < 0 || finalRefCount > int64(largestRefCount) {
		Fail(t, "unexpected refcount:", finalRefCount)
	}
}

func TestBlockValidatorSimpleOnchain(t *testing.T) {
	testBlockValidatorSimple(t, "onchain", 1, false, true)
}

func TestBlockValidatorSimpleLocalDAS(t *testing.T) {
	testBlockValidatorSimple(t, "files", 1, false, true)
}

func TestBlockValidatorSimpleJITOnchain(t *testing.T) {
	testBlockValidatorSimple(t, "files", 8, true, false)
}
