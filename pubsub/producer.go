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
)

const (
	messageKey   = "msg"
	defaultGroup = "default_consumer_group"
)

// clientFromURL returns a redis client from url.
func clientFromURL(url string) (*redis.Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty redis url")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	c := redis.NewClient(opts)
	if c == nil {
		return nil, fmt.Errorf("redis returned nil client")
	}
	return c, nil
}

type Producer struct {
	stopwaiter.StopWaiter
	id     string
	client *redis.Client
	cfg    *ProducerConfig
	// streamName           string
	// groupName            string
	// checkPendingInterval time.Duration
	// keepAliveTimeout     time.Duration
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
	// Redis consumer group name.
	RedisGroup string `koanf:"redis-group"`
}

func NewProducer(cfg *ProducerConfig) (*Producer, error) {
	c, err := clientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	return &Producer{
		id:     uuid.NewString(),
		client: c,
		cfg:    cfg,
		// streamName:           cfg.RedisStream,
		// groupName:            cfg.RedisGroup,
		// checkPendingInterval: cfg.CheckPendingInterval,
		// keepAliveTimeout:     cfg.KeepAliveTimeout,
	}, nil
}

func (p *Producer) Start(ctx context.Context) {
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
			var acked []any
			for _, msg := range msgs {
				if _, err := p.client.XAck(ctx, p.cfg.RedisStream, p.cfg.RedisGroup, msg.ID).Result(); err != nil {
					log.Error("ACKing message", "error", err)
					continue
				}
				acked = append(acked, msg.Value)
			}
			// Only re-insert messages that were removed the the pending list first.
			if err := p.Produce(ctx, acked); err != nil {
				log.Error("Re-inserting pending messages with inactive consumers", "error", err)
			}
			return p.cfg.CheckPendingInterval
		},
	)
}

func (p *Producer) Produce(ctx context.Context, values ...any) error {
	if len(values) == 0 {
		return nil
	}
	for _, value := range values {
		if _, err := p.client.XAdd(ctx, &redis.XAddArgs{
			Stream: p.cfg.RedisStream,
			Values: map[string]any{messageKey: value},
		}).Result(); err != nil {
			return fmt.Errorf("adding values to redis: %w", err)
		}
	}
	return nil
}

// Check if a consumer is with specified ID is alive.
func (p *Producer) isConsumerAlive(ctx context.Context, consumerID string) bool {
	val, err := p.client.Get(ctx, heartBeatKey(consumerID)).Int64()
	if err != nil {
		return false
	}
	return time.Now().UnixMilli()-val < int64(p.cfg.KeepAliveTimeout.Milliseconds())
}

func (p *Producer) checkPending(ctx context.Context) ([]*Message, error) {
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
		if inactive[msg.Consumer] || p.isConsumerAlive(ctx, msg.Consumer) {
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
	var res []*Message
	for _, msg := range claimedMsgs {
		res = append(res, &Message{
			ID:    msg.ID,
			Value: msg.Values[messageKey],
		})
	}
	return res, nil
}
