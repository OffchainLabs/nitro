package arbnode

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/util/redisutil"
)

func prepareTrue() bool  { return true }
func prepareFalse() bool { return false }

const test_attempts = 10
const test_threads = 10
const test_release_frac = 5
const test_delay = time.Millisecond
const test_redisKey_prefix = "__TEMP_SimpleRedisLockTest__"

func attemptLock(ctx context.Context, s *SimpleRedisLock, flag *int32, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < test_attempts; i++ {
		if s.AttemptLock(ctx) {
			atomic.AddInt32(flag, 1)
		} else if rand.Intn(test_release_frac) == 0 {
			s.Release(ctx)
		}
		select {
		case <-time.After(test_delay):
		case <-ctx.Done():
			return
		}
	}
}

func simpleRedisLockTest(t *testing.T, redisKeySuffix string, chosen int, backgound bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisKey := test_redisKey_prefix + redisKeySuffix
	redisUrl := redisutil.GetTestRedisURL(t)
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	Require(t, redisClient.Del(ctx, redisKey).Err())

	conf := &SimpleRedisLockConfig{
		LockoutDuration: test_delay * test_attempts * 10,
		RefreshDuration: test_delay * 2,
		Key:             redisKey,
		BackgroundLock:  backgound,
	}
	confFetcher := func() *SimpleRedisLockConfig { return conf }

	locks := make([]*SimpleRedisLock, 0)
	for i := 0; i < test_threads; i++ {
		var err error
		var lock *SimpleRedisLock
		if chosen < 0 || chosen == i {
			lock, err = NewSimpleRedisLock(redisClient, confFetcher, prepareTrue)
		} else {
			lock, err = NewSimpleRedisLock(redisClient, confFetcher, prepareFalse)
		}
		if err != nil {
			t.Fatal(err)
		}
		lock.Start(ctx)
		defer lock.StopAndWait()
		locks = append(locks, lock)
	}
	if backgound {
		<-time.After(time.Second)
	}
	wg := sync.WaitGroup{}
	counters := make([]int32, test_threads)
	for i, lock := range locks {
		wg.Add(1)
		go attemptLock(ctx, lock, &counters[i], &wg)
	}
	wg.Wait()
	successful := -1
	for i, counter := range counters {
		if counter != 0 {
			if counter != test_attempts {
				t.Fatalf("counter %d value %d", i, counter)
			}
			if successful > 0 {
				t.Fatalf("counter %d and %d both positive", i, successful)
			}
			successful = i
		}
	}
	if successful < 0 {
		t.Fatal("no counter succeeded")
	}
	if chosen >= 0 && chosen != successful {
		t.Fatalf("counter %d succeeded, should have been %d", successful, chosen)
	}
}

func TestRedisLock0(t *testing.T) {
	simpleRedisLockTest(t, "0", 0, false)
}

func TestRedisLock0Bg(t *testing.T) {
	simpleRedisLockTest(t, "0bg", 0, true)
}

func TestRedisLock7(t *testing.T) {
	simpleRedisLockTest(t, "7", 7, false)
}

func TestRedisLockAny(t *testing.T) {
	simpleRedisLockTest(t, "a", -1, false)
}

func TestRedisLockAnyBg(t *testing.T) {
	simpleRedisLockTest(t, "abg", -1, true)
}
