package redisutil

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
)

const CHOSENSEQ_KEY string = "coordinator.chosen"                 // Never overwritten. Expires or released only
const MSG_COUNT_KEY string = "coordinator.msgCount"               // Only written by sequencer holding CHOSEN key
const PRIORITIES_KEY string = "coordinator.priorities"            // Read only
const WANTS_LOCKOUT_KEY_PREFIX string = "coordinator.liveliness." // Per server. Only written by self
const MESSAGE_KEY_PREFIX string = "coordinator.msg."              // Per Message. Only written by sequencer holding CHOSEN
const SIGNATURE_KEY_PREFIX string = "coordinator.msg.sig."        // Per Message. Only written by sequencer holding CHOSEN
const WANTS_LOCKOUT_VAL string = "OK"
const INVALID_VAL string = "INVALID"
const INVALID_URL string = "<?INVALID-URL?>"

type RedisCoordinator struct {
	Client redis.UniversalClient
}

func WantsLockoutKeyFor(url string) string { return WANTS_LOCKOUT_KEY_PREFIX + url }

func NewRedisCoordinator(redisUrl string) (*RedisCoordinator, error) {
	redisClient, err := RedisClientFromURL(redisUrl)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		Client: redisClient,
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
		return url, nil
	}
	log.Error("no sequencer appears to want the lockout on redis", "priorities", prioritiesString)
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
	cursor := uint64(0)
	for {
		keySlice, cursor, err := rc.Client.Scan(ctx, cursor, WANTS_LOCKOUT_KEY_PREFIX+"*", 0).Result()
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

func MessageKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", MESSAGE_KEY_PREFIX, pos)
}

func MessageSigKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", SIGNATURE_KEY_PREFIX, pos)
}
