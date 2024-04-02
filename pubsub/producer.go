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
	"github.com/spf13/pflag"
)

const (
	messageKey   = "msg"
	defaultGroup = "default_consumer_group"
)

type Marshallable[T any] interface {
	Marshal() []byte
	Unmarshal(val []byte) (T, error)
}

type Producer[Request Marshallable[Request], Response Marshallable[Response]] struct {
	stopwaiter.StopWaiter
	id     string
	client redis.UniversalClient
	cfg    *ProducerConfig

	promisesLock sync.RWMutex
	promises     map[string]*containers.Promise[Response]

	// 	Used for running checks for pending messages with inactive consumers
	// and checking responses from consumers iteratively for the first time when
	// Produce is called.
	once sync.Once
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

var DefaultProducerConfig = &ProducerConfig{
	RedisStream:          "default",
	CheckPendingInterval: time.Second,
	KeepAliveTimeout:     5 * time.Minute,
	CheckResultInterval:  5 * time.Second,
	RedisGroup:           defaultGroup,
}

var DefaultTestProducerConfig = &ProducerConfig{
	RedisStream:          "default",
	RedisGroup:           defaultGroup,
	CheckPendingInterval: 10 * time.Millisecond,
	KeepAliveTimeout:     20 * time.Millisecond,
	CheckResultInterval:  5 * time.Millisecond,
}

func ProducerAddConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".redis-url", DefaultConsumerConfig.RedisURL, "redis url for redis stream")
	f.Duration(prefix+".response-entry-timeout", DefaultConsumerConfig.ResponseEntryTimeout, "timeout for response entry")
	f.Duration(prefix+".keepalive-timeout", DefaultConsumerConfig.KeepAliveTimeout, "timeout after which consumer is considered inactive if heartbeat wasn't performed")
	f.String(prefix+".redis-stream", DefaultConsumerConfig.RedisStream, "redis stream name to read from")
	f.String(prefix+".redis-group", DefaultConsumerConfig.RedisGroup, "redis stream consumer group name")
}

func NewProducer[Request Marshallable[Request], Response Marshallable[Response]](cfg *ProducerConfig) (*Producer[Request, Response], error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	c, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	return &Producer[Request, Response]{
		id:       uuid.NewString(),
		client:   c,
		cfg:      cfg,
		promises: make(map[string]*containers.Promise[Response]),
	}, nil
}

// checkAndReproduce reproduce pending messages that were sent to consumers
// that are currently inactive.
func (p *Producer[Request, Response]) checkAndReproduce(ctx context.Context) time.Duration {
	msgs, err := p.checkPending(ctx)
	if err != nil {
		log.Error("Checking pending messages", "error", err)
		return p.cfg.CheckPendingInterval
	}
	if len(msgs) == 0 {
		return p.cfg.CheckPendingInterval
	}
	acked := make(map[string]Request)
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
}

// checkResponses checks iteratively whether response for the promise is ready.
func (p *Producer[Request, Response]) checkResponses(ctx context.Context) time.Duration {
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
		var tmp Response
		val, err := tmp.Unmarshal([]byte(res))
		if err != nil {
			log.Error("Error unmarshaling", "value", res, "error", err)
			continue
		}
		promise.Produce(val)
		delete(p.promises, id)
	}
	return p.cfg.CheckResultInterval
}

func (p *Producer[Request, Response]) Start(ctx context.Context) {
	p.StopWaiter.Start(ctx, p)
}

// reproduce is used when Producer claims ownership on the pending
// message that was sent to inactive consumer and reinserts it into the stream,
// so that seamlessly return the answer in the same promise.
func (p *Producer[Request, Response]) reproduce(ctx context.Context, value Request, oldKey string) (*containers.Promise[Response], error) {
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
		pr := containers.NewPromise[Response](nil)
		promise = &pr
	}
	p.promises[id] = promise
	return promise, nil
}

func (p *Producer[Request, Response]) Produce(ctx context.Context, value Request) (*containers.Promise[Response], error) {
	p.once.Do(func() {
		p.StopWaiter.CallIteratively(p.checkAndReproduce)
		p.StopWaiter.CallIteratively(p.checkResponses)
	})
	return p.reproduce(ctx, value, "")
}

// Check if a consumer is with specified ID is alive.
func (p *Producer[Request, Response]) isConsumerAlive(ctx context.Context, consumerID string) bool {
	val, err := p.client.Get(ctx, heartBeatKey(consumerID)).Int64()
	if err != nil {
		return false
	}
	return time.Now().UnixMilli()-val < int64(p.cfg.KeepAliveTimeout.Milliseconds())
}

func (p *Producer[Request, Response]) checkPending(ctx context.Context) ([]*Message[Request], error) {
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
	active := make(map[string]bool)
	for _, msg := range pendingMessages {
		alive, found := active[msg.Consumer]
		if !found {
			alive = p.isConsumerAlive(ctx, msg.Consumer)
			active[msg.Consumer] = alive
		}
		if alive {
			continue
		}
		ids = append(ids, msg.ID)
	}
	if len(ids) == 0 {
		log.Info("There are no pending messages with inactive consumers")
		return nil, nil
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
	var res []*Message[Request]
	for _, msg := range claimedMsgs {
		data, ok := (msg.Values[messageKey]).([]byte)
		if !ok {
			return nil, fmt.Errorf("casting request to bytes: %w", err)
		}
		var tmp Request
		val, err := tmp.Unmarshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshaling value: %v, error: %w", msg.Values[messageKey], err)
		}
		res = append(res, &Message[Request]{
			ID:    msg.ID,
			Value: val,
		})
	}
	return res, nil
}
