// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type TxForwarder struct {
	enabled   int32
	target    string
	timeout   time.Duration
	rpcClient *rpc.Client
	ethClient *ethclient.Client

	healthMutex   sync.Mutex
	healthErr     error
	healthChecked time.Time
}

func NewForwarder(target string, timeout time.Duration) *TxForwarder {
	return &TxForwarder{
		target:  target,
		timeout: timeout,
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

const cacheUpstreamHealth = 5 * time.Second
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
	rpcClient, err := rpc.DialContext(ctx, f.target)
	if err != nil {
		return err
	}
	f.rpcClient = rpcClient
	f.ethClient = ethclient.NewClient(rpcClient)
	f.enabled = 1
	return nil
}

// Not thread-safe vs. Initialize
func (f *TxForwarder) Disable() {
	atomic.StoreInt32(&f.enabled, 0)
}

func (f *TxForwarder) Start(ctx context.Context) error {
	return nil
}

func (f *TxForwarder) StopAndWait() {}

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
