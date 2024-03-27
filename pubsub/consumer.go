package pubsub

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/spf13/pflag"
)

type ConsumerConfig struct {
	// Intervals in which consumer will update heartbeat.
	KeepAliveInterval time.Duration `koanf:"keepalive-interval"`
	// Duration after which consumer is considered to be dead if heartbeat
	// is not updated.
	KeepAliveTimeout time.Duration `koanf:"keepalive-timeout"`
	// Redis url for Redis streams and locks.
	RedisURL string `koanf:"redis-url"`
	// Redis stream name.
	RedisStream string `koanf:"redis-stream"`
	// Redis consumer group name.
	RedisGroup string `koanf:"redis-group"`
}

func ConsumerConfigAddOptions(prefix string, f *pflag.FlagSet, cfg *ConsumerConfig) {
	f.Duration(prefix+".keepalive-interval", 30*time.Second, "interval in which consumer will perform heartbeat")
	f.Duration(prefix+".keepalive-timeout", 5*time.Minute, "timeout after which consumer is considered inactive if heartbeat wasn't performed")
	f.String(prefix+".redis-url", "", "redis url for redis stream")
	f.String(prefix+".redis-stream", "default", "redis stream name to read from")
	f.String(prefix+".redis-group", defaultGroup, "redis stream consumer group name")
}

// Consumer implements a consumer for redis stream provides heartbeat to
// indicate it is alive.
type Consumer[T Marshallable[T]] struct {
	stopwaiter.StopWaiter
	id     string
	client redis.UniversalClient
	cfg    *ConsumerConfig
}

type Message[T Marshallable[T]] struct {
	ID    string
	Value T
}

func NewConsumer[T Marshallable[T]](ctx context.Context, cfg *ConsumerConfig) (*Consumer[T], error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	c, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	consumer := &Consumer[T]{
		id:     uuid.NewString(),
		client: c,
		cfg:    cfg,
	}
	return consumer, nil
}

// Start starts the consumer to iteratively perform heartbeat in configured intervals.
func (c *Consumer[T]) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
	c.StopWaiter.CallIteratively(
		func(ctx context.Context) time.Duration {
			c.heartBeat(ctx)
			return c.cfg.KeepAliveInterval
		},
	)
}

func (c *Consumer[T]) StopAndWait() {
	c.StopWaiter.StopAndWait()
}

func heartBeatKey(id string) string {
	return fmt.Sprintf("consumer:%s:heartbeat", id)
}

func (c *Consumer[T]) heartBeatKey() string {
	return heartBeatKey(c.id)
}

// heartBeat updates the heartBeat key indicating aliveness.
func (c *Consumer[T]) heartBeat(ctx context.Context) {
	if err := c.client.Set(ctx, c.heartBeatKey(), time.Now().UnixMilli(), 2*c.cfg.KeepAliveTimeout).Err(); err != nil {
		l := log.Info
		if ctx.Err() != nil {
			l = log.Error
		}
		l("Updating heardbeat", "consumer", c.id, "error", err)
	}
}

// Consumer first checks it there exists pending message that is claimed by
// unresponsive consumer, if not then reads from the stream.
func (c *Consumer[T]) Consume(ctx context.Context) (*Message[T], error) {
	res, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.cfg.RedisGroup,
		Consumer: c.id,
		// Receive only messages that were never delivered to any other consumer,
		// that is, only new messages.
		Streams: []string{c.cfg.RedisStream, ">"},
		Count:   1,
		Block:   time.Millisecond, // 0 seems to block the read instead of immediately returning
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading message for consumer: %q: %w", c.id, err)
	}
	if len(res) != 1 || len(res[0].Messages) != 1 {
		return nil, fmt.Errorf("redis returned entries: %+v, for querying single message", res)
	}
	log.Debug(fmt.Sprintf("Consumer: %s consuming message: %s", c.id, res[0].Messages[0].ID))
	var (
		value = res[0].Messages[0].Values[messageKey]
		tmp   T
	)
	val, err := tmp.Unmarshal(value)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling value: %v, error: %w", value, err)
	}

	return &Message[T]{
		ID:    res[0].Messages[0].ID,
		Value: val,
	}, nil
}

func (c *Consumer[T]) ACK(ctx context.Context, messageID string) error {
	log.Info("ACKing message", "consumer-id", c.id, "message-sid", messageID)
	_, err := c.client.XAck(ctx, c.cfg.RedisStream, c.cfg.RedisGroup, messageID).Result()
	return err
}

func (c *Consumer[T]) SetResult(ctx context.Context, messageID string, result string) error {
	acquired, err := c.client.SetNX(ctx, messageID, result, c.cfg.KeepAliveTimeout).Result()
	if err != nil || !acquired {
		return fmt.Errorf("setting result for  message: %v, error: %w", messageID, err)
	}
	return nil
}
