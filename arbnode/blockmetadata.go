package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/ethclient"
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
	Enable          bool                   `koanf:"enable"`
	Source          rpcclient.ClientConfig `koanf:"source" reload:"hot"`
	SyncInterval    time.Duration          `koanf:"sync-interval"`
	MaxSyncInterval time.Duration          `koanf:"max-sync-interval"`
	APIBlocksLimit  uint64                 `koanf:"api-blocks-limit"`
}

var DefaultBlockMetadataFetcherConfig = BlockMetadataFetcherConfig{
	Enable:          false,
	Source:          rpcclient.DefaultClientConfig,
	SyncInterval:    time.Minute * 1,
	MaxSyncInterval: time.Minute * 32,
	APIBlocksLimit:  100,
}

func BlockMetadataFetcherConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockMetadataFetcherConfig.Enable, "enable syncing blockMetadata using a bulk blockMetadata api")
	rpcclient.RPCClientAddOptions(prefix+".source", f, &DefaultBlockMetadataFetcherConfig.Source)
	f.Duration(prefix+".sync-interval", DefaultBlockMetadataFetcherConfig.SyncInterval, "minimum time between blockMetadata requests")
	f.Duration(prefix+".max-sync-interval", DefaultBlockMetadataFetcherConfig.MaxSyncInterval, "maximum time between blockMetadata requests")
	f.Uint64(prefix+".api-blocks-limit", DefaultBlockMetadataFetcherConfig.APIBlocksLimit, "maximum number of blocks per arb_getRawBlockMetadata query")
}

var wrongChainIdErr = errors.New("wrong chain id")

func checkMetadataBackendChainId(ctx context.Context, client *rpcclient.RpcClient, sourceUrl string, expectedChainId uint64) error {
	ethClient := ethclient.NewClient(client)
	chainId, err := ethClient.ChainID(ctx)
	if err != nil {
		log.Error("error when getting ChainId from backend configured with --node.block-metadata-fetcher.source.url", "url", sourceUrl, "err", err)
		return errors.New("failed to get chainid")
	}
	if chainId.Uint64() != expectedChainId {
		log.Error("ChainId from backend configured with --node.block-metadata-fetcher.source.url does not match expected ChainId", "backendChainId", chainId.Uint64(), "expectedChainId", expectedChainId, "url", sourceUrl)
		return wrongChainIdErr
	}
	return nil
}

// BlockMetadataFetcher looks for missing blockMetadata of block numbers starting from trackBlockMetadataFrom (config option of tx streamer)
// and adds them to arbDB. BlockMetadata is fetched by querying the source's bulk blockMetadata fetching API "arb_getRawBlockMetadata".
// Missing trackers are removed after their corresponding blockMetadata are added to the arbDB
type BlockMetadataFetcher struct {
	stopwaiter.StopWaiter
	config                 BlockMetadataFetcherConfig
	db                     ethdb.Database
	client                 *rpcclient.RpcClient
	exec                   execution.ExecutionClient
	trackBlockMetadataFrom arbutil.MessageIndex
	expectedChainId        uint64

	chainIdChecked      bool
	currentSyncInterval time.Duration
	lastRequestTime     time.Time
}

func NewBlockMetadataFetcher(
	ctx context.Context,
	c BlockMetadataFetcherConfig,
	db ethdb.Database,
	exec execution.ExecutionClient,
	startPos uint64,
	expectedChainId uint64,
) (*BlockMetadataFetcher, error) {
	var trackBlockMetadataFrom arbutil.MessageIndex
	var err error
	if startPos != 0 {
		trackBlockMetadataFrom, err = exec.BlockNumberToMessageIndex(startPos).Await(ctx)
		if err != nil {
			return nil, err
		}
	}
	client := rpcclient.NewRpcClient(func() *rpcclient.ClientConfig { return &c.Source }, nil)
	if err = client.Start(ctx); err != nil {
		return nil, err
	}

	chainIdChecked := false
	if err = checkMetadataBackendChainId(ctx, client, c.Source.URL, expectedChainId); err != nil {
		if errors.Is(err, wrongChainIdErr) {
			return nil, err
		}
	} else {
		chainIdChecked = true
	}

	fetcher := &BlockMetadataFetcher{
		config:                 c,
		db:                     db,
		client:                 client,
		exec:                   exec,
		trackBlockMetadataFrom: trackBlockMetadataFrom,
		expectedChainId:        expectedChainId,
		chainIdChecked:         chainIdChecked,
		currentSyncInterval:    c.SyncInterval,
	}
	return fetcher, nil
}

func (b *BlockMetadataFetcher) fetch(ctx context.Context, fromBlock, toBlock uint64) ([]gethexec.NumberAndBlockMetadata, error) {
	// Rate limit: 1 request per second
	now := time.Now()
	if !b.lastRequestTime.IsZero() {
		waitTime := time.Second - now.Sub(b.lastRequestTime)
		if waitTime > 0 {
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	var result []gethexec.NumberAndBlockMetadata
	// #nosec G115
	err := b.client.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(fromBlock), rpc.BlockNumber(toBlock))
	b.lastRequestTime = time.Now()

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (b *BlockMetadataFetcher) persistBlockMetadata(ctx context.Context, query []uint64, result []gethexec.NumberAndBlockMetadata) error {
	batch := b.db.NewBatch()
	queryMap := util.ArrayToSet(query)
	for _, elem := range result {
		pos, err := b.exec.BlockNumberToMessageIndex(elem.BlockNumber).Await(ctx)
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
	if !b.chainIdChecked {
		if err := checkMetadataBackendChainId(ctx, b.client, b.config.Source.URL, b.expectedChainId); err != nil {
			log.Error("Error running the BlockMetadataFetcher", "err", err)
			return time.Minute * 10
		}
		b.chainIdChecked = true
	}

	handleQuery := func(query []uint64) bool {
		fromBlock, err := b.exec.MessageIndexToBlockNumber(arbutil.MessageIndex(query[0])).Await(ctx)
		if err != nil {
			log.Error("Error getting fromBlock", "err", err)
			return false
		}
		toBlock, err := b.exec.MessageIndexToBlockNumber(arbutil.MessageIndex(query[len(query)-1])).Await(ctx)
		if err != nil {
			log.Error("Error getting toBlock", "err", err)
			return false
		}

		result, err := b.fetch(
			ctx,
			fromBlock,
			toBlock,
		)
		if err != nil {
			log.Error("Error getting result from bulk blockMetadata API", "err", err)
			return false
		}
		if err = b.persistBlockMetadata(ctx, query, result); err != nil {
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
				b.currentSyncInterval *= 2
				if b.currentSyncInterval > b.config.MaxSyncInterval {
					b.currentSyncInterval = b.config.MaxSyncInterval
				}
				return b.currentSyncInterval
			}
			if b.currentSyncInterval > b.config.SyncInterval {
				b.currentSyncInterval /= 2
				if b.currentSyncInterval < b.config.SyncInterval {
					b.currentSyncInterval = b.config.SyncInterval
				}
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
