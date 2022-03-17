package arbtest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
)

func initRedisForTest(t *testing.T, ctx context.Context, redisClient *redis.Client, nodeNames []string) {
	var priorities string
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

func getTestRediUrl() string {
	redisUrl := os.Getenv("TEST_REDIS")
	if redisUrl == "" {
		redisUrl = "redis://localhost:6379/0"
	}
	return redisUrl
}

func TestSeqCoordinatorPriorities(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisOptions, err := redis.ParseURL(getTestRediUrl())
	Require(t, err)
	nodeConfig := arbnode.NodeConfigL2Test
	nodeConfig.SeqCoordinator = true
	nodeConfig.SeqCoordinatorConfig = arbnode.TestSeqCoordinatorConfig

	l2Info := NewArbTestInfo(t)

	// stdio protocol makes sure forwarder initialization doesn't fail
	nodeNames := []string{"stdio://A", "stdio://B", "stdio://C", "stdio://D", "stdio://E"}
	nodes := make([]*arbnode.Node, len(nodeNames))

	// init DB to known state
	initRedisForTest(t, ctx, redis.NewClient(redisOptions), nodeNames)

	createStartNode := func(nodeNum int) {
		nodeConfig.SeqCoordinatorConfig.MyUrl = nodeNames[nodeNum]
		_, node, _ := CreateTestL2WithConfig(t, ctx, l2Info, &nodeConfig, redis.NewClient(redisOptions), false)
		node.TxStreamer.StopAndWait() // prevent blocks from building
		nodes[nodeNum] = node
	}

	trySequencing := func(nodeNum int) bool {
		node := nodes[nodeNum]
		curMsgs, err := node.TxStreamer.GetMessageCountSync()
		Require(t, err)
		err = node.SeqCoordinator.SequencingMessage(curMsgs, &arbstate.MessageWithMetadata{})
		if errors.Is(err, arbnode.ErrNotMainSequencer) {
			return false
		}
		Require(t, err)
		Require(t, node.TxStreamer.AddMessages(curMsgs, false, []arbstate.MessageWithMetadata{{}}))
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
						"index1:", succeeded, "debug", nodes[succeeded].SeqCoordinator.DebugPrint(),
						"index2:", nodeNum, "debug", node.SeqCoordinator.DebugPrint(),
						"now", time.Now().UnixMilli())
				}
				succeeded = nodeNum
			}
		}
		return succeeded
	}

	waitForMsgEverywhere := func(msgNum arbutil.MessageIndex) {
		for _, node := range nodes {
			if node == nil {
				continue
			}
			for attempts := 1; ; attempts++ {
				msgCount, err := node.TxStreamer.GetMessageCountSync()
				Require(t, err)
				if msgCount >= msgNum {
					break
				}
				if attempts > 10 {
					Fail(t, "timeout waiting for msg ", msgNum, " debug: ", node.SeqCoordinator.DebugPrint())
				}
				time.Sleep(nodeConfig.SeqCoordinatorConfig.UpdateInterval / 3)
			}
		}
	}

	killNode := func(nodeNum int) {
		nodes[nodeNum].StopAndWait()
		nodes[nodeNum] = nil
	}

	nodeForwardTarget := func(nodeNum int) int {
		fwTarget := nodes[nodeNum].TxPublisher.(*arbnode.Sequencer).ForwardTarget()
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
				time.Sleep(nodeConfig.SeqCoordinatorConfig.LockoutDuration / 5)
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

func TestSeqCoordinatorMessageSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeConfig := arbnode.NodeConfigL2Test
	nodeConfig.SeqCoordinator = true
	nodeConfig.SeqCoordinatorConfig = arbnode.TestSeqCoordinatorConfig

	nodeNames := []string{"stdio://A", "stdio://B"}

	redisOptions, err := redis.ParseURL(getTestRediUrl())
	Require(t, err)
	redisClient := redis.NewClient(redisOptions)

	initRedisForTest(t, ctx, redisClient, nodeNames)

	nodeConfig.SeqCoordinatorConfig.MyUrl = nodeNames[0]
	l2Info, nodeA, clientA := CreateTestL2WithConfig(t, ctx, nil, &nodeConfig, redis.NewClient(redisOptions), false)

	// wait for sequencerA to become master
	for {
		err := redisClient.Get(ctx, arbnode.CHOSENSEQ_KEY).Err()
		if errors.Is(err, redis.Nil) {
			time.Sleep(nodeConfig.SeqCoordinatorConfig.UpdateInterval)
			continue
		}
		Require(t, err)
		break
	}

	nodeConfig.SeqCoordinatorConfig.MyUrl = nodeNames[1]
	_, nodeB, clientB := CreateTestL2WithConfig(t, ctx, l2Info, &nodeConfig, redis.NewClient(redisOptions), false)

	l2Info.GenerateAccount("User2")

	tx := l2Info.PrepareTx("Owner", "User2", l2Info.TransferGas, big.NewInt(1e12), nil)

	err = clientA.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = arbutil.EnsureTxSucceeded(ctx, clientA, tx)
	Require(t, err)

	_, err = arbutil.WaitForTx(ctx, clientB, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := clientB.BalanceAt(ctx, l2Info.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
	nodeA.StopAndWait()
	nodeB.StopAndWait()
}
