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
	RedisUrl              string        `koanf:"redis-url"`
	UpdateInterval        time.Duration `koanf:"update-interval"`
	RetryInterval         time.Duration `koanf:"retry-interval"`
}

var DefaultTestForwarderConfig = ForwarderConfig{
	ConnectionTimeout:     2 * time.Second,
	IdleConnectionTimeout: 2 * time.Second,
	MaxIdleConnections:    1,
	RedisUrl:              redisutil.DefaultTestRedisURL,
	UpdateInterval:        time.Millisecond * 10,
	RetryInterval:         time.Millisecond * 3,
}

var DefaultNodeForwarderConfig = ForwarderConfig{
	ConnectionTimeout:     30 * time.Second,
	IdleConnectionTimeout: 15 * time.Second,
	MaxIdleConnections:    1,
	RedisUrl:              "",
	UpdateInterval:        time.Second,
	RetryInterval:         100 * time.Millisecond,
}

var DefaultSequencerForwarderConfig = ForwarderConfig{
	ConnectionTimeout:     30 * time.Second,
	IdleConnectionTimeout: 60 * time.Second,
	MaxIdleConnections:    100,
	RedisUrl:              "",
	UpdateInterval:        time.Second,
	RetryInterval:         100 * time.Millisecond,
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
	f.String(prefix+".redis-url", defaultConfig.RedisUrl, "the Redis URL to recomend target via")
	f.Duration(prefix+".update-interval", defaultConfig.UpdateInterval, "forwarding target update interval")
	f.Duration(prefix+".retry-interval", defaultConfig.RetryInterval, "minimal time between update retries")
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
}

func (f *TxForwarder) Start(ctx context.Context) error {
	return nil
}

func (f *TxForwarder) StopAndWait() {
	f.ethClient.Close() // internally closes also the rpc client
}

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

	config         *ForwarderConfig
	fallbackTarget string

	errors           int
	currentTarget    string
	redisCoordinator *redisutil.RedisCoordinator

	mtx       sync.RWMutex
	forwarder *TxForwarder
}

func NewRedisTxForwarder(fallbackTarget string, config *ForwarderConfig) *RedisTxForwarder {
	return &RedisTxForwarder{
		config:         config,
		fallbackTarget: fallbackTarget,
	}
}

func (f *RedisTxForwarder) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	forwarder := f.getForwarder()
	if forwarder == nil {
		return ErrNoSequencer
	}
	return forwarder.PublishTransaction(ctx, tx)
}

func (f *RedisTxForwarder) CheckHealth(ctx context.Context) error {
	forwarder := f.getForwarder()
	if forwarder == nil {
		return ErrNoSequencer
	}
	return forwarder.CheckHealth(ctx)
}

// not thread safe vs update and itself
func (f *RedisTxForwarder) Initialize(ctx context.Context) error {
	var err error
	f.redisCoordinator, err = redisutil.NewRedisCoordinator(f.config.RedisUrl)
	if err != nil {
		return errors.Wrap(err, "unable to create redis coordinator")
	}
	f.update(ctx)
	return nil
}

func (f *RedisTxForwarder) retryAfterError() time.Duration {
	f.errors++
	retryIn := f.config.RetryInterval * time.Duration(f.errors)
	if retryIn > f.config.UpdateInterval {
		retryIn = f.config.UpdateInterval
	}
	return retryIn
}

// returns true when retry interval is saturated and there is a fallback url available
func (f *RedisTxForwarder) shouldFallbackToStatic() bool {
	return f.currentTarget == "" ||
		f.config.RetryInterval*time.Duration(f.errors+1) >= f.config.UpdateInterval && f.fallbackTarget != "" && f.fallbackTarget != f.currentTarget
}

func (f *RedisTxForwarder) noError() time.Duration {
	f.errors = 0
	return f.config.UpdateInterval
}

func (f *RedisTxForwarder) getForwarder() *TxForwarder {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.forwarder
}

func (f *RedisTxForwarder) setForwarder(forwarder *TxForwarder) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	if f.forwarder != nil {
		f.forwarder.Disable()
	}
	f.forwarder = forwarder
}

// not thread safe vs initialize and itself
func (f *RedisTxForwarder) update(ctx context.Context) time.Duration {
	nextUpdateIn := f.noError
	var newSequencerUrl string
	var redisErr error
	if f.redisCoordinator != nil {
		newSequencerUrl, redisErr = f.redisCoordinator.CurrentChosenSequencer(ctx)
		if redisErr == nil && newSequencerUrl == "" {
			log.Info("no sequencer is currently chosen, using recommended sequencer instead")
			newSequencerUrl, redisErr = f.redisCoordinator.RecommendSequencerWantingLockout(ctx)
		}
		if redisErr != nil || newSequencerUrl == "" {
			if f.shouldFallbackToStatic() && f.fallbackTarget != "" {
				log.Warn("coordinator failed to find live sequencer, falling back to static url", "err", redisErr, "fallback", f.fallbackTarget)
				newSequencerUrl = f.fallbackTarget
				nextUpdateIn = f.retryAfterError
			} else {
				log.Warn("coordinator failed to find live sequencer", "err", redisErr)
				return f.retryAfterError()
			}
		}
	} else {
		if f.fallbackTarget != "" {
			log.Warn("redis coordinator not initialized, falling back to static url", "fallback", f.fallbackTarget)
			newSequencerUrl = f.fallbackTarget
		} else {
			// TODO panic? - there is no way to recover from this point
			log.Error("redis coordinator not initilized, no fallback available")
			return f.retryAfterError()
		}
	}
	if newSequencerUrl == f.currentTarget {
		return nextUpdateIn()
	}
	var newForwarder *TxForwarder
	for {
		newForwarder = NewForwarder(newSequencerUrl, f.config)
		err := newForwarder.Initialize(ctx)
		if err == nil {
			break
		}
		if f.shouldFallbackToStatic() && newSequencerUrl != f.fallbackTarget {
			log.Error("failed to initialize forward agent, falling back to static url", "err", err, "fallback", f.fallbackTarget)
			newSequencerUrl = f.fallbackTarget
			nextUpdateIn = f.retryAfterError
		} else {
			log.Error("failed to initialize forward agent", "err", err)
			return f.retryAfterError()
		}
	}
	f.currentTarget = newSequencerUrl
	f.setForwarder(newForwarder)
	return nextUpdateIn()
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
	oldForwarder := f.getForwarder()
	if oldForwarder != nil {
		oldForwarder.StopAndWait()
	}
}

func (f *RedisTxForwarder) Started() bool {
	return f.StopWaiterSafe.Started()
}
