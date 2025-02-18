package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type BlockMetadataFetcherConfig struct {
	Enable         bool                   `koanf:"enable"`
	Source         rpcclient.ClientConfig `koanf:"source" reload:"hot"`
	SyncInterval   time.Duration          `koanf:"sync-interval"`
	APIBlocksLimit uint64                 `koanf:"api-blocks-limit"`
}

var DefaultBlockMetadataFetcherConfig = BlockMetadataFetcherConfig{
	Enable:         false,
	Source:         rpcclient.DefaultClientConfig,
	SyncInterval:   time.Minute * 5,
	APIBlocksLimit: 100,
}

func BlockMetadataFetcherConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockMetadataFetcherConfig.Enable, "enable syncing blockMetadata using a bulk blockMetadata api. If the source doesn't have the missing blockMetadata, we keep retyring in every sync-interval (default=5mins) duration")
	rpcclient.RPCClientAddOptions(prefix+".source", f, &DefaultBlockMetadataFetcherConfig.Source)
	f.Duration(prefix+".sync-interval", DefaultBlockMetadataFetcherConfig.SyncInterval, "interval at which blockMetadata are synced regularly")
	f.Uint64(prefix+".api-blocks-limit", DefaultBlockMetadataFetcherConfig.APIBlocksLimit, "maximum number of blocks allowed to be queried for blockMetadata per arb_getRawBlockMetadata query.\n"+
		"This should be set lesser than or equal to the limit on the api provider side")
}

type BlockMetadataFetcher struct {
	stopwaiter.StopWaiter
	config                 BlockMetadataFetcherConfig
	db                     ethdb.Database
	client                 *rpcclient.RpcClient
	exec                   execution.ExecutionClient
	trackBlockMetadataFrom arbutil.MessageIndex
}

func NewBlockMetadataFetcher(ctx context.Context, c BlockMetadataFetcherConfig, db ethdb.Database, exec execution.ExecutionClient, startPos uint64) (*BlockMetadataFetcher, error) {
	var trackBlockMetadataFrom arbutil.MessageIndex
	var err error
	if startPos != 0 {
		trackBlockMetadataFrom, err = exec.BlockNumberToMessageIndex(startPos)
		if err != nil {
			return nil, err
		}
	}
	client := rpcclient.NewRpcClient(func() *rpcclient.ClientConfig { return &c.Source }, nil)
	if err = client.Start(ctx); err != nil {
		return nil, err
	}
	return &BlockMetadataFetcher{
		config:                 c,
		db:                     db,
		client:                 client,
		exec:                   exec,
		trackBlockMetadataFrom: trackBlockMetadataFrom,
	}, nil
}

func (b *BlockMetadataFetcher) fetch(ctx context.Context, fromBlock, toBlock uint64) ([]gethexec.NumberAndBlockMetadata, error) {
	var result []gethexec.NumberAndBlockMetadata
	// #nosec G115
	err := b.client.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(fromBlock), rpc.BlockNumber(toBlock))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (b *BlockMetadataFetcher) persistBlockMetadata(query []uint64, result []gethexec.NumberAndBlockMetadata) error {
	batch := b.db.NewBatch()
	queryMap := util.ArrayToSet(query)
	for _, elem := range result {
		pos, err := b.exec.BlockNumberToMessageIndex(elem.BlockNumber)
		if err != nil {
			return err
		}
		if _, ok := queryMap[uint64(pos)]; ok {
			if err := batch.Put(dbKey(blockMetadataInputFeedPrefix, uint64(pos)), elem.RawMetadata); err != nil {
				return err
			}
			if err := batch.Delete(dbKey(missingBlockMetadataInputFeedPrefix, uint64(pos))); err != nil {
				return err
			}
			// If we reached the ideal batch size, commit and reset
			if batch.ValueSize() >= ethdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					return err
				}
				batch.Reset()
			}
		}
	}
	return batch.Write()
}

func (b *BlockMetadataFetcher) Update(ctx context.Context) time.Duration {
	handleQuery := func(query []uint64) bool {
		result, err := b.fetch(
			ctx,
			b.exec.MessageIndexToBlockNumber(arbutil.MessageIndex(query[0])),
			b.exec.MessageIndexToBlockNumber(arbutil.MessageIndex(query[len(query)-1])),
		)
		if err != nil {
			log.Error("Error getting result from bulk blockMetadata API", "err", err)
			return false
		}
		if err = b.persistBlockMetadata(query, result); err != nil {
			log.Error("Error committing result from bulk blockMetadata API to ArbDB", "err", err)
			return false
		}
		return true
	}
	var start []byte
	if b.trackBlockMetadataFrom != 0 {
		start = uint64ToKey(uint64(b.trackBlockMetadataFrom))
	}
	iter := b.db.NewIterator(missingBlockMetadataInputFeedPrefix, start)
	defer iter.Release()
	var query []uint64
	for iter.Next() {
		keyBytes := bytes.TrimPrefix(iter.Key(), missingBlockMetadataInputFeedPrefix)
		query = append(query, binary.BigEndian.Uint64(keyBytes))
		end := len(query) - 1
		if query[end]-query[0]+1 >= uint64(b.config.APIBlocksLimit) {
			if query[end]-query[0]+1 > uint64(b.config.APIBlocksLimit) && len(query) >= 2 {
				end -= 1
			}
			if success := handleQuery(query[:end+1]); !success {
				return b.config.SyncInterval
			}
			query = query[end+1:]
		}
	}
	if len(query) > 0 {
		_ = handleQuery(query)
	}
	return b.config.SyncInterval
}

func (b *BlockMetadataFetcher) Start(ctx context.Context) {
	b.StopWaiter.Start(ctx, b)
	b.CallIteratively(b.Update)
}

func (b *BlockMetadataFetcher) StopAndWait() {
	b.StopWaiter.StopAndWait()
	b.client.Close()
}
