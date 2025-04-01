package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ConsumerConfig struct {
	// Timeout of result entry in Redis.
	ResponseEntryTimeout time.Duration `koanf:"response-entry-timeout"`
	// Minimum idle time after which messages will be autoclaimed
	IdletimeToAutoclaim time.Duration `koanf:"idletime-to-autoclaim"`
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
	f.Duration(prefix+".idletime-to-autoclaim", DefaultConsumerConfig.IdletimeToAutoclaim, "After a message spends this amount of time in PEL (Pending Entries List i.e claimed by another consumer but not Acknowledged) it will be allowed to be autoclaimed by other consumers")
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
	Ack   func()
}

func NewConsumer[Request any, Response any](client redis.UniversalClient, streamName string, cfg *ConsumerConfig) (*Consumer[Request, Response], error) {
	if streamName == "" {
		return nil, fmt.Errorf("redis stream name cannot be empty")
	}
	return &Consumer[Request, Response]{
		id:          uuid.NewString(),
		client:      client,
		redisStream: streamName,
		redisGroup:  streamName, // There is 1-1 mapping of redis stream and consumer group.
		cfg:         cfg,
	}, nil
}

// Start starts the consumer to iteratively perform heartbeat in configured intervals.
func (c *Consumer[Request, Response]) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
}

func (c *Consumer[Request, Response]) Id() string {
	return c.id
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

func decrementMsgIdByOne(msgId string) string {
	id, err := getUintParts(msgId)
	if err != nil {
		log.Error("Error decrementing start of XAutoClaim by one, defaulting to 0", "err", err)
		return "0"
	}
	if id[1] > 0 {
		return strconv.FormatUint(id[0], 10) + "-" + strconv.FormatUint(id[1]-1, 10)
	} else if id[0] > 0 {
		return strconv.FormatUint(id[0]-1, 10) + "-" + strconv.FormatUint(math.MaxUint64, 10)
	}
	return "0"
}

// Consumer first checks it there exists pending message that is claimed by
// unresponsive consumer, if not then reads from the stream.
func (c *Consumer[Request, Response]) Consume(ctx context.Context) (*Message[Request], error) {
	// First try to XAUTOCLAIM, with start as a random messageID from PEL with MinIdle as IdletimeToAutoclaim
	// this prioritizes processing PEL messages that have been waiting for more than IdletimeToAutoclaim duration
	var messages []redis.XMessage
	if pendingMsgs, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.redisStream,
		Group:  c.redisGroup,
		Start:  "-",
		End:    "+",
		Count:  50,
		Idle:   c.cfg.IdletimeToAutoclaim,
	}).Result(); err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Error("Error from XpendingExt in getting PEL for auto claim", "err", err, "penindlen", len(pendingMsgs))
		}
	} else if len(pendingMsgs) > 0 {
		idx := rand.Intn(len(pendingMsgs))
		messages, _, err = c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Group:    c.redisGroup,
			Consumer: c.id,
			MinIdle:  c.cfg.IdletimeToAutoclaim, // Minimum idle time for messages to claim (in milliseconds)
			Stream:   c.redisStream,
			Start:    decrementMsgIdByOne(pendingMsgs[idx].ID),
			Count:    1,
		}).Result()
		if err != nil {
			log.Info("error from xautoclaim", "err", err)
		}
	}
	if len(messages) == 0 {
		// If we fail to autoclaim then we do not retry but instead fallback to reading new messages
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
		return nil, errors.New("error casting request to string")
	}
	var req Request
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		return nil, fmt.Errorf("unmarshaling value: %v, error: %w", value, err)
	}
	ackNotifier := make(chan struct{})
	c.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			// Use XClaimJustID so that we would have clear difference between invalid requests that are claimed multiple times due to xautoclaim and
			// valid requests that are just being claimed in regular intervals to indicate heartbeat
			if ids, err := c.client.XClaimJustID(ctx, &redis.XClaimArgs{
				Stream:   c.redisStream,
				Group:    c.redisGroup,
				Consumer: c.id,
				MinIdle:  0,
				Messages: []string{messages[0].ID},
			}).Result(); err != nil {
				log.Error("Error claiming message, it might be possible that other consumers might pick this request", "msgID", messages[0].ID)
			} else if len(ids) == 0 {
				log.Warn("XClaimJustID returned empty response when indicating hearbeat", "msgID", messages[0].ID)
			} else if len(ids) > 1 {
				log.Error("XClaimJustID returned response with more than entry", "msgIDs", ids)
			}
			select {
			case <-ackNotifier:
				return
			case <-ctx.Done():
				log.Info("Context done while claiming message to indicate hearbeat", "messageID", messages[0].ID, "error", ctx.Err().Error())
				if c.StopWaiter.GetParentContext().Err() == nil {
					// Proceeding to set the Idle time of message to IdletimeToAutoclaim to allow it to be picked by other consumers
					if err := c.client.Do(c.StopWaiter.GetParentContext(), "XCLAIM", c.redisStream, c.redisGroup, c.id, 0, messages[0].ID, "IDLE", c.cfg.IdletimeToAutoclaim.Milliseconds()).Err(); err != nil {
						log.Error("error when trying to set the idle time of currently worked on message to IdletimeToAutoclaim", "messageID", messages[0].ID, "err", err)
					}
				}
				return
			case <-time.After(c.cfg.IdletimeToAutoclaim / 10):
			}
		}
	})
	log.Debug("Redis stream consuming", "consumer_id", c.id, "message_id", messages[0].ID)
	return &Message[Request]{
		ID:    messages[0].ID,
		Value: req,
		Ack:   func() { close(ackNotifier) },
	}, nil
}

func (c *Consumer[Request, Response]) SetResult(ctx context.Context, messageID string, result Response) error {
	resp, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}
	resultKey := ResultKeyFor(c.StreamName(), messageID)
	log.Debug("consumer: setting result", "cid", c.id, "msgIdInStream", messageID, "resultKeyInRedis", resultKey)
	acquired, err := c.client.SetNX(ctx, resultKey, resp, c.cfg.ResponseEntryTimeout).Result()
	if err != nil || !acquired {
		return fmt.Errorf("setting result for message with message-id in stream: %v, error: %w", messageID, err)
	}
	log.Debug("consumer: xack", "cid", c.id, "messageId", messageID)
	if _, err := c.client.XAck(ctx, c.redisStream, c.redisGroup, messageID).Result(); err != nil {
		return fmt.Errorf("acking message: %v, error: %w", messageID, err)
	}
	if _, err := c.client.XDel(ctx, c.redisStream, messageID).Result(); err != nil {
		return fmt.Errorf("deleting message: %v, error: %w", messageID, err)
	}
	return nil
}
