package arbnode

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/pkg/errors"
)

const CHOSENSEQ_KEY string = "lockout.chosen"              // Never overwritten. Expires or released only
const MSG_COUNT_KEY string = "lockout.msgCount"            // Only written by sequencer holding CHOSEN key
const PRIORITIES_KEY string = "lockout.priorities"         // Read only
const LIVELINESS_KEY_PREFIX string = "lockout.liveliness." // Per server. Only written by self
const LIVELINESS_VAL string = "OK"

type SeqCoordinator struct {
	streamer  *TransactionStreamer
	sequencer *Sequencer
	client    *redis.Client
	config    SeqConfig

	knownChosenSequencer string

	lockoutUntill int64 // atomic
	aliveUntill   int64 // atomic

	chanSeqNotifier chan struct{}
}

type SeqConfig struct {
	lockoutDuration time.Duration
	lockoutSpare    time.Duration
	seqNumDuration  time.Duration
	updateInterval  time.Duration
	retryInterval   time.Duration
	allowedMsgLag   uint64 // will only be marked live if not too far behind
	myUrl           string
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

func (c *SeqCoordinator) chosenOneUpdate(ctx context.Context, msgCount uint64) error {
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
		if current != c.config.myUrl {
			return fmt.Errorf("unexpected chosen sequencer: %s", current)
		}
		lockoutTill := time.Now().Add(c.config.lockoutDuration)
		pipe := tx.TxPipeline()
		if wasEmpty {
			pipe.Set(ctx, CHOSENSEQ_KEY, c.config.myUrl, c.config.lockoutDuration)
		}
		pipe.Set(ctx, MSG_COUNT_KEY, msgCount, c.config.seqNumDuration)
		myLivelinessKey := livelinessKeyFor(c.config.myUrl)
		pipe.Set(ctx, myLivelinessKey, LIVELINESS_VAL, c.config.lockoutDuration)
		pipe.ExpireAt(ctx, CHOSENSEQ_KEY, lockoutTill)
		pipe.ExpireAt(ctx, myLivelinessKey, lockoutTill)
		_, err = pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("chosen sequencer failed to update redis: %w", err)
		}
		atomicTimeWrite(&c.lockoutUntill, lockoutTill.Add(-c.config.lockoutSpare))
		return nil
	}, CHOSENSEQ_KEY)
}

func (c *SeqCoordinator) getMsgCount(ctx context.Context) (uint64, error) {
	res := c.client.Get(ctx, MSG_COUNT_KEY)
	resErr := res.Err()
	if errors.Is(resErr, redis.Nil) {
		return 0, nil
	}
	if resErr != nil {
		return 0, resErr
	}
	return res.Uint64()
}

func (c *SeqCoordinator) livelinessUpdate(ctx context.Context) error {
	myLivelinessKey := livelinessKeyFor(c.config.myUrl)
	aliveTill := time.Now().Add(c.config.lockoutDuration)
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, myLivelinessKey, LIVELINESS_VAL, c.config.lockoutDuration)
	pipe.ExpireAt(ctx, myLivelinessKey, aliveTill)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("liveliness failed to update redis: %w", err)
	}
	atomicTimeWrite(&c.aliveUntill, aliveTill.Add(-c.config.lockoutSpare))
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
		if current != c.config.myUrl {
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
	if current != c.config.myUrl {
		return nil
	}
	return releaseErr
}

func (c *SeqCoordinator) livelinessRelease(ctx context.Context) error {
	myLivelinessKey := livelinessKeyFor(c.config.myUrl)
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
	messageCount, err := c.streamer.GetMessageCount()
	if err != nil {
		log.Crit("cannot read message count", "err", err)
		return err
	}
	if chosenSeq == c.config.myUrl {
		if c.knownChosenSequencer != c.config.myUrl {
			upstreamMsgCount, err := c.getMsgCount(ctx)
			if err != nil {
				return err
			}
			if upstreamMsgCount > messageCount {
				// wait till we have all messages
				return nil
			}
		}
		err = c.chosenOneUpdate(ctx, messageCount)
		if err != nil {
			return err
		}
		if c.knownChosenSequencer != c.config.myUrl {
			c.sequencer.DontForward()
			c.knownChosenSequencer = c.config.myUrl
		}
	} else {
		if c.knownChosenSequencer != chosenSeq {
			c.sequencer.ForwardTo(chosenSeq)
			if c.knownChosenSequencer == c.config.myUrl {
				atomic.StoreInt64(&c.lockoutUntill, 0)
				err := c.chosenOneRelease(ctx)
				if err != nil {
					return err
				}
			}
			c.knownChosenSequencer = chosenSeq
		}
		upstreamMsgCount, err := c.getMsgCount(ctx)
		if err != nil {
			return err
		}
		if messageCount+c.config.allowedMsgLag < upstreamMsgCount {
			err := c.livelinessRelease(ctx)
			if err != nil {
				return err
			}
		}
		if err := c.livelinessUpdate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// TODO: will be improved with implementation of StopWaiter
func (c *SeqCoordinator) Start(ctx context.Context) {
	go func() {
		timesFailed := 0
		for {
			err := c.notifyRedis(ctx)
			if err != nil {
				log.Warn("sequencer coordinator error", "err", err)
				timesFailed++
			} else {
				timesFailed = 0
			}
			var nextInterval time.Duration
			if timesFailed == 0 {
				nextInterval = c.config.updateInterval
			} else {
				nextInterval = c.config.retryInterval * time.Duration(timesFailed)
			}
			timer := time.NewTimer(nextInterval)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
			case <-c.chanSeqNotifier:
				timer.Stop()
			}
		}
	}()
}

var errNotMainSequencer = errors.New("not main sequencer")

func (c *SeqCoordinator) SequencingMessage(pos uint64, msg *arbstate.MessageWithMetadata) error {
	if time.Now().After(atomicTimeRead(&c.lockoutUntill)) {
		return errNotMainSequencer
	}
	select {
	case c.chanSeqNotifier <- struct{}{}:
	default:
	}
	return nil
}
