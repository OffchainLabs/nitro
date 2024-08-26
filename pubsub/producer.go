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
	// RequestTimeout is a TTL for any message sent to the redis stream
	RequestTimeout time.Duration `koanf:"request-timeout"`
}

var DefaultProducerConfig = ProducerConfig{
	CheckResultInterval:  5 * time.Second,
	ResponseEntryTimeout: time.Hour,
	RequestTimeout:       time.Hour, // should we increase this?
}

var TestProducerConfig = ProducerConfig{
	CheckResultInterval:  5 * time.Millisecond,
	ResponseEntryTimeout: time.Minute,
	RequestTimeout:       time.Minute,
}

func ProducerAddConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".check-result-interval", DefaultProducerConfig.CheckResultInterval, "interval in which producer checks pending messages whether consumer processing them is inactive")
	f.Duration(prefix+".response-entry-timeout", DefaultProducerConfig.ResponseEntryTimeout, "timeout after which responses written from producer to the redis are cleared. Currently used for the key mapping unique request id to redis stream message id")
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
		promises:    make(map[string]*MsgIdAndPromise[Response]),
	}, nil
}

// cmpMsgId compares two msgid's and returns (0) if equal, (-1) if msgId1 < msgId2, (1) if msgId1 > msgId2, (-2) if not comparable (or error)
func cmpMsgId(msgId1, msgId2 string) int {
	getUintParts := func(msgId string) ([2]uint64, error) {
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
	pelData, err := p.client.XPending(ctx, p.redisStream, p.redisGroup).Result()
	if err != nil {
		log.Error("error getting PEL data from xpending, xtrimming is disabled", "err", err)
	}
	deletePromise := func(id string) {
		// Try deleting UNIQUEID_MSGID_MAP_KEY corresponding to this id from redis
		if err := p.client.Del(ctx, MessageKeyFor(p.redisStream, id)+UNIQUEID_MSGID_MAP_KEY).Err(); err != nil {
			log.Error("Error deleting key from redis that flags that a request is being processed", "err", err)
		}
		delete(p.promises, id)
	}
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
			} else {
				// The request this producer is waiting for has been past its TTL or is older than current PEL's lower,
				// so safe to error and stop tracking this promise
				allowedOldestID := fmt.Sprintf("%d-0", time.Now().Add(-p.cfg.RequestTimeout).UnixMilli())
				if pelData != nil && pelData.Lower != "" {
					allowedOldestID = pelData.Lower
				}
				if cmpMsgId(msgIDAndPromise.msgID, allowedOldestID) == -1 {
					msgIDAndPromise.promise.ProduceError(errors.New("error getting response, request has been waiting for too long"))
					log.Error("error getting response, request has been waiting past its TTL")
					errored++
					deletePromise(id)
				}
			}
			continue
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
		deletePromise(id)
	}
	// XDEL on consumer side already deletes acked messages (mark as deleted) but doesnt claim the memory back, XTRIM helps in claiming this memory in normal conditions
	// pelData might be outdated when we do the xtrim, but thats ok as the messages are also being trimmed by other producers
	if pelData != nil && pelData.Lower != "" {
		trimmed, trimErr := p.client.XTrimMinID(ctx, p.redisStream, pelData.Lower).Result()
		log.Trace("trimming", "xTrimMinID", pelData.Lower, "trimmed", trimmed, "responded", responded, "errored", errored, "trim-err", trimErr)
		// Check if pelData.Lower has been past its TTL and if it is then ack it to remove from PEL and delete it, once
		// its taken out from PEL the producer that sent this request will handle the corresponding promise accordingly (if PEL is non-empty)
		allowedOldestID := fmt.Sprintf("%d-0", time.Now().Add(-p.cfg.RequestTimeout).UnixMilli())
		if cmpMsgId(pelData.Lower, allowedOldestID) == -1 {
			if err := p.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   p.redisStream,
				Group:    p.redisGroup,
				Consumer: p.id,
				MinIdle:  0,
				Messages: []string{pelData.Lower},
			}).Err(); err != nil {
				log.Error("error claiming PEL's lower message thats past its TTL", "msgID", pelData.Lower, "err", err)
				return p.cfg.CheckResultInterval
			}
			if _, err := p.client.XAck(ctx, p.redisStream, p.redisGroup, pelData.Lower).Result(); err != nil {
				log.Error("error acking PEL's lower message thats past its TTL", "msgID", pelData.Lower, "err", err)
				return p.cfg.CheckResultInterval
			}
			if _, err := p.client.XDel(ctx, p.redisStream, pelData.Lower).Result(); err != nil {
				log.Error("error deleting PEL's lower message thats past its TTL", "msgID", pelData.Lower, "err", err)
			}
		}
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
