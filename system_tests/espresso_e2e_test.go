package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os/exec"
	"testing"
	"time"

	lightclient "github.com/EspressoSystems/espresso-network-go/light-client"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var workingDir = "./espresso-e2e"

// light client proxy
var lightClientAddress = "0x0f1f89aaf1c6fdb7ff9d361e4388f5f3997f12a8"

var hotShotUrl = "http://127.0.0.1:41000"

var (
	jitValidationPort = 54320
	arbValidationPort = 54321
)

func runEspresso() func() {
	shutdown := func() {
		p := exec.Command("docker", "compose", "down", "--volumes")
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
	return waitForWith(ctx, 3*time.Minute, 1*time.Second, func() bool {
		out, err := exec.Command("curl", "http://localhost:20000/api/dev-info", "-L").Output()
		if err != nil {
			log.Warn("retry to check the espresso dev node", "err", err)
			return false
		}
		return len(out) > 0
	})
}

func waitForHotShotLiveness(ctx context.Context, lightClientReader *lightclient.LightClientReader) error {
	return waitForWith(ctx, 500*time.Second, 1*time.Second, func() bool {
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
			log.Warn("retry to check the l1 node", "err", e)
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

	builder, cleanup := createL1AndL2Node(ctx, t, true, false)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	shutdown := runEspresso()
	defer shutdown()

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

	// Test that if espresso node is down, the transaction will be resubmitted once it is back online
	newAccount3 := "User12"
	l2Info.GenerateAccount(newAccount3)
	addr3 := l2Info.GetAddress(newAccount3)
	tx3 := l2Info.PrepareTx("Faucet", newAccount3, 3e7, transferAmount, nil)
	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, tx3, builder.L1Info, "Faucet", 100000),
	})

	// Wait for 1 second to make sure txn is submitted to Espresso
	// but shut down before it can be finalized
	time.Sleep(1 * time.Second)

	log.Info("Pausing espresso node")
	pauseEspresso := func() {
		p := exec.Command("docker", "compose", "pause")
		p.Dir = workingDir
		err := p.Run()
		if err != nil {
			panic(err)
		}
		// Disconnect the container from the network to ensure requests to the dev node
		// don't just hang but actually fail.
		p = exec.Command(
			"docker",
			"network",
			"disconnect",
			"espresso-e2e_default",
			"espresso-e2e-espresso-dev-node-1",
		)
		err = p.Run()
		if err != nil {
			panic(err)
		}

	}
	pauseEspresso()

	log.Info("Waiting for 1 minute before resuming espresso node")
	time.Sleep(1 * time.Minute)

	log.Info("Resuming espresso node")
	unpauseEspresso := func() {
		// reconnect the network first
		p := exec.Command(
			"docker",
			"network",
			"connect",
			"espresso-e2e_default",
			"espresso-e2e-espresso-dev-node-1",
		)
		err := p.Run()
		if err != nil {
			panic(err)
		}
		// resume the dev node
		p = exec.Command("docker", "compose", "unpause")
		p.Dir = workingDir
		err = p.Run()
		if err != nil {
			panic(err)
		}
	}
	unpauseEspresso()

	err = waitForEspressoNode(ctx)
	Require(t, err)

	// Wait for the L2 chain to catch up.
	err = waitForWith(ctx, 180*time.Second, 2*time.Second, func() bool {
		balance3 := l2Node.GetBalance(t, addr3)
		log.Info("waiting for balance in", "account", newAccount3, "addr", addr3, "balance", balance3)
		return balance3.Cmp(transferAmount) >= 0
	})
	Require(t, err)

	// Try submitting the another transaction to make sure the transaction is submitted
	// after espresso processes the resubmitted transaction
	tx4 := l2Info.PrepareTx("Faucet", newAccount3, 3e7, transferAmount, nil)

	builder.L1.SendWaitTestTransactions(t, []*types.Transaction{
		WrapL2ForDelayed(t, tx4, builder.L1Info, "Faucet", 100000),
	})

	err = waitForWith(ctx, 180*time.Second, 2*time.Second, func() bool {
		balance4 := l2Node.GetBalance(t, addr3)
		log.Info("waiting for balance", "account", newAccount3, "addr", addr3, "balance", balance4)
		return balance4.Cmp((&big.Int{}).Add(transferAmount, transferAmount)) >= 0
	})
	Require(t, err)

	// Now send the transaction for message pos 0, the message position 0 should have not been sent to espresso
	// because its a genesis message which originates on L1
	fetcher := func(pos arbutil.MessageIndex) ([]byte, error) {
		msg, err := l2Node.ConsensusNode.TxStreamer.GetMessage(0)
		Require(t, err)
		b, err := rlp.EncodeToBytes(msg)
		Require(t, err)
		return b, err
	}

	payload, _ := arbutil.BuildRawHotShotPayload([]arbutil.MessageIndex{0}, fetcher, 900*1024)
	payload, err = arbutil.SignHotShotPayload(payload, func([]byte) ([]byte, error) {
		return []byte{}, nil
	})
	Require(t, err)

	// Submit the transaction to hotshot
	txhash, err := l2Node.ConsensusNode.TxStreamer.ResubmitEspressoTransactions(ctx, arbutil.SubmittedEspressoTx{Hash: "", Pos: []arbutil.MessageIndex{0}, Payload: payload})
	Require(t, err)
	// Check if the txHash is already finalized in hotshot
	// curl hotshot availability endpoint and this transaction should not be in the response
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:41000/availability/transaction/hash/%s", txhash))
	Require(t, err)
	if resp.StatusCode == 200 {
		t.Fatal("Transaction should not be in the response")
	}
}

func TestEspressoWithBlobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create L1 and L2 nodes with blobs enabled
	builder, cleanup := createL1AndL2Node(ctx, t, true, true)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	shutdown := runEspresso()
	defer shutdown()

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
		return msgCnt >= expected
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

	// Check that the batch sent is greater than 1
	err = waitForWith(ctx, 8*time.Minute, 5*time.Second, func() bool {
		// Check the sequencer inbox contract

		sequencerInbox, err := bridgegen.NewSequencerInbox(builder.L1Info.GetAddress("SequencerInbox"), builder.L1.Client)
		Require(t, err)

		batchCount, err := sequencerInbox.BatchCount(&bind.CallOpts{Context: ctx})
		Require(t, err)
		return batchCount.Uint64() > 1
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

	return waitForWith(ctx, time.Second*300, time.Second*1, func() bool {
		balance := l2Node.GetBalance(t, addr)
		log.Info("waiting for balance", "account", account, "addr", addr, "balance", balance)
		if balance.Cmp(transferAmount) >= 0 {
			log.Info("target balance reached", "account", account, "addr", addr, "balance", balance)
			return true
		}
		return false
	})
}
