package pubsub

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const (
	messageKey   = "msg"
	defaultGroup = "default_consumer_group"
)

type Marshallable[T any] interface {
	Marshal() any
	Unmarshal(val any) (T, error)
}

type Producer[T Marshallable[T]] struct {
	stopwaiter.StopWaiter
	id     string
	client redis.UniversalClient
	cfg    *ProducerConfig

	promisesLock sync.RWMutex
	promises     map[string]*containers.Promise[T]
}

type ProducerConfig struct {
	RedisURL string `koanf:"redis-url"`
	// Redis stream name.
	RedisStream string `koanf:"redis-stream"`
	// Interval duration in which producer checks for pending messages delivered
	// to the consumers that are currently inactive.
	CheckPendingInterval time.Duration `koanf:"check-pending-interval"`
	// Duration after which consumer is considered to be dead if heartbeat
	// is not updated.
	KeepAliveTimeout time.Duration `koanf:"keepalive-timeout"`
	// Interval duration for checking the result set by consumers.
	CheckResultInterval time.Duration `koanf:"check-result-interval"`
	// Redis consumer group name.
	RedisGroup string `koanf:"redis-group"`
}

func NewProducer[T Marshallable[T]](cfg *ProducerConfig) (*Producer[T], error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	c, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	return &Producer[T]{
		id:       uuid.NewString(),
		client:   c,
		cfg:      cfg,
		promises: make(map[string]*containers.Promise[T]),
	}, nil
}

func (p *Producer[T]) Start(ctx context.Context) {
	p.StopWaiter.Start(ctx, p)
	p.StopWaiter.CallIteratively(
		func(ctx context.Context) time.Duration {
			msgs, err := p.checkPending(ctx)
			if err != nil {
				log.Error("Checking pending messages", "error", err)
				return p.cfg.CheckPendingInterval
			}
			if len(msgs) == 0 {
				return p.cfg.CheckPendingInterval
			}
			acked := make(map[string]T)
			for _, msg := range msgs {
				if _, err := p.client.XAck(ctx, p.cfg.RedisStream, p.cfg.RedisGroup, msg.ID).Result(); err != nil {
					log.Error("ACKing message", "error", err)
					continue
				}
				acked[msg.ID] = msg.Value
			}
			for k, v := range acked {
				// Only re-insert messages that were removed the the pending list first.
				_, err := p.reproduce(ctx, v, k)
				if err != nil {
					log.Error("Re-inserting pending messages with inactive consumers", "error", err)
				}
			}
			return p.cfg.CheckPendingInterval
		},
	)
	// Iteratively check whether result were returned for some queries.
	p.StopWaiter.CallIteratively(func(ctx context.Context) time.Duration {
		p.promisesLock.Lock()
		defer p.promisesLock.Unlock()
		for id, promise := range p.promises {
			res, err := p.client.Get(ctx, id).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				log.Error("Error reading value in redis", "key", id, "error", err)
			}
			var tmp T
			val, err := tmp.Unmarshal(res)
			if err != nil {
				log.Error("Error unmarshaling", "value", res, "error", err)
				continue
			}
			promise.Produce(val)
			delete(p.promises, id)
		}
		return p.cfg.CheckResultInterval
	})
}

// reproduce is used when Producer claims ownership on the pending
// message that was sent to inactive consumer and reinserts it into the stream,
// so that seamlessly return the answer in the same promise.
func (p *Producer[T]) reproduce(ctx context.Context, value T, oldKey string) (*containers.Promise[T], error) {
	id, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.cfg.RedisStream,
		Values: map[string]any{messageKey: value.Marshal()},
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("adding values to redis: %w", err)
	}
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	promise := p.promises[oldKey]
	if oldKey == "" || promise == nil {
		p := containers.NewPromise[T](nil)
		promise = &p
	}
	p.promises[id] = promise
	return promise, nil
}

func (p *Producer[T]) Produce(ctx context.Context, value T) (*containers.Promise[T], error) {
	return p.reproduce(ctx, value, "")
}

// Check if a consumer is with specified ID is alive.
func (p *Producer[T]) isConsumerAlive(ctx context.Context, consumerID string) bool {
	val, err := p.client.Get(ctx, heartBeatKey(consumerID)).Int64()
	if err != nil {
		return false
	}
	return time.Now().UnixMilli()-val < int64(p.cfg.KeepAliveTimeout.Milliseconds())
}

func (p *Producer[T]) checkPending(ctx context.Context) ([]*Message[T], error) {
	pendingMessages, err := p.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: p.cfg.RedisStream,
		Group:  p.cfg.RedisGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("querying pending messages: %w", err)
	}
	if len(pendingMessages) == 0 {
		return nil, nil
	}
	// IDs of the pending messages with inactive consumers.
	var ids []string
	inactive := make(map[string]bool)
	for _, msg := range pendingMessages {
		if !inactive[msg.Consumer] || p.isConsumerAlive(ctx, msg.Consumer) {
			continue
		}
		inactive[msg.Consumer] = true
		ids = append(ids, msg.ID)
	}
	log.Info("Attempting to claim", "messages", ids)
	claimedMsgs, err := p.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   p.cfg.RedisStream,
		Group:    p.cfg.RedisGroup,
		Consumer: p.id,
		MinIdle:  p.cfg.KeepAliveTimeout,
		Messages: ids,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("claiming ownership on messages: %v, error: %w", ids, err)
	}
	var res []*Message[T]
	for _, msg := range claimedMsgs {
		var tmp T
		val, err := tmp.Unmarshal(msg.Values[messageKey])
		if err != nil {
			return nil, fmt.Errorf("marshaling value: %v, error: %v", msg.Values[messageKey], err)
		}
		res = append(res, &Message[T]{
			ID:    msg.ID,
			Value: val,
		})
	}
	return res, nil
}