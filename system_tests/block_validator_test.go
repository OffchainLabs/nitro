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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

type workloadType uint

const (
	ethSend workloadType = iota
	smallContract
	depleteGas
	upgradeArbOs
)

func testBlockValidatorSimple(t *testing.T, dasModeString string, workloadLoops int, workload workloadType, arbitrator bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig, l1NodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeString)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	if workload == upgradeArbOs {
		chainConfig.ArbitrumChainParams.InitialArbOSVersion = 10
	}

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

	if workload != upgradeArbOs {
		for i := 0; i < workloadLoops; i++ {
			var tx *types.Transaction

			if workload == ethSend {
				tx = l2info.PrepareTx("Owner", "User2", l2info.TransferGas, perTransfer, nil)
			} else {
				var contractCode []byte
				var gas uint64

				if workload == smallContract {
					contractCode = []byte{byte(vm.PUSH0)}
					contractCode = append(contractCode, byte(vm.PUSH0))
					contractCode = append(contractCode, byte(vm.PUSH1))
					contractCode = append(contractCode, 8) // the prelude length
					contractCode = append(contractCode, byte(vm.PUSH0))
					contractCode = append(contractCode, byte(vm.CODECOPY))
					contractCode = append(contractCode, byte(vm.PUSH0))
					contractCode = append(contractCode, byte(vm.RETURN))
					basefee := GetBaseFee(t, l2client, ctx)
					var err error
					gas, err = l2client.EstimateGas(ctx, ethereum.CallMsg{
						From:     l2info.GetAddress("Owner"),
						GasPrice: basefee,
						Value:    big.NewInt(0),
						Data:     contractCode,
					})
					Require(t, err)
				} else {
					contractCode = []byte{0x5b} // JUMPDEST
					for i := 0; i < 20; i++ {
						contractCode = append(contractCode, 0x60, 0x00, 0x60, 0x00, 0x52) // PUSH1 0 MSTORE
					}
					contractCode = append(contractCode, 0x60, 0x00, 0x56) // JUMP
					gas = l2info.TransferGas*2 + l2pricing.InitialPerBlockGasLimitV6
				}
				tx = l2info.PrepareTxTo("Owner", nil, gas, common.Big0, contractCode)
			}

			err := l2client.SendTransaction(ctx, tx)
			Require(t, err)
			_, err = EnsureTxSucceededWithTimeout(ctx, l2client, tx, time.Second*5)
			if workload != depleteGas {
				Require(t, err)
			}
		}
	} else {
		auth := l2info.GetDefaultTransactOpts("Owner", ctx)
		// make auth a chain owner
		arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), l2client)
		Require(t, err)
		tx, err := arbDebug.BecomeChainOwner(&auth)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), l2client)
		Require(t, err)
		tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, 11, 0)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)

		tx = l2info.PrepareTxTo("Owner", nil, l2info.TransferGas, perTransfer, []byte{byte(vm.PUSH0)})
		err = l2client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceededWithTimeout(ctx, l2client, tx, time.Second*5)
	}

	if workload != depleteGas {
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
	}

	if workload == ethSend {
		l2balance, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("User2"), nil)
		Require(t, err)

		expectedBalance := new(big.Int).Mul(perTransfer, big.NewInt(int64(workloadLoops+1)))
		if l2balance.Cmp(expectedBalance) != 0 {
			Fatal(t, "Unexpected balance:", l2balance)
		}
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
		Fatal(t, "did not validate all blocks")
	}
	finalRefCount := nodeB.BlockValidator.RecordDBReferenceCount()
	lastBlockNow, err := l2clientB.BlockByNumber(ctx, nil)
	Require(t, err)
	// up to 3 extra references: awaiting validation, recently valid, lastValidatedHeader
	largestRefCount := lastBlockNow.NumberU64() - lastBlock.NumberU64() + 3
	if finalRefCount < 0 || finalRefCount > int64(largestRefCount) {
		Fatal(t, "unexpected refcount:", finalRefCount)
	}
}

func TestBlockValidatorSimpleOnchainUpgradeArbOs(t *testing.T) {
	testBlockValidatorSimple(t, "onchain", 1, upgradeArbOs, true)
}

func TestBlockValidatorSimpleOnchain(t *testing.T) {
	testBlockValidatorSimple(t, "onchain", 1, ethSend, true)
}

func TestBlockValidatorSimpleLocalDAS(t *testing.T) {
	testBlockValidatorSimple(t, "files", 1, ethSend, true)
}

func TestBlockValidatorSimpleJITOnchain(t *testing.T) {
	testBlockValidatorSimple(t, "files", 8, smallContract, false)
}
