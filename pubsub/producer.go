// Package pubsub implements publisher/subscriber model (one to many).
// During normal operation, publisher returns "Promise" when publishing a
// message, which will return resposne from consumer when awaited.
// If the consumer processing the request becomes inactive, message is
// re-inserted (if EnableReproduce flag is enabled), and will be picked up by
// another consumer.
// We are assuming here that keeepAliveTimeout is set to some sensible value
// and once consumer becomes inactive, it doesn't activate without restart.
package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/spf13/pflag"
)

const (
	messageKey   = "msg"
	defaultGroup = "default_consumer_group"
)

type Producer[Request any, Response any] struct {
	stopwaiter.StopWaiter
	id          string
	client      redis.UniversalClient
	redisStream string
	redisGroup  string
	cfg         *ProducerConfig

	promisesLock sync.RWMutex
	promises     map[string]*containers.Promise[Response]

	// Used for running checks for pending messages with inactive consumers
	// and checking responses from consumers iteratively for the first time when
	// Produce is called.
	once sync.Once
}

type ProducerConfig struct {
	// When enabled, messages that are sent to consumers that later die before
	// processing them, will be re-inserted into the stream to be proceesed by
	// another consumer
	EnableReproduce bool `koanf:"enable-reproduce"`
	// Interval duration in which producer checks for pending messages delivered
	// to the consumers that are currently inactive.
	CheckPendingInterval time.Duration `koanf:"check-pending-interval"`
	// Duration after which consumer is considered to be dead if heartbeat
	// is not updated.
	KeepAliveTimeout time.Duration `koanf:"keepalive-timeout"`
	// Interval duration for checking the result set by consumers.
	CheckResultInterval time.Duration `koanf:"check-result-interval"`
}

var DefaultProducerConfig = ProducerConfig{
	EnableReproduce:      true,
	CheckPendingInterval: time.Second,
	KeepAliveTimeout:     5 * time.Minute,
	CheckResultInterval:  5 * time.Second,
}

var TestProducerConfig = ProducerConfig{
	EnableReproduce:      true,
	CheckPendingInterval: 10 * time.Millisecond,
	KeepAliveTimeout:     100 * time.Millisecond,
	CheckResultInterval:  5 * time.Millisecond,
}

func ProducerAddConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable-reproduce", DefaultProducerConfig.EnableReproduce, "when enabled, messages with dead consumer will be re-inserted into the stream")
	f.Duration(prefix+".check-pending-interval", DefaultProducerConfig.CheckPendingInterval, "interval in which producer checks pending messages whether consumer processing them is inactive")
	f.Duration(prefix+".keepalive-timeout", DefaultProducerConfig.KeepAliveTimeout, "timeout after which consumer is considered inactive if heartbeat wasn't performed")
}

func NewProducer[Request any, Response any](client redis.UniversalClient, streamName string, cfg *ProducerConfig) (*Producer[Request, Response], error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if streamName == "" {
		return nil, fmt.Errorf("stream name cannot be empty")
	}
	return &Producer[Request, Response]{
		id:          uuid.NewString(),
		client:      client,
		redisStream: streamName,
		redisGroup:  streamName, // There is 1-1 mapping of redis stream and consumer group.
		cfg:         cfg,
		promises:    make(map[string]*containers.Promise[Response]),
	}, nil
}

func (p *Producer[Request, Response]) errorPromisesFor(msgs []*Message[Request]) {
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	for _, msg := range msgs {
		if promise, found := p.promises[msg.ID]; found {
			promise.ProduceError(fmt.Errorf("internal error, consumer died while serving the request"))
			delete(p.promises, msg.ID)
		}
	}
}

// checkAndReproduce reproduce pending messages that were sent to consumers
// that are currently inactive.
func (p *Producer[Request, Response]) checkAndReproduce(ctx context.Context) time.Duration {
	msgs, err := p.checkPending(ctx)
	if err != nil {
		log.Error("Checking pending messages", "error", err)
		return p.cfg.CheckPendingInterval
	}
	if len(msgs) == 0 {
		return p.cfg.CheckPendingInterval
	}
	if !p.cfg.EnableReproduce {
		p.errorPromisesFor(msgs)
		return p.cfg.CheckPendingInterval
	}
	acked := make(map[string]Request)
	for _, msg := range msgs {
		if _, err := p.client.XAck(ctx, p.redisStream, p.redisGroup, msg.ID).Result(); err != nil {
			log.Error("ACKing message", "error", err)
			continue
		}
		acked[msg.ID] = msg.Value
	}
	for k, v := range acked {
		// Only re-insert messages that were removed the the pending list first.
		_, err := p.reproduce(ctx, v, k)
		if err != nil {
			log.Error("Re-inserting pending messages with inactive consumers", "error", err)
		}
	}
	return p.cfg.CheckPendingInterval
}

// checkResponses checks iteratively whether response for the promise is ready.
func (p *Producer[Request, Response]) checkResponses(ctx context.Context) time.Duration {
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	for id, promise := range p.promises {
		res, err := p.client.Get(ctx, id).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			log.Error("Error reading value in redis", "key", id, "error", err)
		}
		var resp Response
		if err := json.Unmarshal([]byte(res), &resp); err != nil {
			log.Error("Error unmarshaling", "value", res, "error", err)
			continue
		}
		promise.Produce(resp)
		delete(p.promises, id)
	}
	return p.cfg.CheckResultInterval
}

func (p *Producer[Request, Response]) Start(ctx context.Context) {
	p.StopWaiter.Start(ctx, p)
}

func (p *Producer[Request, Response]) promisesLen() int {
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	return len(p.promises)
}

// reproduce is used when Producer claims ownership on the pending
// message that was sent to inactive consumer and reinserts it into the stream,
// so that seamlessly return the answer in the same promise.
func (p *Producer[Request, Response]) reproduce(ctx context.Context, value Request, oldKey string) (*containers.Promise[Response], error) {
	val, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling value: %w", err)
	}
	id, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.redisStream,
		Values: map[string]any{messageKey: val},
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("adding values to redis: %w", err)
	}
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	promise := p.promises[oldKey]
	if oldKey != "" && promise == nil {
		// This will happen if the old consumer became inactive but then ack_d
		// the message afterwards.
		return nil, fmt.Errorf("error reproducing the message, could not find existing one")
	}
	if oldKey == "" || promise == nil {
		pr := containers.NewPromise[Response](nil)
		promise = &pr
	}
	delete(p.promises, oldKey)
	p.promises[id] = promise
	return promise, nil
}

func (p *Producer[Request, Response]) Produce(ctx context.Context, value Request) (*containers.Promise[Response], error) {
	log.Debug("Redis stream producing", "value", value)
	p.once.Do(func() {
		p.StopWaiter.CallIteratively(p.checkAndReproduce)
		p.StopWaiter.CallIteratively(p.checkResponses)
	})
	return p.reproduce(ctx, value, "")
}

// Check if a consumer is with specified ID is alive.
func (p *Producer[Request, Response]) isConsumerAlive(ctx context.Context, consumerID string) bool {
	if _, err := p.client.Get(ctx, heartBeatKey(consumerID)).Int64(); err != nil {
		return false
	}
	return true
}

func (p *Producer[Request, Response]) havePromiseFor(messageID string) bool {
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	_, found := p.promises[messageID]
	return found
}

func (p *Producer[Request, Response]) checkPending(ctx context.Context) ([]*Message[Request], error) {
	pendingMessages, err := p.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: p.redisStream,
		Group:  p.redisGroup,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("querying pending messages: %w", err)
	}
	if len(pendingMessages) == 0 {
		return nil, nil
	}
	// IDs of the pending messages with inactive consumers.
	var ids []string
	active := make(map[string]bool)
	for _, msg := range pendingMessages {
		// Ignore messages not produced by this producer.
		if !p.havePromiseFor(msg.ID) {
			continue
		}
		alive, found := active[msg.Consumer]
		if !found {
			alive = p.isConsumerAlive(ctx, msg.Consumer)
			active[msg.Consumer] = alive
		}
		if alive {
			continue
		}
		ids = append(ids, msg.ID)
	}
	if len(ids) == 0 {
		log.Trace("There are no pending messages with inactive consumers")
		return nil, nil
	}
	log.Info("Attempting to claim", "messages", ids)
	claimedMsgs, err := p.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   p.redisStream,
		Group:    p.redisGroup,
		Consumer: p.id,
		MinIdle:  p.cfg.KeepAliveTimeout,
		Messages: ids,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("claiming ownership on messages: %v, error: %w", ids, err)
	}
	var res []*Message[Request]
	for _, msg := range claimedMsgs {
		data, ok := (msg.Values[messageKey]).(string)
		if !ok {
			return nil, fmt.Errorf("casting request: %v to bytes", msg.Values[messageKey])
		}
		var req Request
		if err := json.Unmarshal([]byte(data), &req); err != nil {
			return nil, fmt.Errorf("marshaling value: %v, error: %w", msg.Values[messageKey], err)
		}
		res = append(res, &Message[Request]{
			ID:    msg.ID,
			Value: req,
		})
	}
	return res, nil
}
