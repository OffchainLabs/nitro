package pubsub

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
)

// CreateStream tries to create stream with given name, if it already exists
// does not return an error.
func CreateStream(ctx context.Context, streamName string, client redis.UniversalClient) error {
	_, err := client.XGroupCreateMkStream(ctx, streamName, streamName, "$").Result()
	if err != nil && !StreamExists(ctx, streamName, client) {
		return err
	}
	return nil
}

// StreamExists returns whether there are any consumer group for specified
// redis stream.
func StreamExists(ctx context.Context, streamName string, client redis.UniversalClient) bool {
	groups, err := client.XInfoStream(ctx, streamName).Result()
	if err != nil {
		log.Error("Reading redis streams", "error", err)
		return false
	}
	return groups.Groups > 0
}
