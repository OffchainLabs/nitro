package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type BlockMetadataRebuilderConfig struct {
	Enable          bool          `koanf:"enable"`
	Url             string        `koanf:"url"`
	JWTSecret       string        `koanf:"jwt-secret"`
	RebuildInterval time.Duration `koanf:"rebuild-interval"`
	APIBlocksLimit  uint64        `koanf:"api-blocks-limit"`
}

var DefaultBlockMetadataRebuilderConfig = BlockMetadataRebuilderConfig{
	Enable:          false,
	RebuildInterval: time.Minute * 5,
	APIBlocksLimit:  100,
}

func BlockMetadataRebuilderConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockMetadataRebuilderConfig.Enable, "enable syncing blockMetadata using a bulk metadata api")
	f.String(prefix+".url", DefaultBlockMetadataRebuilderConfig.Url, "url for bulk blockMetadata api")
	f.String(prefix+".jwt-secret", DefaultBlockMetadataRebuilderConfig.JWTSecret, "filepath of jwt secret")
	f.Duration(prefix+".rebuild-interval", DefaultBlockMetadataRebuilderConfig.RebuildInterval, "interval at which blockMetadata is synced regularly")
	f.Uint64(prefix+".api-blocks-limit", DefaultBlockMetadataRebuilderConfig.APIBlocksLimit, "maximum number of blocks allowed to be queried for blockMetadata per arb_getRawBlockMetadata query.\n"+
		"This should be set lesser than or equal to the value set on the api provider side")
}

type BlockMetadataRebuilder struct {
	stopwaiter.StopWaiter
	config BlockMetadataRebuilderConfig
	db     ethdb.Database
	client *rpc.Client
	exec   execution.ExecutionClient
}

func NewBlockMetadataRebuilder(ctx context.Context, c BlockMetadataRebuilderConfig, db ethdb.Database, exec execution.ExecutionClient) (*BlockMetadataRebuilder, error) {
	var err error
	var jwt *common.Hash
	if c.JWTSecret != "" {
		jwt, err = signature.LoadSigningKey(c.JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("BlockMetadataRebuilder: error loading jwt secret: %w", err)
		}
	}
	var client *rpc.Client
	if jwt == nil {
		client, err = rpc.DialOptions(ctx, c.Url)
	} else {
		client, err = rpc.DialOptions(ctx, c.Url, rpc.WithHTTPAuth(node.NewJWTAuth([32]byte(*jwt))))
	}
	if err != nil {
		return nil, fmt.Errorf("BlockMetadataRebuilder: error connecting to bulk blockMetadata API: %w", err)
	}
	return &BlockMetadataRebuilder{
		config: c,
		db:     db,
		client: client,
		exec:   exec,
	}, nil
}

func (b *BlockMetadataRebuilder) Fetch(ctx context.Context, fromBlock, toBlock uint64) ([]gethexec.NumberAndBlockMetadata, error) {
	var result []gethexec.NumberAndBlockMetadata
	err := b.client.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(fromBlock), rpc.BlockNumber(toBlock))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ArrayToMap[T comparable](arr []T) map[T]struct{} {
	ret := make(map[T]struct{})
	for _, elem := range arr {
		ret[elem] = struct{}{}
	}
	return ret
}

func (b *BlockMetadataRebuilder) PushBlockMetadataToDB(query []uint64, result []gethexec.NumberAndBlockMetadata) error {
	batch := b.db.NewBatch()
	queryMap := ArrayToMap(query)
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
		}
	}
	return batch.Write()
}

func (b *BlockMetadataRebuilder) Update(ctx context.Context) time.Duration {
	handleQuery := func(query []uint64) bool {
		result, err := b.Fetch(
			ctx,
			b.exec.MessageIndexToBlockNumber(arbutil.MessageIndex(query[0])),
			b.exec.MessageIndexToBlockNumber(arbutil.MessageIndex(query[len(query)-1])),
		)
		if err != nil {
			log.Error("Error getting result from bulk blockMetadata API", "err", err)
			return false
		}
		if err = b.PushBlockMetadataToDB(query, result); err != nil {
			log.Error("Error committing result from bulk blockMetadata API to ArbDB", "err", err)
			return false
		}
		return true
	}
	iter := b.db.NewIterator(missingBlockMetadataInputFeedPrefix, nil)
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
				return b.config.RebuildInterval
			}
			query = query[end+1:]
		}
	}
	if len(query) > 0 {
		if success := handleQuery(query); !success {
			return b.config.RebuildInterval
		}
	}
	return b.config.RebuildInterval
}

func (b *BlockMetadataRebuilder) Start(ctx context.Context) {
	b.StopWaiter.Start(ctx, b)
	b.CallIteratively(b.Update)
}

func (b *BlockMetadataRebuilder) StopAndWait() {
	b.StopWaiter.StopAndWait()
}
