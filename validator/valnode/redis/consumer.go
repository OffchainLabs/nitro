package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/spf13/pflag"
)

// ValidationServer implements consumer for the requests originated from
// RedisValidationClient producers.
type ValidationServer struct {
	stopwaiter.StopWaiter
	spawner validator.ValidationSpawner

	// consumers stores moduleRoot to consumer mapping.
	consumers map[common.Hash]*pubsub.Consumer[*validator.ValidationInput, validator.GoGlobalState]
}

func NewValidationServer(cfg *ValidationServerConfig, spawner validator.ValidationSpawner) (*ValidationServer, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	consumers := make(map[common.Hash]*pubsub.Consumer[*validator.ValidationInput, validator.GoGlobalState])
	for _, hash := range cfg.ModuleRoots {
		mr := common.HexToHash(hash)
		if err := pubsub.CreateStream(context.TODO(), server_api.RedisStreamForRoot(mr), redisClient); err != nil {
			return nil, fmt.Errorf("creating redis stream: %w", err)
		}
		c, err := pubsub.NewConsumer[*validator.ValidationInput, validator.GoGlobalState](redisClient, server_api.RedisStreamForRoot(mr), &cfg.ConsumerConfig)
		if err != nil {
			return nil, fmt.Errorf("creating consumer for validation: %w", err)
		}
		consumers[mr] = c
	}
	return &ValidationServer{
		consumers: consumers,
		spawner:   spawner,
	}, nil
}

func (s *ValidationServer) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	for moduleRoot, c := range s.consumers {
		c := c
		c.Start(ctx_in)
		s.StopWaiter.CallIteratively(func(ctx context.Context) time.Duration {
			req, err := c.Consume(ctx)
			if err != nil {
				log.Error("Consuming request", "error", err)
				return 0
			}
			if req == nil {
				// There's nothing in the queue.
				return time.Second
			}
			valRun := s.spawner.Launch(req.Value, moduleRoot)
			res, err := valRun.Await(ctx)
			if err != nil {
				log.Error("Error validating", "request value", req.Value, "error", err)
				return 0
			}
			if err := c.SetResult(ctx, req.ID, res); err != nil {
				log.Error("Error setting result for request", "id", req.ID, "result", res, "error", err)
				return 0
			}
			return time.Second
		})
	}
}

type ValidationServerConfig struct {
	RedisURL       string                `koanf:"redis-url"`
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Supported wasm module roots.
	ModuleRoots []string `koanf:"module-roots"`
}

var DefaultValidationServerConfig = ValidationServerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.DefaultConsumerConfig,
	ModuleRoots:    []string{},
}

var TestValidationServerConfig = ValidationServerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.TestConsumerConfig,
	ModuleRoots:    []string{},
}

func ValidationServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
}

func (cfg *ValidationServerConfig) Enabled() bool {
	return cfg.RedisURL != ""
}
