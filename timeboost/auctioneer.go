// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// domainValue holds the Keccak256 hash of the string "TIMEBOOST_BID".
// It is intended to be immutable after initialization.
var domainValue []byte

const (
	AuctioneerNamespace      = "auctioneer"
	validatedBidsRedisStream = "validated_bids"
)

var (
	receivedBidsCounter  = metrics.NewRegisteredCounter("arb/auctioneer/bids/received", nil)
	validatedBidsCounter = metrics.NewRegisteredCounter("arb/auctioneer/bids/validated", nil)
	FirstBidValueGauge   = metrics.NewRegisteredGauge("arb/auctioneer/bids/firstbidvalue", nil)
	SecondBidValueGauge  = metrics.NewRegisteredGauge("arb/auctioneer/bids/secondbidvalue", nil)
)

func init() {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte("TIMEBOOST_BID"))
	domainValue = hash.Sum(nil)
}

type AuctioneerServerConfigFetcher func() *AuctioneerServerConfig

type AuctioneerServerConfig struct {
	Enable         bool                  `koanf:"enable"`
	RedisURL       string                `koanf:"redis-url"`
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Timeout on polling for existence of each redis stream.
	StreamTimeout             time.Duration            `koanf:"stream-timeout"`
	Wallet                    genericconf.WalletConfig `koanf:"wallet"`
	SequencerEndpoint         string                   `koanf:"sequencer-endpoint"`
	SequencerJWTPath          string                   `koanf:"sequencer-jwt-path"`
	UseRedisCoordinator       bool                     `koanf:"use-redis-coordinator"`
	RedisCoordinatorURL       string                   `koanf:"redis-coordinator-url"`
	AuctionContractAddress    string                   `koanf:"auction-contract-address"`
	DbDirectory               string                   `koanf:"db-directory"`
	AuctionResolutionWaitTime time.Duration            `koanf:"auction-resolution-wait-time"`
	S3Storage                 S3StorageServiceConfig   `koanf:"s3-storage"`
}

var DefaultAuctioneerServerConfig = AuctioneerServerConfig{
	Enable:                    true,
	RedisURL:                  "",
	ConsumerConfig:            pubsub.DefaultConsumerConfig,
	StreamTimeout:             10 * time.Minute,
	AuctionResolutionWaitTime: 2 * time.Second,
	S3Storage:                 DefaultS3StorageServiceConfig,
}

var TestAuctioneerServerConfig = AuctioneerServerConfig{
	Enable:                    true,
	RedisURL:                  "",
	ConsumerConfig:            pubsub.TestConsumerConfig,
	StreamTimeout:             time.Minute,
	AuctionResolutionWaitTime: 2 * time.Second,
}

func AuctioneerServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultAuctioneerServerConfig.Enable, "enable auctioneer server")
	f.String(prefix+".redis-url", DefaultAuctioneerServerConfig.RedisURL, "url of redis server to receive bids from bid validators")
	pubsub.ConsumerConfigAddOptions(prefix+".consumer-config", f)
	f.Duration(prefix+".stream-timeout", DefaultAuctioneerServerConfig.StreamTimeout, "Timeout on polling for existence of redis streams")
	genericconf.WalletConfigAddOptions(prefix+".wallet", f, "wallet for auctioneer server")
	f.String(prefix+".sequencer-endpoint", DefaultAuctioneerServerConfig.SequencerEndpoint, "sequencer RPC endpoint")
	f.String(prefix+".sequencer-jwt-path", DefaultAuctioneerServerConfig.SequencerJWTPath, "sequencer jwt file path")
	f.Bool(prefix+".use-redis-coordinator", DefaultAuctioneerServerConfig.UseRedisCoordinator, "use redis coordinator to find active sequencer")
	f.String(prefix+".redis-coordinator-url", DefaultAuctioneerServerConfig.RedisCoordinatorURL, "redis coordinator url for finding active sequencer")
	f.String(prefix+".auction-contract-address", DefaultAuctioneerServerConfig.AuctionContractAddress, "express lane auction contract address")
	f.String(prefix+".db-directory", DefaultAuctioneerServerConfig.DbDirectory, "path to database directory for persisting validated bids in a sqlite file")
	f.Duration(prefix+".auction-resolution-wait-time", DefaultAuctioneerServerConfig.AuctionResolutionWaitTime, "wait time after auction closing before resolving the auction")
	S3StorageServiceConfigAddOptions(prefix+".s3-storage", f)
}

// AuctioneerServer is a struct that represents an autonomous auctioneer.
// It is responsible for receiving bids, validating them, and resolving auctions.
type AuctioneerServer struct {
	stopwaiter.StopWaiter
	consumer                       *pubsub.Consumer[*JsonValidatedBid, error]
	txOpts                         *bind.TransactOpts
	chainId                        *big.Int
	endpointManager                SequencerEndpointManager
	auctionContract                *express_lane_auctiongen.ExpressLaneAuction
	auctionContractAddr            common.Address
	auctionContractDomainSeparator [32]byte
	bidsReceiver                   chan *JsonValidatedBid
	bidCache                       *bidCache
	roundTimingInfo                RoundTimingInfo
	streamTimeout                  time.Duration
	auctionResolutionWaitTime      time.Duration
	database                       *SqliteDatabase
	s3StorageService               *S3StorageService
}

// NewAuctioneerServer creates a new autonomous auctioneer struct.
func NewAuctioneerServer(ctx context.Context, configFetcher AuctioneerServerConfigFetcher) (*AuctioneerServer, error) {
	cfg := configFetcher()
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	if cfg.AuctionContractAddress == "" {
		return nil, fmt.Errorf("auction contract address cannot be empty")
	}
	if cfg.DbDirectory == "" {
		return nil, errors.New("database directory is empty")
	}
	database, err := NewDatabase(cfg.DbDirectory)
	if err != nil {
		return nil, err
	}
	var s3StorageService *S3StorageService
	if cfg.S3Storage.Enable {
		s3StorageService, err = NewS3StorageService(&cfg.S3Storage, database)
		if err != nil {
			return nil, err
		}
	}
	auctionContractAddr := common.HexToAddress(cfg.AuctionContractAddress)
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	c, err := pubsub.NewConsumer[*JsonValidatedBid, error](redisClient, validatedBidsRedisStream, &cfg.ConsumerConfig)
	if err != nil {
		return nil, fmt.Errorf("creating consumer for validation: %w", err)
	}

	var endpointManager SequencerEndpointManager
	if cfg.UseRedisCoordinator {
		redisCoordinator, err := redisutil.NewRedisCoordinator(cfg.RedisCoordinatorURL)
		if err != nil {
			return nil, err
		}
		endpointManager = NewRedisEndpointManager(redisCoordinator, cfg.SequencerJWTPath)
	} else {
		endpointManager = NewStaticEndpointManager(cfg.SequencerEndpoint, cfg.SequencerJWTPath)
	}

	rpcClient, _, err := endpointManager.GetSequencerRPC(ctx)
	if err != nil {
		return nil, err
	}
	sequencerClient := ethclient.NewClient(rpcClient)

	chainId, err := sequencerClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	txOpts, _, err := util.OpenWallet("auctioneer-server", &cfg.Wallet, chainId)
	if err != nil {
		return nil, errors.Wrap(err, "opening wallet")
	}
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, sequencerClient)
	if err != nil {
		return nil, err
	}
	domainSeparator, err := auctionContract.DomainSeparator(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	roundTimingInfo, err := NewRoundTimingInfo(rawRoundTimingInfo)
	if err != nil {
		return nil, err
	}
	if err = roundTimingInfo.ValidateResolutionWaitTime(cfg.AuctionResolutionWaitTime); err != nil {
		return nil, err
	}
	return &AuctioneerServer{
		txOpts:                         txOpts,
		endpointManager:                endpointManager,
		chainId:                        chainId,
		database:                       database,
		s3StorageService:               s3StorageService,
		consumer:                       c,
		auctionContract:                auctionContract,
		auctionContractAddr:            auctionContractAddr,
		auctionContractDomainSeparator: domainSeparator,
		bidsReceiver:                   make(chan *JsonValidatedBid, 100_000), // TODO(Terence): Is 100k enough? Make this configurable?
		bidCache:                       newBidCache(domainSeparator),
		roundTimingInfo:                *roundTimingInfo,
		auctionResolutionWaitTime:      cfg.AuctionResolutionWaitTime,
	}, nil
}

func (a *AuctioneerServer) Start(ctx_in context.Context) {
	a.StopWaiter.Start(ctx_in, a)
	// Start S3 storage service to persist validated bids to s3
	if a.s3StorageService != nil {
		a.s3StorageService.Start(ctx_in)
	}
	// Channel that consumer uses to indicate its readiness.
	readyStream := make(chan struct{}, 1)
	a.consumer.Start(ctx_in)
	// Channel for single consumer, once readiness is indicated in this,
	// consumer will start consuming iteratively.
	ready := make(chan struct{}, 1)
	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			if pubsub.StreamExists(ctx, a.consumer.StreamName(), a.consumer.RedisClient()) {
				ready <- struct{}{}
				readyStream <- struct{}{}
				return
			}
			select {
			case <-ctx.Done():
				log.Info("Context done while checking redis stream existance", "error", ctx.Err().Error())
				return
			case <-time.After(time.Millisecond * 100):
			}
		}
	})
	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		select {
		case <-ctx.Done():
			log.Info("Context done while waiting a redis stream to be ready", "error", ctx.Err().Error())
			return
		case <-ready: // Wait until the stream exists and start consuming iteratively.
		}
		log.Info("Stream exists, now attempting to consume data from it")
		a.StopWaiter.CallIteratively(func(ctx context.Context) time.Duration {
			req, err := a.consumer.Consume(ctx)
			if err != nil {
				log.Error("Consuming request", "error", err)
				return 0
			}
			if req == nil {
				// There's nothing in the queue.
				return time.Millisecond * 250
			}
			// Forward the message over a channel for processing elsewhere in
			// another thread, so as to not block this consumption thread.
			a.bidsReceiver <- req.Value

			// We received the message, then we ack with a nil error.
			if err := a.consumer.SetResult(ctx, req.ID, nil); err != nil {
				log.Error("Error setting result for request", "id", req.ID, "result", nil, "error", err)
				return 0
			}
			req.Ack()
			return 0
		})
	})
	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-readyStream:
				log.Trace("At least one stream is ready")
				return // Don't block Start if at least one of the stream is ready.
			case <-time.After(a.streamTimeout):
				log.Error("Waiting for redis streams timed out")
				return
			case <-ctx.Done():
				log.Info("Context done while waiting redis streams to be ready, failed to start")
				return
			}
		}
	})

	// Bid receiver thread.
	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case bid := <-a.bidsReceiver:
				log.Info("Consumed validated bid", "bidder", bid.Bidder, "amount", bid.Amount, "round", bid.Round)
				a.bidCache.add(JsonValidatedBidToGo(bid))
				// Persist the validated bid to the database as a non-blocking operation.
				go a.persistValidatedBid(bid)
			case <-ctx.Done():
				log.Info("Context done while waiting redis streams to be ready, failed to start")
				return
			}
		}
	})

	// Auction resolution thread.
	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		ticker := newRoundTicker(a.roundTimingInfo)
		go ticker.tickAtAuctionClose()
		for {
			select {
			case <-ctx.Done():
				log.Error("Context closed, autonomous auctioneer shutting down")
				return
			case auctionClosingTime := <-ticker.c:
				log.Info("New auction closing time reached", "closingTime", auctionClosingTime, "totalBids", a.bidCache.size())
				time.Sleep(a.auctionResolutionWaitTime)
				if err := a.resolveAuction(ctx); err != nil {
					log.Error("Could not resolve auction for round", "error", err)
				}
				// Clear the bid cache.
				a.bidCache = newBidCache(a.auctionContractDomainSeparator)
			}
		}
	})
}

// Resolves the auction by calling the smart contract with the top two bids.
func (a *AuctioneerServer) resolveAuction(ctx context.Context) error {
	upcomingRound := a.roundTimingInfo.RoundNumber() + 1
	result := a.bidCache.topTwoBids()
	first := result.firstPlace
	second := result.secondPlace
	var tx *types.Transaction
	var err error
	opts := copyTxOpts(a.txOpts)
	opts.NoSend = true

	sequencerRpc, newRpc, err := a.endpointManager.GetSequencerRPC(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sequencer RPC: %w", err)
	}

	if newRpc {
		a.auctionContract, err = express_lane_auctiongen.NewExpressLaneAuction(a.auctionContractAddr, ethclient.NewClient(sequencerRpc))
		if err != nil {
			return fmt.Errorf("failed to recreate ExpressLaneAuction conctract bindings with new sequencer endpoint: %w", err)
		}
	}

	switch {
	case first != nil && second != nil: // Both bids are present
		tx, err = a.auctionContract.ResolveMultiBidAuction(
			opts,
			express_lane_auctiongen.Bid{
				ExpressLaneController: first.ExpressLaneController,
				Amount:                first.Amount,
				Signature:             first.Signature,
			},
			express_lane_auctiongen.Bid{
				ExpressLaneController: second.ExpressLaneController,
				Amount:                second.Amount,
				Signature:             second.Signature,
			},
		)
		FirstBidValueGauge.Update(first.Amount.Int64())
		SecondBidValueGauge.Update(second.Amount.Int64())
		log.Info("Resolving auction with two bids", "round", upcomingRound)

	case first != nil: // Single bid is present
		tx, err = a.auctionContract.ResolveSingleBidAuction(
			opts,
			express_lane_auctiongen.Bid{
				ExpressLaneController: first.ExpressLaneController,
				Amount:                first.Amount,
				Signature:             first.Signature,
			},
		)
		FirstBidValueGauge.Update(first.Amount.Int64())
		log.Info("Resolving auction with single bid", "round", upcomingRound)

	case second == nil: // No bids received
		log.Info("No bids received for auction resolution", "round", upcomingRound)
		return nil
	}
	if err != nil {
		log.Error("Error resolving auction", "error", err)
		return err
	}

	roundEndTime := a.roundTimingInfo.TimeOfNextRound()
	retryInterval := 1 * time.Second

	if err := retryUntil(ctx, func() error {
		if err := sequencerRpc.CallContext(ctx, nil, "auctioneer_submitAuctionResolutionTransaction", tx); err != nil {
			log.Error("Error submitting auction resolution to sequencer endpoint", "error", err)
			return err
		}

		// Wait for the transaction to be mined
		receipt, err := bind.WaitMined(ctx, ethclient.NewClient(sequencerRpc), tx)
		if err != nil {
			log.Error("Error waiting for transaction to be mined", "error", err)
			return err
		}

		// Check if the transaction was successful
		if tx == nil || receipt == nil || receipt.Status != types.ReceiptStatusSuccessful {
			if tx != nil {
				log.Error("Transaction failed or did not finalize successfully", "txHash", tx.Hash().Hex())
			}
			return errors.New("transaction failed or did not finalize successfully")
		}

		return nil
	}, retryInterval, roundEndTime); err != nil {
		return err
	}

	log.Info("Auction resolved successfully", "txHash", tx.Hash().Hex())
	return nil
}

// retryUntil retries a given operation defined by the closure until the specified duration
// has passed or the operation succeeds. It waits for the specified retry interval between
// attempts. The function returns an error if all attempts fail.
func retryUntil(ctx context.Context, operation func() error, retryInterval time.Duration, endTime time.Time) error {
	for {
		// Execute the operation
		if err := operation(); err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if time.Now().After(endTime) {
			break
		}

		time.Sleep(retryInterval)
	}
	return errors.New("operation failed after multiple attempts")
}

func (a *AuctioneerServer) persistValidatedBid(bid *JsonValidatedBid) {
	if err := a.database.InsertBid(JsonValidatedBidToGo(bid)); err != nil {
		log.Error("Could not persist validated bid to database", "err", err, "bidder", bid.Bidder, "amount", bid.Amount.String())
	}
}

func copyTxOpts(opts *bind.TransactOpts) *bind.TransactOpts {
	if opts == nil {
		return nil
	}
	copied := &bind.TransactOpts{
		From:     opts.From,
		Context:  opts.Context,
		NoSend:   opts.NoSend,
		Signer:   opts.Signer,
		GasLimit: opts.GasLimit,
	}

	if opts.Nonce != nil {
		copied.Nonce = new(big.Int).Set(opts.Nonce)
	}
	if opts.Value != nil {
		copied.Value = new(big.Int).Set(opts.Value)
	}
	if opts.GasPrice != nil {
		copied.GasPrice = new(big.Int).Set(opts.GasPrice)
	}
	if opts.GasFeeCap != nil {
		copied.GasFeeCap = new(big.Int).Set(opts.GasFeeCap)
	}
	if opts.GasTipCap != nil {
		copied.GasTipCap = new(big.Int).Set(opts.GasTipCap)
	}
	return copied
}
