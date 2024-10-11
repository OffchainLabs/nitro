package redisutil

import "github.com/redis/go-redis/v9"

func RedisClientFromURL(url string) (redis.UniversalClient, error) {
	if url == "" {
		return nil, nil
	}
	redisOptions, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(redisOptions), nil
}
