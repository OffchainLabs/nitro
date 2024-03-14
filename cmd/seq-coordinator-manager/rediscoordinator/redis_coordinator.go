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

// UpdatePriorities updates the priority list of sequencers
func (rc *RedisCoordinator) UpdatePriorities(ctx context.Context, priorities []string) error {
	if len(priorities) == 0 {
		return rc.Client.Del(ctx, redisutil.PRIORITIES_KEY).Err()
	}
	prioritiesString := strings.Join(priorities, ",")
	err := rc.Client.Set(ctx, redisutil.PRIORITIES_KEY, prioritiesString, 0).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
		return err
	}
	return nil
}
