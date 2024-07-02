package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	lightclientmock "github.com/EspressoSystems/espresso-sequencer-go/light-client-mock"
	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/validatorwallet"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var workingDir = "./espresso-e2e"

// light client proxy
var lightClientAddress = "0xb075b82c7a23e0994df4793422a1f03dbcf9136f"

var hotShotUrl = "http://127.0.0.1:41000"
var delayThreshold = 10

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
	invocation := []string{"compose", "up", "-d", "--build"}
	nodes := []string{
		"espresso-dev-node",
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

func createL2Node(ctx context.Context, t *testing.T, hotshot_url string, builder *NodeBuilder) (*TestClient, info, func()) {
	nodeConfig := arbnode.ConfigDefaultL1Test()
	builder.takeOwnership = false
	nodeConfig.BatchPoster.Enable = false
	nodeConfig.BlockValidator.Enable = false
	nodeConfig.DelayedSequencer.Enable = true
	nodeConfig.Sequencer = true
	nodeConfig.Espresso = true
	builder.execConfig.Sequencer.LightClientAddress = lightClientAddress
	builder.execConfig.Sequencer.SwitchPollInterval = 10 * time.Second
	builder.execConfig.Sequencer.SwitchDelayThreshold = uint64(delayThreshold)
	builder.execConfig.Sequencer.Enable = true
	builder.execConfig.Sequencer.Espresso = true
	builder.execConfig.Sequencer.EspressoNamespace = builder.chainConfig.ChainID.Uint64()
	builder.execConfig.Sequencer.HotShotUrl = hotshot_url

	builder.chainConfig.ArbitrumChainParams.EnableEspresso = true

	nodeConfig.Feed.Output.Enable = true
	nodeConfig.Feed.Output.Addr = "0.0.0.0"
	nodeConfig.Feed.Output.Enable = true
	nodeConfig.Feed.Output.Port = fmt.Sprintf("%d", broadcastPort)

	client, cleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig})
	return client, builder.L2Info, cleanup
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
	node, err := valnode.CreateValidationNode(configFetcher, stack, nil)
	Require(t, err)

	err = stack.Start()
	Require(t, err)

	err = node.Start(ctx)
	Require(t, err)

	go func() {
		<-ctx.Done()
		node.GetExec().Stop()
		stack.Close()
	}()

	return func() {
		node.GetExec().Stop()
		stack.Close()
	}

}

func createL1ValidatorPosterNode(ctx context.Context, t *testing.T, hotshotUrl string) (*NodeBuilder, func()) {
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
	builder.nodeConfig.BatchPoster.ErrorDelay = 5 * time.Second
	builder.nodeConfig.BatchPoster.MaxSize = 41
	builder.nodeConfig.BatchPoster.PollInterval = 10 * time.Second
	builder.nodeConfig.BatchPoster.MaxDelay = -1000 * time.Hour
	builder.nodeConfig.BatchPoster.LightClientAddress = lightClientAddress
	builder.nodeConfig.BatchPoster.HotShotUrl = hotshotUrl
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.ValidationPoll = 2 * time.Second
	builder.nodeConfig.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", arbValidationPort)
	builder.nodeConfig.BlockValidator.LightClientAddress = lightClientAddress
	builder.nodeConfig.BlockValidator.Espresso = true
	builder.nodeConfig.DelayedSequencer.Enable = false

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
	builder.L1Info.GenerateAccount("Staker3")
	builder.L1.TransferBalance(t, "Faucet", "Staker3", big.NewInt(9e18), builder.L1Info)

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
	staker3Addr := builder.L1Info.GetAddress("Staker3")
	setValidatorCalldata, err := rollupABI.Pack("setValidator", []common.Address{staker1Addr, staker2Addr, staker3Addr}, []bool{true, true, true})
	Require(t, err, "unable to generate setValidator calldata")
	tx, err = upgradeExecutor.ExecuteCall(&deployAuth, builder.L2.ConsensusNode.DeployInfo.Rollup, setValidatorCalldata)
	Require(t, err, "unable to set validators")
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	return builder, cleanup
}

func createStaker(
	ctx context.Context,
	t *testing.T,
	builder *NodeBuilder,
	incorrectHeight uint64,
	account string,
	f func(*validator.ValidationInput)) (*staker.Staker, *staker.BlockValidator, func()) {
	config := arbnode.ConfigDefaultL1Test()
	builder.takeOwnership = false
	config.Sequencer = false
	config.DelayedSequencer.Enable = false
	config.BatchPoster.Enable = false
	config.Staker.Enable = false
	config.BlockValidator.Enable = true

	builder.chainConfig.ArbitrumChainParams.EnableEspresso = true
	builder.execConfig.Sequencer.Enable = false

	config.BlockValidator.ValidationPoll = 2 * time.Second
	config.BlockValidator.LightClientAddress = lightClientAddress
	config.BlockValidator.Espresso = true
	config.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", arbValidationPort)
	testClient, cleanup := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: config})
	l2Node := testClient.ConsensusNode

	auth := builder.L1Info.GetDefaultTransactOpts(account, ctx)

	parentChainID, err := builder.L1.Client.ChainID(ctx)
	Require(t, err)
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(l2Node.ArbDB, storage.StakerPrefix),
		l2Node.L1Reader,
		&auth,
		NewFetcherFromConfig(config),
		nil,
		parentChainID,
	)
	Require(t, err)
	wallet, err := validatorwallet.NewEOA(dp, l2Node.DeployInfo.Rollup, l2Node.L1Reader.Client(), func() uint64 { return 50000 })
	Require(t, err)

	if incorrectHeight > 0 {
		l2Node.StatelessBlockValidator.DebugEspresso_SetTrigger(t, incorrectHeight, f)
		l2Node.BlockValidator.DebugEspresso_SetTrigger(t, incorrectHeight, f)
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

// We run one L1 node, two L2 nodes and the espresso containers in this function.
func runNodes(ctx context.Context, t *testing.T) (*NodeBuilder, *TestClient, *BlockchainTestInfo, func()) {

	cleanValNode := createValidationNode(ctx, t, false)

	builder, cleanup := createL1ValidatorPosterNode(ctx, t, hotShotUrl)

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

	cleanEspresso := runEspresso(t, ctx)

	// wait for the builder
	err = waitForWith(t, ctx, 400*time.Second, 1*time.Second, func() bool {
		out, err := exec.Command("curl", "http://localhost:41000/availability/block/10", "-L").Output()
		if err != nil {
			log.Warn("retry to check the builder", "err", err)
			return false
		}
		return len(out) > 0
	})
	Require(t, err)

	l2Node, l2Info, cleanL2Node := createL2Node(ctx, t, hotShotUrl, builder)

	return builder, l2Node, l2Info, func() {
		cleanL2Node()
		cleanup()
		cleanValNode()
		cleanEspresso()
	}
}

func TestEspressoE2E(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, l2Node, l2Info, cleanup := runNodes(ctx, t)
	defer cleanup()
	node := builder.L2

	// Wait for the initial message
	expected := arbutil.MessageIndex(1)
	err := waitFor(t, ctx, func() bool {
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
		out, err := exec.Command("curl", "http://127.0.0.1:41000/status/block-height", "-L").Output()
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

	// Check if the tx is executed correctly
	err = checkTransferTxOnL2(t, ctx, l2Node, "User10", l2Info)
	Require(t, err)

	// Remember the number of messages
	var msgCnt arbutil.MessageIndex
	err = waitFor(t, ctx, func() bool {
		cnt, err := node.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)
		msgCnt = cnt
		log.Info("waiting for message count", "cnt", msgCnt)
		return msgCnt > 6
	})
	Require(t, err)

	// Wait for the number of validated messages to catch up
	err = waitForWith(t, ctx, 360*time.Second, 5*time.Second, func() bool {
		validatedCnt := node.ConsensusNode.BlockValidator.Validated(t)
		log.Info("waiting for validation", "validatedCnt", validatedCnt, "msgCnt", msgCnt)
		return validatedCnt >= msgCnt
	})
	Require(t, err)

	newAccount2 := "User11"
	l2Info.GenerateAccount(newAccount2)
	addr2 := l2Info.GetAddress(newAccount2)

	// Transfer via the delayed inbox
	delayedTx := l2Info.PrepareTx("Owner", newAccount2, 3e7, transferAmount, nil)
	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, delayedTx, builder.L1Info, "Faucet", 100000),
	})
	err = waitForWith(t, ctx, 180*time.Second, 2*time.Second, func() bool {
		balance2 := l2Node.GetBalance(t, addr2)
		log.Info("waiting for balance", "account", newAccount2, "addr", addr2, "balance", balance2)
		return balance2.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	incorrectHeight := uint64(10)

	goodStaker, blockValidatorA, cleanA := createStaker(ctx, t, builder, 0, "Staker1", nil)
	defer cleanA()
	badStaker1, blockValidatorB, cleanB := createStaker(ctx, t, builder, incorrectHeight, "Staker2", func(input *validator.ValidationInput) {
		log.Info("previousinput", "input", input.HotShotCommitment)
		input.HotShotCommitment = espressoTypes.Commitment{}
		log.Info("afterinput", "input", input.HotShotCommitment)
	})
	defer cleanB()
	badStaker2, blockValidatorC, cleanC := createStaker(ctx, t, builder, incorrectHeight, "Staker3", func(input *validator.ValidationInput) {
		input.HotShotLiveness = !input.HotShotLiveness
	})
	defer cleanC()

	err = waitForWith(t, ctx, 240*time.Second, 1*time.Second, func() bool {
		validatedA := blockValidatorA.Validated(t)
		validatedB := blockValidatorB.Validated(t)
		validatorC := blockValidatorC.Validated(t)
		shouldValidated := arbutil.MessageIndex(incorrectHeight - 1)
		condition := validatedA >= shouldValidated && validatedB >= shouldValidated && validatorC >= shouldValidated
		if !condition {
			log.Info("waiting for stakers to catch up the incorrect hotshot height", "stakerA", validatedA, "stakerB", validatedB, "target", shouldValidated)
		}
		return condition
	})
	Require(t, err)
	validatorUtils, err := rollupgen.NewValidatorUtils(builder.L2.ConsensusNode.DeployInfo.ValidatorUtils, builder.L1.Client)
	Require(t, err)
	goodOpts := builder.L1Info.GetDefaultCallOpts("Staker1", ctx)
	badOpts1 := builder.L1Info.GetDefaultCallOpts("Staker2", ctx)
	badOpts2 := builder.L1Info.GetDefaultCallOpts("Staker3", ctx)
	i := 0
	err = waitForWith(t, ctx, 60*time.Second, 2*time.Second, func() bool {
		log.Info("good staker acts", "step", i)
		txA, err := goodStaker.Act(ctx)
		Require(t, err)
		if txA != nil {
			_, err = builder.L1.EnsureTxSucceeded(txA)
			Require(t, err)
		}

		log.Info("bad staker1 acts", "step", i)
		txB, err := badStaker1.Act(ctx)
		Require(t, err)
		if txB != nil {
			_, err = builder.L1.EnsureTxSucceeded(txB)
			Require(t, err)
		}

		log.Info("bad staker2 acts", "step", i)
		txC, err := badStaker2.Act(ctx)
		Require(t, err)
		if txC != nil {
			_, err = builder.L1.EnsureTxSucceeded(txC)
			Require(t, err)
		}

		i += 1
		conflict1, err := validatorUtils.FindStakerConflict(nil, builder.L2.ConsensusNode.DeployInfo.Rollup, goodOpts.From, badOpts1.From, big.NewInt(1024))
		Require(t, err)
		conflict2, err := validatorUtils.FindStakerConflict(nil, builder.L2.ConsensusNode.DeployInfo.Rollup, goodOpts.From, badOpts2.From, big.NewInt(1024))
		Require(t, err)
		condition := staker.ConflictType(conflict1.Ty) == staker.CONFLICT_TYPE_FOUND && staker.ConflictType(conflict2.Ty) == staker.CONFLICT_TYPE_FOUND
		if !condition {
			log.Info("waiting for the conflict")
		}
		return condition
	})
	Require(t, err)

	// The following tests are very time-consuming and, given that the related code
	// does not change often, it's not necessary to run them every time.
	// Note: If you are modifying the smart contracts, staker-related code or doing overhaul,
	// set the E2E_CHECK_STAKER env variable to any non-empty string to run the check.

	checkStaker := os.Getenv("E2E_CHECK_STAKER")
	if checkStaker == "" {
		log.Info("Checking the escape hatch")
		// Start to check the escape hatch
		address := common.HexToAddress(lightClientAddress)

		txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
		// Freeze the l1 height
		err := lightclientmock.FreezeL1Height(t, builder.L1.Client, address, &txOpts)
		Require(t, err)

		err = waitForWith(t, ctx, 1*time.Minute, 1*time.Second, func() bool {
			isLive, err := lightclientmock.IsHotShotLive(t, builder.L1.Client, address, uint64(delayThreshold))
			if err != nil {
				return false
			}
			return !isLive
		})
		Require(t, err)

		// Wait for the switch to be totally finished
		currMsg, err := node.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)
		var validatedMsg arbutil.MessageIndex
		err = waitForWith(t, ctx, 6*time.Minute, 60*time.Second, func() bool {

			validatedCnt := node.ConsensusNode.BlockValidator.Validated(t)
			if validatedCnt >= currMsg {
				validatedMsg = validatedCnt
				return true
			}
			return false
		})
		Require(t, err)

		err = checkTransferTxOnL2(t, ctx, l2Node, "User12", l2Info)
		Require(t, err)
		err = checkTransferTxOnL2(t, ctx, l2Node, "User13", l2Info)
		Require(t, err)

		err = waitForWith(t, ctx, 3*time.Minute, 20*time.Second, func() bool {
			validated := node.ConsensusNode.BlockValidator.Validated(t)
			return validated >= validatedMsg
		})
		Require(t, err)

		// Unfreeze the l1 height
		err = lightclientmock.UnfreezeL1Height(t, builder.L1.Client, address, &txOpts)
		Require(t, err)

		// Check if the validated count is increasing
		err = waitForWith(t, ctx, 3*time.Minute, 20*time.Second, func() bool {
			validated := node.ConsensusNode.BlockValidator.Validated(t)
			return validated >= validatedMsg+10
		})
		Require(t, err)

		return
	}
	err = waitForWith(
		t,
		ctx,
		time.Minute*20,
		time.Second*5,
		func() bool {
			log.Info("good staker acts", "step", i)
			txA, err := goodStaker.Act(ctx)
			if err != nil {
				return false
			}
			if txA != nil {
				_, err = builder.L1.EnsureTxSucceeded(txA)
				Require(t, err)
			}

			log.Info("bad staker acts", "step", i)
			txB, err := badStaker1.Act(ctx)
			if txB != nil && err == nil {
				_, err = builder.L1.EnsureTxSucceeded(txB)
				Require(t, err)
			} else if err != nil {
				ok := strings.Contains(err.Error(), "ERROR_HOTSHOT_COMMITMENT")
				if ok {
					return true
				} else {
					fmt.Println(err.Error())
					t.Fatal("unexpected err")
				}
			}
			i += 1
			return false

		})
	Require(t, err)
}

func checkTransferTxOnL2(
	t *testing.T,
	ctx context.Context,
	l2Node *TestClient,
	account string,
	l2Info *BlockchainTestInfo,
) error {
	l2Info.GenerateAccount(account)
	transferAmount := big.NewInt(1e16)
	tx := l2Info.PrepareTx("Faucet", account, 3e7, transferAmount, nil)

	err := l2Node.Client.SendTransaction(ctx, tx)
	if err != nil {
		return err
	}

	addr := l2Info.GetAddress(account)

	return waitForWith(t, ctx, time.Second*300, time.Second*1, func() bool {
		balance := l2Node.GetBalance(t, addr)
		log.Info("waiting for balance", "account", account, "addr", addr, "balance", balance)
		return balance.Cmp(transferAmount) >= 0
	})
}
