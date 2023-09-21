// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
package noop

import (
	"context"

	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
)

// Storage implements noop storage for dataposter. This is for clients that want
// to have option to directly post to geth without keeping state.
type Storage struct{}

func (s *Storage) FetchContents(_ context.Context, _, _ uint64) ([]*storage.QueuedTransaction, error) {
	return nil, nil
}

func (s *Storage) FetchLast(ctx context.Context) (*storage.QueuedTransaction, error) {
	return nil, nil
}

func (s *Storage) Prune(_ context.Context, _ uint64) error {
	return nil
}

func (s *Storage) Put(_ context.Context, _ uint64, _, _ *storage.QueuedTransaction) error {
	return nil
}

func (s *Storage) Length(context.Context) (int, error) {
	return 0, nil
}

func (s *Storage) IsPersistent() bool {
	return false
}
