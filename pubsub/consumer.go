package pubsub

import (
	"context"
	"encoding/json"
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
	// Timeout of result entry in Redis.
	ResponseEntryTimeout time.Duration `koanf:"response-entry-timeout"`
	// Duration after which consumer is considered to be dead if heartbeat
	// is not updated.
	KeepAliveTimeout time.Duration `koanf:"keepalive-timeout"`
}

var DefaultConsumerConfig = ConsumerConfig{
	ResponseEntryTimeout: time.Hour,
	KeepAliveTimeout:     5 * time.Minute,
}

var TestConsumerConfig = ConsumerConfig{
	ResponseEntryTimeout: time.Minute,
	KeepAliveTimeout:     30 * time.Millisecond,
}

func ConsumerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".response-entry-timeout", DefaultConsumerConfig.ResponseEntryTimeout, "timeout for response entry")
	f.Duration(prefix+".keepalive-timeout", DefaultConsumerConfig.KeepAliveTimeout, "timeout after which consumer is considered inactive if heartbeat wasn't performed")
}

// Consumer implements a consumer for redis stream provides heartbeat to
// indicate it is alive.
type Consumer[Request any, Response any] struct {
	stopwaiter.StopWaiter
	id          string
	client      redis.UniversalClient
	redisStream string
	redisGroup  string
	cfg         *ConsumerConfig
}

type Message[Request any] struct {
	ID    string
	Value Request
}

func NewConsumer[Request any, Response any](client redis.UniversalClient, streamName string, cfg *ConsumerConfig) (*Consumer[Request, Response], error) {
	if streamName == "" {
		return nil, fmt.Errorf("redis stream name cannot be empty")
	}
	consumer := &Consumer[Request, Response]{
		id:          uuid.NewString(),
		client:      client,
		redisStream: streamName,
		redisGroup:  streamName, // There is 1-1 mapping of redis stream and consumer group.
		cfg:         cfg,
	}
	return consumer, nil
}

// Start starts the consumer to iteratively perform heartbeat in configured intervals.
func (c *Consumer[Request, Response]) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
	c.StopWaiter.CallIteratively(
		func(ctx context.Context) time.Duration {
			c.heartBeat(ctx)
			return c.cfg.KeepAliveTimeout / 10
		},
	)
}

func (c *Consumer[Request, Response]) StopAndWait() {
	c.StopWaiter.StopAndWait()
}

func heartBeatKey(id string) string {
	return fmt.Sprintf("consumer:%s:heartbeat", id)
}

func (c *Consumer[Request, Response]) heartBeatKey() string {
	return heartBeatKey(c.id)
}

// heartBeat updates the heartBeat key indicating aliveness.
func (c *Consumer[Request, Response]) heartBeat(ctx context.Context) {
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
func (c *Consumer[Request, Response]) Consume(ctx context.Context) (*Message[Request], error) {
	res, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.redisGroup,
		Consumer: c.id,
		// Receive only messages that were never delivered to any other consumer,
		// that is, only new messages.
		Streams: []string{c.redisStream, ">"},
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
	var (
		value    = res[0].Messages[0].Values[messageKey]
		data, ok = (value).(string)
	)
	if !ok {
		return nil, fmt.Errorf("casting request to string: %w", err)
	}
	var req Request
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		return nil, fmt.Errorf("unmarshaling value: %v, error: %w", value, err)
	}
	log.Debug("Redis stream consuming", "consumer_id", c.id, "message_id", res[0].Messages[0].ID)
	return &Message[Request]{
		ID:    res[0].Messages[0].ID,
		Value: req,
	}, nil
}

func (c *Consumer[Request, Response]) SetResult(ctx context.Context, messageID string, result Response) error {
	resp, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}
	acquired, err := c.client.SetNX(ctx, messageID, resp, c.cfg.ResponseEntryTimeout).Result()
	if err != nil || !acquired {
		return fmt.Errorf("setting result for  message: %v, error: %w", messageID, err)
	}
	if _, err := c.client.XAck(ctx, c.redisStream, c.redisGroup, messageID).Result(); err != nil {
		return fmt.Errorf("acking message: %v, error: %w", messageID, err)
	}
	return nil
}
