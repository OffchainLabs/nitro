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
	StreamPrefix   string                `koanf:"stream-prefix"`
	Room           int32                 `koanf:"room"`
	RedisURL       string                `koanf:"redis-url"`
	StylusArch     string                `koanf:"stylus-arch"`
	ProducerConfig pubsub.ProducerConfig `koanf:"producer-config"`
	CreateStreams  bool                  `koanf:"create-streams"`
}

func (c ValidationClientConfig) Enabled() bool {
	return c.RedisURL != ""
}

var DefaultValidationClientConfig = ValidationClientConfig{
	Name:           "redis validation client",
	Room:           2,
	RedisURL:       "",
	StylusArch:     "wavm",
	ProducerConfig: pubsub.DefaultProducerConfig,
	CreateStreams:  true,
}

var TestValidationClientConfig = ValidationClientConfig{
	Name:           "test redis validation client",
	Room:           2,
	RedisURL:       "",
	StreamPrefix:   "test-",
	StylusArch:     "wavm",
	ProducerConfig: pubsub.TestProducerConfig,
	CreateStreams:  false,
}

func ValidationClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".name", DefaultValidationClientConfig.Name, "validation client name")
	f.Int32(prefix+".room", DefaultValidationClientConfig.Room, "validation client room")
	f.String(prefix+".redis-url", DefaultValidationClientConfig.RedisURL, "redis url")
	f.String(prefix+".stream-prefix", DefaultValidationClientConfig.StreamPrefix, "prefix for stream name")
	f.String(prefix+".stylus-arch", DefaultValidationClientConfig.StylusArch, "arch for stylus workers")
	pubsub.ProducerAddConfigAddOptions(prefix+".producer-config", f)
	f.Bool(prefix+".create-streams", DefaultValidationClientConfig.CreateStreams, "create redis streams if it does not exist")
}

// ValidationClient implements validation client through redis streams.
type ValidationClient struct {
	stopwaiter.StopWaiter
	config *ValidationClientConfig
	room   atomic.Int32
	// producers stores moduleRoot to producer mapping.
	producers   map[common.Hash]*pubsub.Producer[*validator.ValidationInput, validator.GoGlobalState]
	redisClient redis.UniversalClient
	moduleRoots []common.Hash
}

func NewValidationClient(cfg *ValidationClientConfig) (*ValidationClient, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	validationClient := &ValidationClient{
		config:      cfg,
		producers:   make(map[common.Hash]*pubsub.Producer[*validator.ValidationInput, validator.GoGlobalState]),
		redisClient: redisClient,
	}
	validationClient.room.Store(cfg.Room)
	return validationClient, nil
}

func (c *ValidationClient) Initialize(ctx context.Context, moduleRoots []common.Hash) error {
	for _, mr := range moduleRoots {
		if c.config.CreateStreams {
			if err := pubsub.CreateStream(ctx, server_api.RedisStreamForRoot(c.config.StreamPrefix, mr), c.redisClient); err != nil {
				return fmt.Errorf("creating redis stream: %w", err)
			}
		}
		if _, exists := c.producers[mr]; exists {
			log.Warn("Producer already existsw for module root", "hash", mr)
			continue
		}
		p, err := pubsub.NewProducer[*validator.ValidationInput, validator.GoGlobalState](
			c.redisClient, server_api.RedisStreamForRoot(c.config.StreamPrefix, mr), &c.config.ProducerConfig)
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

func (c *ValidationClient) WasmModuleRoots() ([]common.Hash, error) {
	return c.moduleRoots, nil
}

func (c *ValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	c.room.Add(-1)
	defer c.room.Add(1)
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
	return c.config.Name
}

func (c *ValidationClient) StylusArch() string {
	return c.config.StylusArch
}

func (c *ValidationClient) Room() int {
	return int(c.room.Load())
}
