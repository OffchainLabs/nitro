package redislock

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Simple struct {
	stopwaiter.StopWaiter
	client      redis.UniversalClient
	config      SimpleCfgFetcher
	lockedUntil atomic.Int64
	mutex       sync.Mutex
	stopping    bool
	readyToLock func() bool
	myId        string
}

type SimpleCfg struct {
	Enable          bool          `koanf:"enable"`
	MyId            string        `koanf:"my-id"`
	LockoutDuration time.Duration `koanf:"lockout-duration" reload:"hot"`
	RefreshDuration time.Duration `koanf:"refresh-duration" reload:"hot"`
	Key             string        `koanf:"key"`
	BackgroundLock  bool          `koanf:"background-lock"`
}

type SimpleCfgFetcher func() *SimpleCfg

func AddConfigOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultCfg.Enable, "if false, always treat this as locked and don't write the lock to redis")
	f.String(prefix+".my-id", "", "this node's id prefix when acquiring the lock (optional)")
	f.Duration(prefix+".lockout-duration", DefaultCfg.LockoutDuration, "how long lock is held")
	f.Duration(prefix+".refresh-duration", DefaultCfg.RefreshDuration, "how long between consecutive calls to redis")
	f.String(prefix+".key", DefaultCfg.Key, "key for lock")
	f.Bool(prefix+".background-lock", DefaultCfg.BackgroundLock, "should node always try grabing lock in background")
}

func NewSimple(client redis.UniversalClient, config SimpleCfgFetcher, readyToLock func() bool) (*Simple, error) {
	randBig, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	return &Simple{
		myId:        config().MyId + "-" + strconv.FormatInt(randBig.Int64(), 16), // unique even if config is not
		client:      client,
		config:      config,
		readyToLock: readyToLock,
	}, nil
}

var DefaultCfg = SimpleCfg{
	Enable:          true,
	LockoutDuration: time.Minute,
	RefreshDuration: time.Second * 10,
	Key:             "",
	BackgroundLock:  false,
}

func (l *Simple) attemptLock(ctx context.Context) (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.stopping || l.client == nil {
		return false, nil
	}
	if !l.readyToLock() {
		return false, nil
	}
	gotLock := false
	config := l.config()
	timeAtStart := time.Now()

	err := l.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, config.Key).Result()
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
		pipe.Set(ctx, config.Key, l.myId, config.LockoutDuration)
		pipe.PExpireAt(ctx, config.Key, timeAtStart.Add(config.LockoutDuration))
		err = execTestPipe(pipe, ctx)
		if errors.Is(err, redis.TxFailedErr) {
			return nil
		}
		if err != nil {
			return err
		}
		gotLock = true
		return nil
	}, config.Key)

	if !gotLock {
		atomicTimeWrite(&l.lockedUntil, time.Time{})
	}
	if err != nil {
		return false, err
	}
	if gotLock {
		if config.BackgroundLock {
			atomicTimeWrite(&l.lockedUntil, timeAtStart.Add(config.LockoutDuration))
		} else {
			atomicTimeWrite(&l.lockedUntil, timeAtStart.Add(config.RefreshDuration))
		}
	}
	return gotLock, nil
}

func (l *Simple) AttemptLock(ctx context.Context) bool {
	if l.Locked() {
		return true
	}
	if l.config().BackgroundLock {
		return false
	}
	res, err := l.attemptLock(ctx)
	if err != nil {
		log.Error("attemptLock returned error", "err", err)
		return false
	}
	return res
}

func (l *Simple) Locked() bool {
	if l.client == nil || !l.config().Enable {
		return true
	}
	return time.Now().Before(atomicTimeRead(&l.lockedUntil))
}

// Returns true if a call to AttemptLock will likely succeed
func (l *Simple) CouldAcquireLock(ctx context.Context) (bool, error) {
	if l.Locked() {
		return true, nil
	}
	if l.stopping || !l.readyToLock() {
		return false, nil
	}
	// l.client shouldn't be nil here because Locked would've returned true
	current, err := l.client.Get(ctx, l.config().Key).Result()
	if errors.Is(err, redis.Nil) {
		// Lock is free for the taking
		return true, nil
	}
	if err != nil {
		return false, err
	}
	// return true if the lock is free for the taking or is already ours
	return current == "" || current == l.myId, nil
}

func (l *Simple) Release(ctx context.Context) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.client == nil {
		return
	}

	config := l.config()
	err := l.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, config.Key).Result()
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
		pipe.Del(ctx, config.Key, l.myId)
		err = execTestPipe(pipe, ctx)
		if errors.Is(err, redis.TxFailedErr) {
			return nil
		}
		if err != nil {
			return err
		}
		return nil
	}, config.Key)

	if err != nil {
		log.Error("release returned error", "err", err)
	}
}

func (l *Simple) Start(ctxin context.Context) {
	l.StopWaiter.Start(ctxin, l)
	if l.config().BackgroundLock && l.client != nil {
		l.CallIteratively(func(ctx context.Context) time.Duration {
			_, err := l.attemptLock(ctx)
			if err != nil {
				log.Error("attemptLock returned error", "err", err)
			}
			return l.config().RefreshDuration
		})
	}
}

func (l *Simple) StopAndWait() {
	l.mutex.Lock()
	l.stopping = true
	l.mutex.Unlock()
	l.Release(l.GetContext())
	l.StopWaiter.StopAndWait()
}

func execTestPipe(pipe redis.Pipeliner, ctx context.Context) error {
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	for _, cmder := range cmders {
		if err := cmder.Err(); err != nil {
			return err
		}
	}
	return nil
}

// notice: It is possible for two consecutive reads to get decreasing values. That shouldn't matter.
func atomicTimeRead(addr *atomic.Int64) time.Time {
	asint64 := addr.Load()
	return time.UnixMilli(asint64)
}

func atomicTimeWrite(addr *atomic.Int64, t time.Time) {
	asint64 := t.UnixMilli()
	addr.Store(asint64)
}
