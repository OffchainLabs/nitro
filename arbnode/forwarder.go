// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type ForwarderConfig struct {
	ConnectionTimeout     time.Duration `koanf:"connection-timeout"`
	IdleConnectionTimeout time.Duration `koanf:"idle-connection-timeout"`
	MaxIdleConnections    int           `koanf:"max-idle-connections"`
}

var DefaultTestForwarderConfig = ForwarderConfig{
	ConnectionTimeout:     2 * time.Second,
	IdleConnectionTimeout: 2 * time.Second,
	MaxIdleConnections:    1,
}

var DefaultNodeForwarderConfig = ForwarderConfig{
	ConnectionTimeout:     30 * time.Second,
	IdleConnectionTimeout: 15 * time.Second,
	MaxIdleConnections:    1,
}

var DefaultSequencerForwarderConfig = ForwarderConfig{
	ConnectionTimeout:     30 * time.Second,
	IdleConnectionTimeout: 60 * time.Second,
	MaxIdleConnections:    100,
}

func AddOptionsForNodeForwarderConfig(prefix string, f *flag.FlagSet) {
	AddOptionsForForwarderConfigImpl(prefix, &DefaultNodeForwarderConfig, f)
}

func AddOptionsForSequencerForwarderConfig(prefix string, f *flag.FlagSet) {
	AddOptionsForForwarderConfigImpl(prefix, &DefaultSequencerForwarderConfig, f)
}

func AddOptionsForForwarderConfigImpl(prefix string, defaultConfig *ForwarderConfig, f *flag.FlagSet) {
	f.Duration(prefix+".connection-timeout", defaultConfig.ConnectionTimeout, "total time to wait before cancelling connection")
	f.Duration(prefix+".idle-connection-timeout", defaultConfig.IdleConnectionTimeout, "time until idle connections are closed")
	f.Int(prefix+".max-idle-connections", defaultConfig.MaxIdleConnections, "maximum number of idle connections to keep open")
}

type TxForwarder struct {
	enabled   int32
	target    string
	timeout   time.Duration
	transport *http.Transport
	rpcClient *rpc.Client
	ethClient *ethclient.Client

	healthMutex   sync.Mutex
	healthErr     error
	healthChecked time.Time
}

func NewForwarder(target string, config *ForwarderConfig) *TxForwarder {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 2 * time.Second,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConnections,
		MaxIdleConnsPerHost:   config.MaxIdleConnections,
		IdleConnTimeout:       config.IdleConnectionTimeout,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &TxForwarder{
		target:    target,
		timeout:   config.ConnectionTimeout,
		transport: transport,
	}
}

func (f *TxForwarder) ctxWithTimeout(inctx context.Context) (context.Context, context.CancelFunc) {
	if f.timeout == time.Duration(0) {
		return context.WithCancel(inctx)
	}
	return context.WithTimeout(inctx, f.timeout)
}

func (f *TxForwarder) PublishTransaction(inctx context.Context, tx *types.Transaction) error {
	if atomic.LoadInt32(&f.enabled) == 0 {
		return ErrNoSequencer
	}
	ctx, cancelFunc := f.ctxWithTimeout(inctx)
	defer cancelFunc()
	return f.ethClient.SendTransaction(ctx, tx)
}

const cacheUpstreamHealth = 2 * time.Second
const maxHealthTimeout = 10 * time.Second

func (f *TxForwarder) CheckHealth(inctx context.Context) error {
	if atomic.LoadInt32(&f.enabled) == 0 {
		return ErrNoSequencer
	}
	f.healthMutex.Lock()
	defer f.healthMutex.Unlock()
	if time.Since(f.healthChecked) > cacheUpstreamHealth {
		timeout := f.timeout
		if timeout == time.Duration(0) || timeout >= maxHealthTimeout {
			timeout = maxHealthTimeout
		}
		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()
		f.healthErr = f.rpcClient.CallContext(ctx, nil, "arb_checkPublisherHealth")
		f.healthChecked = time.Now()
	}
	return f.healthErr
}

func (f *TxForwarder) Initialize(inctx context.Context) error {
	if f.target == "" {
		f.rpcClient = nil
		f.ethClient = nil
		f.enabled = 0
		return nil
	}
	ctx, cancelFunc := f.ctxWithTimeout(inctx)
	defer cancelFunc()
	rpcClient, err := rpc.DialTransport(ctx, f.target, f.transport)
	if err != nil {
		return err
	}
	f.rpcClient = rpcClient
	f.ethClient = ethclient.NewClient(rpcClient)
	f.enabled = 1
	return nil
}

// Disable is not thread-safe vs. Initialize
func (f *TxForwarder) Disable() {
	atomic.StoreInt32(&f.enabled, 0)
	// TODO should we close ethClient / rpcClient here or in StopAndWait?
}

func (f *TxForwarder) Start(ctx context.Context) error {
	return nil
}

func (f *TxForwarder) StopAndWait() {}

func (f *TxForwarder) Started() bool {
	return true
}

type TxDropper struct{}

func NewTxDropper() *TxDropper {
	return &TxDropper{}
}

var txDropperErr = errors.New("publishing transactions not supported by this endpoint")

func (f *TxDropper) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	return txDropperErr
}

func (f *TxDropper) CheckHealth(ctx context.Context) error {
	return txDropperErr
}

func (f *TxDropper) Initialize(ctx context.Context) error { return nil }

func (f *TxDropper) Start(ctx context.Context) error { return nil }

func (f *TxDropper) StopAndWait() {}

func (f *TxDropper) Started() bool {
	return true
}

type RedisTxForwarder struct {
	stopwaiter.StopWaiterSafe

	forwarderConfig *ForwarderConfig
	redisUrl        string
	updateInterval  time.Duration
	retryInterval   time.Duration
	redisErrors     int

	forwarder atomic.Pointer[TxForwarder]
	// not thread safe fields used in .Initialize and/or .update methods
	redisCoordinator *redisutil.RedisCoordinator
	currentTarget    string
}

func NewRedisTxForwarder(redisUrl string, updateInterval time.Duration, retryInterval time.Duration, config *ForwarderConfig) *RedisTxForwarder {
	return &RedisTxForwarder{
		forwarderConfig: config,
		redisUrl:        redisUrl,
		updateInterval:  updateInterval,
		retryInterval:   retryInterval,
	}
}

func (f *RedisTxForwarder) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	forwarder := f.forwarder.Load()
	if forwarder == nil {
		return ErrNoSequencer
	}
	return forwarder.PublishTransaction(ctx, tx)
}

func (f *RedisTxForwarder) CheckHealth(ctx context.Context) error {
	forwarder := f.forwarder.Load()
	if forwarder == nil {
		return ErrNoSequencer
	}
	return forwarder.CheckHealth(ctx)
}

// not thread safe vs update and itself
func (f *RedisTxForwarder) Initialize(ctx context.Context) error {
	var err error
	f.redisCoordinator, err = redisutil.NewRedisCoordinator(f.redisUrl)
	if err != nil {
		return errors.Wrap(err, "unable to create redis coordinator")
	}
	return nil
}

func (f *RedisTxForwarder) retryAfterRedisError() time.Duration {
	f.redisErrors++
	retryIn := f.retryInterval * time.Duration(f.redisErrors)
	if retryIn > f.updateInterval {
		retryIn = f.updateInterval
	}
	return retryIn
}

func (f *RedisTxForwarder) noRedisError() time.Duration {
	f.redisErrors = 0
	return f.updateInterval
}

// not thread safe vs initialize and itself
func (f *RedisTxForwarder) update(ctx context.Context) time.Duration {
	newSequencerUrl, err := f.redisCoordinator.RecommendLiveSequencer(ctx)
	if err != nil {
		log.Warn("coordinator failed to find live sequencer", "err", err)
		return f.retryAfterRedisError()
	}
	if newSequencerUrl == f.currentTarget {
		return f.noRedisError()
	}
	newForwarder := NewForwarder(newSequencerUrl, f.forwarderConfig)
	err = newForwarder.Initialize(ctx)
	if err != nil {
		log.Error("failed to initialize forward agent", "err", err)
		return f.noRedisError()
	}
	f.forwarder.Load().Disable()
	// TODO should we stop the old forwarder to close old rpc connection?
	// oldForwarder.StopAndWait()
	f.currentTarget = newSequencerUrl
	f.forwarder.Store(newForwarder)
	return f.noRedisError()
}

func (f *RedisTxForwarder) Start(ctx context.Context) error {
	if err := f.StopWaiterSafe.Start(ctx, f); err != nil {
		return err
	}
	if err := f.CallIteratively(f.update); err != nil {
		return errors.Wrap(err, "failed to start forwarder update thread")
	}
	return nil
}

func (f *RedisTxForwarder) StopAndWait() {
	err := f.StopWaiterSafe.StopAndWait()
	if err != nil {
		log.Error("Failed to stop forwarder", "err", err)
	}
	// oldForwarder.StopAndWait() // TODO should we stop the forwarder?
}

func (f *RedisTxForwarder) Started() bool {
	return f.StopWaiterSafe.Started()
}
