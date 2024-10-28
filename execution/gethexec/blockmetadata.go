package gethexec

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
)

type BlockMetadataFetcher interface {
	BlockMetadataAtCount(count arbutil.MessageIndex) (arbostypes.BlockMetadata, error)
	BlockNumberToMessageIndex(blockNum uint64) (arbutil.MessageIndex, error)
	MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64
}

type BulkBlockMetadataFetcher struct {
	bc      *core.BlockChain
	fetcher BlockMetadataFetcher
	cache   *containers.LruCache[arbutil.MessageIndex, arbostypes.BlockMetadata]
}

func NewBulkBlockMetadataFetcher(bc *core.BlockChain, fetcher BlockMetadataFetcher, cacheSize int) *BulkBlockMetadataFetcher {
	var cache *containers.LruCache[arbutil.MessageIndex, arbostypes.BlockMetadata]
	if cacheSize != 0 {
		cache = containers.NewLruCache[arbutil.MessageIndex, arbostypes.BlockMetadata](cacheSize)
	}
	return &BulkBlockMetadataFetcher{
		bc:      bc,
		fetcher: fetcher,
		cache:   cache,
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
			data, found = b.cache.Get(i)
		}
		if !found {
			data, err = b.fetcher.BlockMetadataAtCount(i + 1)
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
