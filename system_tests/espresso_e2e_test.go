package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"testing"
	"time"

	lightclient "github.com/EspressoSystems/espresso-sequencer-go/light-client"
	lightclientmock "github.com/EspressoSystems/espresso-sequencer-go/light-client-mock"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var workingDir = "./espresso-e2e"

// light client proxy
var lightClientAddress = "0x60571c8f4b52954a24a5e7306d435e951528d963"

var hotShotUrl = "http://127.0.0.1:41000"
var delayThreshold uint64 = 10

var (
	jitValidationPort = 54320
	arbValidationPort = 54321
)

func runEspresso() func() {
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
	stackConf.DBEngine = "leveldb" // TODO Try pebble again in future once iterator race condition issues are fixed

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

func waitFor(
	ctxinput context.Context,
	condition func() bool,
) error {
	return waitForWith(ctxinput, 30*time.Second, time.Second, condition)
}

func waitForWith(
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

func waitForEspressoNode(ctx context.Context) error {
	return waitForWith(ctx, 400*time.Second, 1*time.Second, func() bool {
		out, err := exec.Command("curl", "http://localhost:20000/api/dev-info", "-L").Output()
		if err != nil {
			log.Warn("retry to check the espresso dev node", "err", err)
			return false
		}
		return len(out) > 0
	})
}

func waitForHotShotLiveness(ctx context.Context, lightClientReader *lightclient.LightClientReader) error {
	return waitForWith(ctx, 400*time.Second, 1*time.Second, func() bool {
		log.Info("Waiting for HotShot Liveness")
		_, err := lightClientReader.FetchMerkleRoot(1, nil)
		return err == nil
	})
}

func waitForL1Node(ctx context.Context) error {
	err := waitFor(ctx, func() bool {
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
	if err != nil {
		return err
	}

	// wait for L1 to be totally ready to better simulate real-world environment
	// this is necessary right now to avoid some unknown issues in the dev-node
	// TODO: find a better way
	time.Sleep(10 * time.Second)
	return nil
}

func TestEspressoE2E(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := createL1AndL2Node(ctx, t)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	cleanEspresso := runEspresso()
	defer cleanEspresso()

	// wait for the builder
	err = waitForEspressoNode(ctx)
	Require(t, err)

	l2Node := builder.L2
	l2Info := builder.L2Info

	// Wait for the initial message
	expected := arbutil.MessageIndex(1)
	err = waitFor(ctx, func() bool {
		msgCnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			panic(err)
		}

		validatedCnt := l2Node.ConsensusNode.BlockValidator.Validated(t)
		return msgCnt >= expected && validatedCnt >= expected
	})
	Require(t, err)

	// wait for the latest hotshot block
	err = waitFor(ctx, func() bool {
		out, err := exec.Command("curl", "http://127.0.0.1:41000/status/block-height", "-L").Output()
		if err != nil {
			return false
		}
		h := 0
		err = json.Unmarshal(out, &h)
		if err != nil {
			return false
		}
		// Wait for the hotshot to generate some blocks to better simulate the real-world environment.
		// Chosen based on intuition; no empirical data supports this value.
		return h > 10
	})
	Require(t, err)

	// make light client reader

	lightClientReader, err := lightclient.NewLightClientReader(common.HexToAddress(lightClientAddress), builder.L1.Client)
	Require(t, err)
	// wait for hotshot liveness

	err = waitForHotShotLiveness(ctx, lightClientReader)
	Require(t, err)

	// Check if the tx is executed correctly
	err = checkTransferTxOnL2(t, ctx, l2Node, "User10", l2Info)
	Require(t, err)

	// Remember the number of messages
	var msgCnt arbutil.MessageIndex
	err = waitFor(ctx, func() bool {
		cnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)
		msgCnt = cnt
		log.Info("waiting for message count", "cnt", msgCnt)
		return msgCnt >= 2
	})
	Require(t, err)

	// Wait for the number of validated messages to catch up
	err = waitForWith(ctx, 8*time.Minute, 5*time.Second, func() bool {
		validatedCnt := l2Node.ConsensusNode.BlockValidator.Validated(t)
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

	err = waitForWith(ctx, 180*time.Second, 2*time.Second, func() bool {
		balance2 := l2Node.GetBalance(t, addr2)
		log.Info("waiting for balance", "account", newAccount2, "addr", addr2, "balance", balance2)
		return balance2.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	// Pause l1 height and verify that the escape hatch is working
	checkStaker := os.Getenv("E2E_SKIP_ESCAPE_HATCH_TEST")
	if checkStaker == "" {
		log.Info("Checking the escape hatch")
		// Start to check the escape hatch
		address := common.HexToAddress(lightClientAddress)

		txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)

		// Freeze the l1 height
		err := lightclientmock.FreezeL1Height(t, builder.L1.Client, address, &txOpts)
		log.Info("waiting for light client to report hotshot is down")
		Require(t, err)
		err = waitForWith(ctx, 10*time.Minute, 1*time.Second, func() bool {
			isLive, err := lightclientmock.IsHotShotLive(t, builder.L1.Client, address, uint64(delayThreshold))
			if err != nil {
				return false
			}
			return !isLive
		})
		Require(t, err)
		log.Info("light client has reported that hotshot is down")
		// Wait for the switch to be totally finished
		currMsg, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
		Require(t, err)
		log.Info("waiting for message count", "currMsg", currMsg)
		var validatedMsg arbutil.MessageIndex
		err = waitForWith(ctx, 6*time.Minute, 60*time.Second, func() bool {
			validatedCnt := builder.L2.ConsensusNode.BlockValidator.Validated(t)
			log.Info("Validation status", "validatedCnt", validatedCnt, "msgCnt", msgCnt)
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

		err = waitForWith(ctx, 3*time.Minute, 20*time.Second, func() bool {
			validated := builder.L2.ConsensusNode.BlockValidator.Validated(t)
			return validated >= validatedMsg
		})
		Require(t, err)

		// Unfreeze the l1 height
		err = lightclientmock.UnfreezeL1Height(t, builder.L1.Client, address, &txOpts)
		Require(t, err)

		// Check if the validated count is increasing
		err = waitForWith(ctx, 3*time.Minute, 20*time.Second, func() bool {
			validated := builder.L2.ConsensusNode.BlockValidator.Validated(t)
			return validated >= validatedMsg+10
		})
		Require(t, err)
	}
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

	return waitForWith(ctx, time.Second*300, time.Second*1, func() bool {
		balance := l2Node.GetBalance(t, addr)
		log.Info("waiting for balance", "account", account, "addr", addr, "balance", balance)
		if balance.Cmp(transferAmount) >= 0 {
			log.Info("target balance reached", "account", account, "addr", addr, "balance", balance)
		}
		return true
	})
}
