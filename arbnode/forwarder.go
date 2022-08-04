// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

type TxForwarder struct {
	enabled int32
	target  string
	timeout time.Duration
	client  *ethclient.Client
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
	return f.client.SendTransaction(ctx, tx)
}

func (f *TxForwarder) Initialize(inctx context.Context) error {
	if f.target == "" {
		f.client = nil
		f.enabled = 0
		return nil
	}
	ctx, cancelFunc := f.ctxWithTimeout(inctx)
	defer cancelFunc()
	client, err := ethclient.DialContext(ctx, f.target)
	if err != nil {
		return err
	}
	f.client = client
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

func (f *TxDropper) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	return errors.New("transactions not supported by this endpoint")
}

func (f *TxDropper) Initialize(ctx context.Context) error { return nil }

func (f *TxDropper) Start(ctx context.Context) error { return nil }

func (f *TxDropper) StopAndWait() {}
