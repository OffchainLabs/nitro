package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/jarcoal/httpmock"
	"github.com/offchainlabs/nitro/arbos/espresso"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var (
	validationPort     = 54320
	broadcastPort      = 9642
	maxHotShotBlock    = 100
	malformedBlockNum  = 5
	firstGoodBlockNum  = 15
	secondGoodBlockNum = 25
)

func espresso_block_txs_generators(t *testing.T, l2Info *BlockchainTestInfo) map[int][][]byte {
	return map[int][][]byte{
		malformedBlockNum: onlyMalformedTxs(t),
		firstGoodBlockNum: userTxs(t, l2Info),
		secondGoodBlockNum: func(t *testing.T) [][]byte {
			// Contains malformed txes, valid transactions and invalid transactions
			r := [][]byte{}
			r = append(r, onlyMalformedTxs(t)...)
			r = append(r, userTxs(t, l2Info)...)
			return r
		}(t),
	}
}

func onlyMalformedTxs(t *testing.T) [][]byte {
	return [][]byte{
		{1, 2, 3},
		{1, 2, 3},
		{4, 5, 6},
		{1, 2, 4},
	}
}

// Two valid transactions and two invalid transactions with invalid nonces
func userTxs(t *testing.T, l2Info *BlockchainTestInfo) [][]byte {
	tx1 := l2Info.PrepareTx("Faucet", "Owner", 3e7, big.NewInt(1e16), nil)
	tx1Bin, err := json.Marshal(tx1)
	if err != nil {
		panic(err)
	}
	tx2 := l2Info.PrepareTx("Owner", "Faucet", 3e7, big.NewInt(1e16), nil)
	tx2Bin, err := json.Marshal(tx2)
	if err != nil {
		panic(err)
	}
	// 2 valid transactions here
	return [][]byte{
		tx1Bin,
		tx1Bin,
		tx2Bin,
		tx2Bin,
	}
}

func createMockHotShot(ctx context.Context, t *testing.T, l2Info *BlockchainTestInfo) func() {
	httpmock.Activate()

	httpmock.RegisterResponder(
		"GET",
		`=~http://127.0.0.1:50000/availability/header/(\d+)`,
		func(req *http.Request) (*http.Response, error) {
			log.Info("GET", "url", req.URL)
			block := uint64(httpmock.MustGetSubmatchAsUint(req, 1))
			header := espresso.Header{
				// Since we don't realize the validation of espresso yet,
				// mock a simple nmt root here
				// See: arbos/espresso/nmt.go
				TransactionsRoot: espresso.NmtRoot{Root: []byte{}},
				Metadata: espresso.Metadata{
					L1Head:    block,
					Timestamp: uint64(time.Now().Unix()),
				},
			}
			return httpmock.NewJsonResponse(200, header)
		})

	generators := espresso_block_txs_generators(t, l2Info)

	httpmock.RegisterResponder(
		"GET",
		`=~http://127.0.0.1:50000/availability/block/(\d+)/namespace/100`,
		func(req *http.Request) (*http.Response, error) {
			txes := []espresso.Transaction{}
			block := int(httpmock.MustGetSubmatchAsInt(req, 1))
			data, ok := generators[block]
			// Since we don't realize the validation of espresso yet,
			// we can mock the proof easily.
			// See: arbos/espresso/nmt.go
			dummyProof, _ := json.Marshal(map[int]int{0: 0})
			if block > maxHotShotBlock {
				// make the debug message cleaner
				return httpmock.NewJsonResponse(404, 0)
			}
			log.Info("GET", "url", req.URL)
			if !ok {
				r := espresso.NamespaceResponse{
					Proof:        (*json.RawMessage)(&dummyProof),
					Transactions: &[]espresso.Transaction{},
				}
				return httpmock.NewJsonResponse(200, r)
			}
			for _, rawTx := range data {
				tx := espresso.Transaction{
					Vm:      100,
					Payload: rawTx,
				}
				txes = append(txes, tx)
			}
			resp := espresso.NamespaceResponse{
				Proof:        (*json.RawMessage)(&dummyProof),
				Transactions: &txes,
			}
			return httpmock.NewJsonResponse(200, resp)
		})

	return httpmock.DeactivateAndReset
}

func createL2Node(ctx context.Context, t *testing.T, hotshot_url string) (*TestClient, info, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.Sequencer = true
	builder.nodeConfig.Espresso = true
	builder.execConfig.Sequencer.Enable = true
	builder.execConfig.Sequencer.Espresso = true
	builder.execConfig.Sequencer.EspressoNamespace = 100
	builder.execConfig.Sequencer.HotShotUrl = hotshot_url

	builder.nodeConfig.Feed.Output.Enable = true
	builder.nodeConfig.Feed.Output.Port = fmt.Sprintf("%d", broadcastPort)

	cleanup := builder.Build(t)
	return builder.L2, builder.L2Info, cleanup
}

func createValidatorAndPosterNode(ctx context.Context, t *testing.T) (*TestClient, func()) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.Feed.Input.URL = []string{fmt.Sprintf("ws://127.0.0.1:%d", broadcastPort)}
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.BlockValidator.ValidationServer.URL = fmt.Sprintf("ws://127.0.0.1:%d", validationPort)
	cleanup := builder.Build(t)
	return builder.L2, cleanup
}

func createValidationNode(ctx context.Context, t *testing.T) func() {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = validationPort
	stackConf.WSModules = []string{server_api.Namespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	valnode.EnsureValidationExposedViaAuthRPC(&stackConf)
	config := &valnode.TestValidationConfig

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

func waitFor(t *testing.T, ctxinput context.Context, condition func() bool) error {
	ctx, cancel := context.WithTimeout(ctxinput, 30*time.Second)
	defer cancel()

	for {
		if condition() {
			return nil
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func TestEspresso(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2Node, l2Info, cleanL2Node := createL2Node(ctx, t, "http://127.0.0.1:50000")
	defer cleanL2Node()

	cleanHotShot := createMockHotShot(ctx, t, l2Info)
	defer cleanHotShot()

	// An initial message for genesis block and every non-empty espresso block
	// should lead to a message
	expectedMsgCnt := 1 + maxHotShotBlock

	err := waitFor(t, ctx, func() bool {
		cnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			panic(err)
		}
		expected := arbutil.MessageIndex(expectedMsgCnt)
		return cnt >= expected
	})
	Require(t, err)

	cleanValNode := createValidationNode(ctx, t)
	defer cleanValNode()

	node, cleanup := createValidatorAndPosterNode(ctx, t)
	defer cleanup()

	// Check the validated message
	err = waitFor(t, ctx, func() bool {
		cnt := node.ConsensusNode.BlockValidator.Validated(t)
		expected := arbutil.MessageIndex(expectedMsgCnt)
		return cnt >= expected
	})
	Require(t, err)

	blockNum, err := l2Node.Client.BlockNumber(ctx)
	Require(t, err)

	if blockNum != uint64(maxHotShotBlock)+1 {
		Fatal(t, "every espresso block should lead to one L2 block", "expected", blockNum, "recieved", blockNum)
	}

	block2, err := l2Node.Client.BlockByNumber(ctx, big.NewInt(int64(firstGoodBlockNum)+1))
	Require(t, err)

	// Every arbitrum block has one internal tx
	if len(block2.Body().Transactions) != 3 {
		Fatal(t, "block ", firstGoodBlockNum+1, " should contain 2 valid transactions")
	}

	block3, err := l2Node.Client.BlockByNumber(ctx, big.NewInt(int64(firstGoodBlockNum)+1))
	Require(t, err)

	if len(block3.Body().Transactions) != 3 {
		Fatal(t, "block", secondGoodBlockNum, " should contain 2 valid transactions")
	}
}
