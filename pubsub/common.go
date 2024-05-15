package pubsub

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// CreateStream tries to create stream with given name, if it already exists
// does not return an error.
func CreateStream(ctx context.Context, streamName string, client redis.UniversalClient) error {
	_, err := client.XGroupCreateMkStream(ctx, streamName, streamName, "$").Result()
	if err == nil || err.Error() == "BUSYGROUP Consumer Group name already exists" {
		return nil
	}
	return err
}
