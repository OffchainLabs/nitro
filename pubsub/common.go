package pubsub

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"
)

func ResultKeyFor(streamName, id string) string { return fmt.Sprintf("%s.%s", streamName, id) }

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
	got, err := client.Do(ctx, "XINFO", "STREAM", streamName).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "no such key") {
			log.Error("redis error", "err", err, "searching stream", streamName)
		}
		return false
	}
	return got != nil
}
