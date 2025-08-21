package arbnode

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode/redislock"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func prepareTrue() bool  { return true }
func prepareFalse() bool { return false }

const test_attempts = 10
const test_threads = 10
const test_release_frac = 5
const test_delay = time.Millisecond
const test_redisKey_prefix = "__TEMP_SimpleRedisLockTest__"

func attemptLock(ctx context.Context, s *redislock.Simple, flag *atomic.Int32, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < test_attempts; i++ {
		if s.AttemptLock(ctx) {
			flag.Add(1)
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

func simpleRedisLockTest(t *testing.T, redisKeySuffix string, chosen int, background bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisKey := test_redisKey_prefix + redisKeySuffix
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	Require(t, redisClient.Del(ctx, redisKey).Err())

	conf := &redislock.SimpleCfg{
		Enable:          true,
		LockoutDuration: test_delay * test_attempts * 10,
		RefreshDuration: test_delay * 2,
		Key:             redisKey,
		BackgroundLock:  background,
	}
	confFetcher := func() *redislock.SimpleCfg { return conf }

	locks := make([]*redislock.Simple, 0)
	for i := 0; i < test_threads; i++ {
		var err error
		var lock *redislock.Simple
		if chosen < 0 || chosen == i {
			lock, err = redislock.NewSimple(redisClient, confFetcher, prepareTrue)
		} else {
			lock, err = redislock.NewSimple(redisClient, confFetcher, prepareFalse)
		}
		if err != nil {
			t.Fatal(err)
		}
		lock.Start(ctx)
		defer lock.StopAndWait()
		locks = append(locks, lock)
	}
	if background {
		<-time.After(time.Second)
	}
	wg := sync.WaitGroup{}
	counters := make([]atomic.Int32, test_threads)
	for i, lock := range locks {
		wg.Add(1)
		go attemptLock(ctx, lock, &counters[i], &wg)
	}
	wg.Wait()
	successful := -1
	for i := range counters {
		if counters[i].Load() != 0 {
			if counters[i].Load() != test_attempts {
				t.Fatalf("counter %d value %d", i, counters[i].Load())
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

func TestAttemptLockAndPeriodicallyRefreshIt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisKey := test_redisKey_prefix + "PeriodicallyRefreshed"
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	Require(t, err)
	Require(t, redisClient.Del(ctx, redisKey).Err())

	conf := &redislock.SimpleCfg{
		Enable:          true,
		LockoutDuration: 3 * time.Second,
		RefreshDuration: 1 * time.Second,
		Key:             redisKey,
		BackgroundLock:  true,
	}
	confFetcher := func() *redislock.SimpleCfg { return conf }

	lock1, err := redislock.NewSimple(redisClient, confFetcher, prepareTrue)
	Require(t, err)

	release := make(chan struct{})
	gotLock := lock1.AttemptLockAndPeriodicallyRefreshIt(ctx, release)
	if !gotLock {
		t.Fatal("lock not obtained")
	}

	// still locked after LockoutDuration
	time.Sleep(7 * time.Second)
	if !lock1.Locked() {
		t.Fatal("lock not held after 7 seconds")
	}

	// another redislock instance should not be able to obtain the lock
	lock2, err := redislock.NewSimple(redisClient, confFetcher, prepareTrue)
	Require(t, err)
	gotLock = lock2.AttemptLockAndPeriodicallyRefreshIt(ctx, make(chan struct{}))
	if gotLock {
		t.Fatal("lock obtained when it should not have been")
	}

	// releases the lock
	release <- struct{}{}
	time.Sleep(100 * time.Millisecond)
	if lock1.Locked() {
		t.Fatal("lock not released")
	}

	// another redislock instance should be able to obtain the lock
	gotLock = lock2.AttemptLockAndPeriodicallyRefreshIt(ctx, release)
	if !gotLock {
		t.Fatal("lock not obtained")
	}

	release <- struct{}{}
}
