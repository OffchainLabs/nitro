package rediscoordinator

import (
	"context"
	"errors"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/util/redisutil"
)

// RedisCoordinator builds upon RedisCoordinator of redisutil with additional functionality
type RedisCoordinator struct {
	*redisutil.RedisCoordinator
}

// GetPriorities returns the priority list of sequencers
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

// GetLivelinessMap returns a map whose keys are sequencers that have their liveliness set to OK
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

// UpdatePriorities updates the priority list of sequencers
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
