package redis

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTimeout(t *testing.T) {
	handler := testhelpers.InitTestLog(t, log.LevelInfo)
	ctx, cancel := context.WithCancel(context.Background())
	redisURL := redisutil.CreateTestRedis(ctx, t)
	TestValidationServerConfig.RedisURL = redisURL
	TestValidationServerConfig.ModuleRoots = []string{"0x123"}
	TestValidationServerConfig.StreamTimeout = 100 * time.Millisecond
	vs, err := NewValidationServer(&TestValidationServerConfig, nil)
	if err != nil {
		t.Fatalf("NewValidationSever() unexpected error: %v", err)
	}
	vs.Start(ctx)
	time.Sleep(time.Second)
	if !handler.WasLogged("Waiting for redis streams timed out") {
		t.Error("Expected message about stream time-outs was not logged")
	}
	cancel()
}
