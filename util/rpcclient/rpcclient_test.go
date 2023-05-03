package rpcclient

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestLogArgs(t *testing.T) {
	t.Parallel()

	str := logArgs(0, 1, 2, 3, "hello, world")
	if str != "[1, 2, 3, hello, world]" {
		Fail(t, "unexpected logs limit 0 got:", str)
	}

	str = logArgs(100, 1, 2, 3, "hello, world")
	if str != "[1, 2, 3, hello, world]" {
		Fail(t, "unexpected logs limit 100 got:", str)
	}

	str = logArgs(4, 1, 2, 3, "hello, world")
	if str != "[1, 2, 3, h...d]" {
		Fail(t, "unexpected logs limit 4 got:", str)
	}

}

func createTestNode(t *testing.T, ctx context.Context, stuckOrFailed int64) *node.Node {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{"test"}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	Require(t, err)

	testAPIs := []rpc.API{{
		Namespace:     "test",
		Version:       "1.0",
		Service:       &testAPI{stuckOrFailed, stuckOrFailed},
		Public:        true,
		Authenticated: false,
	}}
	stack.RegisterAPIs(testAPIs)

	err = stack.Start()
	Require(t, err)

	go func() {
		<-ctx.Done()
		stack.Close()
	}()

	return stack
}

type testAPI struct {
	stuckCalls  int64
	failedCalls int64
}

func (t *testAPI) StuckAtFirst(ctx context.Context) error {
	stuckRemaining := atomic.AddInt64(&t.stuckCalls, -1) + 1
	if stuckRemaining <= 0 {
		return nil
	}
	<-ctx.Done()
	return errors.New("error")
}

func (t *testAPI) FailAtFirst(ctx context.Context) error {
	failedRemaining := atomic.AddInt64(&t.failedCalls, -1) + 1
	if failedRemaining <= 0 {
		return nil
	}
	return errors.New("error")
}

func TestRpcClientRetry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	configFetcher := func() *ClientConfig {
		return &ClientConfig{
			URL:     "self",
			Timeout: time.Second * 5,
			Retries: 2,
		}
	}

	serverGood := createTestNode(t, ctx, 0)
	clientGood := NewRpcClient(configFetcher, serverGood)
	clientGood.Start(ctx)
	err := clientGood.CallContext(ctx, nil, "test_failAtFirst")
	Require(t, err)
	err = clientGood.CallContext(ctx, nil, "test_stuckAtFirst")
	Require(t, err)

	serverBad := createTestNode(t, ctx, 1000)
	clientBad := NewRpcClient(configFetcher, serverBad)
	clientBad.Start(ctx)
	err = clientBad.CallContext(ctx, nil, "test_failAtFirst")
	if err == nil {
		Fail(t, "no error for failAtFirst")
	}
	err = clientBad.CallContext(ctx, nil, "test_stuckAtFirst")
	if err == nil {
		Fail(t, "no error for stuckAtFirst")
	}

	serverRetry := createTestNode(t, ctx, 1)
	clientRetry := NewRpcClient(configFetcher, serverRetry)
	clientRetry.Start(ctx)
	err = clientRetry.CallContext(ctx, nil, "test_failAtFirst")
	if err == nil {
		Fail(t, "no error for failAtFirst")
	}
	err = clientRetry.CallContext(ctx, nil, "test_stuckAtFirst")
	Require(t, err)
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
