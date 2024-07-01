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
	// Timeout on polling for existence of each redis stream.
	StreamTimeout time.Duration `koanf:"stream-timeout"`
}

var DefaultExecutionSpawnerConfig = ExecutionSpawnerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.DefaultConsumerConfig,
	ModuleRoots:    []string{},
	StreamTimeout:  10 * time.Minute,
}

var TestExecutionSpawnerConfig = ExecutionSpawnerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.TestConsumerConfig,
	ModuleRoots:    []string{},
	StreamTimeout:  time.Minute,
}

func ExecutionSpawnerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
	f.Duration(prefix+".stream-timeout", DefaultExecutionSpawnerConfig.StreamTimeout, "Timeout on polling for existence of redis streams")
}

func (cfg *ExecutionSpawnerConfig) Enabled() bool {
	return cfg.RedisURL != ""
}

type ExecutionSpawner struct {
	stopwaiter.StopWaiter
	spawner validator.ExecutionSpawner

	// consumers stores moduleRoot to consumer mapping.
	consumers     map[common.Hash]*pubsub.Consumer[*server_api.GetLeavesWithStepSizeInput, []common.Hash]
	streamTimeout time.Duration
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
		consumers:     consumers,
		spawner:       spawner,
		streamTimeout: cfg.StreamTimeout,
	}, nil
}

func (s *ExecutionSpawner) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	// Channel that all consumers use to indicate their readiness.
	readyStreams := make(chan struct{}, len(s.consumers))
	for moduleRoot, c := range s.consumers {
		c := c
		moduleRoot := moduleRoot
		c.Start(ctx_in)
		// Channel for single consumer, once readiness is indicated in this,
		// consumer will start consuming iteratively.
		ready := make(chan struct{}, 1)
		s.StopWaiter.LaunchThread(func(ctx context.Context) {
			for {
				if pubsub.StreamExists(ctx, c.StreamName(), c.RedisClient()) {
					ready <- struct{}{}
					readyStreams <- struct{}{}
					return
				}
				select {
				case <-ctx.Done():
					log.Info("Context done", "error", ctx.Err().Error())
					return
				case <-time.After(time.Millisecond * 100):
				}
			}
		})
		s.StopWaiter.LaunchThread(func(ctx context.Context) {
			select {
			case <-ctx.Done():
				log.Info("Context done", "error", ctx.Err().Error())
				return
			case <-ready: // Wait until the stream exists and start consuming iteratively.
			}
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
				run, err := s.spawner.CreateExecutionRun(moduleRoot,
					req.Value.ValidationInput).Await(ctx)
				if err != nil {
					log.Error("Creationg BOLD execution", "error", err)
					return 0
				}
				hashes, err := run.GetMachineHashesWithStepSize(
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
		})
	}
	s.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-readyStreams:
				log.Trace("At least one stream is ready")
				return // Don't block Start if at least one of the stream is ready.
			case <-time.After(s.streamTimeout):
				log.Error("Waiting for redis streams timed out")
			case <-ctx.Done():
				log.Info(("Context expired, failed to start"))
				return
			}
		}
	})
}
