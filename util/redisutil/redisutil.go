package redisutil

import "github.com/go-redis/redis/v8"

func RedisClientFromURL(url string) (*redis.Client, error) {
	if url == "" {
		return nil, nil
	}
	redisOptions, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(redisOptions), nil
}
