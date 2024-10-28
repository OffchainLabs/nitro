package gethexec

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type BlockMetadataFetcher interface {
	BlockMetadataAtCount(count arbutil.MessageIndex) (arbostypes.BlockMetadata, error)
	BlockNumberToMessageIndex(blockNum uint64) (arbutil.MessageIndex, error)
	MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64
	SetReorgEventsReader(reorgEventsReader chan struct{})
}

type BulkBlockMetadataFetcher struct {
	stopwaiter.StopWaiter
	bc            *core.BlockChain
	fetcher       BlockMetadataFetcher
	reorgDetector chan struct{}
	cacheMutex    sync.RWMutex
	cache         *containers.LruCache[arbutil.MessageIndex, arbostypes.BlockMetadata]
}

func NewBulkBlockMetadataFetcher(bc *core.BlockChain, fetcher BlockMetadataFetcher, cacheSize int) *BulkBlockMetadataFetcher {
	var cache *containers.LruCache[arbutil.MessageIndex, arbostypes.BlockMetadata]
	var reorgDetector chan struct{}
	if cacheSize != 0 {
		cache = containers.NewLruCache[arbutil.MessageIndex, arbostypes.BlockMetadata](cacheSize)
		reorgDetector = make(chan struct{})
		fetcher.SetReorgEventsReader(reorgDetector)
	}
	return &BulkBlockMetadataFetcher{
		bc:            bc,
		fetcher:       fetcher,
		cache:         cache,
		reorgDetector: reorgDetector,
	}
}

func (b *BulkBlockMetadataFetcher) Fetch(fromBlock, toBlock rpc.BlockNumber) ([]NumberAndBlockMetadata, error) {
	fromBlock, _ = b.bc.ClipToPostNitroGenesis(fromBlock)
	toBlock, _ = b.bc.ClipToPostNitroGenesis(toBlock)
	start, err := b.fetcher.BlockNumberToMessageIndex(uint64(fromBlock))
	if err != nil {
		return nil, fmt.Errorf("error converting fromBlock blocknumber to message index: %w", err)
	}
	end, err := b.fetcher.BlockNumberToMessageIndex(uint64(toBlock))
	if err != nil {
		return nil, fmt.Errorf("error converting toBlock blocknumber to message index: %w", err)
	}
	var result []NumberAndBlockMetadata
	for i := start; i <= end; i++ {
		var data arbostypes.BlockMetadata
		var found bool
		if b.cache != nil {
			b.cacheMutex.RLock()
			data, found = b.cache.Get(i)
			b.cacheMutex.RUnlock()
		}
		if !found {
			data, err = b.fetcher.BlockMetadataAtCount(i + 1)
			if err != nil {
				return nil, err
			}
			if data != nil && b.cache != nil {
				b.cacheMutex.Lock()
				b.cache.Add(i, data)
				b.cacheMutex.Unlock()
			}
		}
		if data != nil {
			result = append(result, NumberAndBlockMetadata{
				BlockNumber: b.fetcher.MessageIndexToBlockNumber(i),
				RawMetadata: (hexutil.Bytes)(data),
			})
		}
	}
	return result, nil
}

func (b *BulkBlockMetadataFetcher) ClearCache(ctx context.Context, ignored struct{}) {
	b.cacheMutex.Lock()
	b.cache.Clear()
	b.cacheMutex.Unlock()
}

func (b *BulkBlockMetadataFetcher) Start(ctx context.Context) {
	b.StopWaiter.Start(ctx, b)
	if b.reorgDetector != nil {
		stopwaiter.CallWhenTriggeredWith[struct{}](&b.StopWaiterSafe, b.ClearCache, b.reorgDetector)
	}
}

func (b *BulkBlockMetadataFetcher) StopAndWait() {
	b.StopWaiter.StopAndWait()
}
