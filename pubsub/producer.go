package pubsub

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
)

const msgKey = "msg"

var (
	// Interval in which producer polls for checking the response from the consumer.
	resultPollInterval = time.Minute
	// Timeout for polling on response from the consumer.
	resultPollTimeout = 18 * time.Hour
)

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

// Producer implements functionality to produce messages in a single redis stream.
type Producer struct {
	streamName string
	client     *redis.Client
}

// NewProducer returns a new producer from specified stream name and redis url.
func NewProducer(streamName string, url string) (*Producer, error) {
	c, err := clientFromURL(url)
	if err != nil {
		return nil, err
	}
	if resultPollInterval > resultPollTimeout {
		return nil, fmt.Errorf("polling interval (%v) can not be greater than polling timeout (%v)", resultPollInterval, resultPollTimeout)
	}
	return &Producer{
		streamName: streamName,
		client:     c,
	}, nil
}

// produce produces a message in a redis stream.
func (p *Producer) produce(ctx context.Context, value any) (string, error) {
	id, err := p.client.XAdd(ctx,
		&redis.XAddArgs{
			Stream: p.streamName,
			Values: map[string]any{msgKey: value},
		}).Result()
	if err != nil {
		return "", fmt.Errorf("adding values to redis: %w", err)
	}
	return id, nil
}

func (p *Producer) waitFor(ctx context.Context, id string) (string, error) {
	timeout := time.After(resultPollTimeout)
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for the consumer response %v", ctx.Err())
		case <-time.After(KeepAliveInterval):
			res, err := p.client.Get(ctx, id).Result()
			if err != nil {
				log.Debug("Waiting for the response for message", "id", id)
				continue
			}
			return res, nil
		case <-timeout:
			return "", fmt.Errorf("waiting for message: %s response timed out", id)
		}
	}
}

// ProduceAndWait produces a message (request) and waits until the consumer
// processes and returns the result.
func (p *Producer) ProduceAndWait(ctx context.Context, value any) (string, error) {
	id, err := p.produce(ctx, value)
	if err != nil {
		return "", fmt.Errorf("produceAndWait() producing value: %v", err)
	}
	return p.waitFor(ctx, id)
}
