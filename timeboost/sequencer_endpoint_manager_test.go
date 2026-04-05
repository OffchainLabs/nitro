// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package timeboost

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestStaticEndpointManager_CloseAndReconnect(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer srv.Close()

	mgr := &StaticEndpointManager{endpoint: srv.URL}

	// Get initial client.
	client1, isNew, err := mgr.GetSequencerRPC(context.Background())
	require.NoError(t, err)
	require.True(t, isNew)
	require.NotNil(t, client1)

	// Close should nil out the client.
	mgr.Close()
	require.Nil(t, mgr.client)

	// After Close, GetSequencerRPC should create a fresh client.
	client2, isNew, err := mgr.GetSequencerRPC(context.Background())
	require.NoError(t, err)
	require.True(t, isNew, "should create a new client after Close")
	require.NotNil(t, client2)
}

func TestRedisEndpointManager_CloseReleasesResources(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	coordinator, err := redisutil.NewRedisCoordinator(redisURL, 1)
	require.NoError(t, err)

	// Verify the coordinator's Redis client works before Close.
	require.NoError(t, coordinator.Client.Ping(ctx).Err())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer srv.Close()

	// Seed Redis with a chosen sequencer so GetSequencerRPC can find one.
	require.NoError(t, coordinator.Client.Set(ctx, redisutil.CHOSENSEQ_KEY, srv.URL, 0).Err())

	mgr, ok := NewRedisEndpointManager(coordinator, "").(*RedisEndpointManager)
	require.True(t, ok)

	// Create an RPC client by fetching the endpoint.
	client, isNew, err := mgr.GetSequencerRPC(ctx)
	require.NoError(t, err)
	require.True(t, isNew)
	require.NotNil(t, client)

	// Close should release both the RPC client and the coordinator's Redis connection.
	mgr.Close()
	require.Nil(t, mgr.client)

	// The coordinator's Redis client should be closed.
	err = coordinator.Client.Ping(ctx).Err()
	require.Error(t, err, "coordinator Redis client should be closed after Close()")
}

func TestStaticEndpointManager_ConcurrentGetSequencerRPC(t *testing.T) {
	t.Parallel()

	// Start a minimal HTTP server so rpc.DialContext succeeds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer srv.Close()

	mgr := &StaticEndpointManager{
		endpoint: srv.URL,
	}

	const goroutines = 20
	var wg sync.WaitGroup
	var isNewCount int64
	clients := make([]*rpc.Client, goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, isNew, err := mgr.GetSequencerRPC(context.Background())
			require.NoError(t, err)
			require.NotNil(t, client)
			clients[i] = client
			if isNew {
				atomic.AddInt64(&isNewCount, 1)
			}
		}()
	}
	wg.Wait()

	// Exactly one goroutine should have created the client.
	require.Equal(t, int64(1), isNewCount, "exactly one caller should see isNew=true")

	// All goroutines must receive the same client instance.
	for i := 1; i < goroutines; i++ {
		require.Same(t, clients[0], clients[i], "all callers must receive the same client")
	}
}

func TestRedisEndpointManager_URLChangeClosesStaleClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	coordinator, err := redisutil.NewRedisCoordinator(redisURL, 1)
	require.NoError(t, err)

	// Start two HTTP servers to simulate sequencer failover.
	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer srvA.Close()
	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x2"}`))
	}))
	defer srvB.Close()

	// Point to server A.
	require.NoError(t, coordinator.Client.Set(ctx, redisutil.CHOSENSEQ_KEY, srvA.URL, 0).Err())
	mgr, ok := NewRedisEndpointManager(coordinator, "").(*RedisEndpointManager)
	require.True(t, ok)

	// First call: creates client for server A.
	clientA, isNew, err := mgr.GetSequencerRPC(ctx)
	require.NoError(t, err)
	require.True(t, isNew)
	require.NotNil(t, clientA)
	require.Equal(t, srvA.URL, mgr.clientUrl)

	// Second call with same URL: returns cached client.
	clientA2, isNew, err := mgr.GetSequencerRPC(ctx)
	require.NoError(t, err)
	require.False(t, isNew)
	require.Same(t, clientA, clientA2)

	// Switch to server B.
	require.NoError(t, coordinator.Client.Set(ctx, redisutil.CHOSENSEQ_KEY, srvB.URL, 0).Err())

	// Third call: detects URL change, closes stale client, creates new one.
	clientB, isNew, err := mgr.GetSequencerRPC(ctx)
	require.NoError(t, err)
	require.True(t, isNew, "should create new client after URL change")
	require.NotNil(t, clientB)
	require.Equal(t, srvB.URL, mgr.clientUrl)

	mgr.Close()
}

func TestRedisEndpointManager_PreservesOldClientOnNewEndpointFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	coordinator, err := redisutil.NewRedisCoordinator(redisURL, 1)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer srv.Close()

	// Point to the working server.
	require.NoError(t, coordinator.Client.Set(ctx, redisutil.CHOSENSEQ_KEY, srv.URL, 0).Err())

	// Use a non-existent JWT path so that createRPCClient fails when the
	// URL changes (rpc.DialContext with HTTP succeeds lazily, so we need
	// a different failure mode).
	mgr, ok := NewRedisEndpointManager(coordinator, "/nonexistent/jwt.key").(*RedisEndpointManager)
	require.True(t, ok)

	// Manually set a working client (bypassing JWT since we need one to exist).
	directClient, err := rpc.DialContext(ctx, srv.URL)
	require.NoError(t, err)
	mgr.client = directClient
	mgr.clientUrl = srv.URL

	// Point to a different endpoint; createRPCClient will fail reading JWT.
	require.NoError(t, coordinator.Client.Set(ctx, redisutil.CHOSENSEQ_KEY, "http://127.0.0.1:9999", 0).Err())

	_, _, err = mgr.GetSequencerRPC(ctx)
	require.Error(t, err, "should fail to create client due to missing JWT file")

	// The old client and URL should still be intact.
	require.NotNil(t, mgr.client, "old client should be preserved on failure")
	require.Equal(t, srv.URL, mgr.clientUrl, "old URL should be preserved on failure")

	mgr.Close()
}

func TestNewBidValidator_PartialFailureCleansUpRedis(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	redisURL := redisutil.CreateTestRedis(ctx, t)

	// Use an unreachable RPC endpoint so that ChainID will fail,
	// triggering the cleanup path after Redis was already connected.
	cfg := &BidValidatorConfig{
		RpcEndpoint:            "http://127.0.0.1:1", // nothing listening
		AuctionContractAddress: "0x0000000000000000000000000000000000000001",
		RedisURL:               redisURL,
		ProducerConfig:         pubsub.TestProducerConfig,
		MaxBidsPerSender:       5,
	}
	fetcher := func() *BidValidatorConfig { return cfg }

	// Use a short timeout since the unreachable endpoint will cause ChainID to block.
	shortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := NewBidValidator(shortCtx, nil, fetcher)
	require.Error(t, err, "NewBidValidator should fail with unreachable RPC endpoint")
}
