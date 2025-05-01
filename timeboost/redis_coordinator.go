package timeboost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const EXPRESS_LANE_ROUND_SEQUENCE_KEY_PREFIX string = "expressLane.roundSequence." // Only written by sequencer holding CHOSEN (seqCoordinator) key
const EXPRESS_LANE_ACCEPTED_TX_KEY_PREFIX string = "expressLane.acceptedTx."       // Only written by sequencer holding CHOSEN (seqCoordinator) key

type roundSeqUpdateItem struct {
	round    uint64
	sequence uint64
}

type RedisCoordinator struct {
	stopwaiter.StopWaiter
	roundTimingInfo *RoundTimingInfo
	client          redis.UniversalClient

	roundSeqMap        *containers.LruCache[uint64, uint64]
	roundSeqUpdateChan chan roundSeqUpdateItem
	msgChan            chan *ExpressLaneSubmission
}

func NewRedisCoordinator(redisUrl string, roundTimingInfo *RoundTimingInfo, updateEventsChannelSize uint64) (*RedisCoordinator, error) {
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		roundTimingInfo:    roundTimingInfo,
		client:             redisClient,
		roundSeqMap:        containers.NewLruCache[uint64, uint64](4),
		roundSeqUpdateChan: make(chan roundSeqUpdateItem, updateEventsChannelSize),
		msgChan:            make(chan *ExpressLaneSubmission, updateEventsChannelSize),
	}, nil
}

func (rc *RedisCoordinator) Start(ctxIn context.Context) {
	rc.StopWaiter.Start(ctxIn, rc)
	rc.LaunchThread(rc.trackSequenceCountUpdates)
	rc.LaunchThread(rc.trackAcceptedTxAddition)
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

func (rc *RedisCoordinator) UpdateSequenceCount(round, sequence uint64) error {
	roundSeqUpdate := roundSeqUpdateItem{
		round:    round,
		sequence: sequence,
	}
	select {
	case rc.roundSeqUpdateChan <- roundSeqUpdate:
	default:
		log.Warn("Unable to queue round sequence update operation in redis coordinator", "round", round, "sequence", sequence)
	}
	return nil
}

func (rc *RedisCoordinator) trackSequenceCountUpdates(ctx context.Context) {
	for {
		var roundSeqUpdate roundSeqUpdateItem
		select {
		case update := <-rc.roundSeqUpdateChan:
			if update.round < rc.roundTimingInfo.RoundNumber() ||
				update.round < roundSeqUpdate.round ||
				(update.round == roundSeqUpdate.round && update.sequence < roundSeqUpdate.sequence) {
				// This prevents stale roundSeqUpdates from being written to redis and unclogs roundSeqUpdateChan
				continue
			}
			roundSeqUpdate = update
			// Attempt to pull upto next 5 updates from the channel (batching logic)
			for i := 0; i < 5; i++ {
				select {
				case update := <-rc.roundSeqUpdateChan:
					if update.round < rc.roundTimingInfo.RoundNumber() ||
						update.round < roundSeqUpdate.round ||
						(update.round == roundSeqUpdate.round && update.sequence < roundSeqUpdate.sequence) {
						// This prevents stale roundSeqUpdates from being written to redis and unclogs roundSeqUpdateChan
						continue
					}
					roundSeqUpdate = update // update roundSeqUpdate with local maxima
				case <-ctx.Done():
					return
				default:
				}
			}
		case <-ctx.Done():
			return
		}
		curSeq, _ := rc.roundSeqMap.Get(roundSeqUpdate.round)
		if roundSeqUpdate.sequence <= curSeq {
			continue
		}
		rc.roundSeqMap.Add(roundSeqUpdate.round, roundSeqUpdate.sequence)
		key := roundSequenceKeyFor(roundSeqUpdate.round)
		if err := rc.client.Set(ctx, key, arbmath.UintToBytes(roundSeqUpdate.sequence), rc.roundTimingInfo.Round*2).Err(); err != nil {
			log.Error("Error updating round's sequence count in redis", "key", key, "err", err) // this shouldn't be a problem if future msgs succeed in updating the count
		}
	}
}

func acceptedTxKeyFor(round, seqNum uint64) string {
	return fmt.Sprintf("%s%d.%d", EXPRESS_LANE_ACCEPTED_TX_KEY_PREFIX, round, seqNum)
}

func (rc *RedisCoordinator) GetAcceptedTxs(round, startSeqNum, endSeqNum uint64) []*ExpressLaneSubmission {
	ctx := rc.GetContext()
	fetchMsg := func(key string) *ExpressLaneSubmission {
		msgBytes, err := rc.client.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			log.Debug("ExpressLane tx not found in redis", "key", key)
			return nil
		}
		if err != nil {
			log.Warn("Error fetching accepted expressLane tx", "key", key, "err", err)
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
	select {
	case rc.msgChan <- msg:
	default:
		return errors.New("couldn't queue addition of expressLaneSubmission to redis")
	}
	return nil
}

func (rc *RedisCoordinator) trackAcceptedTxAddition(ctx context.Context) {
	for {
		var msg *ExpressLaneSubmission
		select {
		case msg = <-rc.msgChan:
			if msg.Round < rc.roundTimingInfo.RoundNumber() {
				// This prevents stale messages from being written to redis and unclogs msgChan
				continue
			}
		case <-ctx.Done():
			return
		}
		msgJson, err := msg.ToJson()
		if err != nil {
			log.Error("Failed to convert ExpressLaneSubmission to JsonExpressLaneSubmission", "err", err)
			continue
		}
		msgBytes, err := json.Marshal(msgJson)
		if err != nil {
			log.Error("Failed to marshal JsonExpressLaneSubmission", "err", err)
			continue
		}
		key := acceptedTxKeyFor(msg.Round, msg.SequenceNumber)
		if err := rc.client.Set(ctx, key, msgBytes, rc.roundTimingInfo.Round*2).Err(); err != nil {
			log.Error("Couldn't set key for accepted expressLane transaction in redis", "key", key, "err", err)
		}
	}
}
