// Package pubsub implements publisher/subscriber model (one to many).
// During normal operation, publisher returns "Promise" when publishing a
// message, which will return response from consumer when awaited.
// If the consumer processing the request becomes inactive, message is
// re-inserted (if EnableReproduce flag is enabled), and will be picked up by
// another consumer.
// We are assuming here that keepAliveTimeout is set to some sensible value
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

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
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

	// Used for checking responses from consumers iteratively
	// For the first time when Produce is called.
	once sync.Once
}

// lint:require-exhaustive-initialization
type ProducerConfig struct {
	// Interval duration for checking the result set by consumers.
	CheckResultInterval time.Duration `koanf:"check-result-interval"`
	// RequestTimeout is a TTL for any message sent to the redis stream
	RequestTimeout time.Duration `koanf:"request-timeout"`
}

var DefaultProducerConfig = ProducerConfig{
	CheckResultInterval: 5 * time.Second,
	RequestTimeout:      3 * time.Hour,
}

var TestProducerConfig = ProducerConfig{
	CheckResultInterval: 5 * time.Millisecond,
	RequestTimeout:      time.Minute,
}

func ProducerAddConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".check-result-interval", DefaultProducerConfig.CheckResultInterval, "interval in which producer checks pending messages whether consumer processing them is inactive")
	f.Duration(prefix+".request-timeout", DefaultProducerConfig.RequestTimeout, "timeout after which the message in redis stream is considered as errored, this prevents workers from working on wrong requests indefinitely")
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

func getUintParts(msgId string) ([2]uint64, error) {
	idParts := strings.Split(msgId, "-")
	if len(idParts) != 2 {
		return [2]uint64{}, fmt.Errorf("invalid i.d: %v", msgId)
	}
	idTimeStamp, err := strconv.ParseUint(idParts[0], 10, 64)
	if err != nil {
		return [2]uint64{}, fmt.Errorf("invalid i.d: %v err: %w", msgId, err)
	}
	idSerial, err := strconv.ParseUint(idParts[1], 10, 64)
	if err != nil {
		return [2]uint64{}, fmt.Errorf("invalid i.d serial: %v err: %w", msgId, err)
	}
	return [2]uint64{idTimeStamp, idSerial}, nil
}

// cmpMsgId compares two msgid's and returns (0) if equal, (-1) if msgId1 < msgId2, (1) if msgId1 > msgId2, (-2) if not comparable (or error)
func cmpMsgId(msgId1, msgId2 string) int {
	id1, err := getUintParts(msgId1)
	if err != nil {
		log.Trace("error comparing msgIds", "msgId1", msgId1, "msgId2", msgId2)
		return -2
	}
	id2, err := getUintParts(msgId2)
	if err != nil {
		log.Trace("error comparing msgIds", "msgId1", msgId1, "msgId2", msgId2)
		return -2
	}
	if id1[0] < id2[0] {
		return -1
	} else if id1[0] > id2[0] {
		return 1
	} else if id1[1] < id2[1] {
		return -1
	} else if id1[1] > id2[1] {
		return 1
	}
	return 0
}

// checkResponses checks iteratively whether response for the promise is ready.
func (p *Producer[Request, Response]) checkResponses(ctx context.Context) time.Duration {
	log.Debug("redis producer: check responses starting")
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	responded := 0
	errored := 0
	checked := 0
	allowedOldestID := fmt.Sprintf("%d-0", time.Now().Add(-p.cfg.RequestTimeout).UnixMilli())
	for id, promise := range p.promises {
		if ctx.Err() != nil {
			return 0
		}
		checked++
		// First check if there is an error for this promise
		errorKey := ErrorKeyFor(p.redisStream, id)
		errorResponse, err := p.client.Get(ctx, errorKey).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			// If we get an error that is not redis.Nil, then log it and continue.
			log.Error("Error reading error in redis", "key", errorKey, "error", err)
			continue
		}
		if err == nil {
			// If we found the error key, then delete it and return the error to the promise and continue.
			p.client.Del(ctx, errorKey)
			promise.ProduceError(errors.New(errorResponse))
			log.Debug("consumer returned error", "error", errorResponse, "msgId", id)
			errored++
			delete(p.promises, id)
			continue
		}
		// If we do not find the error key, then check for the result key.
		resultKey := ResultKeyFor(p.redisStream, id)
		res, err := p.client.Get(ctx, resultKey).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				log.Error("Error reading value in redis", "key", resultKey, "error", err)
			} else if cmpMsgId(id, allowedOldestID) == -1 {
				// The request this producer is waiting for has been past its TTL or is older than current PEL's lower,
				// so safe to error and stop tracking this promise
				promise.ProduceError(errors.New("error getting response, request has been waiting for too long"))
				log.Debug("request timed out waiting for response", "msgId", id, "allowedOldestId", allowedOldestID)
				errored++
				delete(p.promises, id)
			}
			continue
		}
		var resp Response
		if err := json.Unmarshal([]byte(res), &resp); err != nil {
			promise.ProduceError(fmt.Errorf("error unmarshalling: %w", err))
			log.Error("redis producer: Error unmarshaling", "value", res, "error", err)
			errored++
		} else {
			promise.Produce(resp)
			responded++
		}
		p.client.Del(ctx, resultKey)
		delete(p.promises, id)
	}
	log.Debug("checkResponses", "responded", responded, "errored", errored, "checked", checked)
	return p.cfg.CheckResultInterval
}

func (p *Producer[Request, Response]) clearMessages(ctx context.Context) time.Duration {
	pelData, err := p.client.XPending(ctx, p.redisStream, p.redisGroup).Result()
	if err != nil {
		log.Error("error getting PEL data from xpending, xtrimming is disabled", "err", err)
	}
	// XDEL on consumer side already deletes acked messages (mark as deleted) but doesn't claim the memory back, XTRIM helps in claiming this memory in normal conditions
	// pelData might be outdated when we do the xtrim, but that's ok as the messages are also being trimmed by other producers
	if pelData != nil && pelData.Lower != "" {
		trimmed, trimErr := p.client.XTrimMinID(ctx, p.redisStream, pelData.Lower).Result()
		log.Debug("trimming", "xTrimMinID", pelData.Lower, "trimmed", trimmed, "trim-err", trimErr)
		// Check if pelData.Lower has been past its TTL and if it is then ack it to remove from PEL and delete it, once
		// its taken out from PEL the producer that sent this request will handle the corresponding promise accordingly (as its past TTL)
		allowedOldestID := fmt.Sprintf("%d-0", time.Now().Add(-p.cfg.RequestTimeout).UnixMilli())
		if cmpMsgId(pelData.Lower, allowedOldestID) == -1 {
			if err := p.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   p.redisStream,
				Group:    p.redisGroup,
				Consumer: p.id,
				MinIdle:  0,
				Messages: []string{pelData.Lower},
			}).Err(); err != nil {
				log.Error("error claiming PEL's lower message that's past its TTL", "msgID", pelData.Lower, "err", err)
				return 5 * p.cfg.CheckResultInterval
			}
			if _, err := p.client.XAck(ctx, p.redisStream, p.redisGroup, pelData.Lower).Result(); err != nil {
				log.Error("error acking PEL's lower message that's past its TTL", "msgID", pelData.Lower, "err", err)
				return 5 * p.cfg.CheckResultInterval
			}
			if _, err := p.client.XDel(ctx, p.redisStream, pelData.Lower).Result(); err != nil {
				log.Error("error deleting PEL's lower message that's past its TTL", "msgID", pelData.Lower, "err", err)
				return 5 * p.cfg.CheckResultInterval
			}
			return 0
		}
	}
	return 5 * p.cfg.CheckResultInterval
}

func (p *Producer[Request, Response]) Start(ctx context.Context) {
	p.StopWaiter.Start(ctx, p)
}

func (p *Producer[Request, Response]) promisesLen() int {
	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	return len(p.promises)
}

func (p *Producer[Request, Response]) produce(ctx context.Context, value Request) (*containers.Promise[Response], error) {
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
	promise := containers.NewPromise[Response](nil)
	p.promises[msgId] = &promise
	return &promise, nil
}

func (p *Producer[Request, Response]) Produce(ctx context.Context, value Request) (*containers.Promise[Response], error) {
	log.Debug("Redis stream producing", "value", value)
	p.once.Do(func() {
		p.StopWaiter.CallIteratively(p.checkResponses)
		p.StopWaiter.CallIteratively(p.clearMessages)
	})
	return p.produce(ctx, value)
}
