// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"sync"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/pretty"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
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

type ChainFetchDAS struct {
	DataAvailabilityService
	seqInboxCaller   *bridgegen.SequencerInboxCaller
	seqInboxFilterer *bridgegen.SequencerInboxFilterer
	keysetCache      syncedKeysetCache
}

type ChainFetchReader struct {
	arbstate.DataAvailabilityReader
	seqInboxCaller   *bridgegen.SequencerInboxCaller
	seqInboxFilterer *bridgegen.SequencerInboxFilterer
	keysetCache      syncedKeysetCache
}

func NewChainFetchDAS(inner DataAvailabilityService, l1client arbutil.L1Interface, seqInboxAddr common.Address) (*ChainFetchDAS, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1client)
	if err != nil {
		return nil, err
	}
	return NewChainFetchDASWithSeqInbox(inner, seqInbox)
}

func NewChainFetchDASWithSeqInbox(inner DataAvailabilityService, seqInbox *bridgegen.SequencerInbox) (*ChainFetchDAS, error) {
	return &ChainFetchDAS{
		DataAvailabilityService: inner,
		seqInboxCaller:          &seqInbox.SequencerInboxCaller,
		seqInboxFilterer:        &seqInbox.SequencerInboxFilterer,
		keysetCache:             syncedKeysetCache{cache: make(map[[32]byte][]byte)},
	}, nil
}

func NewChainFetchReader(inner arbstate.DataAvailabilityReader, l1client arbutil.L1Interface, seqInboxAddr common.Address) (*ChainFetchReader, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1client)
	if err != nil {
		return nil, err
	}

	return NewChainFetchReaderWithSeqInbox(inner, seqInbox)
}

func NewChainFetchReaderWithSeqInbox(inner arbstate.DataAvailabilityReader, seqInbox *bridgegen.SequencerInbox) (*ChainFetchReader, error) {
	return &ChainFetchReader{
		DataAvailabilityReader: inner,
		seqInboxCaller:         &seqInbox.SequencerInboxCaller,
		seqInboxFilterer:       &seqInbox.SequencerInboxFilterer,
		keysetCache:            syncedKeysetCache{cache: make(map[[32]byte][]byte)},
	}, nil
}

func (this *ChainFetchDAS) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	log.Trace("das.ChainFetchDAS.GetByHash", "hash", pretty.FirstFewBytes(hash))
	return chainFetchGetByHash(ctx, this.DataAvailabilityService, &this.keysetCache, this.seqInboxCaller, this.seqInboxFilterer, hash)
}

func (this *ChainFetchReader) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	log.Trace("das.ChainFetchReader.GetByHash", "hash", pretty.FirstFewBytes(hash))
	return chainFetchGetByHash(ctx, this.DataAvailabilityReader, &this.keysetCache, this.seqInboxCaller, this.seqInboxFilterer, hash)
}

func chainFetchGetByHash(
	ctx context.Context,
	daReader arbstate.DataAvailabilityReader,
	cache *syncedKeysetCache,
	seqInboxCaller *bridgegen.SequencerInboxCaller,
	seqInboxFilterer *bridgegen.SequencerInboxFilterer,
	hash []byte,
) ([]byte, error) {
	// try to fetch from the cache
	var hash32 [32]byte
	copy(hash32[:], hash)
	res, ok := cache.get(hash32)
	if ok {
		return res, nil
	}

	// try to fetch from the inner DAS
	innerRes, err := daReader.GetByHash(ctx, hash)
	if err == nil && bytes.Equal(hash, crypto.Keccak256(innerRes)) {
		return innerRes, nil
	}

	// try to fetch from the L1 chain
	blockNumBig, err := seqInboxCaller.GetKeysetCreationBlock(&bind.CallOpts{Context: ctx}, hash32)
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
	iter, err := seqInboxFilterer.FilterSetValidKeyset(filterOpts, [][32]byte{hash32})
	if err != nil {
		return nil, err
	}
	for iter.Next() {
		if bytes.Equal(hash, crypto.Keccak256(iter.Event.KeysetBytes)) {
			cache.put(hash32, iter.Event.KeysetBytes)
			return iter.Event.KeysetBytes, nil
		}
	}
	if iter.Error() != nil {
		return nil, iter.Error()
	}

	return nil, ErrNotFound
}
