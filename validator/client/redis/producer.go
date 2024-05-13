package redis

import (
	"context"
	"fmt"
	"os"
	"strings"
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

type ValidationClientInitConfig struct {
	// If set, will terminate the binary once ValidationClient is initialized.
	ThenQuit bool `koanf:"then-quit"`
	// If set, this will be used instead of ones read from locator.
	ModuleRoots []string `koanf:"module-roots"`
}

var DefaultValidationClientInitConfig = &ValidationClientInitConfig{
	ThenQuit:    false,
	ModuleRoots: []string{},
}

var TestValidationClientInitConfig = &ValidationClientInitConfig{
	ThenQuit:    false,
	ModuleRoots: []string{},
}

func ValidationClientInitConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".then-quit", DefaultValidationClientInitConfig.ThenQuit, "quit after init is done")
	f.StringSlice(prefix+".module-roots", DefaultValidationClientInitConfig.ModuleRoots, "list of WASM module roots")
}

type ValidationClientConfig struct {
	Name           string                     `koanf:"name"`
	Room           int32                      `koanf:"room"`
	RedisURL       string                     `koanf:"redis-url"`
	ProducerConfig pubsub.ProducerConfig      `koanf:"producer-config"`
	InitConfig     ValidationClientInitConfig `koanf:"init-config"`
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
	thenQuit       bool // If set, halts binary after initialize function.
}

func NewValidationClient(cfg *ValidationClientConfig) (*ValidationClient, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	ret := &ValidationClient{
		name:           cfg.Name,
		room:           cfg.Room,
		producers:      make(map[common.Hash]*pubsub.Producer[*validator.ValidationInput, validator.GoGlobalState]),
		producerConfig: cfg.ProducerConfig,
		redisClient:    redisClient,
	}
	// Parse module roots from init config, if there are any.
	for _, mrHex := range cfg.InitConfig.ModuleRoots {
		if mr := common.HexToHash(strings.TrimSpace(string(mrHex))); mr != (common.Hash{}) {
			ret.moduleRoots = append(ret.moduleRoots, mr)
		}
	}
	return ret, nil
}

// Initialize creates redis stream for each  WASM module root, if exists skips,
// and starts producers if `then-quit` flag is not set.
func (c *ValidationClient) Initialize(ctx context.Context, moduleRoots []common.Hash) error {
	if c.moduleRoots != nil {
		moduleRoots = c.moduleRoots
	}
	for _, mr := range moduleRoots {
		if _, exists := c.producers[mr]; exists {
			log.Warn("Producer already existsw for module root", "hash", mr)
			continue
		}
		if err := pubsub.CreateStream(ctx, server_api.RedisStreamForRoot(mr), c.redisClient); err != nil {
			return fmt.Errorf("creating redis stream: %w", err)
		}
		p, err := pubsub.NewProducer[*validator.ValidationInput, validator.GoGlobalState](
			c.redisClient, server_api.RedisStreamForRoot(mr), &c.producerConfig)
		if err != nil {
			log.Warn("failed init redis for %v: %w", mr, err)
			continue
		}
		p.Start(c.GetContext())
		c.producers[mr] = p
		c.moduleRoots = append(c.moduleRoots, mr)
	}

	if c.thenQuit {
		log.Info("Validation client initialized, then-quit flag is set, existing.")
		os.Exit(0)
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
