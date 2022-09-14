package arbnode

import (
	"context"
	"errors"
	"strings"

	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/redisutil"
)

type RedisCoordinator struct {
	client redis.UniversalClient
	myUrl  string
}

func NewRedisCoordinator(redisUrl string, myUrl string) (*RedisCoordinator, error) {
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		client: redisClient,
		myUrl:  myUrl,
	}, nil
}

func (c *RedisCoordinator) recommendLiveSequencer(ctx context.Context) (string, error) {
	prioritiesString, err := c.client.Get(ctx, PRIORITIES_KEY).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
		return "", err
	}
	priorities := strings.Split(prioritiesString, ",")
	for _, url := range priorities {
		err := c.client.Get(ctx, livelinessKeyFor(url)).Err()
		if errors.Is(err, redis.Nil) { // liveliness not set
			continue
		}
		if err != nil {
			return "", err
		}
		return url, nil
	}
	log.Info("no sequencer appears live on redis", "priorities", prioritiesString, "self", c.myUrl)
	return "", nil
}
