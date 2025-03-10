// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/signature"
)

const messagesPerRound = 20

type CoordinatorTestData struct {
	messageCount atomic.Uint64

	sequencer []string
	err       error
	mutex     sync.Mutex

	waitForCoords  sync.WaitGroup
	testStartRound atomic.Int32
}

func coordinatorTestThread(ctx context.Context, coord *SeqCoordinator, data *CoordinatorTestData) {
	nextRound := int32(0)
	for {
		sequenced := make([]bool, messagesPerRound)
		for data.testStartRound.Load() < nextRound {
			if ctx.Err() != nil {
				return
			}
		}
		atomicTimeWrite(&coord.lockoutUntil, time.Time{})
		nextRound++
		var execError error
		for {
			messageCount := data.messageCount.Load()
			if messageCount >= messagesPerRound {
				break
			}
			asIndex := arbutil.MessageIndex(messageCount)
			holdingLockout := atomicTimeRead(&coord.lockoutUntil)
			err := coord.acquireLockoutAndWriteMessage(ctx, asIndex, asIndex+1, &arbostypes.EmptyTestMessageWithMetadata, nil)
			if err == nil {
				sequenced[messageCount] = true
				data.messageCount.Store(messageCount + 1)
				randNr := rand.Intn(20)
				if randNr > 15 {
					execError = coord.chosenOneRelease(ctx)
					if execError != nil {
						break
					}
					atomicTimeWrite(&coord.lockoutUntil, time.Time{})
				} else {
					time.Sleep(coord.config.LockoutDuration * time.Duration(randNr) / 10)
				}
				continue
			}
			timeLaunching := time.Now()
			// didn't sequence.. should we have succeeded?
			if timeLaunching.Before(holdingLockout) {
				execError = fmt.Errorf("failed while holding lock %s err %w", coord.config.Url(), err)
				break
			}
		}
		data.mutex.Lock()
		for i, me := range sequenced {
			if !me {
				continue
			}
			if data.sequencer[i] != "" {
				execError = fmt.Errorf("two sequencers for same msg: submsg %d, success for %s, %s", i, data.sequencer[i], coord.config.Url())
			}
			data.sequencer[i] = coord.config.Url()
		}
		if execError != nil {
			data.err = execError
		}
		data.mutex.Unlock()
		data.waitForCoords.Done()
	}
}

func TestRedisSeqCoordinatorAtomic(t *testing.T) {
	NumOfThreads := 10
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coordConfig := TestSeqCoordinatorConfig
	coordConfig.LockoutDuration = time.Millisecond * 100
	coordConfig.LockoutSpare = time.Millisecond * 10
	coordConfig.Signer.ECDSA.AcceptSequencer = false
	coordConfig.Signer.SymmetricFallback = true
	coordConfig.Signer.SymmetricSign = true
	coordConfig.Signer.Symmetric.Dangerous.DisableSignatureVerification = true
	coordConfig.Signer.Symmetric.SigningKey = ""
	testData := CoordinatorTestData{
		sequencer: make([]string, messagesPerRound),
	}
	testData.testStartRound.Store(-1)
	nullSigner, err := signature.NewSignVerify(&coordConfig.Signer, nil, nil)
	Require(t, err)

	redisUrl := redisutil.CreateTestRedis(ctx, t)
	coordConfig.RedisUrl = redisUrl
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	if redisClient == nil {
		t.Fatal("redisClient is nil")
	}

	for i := 0; i < NumOfThreads; i++ {
		config := coordConfig
		config.MyUrl = fmt.Sprint(i)
		redisCoordinator, err := redisutil.NewRedisCoordinator(config.RedisUrl)
		Require(t, err)
		coordinator := &SeqCoordinator{
			redisCoordinator: *redisCoordinator,
			config:           config,
			signer:           nullSigner,
		}
		go coordinatorTestThread(ctx, coordinator, &testData)
	}

	for round := int32(0); round < 10; round++ {
		redisClient.Del(ctx, redisutil.CHOSENSEQ_KEY, redisutil.MSG_COUNT_KEY)
		testData.messageCount.Store(0)
		for i := 0; i < messagesPerRound; i++ {
			testData.sequencer[i] = ""
		}
		testData.waitForCoords.Add(NumOfThreads)
		testData.testStartRound.Store(round)
		testData.waitForCoords.Wait()
		Require(t, testData.err)
		seqList := ""
		for i := 0; i < messagesPerRound; i++ {
			if testData.sequencer[i] == "" {
				Fail(t, "no sequencer succeeded", "round", round, "message", i)
			}
			seqList = seqList + testData.sequencer[i] + ","
		}

		t.Log("Round", round, "sequencers", seqList)
		// wait out the current lock
		time.Sleep(time.Millisecond * 20)
	}

}

func TestSeqCoordinatorDeletesFinalizedMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coordConfig := TestSeqCoordinatorConfig
	coordConfig.LockoutDuration = time.Millisecond * 100
	coordConfig.LockoutSpare = time.Millisecond * 10
	coordConfig.Signer.ECDSA.AcceptSequencer = false
	coordConfig.Signer.SymmetricFallback = true
	coordConfig.Signer.SymmetricSign = true
	coordConfig.Signer.Symmetric.Dangerous.DisableSignatureVerification = true
	coordConfig.Signer.Symmetric.SigningKey = ""

	nullSigner, err := signature.NewSignVerify(&coordConfig.Signer, nil, nil)
	Require(t, err)

	redisUrl := redisutil.CreateTestRedis(ctx, t)
	coordConfig.RedisUrl = redisUrl

	config := coordConfig
	config.MyUrl = "test"
	redisCoordinator, err := redisutil.NewRedisCoordinator(config.RedisUrl)
	Require(t, err)
	coordinator := &SeqCoordinator{
		redisCoordinator: *redisCoordinator,
		config:           config,
		signer:           nullSigner,
	}

	// Add messages to redis
	var keys []string
	msgBytes, err := coordinator.msgCountToSignedBytes(0)
	Require(t, err)
	for i := arbutil.MessageIndex(1); i <= 10; i++ {
		err = coordinator.RedisCoordinator().Client.Set(ctx, redisutil.MessageKeyFor(i), msgBytes, time.Hour).Err()
		Require(t, err)
		err = coordinator.RedisCoordinator().Client.Set(ctx, redisutil.MessageSigKeyFor(i), msgBytes, time.Hour).Err()
		Require(t, err)
		keys = append(keys, redisutil.MessageKeyFor(i), redisutil.MessageSigKeyFor(i))
	}
	// Set msgCount key
	msgCountBytes, err := coordinator.msgCountToSignedBytes(11)
	Require(t, err)
	err = coordinator.RedisCoordinator().Client.Set(ctx, redisutil.MSG_COUNT_KEY, msgCountBytes, time.Hour).Err()
	Require(t, err)
	exists, err := coordinator.RedisCoordinator().Client.Exists(ctx, keys...).Result()
	Require(t, err)
	if exists != 20 {
		t.Fatal("couldn't find all messages and signatures in redis")
	}

	// Set finalizedMsgCount and delete finalized messages
	err = coordinator.deleteFinalizedMsgsFromRedis(ctx, 5)
	Require(t, err)

	// Check if messages and signatures were deleted successfully
	exists, err = coordinator.RedisCoordinator().Client.Exists(ctx, keys[:8]...).Result()
	Require(t, err)
	if exists != 0 {
		t.Fatal("finalized messages and signatures in range 1 to 4 were not deleted")
	}

	// Check if finalizedMsgCount was set to correct value
	finalized, err := coordinator.getRemoteFinalizedMsgCount(ctx)
	Require(t, err)
	if finalized != 5 {
		t.Fatalf("incorrect finalizedMsgCount, want: 5, have: %d", finalized)
	}

	// Try deleting finalized messages when theres already a finalizedMsgCount
	err = coordinator.deleteFinalizedMsgsFromRedis(ctx, 7)
	Require(t, err)
	exists, err = coordinator.RedisCoordinator().Client.Exists(ctx, keys[8:12]...).Result()
	Require(t, err)
	if exists != 0 {
		t.Fatal("finalized messages and signatures in range 5 to 6 were not deleted")
	}
	finalized, err = coordinator.getRemoteFinalizedMsgCount(ctx)
	Require(t, err)
	if finalized != 7 {
		t.Fatalf("incorrect finalizedMsgCount, want: 7, have: %d", finalized)
	}

	// Check that non-finalized messages are still available in redis
	exists, err = coordinator.RedisCoordinator().Client.Exists(ctx, keys[12:]...).Result()
	Require(t, err)
	if exists != 8 {
		t.Fatal("non-finalized messages and signatures in range 7 to 10 are not fully available")
	}
}

func TestSeqCoordinatorAddsBlockMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coordConfig := TestSeqCoordinatorConfig
	coordConfig.LockoutDuration = time.Millisecond * 100
	coordConfig.LockoutSpare = time.Millisecond * 10
	coordConfig.Signer.ECDSA.AcceptSequencer = false
	coordConfig.Signer.SymmetricFallback = true
	coordConfig.Signer.SymmetricSign = true
	coordConfig.Signer.Symmetric.Dangerous.DisableSignatureVerification = true
	coordConfig.Signer.Symmetric.SigningKey = ""

	nullSigner, err := signature.NewSignVerify(&coordConfig.Signer, nil, nil)
	Require(t, err)

	redisUrl := redisutil.CreateTestRedis(ctx, t)
	coordConfig.RedisUrl = redisUrl

	config := coordConfig
	config.MyUrl = "test"
	redisCoordinator, err := redisutil.NewRedisCoordinator(config.RedisUrl)
	Require(t, err)
	coordinator := &SeqCoordinator{
		redisCoordinator: *redisCoordinator,
		config:           config,
		signer:           nullSigner,
	}

	pos := arbutil.MessageIndex(1)
	blockMetadataWant := common.BlockMetadata{0, 4}
	Require(t, coordinator.acquireLockoutAndWriteMessage(ctx, pos, pos+1, &arbostypes.EmptyTestMessageWithMetadata, blockMetadataWant))
	blockMetadataGot, err := coordinator.blockMetadataAt(ctx, pos)
	Require(t, err)
	if !bytes.Equal(blockMetadataWant, blockMetadataGot) {
		t.Fatal("got incorrect blockMetadata")
	}
}
