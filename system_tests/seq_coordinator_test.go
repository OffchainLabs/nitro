//go:build redistest
// +build redistest

package arbtest

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestSeqCoordinator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisUrl := os.Getenv("TEST_REDIS")
	if redisUrl == "" {
		redisUrl = "redis://localhost:6379/0"
	}
	redisOptions, err := redis.ParseURL(redisUrl)
	Require(t, err)
	redisClient := redis.NewClient(redisOptions)
	nodeConfig := arbnode.ConfigDefaultL2Test()
	nodeConfig.SeqCoordinator = true
	nodeConfig.SeqCoordinatorConfig = arbnode.TestSeqCoordinatorConfig

	l2Info := NewArbTestInfo(t)

	// stdio protocol makes sure forwarder initialization doesn't fail
	nodeNames := []string{"stdio://A", "stdio://B", "stdio://C", "stdio://D", "stdio://E"}
	nodes := make([]*arbnode.Node, len(nodeNames))

	// init DB to known state
	var priorities string
	for _, name := range nodeNames {
		priorities = priorities + name + ","
		redisClient.Del(ctx, arbnode.LIVELINESS_KEY_PREFIX+name)
	}
	priorities = priorities[:len(priorities)-1] // remove last ","
	Require(t, redisClient.Set(ctx, arbnode.PRIORITIES_KEY, priorities, time.Duration(0)).Err())
	redisClient.Del(ctx, arbnode.CHOSENSEQ_KEY, arbnode.MSG_COUNT_KEY)

	createStartNode := func(nodeNum int, msgNum arbutil.MessageIndex) {
		nodeConfig.SeqCoordinatorConfig.MyUrl = nodeNames[nodeNum]
		_, stack, chainDb, blockchain := createL2BlockChain(t, l2Info)
		node, err := arbnode.CreateNode(stack, chainDb, nodeConfig, blockchain, nil, nil, nil, nil, redis.NewClient(redisOptions))
		Require(t, err)
		if msgNum > 0 {
			messages := make([]arbstate.MessageWithMetadata, msgNum)
			node.TxStreamer.AddMessages(0, true, messages)
		}
		node.TxPublisher.Start(ctx)
		node.SeqCoordinator.Start(ctx)
		nodes[nodeNum] = node
	}

	trySequenceing := func(nodeNum int) bool {
		node := nodes[nodeNum]
		curMsgs, err := node.TxStreamer.GetMessageCountSync()
		Require(t, err)
		err = node.SeqCoordinator.SequencingMessage(curMsgs, &arbstate.MessageWithMetadata{})
		if errors.Is(err, arbnode.ErrNotMainSequencer) {
			return false
		}
		Require(t, err)
		messages := make([]arbstate.MessageWithMetadata, 1)
		Require(t, node.TxStreamer.AddMessages(curMsgs, true, messages))
		return true
	}

	trySequenceingEverywhere := func() int {
		succeeded := -1
		for nodeNum, node := range nodes {
			if node == nil {
				continue
			}
			if trySequenceing(nodeNum) {
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

	addMessagesUptoEverywhere := func(msgNum arbutil.MessageIndex) {
		for _, node := range nodes {
			if node == nil {
				continue
			}
			curMsgs, err := node.TxStreamer.GetMessageCountSync()
			Require(t, err)
			if curMsgs >= msgNum {
				return
			}
			messages := make([]arbstate.MessageWithMetadata, msgNum-curMsgs)
			Require(t, node.TxStreamer.AddMessages(curMsgs, true, messages))
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
	maxAttempts := 10
	delayPerAttempt := nodeConfig.SeqCoordinatorConfig.LockoutDuration / 5
	currentSequencer := 0
	sequencedMesssages := arbutil.MessageIndex(1) // we start with 1 so messageCountKey will be written
	followerMessages := sequencedMesssages

	t.Log("Starting node 0")
	// give node 0 room to set himself primary
	createStartNode(0, sequencedMesssages)

	for attempts := 1; !trySequenceing(0); attempts++ {
		if attempts > maxAttempts {
			t.Fatal("failed first sequencing")
		}
		time.Sleep(time.Millisecond * 200)
	}
	sequencedMesssages++

	t.Log("Starting other nodes")

	for i := 1; i < len(nodes); i++ {
		createStartNode(i, followerMessages)
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
			if sequencer := trySequenceingEverywhere(); sequencer != currentSequencer {
				t.Fatal("unexpected sequencer")
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
			createStartNode(currentSequencer, followerMessages)
		} else {
			t.Log("killing node")
			// current scheme does not protect from sudden death.
			// Wait for master sequencer to update redis
			for attempts := 1; ; attempts++ {
				msgCount, err := nodes[currentSequencer].SeqCoordinator.GetRemoteMsgCount(ctx)
				Require(t, err)
				if msgCount == sequencedMesssages {
					break
				}
				if attempts > maxAttempts {
					t.Fatal("currentSequencer didn't update redis", "sequencer", currentSequencer)
				}
				time.Sleep(delayPerAttempt)
			}
			killNode(currentSequencer)
			currentSequencer++
		}

		// cannot sequence until up to date with all messages
		for followerMessages < sequencedMesssages {
			sequencer := trySequenceingEverywhere()
			if addNodes && (sequencer == currentSequencer+1) {
				sequencedMesssages++
			} else if sequencer != -1 {
				Fail(t, "unexpected sequencer", "expected", currentSequencer, "got", sequencer, "messages", sequencedMesssages)
			}
			time.Sleep(delayPerAttempt)
			// followeMessages will catch up
			followerMessages++
			if followerMessages < sequencedMesssages {
				followerMessages++
			}
			addMessagesUptoEverywhere(followerMessages)
		}

		// can sequence after up to date
		for attempts := 1; ; attempts++ {
			sequencer := trySequenceingEverywhere()
			if sequencer == currentSequencer {
				sequencedMesssages++
				break
			}
			if addNodes && (sequencer == currentSequencer+1) {
				sequencedMesssages++
				followerMessages++
				addMessagesUptoEverywhere(followerMessages)
			} else if sequencer >= 0 {
				Fail(t, "unexpected sequencer", "expected", currentSequencer, "got", sequencer, "messages", sequencedMesssages)
			}
			if attempts > maxAttempts {
				Fail(t, "failed to sequence new message", "expected", currentSequencer, "messages", sequencedMesssages)
			}
			time.Sleep(delayPerAttempt)
		}
	}

	for nodeNum := range nodes {
		killNode(nodeNum)
	}

}
