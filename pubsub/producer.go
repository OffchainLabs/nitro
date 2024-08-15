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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
)

const (
	messageKey   = "msg"
	defaultGroup = "default_consumer_group"
)

type MsgIdAndPromise[Response any] struct {
	msgID   string
	promise *containers.Promise[Response]
}

type Producer[Request any, Response any] struct {
	stopwaiter.StopWaiter
	id          string
	client      redis.UniversalClient
	redisStream string
	redisGroup  string
	cfg         *ProducerConfig

	promisesLock sync.RWMutex
	promises     map[string]*MsgIdAndPromise[Response]

	// Used for checking responses from consumers iteratively
	// For the first time when Produce is called.
	once sync.Once
}

type ProducerConfig struct {
	// Interval duration for checking the result set by consumers.
	CheckResultInterval time.Duration `koanf:"check-result-interval"`
	// Timeout of entry's written to redis by producer
	ResponseEntryTimeout time.Duration `koanf:"response-entry-timeout"`
}

var DefaultProducerConfig = ProducerConfig{
	CheckResultInterval:  5 * time.Second,
	ResponseEntryTimeout: time.Hour,
}

var TestProducerConfig = ProducerConfig{
	CheckResultInterval:  5 * time.Millisecond,
	ResponseEntryTimeout: time.Minute,
}

func ProducerAddConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".check-result-interval", DefaultProducerConfig.CheckResultInterval, "interval in which producer checks pending messages whether consumer processing them is inactive")
	f.Duration(prefix+".response-entry-timeout", DefaultProducerConfig.ResponseEntryTimeout, "timeout after which responses written from producer to the redis are cleared. Currently used for the key mapping unique request id to redis stream message id")
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
		promises:    make(map[string]*MsgIdAndPromise[Response]),
	}, nil
}

func setMaxMsgIdInt(maxMsgIdInt *[2]uint64, msgId string) error {
	idParts := strings.Split(msgId, "-")
	if len(idParts) != 2 {
		return fmt.Errorf("invalid i.d: %v", msgId)
	}
	idTimeStamp, err := strconv.ParseUint(idParts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid i.d: %v err: %w", msgId, err)
	}
	if idTimeStamp < maxMsgIdInt[0] {
		return nil
	}
	idSerial, err := strconv.ParseUint(idParts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid i.d serial: %v err: %w", msgId, err)
	}
	if idTimeStamp > maxMsgIdInt[0] {
		maxMsgIdInt[0] = idTimeStamp
		maxMsgIdInt[1] = idSerial
		return nil
	}
	// idTimeStamp == maxMsgIdInt[0]
	if idSerial > maxMsgIdInt[1] {
		maxMsgIdInt[1] = idSerial
	}
	return nil
}

// checkResponses checks iteratively whether response for the promise is ready.
func (p *Producer[Request, Response]) checkResponses(ctx context.Context) time.Duration {
	maxMsgIdInt := [2]uint64{0, 0}
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	responded := 0
	errored := 0
	for id, msgIDAndPromise := range p.promises {
		if ctx.Err() != nil {
			return 0
		}
		msgKey := MessageKeyFor(p.redisStream, id)
		res, err := p.client.Get(ctx, msgKey).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				log.Error("Error reading value in redis", "key", id, "error", err)
			}
			continue
		}
		// We keep track of a maxMsgId of a successfully solved request, because messages
		// with id lower than this are either ack-ed or in PEL, so its safe to call XTRIMMINID on maxMsgId
		errSetId := setMaxMsgIdInt(&maxMsgIdInt, msgIDAndPromise.msgID)
		if errSetId != nil {
			log.Error("error setting maxMsgId", "err", err)
			return p.cfg.CheckResultInterval
		}
		var resp Response
		if err := json.Unmarshal([]byte(res), &resp); err != nil {
			msgIDAndPromise.promise.ProduceError(fmt.Errorf("error unmarshalling: %w", err))
			log.Error("Error unmarshaling", "value", res, "error", err)
			errored++
		} else {
			msgIDAndPromise.promise.Produce(resp)
			responded++
		}
		// Try deleting UNIQUEID_MSGID_MAP_KEY corresponding to this id from redis
		if err := p.client.Del(ctx, msgKey+UNIQUEID_MSGID_MAP_KEY).Err(); err != nil {
			log.Error("Error deleting key from redis that flags that a request is being processed", "err", err)
		}
		delete(p.promises, id)
	}
	var trimmed int64
	var trimErr error
	maxMsgId := "+"
	// If at least response for one promise was found, find the maximum of the found ones and XTRIMMINID from that msg id + 1
	if maxMsgIdInt[0] > 0 {
		maxMsgId = fmt.Sprintf("%d-%d", maxMsgIdInt[0], maxMsgIdInt[1]+1)
		trimmed, trimErr = p.client.XTrimMinID(ctx, p.redisStream, maxMsgId).Result()
	}
	log.Trace("trimming", "xTrimMinID", maxMsgId, "trimmed", trimmed, "responded", responded, "errored", errored, "trim-err", trimErr)
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

func (p *Producer[Request, Response]) produce(ctx context.Context, id string, value Request) (*containers.Promise[Response], error) {
	if id != "" {
		msgKey := MessageKeyFor(p.redisStream, id)

		// If the request has already been solved by a consumer
		if res, err := p.client.Get(ctx, msgKey).Result(); err == nil {
			var resp Response
			if err := json.Unmarshal([]byte(res), &resp); err != nil {
				log.Error("Error unmarshaling", "value", res, "error", err)
				return nil, fmt.Errorf("error unmarshalling: %w", err)
			} else {
				pr := containers.NewPromise[Response](nil)
				pr.Produce(resp)
				return &pr, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			log.Error("error while checking for response to a request in redis", "err", err)
		}

		// Check for duplicate unsolved request messages in stream
		if res, err := p.client.Get(ctx, msgKey+UNIQUEID_MSGID_MAP_KEY).Result(); err == nil {
			log.Info("Request already submitted by another producer", "msgId", res, "requestUniqueId", id)
			p.promisesLock.Lock()
			defer p.promisesLock.Unlock()
			pr := containers.NewPromise[Response](nil)
			p.promises[id] = &MsgIdAndPromise[Response]{
				msgID:   res,
				promise: &pr,
			}
			return &pr, nil
		}
	}

	val, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling value: %w", err)
	}
	// catching the promiseLock before we sendXadd makes sure promise ids will be always ascending
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	msgId, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.redisStream,
		Values: map[string]any{messageKey: val},
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("adding values to redis: %w", err)
	}

	if id == "" {
		// If unique id doesn't exist, use the newly created msgId as unique id and follow the same steps as before
		log.Info("Request doesn't have a unique identifier (SelfHash field set), defaulting to using redis stream messageId", "msgId", msgId)
		id = msgId
	}

	// Try adding key that flags that request is being processed
	if err := p.client.Set(ctx, MessageKeyFor(p.redisStream, id)+UNIQUEID_MSGID_MAP_KEY, msgId, p.cfg.ResponseEntryTimeout).Err(); err != nil {
		log.Error("Error adding key to redis that flags that a request is being processed, stream may encounter duplicate requests", "err", err)
	}

	pr := containers.NewPromise[Response](nil)
	p.promises[id] = &MsgIdAndPromise[Response]{
		msgID:   msgId,
		promise: &pr,
	}
	return &pr, nil
}

func (p *Producer[Request, Response]) Produce(ctx context.Context, id string, value Request) (*containers.Promise[Response], error) {
	log.Debug("Redis stream producing", "value", value)
	p.once.Do(func() {
		p.StopWaiter.CallIteratively(p.checkResponses)
	})
	return p.produce(ctx, id, value)
}
