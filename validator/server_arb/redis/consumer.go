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

type ExecutionSpawnerConfig struct {
	RedisURL       string                `koanf:"redis-url"`
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Supported wasm module roots.
	ModuleRoots []string `koanf:"module-roots"`
}

var DefaultExecutionSpawnerConfig = ExecutionSpawnerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.DefaultConsumerConfig,
	ModuleRoots:    []string{},
}

var TestExecutionSpawnerConfig = ExecutionSpawnerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.TestConsumerConfig,
	ModuleRoots:    []string{},
}

func ExecutionSpawnerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
}

func (cfg *ExecutionSpawnerConfig) Enabled() bool {
	return cfg.RedisURL != ""
}

type ExecutionSpawner struct {
	stopwaiter.StopWaiter
	spawner validator.ExecutionSpawner

	// consumers stores moduleRoot to consumer mapping.
	consumers map[common.Hash]*pubsub.Consumer[*server_api.GetLeavesWithStepSizeInput, []common.Hash]
}

func NewExecutionSpawner(cfg *ExecutionSpawnerConfig, spawner validator.ExecutionSpawner) (*ExecutionSpawner, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	consumers := make(map[common.Hash]*pubsub.Consumer[*server_api.GetLeavesWithStepSizeInput, []common.Hash])
	for _, hash := range cfg.ModuleRoots {
		mr := common.HexToHash(hash)
		c, err := pubsub.NewConsumer[*server_api.GetLeavesWithStepSizeInput, []common.Hash](redisClient, server_api.RedisBoldStreamForRoot(mr), &cfg.ConsumerConfig)
		if err != nil {
			return nil, fmt.Errorf("creating consumer for validation: %w", err)
		}
		consumers[mr] = c
	}
	return &ExecutionSpawner{
		consumers: consumers,
		spawner:   spawner,
	}, nil
}

func (s *ExecutionSpawner) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.spawner.Start(ctx_in)
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
			run, err := s.spawner.CreateBoldExecutionRun(moduleRoot, req.Value.StepSize,
				req.Value.ValidationInput).Await(ctx)
			if err != nil {
				log.Error("Creationg BOLD execution", "error", err)
				return 0
			}
			hashes, err := run.GetLeavesWithStepSize(
				req.Value.FromBatch,
				req.Value.MachineStartIndex,
				req.Value.StepSize,
				req.Value.NumDesiredLeaves).Await(ctx)
			if err != nil {
				log.Error("Getting leave hashes", "error", err)
				return 0
			}
			if err := c.SetResult(ctx, req.ID, hashes); err != nil {
				log.Error("Error setting result for request", "id", req.ID, "result", hashes, "error", err)
				return 0
			}
			return time.Second
		})
	}
}
