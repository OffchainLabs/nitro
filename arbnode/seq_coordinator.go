package arbnode

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/util"
)

const CHOSENSEQ_KEY string = "lockout.chosen"              // Never overwritten. Expires or released only
const MSG_COUNT_KEY string = "lockout.msgCount"            // Only written by sequencer holding CHOSEN key
const PRIORITIES_KEY string = "lockout.priorities"         // Read only
const LIVELINESS_KEY_PREFIX string = "lockout.liveliness." // Per server. Only written by self
const LIVELINESS_VAL string = "OK"

type SeqCoordinator struct {
	util.StopWaiter

	streamer  *TransactionStreamer
	sequencer *Sequencer
	client    redis.UniversalClient
	config    SeqCoordinatorConfig

	knownChosenSequencer string
	remoteMsgCount       arbutil.MessageIndex

	lockoutUntil int64 // atomic
	aliveUntil   int64 // atomic

	chanSeqNotifier chan struct{}
}

type SeqCoordinatorConfig struct {
	LockoutDuration time.Duration
	LockoutSpare    time.Duration
	SeqNumDuration  time.Duration
	UpdateInterval  time.Duration
	RetryInterval   time.Duration
	AllowedMsgLag   arbutil.MessageIndex // will only be marked live if not too far behind
	MyUrl           string
}

var DefaultSeqCoordinatorConfig = SeqCoordinatorConfig{
	LockoutDuration: time.Duration(5) * time.Minute,
	LockoutSpare:    time.Duration(30) * time.Second,
	SeqNumDuration:  time.Duration(24) * time.Hour,
	UpdateInterval:  time.Duration(10) * time.Second,
	RetryInterval:   time.Second,
	AllowedMsgLag:   arbutil.MessageIndex(50),
}

var TestSeqCoordinatorConfig = SeqCoordinatorConfig{
	LockoutDuration: time.Millisecond * 500,
	LockoutSpare:    time.Millisecond * 10,
	SeqNumDuration:  time.Minute * 10,
	UpdateInterval:  time.Millisecond * 10,
	RetryInterval:   time.Millisecond * 3,
	AllowedMsgLag:   3,
}

func NewSeqCoordinator(streamer *TransactionStreamer, sequencer *Sequencer, client redis.UniversalClient, config SeqCoordinatorConfig) *SeqCoordinator {
	coordinator := &SeqCoordinator{
		streamer:        streamer,
		sequencer:       sequencer,
		client:          client,
		config:          config,
		chanSeqNotifier: make(chan struct{}),
	}
	streamer.SetSeqCoordinator(coordinator)
	return coordinator
}

func (c *SeqCoordinator) chooseLiveSequencer(ctx context.Context) (string, error) {
	prioritiesString, err := c.client.Get(ctx, PRIORITIES_KEY).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
		return "", err
	}
	priorities := strings.Split(prioritiesString, ",")
	for _, url := range priorities {
		err := c.client.Get(ctx, livelinessKeyFor(url)).Err()
		if errors.Is(err, redis.Nil) { // liveliness not set
			continue
		}
		if err != nil {
			return "", err
		}
		return url, nil
	}
	log.Info("not found sequencer", "priorities", prioritiesString, "self", c.config.MyUrl)
	return "", nil
}

func atomicTimeWrite(addr *int64, t time.Time) {
	asint64 := t.UnixMilli()
	atomic.StoreInt64(addr, asint64)
}

func atomicTimeRead(addr *int64) time.Time {
	asint64 := atomic.LoadInt64(addr)
	return time.UnixMilli(asint64)
}

func livelinessKeyFor(url string) string { return LIVELINESS_KEY_PREFIX + url }

func (c *SeqCoordinator) chosenOneUpdate(ctx context.Context, msgCount arbutil.MessageIndex) error {
	return c.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, CHOSENSEQ_KEY).Result()
		var wasEmpty bool
		if errors.Is(err, redis.Nil) {
			wasEmpty = true
			err = nil
		}
		if err != nil {
			return err
		}
		if !wasEmpty && (current != c.config.MyUrl) {
			return fmt.Errorf("unexpected chosen sequencer: %s", current)
		}
		remoteMsgCount, err := tx.Get(ctx, MSG_COUNT_KEY).Int64()
		if !errors.Is(err, redis.Nil) {
			if err != nil {
				return err
			}
			if arbutil.MessageIndex(remoteMsgCount) > c.remoteMsgCount {
				return fmt.Errorf("found message count %d > %d", remoteMsgCount, c.remoteMsgCount)
			}
		}
		lockoutUntil := time.Now().Add(c.config.LockoutDuration)
		pipe := tx.TxPipeline()
		initialDuration := c.config.LockoutDuration
		if initialDuration < 2*time.Second {
			initialDuration = 2 * time.Second
		}
		if wasEmpty {
			pipe.Set(ctx, CHOSENSEQ_KEY, c.config.MyUrl, initialDuration)
		}
		pipe.Set(ctx, MSG_COUNT_KEY, strconv.FormatUint(uint64(msgCount), 10), c.config.SeqNumDuration)
		myLivelinessKey := livelinessKeyFor(c.config.MyUrl)
		pipe.Set(ctx, myLivelinessKey, LIVELINESS_VAL, initialDuration)
		pipe.PExpireAt(ctx, CHOSENSEQ_KEY, lockoutUntil)
		pipe.PExpireAt(ctx, myLivelinessKey, lockoutUntil)
		_, err = pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("chosen sequencer failed to update redis: %w", err)
		}
		atomicTimeWrite(&c.lockoutUntil, lockoutUntil.Add(-c.config.LockoutSpare))
		atomicTimeWrite(&c.aliveUntil, lockoutUntil.Add(-c.config.LockoutSpare))
		c.remoteMsgCount = msgCount
		return nil
	}, CHOSENSEQ_KEY, MSG_COUNT_KEY)
}

func (c *SeqCoordinator) GetRemoteMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	res := c.client.Get(ctx, MSG_COUNT_KEY)
	resErr := res.Err()
	if errors.Is(resErr, redis.Nil) {
		return 0, nil
	}
	if resErr != nil {
		return 0, resErr
	}
	resuint, err := res.Uint64()
	return arbutil.MessageIndex(resuint), err
}

func (c *SeqCoordinator) livelinessUpdate(ctx context.Context) error {
	myLivelinessKey := livelinessKeyFor(c.config.MyUrl)
	aliveUntil := time.Now().Add(c.config.LockoutDuration)
	pipe := c.client.TxPipeline()
	initialDuration := c.config.LockoutDuration
	if initialDuration < 2*time.Second {
		initialDuration = 2 * time.Second
	}
	pipe.Set(ctx, myLivelinessKey, LIVELINESS_VAL, initialDuration)
	pipe.PExpireAt(ctx, myLivelinessKey, aliveUntil)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("liveliness failed to update redis: %w", err)
	}
	atomicTimeWrite(&c.aliveUntil, aliveUntil.Add(-c.config.LockoutSpare))
	return nil
}

func (c *SeqCoordinator) chosenOneRelease(ctx context.Context) error {
	releaseErr := c.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, CHOSENSEQ_KEY).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return err
		}
		if current != c.config.MyUrl {
			return nil
		}
		pipe := tx.TxPipeline()
		pipe.Del(ctx, CHOSENSEQ_KEY)
		_, err = pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("chosen sequencer failed to update redis: %w", err)
		}
		return nil
	}, CHOSENSEQ_KEY)
	if releaseErr == nil {
		return nil
	}
	// got error - was it still released?
	current, readErr := c.client.Get(ctx, CHOSENSEQ_KEY).Result()
	if errors.Is(readErr, redis.Nil) {
		return nil
	}
	if current != c.config.MyUrl {
		return nil
	}
	return releaseErr
}

func (c *SeqCoordinator) livelinessRelease(ctx context.Context) error {
	myLivelinessKey := livelinessKeyFor(c.config.MyUrl)
	releaseErr := c.client.Del(ctx, myLivelinessKey).Err()
	if releaseErr == nil {
		return nil
	}
	// got error - was it still deleted?
	readErr := c.client.Get(ctx, myLivelinessKey).Err()
	if errors.Is(readErr, redis.Nil) {
		return nil
	}
	return releaseErr
}

func (c *SeqCoordinator) notifyRedis(ctx context.Context) error {
	chosenSeq, err := c.chooseLiveSequencer(ctx)
	if err != nil {
		return err
	}
	localMsgCount, err := c.streamer.GetMessageCount()
	if err != nil {
		log.Crit("cannot read message count", "err", err)
		return err
	}
	if chosenSeq == c.config.MyUrl {
		if c.knownChosenSequencer != c.config.MyUrl {
			remoteMsgCount, err := c.GetRemoteMsgCount(ctx)
			if err != nil {
				return err
			}
			if localMsgCount < remoteMsgCount {
				// we are not in sync with redis
				log.Info("chosen sequencer: still reading messages", "local", localMsgCount, "remote", remoteMsgCount)
				return nil
			}
			// chosenOneUpdate should succeed unless somebody else writes a higher messagecount
			c.remoteMsgCount = remoteMsgCount
		}
		if err := c.chosenOneUpdate(ctx, localMsgCount); err != nil {
			return err
		}
		c.remoteMsgCount = localMsgCount
		if c.knownChosenSequencer != c.config.MyUrl {
			c.sequencer.DontForward()
			c.knownChosenSequencer = c.config.MyUrl
		}
		return nil
	}
	if c.knownChosenSequencer != chosenSeq {
		c.sequencer.ForwardTo(chosenSeq)
		if c.knownChosenSequencer == c.config.MyUrl {
			atomic.StoreInt64(&c.lockoutUntil, 0)
			// make sure we updated message count in server to latest value
			localMsgCount, err = c.streamer.GetMessageCountSync()
			if err != nil {
				return err
			}
			if c.remoteMsgCount < localMsgCount {
				err := c.chosenOneUpdate(ctx, localMsgCount)
				if err != nil {
					return err
				}
			}
			err := c.chosenOneRelease(ctx)
			if err != nil {
				return err
			}
		}
		c.knownChosenSequencer = chosenSeq
	}
	remoteMsgCount, err := c.GetRemoteMsgCount(ctx)
	if err != nil {
		return err
	}
	if localMsgCount+c.config.AllowedMsgLag < remoteMsgCount {
		return c.livelinessRelease(ctx)
	}
	return c.livelinessUpdate(ctx)
}

func (c *SeqCoordinator) Start(ctxIn context.Context) {
	c.StopWaiter.Start(ctxIn)
	c.LaunchThread(func(ctx context.Context) {
		timesFailed := 0
		for {
			err := c.notifyRedis(ctx)
			if err != nil {
				log.Warn("sequencer coordinator error", "err", err)
				timesFailed++
			} else {
				log.Debug("sequencer coordinator no error", "self", c.config.MyUrl,
					"known chosen", c.knownChosenSequencer,
					"remoteMsgCount", c.remoteMsgCount,
					"alive until", c.aliveUntil,
					"lockoutUntil", c.lockoutUntil,
					"now", time.Now().UnixMilli())
				timesFailed = 0
			}
			var nextInterval time.Duration
			if timesFailed == 0 {
				nextInterval = c.config.UpdateInterval
			} else {
				nextInterval = c.config.RetryInterval * time.Duration(timesFailed)
			}
			timer := time.NewTimer(nextInterval)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			case <-c.chanSeqNotifier:
				timer.Stop()
			}
		}
	})
}

var ErrNotMainSequencer = errors.New("not main sequencer")

func (c *SeqCoordinator) SequencingMessage(pos arbutil.MessageIndex, msg *arbstate.MessageWithMetadata) error {
	if time.Now().After(atomicTimeRead(&c.lockoutUntil)) {
		return ErrNotMainSequencer
	}
	select {
	case c.chanSeqNotifier <- struct{}{}:
	default:
	}
	return nil
}
