//go:build redistest
// +build redistest

package arbnode

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbutil"
)

const messagesPerRound = 20

type CoordinatorTestData struct {
	messageCount uint64

	sequencer []string
	err       error
	mutex     sync.Mutex

	waitForCoords  sync.WaitGroup
	testStartRound int32
}

func coordinatorTestThread(ctx context.Context, coord *SeqCoordinator, data *CoordinatorTestData) {
	nextRound := int32(0)
	for {
		sequenced := make([]bool, messagesPerRound)
		for atomic.LoadInt32(&data.testStartRound) < nextRound {
			if ctx.Err() != nil {
				return
			}
		}
		nextRound++
		var execError error
		var lockupUntil time.Time
		for {
			messageCount := atomic.LoadUint64(&data.messageCount)
			if messageCount >= messagesPerRound {
				break
			}
			coord.prevMsgCount = arbutil.MessageIndex(messageCount)
			maybeLockoutUntil, err := coord.chosenOneUpdate(ctx, arbutil.MessageIndex(messageCount+1))
			timeLaunching := time.Now()
			if err == nil {
				lockupUntil = maybeLockoutUntil
				sequenced[messageCount] = true
				atomic.StoreUint64(&data.messageCount, messageCount+1)
				randNr := rand.Intn(20)
				if randNr > 15 {
					coord.chosenOneRelease(ctx)
					lockupUntil = time.Time{}
				} else {
					time.Sleep(coord.config.LockoutDuration * time.Duration(randNr) / 10)
				}
				continue
			}
			// didn't sequence.. should we have succeeded?
			if timeLaunching.Before(lockupUntil) {
				execError = fmt.Errorf("failed while holding lock %s err %w", coord.config.MyUrl, err)
				break
			}
		}
		data.mutex.Lock()
		for i, me := range sequenced {
			if !me {
				continue
			}
			if data.sequencer[i] != "" {
				execError = fmt.Errorf("two sequencers for same msg: submsg %d, success for %s, %s", i, data.sequencer[i], coord.config.MyUrl)
			}
			data.sequencer[i] = coord.config.MyUrl
		}
		if execError != nil {
			data.err = execError
		}
		data.mutex.Unlock()
		data.waitForCoords.Done()
	}
}

func TestSeqCoordinatorAtomic(t *testing.T) {
	NumOfThreads := 10
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coordConfig := TestSeqCoordinatorConfig
	coordConfig.LockoutDuration = time.Millisecond * 10
	testData := CoordinatorTestData{
		testStartRound: -1,
		sequencer:      make([]string, messagesPerRound),
	}

	redisUrl := os.Getenv("TEST_REDIS")
	if redisUrl == "" {
		redisUrl = "redis://localhost:6379/0"
	}
	redisOptions, err := redis.ParseURL(redisUrl)
	Require(t, err)

	redisClient := redis.NewClient(redisOptions)

	for i := 0; i < NumOfThreads; i++ {
		config := coordConfig
		config.MyUrl = fmt.Sprint(i)
		coordinator := &SeqCoordinator{
			client: redis.NewClient(redisOptions),
			config: config,
		}
		go coordinatorTestThread(ctx, coordinator, &testData)
	}

	for round := int32(0); round < 10; round++ {
		redisClient.Del(ctx, CHOSENSEQ_KEY, MSG_COUNT_KEY)
		testData.messageCount = 0
		for i := 0; i < messagesPerRound; i++ {
			testData.sequencer[i] = ""
		}
		testData.waitForCoords.Add(NumOfThreads)
		atomic.StoreInt32(&testData.testStartRound, round)
		testData.waitForCoords.Wait()
		Require(t, testData.err)
		seqList := ""
		for i := 0; i < messagesPerRound; i++ {
			if testData.sequencer[i] == "" {
				Fail(t, "no sequencer succeded", "round", round, "message", i)
			}
			seqList = seqList + testData.sequencer[i] + ","
		}

		t.Log("Round", round, "sequencers", seqList)
		// wait out the current lock
		time.Sleep(time.Millisecond * 20)
	}

}
