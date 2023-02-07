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

const CHOSENSEQ_KEY string = "coordinator.chosen"              // Never overwritten. Expires or released only
const MSG_COUNT_KEY string = "coordinator.msgCount"            // Only written by sequencer holding CHOSEN key
const PRIORITIES_KEY string = "coordinator.priorities"         // Read only
const LIVELINESS_KEY_PREFIX string = "coordinator.liveliness." // Per server. Only written by self
const MESSAGE_KEY_PREFIX string = "coordinator.msg."           // Per Message. Only written by sequencer holding CHOSEN
const SIGNATURE_KEY_PREFIX string = "coordinator.msg.sig."     // Per Message. Only written by sequencer holding CHOSEN
const LIVELINESS_VAL string = "OK"
const INVALID_VAL string = "INVALID"
const INVALID_URL string = "<?INVALID-URL?>"

type RedisCoordinator struct {
	Client redis.UniversalClient
}

func LivelinessKeyFor(url string) string { return LIVELINESS_KEY_PREFIX + url }

func NewRedisCoordinator(redisUrl string) (*RedisCoordinator, error) {
	redisClient, err := RedisClientFromURL(redisUrl)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		Client: redisClient,
	}, nil
}

// RecommendLiveSequencer returns the top priority live sequencer
func (c *RedisCoordinator) RecommendLiveSequencer(ctx context.Context) (string, error) {
	return c.RecommendLiveSequencerIgnoring(ctx, "")
}

func (c *RedisCoordinator) RecommendLiveSequencerIgnoring(ctx context.Context, ignore string) (string, error) {
	prioritiesString, err := c.Client.Get(ctx, PRIORITIES_KEY).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
		return "", err
	}
	priorities := strings.Split(prioritiesString, ",")
	foundIgnored := false
	for _, url := range priorities {
		err := c.Client.Get(ctx, LivelinessKeyFor(url)).Err()
		if errors.Is(err, redis.Nil) { // liveliness not set
			continue
		}
		if err != nil {
			return "", err
		}
		if url == ignore {
			foundIgnored = true
			continue
		}
		return url, nil
	}
	if ignore != "" && foundIgnored {
		log.Warn("no other sequencer appears live on redis", "priorities", prioritiesString, "ignored", ignore)
	} else {
		log.Error("no sequencer appears live on redis", "priorities", prioritiesString)
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
	err = c.Client.Get(ctx, LivelinessKeyFor(current)).Err()
	if errors.Is(err, redis.Nil) {
		return "", nil // lock owner but not alive
	}
	if err != nil {
		return "", err
	}
	return current, nil
}

func MessageKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", MESSAGE_KEY_PREFIX, pos)
}

func MessageSigKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", SIGNATURE_KEY_PREFIX, pos)
}
