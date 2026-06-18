// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package dataposter implements generic functionality to post transactions.
package dataposter

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Knetic/govaluate"
	"github.com/holiman/uint256"
	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/dataposter/dbstorage"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsigner"
	"github.com/offchainlabs/nitro/arbnode/dataposter/noop"
	redisstorage "github.com/offchainlabs/nitro/arbnode/dataposter/redis"
	"github.com/offchainlabs/nitro/arbnode/dataposter/slice"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// DataPoster implements functionality to post transactions on the chain. It
// is initialized with specified sender/signer and keeps nonce of that address
// as it posts transactions.
// Transactions are also saved in the queue when it's being sent, and when
// persistent storage is used for the queue, after restarting the node
// dataposter will pick up where it left.
// DataPoster must be RLP serializable and deserializable
type DataPoster struct {
	stopwaiter.StopWaiter
	headerReader      *headerreader.HeaderReader
	client            *ethclient.Client
	auth              *bind.TransactOpts
	signer            externalsigner.SignerFn
	config            config.ConfigFetcher
	usingNoOpStorage  bool
	metadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)
	extraBacklog      func() uint64
	parentChainID256  *uint256.Int
	parentChain       *parent.ParentChain

	internalState *state.InternalState

	maxFeeCapExpression *govaluate.EvaluableExpression
}

type DataPosterOpts struct {
	Database          ethdb.Database
	HeaderReader      *headerreader.HeaderReader
	Auth              *bind.TransactOpts
	RedisClient       redis.UniversalClient
	Config            config.ConfigFetcher
	MetadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)
	ExtraBacklog      func() uint64
	RedisKey          string // Redis storage key
	ParentChain       *parent.ParentChain
}

func NewDataPoster(ctx context.Context, opts *DataPosterOpts) (*DataPoster, error) {
	cfg := opts.Config()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	useNoOpStorage := cfg.UseNoOpStorage
	if opts.HeaderReader.IsParentChainArbitrum() && !cfg.UseNoOpStorage {
		useNoOpStorage = true
		log.Info("Disabling data poster storage, as parent chain appears to be an Arbitrum chain without a mempool")
	}
	encF := func() storage.EncoderDecoderInterface {
		if opts.Config().LegacyStorageEncoding {
			return &storage.LegacyEncoderDecoder{}
		}
		return &storage.EncoderDecoder{}
	}
	var queue state.QueueStorage
	switch {
	case useNoOpStorage:
		queue = &noop.Storage{}
	case opts.RedisClient != nil:
		var err error
		queue, err = redisstorage.NewStorage(opts.RedisClient, opts.RedisKey, &cfg.RedisSigner, encF)
		if err != nil {
			return nil, err
		}
	case cfg.UseDBStorage:
		queue = dbstorage.New(opts.Database, func() storage.EncoderDecoderInterface { return &storage.EncoderDecoder{} })
	default:
		queue = slice.NewStorage(func() storage.EncoderDecoderInterface { return &storage.EncoderDecoder{} })
	}
	if cfg.Dangerous.ClearDBStorage {
		log.Warn("clearing dataposter queue", "flag", "--data-poster.dangerous.clear-dbstorage")
		if err := queue.PruneAll(ctx); err != nil {
			return nil, fmt.Errorf("clearing dataposter queue: %w", err)
		}
	}
	expression, err := govaluate.NewEvaluableExpression(cfg.MaxFeeCapFormula)
	if err != nil {
		return nil, fmt.Errorf("error creating govaluate evaluable expression for calculating maxFeeCap: %w", err)
	}
	dp := &DataPoster{
		headerReader: opts.HeaderReader,
		client:       opts.HeaderReader.Client(),
		auth:         opts.Auth,
		signer: func(_ context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return opts.Auth.Signer(addr, tx)
		},
		config:              opts.Config,
		usingNoOpStorage:    useNoOpStorage,
		metadataRetriever:   opts.MetadataRetriever,
		internalState:       state.NewInternalState(queue),
		maxFeeCapExpression: expression,
		extraBacklog:        opts.ExtraBacklog,
		parentChain:         opts.ParentChain,
	}
	var overflow bool
	dp.parentChainID256, overflow = uint256.FromBig(opts.ParentChain.ChainID)
	if overflow {
		return nil, fmt.Errorf("parent chain ID %v overflows uint256 (necessary for blob transactions)", opts.ParentChain.ChainID)
	}
	if dp.extraBacklog == nil {
		dp.extraBacklog = func() uint64 { return 0 }
	}
	if cfg.ExternalSigner.URL != "" {
		xsign, err := externalsigner.NewExternalSigner(ctx, &cfg.ExternalSigner)
		if err != nil {
			return nil, err
		}
		dp.signer = xsign.Signer
		dp.auth = xsign.TxOpts()
	}

	return dp, nil
}

func (p *DataPoster) Client() *ethclient.Client {
	return p.client
}

func (p *DataPoster) Config() *config.DataPosterConfig {
	return p.config()
}

func (p *DataPoster) ExtraBacklog() uint64 {
	return p.extraBacklog()
}

func (p *DataPoster) MaxFeeCapExpression() *govaluate.EvaluableExpression {
	return p.maxFeeCapExpression
}

func (p *DataPoster) ParentChain() *parent.ParentChain {
	return p.parentChain
}

func (p *DataPoster) ParentChainID256() *uint256.Int {
	return p.parentChainID256
}

func (p *DataPoster) Auth() *bind.TransactOpts {
	return p.auth
}

func (p *DataPoster) Sender() common.Address {
	return p.auth.From
}

func (p *DataPoster) Signer(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	return p.signer(ctx, addr, tx)
}

func (p *DataPoster) MaxMempoolTransactions() uint64 {
	if p.usingNoOpStorage {
		return 1
	}
	config := p.config()
	return arbmath.MinInt(config.MaxMempoolTransactions, config.MaxMempoolWeight)
}

func (p *DataPoster) UsingNoOpStorage() bool {
	return p.usingNoOpStorage
}

func (p *DataPoster) HeaderReader() *headerreader.HeaderReader {
	return p.headerReader
}

func (p *DataPoster) InternalState() *state.InternalState {
	return p.internalState
}

func (p *DataPoster) RetrieveMetadata(ctx context.Context, blockNum *big.Int) ([]byte, error) {
	return p.metadataRetriever(ctx, blockNum)
}

// Tries to acquire redis lock, updates balance and nonce,
func (p *DataPoster) Start(ctxIn context.Context) {
	p.StopWaiter.Start(ctxIn, p)
	p.CallIteratively(p.tick)
}
