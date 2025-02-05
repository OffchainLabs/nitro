package timeboost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const EXPRESS_LANE_ROUND_SEQUENCE_KEY_PREFIX string = "expressLane.roundSequence." // Only written by sequencer holding CHOSEN (seqCoordinator) key
const EXPRESS_LANE_ACCEPTED_TX_KEY_PREFIX string = "expressLane.acceptedTx."       // Only written by sequencer holding CHOSEN (seqCoordinator) key

type RedisCoordinator struct {
	stopwaiter.StopWaiter
	roundDuration time.Duration
	client        redis.UniversalClient

	roundSeqMapMutex sync.Mutex
	roundSeqMap      *containers.LruCache[uint64, uint64]
}

func NewRedisCoordinator(redisUrl string, roundDuration time.Duration) (*RedisCoordinator, error) {
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		roundDuration: roundDuration,
		client:        redisClient,
		roundSeqMap:   containers.NewLruCache[uint64, uint64](4),
	}, nil
}

func (rc *RedisCoordinator) Start(ctxIn context.Context) {
	rc.StopWaiter.Start(ctxIn, rc)
}

func roundSequenceKeyFor(round uint64) string {
	return fmt.Sprintf("%s%d", EXPRESS_LANE_ROUND_SEQUENCE_KEY_PREFIX, round)
}

func (rc *RedisCoordinator) GetSequenceCount(round uint64) (uint64, error) {
	ctx := rc.GetContext()
	key := roundSequenceKeyFor(round)
	seqCountBytes, err := rc.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return arbmath.BytesToUint(seqCountBytes), nil
}

// Thread safe
func (rc *RedisCoordinator) UpdateSequenceCount(round, seqCount uint64) error {
	ctx := rc.GetContext()
	rc.roundSeqMapMutex.Lock()
	defer rc.roundSeqMapMutex.Unlock()

	curSeq, _ := rc.roundSeqMap.Get(round)
	if seqCount < curSeq {
		return nil // We only update seqCount to redis if it is greater than all the previously seen values
	}
	rc.roundSeqMap.Add(round, seqCount)

	key := roundSequenceKeyFor(round)
	if err := rc.client.Set(ctx, key, arbmath.UintToBytes(seqCount), rc.roundDuration*2).Err(); err != nil {
		return fmt.Errorf("couldn't set %s key for current round's global sequence count in redis: %w", key, err)
	}
	return nil
}

func acceptedTxKeyFor(round, seqNum uint64) string {
	return fmt.Sprintf("%s%d.%d", EXPRESS_LANE_ACCEPTED_TX_KEY_PREFIX, round, seqNum)
}

func (rc *RedisCoordinator) GetAcceptedTxs(round, startSeqNum, endSeqNum uint64) []*ExpressLaneSubmission {
	ctx := rc.GetContext()
	fetchMsg := func(key string) *ExpressLaneSubmission {
		msgBytes, err := rc.client.Get(ctx, key).Bytes()
		if err != nil {
			log.Error("Error fetching accepted expressLane tx", "key", key, "err", err)
			return nil
		}
		msgJson := JsonExpressLaneSubmission{}
		if err := json.Unmarshal(msgBytes, &msgJson); err != nil {
			log.Error("Error unmarshalling", "key", key, "err", err)
			return nil
		}
		msg, err := JsonSubmissionToGo(&msgJson)
		if err != nil {
			log.Error("Error converting JsonExpressLaneSubmission to ExpressLaneSubmission", "key", key, "err", err)
			return nil
		}
		return msg
	}

	var msgs []*ExpressLaneSubmission
	for seq := startSeqNum; seq <= endSeqNum; seq++ {
		if msg := fetchMsg(acceptedTxKeyFor(round, seq)); msg != nil {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

func (rc *RedisCoordinator) AddAcceptedTx(msg *ExpressLaneSubmission) error {
	ctx := rc.GetContext()
	msgJson, err := msg.ToJson()
	if err != nil {
		return fmt.Errorf("failed to convert ExpressLaneSubmission to JsonExpressLaneSubmission: %w", err)
	}
	msgBytes, err := json.Marshal(msgJson)
	if err != nil {
		return fmt.Errorf("failed to marshal JsonExpressLaneSubmission: %w", err)
	}
	key := acceptedTxKeyFor(msg.Round, msg.SequenceNumber)
	if err := rc.client.Set(ctx, key, msgBytes, rc.roundDuration*2).Err(); err != nil {
		return fmt.Errorf("couldn't set %s key for accepted expressLane transaction in redis: %w", key, err)
	}
	return nil
}
