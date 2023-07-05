// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package leveldb

import (
	"context"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/nitro/arbnode"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Storage implements leveldb based storage for batch poster.
type Storage[Item any] struct {
	db ethdb.Database
}

type Options struct {
	File      string
	Cache     int
	Handles   int
	Namespace string
	ReadOnly  bool
}

func New[Item any](o *Options) (*Storage[Item], error) {
	db, err := rawdb.NewLevelDBDatabase(o.File, o.Cache, o.Handles, o.Namespace, o.ReadOnly)
	if err != nil {
		return nil, err
	}
	return &Storage[Item]{
		db: rawdb.NewTable(db, arbnode.DataPosterPrefix),
	}, nil
}

func (s *Storage[Item]) GetContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error) {
	s.db.
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (s *Storage[Item]) GetLast(ctx context.Context) (*Item, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (s *Storage[Item]) Prune(ctx context.Context, keepStartingAt uint64) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

func (s *Storage[Item]) Put(ctx context.Context, index uint64, prevItem *Item, newItem *Item) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

func (s *Storage[Item]) Length(ctx context.Context) (int, error) {
	return 0, status.Errorf(codes.Unimplemented, "not implemented")
}

func (s *Storage[Item]) IsPersistent() bool {
	return false
}
