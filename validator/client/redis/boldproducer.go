package redis

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator/server_api"
)

// BoldValidationClient implements bold validation client through redis streams.
type BoldValidationClient struct {
	stopwaiter.StopWaiter
	// producers stores moduleRoot to producer mapping.
	producers map[common.Hash]*pubsub.Producer[*server_api.GetLeavesWithStepSizeInput, []common.Hash]
	config    *ValidationClientConfig
}

func NewBoldValidationClient(cfg *ValidationClientConfig) (*BoldValidationClient, error) {
	return &BoldValidationClient{
		producers: make(map[common.Hash]*pubsub.Producer[*server_api.GetLeavesWithStepSizeInput, []common.Hash]),
		config:    cfg,
	}, nil
}

func (c *BoldValidationClient) Initialize(ctx context.Context, moduleRoots []common.Hash) error {
	if c.config.RedisURL == "" {
		return fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(c.config.RedisURL)
	if err != nil {
		return err
	}
	for _, mr := range moduleRoots {
		if c.config.CreateStreams {
			if err := pubsub.CreateStream(ctx, server_api.RedisBoldStreamForRoot(c.config.StreamPrefix, mr), redisClient); err != nil {
				return fmt.Errorf("creating redis stream: %w", err)
			}
		}
		if _, exists := c.producers[mr]; exists {
			log.Warn("Producer already exists for module root", "hash", mr)
			continue
		}
		p, err := pubsub.NewProducer[*server_api.GetLeavesWithStepSizeInput, []common.Hash](
			redisClient, server_api.RedisBoldStreamForRoot(c.config.StreamPrefix, mr), &c.config.ProducerConfig)
		if err != nil {
			log.Warn("failed init redis for %v: %w", mr, err)
			continue
		}
		c.producers[mr] = p
	}
	return nil
}

func (c *BoldValidationClient) GetLeavesWithStepSize(req *server_api.GetLeavesWithStepSizeInput) containers.PromiseInterface[[]common.Hash] {
	producer, found := c.producers[req.ModuleRoot]
	if !found {
		return containers.NewReadyPromise([]common.Hash{}, fmt.Errorf("no validation is configured for wasm root %v", req.ModuleRoot))
	}
	promise, err := producer.Produce(c.GetContext(), req)
	if err != nil {
		return containers.NewReadyPromise([]common.Hash{}, fmt.Errorf("error producing input: %w", err))
	}
	return promise
}

func (c *BoldValidationClient) Start(ctx_in context.Context) error {
	for _, p := range c.producers {
		p.Start(ctx_in)
	}
	c.StopWaiter.Start(ctx_in, c)
	return nil
}

func (c *BoldValidationClient) Stop() {
	for _, p := range c.producers {
		p.StopAndWait()
	}
	c.StopWaiter.StopAndWait()
}
