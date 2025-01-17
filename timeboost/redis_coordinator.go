package timeboost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
)

const EXPRESS_LANE_ROUND_SEQUENCE_KEY_PREFIX string = "expressLane.roundSequence." // Only written by sequencer holding CHOSEN (seqCoordinator) key
const EXPRESS_LANE_ACCEPTED_TX_KEY_PREFIX string = "expressLane.acceptedTx."       // Only written by sequencer holding CHOSEN (seqCoordinator) key

type RedisCoordinator struct {
	roundDuration time.Duration
	Client        redis.UniversalClient

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
		Client:        redisClient,
		roundSeqMap:   containers.NewLruCache[uint64, uint64](4),
	}, nil
}

func roundSequenceKeyFor(round uint64) string {
	return fmt.Sprintf("%s%d", EXPRESS_LANE_ROUND_SEQUENCE_KEY_PREFIX, round)
}

func (rc *RedisCoordinator) GetSequenceCount(ctx context.Context, round uint64) (uint64, error) {
	key := roundSequenceKeyFor(round)
	seqCountBytes, err := rc.Client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return arbmath.BytesToUint(seqCountBytes), nil
}

// Thread safe
func (rc *RedisCoordinator) UpdateSequenceCount(ctx context.Context, round, seqCount uint64) error {
	rc.roundSeqMapMutex.Lock()
	defer rc.roundSeqMapMutex.Unlock()

	curSeq, _ := rc.roundSeqMap.Get(round)
	if seqCount < curSeq {
		return nil // We only update seqCount to redis if it is greater than all the previously seen values
	}
	rc.roundSeqMap.Add(round, seqCount)

	key := roundSequenceKeyFor(round)
	if err := rc.Client.Set(ctx, key, arbmath.UintToBytes(seqCount), rc.roundDuration*2).Err(); err != nil {
		return fmt.Errorf("couldn't set %s key for current round's global sequence count in redis: %w", key, err)
	}
	return nil
}

func acceptedTxKeyFor(round, seqNum uint64) string {
	return fmt.Sprintf("%s%d.%d", EXPRESS_LANE_ACCEPTED_TX_KEY_PREFIX, round, seqNum)
}

func (rc *RedisCoordinator) GetAcceptedTxs(ctx context.Context, round, startSeqNum uint64) []*ExpressLaneSubmission {
	fetchMsg := func(key string) *ExpressLaneSubmission {
		msgBytes, err := rc.Client.Get(ctx, key).Bytes()
		if err != nil {
			log.Error("Error fetching accepted expressLane tx", "err", err)
			return nil
		}
		msgJson := JsonExpressLaneSubmission{}
		if err := json.Unmarshal(msgBytes, &msgJson); err != nil {
			log.Error("Error unmarshalling", "err", err)
			return nil
		}
		msg, err := JsonSubmissionToGo(&msgJson)
		if err != nil {
			log.Error("Error converting JsonExpressLaneSubmission to ExpressLaneSubmission", "err", err)
			return nil
		}
		return msg
	}

	var msgs []*ExpressLaneSubmission
	prefix := fmt.Sprintf("%s%d.", EXPRESS_LANE_ACCEPTED_TX_KEY_PREFIX, round)
	cursor := uint64(0)
	for {
		keys, cursor, err := rc.Client.Scan(ctx, cursor, prefix+"*", 0).Result()
		if err != nil {
			break // Best effort
		}
		for _, key := range keys {
			seq, err := strconv.Atoi(strings.TrimPrefix(key, prefix))
			if err != nil {
				log.Error("")
				continue
			}
			// #nosec G115
			if uint64(seq) >= startSeqNum {
				if msg := fetchMsg(key); msg != nil {
					msgs = append(msgs, msg)
				}
			}
		}
		if cursor == 0 {
			break
		}
	}
	return msgs
}

func (rc *RedisCoordinator) AddAcceptedTx(ctx context.Context, msg *ExpressLaneSubmission) error {
	msgJson, err := msg.ToJson()
	if err != nil {
		return fmt.Errorf("failed to convert ExpressLaneSubmission to JsonExpressLaneSubmission: %w", err)
	}
	msgBytes, err := json.Marshal(msgJson)
	if err != nil {
		return fmt.Errorf("failed to marshal JsonExpressLaneSubmission: %w", err)
	}
	key := acceptedTxKeyFor(msg.Round, msg.SequenceNumber)
	if err := rc.Client.Set(ctx, key, msgBytes, rc.roundDuration*2).Err(); err != nil {
		return fmt.Errorf("couldn't set %s key for accepted expressLane transaction in redis: %w", key, err)
	}
	return nil
}
