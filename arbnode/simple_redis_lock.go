package arbnode

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type SimpleRedisLock struct {
	stopwaiter.StopWaiter
	client      redis.UniversalClient
	config      *SimpleRedisLockConfig
	lockedUntil int64
	mutex       sync.Mutex
	stopping    bool
	readyToLock func() bool
	myId        string
}

type SimpleRedisLockConfig struct {
	RedisUrl        string        `koanf:"redis-url"`
	MyId            string        `koanf:"my-id"`
	LockoutDuration time.Duration `koanf:"lockout-duration"`
	RefreshDuration time.Duration `koanf:"refresh-duration"`
	Key             string        `koanf:"key"`
	BackgroundLock  bool          `koanf:"background-lock"`
}

func RedisLockConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".redis-url", DefaultRedisLockConfig.RedisUrl, "URL of redis server for coordination")
	f.String(prefix+".my-id", DefaultRedisLockConfig.RedisUrl, "this node's id prefix when acquiring the lock (optional)")
	f.Duration(prefix+".lockout-duration", DefaultRedisLockConfig.LockoutDuration, "how long lock is held")
	f.Duration(prefix+".refresh-duration", DefaultRedisLockConfig.RefreshDuration, "how long between consecutive calls to redis")
	f.String(prefix+".key", prefix+"simple-lock", "key for lock")
	f.Bool(prefix+".background-lock", DefaultRedisLockConfig.BackgroundLock, "should node always try grabing lock in background")
}

func NewSimpleRedisLock(config *SimpleRedisLockConfig, readyToLock func() bool) (*SimpleRedisLock, error) {
	var client redis.UniversalClient
	if config.RedisUrl == "" {
		client = nil
	} else {
		redisOptions, err := redis.ParseURL(config.RedisUrl)
		if err != nil {
			return nil, err
		}
		client = redis.NewClient(redisOptions)
	}
	return &SimpleRedisLock{
		myId:        config.MyId + "-" + strconv.FormatInt(rand.Int63(), 16), // unique even if config is not
		client:      client,
		config:      config,
		readyToLock: readyToLock,
	}, nil
}

var DefaultRedisLockConfig = SimpleRedisLockConfig{
	RedisUrl:        "",
	LockoutDuration: time.Minute,
	RefreshDuration: time.Second * 10,
	Key:             "",
	BackgroundLock:  false,
}

func (l *SimpleRedisLock) attemptLock(ctx context.Context) (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.stopping || l.client == nil {
		return false, nil
	}
	if !l.readyToLock() {
		return false, nil
	}
	gotLock := false
	timeAtStart := time.Now()

	err := l.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, l.config.Key).Result()
		if errors.Is(err, redis.Nil) {
			current = ""
			err = nil
		}
		if err != nil {
			return err
		}
		if current != "" && (current != l.myId) {
			return nil
		}
		pipe := tx.TxPipeline()
		pipe.Set(ctx, l.config.Key, l.myId, l.config.LockoutDuration)
		pipe.PExpireAt(ctx, l.config.Key, timeAtStart.Add(l.config.LockoutDuration))
		err = execTestPipe(pipe, ctx)
		if errors.Is(err, redis.TxFailedErr) {
			return nil
		}
		if err != nil {
			return err
		}
		gotLock = true
		return nil
	}, l.config.Key)

	if !gotLock {
		atomicTimeWrite(&l.lockedUntil, time.Time{})
	}
	if err != nil {
		return false, err
	}
	if gotLock {
		if l.config.BackgroundLock {
			atomicTimeWrite(&l.lockedUntil, timeAtStart.Add(l.config.LockoutDuration))
		} else {
			atomicTimeWrite(&l.lockedUntil, timeAtStart.Add(l.config.RefreshDuration))
		}
	}
	return gotLock, nil
}

func (l *SimpleRedisLock) AttemptLock(ctx context.Context) bool {
	if l.Locked() {
		return true
	}
	if l.config.BackgroundLock {
		return false
	}
	res, err := l.attemptLock(ctx)
	if err != nil {
		log.Error("attemptLock returned error", "err", err)
		return false
	}
	return res
}

func (l *SimpleRedisLock) Locked() bool {
	if l.client == nil {
		return true
	}
	return time.Now().Before(atomicTimeRead(&l.lockedUntil))
}

func (l *SimpleRedisLock) Release(ctx context.Context) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.client == nil {
		return
	}

	err := l.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, l.config.Key).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return err
		}
		if current != l.myId {
			return nil
		}
		pipe := tx.TxPipeline()
		pipe.Del(ctx, l.config.Key, l.myId)
		err = execTestPipe(pipe, ctx)
		if errors.Is(err, redis.TxFailedErr) {
			return nil
		}
		if err != nil {
			return err
		}
		return nil
	}, l.config.Key)

	if err != nil {
		log.Error("release returned error", "err", err)
	}
}

func (l *SimpleRedisLock) Start(ctxin context.Context) {
	l.StopWaiter.Start(ctxin)
	if l.config.BackgroundLock && l.client != nil {
		l.CallIteratively(func(ctx context.Context) time.Duration {
			_, err := l.attemptLock(ctx)
			if err != nil {
				log.Error("attemptLock returned error", "err", err)
			}
			return l.config.RefreshDuration
		})
	}
}

func (l *SimpleRedisLock) StopAndWait() {
	l.mutex.Lock()
	l.stopping = true
	l.mutex.Unlock()
	l.Release(l.GetContext())
	l.StopWaiter.StopAndWait()
}
