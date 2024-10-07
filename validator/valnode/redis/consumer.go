package redis

import (
	"context"
	"fmt"
	"runtime"
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

	config *ValidationServerConfig
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
		c, err := pubsub.NewConsumer[*validator.ValidationInput, validator.GoGlobalState](redisClient, server_api.RedisStreamForRoot(cfg.StreamPrefix, mr), &cfg.ConsumerConfig)
		if err != nil {
			return nil, fmt.Errorf("creating consumer for validation: %w", err)
		}
		consumers[mr] = c
	}
	return &ValidationServer{
		consumers: consumers,
		spawner:   spawner,
		config:    cfg,
	}, nil
}

func (s *ValidationServer) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	// Channel that all consumers use to indicate their readiness.
	readyStreams := make(chan struct{}, len(s.consumers))
	type workUnit struct {
		req        *pubsub.Message[*validator.ValidationInput]
		moduleRoot common.Hash
	}
	workers := s.config.Workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}
	workQueue := make(chan workUnit, workers)
	tokensCount := workers
	if s.config.BufferReads {
		tokensCount += workers
	}
	requestTokenQueue := make(chan struct{}, tokensCount)
	for i := 0; i < tokensCount; i++ {
		requestTokenQueue <- struct{}{}
	}
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
				log.Debug("waiting for request token", "cid", c.Id())
				select {
				case <-ctx.Done():
					return 0
				case <-requestTokenQueue:
				}
				log.Debug("got request token", "cid", c.Id())
				req, err := c.Consume(ctx)
				if err != nil {
					log.Error("Consuming request", "error", err)
					requestTokenQueue <- struct{}{}
					return 0
				}
				if req == nil {
					log.Debug("consumed nil", "cid", c.Id())
					// There's nothing in the queue
					requestTokenQueue <- struct{}{}
					return time.Second
				}
				log.Debug("forwarding work", "cid", c.Id(), "workid", req.ID)
				select {
				case <-ctx.Done():
				case workQueue <- workUnit{req, moduleRoot}:
				}
				return 0
			})
		})
	}
	s.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-readyStreams:
				log.Debug("At least one stream is ready")
				return // Don't block Start if at least one of the stream is ready.
			case <-time.After(s.config.StreamTimeout):
				log.Error("Waiting for redis streams timed out")
			case <-ctx.Done():
				log.Info("Context done while waiting redis streams to be ready, failed to start")
				return
			}
		}
	})
	for i := 0; i < workers; i++ {
		i := i
		s.StopWaiter.LaunchThread(func(ctx context.Context) {
			for {
				log.Debug("waiting for work", "thread", i)
				var work workUnit
				select {
				case <-ctx.Done():
					return
				case work = <-workQueue:
				}
				log.Debug("got work", "thread", i, "workid", work.req.ID)
				valRun := s.spawner.Launch(work.req.Value, work.moduleRoot)
				res, err := valRun.Await(ctx)
				if err != nil {
					log.Error("Error validating", "request value", work.req.Value, "error", err)
				} else {
					log.Debug("done work", "thread", i, "workid", work.req.ID)
					if err := s.consumers[work.moduleRoot].SetResult(ctx, work.req.ID, res); err != nil {
						log.Error("Error setting result for request", "id", work.req.ID, "result", res, "error", err)
					}
					log.Debug("set result", "thread", i, "workid", work.req.ID)
				}
				select {
				case <-ctx.Done():
					return
				case requestTokenQueue <- struct{}{}:
				}
			}
		})
	}
}

type ValidationServerConfig struct {
	RedisURL       string                `koanf:"redis-url"`
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Supported wasm module roots.
	ModuleRoots []string `koanf:"module-roots"`
	// Timeout on polling for existence of each redis stream.
	StreamTimeout time.Duration `koanf:"stream-timeout"`
	StreamPrefix  string        `koanf:"stream-prefix"`
	Workers       int           `koanf:"workers"`
	BufferReads   bool          `koanf:"buffer-reads"`
}

var DefaultValidationServerConfig = ValidationServerConfig{
	RedisURL:       "",
	StreamPrefix:   "",
	ConsumerConfig: pubsub.DefaultConsumerConfig,
	ModuleRoots:    []string{},
	StreamTimeout:  10 * time.Minute,
	Workers:        0,
	BufferReads:    true,
}

var TestValidationServerConfig = ValidationServerConfig{
	RedisURL:       "",
	StreamPrefix:   "test-",
	ConsumerConfig: pubsub.TestConsumerConfig,
	ModuleRoots:    []string{},
	StreamTimeout:  time.Minute,
	Workers:        1,
	BufferReads:    true,
}

func ValidationServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.StringSlice(prefix+".module-roots", nil, "Supported module root hashes")
	f.String(prefix+".redis-url", DefaultValidationServerConfig.RedisURL, "url of redis server")
	f.String(prefix+".stream-prefix", DefaultValidationServerConfig.StreamPrefix, "prefix for stream name")
	f.Duration(prefix+".stream-timeout", DefaultValidationServerConfig.StreamTimeout, "Timeout on polling for existence of redis streams")
	f.Int(prefix+".workers", DefaultValidationServerConfig.Workers, "number of validation threads (0 to use number of CPUs)")
	f.Bool(prefix+".buffer-reads", DefaultValidationServerConfig.BufferReads, "buffer reads (read next while working)")
}

func (cfg *ValidationServerConfig) Enabled() bool {
	return cfg.RedisURL != ""
}
