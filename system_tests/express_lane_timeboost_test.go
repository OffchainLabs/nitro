package arbtest

import (
	"context"
	"testing"

	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestBidValidatorAuctioneerRedisStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	redisURL := redisutil.CreateTestRedis(ctx, t)
	_ = redisURL
}
