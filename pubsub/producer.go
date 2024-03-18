package pubsub

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

const msgKey = "msg"

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
	streamName string
	client     *redis.Client
}

func NewProducer(streamName string, url string) (*Producer, error) {
	c, err := clientFromURL(url)
	if err != nil {
		return nil, err
	}
	return &Producer{
		streamName: streamName,
		client:     c,
	}, nil
}

func (p *Producer) Produce(ctx context.Context, value any) error {
	if _, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.streamName,
		Values: map[string]any{msgKey: value},
	}).Result(); err != nil {
		return fmt.Errorf("adding values to redis: %w", err)
	}
	return nil
}
