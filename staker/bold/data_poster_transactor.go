// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package bold

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
)

// DataPosterTransactor is a wrapper around a DataPoster that implements the Transactor interface.
type DataPosterTransactor struct {
	fifo *solimpl.FIFO
	*dataposter.DataPoster
}

func NewDataPosterTransactor(dataPoster *dataposter.DataPoster) *DataPosterTransactor {
	return &DataPosterTransactor{
		fifo:       solimpl.NewFIFO(1000),
		DataPoster: dataPoster,
	}
}

func (d *DataPosterTransactor) SendTransaction(ctx context.Context, fn func(opts *bind.TransactOpts) (*types.Transaction, error), opts *bind.TransactOpts, gas uint64) (*types.Transaction, error) {
	// Try to acquire lock and if it fails, wait for a bit and try again.
	for !d.fifo.Lock() {
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	defer d.fifo.Unlock()
	tx, err := fn(opts)
	if err != nil {
		return nil, err
	}
	return d.PostSimpleTransaction(ctx, *tx.To(), tx.Data(), gas, tx.Value())
}
