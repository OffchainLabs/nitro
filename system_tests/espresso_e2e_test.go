package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var workingDir = "./espresso-e2e"
var hotShotAddress = "0x217788c286797d56cd59af5e493f3699c39cbbe8"
var hostIoAddress = "0xF34C2fac45527E55ED122f80a969e79A40547e6D"

var (
	jitValidationPort = 54320
	arbValidationPort = 54321
	broadcastPort     = 9642
)

func runEspresso(t *testing.T, ctx context.Context) func() {
	shutdown := func() {
		p := exec.Command("docker", "compose", "down")
		p.Dir = workingDir
		err := p.Run()
		if err != nil {
			panic(err)
		}
	}

	shutdown()
	invocation := []string{"compose", "up", "-d"}
	nodes := []string{
		"orchestrator",
		"da-server",
		"consensus-server",
		"espresso-sequencer0",
		"espresso-sequencer1",
		"commitment-task",
		"state-relay-server",
	}
	invocation = append(invocation, nodes...)
	procees := exec.Command("docker", invocation...)
	procees.Dir = workingDir

	go func() {
		if err := procees.Run(); err != nil {
			log.Error(err.Error())
			panic(err)
		}
	}()
	return shutdown
}

func createL2Node(ctx context.Context, t *testing.T, hotshot_url string) (*TestClient, info, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.Sequencer = true
	builder.nodeConfig.Espresso = true
	builder.execConfig.Sequencer.Enable = true
	builder.execConfig.Sequencer.Espresso = true
	builder.execConfig.Sequencer.EspressoNamespace = 412346
	builder.execConfig.Sequencer.HotShotUrl = hotshot_url

	builder.chainConfig.ArbitrumChainParams.EnableEspresso = true

	builder.nodeConfig.Feed.Output.Enable = true
	builder.nodeConfig.Feed.Output.Port = fmt.Sprintf("%d", broadcastPort)

	cleanup := builder.Build(t)
	return builder.L2, builder.L2Info, cleanup
}

func createValidationNode(ctx context.Context, t *testing.T, jit bool) func() {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	port := jitValidationPort
	if !jit {
		port = arbValidationPort
	}
	stackConf.WSPort = port
	stackConf.WSModules = []string{server_api.Namespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	valnode.EnsureValidationExposedViaAuthRPC(&stackConf)
	config := &valnode.TestValidationConfig
	config.UseJit = jit

	stack, err := node.New(&stackConf)
	Require(t, err)

	configFetcher := func() *valnode.Config { return config }
	valnode, err := valnode.CreateValidationNode(configFetcher, stack, nil)
	Require(t, err)

	err = stack.Start()
	Require(t, err)

	err = valnode.Start(ctx)
	Require(t, err)

	go func() {
		<-ctx.Done()
		stack.Close()
	}()

	return func() {
		valnode.GetExec().Stop()
		stack.Close()
	}

}

func createL1ValidatorPosterNode(ctx context.Context, t *testing.T) (*NodeBuilder, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l1StackConfig.HTTPPort = 8545
	builder.l1StackConfig.WSPort = 8546
	builder.l1StackConfig.HTTPHost = "0.0.0.0"
	builder.l1StackConfig.HTTPVirtualHosts = []string{"*"}
	builder.l1StackConfig.WSHost = "0.0.0.0"
	builder.l1StackConfig.DataDir = t.TempDir()
	builder.l1StackConfig.WSModules = append(builder.l1StackConfig.WSModules, "eth")

	builder.chainConfig.ArbitrumChainParams.EnableEspresso = true

	builder.nodeConfig.Feed.Input.URL = []string{fmt.Sprintf("ws://127.0.0.1:%d", broadcastPort)}
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.BatchPoster.MaxSize = 41
	builder.nodeConfig.BatchPoster.MaxDelay = -1000 * time.Hour
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", arbValidationPort)
	builder.nodeConfig.BlockValidator.HotShotAddress = hotShotAddress
	builder.nodeConfig.BlockValidator.Espresso = true

	cleanup := builder.Build(t)

	// Fund the commitment task
	mnemonic := "indoor dish desk flag debris potato excuse depart ticket judge file exit"
	err := builder.L1Info.GenerateAccountWithMnemonic("CommitmentTask", mnemonic, 5)
	Require(t, err)
	builder.L1.TransferBalance(t, "Faucet", "CommitmentTask", big.NewInt(9e18), builder.L1Info)

	// Fund the stakers
	builder.L1Info.GenerateAccount("Staker1")
	builder.L1.TransferBalance(t, "Faucet", "Staker1", big.NewInt(9e18), builder.L1Info)
	builder.L1Info.GenerateAccount("Staker2")
	builder.L1.TransferBalance(t, "Faucet", "Staker2", big.NewInt(9e18), builder.L1Info)

	// Update the rollup
	deployAuth := builder.L1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(builder.L2.ConsensusNode.DeployInfo.UpgradeExecutor, builder.L1.Client)
	Require(t, err)
	rollupABI, err := abi.JSON(strings.NewReader(rollupgen.RollupAdminLogicABI))
	Require(t, err)

	setMinAssertPeriodCalldata, err := rollupABI.Pack("setMinimumAssertionPeriod", big.NewInt(0))
	Require(t, err, "unable to generate setMinimumAssertionPeriod calldata")
	tx, err := upgradeExecutor.ExecuteCall(&deployAuth, builder.L2.ConsensusNode.DeployInfo.Rollup, setMinAssertPeriodCalldata)
	Require(t, err, "unable to set minimum assertion period")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	// Add the stakers into the validator whitelist
	staker1Addr := builder.L1Info.GetAddress("Staker1")
	staker2Addr := builder.L1Info.GetAddress("Staker2")
	setValidatorCalldata, err := rollupABI.Pack("setValidator", []common.Address{staker1Addr, staker2Addr}, []bool{true, true})
	Require(t, err, "unable to generate setValidator calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, builder.L2.ConsensusNode.DeployInfo.Rollup, setValidatorCalldata)
	Require(t, err, "unable to set validators")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	return builder, cleanup
}

func createStaker(ctx context.Context, t *testing.T, builder *NodeBuilder, incorrectHeight uint64) (*staker.Staker, *staker.BlockValidator, func()) {
	config := arbnode.ConfigDefaultL1Test()
	config.Sequencer = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	config.Staker.Enable = false
	config.BlockValidator.Enable = true
	config.BlockValidator.HotShotAddress = hotShotAddress
	config.BlockValidator.Espresso = true
	config.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", arbValidationPort)
	testClient, cleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: config})
	l2Node := testClient.ConsensusNode

	var auth bind.TransactOpts
	if incorrectHeight > 0 {
		auth = builder.L1Info.GetDefaultTransactOpts("Staker2", ctx)
	} else {
		auth = builder.L1Info.GetDefaultTransactOpts("Staker1", ctx)
	}

	cfg := arbnode.ConfigDefaultL1NonSequencerTest()
	parentChainID, err := builder.L1.Client.ChainID(ctx)
	Require(t, err)
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2Node.ArbDB, storage.StakerPrefix),
		l2Node.L1Reader,
		&auth,
		NewFetcherFromConfig(cfg),
		nil,
		parentChainID,
	)
	Require(t, err)
	wallet, err := validatorwallet.NewEOA(dp, l2Node.DeployInfo.Rollup, l2Node.L1Reader.Client(), func() uint64 { return 50000 })
	Require(t, err)

	if incorrectHeight > 0 {
		l2Node.StatelessBlockValidator.DebugEspresso_SetIncorrectHeight(incorrectHeight, t)
		l2Node.BlockValidator.DebugEspresso_SetIncorrectHeight(incorrectHeight, t)
	}

	err = wallet.Initialize(ctx)
	Require(t, err)
	valConfig := staker.TestL1ValidatorConfig
	valConfig.Strategy = "MakeNodes"
	valConfig.StartValidationFromStaked = false
	staker, err := staker.NewStaker(
		l2Node.L1Reader,
		wallet,
		bind.CallOpts{},
		valConfig,
		l2Node.BlockValidator,
		l2Node.StatelessBlockValidator,
		nil,
		nil,
		l2Node.DeployInfo.ValidatorUtils,
		nil,
	)
	Require(t, err)
	err = staker.Initialize(ctx)
	Require(t, err)
	return staker, l2Node.BlockValidator, cleanup
}

func waitFor(
	t *testing.T,
	ctxinput context.Context,
	condition func() bool,
) error {
	return waitForWith(t, ctxinput, 30*time.Second, time.Second, condition)
}

func waitForWith(
	t *testing.T,
	ctxinput context.Context,
	timeout time.Duration,
	interval time.Duration,
	condition func() bool,
) error {
	ctx, cancel := context.WithTimeout(ctxinput, timeout)
	defer cancel()

	for {
		if condition() {
			return nil
		}
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func TestEspressoE2E(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanValNode := createValidationNode(ctx, t, false)
	defer cleanValNode()

	builder, cleanup := createL1ValidatorPosterNode(ctx, t)
	defer cleanup()
	node := builder.L2

	err := waitFor(t, ctx, func() bool {
		if e := exec.Command(
			"curl",
			"-X",
			"POST",
			"-H",
			"Content-Type: application/json",
			"-d",
			"{'jsonrpc':'2.0','id':45678,'method':'eth_chainId','params':[]}",
			"http://localhost:8545",
		).Run(); e != nil {
			return false
		}
		return true
	})
	Require(t, err)

	l2Node, l2Info, cleanL2Node := createL2Node(ctx, t, "http://127.0.0.1:50000")
	defer cleanL2Node()

	cleanEspresso := runEspresso(t, ctx)
	defer cleanEspresso()

	// wait for the commitment task
	err = waitFor(t, ctx, func() bool {
		out, err := exec.Command("curl", "http://127.0.0.1:60000/api/hotshot_contract").Output()
		if err != nil {
			return false
		}
		return len(out) > 0
	})
	Require(t, err)

	// Wait for the initial message
	expected := arbutil.MessageIndex(1)
	err = waitFor(t, ctx, func() bool {
		msgCnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			panic(err)
		}

		validatedCnt := node.ConsensusNode.BlockValidator.Validated(t)
		return msgCnt >= expected && validatedCnt >= expected
	})
	Require(t, err)

	// wait for the latest hotshot block
	err = waitFor(t, ctx, func() bool {
		out, err := exec.Command("curl", "http://127.0.0.1:50000/status/block-height").Output()
		if err != nil {
			return false
		}
		h := 0
		err = json.Unmarshal(out, &h)
		if err != nil {
			return false
		}
		return h > 0
	})
	Require(t, err)

	// Make sure it is a totally new account
	newAccount := "User10"
	l2Info.GenerateAccount(newAccount)
	addr := l2Info.GetAddress(newAccount)
	balance := l2Node.GetBalance(t, addr)
	if balance.Cmp(big.NewInt(0)) > 0 {
		Fatal(t, "empty account")
	}

	// Check if the tx is executed correctly
	transfer_amount := big.NewInt(1e16)
	tx := l2Info.PrepareTx("Faucet", newAccount, 3e7, transfer_amount, nil)
	err = l2Node.Client.SendTransaction(ctx, tx)
	Require(t, err)

	err = waitFor(t, ctx, func() bool {
		balance := l2Node.GetBalance(t, addr)
		log.Info("waiting for balance", "addr", addr, "balance", balance)
		return balance.Cmp(transfer_amount) >= 0
	})
	Require(t, err)

	// Remember the number of messages
	msgCnt, err := node.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	// Wait for the number of validated messages to catch up
	err = waitFor(t, ctx, func() bool {
		validatedCnt := node.ConsensusNode.BlockValidator.Validated(t)
		log.Info("waiting for validation", "validatedCnt", validatedCnt, "msgCnt", msgCnt)
		return validatedCnt >= msgCnt
	})
	Require(t, err)

	// TODO: investigate why this now fails
	// hostIo, err := ospgen.NewOneStepProverHostIo(common.HexToAddress(hostIoAddress), builder.L1.Client)
	// Require(t, err)
	// actualCommitment, err := hostIo.GetHotShotCommitment(&bind.CallOpts{}, big.NewInt(1))
	// Require(t, err)
	// commitmentBytes := actualCommitment.Bytes()
	// if len(commitmentBytes) != 32 {
	// 	t.Fatal("failed to read hotshot via hostio contract, length is not 32")
	// }
	// empty := actualCommitment.Cmp(big.NewInt(0)) == 0
	// if empty {
	// 	t.Fatal("failed to read hotshot via hostio contract, empty")
	// }
	// log.Info("Read hotshot commitment via hostio contract successfully", "height", 1, "commitment", commitmentBytes)

	// lastValidatedInfo, err := node.ConsensusNode.BlockValidator.ReadLastValidatedInfo()
	// Require(t, err)
	// incorrectHeight := lastValidatedInfo.GlobalState.HotShotHeight
	// log.Info("setting incorrect hotshot height!", "height", incorrectHeight)
	// validated := node.ConsensusNode.BlockValidator.Validated(t)

	// goodStaker, blockValidatorA, cleanA := createStaker(ctx, t, builder, 0)
	// defer cleanA()
	// badStaker, blockValidatorB, cleanB := createStaker(ctx, t, builder, incorrectHeight)
	// defer cleanB()

	// err = waitFor(t, ctx, func() bool {
	// 	validatedA := blockValidatorA.Validated(t)
	// 	validatedB := blockValidatorB.Validated(t)
	// 	shouldValidated := validated
	// 	condition := validatedA >= shouldValidated && validatedB >= shouldValidated
	// 	if !condition {
	// 		log.Info("waiting for stakers to catch up the incorrect hotshot height", "stakerA", validatedA, "stakerB", validatedB, "target", shouldValidated)
	// 	}
	// 	return condition
	// })
	// Require(t, err)
	// validatorUtils, err := rollupgen.NewValidatorUtils(builder.L2.ConsensusNode.DeployInfo.ValidatorUtils, builder.L1.Client)
	// Require(t, err)
	// goodOpts := builder.L1Info.GetDefaultCallOpts("Staker1", ctx)
	// badOpts := builder.L1Info.GetDefaultCallOpts("Staker2", ctx)
	// i := 0
	// err = waitFor(t, ctx, func() bool {
	// 	log.Info("good staker acts", "step", i)
	// 	txA, err := goodStaker.Act(ctx)
	// 	Require(t, err)
	// 	if txA != nil {
	// 		_, err = builder.L1.EnsureTxSucceeded(txA)
	// 		Require(t, err)
	// 	}

	// 	log.Info("bad staker acts", "step", i)
	// 	txB, err := badStaker.Act(ctx)
	// 	Require(t, err)
	// 	if txB != nil {
	// 		_, err = builder.L1.EnsureTxSucceeded(txB)
	// 		Require(t, err)
	// 	}
	// 	i += 1
	// 	conflict, err := validatorUtils.FindStakerConflict(&bind.CallOpts{}, builder.L2.ConsensusNode.DeployInfo.Rollup, goodOpts.From, badOpts.From, big.NewInt(1024))
	// 	Require(t, err)
	// 	condition := staker.ConflictType(conflict.Ty) == staker.CONFLICT_TYPE_FOUND
	// 	if !condition {
	// 		log.Info("waiting for the conflict")
	// 	}
	// 	return condition
	// })
	// Require(t, err)
	// err = waitForWith(
	// 	t,
	// 	ctx,
	// 	time.Minute*10,
	// 	time.Second*10,
	// 	func() bool {
	// 		log.Info("good staker acts", "step", i)
	// 		txA, err := goodStaker.Act(ctx)
	// 		Require(t, err)
	// 		if txA != nil {
	// 			_, err = builder.L1.EnsureTxSucceeded(txA)
	// 			Require(t, err)
	// 		}

	// 		log.Info("bad staker acts", "step", i)
	// 		txB, err := badStaker.Act(ctx)
	// 		if txB != nil {
	// 			_, err = builder.L1.EnsureTxSucceeded(txB)
	// 			Require(t, err)
	// 		}
	// 		if err != nil {
	// 			ok := strings.Contains(err.Error(), "ERROR_HOTSHOT_COMMITMENT")
	// 			if ok {
	// 				return true
	// 			} else {
	// 				t.Fatal("unexpected err")
	// 			}
	// 		}
	// 		i += 1
	// 		return false

	// 	})
	// Require(t, err)
}
