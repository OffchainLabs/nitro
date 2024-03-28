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

	args := []any{1, 2, 3, "hello, world"}
	str := limitedArgumentsMarshal{0, args}.String()
	if str != "[1, 2, 3, \"hello, world\"]" {
		Fail(t, "unexpected logs limit 0 got:", str)
	}

	str = limitedArgumentsMarshal{100, args}.String()
	if str != "[1, 2, 3, \"hello, world\"]" {
		Fail(t, "unexpected logs limit 100 got:", str)
	}

	str = limitedArgumentsMarshal{6, args}.String()
	if str != "[1, 2, 3, \"h..d\"]" {
		Fail(t, "unexpected logs limit 6 got:", str)
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

func (t *testAPI) Delay(ctx context.Context, msec int64) error {
	<-time.After(time.Millisecond * time.Duration(msec))
	return nil
}

func TestRpcClientRetry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	config := &ClientConfig{
		URL:         "self",
		Timeout:     time.Second * 5,
		Retries:     2,
		RetryErrors: "",
	}
	Require(t, config.Validate())
	configFetcher := func() *ClientConfig { return config }

	serverGood := createTestNode(t, ctx, 0)
	clientGood := NewRpcClient(configFetcher, serverGood)
	err := clientGood.Start(ctx)
	Require(t, err)
	err = clientGood.CallContext(ctx, nil, "test_failAtFirst")
	Require(t, err)
	err = clientGood.CallContext(ctx, nil, "test_stuckAtFirst")
	Require(t, err)

	serverBad := createTestNode(t, ctx, 1000)
	clientBad := NewRpcClient(configFetcher, serverBad)
	err = clientBad.Start(ctx)
	Require(t, err)
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
	err = clientRetry.Start(ctx)
	Require(t, err)
	err = clientRetry.CallContext(ctx, nil, "test_failAtFirst")
	if err == nil {
		Fail(t, "no error for failAtFirst")
	}
	err = clientRetry.CallContext(ctx, nil, "test_stuckAtFirst")
	Require(t, err)

	retryConfig := &ClientConfig{
		URL:         "self",
		Timeout:     time.Second * 5,
		Retries:     2,
		RetryErrors: "er.*",
	}
	Require(t, retryConfig.Validate())
	retryErrConfigFetcher := func() *ClientConfig { return retryConfig }

	serverWorkWithRetry := createTestNode(t, ctx, 1)
	clientWorkWithRetry := NewRpcClient(retryErrConfigFetcher, serverWorkWithRetry)
	err = clientWorkWithRetry.Start(ctx)
	Require(t, err)
	err = clientWorkWithRetry.CallContext(ctx, nil, "test_failAtFirst")
	Require(t, err)

	clientFailsWithRetry := NewRpcClient(retryErrConfigFetcher, serverBad)
	err = clientFailsWithRetry.Start(ctx)
	Require(t, err)
	err = clientFailsWithRetry.CallContext(ctx, nil, "test_failAtFirst")
	if err == nil {
		Fail(t, "no error for failAtFirst")
	}

	noMatchconfig := &ClientConfig{
		URL:         "self",
		Timeout:     time.Second * 5,
		Retries:     2,
		RetryErrors: "b.*",
	}
	Require(t, config.Validate())
	noMatchFetcher := func() *ClientConfig { return noMatchconfig }
	serverWorkWithRetry2 := createTestNode(t, ctx, 1)
	clientNoMatch := NewRpcClient(noMatchFetcher, serverWorkWithRetry2)
	err = clientNoMatch.Start(ctx)
	Require(t, err)
	err = clientNoMatch.CallContext(ctx, nil, "test_failAtFirst")
	if err == nil {
		Fail(t, "no error for failAtFirst")
	}
}

func TestIsAlreadyKnownError(t *testing.T) {
	for _, testCase := range []struct {
		input    string
		expected bool
	}{
		{"already known", true},
		{"insufficient balance", false},
		{"foo already known\nbar", true},
		{"replacement transaction underpriced: new tx gas fee cap 3824396284 \u003c= 3824396284 queued", true},
		{"replacement transaction underpriced: new tx gas fee cap 1234 \u003c= 5678 queued", false},
		{"foo replacement transaction underpriced: new tx gas fee cap 3824396284 \u003c= 3824396284 queued bar", true},
	} {
		got := IsAlreadyKnownError(errors.New(testCase.input))
		if got != testCase.expected {
			t.Errorf("IsAlreadyKnownError(%q) = %v expected %v", testCase.input, got, testCase.expected)
		}
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
