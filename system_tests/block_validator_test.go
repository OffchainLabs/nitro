// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// race detection makes things slow and miss timeouts
//go:build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/redisutil"
	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
	"github.com/offchainlabs/nitro/util/testhelpers/github"
	"github.com/offchainlabs/nitro/validator/client/redis"
)

type workloadType uint

const (
	ethSend workloadType = iota
	smallContract
	depleteGas
	upgradeArbOs
)

type Options struct {
	dasModeString   string
	workloadLoops   int
	workload        workloadType
	arbitrator      bool
	useRedisStreams bool
	wasmRootDir     string
	arbosVersion    uint64 // sets InitialArbOSVersion, overwrites any other operation setting it like upgradeArbOs worload
}

func testBlockValidatorSimple(t *testing.T, opts Options) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig, l1NodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, opts.dasModeString)
	if lifecycleManager != nil {
		defer lifecycleManager.StopAndWaitUntil(time.Second)
	}
	if opts.workload == upgradeArbOs {
		chainConfig.ArbitrumChainParams.InitialArbOSVersion = params.ArbosVersion_10
	}

	var delayEvery int
	if opts.workloadLoops > 1 {
		delayEvery = opts.workloadLoops / 3
	}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder = builder.WithWasmRootDir(opts.wasmRootDir)
	// For now PathDB is not supported when using block validation
	builder.RequireScheme(t, rawdb.HashScheme)

	builder.nodeConfig = l1NodeConfigA
	builder.chainConfig = chainConfig
	if opts.arbosVersion != 0 {
		builder.WithArbOSVersion(opts.arbosVersion)
	}
	builder.L2Info = nil

	// Configure for referenceda mode - deploy validator contract and create provider server
	if opts.dasModeString == "referenceda" {
		builder.WithReferenceDA()
	}

	cleanup := builder.Build(t)
	defer cleanup()

	// Only authorize DAS keyset if we're using traditional DAS
	if opts.dasModeString != "onchain" && opts.dasModeString != "referenceda" && dasSignerKey != nil {
		authorizeDASKeyset(t, ctx, dasSignerKey, builder.L1Info, builder.L1.Client)
	}

	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.BlockValidator.Enable = true

	// Configure validator based on DA mode
	if opts.dasModeString == "referenceda" {
		// For external referenceda, configure the validator to use external provider
		validatorConfig.DA.ExternalProvider.Enable = true
		validatorConfig.DA.ExternalProvider.RPC.URL = builder.referenceDAURL
		validatorConfig.DataAvailability.Enable = false
	} else {
		// For traditional DAS, copy DataAvailability configuration
		validatorConfig.DataAvailability = l1NodeConfigA.DataAvailability
		validatorConfig.DataAvailability.RPCAggregator.Enable = false
	}
	redisURL := ""
	if opts.useRedisStreams {
		redisURL = redisutil.CreateTestRedis(ctx, t)
		validatorConfig.BlockValidator.RedisValidationClientConfig = redis.TestValidationClientConfig
		validatorConfig.BlockValidator.RedisValidationClientConfig.RedisURL = redisURL
	} else {
		validatorConfig.BlockValidator.RedisValidationClientConfig = redis.ValidationClientConfig{}
	}

	AddValNode(t, ctx, validatorConfig, !opts.arbitrator, redisURL, opts.wasmRootDir)

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: validatorConfig})
	defer cleanupB()
	builder.L2Info.GenerateAccount("User2")

	perTransfer := big.NewInt(1e12)

	var simple *localgen.Simple
	if opts.workload != upgradeArbOs {
		for i := 0; i < opts.workloadLoops; i++ {
			var tx *types.Transaction

			if opts.workload == ethSend {
				tx = builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, perTransfer, nil)
			} else {
				var contractCode []byte
				var gas uint64

				if opts.workload == smallContract {
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
			if opts.workload != depleteGas {
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
		_, _, simple, err = localgen.DeploySimple(&auth, builder.L2.Client)
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

	if opts.workload != depleteGas {
		delayedTx := builder.L2Info.PrepareTx("Owner", "User2", 30002, perTransfer, nil)
		builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
			WrapL2ForDelayed(t, delayedTx, builder.L1Info, "User", 100000),
		})

		// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
		for i := 0; i < 30; i++ {
			builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
				builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
			})
		}

		_, err := WaitForTx(ctx, testClientB.Client, delayedTx.Hash(), time.Second*30)
		Require(t, err)
	}

	if opts.workload == ethSend {
		l2balance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
		Require(t, err)

		expectedBalance := new(big.Int).Mul(perTransfer, big.NewInt(int64(opts.workloadLoops+1)))
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
	gethExec, ok := testClientB.ConsensusNode.ExecutionClient.(*gethexec.ExecutionNode)
	if !ok {
		t.Fail()
	}
	gethExec.Recorder.TrimAllPrepared(t)
	finalRefCount := gethExec.Recorder.RecordingDBReferenceCount()
	lastBlockNow, err := testClientB.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	// up to 3 extra references: awaiting validation, recently valid, lastValidatedHeader
	largestRefCount := lastBlockNow.NumberU64() - lastBlock.NumberU64() + 3
	// #nosec G115
	if finalRefCount < 0 || finalRefCount > int64(largestRefCount) {
		Fatal(t, "unexpected refcount:", finalRefCount)
	}
}

func TestBlockRecordSimple(t *testing.T) {
	if !*testflag.RecordBlockInputsEnable {
		t.Skip("not recording")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BlockValidator.Enable = true
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	recordBlock(t, receipt.BlockNumber.Uint64(), builder, rawdb.TargetWavm, rawdb.LocalTarget())
	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)
}

func TestBlockValidatorSimpleOnchainUpgradeArbOs(t *testing.T) {
	opts := Options{
		dasModeString: "onchain",
		workloadLoops: 1,
		workload:      upgradeArbOs,
		arbitrator:    true,
	}
	testBlockValidatorSimple(t, opts)
}

func TestBlockValidatorSimpleOnchain(t *testing.T) {
	opts := Options{
		dasModeString: "onchain",
		workloadLoops: 1,
		workload:      ethSend,
		arbitrator:    true,
	}
	testBlockValidatorSimple(t, opts)
}

func TestBlockValidatorSimpleJITOnchainWithPublishedMachine(t *testing.T) {
	cr, err := github.LatestConsensusRelease(context.Background())
	Require(t, err)
	machPath := populateMachineDir(t, cr)
	opts := Options{
		dasModeString: "onchain",
		workloadLoops: 1,
		workload:      ethSend,
		arbitrator:    false,
		wasmRootDir:   machPath,
		arbosVersion:  cr.ArbosVersion,
	}
	testBlockValidatorSimple(t, opts)
}

func TestBlockValidatorSimpleOnchainWithPublishedMachine(t *testing.T) {
	cr, err := github.LatestConsensusRelease(context.Background())
	Require(t, err)
	machPath := populateMachineDir(t, cr)
	opts := Options{
		dasModeString: "onchain",
		workloadLoops: 1,
		workload:      ethSend,
		arbitrator:    true,
		wasmRootDir:   machPath,
		arbosVersion:  cr.ArbosVersion,
	}
	testBlockValidatorSimple(t, opts)
}

func TestBlockValidatorSimpleOnchainWithRedisStreams(t *testing.T) {
	opts := Options{
		dasModeString:   "onchain",
		workloadLoops:   1,
		workload:        ethSend,
		arbitrator:      true,
		useRedisStreams: true,
	}
	testBlockValidatorSimple(t, opts)
}

func TestBlockValidatorSimpleLocalDAS(t *testing.T) {
	opts := Options{
		dasModeString: "files",
		workloadLoops: 1,
		workload:      ethSend,
		arbitrator:    true,
	}
	testBlockValidatorSimple(t, opts)
}

func TestBlockValidatorSimpleJITOnchain(t *testing.T) {
	opts := Options{
		dasModeString: "files",
		workloadLoops: 8,
		workload:      smallContract,
	}
	testBlockValidatorSimple(t, opts)
}

// TestBlockValidatorReferenceDAWithProver tests the block validator with prover
// with the embedded reference DA
func TestBlockValidatorReferenceDAWithProver(t *testing.T) {
	opts := Options{
		dasModeString: "referenceda",
		workloadLoops: 1,
		workload:      ethSend,
		arbitrator:    true,
	}
	testBlockValidatorSimple(t, opts)
}

// TestBlockValidatorReferenceDAWithJIT tests the block validator with JIT
// with the embedded reference DA
func TestBlockValidatorReferenceDAWithJIT(t *testing.T) {
	opts := Options{
		dasModeString: "referenceda",
		workloadLoops: 1,
		workload:      ethSend,
		arbitrator:    false,
	}
	testBlockValidatorSimple(t, opts)
}
