package pubsub

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"
)

// Make sure to never start the key with stream name, as we use patterns like "--check-streams=streamname-*"
// to scrape stream with redis export.
// https://github.com/oliver006/redis_exporter/blob/71dbe37fb14a4ae2537c1790a239dc1e568ffba5/main.go#L68
func ResultKeyFor(streamName, id string) string {
	return fmt.Sprintf("result-key:%s.%s", streamName, id)
}
func ErrorKeyFor(streamName, id string) string {
	return fmt.Sprintf("error-key:%s.%s.error", streamName, id)
}

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
