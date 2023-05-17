package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
)

type ConsensusMock struct {
	Messages map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata
}

func NewConsensusMock() *ConsensusMock {
	return &ConsensusMock{make(map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata)}
}

func (c *ConsensusMock) FindL1BatchForMessage(message arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](1, nil)
}

func (c *ConsensusMock) GetBatchL1Block(seqNum uint64) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](1, nil)
}

func (c *ConsensusMock) SyncProgressMap() containers.PromiseInterface[map[string]interface{}] {
	return containers.NewReadyPromise[map[string]interface{}](make(map[string]interface{}), nil)
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
	resultMap := make(map[arbutil.MessageIndex]*execution.MessageResult)
	for msgNum, msg := range mock.Messages {
		for _, sendMsgNum := range []arbutil.MessageIndex{1, 2, 3, 4, 5} {
			result, err := engine.DigestMessage(sendMsgNum, msg).Await(ctx)
			if sendMsgNum == msgNum {
				Require(t, err)
				resultMap[msgNum] = result
			} else if err == nil {
				Fail(t, "message wrongfully accepted. sendNum: ", sendMsgNum, " msgNum: ", msgNum)
			}
		}
	}
	for msgNum, result := range resultMap {
		curResult, err := engine.DigestMessage(msgNum, mock.Messages[msgNum]).Await(ctx)
		Require(t, err)
		if *curResult != *result {
			Fail(t, "result inconsistent for msg ", msgNum, " old ", *result, " new ", *curResult)
		}
	}
}
