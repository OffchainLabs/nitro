// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
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
	RedisUrl:              "",
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
	ctx context.Context

	enabled   atomic.Bool
	timeout   time.Duration
	transport *http.Transport

	healthMutex   sync.Mutex
	healthErr     error
	healthChecked time.Time

	targets               []string
	rpcClients            []*rpc.Client
	ethClients            []*ethclient.Client
	tryNewForwarderErrors *regexp.Regexp
}

func NewForwarder(targets []string, config *ForwarderConfig) *TxForwarder {
	dialer := net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 2 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// For tcp connections, prefer IPv4 over IPv6
			if network == "tcp" {
				conn, err := dialer.DialContext(ctx, "tcp4", addr)
				if err == nil {
					return conn, nil
				}
				return dialer.DialContext(ctx, "tcp6", addr)
			}
			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:          config.MaxIdleConnections,
		MaxIdleConnsPerHost:   config.MaxIdleConnections,
		IdleConnTimeout:       config.IdleConnectionTimeout,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &TxForwarder{
		targets:               targets,
		timeout:               config.ConnectionTimeout,
		transport:             transport,
		tryNewForwarderErrors: regexp.MustCompile(`(?i)(^http:|^json:|^i/0|timeout exceeded|no such host)`),
	}
}

func (f *TxForwarder) ctxWithTimeout() (context.Context, context.CancelFunc) {
	if f.timeout == time.Duration(0) {
		return context.WithCancel(f.ctx)
	}
	return context.WithTimeout(f.ctx, f.timeout)
}

func (f *TxForwarder) PublishTransaction(inctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	if !f.enabled.Load() {
		return ErrNoSequencer
	}
	ctx, cancelFunc := f.ctxWithTimeout()
	defer cancelFunc()
	for pos, rpcClient := range f.rpcClients {
		var err error
		if options == nil {
			err = f.ethClients[pos].SendTransaction(ctx, tx)
		} else {
			err = arbitrum.SendConditionalTransactionRPC(ctx, rpcClient, tx, options)
		}
		if err == nil || !f.tryNewForwarderErrors.MatchString(err.Error()) {
			return err
		}
		log.Warn("error forwarding transaction to a backup target", "target", f.targets[pos], "err", err)
	}
	return errors.New("failed to publish transaction to any of the forwarding targets")
}

const cacheUpstreamHealth = 2 * time.Second
const maxHealthTimeout = 10 * time.Second

// CheckHealth returns health of the highest priority forwarding target
func (f *TxForwarder) CheckHealth(inctx context.Context) error {
	// If f.enabled is true, len(f.rpcClients) should always be greater than zero,
	// but better safe than sorry.
	if !f.enabled.Load() || len(f.rpcClients) == 0 {
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
		f.healthErr = f.rpcClients[0].CallContext(ctx, nil, "arb_checkPublisherHealth")
		f.healthChecked = time.Now()
	}
	return f.healthErr
}

func (f *TxForwarder) Initialize(inctx context.Context) error {
	if f.ctx == nil {
		f.ctx = inctx
	}
	ctx, cancelFunc := f.ctxWithTimeout()
	defer cancelFunc()
	var targets []string
	var lastError error
	for _, target := range f.targets {
		if target == "" {
			continue
		}
		rpcClient, err := rpc.DialTransport(ctx, target, f.transport)
		if err != nil {
			log.Warn("error initializing a forwarding client in txForwarder", "forwarding url", target, "err", err)
			lastError = err
			continue
		}
		targets = append(targets, target)
		ethClient := ethclient.NewClient(rpcClient)
		f.rpcClients = append(f.rpcClients, rpcClient)
		f.ethClients = append(f.ethClients, ethClient)
	}
	f.targets = targets
	if len(f.rpcClients) > 0 {
		f.enabled.Store(true)
	} else {
		return lastError
	}
	return nil
}

// Disable is not thread-safe vs. Initialize
func (f *TxForwarder) Disable() {
	f.enabled.Store(false)
}

func (f *TxForwarder) Start(ctx context.Context) error {
	return nil
}

func (f *TxForwarder) StopAndWait() {
	for _, ethClient := range f.ethClients {
		ethClient.Close() // internally closes also the rpc client
	}
}

func (f *TxForwarder) Started() bool {
	return true
}

// Returns the URL of the first forwarding target, or an empty string if none are set.
func (f *TxForwarder) PrimaryTarget() string {
	if len(f.targets) == 0 {
		return ""
	}
	return f.targets[0]
}

type TxDropper struct{}

func NewTxDropper() *TxDropper {
	return &TxDropper{}
}

var txDropperErr = errors.New("publishing transactions not supported by this endpoint")

func (f *TxDropper) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
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

func (f *RedisTxForwarder) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	forwarder := f.getForwarder()
	if forwarder == nil {
		return ErrNoSequencer
	}
	return forwarder.PublishTransaction(ctx, tx, options)
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
		return fmt.Errorf("unable to create redis coordinator: %w", err)
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
		newForwarder = NewForwarder([]string{newSequencerUrl}, f.config)
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
	if err := f.CallIterativelySafe(f.update); err != nil {
		return fmt.Errorf("failed to start forwarder update thread: %w", err)
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
