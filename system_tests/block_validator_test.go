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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/validator/server_api"
)

type workloadType uint

const (
	ethSend workloadType = iota
	smallContract
	depleteGas
	upgradeArbOs
)

func testBlockValidatorSimple(t *testing.T, dasModeString string, workloadLoops int, workload workloadType, arbitrator bool, useRedisStreams bool) {
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

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig = l1NodeConfigA
	builder.chainConfig = chainConfig
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	authorizeDASKeyset(t, ctx, dasSignerKey, builder.L1Info, builder.L1.Client)

	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.BlockValidator.Enable = true
	validatorConfig.DataAvailability = l1NodeConfigA.DataAvailability
	validatorConfig.DataAvailability.RPCAggregator.Enable = false
	redisURL := ""
	if useRedisStreams {
		redisURL = redisutil.CreateTestRedis(ctx, t)
		validatorConfig.BlockValidator.RedisValidationClientConfig = server_api.DefaultRedisValidationClientConfig
		validatorConfig.BlockValidator.RedisValidationClientConfig.ModuleRoots = []string{wasmModuleRoot}
		stream := server_api.RedisStreamForRoot(common.HexToHash(wasmModuleRoot))
		validatorConfig.BlockValidator.RedisValidationClientConfig.RedisStream = stream
		validatorConfig.BlockValidator.RedisValidationClientConfig.RedisURL = redisURL
	}

	AddDefaultValNode(t, ctx, validatorConfig, !arbitrator, redisURL)

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: validatorConfig})
	if useRedisStreams {
		if err := testClientB.ConsensusNode.BlockValidator.SetCurrentWasmModuleRoot(common.HexToHash(wasmModuleRoot)); err != nil {
			t.Fatalf("Error setting wasm module root: %v", err)
		}
	}
	defer cleanupB()
	builder.L2Info.GenerateAccount("User2")

	perTransfer := big.NewInt(1e12)

	var simple *mocksgen.Simple
	if workload != upgradeArbOs {
		for i := 0; i < workloadLoops; i++ {
			var tx *types.Transaction

			if workload == ethSend {
				tx = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, perTransfer, nil)
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
					contractCode = append(contractCode, byte(vm.BLOBHASH))
					contractCode = append(contractCode, byte(vm.RETURN))
					basefee := builder.L2.GetBaseFee(t)
					var err error
					gas, err = builder.L2.Client.EstimateGas(ctx, ethereum.CallMsg{
						From:     builder.L2Info.GetAddress("Owner"),
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
					gas = builder.L2Info.TransferGas*2 + l2pricing.InitialPerBlockGasLimitV6
				}
				tx = builder.L2Info.PrepareTxTo("Owner", nil, gas, common.Big0, contractCode)
			}

			err := builder.L2.Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, err = builder.L2.EnsureTxSucceeded(tx)
			if workload != depleteGas {
				Require(t, err)
			}
			if delayEvery > 0 && i%delayEvery == (delayEvery-1) {
				<-time.After(time.Second)
			}
		}
	} else {
		auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
		// deploy a test contract
		var err error
		_, _, simple, err = mocksgen.DeploySimple(&auth, builder.L2.Client)
		Require(t, err, "could not deploy contract")

		tx, err := simple.StoreDifficulty(&auth)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
		Require(t, err)
		difficulty, err := simple.GetBlockDifficulty(&bind.CallOpts{})
		Require(t, err)
		if !arbmath.BigEquals(difficulty, common.Big1) {
			Fatal(t, "Expected difficulty to be 1 but got:", difficulty)
		}
		// make auth a chain owner
		arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
		Require(t, err)
		tx, err = arbDebug.BecomeChainOwner(&auth)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
		Require(t, err)
		tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, 11, 0)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)

		tx, err = simple.StoreDifficulty(&auth)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
		Require(t, err)
		difficulty, err = simple.GetBlockDifficulty(&bind.CallOpts{})
		Require(t, err)
		if !arbmath.BigEquals(difficulty, common.Big1) {
			Fatal(t, "Expected difficulty to be 1 but got:", difficulty)
		}

		tx = builder.L2Info.PrepareTxTo("Owner", nil, builder.L2Info.TransferGas, perTransfer, []byte{byte(vm.PUSH0)})
		err = builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	if workload != depleteGas {
		delayedTx := builder.L2Info.PrepareTx("Owner", "User2", 30002, perTransfer, nil)
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			WrapL2ForDelayed(t, delayedTx, builder.L1Info, "User", 100000),
		})
		// give the inbox reader a bit of time to pick up the delayed message
		time.Sleep(time.Millisecond * 500)

		// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
		for i := 0; i < 30; i++ {
			builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
				builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
			})
		}

		_, err := WaitForTx(ctx, testClientB.Client, delayedTx.Hash(), time.Second*30)
		Require(t, err)
	}

	if workload == ethSend {
		l2balance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
		Require(t, err)

		expectedBalance := new(big.Int).Mul(perTransfer, big.NewInt(int64(workloadLoops+1)))
		if l2balance.Cmp(expectedBalance) != 0 {
			Fatal(t, "Unexpected balance:", l2balance)
		}
	}

	lastBlock, err := testClientB.Client.BlockByNumber(ctx, nil)
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
		lastBlock, err = testClientB.Client.BlockByHash(ctx, lastBlock.ParentHash())
		Require(t, err)
	}
	t.Log("waiting for block: ", lastBlock.NumberU64())
	timeout := getDeadlineTimeout(t, time.Minute*10)
	// messageindex is same as block number here
	if !testClientB.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(lastBlock.NumberU64()), timeout) {
		Fatal(t, "did not validate all blocks")
	}
	gethExec, ok := testClientB.ConsensusNode.Execution.(*gethexec.ExecutionNode)
	if !ok {
		t.Fail()
	}
	gethExec.Recorder.TrimAllPrepared(t)
	finalRefCount := gethExec.Recorder.RecordingDBReferenceCount()
	lastBlockNow, err := testClientB.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	// up to 3 extra references: awaiting validation, recently valid, lastValidatedHeader
	largestRefCount := lastBlockNow.NumberU64() - lastBlock.NumberU64() + 3
	if finalRefCount < 0 || finalRefCount > int64(largestRefCount) {
		Fatal(t, "unexpected refcount:", finalRefCount)
	}
}

func TestBlockValidatorSimpleOnchainUpgradeArbOs(t *testing.T) {
	testBlockValidatorSimple(t, "onchain", 1, upgradeArbOs, true, false)
}

func TestBlockValidatorSimpleOnchain(t *testing.T) {
	testBlockValidatorSimple(t, "onchain", 1, ethSend, true, false)
}

func TestBlockValidatorSimpleOnchainWithRedisStreams(t *testing.T) {
	testBlockValidatorSimple(t, "onchain", 1, ethSend, true, true)
}

func TestBlockValidatorSimpleLocalDAS(t *testing.T) {
	testBlockValidatorSimple(t, "files", 1, ethSend, true, false)
}

func TestBlockValidatorSimpleJITOnchain(t *testing.T) {
	testBlockValidatorSimple(t, "files", 8, smallContract, false, false)
}
