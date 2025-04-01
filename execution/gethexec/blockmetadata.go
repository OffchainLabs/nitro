package gethexec

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var ErrBlockMetadataApiBlocksLimitExceeded = errors.New("number of blocks requested for blockMetadata exceeded")

type BlockMetadataFetcher interface {
	BlockMetadataAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (common.BlockMetadata, error)
	BlockNumberToMessageIndex(blockNum uint64) (arbutil.MessageIndex, error)
	MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64
	SetReorgEventsNotifier(reorgEventsNotifier chan struct{})
}

// BulkBlockMetadataFetcher is the underlying provider of bulk blockMetadata to service arb_getRawBlockMetadata api. Given a starting
// and ending block number, it returns an array of struct (NumberAndBlockMetadata) containing blockMetadata and their corresponding blockNumbers
type BulkBlockMetadataFetcher struct {
	stopwaiter.StopWaiter
	bc            *core.BlockChain
	fetcher       BlockMetadataFetcher
	reorgDetector chan struct{}
	blocksLimit   uint64
	cache         *lru.SizeConstrainedCache[arbutil.MessageIndex, common.BlockMetadata]
}

func NewBulkBlockMetadataFetcher(bc *core.BlockChain, fetcher BlockMetadataFetcher, cacheSize, blocksLimit uint64) *BulkBlockMetadataFetcher {
	var cache *lru.SizeConstrainedCache[arbutil.MessageIndex, common.BlockMetadata]
	var reorgDetector chan struct{}
	if cacheSize != 0 {
		cache = lru.NewSizeConstrainedCache[arbutil.MessageIndex, common.BlockMetadata](cacheSize)
		reorgDetector = make(chan struct{})
		fetcher.SetReorgEventsNotifier(reorgDetector)
	}
	return &BulkBlockMetadataFetcher{
		bc:            bc,
		fetcher:       fetcher,
		cache:         cache,
		reorgDetector: reorgDetector,
		blocksLimit:   blocksLimit,
	}
}

// Fetch won't include block numbers for whom consensus (arbDB) doesn't have blockMetadata, it stores recently fetched blockMetadata into an LRU
// which is cleared in the events of reorg in order to provide accurate blockMetadata
func (b *BulkBlockMetadataFetcher) Fetch(ctx context.Context, fromBlock, toBlock rpc.BlockNumber) ([]NumberAndBlockMetadata, error) {
	fromBlock, _ = b.bc.ClipToPostNitroGenesis(fromBlock)
	toBlock, _ = b.bc.ClipToPostNitroGenesis(toBlock)
	// #nosec G115
	start, err := b.fetcher.BlockNumberToMessageIndex(uint64(fromBlock))
	if err != nil {
		return nil, fmt.Errorf("error converting fromBlock blocknumber to message index: %w", err)
	}
	// #nosec G115
	end, err := b.fetcher.BlockNumberToMessageIndex(uint64(toBlock))
	if err != nil {
		return nil, fmt.Errorf("error converting toBlock blocknumber to message index: %w", err)
	}
	if start > end {
		return nil, fmt.Errorf("invalid inputs, fromBlock: %d is greater than toBlock: %d", fromBlock, toBlock)
	}
	if b.blocksLimit > 0 && end-start+1 > arbutil.MessageIndex(b.blocksLimit) {
		return nil, fmt.Errorf("%w. Range requested- %d, Limit- %d", ErrBlockMetadataApiBlocksLimitExceeded, end-start+1, b.blocksLimit)
	}
	var result []NumberAndBlockMetadata
	for i := start; i <= end; i++ {
		var data common.BlockMetadata
		var found bool
		if b.cache != nil {
			data, found = b.cache.Get(i)
		}
		if !found {
			data, err = b.fetcher.BlockMetadataAtMessageIndex(ctx, i)
			if err != nil {
				return nil, err
			}
			if data != nil && b.cache != nil {
				b.cache.Add(i, data)
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
	b.cache.Clear()
}

func (b *BulkBlockMetadataFetcher) Start(ctx context.Context) {
	b.StopWaiter.Start(ctx, b)
	if b.reorgDetector != nil {
		_ = stopwaiter.CallWhenTriggeredWith[struct{}](&b.StopWaiterSafe, b.ClearCache, b.reorgDetector)
	}
}

func (b *BulkBlockMetadataFetcher) StopAndWait() {
	b.StopWaiter.StopAndWait()
}
