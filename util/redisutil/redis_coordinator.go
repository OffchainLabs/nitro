package redisutil

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
)

const CHOSENSEQ_KEY string = "coordinator.chosen"                      // Never overwritten. Expires or released only
const MSG_COUNT_KEY string = "coordinator.msgCount"                    // Only written by sequencer holding CHOSEN key
const FINALIZED_MSG_COUNT_KEY string = "coordinator.finalizedMsgCount" // Only written by sequencer holding CHOSEN key
const PRIORITIES_KEY string = "coordinator.priorities"                 // Read only
const WANTS_LOCKOUT_KEY_PREFIX string = "coordinator.liveliness."      // Per server. Only written by self
const MESSAGE_KEY_PREFIX string = "coordinator.msg."                   // Per Message. Only written by sequencer holding CHOSEN
const SIGNATURE_KEY_PREFIX string = "coordinator.msg.sig."             // Per Message. Only written by sequencer holding CHOSEN
const BLOCKMETADATA_KEY_PREFIX string = "coordinator.blockMetadata."   // Per Message. Only written by sequencer holding CHOSEN
const WANTS_LOCKOUT_VAL string = "OK"
const SWITCHED_REDIS string = "SWITCHED_REDIS"
const INVALID_VAL string = "INVALID"
const INVALID_URL string = "<?INVALID-URL?>"

type RedisCoordinator struct {
	Client                                redis.UniversalClient
	firstSequencerWantingLockoutErrorTime atomic.Int64 // Time of the first error logged for no sequencer wanting the lockout.
	lastLockoutErrorLogTime               atomic.Int64 // Add this field to track when we last logged lockout errors.

	// If Client is a sentinel client,
	sentinelMaster string // The master name of the sentinel client.
	quorumSize     uint64 // Quorum size needed to qualify a redis GET as valid.
}

func WantsLockoutKeyFor(url string) string { return WANTS_LOCKOUT_KEY_PREFIX + url }

func NewRedisCoordinator(redisUrl string, quorumSize uint64) (*RedisCoordinator, error) {
	redisClient, sentinelMaster, err := RedisClientWithSentinelMasterNameFromURL(redisUrl)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		Client:         redisClient,
		sentinelMaster: sentinelMaster,
		quorumSize:     quorumSize,
	}, nil
}

// RecommendSequencerWantingLockout returns the top priority sequencer wanting the lockout
func (c *RedisCoordinator) RecommendSequencerWantingLockout(ctx context.Context) (string, error) {
	prioritiesString, err := c.Client.Get(ctx, PRIORITIES_KEY).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
		return "", err
	}
	priorities := strings.Split(prioritiesString, ",")
	for _, url := range priorities {
		err := c.Client.Get(ctx, WantsLockoutKeyFor(url)).Err()
		if errors.Is(err, redis.Nil) { // wants lockout not set
			continue
		}
		if err != nil {
			return "", err
		}
		// We found a sequencer that wants the lockout, so we reset the last time we observed the error
		// to a value of zero for logging purposes below.
		c.firstSequencerWantingLockoutErrorTime.Store(0)
		c.lastLockoutErrorLogTime.Store(0) // Reset log throttling timer when state changes.
		return url, nil
	}

	// If we hit this line, it means no sequencer is currently wanting the lockout from Redis.
	// A log will be emitted at different levels depending on how long it has been since the first error was logged.
	// At first, the log will be at the debug level, but if it persists for more than 10 seconds, it will be logged at the warn level.
	// If it persists for more than 20 seconds, it will be logged at the error level.
	logMessage := func(level func(msg string, ctx ...interface{})) {
		args := []interface{}{"priorities", prioritiesString}
		level("no sequencer appears to want the lockout on redis", args...)
	}

	if c.firstSequencerWantingLockoutErrorTime.Load() == 0 {
		now := time.Now().UnixMilli()
		c.firstSequencerWantingLockoutErrorTime.Store(now)
		c.lastLockoutErrorLogTime.Store(now)
		logMessage(log.Debug)
	} else {
		elapsedTime := time.Since(time.UnixMilli(c.firstSequencerWantingLockoutErrorTime.Load()))
		// Only log if it's been at least 5 seconds since the last log,
		// as these logs would otherwise be spammed at a high rate when they occur.
		if time.Since(time.UnixMilli(c.lastLockoutErrorLogTime.Load())) >= 5*time.Second {
			if elapsedTime > 20*time.Second {
				logMessage(log.Error)
			} else if elapsedTime > 10*time.Second {
				logMessage(log.Warn)
			} else {
				logMessage(log.Debug)
			}
			c.lastLockoutErrorLogTime.Store(time.Now().UnixMilli()) // Update last log time.
		}
	}
	return "", nil
}

// CurrentChosenSequencer retrieves the current chosen sequencer holding the lock
func (c *RedisCoordinator) CurrentChosenSequencer(ctx context.Context) (string, error) {
	current, err := c.Client.Get(ctx, CHOSENSEQ_KEY).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return current, nil
}

// GetPriorities returns the priority list of sequencers
func (rc *RedisCoordinator) GetPriorities(ctx context.Context) ([]string, error) {
	prioritiesString, err := rc.Client.Get(ctx, PRIORITIES_KEY).Result()
	if errors.Is(err, redis.Nil) {
		return []string{}, nil
	}
	if err != nil {
		return []string{}, err
	}
	prioritiesList := strings.Split(prioritiesString, ",")
	return prioritiesList, nil
}

// GetLiveliness returns a list of sequencers that have their liveliness set to OK
func (rc *RedisCoordinator) GetLiveliness(ctx context.Context) ([]string, error) {
	var livelinessList []string
	var cursor uint64
	for {
		var keySlice []string
		var err error
		keySlice, cursor, err = rc.Client.Scan(ctx, cursor, WANTS_LOCKOUT_KEY_PREFIX+"*", 0).Result()
		if err != nil {
			return []string{}, err
		}
		livelinessList = append(livelinessList, keySlice...)
		if cursor == 0 {
			break
		}
	}
	for i, elem := range livelinessList {
		url := strings.TrimPrefix(elem, WANTS_LOCKOUT_KEY_PREFIX)
		livelinessList[i] = url
	}
	return livelinessList, nil
}

// GetIfInQuorum acts as normal redis GET, but also error out if the key is not in the quorum of redis nodes
// if redis is a sentinel client.
func (rc *RedisCoordinator) GetIfInQuorum(ctx context.Context, key string) (string, error) {
	// If redis is not a sentinel client, or if the quorum size is less than 2, no need to check quorum
	if rc.sentinelMaster == "" || rc.quorumSize < 2 {
		return rc.Client.Get(ctx, key).Result()
	}

	// Get the master address and replicas from sentinel
	masterAddrCmd := redis.NewStringSliceCmd(ctx, "sentinel", "get-master-addr-by-name", rc.sentinelMaster)
	replicasCmd := redis.NewMapStringStringSliceCmd(ctx, "sentinel", "replicas", rc.sentinelMaster)

	pipe := rc.Client.Pipeline()
	err := pipe.Process(ctx, masterAddrCmd)
	if err != nil {
		return "", err
	}
	err = pipe.Process(ctx, replicasCmd)
	if err != nil {
		return "", err
	}
	// Get the key result as well, so as to avoid another separate call to redis
	getCmd := pipe.Get(ctx, key)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", err
	}
	masterAddr, err := masterAddrCmd.Result()
	if err != nil {
		return "", err
	}
	replicas, err := replicasCmd.Result()
	if err != nil {
		return "", err
	}
	result, err := getCmd.Result()
	if err != nil {
		return "", err
	}
	var urls []string
	urls = append(urls, masterAddr[0]+":"+masterAddr[1])
	for _, replica := range replicas {
		urls = append(urls, replica["ip"]+":"+replica["port"])
	}

	// Check if the key exists in the quorum
	numKeyOccurrences := atomic.Uint64{}
	wg := sync.WaitGroup{}
	for _, url := range urls {
		wg.Add(1)
		go func(redisUrl string) {
			defer wg.Done()
			r := redis.NewClient(&redis.Options{Addr: redisUrl})
			defer r.Close()
			exists, err := r.Exists(ctx, key).Result()
			if err != nil {
				log.Warn("Error checking redis key", "key", key, "err", err)
				return
			}
			if exists != 0 {
				numKeyOccurrences.Add(1)
			}
		}(url)
	}
	wg.Wait()
	if numKeyOccurrences.Load() < rc.quorumSize {
		return "", fmt.Errorf("redis key %s not in quorum, only %d redis nodes have it, wanted quorum size is %d", key, numKeyOccurrences.Load(), rc.quorumSize)
	}
	return result, nil
}

func MessageKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", MESSAGE_KEY_PREFIX, pos)
}

func MessageSigKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", SIGNATURE_KEY_PREFIX, pos)
}

func BlockMetadataKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", BLOCKMETADATA_KEY_PREFIX, pos)
}
