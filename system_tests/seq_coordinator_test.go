// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build redistest
// +build redistest

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
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func initRedisForTest(t *testing.T, ctx context.Context, redisUrl string, nodeNames []string) {
	var priorities string

	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	defer redisClient.Close()

	for _, name := range nodeNames {
		priorities = priorities + name + ","
		redisClient.Del(ctx, arbnode.LIVELINESS_KEY_PREFIX+name)
	}
	priorities = priorities[:len(priorities)-1] // remove last ","
	Require(t, redisClient.Set(ctx, arbnode.PRIORITIES_KEY, priorities, time.Duration(0)).Err())
	for msg := 0; msg < 1000; msg++ {
		redisClient.Del(ctx, fmt.Sprintf("%s%d", arbnode.MESSAGE_KEY_PREFIX, msg))
	}
	redisClient.Del(ctx, arbnode.CHOSENSEQ_KEY, arbnode.MSG_COUNT_KEY)
}

func TestRedisSeqCoordinatorPriorities(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeConfig := arbnode.ConfigDefaultL2Test()
	nodeConfig.SeqCoordinator.Enable = true
	nodeConfig.SeqCoordinator.RedisUrl = redisutil.GetTestRedisURL(t)

	l2Info := NewArbTestInfo(t, params.ArbitrumDevTestChainConfig().ChainID)

	// stdio protocol makes sure forwarder initialization doesn't fail
	nodeNames := []string{"stdio://A", "stdio://B", "stdio://C", "stdio://D", "stdio://E"}

	type nodeInfo struct {
		n *arbnode.Node
		s *node.Node
	}
	nodes := make([]*nodeInfo, len(nodeNames))

	// init DB to known state
	initRedisForTest(t, ctx, nodeConfig.SeqCoordinator.RedisUrl, nodeNames)

	createStartNode := func(nodeNum int) {
		nodeConfig.SeqCoordinator.MyUrl = nodeNames[nodeNum]
		_, node, _, l2stack := CreateTestL2WithConfig(t, ctx, l2Info, nodeConfig, false)
		nodes[nodeNum] = &nodeInfo{n: node, s: l2stack}
	}

	trySequencing := func(nodeNum int) bool {
		node := nodes[nodeNum]
		curMsgs, err := node.n.TxStreamer.GetMessageCountSync()
		Require(t, err)
		emptyMessage := arbstate.MessageWithMetadata{
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{
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
		err = node.n.SeqCoordinator.SequencingMessage(curMsgs, &emptyMessage)
		if errors.Is(err, arbnode.ErrRetrySequencer) {
			return false
		}
		Require(t, err)
		Require(t, node.n.TxStreamer.AddMessages(curMsgs, false, []arbstate.MessageWithMetadata{emptyMessage}))
		return true
	}

	// node(n) has higher prio than node(n+1), so should be impossible for more than one to succeed
	trySequencingEverywhere := func() int {
		succeeded := -1
		for nodeNum, node := range nodes {
			if node == nil {
				continue
			}
			if trySequencing(nodeNum) {
				if succeeded >= 0 {
					t.Fatal("sequnced succeeded in parallel",
						"index1:", succeeded, "debug", nodes[succeeded].n.SeqCoordinator.DebugPrint(),
						"index2:", nodeNum, "debug", node.n.SeqCoordinator.DebugPrint(),
						"now", time.Now().UnixMilli())
				}
				succeeded = nodeNum
			}
		}
		return succeeded
	}

	waitForMsgEverywhere := func(msgNum arbutil.MessageIndex) {
		for _, currentNode := range nodes {
			if currentNode == nil {
				continue
			}
			for attempts := 1; ; attempts++ {
				msgCount, err := currentNode.n.TxStreamer.GetMessageCountSync()
				Require(t, err)
				if msgCount >= msgNum {
					break
				}
				if attempts > 10 {
					Fail(t, "timeout waiting for msg ", msgNum, " debug: ", currentNode.n.SeqCoordinator.DebugPrint())
				}
				select {
				case <-time.After(nodeConfig.SeqCoordinator.UpdateInterval / 3):
				}
			}
		}
	}

	killNode := func(nodeNum int) {
		requireClose(t, nodes[nodeNum].s)
		nodes[nodeNum] = nil
	}

	nodeForwardTarget := func(nodeNum int) int {
		fwTarget := nodes[nodeNum].n.TxPublisher.(*arbnode.TxPreChecker).TransactionPublisher.(*arbnode.Sequencer).ForwardTarget()
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

	for i := 1; i < len(nodes); i++ {
		createStartNode(i)
	}

	addNodes := false

	// remove sequencers one by one

	for {

		// all remaining nodes know which is the chosen one
		for i := currentSequencer + 1; i < len(nodes); i++ {
			for attempts := 1; nodeForwardTarget(i) != currentSequencer; attempts++ {
				if attempts > 10 {
					t.Fatal("initial forward target not set")
				}
				time.Sleep(time.Millisecond * 100)
			}
		}

		// sequencing suceeds only on the leder
		for i := arbutil.MessageIndex(0); i < messagesPerRound; i++ {
			if sequencer := trySequencingEverywhere(); sequencer != currentSequencer {
				Fail(t, "unexpected sequencer. expected: ", currentSequencer, " got ", sequencer)
			}
			sequencedMesssages++
		}

		if currentSequencer == len(nodes)-1 {
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
				Fail(t, "failed to sequence")
			}
			if sequencer != -1 {
				sequencedMesssages++
			}
			if sequencer == -1 ||
				(addNodes && (sequencer == currentSequencer+1)) {
				time.Sleep(nodeConfig.SeqCoordinator.LockoutDuration / 5)
				continue
			}
			if sequencer == currentSequencer {
				break
			}
			Fail(t, "unexpected sequencer", "expected", currentSequencer, "got", sequencer, "messages", sequencedMesssages)
		}

		// all nodes get messages
		waitForMsgEverywhere(sequencedMesssages)

		// can sequence after up to date
		for i := arbutil.MessageIndex(0); i < messagesPerRound; i++ {
			sequencer := trySequencingEverywhere()
			if sequencer != currentSequencer {
				Fail(t, "unexpected sequencer", "expected", currentSequencer, "got", sequencer, "messages", sequencedMesssages)
			}
			sequencedMesssages++
		}

		// all nodes get messages
		waitForMsgEverywhere(sequencedMesssages)
	}

	for nodeNum := range nodes {
		killNode(nodeNum)
	}

}

func TestRedisSeqCoordinatorMessageSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeConfig := arbnode.ConfigDefaultL2Test()
	nodeConfig.SeqCoordinator.Enable = true
	nodeConfig.SeqCoordinator.RedisUrl = redisutil.GetTestRedisURL(t)

	nodeNames := []string{"stdio://A", "stdio://B"}

	initRedisForTest(t, ctx, nodeConfig.SeqCoordinator.RedisUrl, nodeNames)

	nodeConfig.SeqCoordinator.MyUrl = nodeNames[0]
	l2Info, _, clientA, l2stackA := CreateTestL2WithConfig(t, ctx, nil, nodeConfig, false)
	defer requireClose(t, l2stackA)

	redisClient, err := redisutil.RedisClientFromURL(nodeConfig.SeqCoordinator.RedisUrl)
	Require(t, err)
	defer redisClient.Close()

	// wait for sequencerA to become master
	for {
		err := redisClient.Get(ctx, arbnode.CHOSENSEQ_KEY).Err()
		if errors.Is(err, redis.Nil) {
			time.Sleep(nodeConfig.SeqCoordinator.UpdateInterval)
			continue
		}
		Require(t, err)
		break
	}

	nodeConfig.SeqCoordinator.MyUrl = nodeNames[1]
	_, _, clientB, l2stackB := CreateTestL2WithConfig(t, ctx, l2Info, nodeConfig, false)
	defer requireClose(t, l2stackB)

	l2Info.GenerateAccount("User2")

	tx := l2Info.PrepareTx("Owner", "User2", l2Info.TransferGas, big.NewInt(1e12), nil)

	err = clientA.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, clientA, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, clientB, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := clientB.BalanceAt(ctx, l2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

func TestRedisSeqCoordinatorWrongKeyMessageSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeConfig := arbnode.ConfigDefaultL2Test()
	nodeConfig.SeqCoordinator.Enable = true
	nodeConfig.SeqCoordinator.RedisUrl = redisutil.GetTestRedisURL(t)

	nodeNames := []string{"stdio://A", "stdio://B"}

	initRedisForTest(t, ctx, nodeConfig.SeqCoordinator.RedisUrl, nodeNames)

	nodeConfig.SeqCoordinator.MyUrl = nodeNames[0]
	l2Info, _, clientA, l2stackA := CreateTestL2WithConfig(t, ctx, nil, nodeConfig, false)
	defer requireClose(t, l2stackA)

	redisClient, err := redisutil.RedisClientFromURL(nodeConfig.SeqCoordinator.RedisUrl)
	Require(t, err)
	defer redisClient.Close()

	// wait for sequencerA to become master
	for {
		err := redisClient.Get(ctx, arbnode.CHOSENSEQ_KEY).Err()
		if errors.Is(err, redis.Nil) {
			time.Sleep(nodeConfig.SeqCoordinator.UpdateInterval)
			continue
		}
		Require(t, err)
		break
	}

	nodeConfigCopy := *nodeConfig
	nodeConfig = &nodeConfigCopy
	nodeConfig.SeqCoordinator.MyUrl = nodeNames[1]
	nodeConfig.SeqCoordinator.Signing.SigningKey = "629b39225c813bf1975fb49bcb6ca2622f2c62509f138ac609f0c048764a95ee"
	_, _, clientB, l2stackB := CreateTestL2WithConfig(t, ctx, l2Info, nodeConfig, false)
	defer requireClose(t, l2stackB)

	l2Info.GenerateAccount("User2")

	tx := l2Info.PrepareTx("Owner", "User2", l2Info.TransferGas, big.NewInt(1e12), nil)

	err = clientA.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, clientA, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, clientB, tx.Hash(), time.Second)
	if err == nil {
		Fail(t, "tx received by node with different seq coordinator signing key")
	}
}
