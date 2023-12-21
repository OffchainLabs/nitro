package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"os/exec"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

var workingDir = "./espresso-e2e"

func runEspresso(t *testing.T, ctx context.Context) func() {
	shutdown := func() {
		p := exec.Command("docker-compose", "down")
		p.Dir = workingDir
		err := p.Run()
		if err != nil {
			panic(err)
		}
	}

	shutdown()
	invocation := []string{"up", "-d"}
	nodes := []string{
		"orchestrator",
		"da-server",
		"consensus-server",
		"espresso-sequencer0",
		"espresso-sequencer1",
		"commitment-task",
	}
	invocation = append(invocation, nodes...)
	procees := exec.Command("docker-compose", invocation...)
	procees.Dir = workingDir

	go func() {
		if err := procees.Run(); err != nil {
			log.Error(err.Error())
			panic(err)
		}
	}()
	return shutdown
}

func createL1ValidatorPosterNode(ctx context.Context, t *testing.T) (*TestClient, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l1StackConfig.HTTPPort = 8545
	builder.l1StackConfig.WSPort = 8546
	builder.l1StackConfig.HTTPHost = "0.0.0.0"
	builder.l1StackConfig.HTTPVirtualHosts = []string{"*"}
	builder.l1StackConfig.WSHost = "0.0.0.0"
	builder.l1StackConfig.DataDir = t.TempDir()
	builder.l1StackConfig.WSModules = append(builder.l1StackConfig.WSModules, "eth")

	builder.nodeConfig.Feed.Input.URL = []string{fmt.Sprintf("ws://127.0.0.1:%d", broadcastPort)}
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", validationPort)
	builder.nodeConfig.BlockValidator.HotShotAddress = "0x217788c286797d56cd59af5e493f3699c39cbbe8"
	builder.nodeConfig.BlockValidator.Espresso = true
	cleanup := builder.Build(t)

	mnemonic := "indoor dish desk flag debris potato excuse depart ticket judge file exit"
	err := builder.L1Info.GenerateAccountWithMnemonic("CommitmentTask", mnemonic, 5)
	if err != nil {
		panic(err)
	}
	builder.L1.TransferBalance(t, "Faucet", "CommitmentTask", big.NewInt(9e18), builder.L1Info)

	return builder.L2, cleanup
}

func TestEspressoE2E(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanValNode := createValidationNode(ctx, t)
	defer cleanValNode()

	node, cleanup := createL1ValidatorPosterNode(ctx, t)
	defer cleanup()

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
}
