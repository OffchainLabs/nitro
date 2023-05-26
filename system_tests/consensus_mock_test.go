package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/consensus/consensusapi"
	"github.com/offchainlabs/nitro/consensus/consensusclient"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

type ConsensusMock struct {
	Messages map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata
	Results  map[arbutil.MessageIndex]*execution.MessageResult
}

func NewConsensusMock() *ConsensusMock {
	return &ConsensusMock{
		Messages: make(map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata),
		Results:  make(map[arbutil.MessageIndex]*execution.MessageResult),
	}
}

func (c *ConsensusMock) FindL1BatchForMessage(message arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](uint64(message)/2+1, nil)
}

func (c *ConsensusMock) GetBatchL1Block(seqNum uint64) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](1000+uint64(seqNum), nil)
}

func (c *ConsensusMock) SyncProgressMap() containers.PromiseInterface[map[string]interface{}] {
	res := make(map[string]interface{})
	res["hello"] = "world"
	return containers.NewReadyPromise[map[string]interface{}](res, nil)
}

func (c *ConsensusMock) SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise[arbutil.MessageIndex](arbutil.MessageIndex(len(c.Messages)), nil)
}

func (c *ConsensusMock) GetSafeMsgCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise[arbutil.MessageIndex](arbutil.MessageIndex(len(c.Messages)), nil)
}

func (c *ConsensusMock) GetFinalizedMsgCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise[arbutil.MessageIndex](arbutil.MessageIndex(len(c.Messages)), nil)
}

func (c *ConsensusMock) WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, result execution.MessageResult) containers.PromiseInterface[struct{}] {
	c.Messages[pos] = &msgWithMeta
	c.Results[pos] = &result
	return containers.NewReadyPromise[struct{}](struct{}{}, nil)
}

func (c *ConsensusMock) ExpectChosenSequencer() containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise[struct{}](struct{}{}, nil)
}

func TxToDelayedMessage(t *testing.T, info *BlockchainTestInfo, tx *types.Transaction, delayedNum uint64) *arbostypes.L1IncomingMessage {
	txbytes, err := tx.MarshalBinary()
	Require(t, err)
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	reqIdHash := common.BigToHash(new(big.Int).SetUint64(delayedNum))

	return &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{
			Kind:        arbostypes.L1MessageType_L2Message,
			Poster:      info.GetAddress("Owner"),
			BlockNumber: 1,
			Timestamp:   uint64(time.Now().Unix()),
			RequestId:   &reqIdHash,
			L1BaseFee:   big.NewInt(100),
		},
		L2msg: txwrapped,
	}
}

func TestExecutionMessageStore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mock := NewConsensusMock()
	l2Info, _, arbDb, _, blockchain := createL2BlockChain(t, nil, "", params.ArbitrumDevTestChainConfig())
	engine, err := gethexec.NewExecutionEngine(blockchain, arbDb, mock)
	Require(t, err)
	engine.Start(ctx)

	l2Info.GenerateAccount("User")
	tx1 := l2Info.PrepareTx("Owner", "User", l2Info.TransferGas, big.NewInt(1000), nil)
	tx2 := l2Info.PrepareTx("Owner", "User", l2Info.TransferGas, big.NewInt(1000), nil)
	tx3 := l2Info.PrepareTx("Owner", "User", l2Info.TransferGas, big.NewInt(1000), nil)

	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: 1,
		Timestamp:   uint64(time.Now().Unix()),
		RequestId:   nil,
		L1BaseFee:   nil,
	}
	_, err = engine.SequenceTransactions(ctx, header, types.Transactions{tx1}, arbos.NoopSequencingHooks())
	Require(t, err)
	_, err = engine.SequenceTransactions(ctx, header, types.Transactions{tx2}, arbos.NoopSequencingHooks())
	Require(t, err)
	_, err = engine.SequenceDelayedMessage(TxToDelayedMessage(t, l2Info, tx1, 1), 1).Await(ctx)
	Require(t, err)
	_, err = engine.SequenceDelayedMessage(TxToDelayedMessage(t, l2Info, tx2, 2), 2).Await(ctx)
	Require(t, err)
	_, err = engine.SequenceDelayedMessage(TxToDelayedMessage(t, l2Info, tx3, 3), 3).Await(ctx)
	Require(t, err)

	if len(mock.Messages) != 5 {
		Fail(t, "expecting 5 messages, got: ", len(mock.Messages))
	}

	state, err := blockchain.State()
	Require(t, err)
	userBalance := state.GetBalance(l2Info.GetAddress("User"))
	if userBalance.Cmp(big.NewInt(3000)) != 0 {
		Fail(t, "unexpected user balance: ", userBalance)
	}
	for msgNum, result := range mock.Results {
		curResult, err := engine.DigestMessage(msgNum, mock.Messages[msgNum]).Await(ctx)
		Require(t, err)
		if *curResult != *result {
			Fail(t, "result inconsistent for msg ", msgNum, " old ", *result, " new ", *curResult)
		}
	}
	for msgNum, msg := range mock.Messages {
		for _, sendMsgNum := range []arbutil.MessageIndex{1, 2, 3, 4, 5} {
			result, err := engine.DigestMessage(sendMsgNum, msg).Await(ctx)
			if sendMsgNum == msgNum {
				Require(t, err)
				if *result != *mock.Results[msgNum] {
					Fail(t, "result inconsistent for msg ", msgNum, " old ", *result, " new ", *mock.Results[msgNum])
				}
			} else if err == nil {
				Fail(t, "message wrongfully accepted. sendNum: ", sendMsgNum, " msgNum: ", msgNum)
			}
		}
	}
}

func createMockConsensusNode(t *testing.T, ctx context.Context) (*ConsensusMock, *node.Node) {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{consensus.RPCNamespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	Require(t, err)

	mock := NewConsensusMock()
	serverAPI := consensusapi.NewConsensusAPI(mock)

	consAPIs := []rpc.API{{
		Namespace:     consensus.RPCNamespace,
		Version:       "1.0",
		Service:       serverAPI,
		Public:        true,
		Authenticated: false,
	}}
	stack.RegisterAPIs(consAPIs)

	err = stack.Start()
	Require(t, err)

	go func() {
		<-ctx.Done()
		stack.Close()
	}()

	return mock, stack
}

func TestConsensusRPC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock, stack := createMockConsensusNode(t, ctx)

	client := consensusclient.NewClient(StaticFetcherFrom(t, &rpcclient.TestClientConfig), stack)
	err := client.Start(ctx)
	Require(t, err)
	_, err = client.ExpectChosenSequencer().Await(ctx)
	Require(t, err)

	num, err := client.FindL1BatchForMessage(12).Await(ctx)
	Require(t, err)
	if num != 7 {
		Fail(t, "unexpected num:", num)
	}
	num, err = client.GetBatchL1Block(12).Await(ctx)
	Require(t, err)
	if num != 1012 {
		Fail(t, "unexpected num:", num)
	}
	syncmap, err := client.SyncProgressMap().Await(ctx)
	Require(t, err)
	if len(syncmap) != 1 {
		Fail(t, "unexpected syncmap: ", syncmap)
	}
	world, ok := syncmap["hello"].(string)
	if world != "world" {
		Fail(t, "unexpected val: ", world, " ok:", ok)
	}
	testMsg := arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        7,
				Poster:      common.HexToAddress("0x33445"),
				BlockNumber: 99,
				Timestamp:   1976,
				L1BaseFee:   big.NewInt(150),
			},
			L2msg: []byte("mockmessage"),
		},
		DelayedMessagesRead: 7,
	}
	testResult := execution.MessageResult{
		BlockHash: common.HexToHash("0x1122"),
		SendRoot:  common.HexToHash("0x3344"),
	}
	_, err = client.WriteMessageFromSequencer(3, testMsg, testResult).Await(ctx)
	Require(t, err)
	gotMessage := mock.Messages[3]
	if len(mock.Messages) != 1 || gotMessage == nil {
		Fail(t, "unexpected messages ", mock.Messages, " message ", *mock.Messages[3].Message, " delayed ", mock.Messages[3].DelayedMessagesRead)
	}
	if (!gotMessage.Message.Equals(testMsg.Message)) || gotMessage.DelayedMessagesRead != testMsg.DelayedMessagesRead {
		Fail(t, "unexpected message")
	}
	if len(mock.Results) != 1 || *mock.Results[3] != testResult {
		Fail(t, "unexpected results ", mock.Results)
	}
	pos, err := client.SyncTargetMessageCount().Await(ctx)
	Require(t, err)
	if pos != 1 {
		Fail(t, "unexpected pos:", pos)
	}
	pos, err = client.GetSafeMsgCount().Await(ctx)
	Require(t, err)
	if pos != 1 {
		Fail(t, "unexpected pos:", pos)
	}
	pos, err = client.GetFinalizedMsgCount().Await(ctx)
	Require(t, err)
	if pos != 1 {
		Fail(t, "unexpected pos:", pos)
	}
	_, err = client.ExpectChosenSequencer().Await(ctx)
	Require(t, err)
}
