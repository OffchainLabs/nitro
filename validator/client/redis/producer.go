package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"
)

type ValidationClientConfig struct {
	Name           string                `koanf:"name"`
	StreamPrefix   string                `koanf:"stream-prefix"`
	Room           int32                 `koanf:"room"`
	RedisURL       string                `koanf:"redis-url"`
	StylusArchs    []string              `koanf:"stylus-archs"`
	ProducerConfig pubsub.ProducerConfig `koanf:"producer-config"`
	CreateStreams  bool                  `koanf:"create-streams"`
}

func (c ValidationClientConfig) Enabled() bool {
	return c.RedisURL != ""
}

func (c ValidationClientConfig) Validate() error {
	for _, arch := range c.StylusArchs {
		if !rawdb.IsSupportedWasmTarget(rawdb.WasmTarget(arch)) {
			return fmt.Errorf("Invalid stylus arch: %v", arch)
		}
	}
	return nil
}

var DefaultValidationClientConfig = ValidationClientConfig{
	Name:           "redis validation client",
	Room:           2,
	RedisURL:       "",
	StylusArchs:    []string{string(rawdb.TargetWavm)},
	ProducerConfig: pubsub.DefaultProducerConfig,
	CreateStreams:  true,
}

var TestValidationClientConfig = ValidationClientConfig{
	Name:           "test redis validation client",
	Room:           2,
	RedisURL:       "",
	StreamPrefix:   "test-",
	StylusArchs:    []string{string(rawdb.TargetWavm)},
	ProducerConfig: pubsub.TestProducerConfig,
	CreateStreams:  false,
}

func ValidationClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".name", DefaultValidationClientConfig.Name, "validation client name")
	f.Int32(prefix+".room", DefaultValidationClientConfig.Room, "validation client room")
	f.String(prefix+".redis-url", DefaultValidationClientConfig.RedisURL, "redis url")
	f.String(prefix+".stream-prefix", DefaultValidationClientConfig.StreamPrefix, "prefix for stream name")
	f.StringSlice(prefix+".stylus-archs", DefaultValidationClientConfig.StylusArchs, "archs required for stylus workers")
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
			log.Warn("Producer already exists for module root", "hash", mr)
			continue
		}
		p, err := pubsub.NewProducer[*validator.ValidationInput, validator.GoGlobalState](
			c.redisClient, server_api.RedisStreamForRoot(c.config.StreamPrefix, mr), &c.config.ProducerConfig)
		if err != nil {
			log.Warn("failed init redis for %v: %w", mr, err)
			continue
		}
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

func (c *ValidationClient) StylusArchs() []rawdb.WasmTarget {
	stylusArchs := make([]rawdb.WasmTarget, 0, len(c.config.StylusArchs))
	for _, arch := range c.config.StylusArchs {
		stylusArchs = append(stylusArchs, rawdb.WasmTarget(arch))
	}
	return stylusArchs
}

func (c *ValidationClient) Room() int {
	return int(c.room.Load())
}

var _ validator.BOLDExecutionSpawner = (*BOLDRedisExecutionClient)(nil)

type BOLDRedisExecutionClient struct {
	stopwaiter.StopWaiter
	redisValidationClient *ValidationClient
	// producers stores moduleRoot to producer mapping.
	producers map[common.Hash]*pubsub.Producer[*server_api.BoldValidationInput, []byte]
}

func NewBOLDRedisExecutionClient(redisValClient *ValidationClient) *BOLDRedisExecutionClient {
	return &BOLDRedisExecutionClient{
		redisValidationClient: redisValClient,
		producers:             make(map[common.Hash]*pubsub.Producer[*server_api.BoldValidationInput, []byte]),
	}
}

func (br *BOLDRedisExecutionClient) Initialize(ctx context.Context, moduleRoots []common.Hash) error {
	if br.redisValidationClient.config.RedisURL == "" {
		return fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(br.redisValidationClient.config.RedisURL)
	if err != nil {
		return err
	}
	for _, mr := range moduleRoots {
		if br.redisValidationClient.config.CreateStreams {
			if err := pubsub.CreateStream(ctx, server_api.RedisBoldStreamForRoot(br.redisValidationClient.config.StreamPrefix, mr), redisClient); err != nil {
				return fmt.Errorf("creating redis stream: %w", err)
			}
		}
		if _, exists := br.producers[mr]; exists {
			log.Warn("Producer already exists for module root", "hash", mr)
			continue
		}
		p, err := pubsub.NewProducer[*server_api.BoldValidationInput, []byte](
			redisClient, server_api.RedisBoldStreamForRoot(br.redisValidationClient.config.StreamPrefix, mr), &br.redisValidationClient.config.ProducerConfig)
		if err != nil {
			log.Warn("failed init redis for %v: %w", mr, err)
			continue
		}
		br.producers[mr] = p
	}
	return nil
}

func (br *BOLDRedisExecutionClient) produce(req *server_api.BoldValidationInput) containers.PromiseInterface[[]byte] {
	producer, found := br.producers[req.ModuleRoot]
	if !found {
		return containers.NewReadyPromise([]byte{}, fmt.Errorf("no validation is configured for wasm root %v", req.ModuleRoot))
	}
	promise, err := producer.Produce(br.GetContext(), req)
	if err != nil {
		return containers.NewReadyPromise([]byte{}, fmt.Errorf("error producing input: %w", err))
	}
	return promise
}

func (br *BOLDRedisExecutionClient) Start(ctx_in context.Context) error {
	if err := br.Initialize(ctx_in, br.redisValidationClient.moduleRoots); err != nil {
		return err
	}
	for _, p := range br.producers {
		p.Start(ctx_in)
	}
	br.StopWaiter.Start(ctx_in, br)
	return nil
}

func (br *BOLDRedisExecutionClient) Stop() {
	for _, p := range br.producers {
		p.StopAndWait()
	}
	br.StopWaiter.StopAndWait()
}

func (br *BOLDRedisExecutionClient) WasmModuleRoots() ([]common.Hash, error) {
	return br.redisValidationClient.WasmModuleRoots()
}

func (br *BOLDRedisExecutionClient) GetMachineHashesWithStepSize(ctx context.Context, wasmModuleRoot common.Hash, input *validator.ValidationInput, machineStartIndex, stepSize, maxIterations uint64) ([]common.Hash, error) {
	res, err := br.produce(&server_api.BoldValidationInput{
		ModuleRoot:        wasmModuleRoot,
		MachineStartIndex: machineStartIndex,
		StepSize:          stepSize,
		NumDesiredLeaves:  maxIterations,
		ValidationInput:   input,
	}).Await(ctx)
	if err != nil {
		return nil, err
	}
	var resJson []common.Hash
	err = json.Unmarshal(res, &resJson)
	if err != nil {
		return nil, err
	}
	return resJson, nil
}

func (br *BOLDRedisExecutionClient) GetProofAt(ctx context.Context, wasmModuleRoot common.Hash, input *validator.ValidationInput, position uint64) ([]byte, error) {
	res, err := br.produce(&server_api.BoldValidationInput{
		ModuleRoot:        wasmModuleRoot,
		MachineStartIndex: position,
		ValidationInput:   input,
	}).Await(ctx)
	if err != nil {
		return nil, err
	}
	var resJson []byte
	err = json.Unmarshal(res, &resJson)
	if err != nil {
		return nil, err
	}
	return resJson, nil
}
