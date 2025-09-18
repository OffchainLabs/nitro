package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
)

// ValidationServer implements consumer for the requests originated from
// RedisValidationClient producers.
type ValidationServer struct {
	stopwaiter.StopWaiter
	spawner validator.ExecutionSpawner

	// consumers stores moduleRoot to consumer mapping.
	consumers map[common.Hash]*pubsub.Consumer[*validator.ValidationInput, validator.GoGlobalState]

	config *ValidationServerConfig
}

func NewValidationServer(cfg *ValidationServerConfig, spawner validator.ExecutionSpawner) (*ValidationServer, error) {
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
	s.StartBoldSpawner(ctx_in)
	// Channel that all consumers use to indicate their readiness.
	readyStreams := make(chan struct{}, len(s.consumers))
	type workUnit struct {
		req        *pubsub.Message[*validator.ValidationInput]
		moduleRoot common.Hash
	}
	workers := s.config.Workers
	if workers == 0 {
		workers = util.GoMaxProcs()
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
					log.Info("Context done while checking redis stream existence", "error", ctx.Err().Error())
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
					err := s.consumers[work.moduleRoot].SetError(ctx, work.req.ID, err.Error())
					work.req.Ack()
					if err != nil {
						log.Error("Error setting error for request", "id", work.req.ID, "error", err)
					}
				} else {
					log.Debug("done work", "thread", i, "workid", work.req.ID)
					err := s.consumers[work.moduleRoot].SetResult(ctx, work.req.ID, res)
					// Even in error we close ackNotifier as there's no retry mechanism here and closing it will allow other consumers to autoclaim
					work.req.Ack()
					if err != nil {
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

func (s *ValidationServer) StartBoldSpawner(ctx context.Context) {
	boldSpawner, err := NewExecutionSpawner(s.config, s.spawner)
	if err != nil {
		log.Error("creating redis execution spawner", "error", err)
	}
	boldSpawner.Start(ctx)
}

type ExecutionSpawner struct {
	stopwaiter.StopWaiter
	spawner validator.ExecutionSpawner

	// consumers stores moduleRoot to consumer mapping.
	consumers map[common.Hash]*pubsub.Consumer[*server_api.BoldValidationInput, []byte]
	config    *ValidationServerConfig
}

func NewExecutionSpawner(cfg *ValidationServerConfig, spawner validator.ExecutionSpawner) (*ExecutionSpawner, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	consumers := make(map[common.Hash]*pubsub.Consumer[*server_api.BoldValidationInput, []byte])
	for _, hash := range cfg.ModuleRoots {
		mr := common.HexToHash(hash)
		c, err := pubsub.NewConsumer[*server_api.BoldValidationInput, []byte](redisClient, server_api.RedisBoldStreamForRoot(cfg.StreamPrefix, mr), &cfg.ConsumerConfig)
		if err != nil {
			return nil, fmt.Errorf("creating consumer for validation: %w", err)
		}
		consumers[mr] = c
	}
	return &ExecutionSpawner{
		consumers: consumers,
		spawner:   spawner,
		config:    cfg,
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
					req.Value.ValidationInput, true).Await(ctx)
				if err != nil {
					log.Error("Creating BOLD execution", "error", err)
					return 0
				}
				var res interface{}
				if req.Value.NumDesiredLeaves != 0 {
					res, err = run.GetMachineHashesWithStepSize(
						req.Value.MachineStartIndex,
						req.Value.StepSize,
						req.Value.NumDesiredLeaves).Await(ctx)
				} else {
					res, err = run.GetProofAt(
						req.Value.MachineStartIndex,
					).Await(ctx)
				}
				if err != nil {
					log.Error("Getting machine hashes", "error", err)
					return 0
				}
				jsonRes, err := json.Marshal(res)
				if err != nil {
					log.Error("Marshaling result", "error", err)
					return 0
				}
				if err := c.SetResult(ctx, req.ID, jsonRes); err != nil {
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
			case <-time.After(s.config.StreamTimeout):
				log.Error("Waiting for redis streams timed out")
			case <-ctx.Done():
				log.Info("Context expired, failed to start")
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
