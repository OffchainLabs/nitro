package server_api

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/spf13/pflag"
)

type RedisValidationClientConfig struct {
	Name           string                `koanf:"name"`
	Room           int32                 `koanf:"room"`
	ProducerConfig pubsub.ProducerConfig `koanf:"producer-config"`
	// Supported wasm module roots.
	ModuleRoots []string `koanf:"module-roots"`
}

var DefaultRedisValidationClientConfig = RedisValidationClientConfig{
	Name:           "redis validation client",
	Room:           2,
	ProducerConfig: pubsub.DefaultProducerConfig,
}

var TestRedisValidationClientConfig = RedisValidationClientConfig{
	Name:           "test redis validation client",
	Room:           2,
	ProducerConfig: pubsub.TestProducerConfig,
}

func RedisValidationClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".name", DefaultRedisValidationClientConfig.Name, "validation client name")
	f.Uint64(prefix+".room", uint64(DefaultRedisValidationClientConfig.Room), "validation client room")
	pubsub.ProducerAddConfigAddOptions(prefix+".producer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
}

// RedisValidationClient implements validation client through redis streams.
type RedisValidationClient struct {
	stopwaiter.StopWaiter
	name string
	room int32
	// producers stores moduleRoot to producer mapping.
	producers map[common.Hash]*pubsub.Producer[*validator.ValidationInput, validator.GoGlobalState]
}

func redisGroupForRoot(moduleRoot common.Hash) string {
	return fmt.Sprintf("group:%s", moduleRoot.Hex())
}

func redisStreamForRoot(moduleRoot common.Hash) string {
	return fmt.Sprintf("stream:%s", moduleRoot.Hex())
}

func NewRedisValidationClient(cfg *RedisValidationClientConfig) (*RedisValidationClient, error) {
	res := &RedisValidationClient{
		name:      cfg.Name,
		room:      cfg.Room,
		producers: make(map[common.Hash]*pubsub.Producer[*validator.ValidationInput, validator.GoGlobalState]),
	}
	for _, hash := range cfg.ModuleRoots {
		mr := common.HexToHash(hash)
		c := cfg.ProducerConfig.Clone()
		c.RedisStream, c.RedisGroup = redisGroupForRoot(mr), redisStreamForRoot(mr)
		p, err := pubsub.NewProducer[*validator.ValidationInput, validator.GoGlobalState](&c)
		if err != nil {
			return nil, fmt.Errorf("creating producer for validation: %w", err)
		}
		res.producers[mr] = p
	}
	return res, nil
}

func (c *RedisValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
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

func (c *RedisValidationClient) Start(ctx_in context.Context) error {
	for _, p := range c.producers {
		p.Start(ctx_in)
	}
	c.StopWaiter.Start(ctx_in, c)
	return nil
}

func (c *RedisValidationClient) Stop() {
	for _, p := range c.producers {
		p.StopAndWait()
	}
	c.StopWaiter.StopAndWait()
}

func (c *RedisValidationClient) Name() string {
	if c.Started() {
		return c.name
	}
	return "(not started)"
}

func (c *RedisValidationClient) Room() int {
	return int(c.room)
}
