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
	"github.com/offchainlabs/nitro/arbutil"
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

	var delayEvery int
	if workloadLoops > 1 {
		l1NodeConfigA.BatchPoster.MaxDelay = time.Millisecond * 500
		delayEvery = workloadLoops / 3
	}

	testNodeA := NewNodeBuilder(ctx).SetIsSequencer(true).SetNodeConfig(l1NodeConfigA).SetChainConfig(chainConfig).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNodeA.L1Stack)
	defer testNodeA.L2Node.StopAndWait()

	authorizeDASKeyset(t, ctx, dasSignerKey, testNodeA.L1Info, testNodeA.L1Client)

	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.BlockValidator.Enable = true
	validatorConfig.DataAvailability = l1NodeConfigA.DataAvailability
	validatorConfig.DataAvailability.RPCAggregator.Enable = false
	AddDefaultValNode(t, ctx, validatorConfig, !arbitrator)
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, testNodeA.L2Node, testNodeA.L1Stack, testNodeA.L1Info, &testNodeA.L2Info.ArbInitData, validatorConfig, nil)
	defer nodeB.StopAndWait()
	testNodeA.L2Info.GenerateAccount("User2")

	perTransfer := big.NewInt(1e12)

	if workload != upgradeArbOs {
		for i := 0; i < workloadLoops; i++ {
			var tx *types.Transaction

			if workload == ethSend {
				tx = testNodeA.L2Info.PrepareTx("Owner", "User2", testNodeA.L2Info.TransferGas, perTransfer, nil)
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
					basefee := testNodeA.GetBaseFeeAtViaL2(t, nil)
					var err error
					gas, err = testNodeA.L2Client.EstimateGas(ctx, ethereum.CallMsg{
						From:     testNodeA.L2Info.GetAddress("Owner"),
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
					gas = testNodeA.L2Info.TransferGas*2 + l2pricing.InitialPerBlockGasLimitV6
				}
				tx = testNodeA.L2Info.PrepareTxTo("Owner", nil, gas, common.Big0, contractCode)
			}

			err := testNodeA.L2Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, err = EnsureTxSucceededWithTimeout(ctx, testNodeA.L2Client, tx, time.Second*5)
			if workload != depleteGas {
				Require(t, err)
			}
			if delayEvery > 0 && i%delayEvery == (delayEvery-1) {
				<-time.After(time.Second)
			}
		}
	} else {
		auth := testNodeA.L2Info.GetDefaultTransactOpts("Owner", ctx)
		// make auth a chain owner
		arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), testNodeA.L2Client)
		Require(t, err)
		tx, err := arbDebug.BecomeChainOwner(&auth)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, testNodeA.L2Client, tx)
		Require(t, err)
		arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), testNodeA.L2Client)
		Require(t, err)
		tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, 11, 0)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, testNodeA.L2Client, tx)
		Require(t, err)

		tx = testNodeA.L2Info.PrepareTxTo("Owner", nil, testNodeA.L2Info.TransferGas, perTransfer, []byte{byte(vm.PUSH0)})
		err = testNodeA.L2Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceededWithTimeout(ctx, testNodeA.L2Client, tx, time.Second*5)
		Require(t, err)
	}

	if workload != depleteGas {
		delayedTx := testNodeA.L2Info.PrepareTx("Owner", "User2", 30002, perTransfer, nil)
		SendWaitTestTransactions(t, ctx, testNodeA.L1Client, []*types.Transaction{
			WrapL2ForDelayed(t, delayedTx, testNodeA.L1Info, "User", 100000),
		})
		// give the inbox reader a bit of time to pick up the delayed message
		time.Sleep(time.Millisecond * 500)

		// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
		for i := 0; i < 30; i++ {
			SendWaitTestTransactions(t, ctx, testNodeA.L1Client, []*types.Transaction{
				testNodeA.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
			})
		}

		_, err := WaitForTx(ctx, l2clientB, delayedTx.Hash(), time.Second*5)
		Require(t, err)
	}

	if workload == ethSend {
		l2balance, err := l2clientB.BalanceAt(ctx, testNodeA.L2Info.GetAddress("User2"), nil)
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
	// messageindex is same as block number here
	if !nodeB.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlock.NumberU64()), timeout) {
		Fatal(t, "did not validate all blocks")
	}
	nodeB.Execution.Recorder.TrimAllPrepared(t)
	finalRefCount := nodeB.Execution.Recorder.RecordingDBReferenceCount()
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
