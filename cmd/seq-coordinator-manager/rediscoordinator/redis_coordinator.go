package rediscoordinator

import (
	"context"
	"errors"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/util/redisutil"
)

type RedisCoordinator struct {
	Client redis.UniversalClient
}

func NewRedisCoordinator(redisURL string) (*RedisCoordinator, error) {
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		Client: redisClient,
	}, nil
}

func (rc *RedisCoordinator) GetPriorities(ctx context.Context) ([]string, map[string]int, error) {
	prioritiesMap := make(map[string]int)
	prioritiesString, err := rc.Client.Get(ctx, redisutil.PRIORITIES_KEY).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
		return []string{}, prioritiesMap, err
	}
	priorities := strings.Split(prioritiesString, ",")
	for _, url := range priorities {
		prioritiesMap[url]++
	}
	return priorities, prioritiesMap, nil
}

func (rc *RedisCoordinator) GetLivelinessMap(ctx context.Context) (map[string]int, error) {
	livelinessMap := make(map[string]int)
	livelinessList, _, err := rc.Client.Scan(ctx, 0, redisutil.WANTS_LOCKOUT_KEY_PREFIX+"*", 0).Result()
	if err != nil {
		return livelinessMap, err
	}
	for _, elem := range livelinessList {
		url := strings.TrimPrefix(elem, redisutil.WANTS_LOCKOUT_KEY_PREFIX)
		livelinessMap[url]++
	}
	return livelinessMap, nil
}

func (rc *RedisCoordinator) UpdatePriorities(ctx context.Context, priorities []string) error {
	prioritiesString := strings.Join(priorities, ",")
	err := rc.Client.Set(ctx, redisutil.PRIORITIES_KEY, prioritiesString, 0).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
	}
	return err
}

// CurrentChosenSequencer retrieves the current chosen sequencer holding the lock
func (c *RedisCoordinator) CurrentChosenSequencer(ctx context.Context) (string, error) {
	current, err := c.Client.Get(ctx, redisutil.CHOSENSEQ_KEY).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return current, nil
}
