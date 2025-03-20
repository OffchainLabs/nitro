package timeboost

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestTimeboostRedisCoordinator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisUrl := redisutil.CreateTestRedis(ctx, t)
	timingInfo := &RoundTimingInfo{
		Offset: time.Now(),
		Round:  time.Second * 5,
	}
	redisCoordinator, err := NewRedisCoordinator(redisUrl, timingInfo, 50)
	if err != nil {
		t.Fatalf("error initializing redis coordinator: %v", err)
	}
	redisCoordinator.Start(ctx)

	// Verify adding and retrieving global sequence count of a round
	var round uint64
	checkSeqCountInRedis := func(expected uint64) {
		globalSeq, err := redisCoordinator.GetSequenceCount(round)
		if err != nil {
			t.Fatalf("error getting sequence count of a round: %v", err)
		}
		if globalSeq != expected {
			t.Fatal("round's seq count mismatch")
		}
	}
	err = redisCoordinator.UpdateSequenceCount(round, 3) // should succeed
	time.Sleep(10 * time.Millisecond)                    // necessary since channels are being used to carry out the updates
	if err != nil {
		t.Fatalf("error setting round number and sequence count: %v", err)
	}
	checkSeqCountInRedis(3)
	err = redisCoordinator.UpdateSequenceCount(round, 1) // shouldn't succeed as the sequence count is a lower value
	time.Sleep(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("error setting round number and sequence count: %v", err)
	}
	checkSeqCountInRedis(3)
	round = 1
	err = redisCoordinator.UpdateSequenceCount(round, 4) // should succeed
	time.Sleep(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("error setting round number and sequence count: %v", err)
	}
	checkSeqCountInRedis(4)

	// Test adding and retrieval of expressLane messages
	var addedMsgs []*ExpressLaneSubmission
	emptyTx := types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), nil)
	for i := uint64(0); i < 5; i++ {
		msg := &ExpressLaneSubmission{ChainId: common.Big0, Round: round, SequenceNumber: i, Transaction: emptyTx}
		if err := redisCoordinator.AddAcceptedTx(msg); err != nil {
			t.Fatalf("error adding expressLane msg to redis: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
		addedMsgs = append(addedMsgs, msg)
	}

	checkCorrectness := func(startSeqNum uint64) {
		fetchedMsgs := redisCoordinator.GetAcceptedTxs(round, startSeqNum, startSeqNum+5)
		if len(fetchedMsgs) != len(addedMsgs[startSeqNum:]) {
			t.Fatal("mismatch in number of fetched msgs")
		}
		for i, msg := range fetchedMsgs {
			haveBytes, err := msg.ToMessageBytes()
			if err != nil {
				t.Fatalf("error getting messageBytes: %v", err)
			}
			// #nosec G115
			wantBytes, err := addedMsgs[int(startSeqNum)+i].ToMessageBytes()
			if err != nil {
				t.Fatalf("error getting messageBytes: %v", err)
			}
			if !bytes.Equal(haveBytes, wantBytes) {
				t.Fatal("mismatch in message fetched from redis")
			}
		}
	}
	checkCorrectness(0) // when all messages are fetched
	checkCorrectness(3) // when messages are filtered with startSeqNum=3
}
