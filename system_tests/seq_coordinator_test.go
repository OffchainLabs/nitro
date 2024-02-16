// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func initRedisForTest(t *testing.T, ctx context.Context, redisUrl string, nodeNames []string) {
	var priorities string

	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	defer redisClient.Close()

	for _, name := range nodeNames {
		priorities = priorities + name + ","
		redisClient.Del(ctx, redisutil.WANTS_LOCKOUT_KEY_PREFIX+name)
	}
	priorities = priorities[:len(priorities)-1] // remove last ","
	Require(t, redisClient.Set(ctx, redisutil.PRIORITIES_KEY, priorities, time.Duration(0)).Err())
	for msg := 0; msg < 1000; msg++ {
		redisClient.Del(ctx, fmt.Sprintf("%s%d", redisutil.MESSAGE_KEY_PREFIX, msg))
	}
	redisClient.Del(ctx, redisutil.CHOSENSEQ_KEY, redisutil.MSG_COUNT_KEY)
}

func TestRedisSeqCoordinatorPriorities(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.takeOwnership = false
	builder.nodeConfig.SeqCoordinator.Enable = true
	builder.nodeConfig.SeqCoordinator.RedisUrl = redisutil.CreateTestRedis(ctx, t)

	l2Info := builder.L2Info

	// stdio protocol makes sure forwarder initialization doesn't fail
	nodeNames := []string{"stdio://A", "stdio://B", "stdio://C", "stdio://D", "stdio://E"}

	testNodes := make([]*TestClient, len(nodeNames))

	// init DB to known state
	initRedisForTest(t, ctx, builder.nodeConfig.SeqCoordinator.RedisUrl, nodeNames)

	createStartNode := func(nodeNum int) {
		builder.nodeConfig.SeqCoordinator.MyUrl = nodeNames[nodeNum]
		builder.L2Info = l2Info
		builder.Build(t)
		testNodes[nodeNum] = builder.L2
	}

	trySequencing := func(nodeNum int) bool {
		node := testNodes[nodeNum].ConsensusNode
		curMsgs, err := node.TxStreamer.GetMessageCountSync(t)
		Require(t, err)
		emptyMessage := arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:        0,
					Poster:      common.Address{},
					BlockNumber: 0,
					Timestamp:   0,
					RequestId:   &common.Hash{},
					L1BaseFee:   common.Big0,
				},
				L2msg: nil,
			},
			DelayedMessagesRead: 1,
		}
		err = node.SeqCoordinator.SequencingMessage(curMsgs, &emptyMessage)
		if errors.Is(err, execution.ErrRetrySequencer) {
			return false
		}
		Require(t, err)
		Require(t, node.TxStreamer.AddMessages(curMsgs, false, []arbostypes.MessageWithMetadata{emptyMessage}))
		return true
	}

	// node(n) has higher prio than node(n+1), so should be impossible for more than one to succeed
	trySequencingEverywhere := func() int {
		succeeded := -1
		for nodeNum, testNode := range testNodes {
			node := testNode.ConsensusNode
			if node == nil {
				continue
			}
			if trySequencing(nodeNum) {
				if succeeded >= 0 {
					t.Fatal("sequnced succeeded in parallel",
						"index1:", succeeded, "debug", testNodes[succeeded].ConsensusNode.SeqCoordinator.DebugPrint(),
						"index2:", nodeNum, "debug", node.SeqCoordinator.DebugPrint(),
						"now", time.Now().UnixMilli())
				}
				succeeded = nodeNum
			}
		}
		return succeeded
	}

	waitForMsgEverywhere := func(msgNum arbutil.MessageIndex) {
		for _, testNode := range testNodes {
			currentNode := testNode.ConsensusNode
			if currentNode == nil {
				continue
			}
			for attempts := 1; ; attempts++ {
				msgCount, err := currentNode.TxStreamer.GetMessageCountSync(t)
				Require(t, err)
				if msgCount >= msgNum {
					break
				}
				if attempts > 10 {
					Fatal(t, "timeout waiting for msg ", msgNum, " debug: ", currentNode.SeqCoordinator.DebugPrint())
				}
				<-time.After(builder.nodeConfig.SeqCoordinator.UpdateInterval / 3)
			}
		}
	}

	var needsStop []*arbnode.Node
	killNode := func(nodeNum int) {
		if nodeNum%3 == 0 {
			testNodes[nodeNum].ConsensusNode.SeqCoordinator.PrepareForShutdown()
			needsStop = append(needsStop, testNodes[nodeNum].ConsensusNode)
		} else {
			testNodes[nodeNum].ConsensusNode.StopAndWait()
		}
		testNodes[nodeNum].ConsensusNode = nil
	}

	nodeForwardTarget := func(nodeNum int) int {
		execNode := testNodes[nodeNum].ExecNode
		fwTarget := execNode.TxPublisher.(*gethexec.TxPreChecker).TransactionPublisher.(*gethexec.Sequencer).ForwardTarget()
		if fwTarget == "" {
			return -1
		}
		for cNum, name := range nodeNames {
			if name == fwTarget {
				return cNum
			}
		}
		t.Fatal("Bad FW target")
		return -2
	}

	messagesPerRound := arbutil.MessageIndex(10)
	currentSequencer := 0
	sequencedMesssages := arbutil.MessageIndex(1) // we start with 1 so messageCountKey will be written

	t.Log("Starting node 0")
	// give node 0 room to set himself primary
	createStartNode(0)

	for attempts := 1; !trySequencing(0); attempts++ {
		if attempts > 10 {
			t.Fatal("failed first sequencing")
		}
		time.Sleep(time.Millisecond * 200)
	}
	sequencedMesssages++

	t.Log("Starting other nodes")

	for i := 1; i < len(testNodes); i++ {
		createStartNode(i)
	}

	addNodes := false

	// remove sequencers one by one

	for {

		// all remaining nodes know which is the chosen one
		for i := currentSequencer + 1; i < len(testNodes); i++ {
			for attempts := 1; nodeForwardTarget(i) != currentSequencer; attempts++ {
				if attempts > 10 {
					t.Fatal("initial forward target not set")
				}
				time.Sleep(time.Millisecond * 100)
			}
		}

		// sequencing succeeds only on the leder
		for i := arbutil.MessageIndex(0); i < messagesPerRound; i++ {
			if sequencer := trySequencingEverywhere(); sequencer != currentSequencer {
				Fatal(t, "unexpected sequencer. expected: ", currentSequencer, " got ", sequencer)
			}
			sequencedMesssages++
		}

		if currentSequencer == len(testNodes)-1 {
			addNodes = true
		}
		if addNodes {
			if currentSequencer == 0 {
				break
			}
			t.Log("adding node")
			currentSequencer--
			createStartNode(currentSequencer)
		} else {
			t.Log("killing node")
			killNode(currentSequencer)
			currentSequencer++
		}

		// cannot sequence until up to date with all messages
		for attempts := 0; ; attempts++ {
			sequencer := trySequencingEverywhere()
			if sequencer == -1 && attempts > 15 {
				Fatal(t, "failed to sequence")
			}
			if sequencer != -1 {
				sequencedMesssages++
			}
			if sequencer == -1 ||
				(addNodes && (sequencer == currentSequencer+1)) {
				time.Sleep(builder.nodeConfig.SeqCoordinator.LockoutDuration / 5)
				continue
			}
			if sequencer == currentSequencer {
				break
			}
			Fatal(t, "unexpected sequencer", "expected", currentSequencer, "got", sequencer, "messages", sequencedMesssages)
		}

		// all nodes get messages
		waitForMsgEverywhere(sequencedMesssages)

		// can sequence after up to date
		for i := arbutil.MessageIndex(0); i < messagesPerRound; i++ {
			sequencer := trySequencingEverywhere()
			if sequencer != currentSequencer {
				Fatal(t, "unexpected sequencer", "expected", currentSequencer, "got", sequencer, "messages", sequencedMesssages)
			}
			sequencedMesssages++
		}

		// all nodes get messages
		waitForMsgEverywhere(sequencedMesssages)
	}

	for nodeNum := range testNodes {
		killNode(nodeNum)
	}
	for _, node := range needsStop {
		node.StopAndWait()
	}

}

func testCoordinatorMessageSync(t *testing.T, successCase bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.SeqCoordinator.Enable = true
	builder.nodeConfig.SeqCoordinator.RedisUrl = redisutil.CreateTestRedis(ctx, t)
	builder.nodeConfig.BatchPoster.Enable = false

	nodeNames := []string{"stdio://A", "stdio://B"}
	initRedisForTest(t, ctx, builder.nodeConfig.SeqCoordinator.RedisUrl, nodeNames)
	builder.nodeConfig.SeqCoordinator.MyUrl = nodeNames[0]

	cleanup := builder.Build(t)
	defer cleanup()

	redisClient, err := redisutil.RedisClientFromURL(builder.nodeConfig.SeqCoordinator.RedisUrl)
	Require(t, err)
	defer redisClient.Close()

	// wait for sequencerA to become master
	for {
		err := redisClient.Get(ctx, redisutil.CHOSENSEQ_KEY).Err()
		if errors.Is(err, redis.Nil) {
			time.Sleep(builder.nodeConfig.SeqCoordinator.UpdateInterval)
			continue
		}
		Require(t, err)
		break
	}

	builder.L2Info.GenerateAccount("User2")

	nodeConfigDup := *builder.nodeConfig
	builder.nodeConfig = &nodeConfigDup

	builder.nodeConfig.SeqCoordinator.MyUrl = nodeNames[1]
	if !successCase {
		builder.nodeConfig.SeqCoordinator.Signer.ECDSA.AcceptSequencer = false
		builder.nodeConfig.SeqCoordinator.Signer.ECDSA.AllowedAddresses = []string{builder.L2Info.GetAddress("User2").Hex()}
	}

	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: builder.nodeConfig})
	defer cleanupB()

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	if successCase {
		_, err = WaitForTx(ctx, testClientB.Client, tx.Hash(), time.Second*5)
		Require(t, err)
		l2balance, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
		Require(t, err)
		if l2balance.Cmp(big.NewInt(1e12)) != 0 {
			t.Fatal("Unexpected balance:", l2balance)
		}
	} else {
		_, err = WaitForTx(ctx, testClientB.Client, tx.Hash(), time.Second)
		if err == nil {
			Fatal(t, "tx received by node with different seq coordinator signing key")
		}
	}
}

func TestRedisSeqCoordinatorMessageSync(t *testing.T) {
	testCoordinatorMessageSync(t, true)
}

func TestRedisSeqCoordinatorWrongKeyMessageSync(t *testing.T) {
	testCoordinatorMessageSync(t, false)
}
