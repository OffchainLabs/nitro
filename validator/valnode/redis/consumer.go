package redis

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
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
		c, err := pubsub.NewConsumer[*validator.ValidationInput, validator.GoGlobalState](redisClient, server_api.RedisStreamForRoot(mr), &cfg.ConsumerConfig)
		if err != nil {
			return nil, fmt.Errorf("creating consumer for validation: %w", err)
		}
		consumers[mr] = c
	}
	var (
		wg          sync.WaitGroup
		initialized atomic.Bool
	)
	initialized.Store(true)
	for i := 0; i < len(cfg.ModuleRoots); i++ {
		mr := cfg.ModuleRoots[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			done := waitForStream(redisClient, mr)
			select {
			case <-time.After(cfg.StreamTimeout):
				initialized.Store(false)
				return
			case <-done:
				return
			}
		}()
	}
	wg.Wait()
	if !initialized.Load() {
		return nil, fmt.Errorf("waiting for streams to be created: timed out")
	}
	return &ValidationServer{
		consumers: consumers,
		spawner:   spawner,
	}, nil
}

func streamExists(client redis.UniversalClient, streamName string) bool {
	groups, err := client.XInfoStream(context.TODO(), streamName).Result()
	if err != nil {
		log.Error("Reading redis streams", "error", err)
		return false
	}
	return groups.Groups > 0
}

func waitForStream(client redis.UniversalClient, streamName string) chan struct{} {
	var ret chan struct{}
	go func() {
		if streamExists(client, streamName) {
			ret <- struct{}{}
		}
		time.Sleep(time.Millisecond * 100)
	}()
	return ret
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
	// Timeout on polling for existence of each redis stream.
	StreamTimeout time.Duration `koanf:"stream-timeout"`
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
