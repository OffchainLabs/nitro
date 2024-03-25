package pubsub

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/spf13/pflag"
)

const pendingMessagesKey = "lock:pending"

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
}

func ConsumerConfigAddOptions(prefix string, f *pflag.FlagSet, cfg *ConsumerConfig) {
	f.Duration(prefix+".keepalive-interval", 30*time.Second, "interval in which consumer will perform heartbeat")
	f.Duration(prefix+".keepalive-timeout", 5*time.Minute, "timeout after which consumer is considered inactive if heartbeat wasn't performed")
	f.String(prefix+".redis-url", "", "redis url for redis stream")
	f.String(prefix+".redis-stream", "default", "redis stream name to read from")
	f.String(prefix+".redis-group", "default", "redis stream consumer group name")
}

type Consumer struct {
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
	id := uuid.NewString()

	consumer := &Consumer{
		id:                id,
		streamName:        cfg.RedisStream,
		groupName:         "default",
		client:            c,
		keepAliveInterval: cfg.KeepAliveInterval,
		keepAliveTimeout:  cfg.KeepAliveTimeout,
	}
	go consumer.keepAlive(ctx)
	return consumer, nil
}

func keepAliveKey(id string) string {
	return fmt.Sprintf("consumer:%s:heartbeat", id)
}

func (c *Consumer) keepAliveKey() string {
	return keepAliveKey(c.id)
}

// keepAlive polls in keepAliveIntervals and updates heartbeat entry for itself.
func (c *Consumer) keepAlive(ctx context.Context) {
	log.Info("Consumer polling for heartbeat updates", "id", c.id)
	for {
		if err := c.client.Set(ctx, c.keepAliveKey(), time.Now().UnixMilli(), c.keepAliveTimeout).Err(); err != nil {
			l := log.Error
			if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				l = log.Info
			}
			l("Updating heardbeat", "consumer", c.id, "error", err)
		}
		select {
		case <-ctx.Done():
			log.Info("Error keeping alive", "error", ctx.Err())
			return
		case <-time.After(c.keepAliveInterval):
		}
	}
}

// Consumer first checks it there exists pending message that is claimed by
// unresponsive consumer, if not then reads from the stream.
func (c *Consumer) Consume(ctx context.Context) (*Message, error) {
	log.Debug("Attempting to consume a message", "consumer-id", c.id)
	msg, err := c.checkPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("consumer: %v checking pending messages with unavailable consumer: %w", c.id, err)
	}
	if msg != nil {
		return msg, nil
	}
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
		Value: res[0].Messages[0].Values[msgKey],
	}, nil
}

func (c *Consumer) ACK(ctx context.Context, messageID string) error {
	log.Info("ACKing message", "consumer-id", c.id, "message-sid", messageID)
	_, err := c.client.XAck(ctx, c.streamName, c.groupName, messageID).Result()
	return err
}

// Check if a consumer is with specified ID is alive.
func (c *Consumer) isConsumerAlive(ctx context.Context, consumerID string) bool {
	val, err := c.client.Get(ctx, keepAliveKey(consumerID)).Int64()
	if err != nil {
		return false
	}
	return time.Now().UnixMilli()-val < 2*int64(c.keepAliveTimeout.Milliseconds())
}

func (c *Consumer) lockPending(ctx context.Context, consumerID string) bool {
	acquired, err := c.client.SetNX(ctx, pendingMessagesKey, consumerID, c.keepAliveInterval).Result()
	if err != nil || !acquired {
		return false
	}
	return true
}

func (c *Consumer) unlockPending(ctx context.Context) {
	log.Debug("Releasing lock", "consumer-id", c.id)
	c.client.Del(ctx, pendingMessagesKey)

}

// checkPending lists pending messages, and checks unavailable consumers that
// have ownership on pending message.
// If such message and consumer exists, it claims ownership on it.
func (c *Consumer) checkPending(ctx context.Context) (*Message, error) {
	// Locking pending list avoid the race where two instances query pending
	// list and try to claim ownership on the same message.
	if !c.lockPending(ctx, c.id) {
		return nil, nil
	}
	log.Info("Consumer acquired pending lock", "consumer=id", c.id)
	defer c.unlockPending(ctx)
	pendingMessages, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.streamName,
		Group:  c.groupName,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()
	log.Info("Pending messages", "consumer", c.id, "pendingMessages", pendingMessages, "error", err)

	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("querying pending messages: %w", err)
	}
	if len(pendingMessages) == 0 {
		return nil, nil
	}
	inactive := make(map[string]bool)
	for _, msg := range pendingMessages {
		if inactive[msg.Consumer] {
			continue
		}
		if c.isConsumerAlive(ctx, msg.Consumer) {
			continue
		}
		inactive[msg.Consumer] = true
		log.Info("Consumer is not alive", "id", msg.Consumer)
		msgs, err := c.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   c.streamName,
			Group:    c.groupName,
			Consumer: c.id,
			MinIdle:  c.keepAliveTimeout,
			Messages: []string{msg.ID},
		}).Result()
		if err != nil {
			log.Error("Error claiming ownership on message", "id", msg.ID, "consumer", c.id, "error", err)
			continue
		}
		if len(msgs) != 1 {
			log.Error("Attempted to claim ownership on single messsage", "id", msg.ID, "number of received messages", len(msgs))
			if len(msgs) == 0 {
				continue
			}
		}
		log.Info(fmt.Sprintf("Consumer: %s claimed ownership on message: %s", c.id, msgs[0].ID))
		return &Message{
			ID:    msgs[0].ID,
			Value: msgs[0].Values[msgKey],
		}, nil
	}
	return nil, nil
}
