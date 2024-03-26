package pubsub

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
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

type Consumer struct {
	stopwaiter.StopWaiter
	id                string
	streamName        string
	groupName         string
	client            *redis.Client
	keepAliveInterval time.Duration
	keepAliveTimeout  time.Duration
}

type Message struct {
	ID    string
	Value any
}

func NewConsumer(ctx context.Context, cfg *ConsumerConfig) (*Consumer, error) {
	c, err := clientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	consumer := &Consumer{
		id:                uuid.NewString(),
		streamName:        cfg.RedisStream,
		groupName:         cfg.RedisGroup,
		client:            c,
		keepAliveInterval: cfg.KeepAliveInterval,
		keepAliveTimeout:  cfg.KeepAliveTimeout,
	}
	return consumer, nil
}

func (c *Consumer) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
	c.StopWaiter.CallIteratively(
		func(ctx context.Context) time.Duration {
			c.heartBeat(ctx)
			return c.keepAliveInterval
		},
	)
}

func (c *Consumer) StopAndWait() {
	c.StopWaiter.StopAndWait()
}

func heartBeatKey(id string) string {
	return fmt.Sprintf("consumer:%s:heartbeat", id)
}

func (c *Consumer) heartBeatKey() string {
	return heartBeatKey(c.id)
}

// heartBeat updates the heartBeat key indicating aliveness.
func (c *Consumer) heartBeat(ctx context.Context) {
	if err := c.client.Set(ctx, c.heartBeatKey(), time.Now().UnixMilli(), c.keepAliveTimeout).Err(); err != nil {
		l := log.Error
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			l = log.Info
		}
		l("Updating heardbeat", "consumer", c.id, "error", err)
	}
}

// Consumer first checks it there exists pending message that is claimed by
// unresponsive consumer, if not then reads from the stream.
func (c *Consumer) Consume(ctx context.Context) (*Message, error) {
	res, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.groupName,
		Consumer: c.id,
		// Receive only messages that were never delivered to any other consumer,
		// that is, only new messages.
		Streams: []string{c.streamName, ">"},
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
	return &Message{
		ID:    res[0].Messages[0].ID,
		Value: res[0].Messages[0].Values[messageKey],
	}, nil
}

func (c *Consumer) ACK(ctx context.Context, messageID string) error {
	log.Info("ACKing message", "consumer-id", c.id, "message-sid", messageID)
	_, err := c.client.XAck(ctx, c.streamName, c.groupName, messageID).Result()
	return err
}
