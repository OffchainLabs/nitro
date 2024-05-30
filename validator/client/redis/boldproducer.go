package redis

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/spf13/pflag"
)

type BoldValidationClientConfig struct {
	RedisURL string `koanf:"redis-url"`
}

func (c BoldValidationClientConfig) Enabled() bool {
	return c.RedisURL != ""
}

var DefaultBoldValidationClientConfig = BoldValidationClientConfig{
	RedisURL: "",
}

var TestBoldValidationClientConfig = BoldValidationClientConfig{
	RedisURL: "",
}

func BoldValidationClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ProducerAddConfigAddOptions(prefix+".producer-config", f)
}

// BoldValidationClient implements bold validation client through redis streams.
type BoldValidationClient struct {
	stopwaiter.StopWaiter
	// producers stores moduleRoot to producer mapping.
	producers      map[common.Hash]*pubsub.Producer[*server_api.GetLeavesWithStepSizeInput, []common.Hash]
	producerConfig pubsub.ProducerConfig
	redisClient    redis.UniversalClient
	moduleRoots    []common.Hash
}

func NewBoldValidationClient(cfg *BoldValidationClientConfig) (*BoldValidationClient, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	return &BoldValidationClient{
		producers:   make(map[common.Hash]*pubsub.Producer[*server_api.GetLeavesWithStepSizeInput, []common.Hash]),
		redisClient: redisClient,
	}, nil
}

func (c *BoldValidationClient) Initialize(moduleRoots []common.Hash) error {
	for _, mr := range moduleRoots {
		if _, exists := c.producers[mr]; exists {
			log.Warn("Producer already existsw for module root", "hash", mr)
			continue
		}
		p, err := pubsub.NewProducer[*server_api.GetLeavesWithStepSizeInput, []common.Hash](
			c.redisClient, server_api.RedisStreamForRoot(mr), &c.producerConfig)
		if err != nil {
			log.Warn("failed init redis for %v: %w", mr, err)
			continue
		}
		p.Start(c.GetContext())
		c.producers[mr] = p
		c.moduleRoots = append(c.moduleRoots, mr)
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
