package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
)

type ConsumerConfig struct {
	// Timeout of result entry in Redis.
	ResponseEntryTimeout time.Duration `koanf:"response-entry-timeout"`
	// Minimum idle time after which messages will be autoclaimed
	IdletimeToAutoclaim time.Duration `koanf:"Idletime-to-autoclaim"`
}

var DefaultConsumerConfig = ConsumerConfig{
	ResponseEntryTimeout: time.Hour,
	IdletimeToAutoclaim:  5 * time.Minute,
}

var TestConsumerConfig = ConsumerConfig{
	ResponseEntryTimeout: time.Minute,
	IdletimeToAutoclaim:  30 * time.Millisecond,
}

func ConsumerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".response-entry-timeout", DefaultConsumerConfig.ResponseEntryTimeout, "timeout for response entry")
	f.Duration(prefix+".Idletime-to-autoclaim", DefaultConsumerConfig.IdletimeToAutoclaim, "After a message spends this amount of time in PEL (Pending Entries List i.e claimed by another consumer but not Acknowledged) it will be allowed to be autoclaimed by other consumers")
}

// Consumer implements a consumer for redis stream provides heartbeat to
// indicate it is alive.
type Consumer[Request any, Response any] struct {
	stopwaiter.StopWaiter
	id           string
	client       redis.UniversalClient
	redisStream  string
	redisGroup   string
	cfg          *ConsumerConfig
	ackNotifiers map[string]chan struct{}
}

type Message[Request any] struct {
	ID    string
	Value Request
}

func NewConsumer[Request any, Response any](client redis.UniversalClient, streamName string, cfg *ConsumerConfig) (*Consumer[Request, Response], error) {
	if streamName == "" {
		return nil, fmt.Errorf("redis stream name cannot be empty")
	}
	return &Consumer[Request, Response]{
		id:           uuid.NewString(),
		client:       client,
		redisStream:  streamName,
		redisGroup:   streamName, // There is 1-1 mapping of redis stream and consumer group.
		cfg:          cfg,
		ackNotifiers: make(map[string]chan struct{}),
	}, nil
}

// Start starts the consumer to iteratively perform heartbeat in configured intervals.
func (c *Consumer[Request, Response]) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
}

func (c *Consumer[Request, Response]) StopAndWait() {
	c.StopWaiter.StopAndWait()
}

func (c *Consumer[Request, Response]) RedisClient() redis.UniversalClient {
	return c.client
}

func (c *Consumer[Request, Response]) StreamName() string {
	return c.redisStream
}

// Consumer first checks it there exists pending message that is claimed by
// unresponsive consumer, if not then reads from the stream.
func (c *Consumer[Request, Response]) Consume(ctx context.Context) (*Message[Request], error) {
	// First try to XAUTOCLAIM, this prioritizes processing PEL messages
	// that have been waiting for more than IdletimeToAutoclaim duration
	messages, _, err := c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Group:    c.redisGroup,
		Consumer: c.id,
		MinIdle:  c.cfg.IdletimeToAutoclaim, // Minimum idle time for messages to claim (in milliseconds)
		Stream:   c.redisStream,
		Start:    "0",
		Count:    1, // Limit the number of messages to claim
	}).Result()
	if len(messages) != 1 || err != nil {
		// Fallback to reading new messages
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
		messages = res[0].Messages
	}

	var (
		value    = messages[0].Values[messageKey]
		data, ok = (value).(string)
	)
	if !ok {
		return nil, fmt.Errorf("casting request to string: %w", err)
	}
	var req Request
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		return nil, fmt.Errorf("unmarshaling value: %v, error: %w", value, err)
	}
	ackNotifier := make(chan struct{})
	c.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			if err := c.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   c.redisStream,
				Group:    c.redisGroup,
				Consumer: c.id,
				MinIdle:  0,
				Messages: []string{messages[0].ID},
			}).Err(); err != nil {
				log.Error("error claiming message, it might be possible that other consumers might pick this request", "msgID", messages[0].ID)
			}
			select {
			case <-ackNotifier:
				return
			case <-ctx.Done():
				log.Info("Context done while claiming message to indicate hearbeat", "error", ctx.Err().Error())
				return
			case <-time.After(c.cfg.IdletimeToAutoclaim / 10):
			}
		}
	})
	c.ackNotifiers[messages[0].ID] = ackNotifier
	log.Debug("Redis stream consuming", "consumer_id", c.id, "message_id", messages[0].ID)
	return &Message[Request]{
		ID:    messages[0].ID,
		Value: req,
	}, nil
}

func (c *Consumer[Request, Response]) SetResult(ctx context.Context, id string, messageID string, result Response) error {
	if id == "" {
		log.Info("Request doesn't have a unique identifier (SelfHash field is not set), defaulting to using redis stream messageId", "msgId", messageID)
		id = messageID
	}
	resp, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}
	acquired, err := c.client.SetNX(ctx, MessageKeyFor(c.StreamName(), id), resp, c.cfg.ResponseEntryTimeout).Result()
	if err != nil || !acquired {
		return fmt.Errorf("setting result for message with message-id in stream: %v, unique request identifier: %v, error: %w", messageID, id, err)
	}
	if _, err := c.client.XAck(ctx, c.redisStream, c.redisGroup, messageID).Result(); err != nil {
		return fmt.Errorf("acking message: %v, error: %w", messageID, err)
	}
	if ackNotifier, found := c.ackNotifiers[messageID]; found {
		close(ackNotifier)
		delete(c.ackNotifiers, messageID)
	}
	return nil
}
