// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"sync"

	"github.com/offchainlabs/nitro/util/pretty"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/daprovider/das/dastree"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type syncedKeysetCache struct {
	cache map[[32]byte][]byte
	sync.RWMutex
}

func (c *syncedKeysetCache) get(key [32]byte) ([]byte, bool) {
	c.RLock()
	defer c.RUnlock()
	res, ok := c.cache[key]
	return res, ok
}

func (c *syncedKeysetCache) put(key [32]byte, value []byte) {
	c.Lock()
	defer c.Unlock()
	c.cache[key] = value
}

type KeysetFetcher struct {
	seqInboxCaller   *bridgegen.SequencerInboxCaller
	seqInboxFilterer *bridgegen.SequencerInboxFilterer
	keysetCache      syncedKeysetCache
}

func NewKeysetFetcher(l1client arbutil.L1Interface, seqInboxAddr common.Address) (*KeysetFetcher, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1client)
	if err != nil {
		return nil, err
	}

	return NewKeysetFetcherWithSeqInbox(seqInbox)
}

func NewKeysetFetcherWithSeqInbox(seqInbox *bridgegen.SequencerInbox) (*KeysetFetcher, error) {
	return &KeysetFetcher{
		seqInboxCaller:   &seqInbox.SequencerInboxCaller,
		seqInboxFilterer: &seqInbox.SequencerInboxFilterer,
		keysetCache:      syncedKeysetCache{cache: make(map[[32]byte][]byte)},
	}, nil
}

func (c *KeysetFetcher) GetKeysetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	log.Trace("das.KeysetFetcher.GetKeysetByHash", "hash", pretty.PrettyHash(hash))
	cache := &c.keysetCache
	seqInboxCaller := c.seqInboxCaller
	seqInboxFilterer := c.seqInboxFilterer

	// try to fetch from the cache
	res, ok := cache.get(hash)
	if ok {
		return res, nil
	}

	// try to fetch from the L1 chain
	blockNumBig, err := seqInboxCaller.GetKeysetCreationBlock(&bind.CallOpts{Context: ctx}, hash)
	if err != nil {
		return nil, err
	}
	if !blockNumBig.IsUint64() {
		return nil, errors.New("block number too large")
	}
	blockNum := blockNumBig.Uint64()
	blockNumPlus1 := blockNum + 1

	filterOpts := &bind.FilterOpts{
		Start:   blockNum,
		End:     &blockNumPlus1,
		Context: ctx,
	}
	iter, err := seqInboxFilterer.FilterSetValidKeyset(filterOpts, [][32]byte{hash})
	if err != nil {
		return nil, err
	}
	for iter.Next() {
		if dastree.ValidHash(hash, iter.Event.KeysetBytes) {
			cache.put(hash, iter.Event.KeysetBytes)
			return iter.Event.KeysetBytes, nil
		}
	}
	if iter.Error() != nil {
		return nil, iter.Error()
	}

	return nil, ErrNotFound
}
