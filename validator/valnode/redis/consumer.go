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
	consumers     map[common.Hash]*pubsub.Consumer[*validator.ValidationInput, validator.GoGlobalState]
	streamTimeout time.Duration
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
		c, err := pubsub.NewConsumer[*validator.ValidationInput, validator.GoGlobalState](redisClient, server_api.RedisStreamForRoot(mr), &cfg.ConsumerConfig)
		if err != nil {
			return nil, fmt.Errorf("creating consumer for validation: %w", err)
		}
		consumers[mr] = c
	}
	return &ValidationServer{
		consumers:     consumers,
		spawner:       spawner,
		streamTimeout: cfg.StreamTimeout,
	}, nil
}

func (s *ValidationServer) Start(ctx_in context.Context) {
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
					log.Info("Context done while checking redis stream existance", "error", ctx.Err().Error())
					return
				case <-time.After(time.Millisecond * 100):
				}
			}
		})
		s.StopWaiter.LaunchThread(func(ctx context.Context) {
			select {
			case <-ctx.Done():
				log.Info("Context done while waiting a redis stream to be ready", "error", ctx.Err().Error())
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
				log.Info("Context done while waiting redis streams to be ready, failed to start")
				return
			}
		}
	})
}

type ValidationServerConfig struct {
	RedisURL       string                `koanf:"redis-url"`
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Supported wasm module roots.
	ModuleRoots []string `koanf:"module-roots"`
	// Timeout on polling for existence of each redis stream.
	StreamTimeout time.Duration `koanf:"stream-timeout"`
}

var DefaultValidationServerConfig = ValidationServerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.DefaultConsumerConfig,
	ModuleRoots:    []string{},
	StreamTimeout:  10 * time.Minute,
}

var TestValidationServerConfig = ValidationServerConfig{
	RedisURL:       "",
	ConsumerConfig: pubsub.TestConsumerConfig,
	ModuleRoots:    []string{},
	StreamTimeout:  time.Minute,
}

func ValidationServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
	f.Duration(prefix+".stream-timeout", DefaultValidationServerConfig.StreamTimeout, "Timeout on polling for existence of redis streams")
}

func (cfg *ValidationServerConfig) Enabled() bool {
	return cfg.RedisURL != ""
}
