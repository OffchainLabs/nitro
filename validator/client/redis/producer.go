package redis

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/spf13/pflag"
)

type ValidationClientConfig struct {
	Name           string                `koanf:"name"`
	Room           int32                 `koanf:"room"`
	RedisURL       string                `koanf:"redis-url"`
	ProducerConfig pubsub.ProducerConfig `koanf:"producer-config"`
}

func (c ValidationClientConfig) Enabled() bool {
	return c.RedisURL != ""
}

var DefaultValidationClientConfig = ValidationClientConfig{
	Name:           "redis validation client",
	Room:           2,
	RedisURL:       "",
	ProducerConfig: pubsub.DefaultProducerConfig,
}

var TestValidationClientConfig = ValidationClientConfig{
	Name:           "test redis validation client",
	Room:           2,
	RedisURL:       "",
	ProducerConfig: pubsub.TestProducerConfig,
}

func ValidationClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".name", DefaultValidationClientConfig.Name, "validation client name")
	f.Int32(prefix+".room", DefaultValidationClientConfig.Room, "validation client room")
	pubsub.ProducerAddConfigAddOptions(prefix+".producer-config", f)
}

// ValidationClient implements validation client through redis streams.
type ValidationClient struct {
	stopwaiter.StopWaiter
	name string
	room int32
	// producers stores moduleRoot to producer mapping.
	producers      map[common.Hash]*pubsub.Producer[*validator.ValidationInput, validator.GoGlobalState]
	producerConfig pubsub.ProducerConfig
	redisClient    redis.UniversalClient
	moduleRoots    []common.Hash
}

func NewValidationClient(cfg *ValidationClientConfig) (*ValidationClient, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	return &ValidationClient{
		name:           cfg.Name,
		room:           cfg.Room,
		producers:      make(map[common.Hash]*pubsub.Producer[*validator.ValidationInput, validator.GoGlobalState]),
		producerConfig: cfg.ProducerConfig,
		redisClient:    redisClient,
	}, nil
}

func (c *ValidationClient) Initialize(moduleRoots []common.Hash) error {
	for _, mr := range moduleRoots {
		if _, exists := c.producers[mr]; exists {
			log.Warn("Producer already existsw for module root", "hash", mr)
			continue
		}
		p, err := pubsub.NewProducer[*validator.ValidationInput, validator.GoGlobalState](
			c.redisClient, server_api.RedisStreamForRoot(mr), &c.producerConfig)
		if err != nil {
			return fmt.Errorf("creating producer for validation: %w", err)
		}
		p.Start(c.GetContext())
		c.producers[mr] = p
		c.moduleRoots = append(c.moduleRoots, mr)
	}
	return nil
}

func (c *ValidationClient) WasmModuleRoots() ([]common.Hash, error) {
	return c.moduleRoots, nil
}

func (c *ValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	atomic.AddInt32(&c.room, -1)
	defer atomic.AddInt32(&c.room, 1)
	producer, found := c.producers[moduleRoot]
	if !found {
		errPromise := containers.NewReadyPromise(validator.GoGlobalState{}, fmt.Errorf("no validation is configured for wasm root %v", moduleRoot))
		return server_common.NewValRun(errPromise, moduleRoot)
	}
	promise, err := producer.Produce(c.GetContext(), entry)
	if err != nil {
		errPromise := containers.NewReadyPromise(validator.GoGlobalState{}, fmt.Errorf("error producing input: %w", err))
		return server_common.NewValRun(errPromise, moduleRoot)
	}
	return server_common.NewValRun(promise, moduleRoot)
}

func (c *ValidationClient) Start(ctx_in context.Context) error {
	for _, p := range c.producers {
		p.Start(ctx_in)
	}
	c.StopWaiter.Start(ctx_in, c)
	return nil
}

func (c *ValidationClient) Stop() {
	for _, p := range c.producers {
		p.StopAndWait()
	}
	c.StopWaiter.StopAndWait()
}

func (c *ValidationClient) Name() string {
	if c.Started() {
		return c.name
	}
	return "(not started)"
}

func (c *ValidationClient) Room() int {
	return int(c.room)
}
