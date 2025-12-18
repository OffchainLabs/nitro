// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package timeboost

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
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

	// Auctioneer coordination key for failover
	AUCTIONEER_CHOSEN_KEY = "auctioneer.chosen"

	// Default buffer size for bids receiver channel
	DefaultBidsReceiverBufferSize = 100_000
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
	BidsReceiverBufferSize    uint64                   `koanf:"bids-receiver-buffer-size"`
	S3Storage                 S3StorageServiceConfig   `koanf:"s3-storage"`
}

var DefaultAuctioneerConsumerConfig = pubsub.ConsumerConfig{
	ResponseEntryTimeout: time.Minute * 5,
	// Messages with no heartbeat for over 1s will be reclaimed by the auctioneer
	IdletimeToAutoclaim: time.Second,
	Retry:               true,
	MaxRetryCount:       -1,
}

var DefaultAuctioneerServerConfig = AuctioneerServerConfig{
	Enable:                    true,
	RedisURL:                  "",
	ConsumerConfig:            DefaultAuctioneerConsumerConfig,
	StreamTimeout:             10 * time.Minute,
	AuctionResolutionWaitTime: 2 * time.Second,
	BidsReceiverBufferSize:    DefaultBidsReceiverBufferSize,
	S3Storage:                 DefaultS3StorageServiceConfig,
}

var TestAuctioneerServerConfig = AuctioneerServerConfig{
	Enable:                    true,
	RedisURL:                  "",
	ConsumerConfig:            DefaultAuctioneerConsumerConfig,
	StreamTimeout:             time.Minute,
	AuctionResolutionWaitTime: 2 * time.Second,
	BidsReceiverBufferSize:    1_000,
}

func AuctioneerServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultAuctioneerServerConfig.Enable, "enable auctioneer server")
	f.String(prefix+".redis-url", DefaultAuctioneerServerConfig.RedisURL, "url of redis server to receive bids from bid validators")
	pubsub.ConsumerConfigAddOptionsWithDefaults(prefix+".consumer-config", f, DefaultAuctioneerConsumerConfig)
	f.Duration(prefix+".stream-timeout", DefaultAuctioneerServerConfig.StreamTimeout, "Timeout on polling for existence of redis streams")
	genericconf.WalletConfigAddOptions(prefix+".wallet", f, "wallet for auctioneer server")
	f.String(prefix+".sequencer-endpoint", DefaultAuctioneerServerConfig.SequencerEndpoint, "sequencer RPC endpoint")
	f.String(prefix+".sequencer-jwt-path", DefaultAuctioneerServerConfig.SequencerJWTPath, "sequencer jwt file path")
	f.Bool(prefix+".use-redis-coordinator", DefaultAuctioneerServerConfig.UseRedisCoordinator, "use redis coordinator to find active sequencer")
	f.String(prefix+".redis-coordinator-url", DefaultAuctioneerServerConfig.RedisCoordinatorURL, "redis coordinator url for finding active sequencer")
	f.String(prefix+".auction-contract-address", DefaultAuctioneerServerConfig.AuctionContractAddress, "express lane auction contract address")
	f.String(prefix+".db-directory", DefaultAuctioneerServerConfig.DbDirectory, "path to database directory for persisting validated bids in a sqlite file")
	f.Duration(prefix+".auction-resolution-wait-time", DefaultAuctioneerServerConfig.AuctionResolutionWaitTime, "wait time after auction closing before resolving the auction")
	f.Uint64(prefix+".bids-receiver-buffer-size", DefaultAuctioneerServerConfig.BidsReceiverBufferSize, fmt.Sprintf("buffer size for the bids receiver channel (0 = use default of %d)", DefaultBidsReceiverBufferSize))
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
	unackedBidsMutex               sync.Mutex
	unackedBids                    map[string]*pubsub.Message[*JsonValidatedBid]

	// Coordination fields
	redisClient               redis.UniversalClient
	myId                      string
	isPrimary                 atomic.Bool
	lastPrimaryStatus         bool // Track state changes for logging
	auctioneerLivenessTimeout time.Duration
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
	c.EnableDeterministicReprocessing()

	var endpointManager SequencerEndpointManager
	if cfg.UseRedisCoordinator {
		redisCoordinator, err := redisutil.NewRedisCoordinator(cfg.RedisCoordinatorURL, 1)
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

	bufferSize := cfg.BidsReceiverBufferSize
	if bufferSize == 0 {
		bufferSize = DefaultBidsReceiverBufferSize
	}
	if bufferSize > uint64(math.MaxInt) {
		return nil, fmt.Errorf("bids receiver buffer size %d exceeds maximum int value", bufferSize)
	}

	// Generate unique ID for this auctioneer instance
	myId := fmt.Sprintf("auctioneer-%s-%d",
		uuid.New().String()[:8], // Short UUID
		time.Now().UnixNano())   // Timestamp for uniqueness

	log.Info("Auctioneer coordinator initialized", "id", myId)

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
		bidsReceiver:                   make(chan *JsonValidatedBid, int(bufferSize)),
		bidCache:                       newBidCache(domainSeparator),
		roundTimingInfo:                *roundTimingInfo,
		auctionResolutionWaitTime:      cfg.AuctionResolutionWaitTime,
		streamTimeout:                  cfg.StreamTimeout,
		unackedBids:                    make(map[string]*pubsub.Message[*JsonValidatedBid]),
		redisClient:                    redisClient,
		myId:                           myId,
		auctioneerLivenessTimeout:      cfg.ConsumerConfig.IdletimeToAutoclaim * 3,
	}, nil
}

func (a *AuctioneerServer) consumeNextBid(ctx context.Context) time.Duration {
	// Only consume if we're primary
	if !a.isPrimary.Load() {
		return 250 * time.Millisecond
	}

	req, err := a.consumer.Consume(ctx)
	if err != nil {
		log.Error("Consuming request", "error", err)
		return 0
	}
	if req == nil {
		// There's nothing in the queue.
		return time.Millisecond * 250
	}

	if err := validateBidTimeConstraints(&a.roundTimingInfo, (uint64)(req.Value.Round)); err != nil {
		log.Info("Consumed bid that was no longer valid, skipping", "err", err, "msgId", req.ID)
		req.Ack()
		if errerr := a.consumer.SetError(ctx, req.ID, err.Error()); errerr != nil {
			log.Warn("Error setting error response to bid", "err", err, "msgId", req.ID)
			// We tried, all we can do here is warn.
			// It will be cleaned up by the Consumer
			// on the next try or ultimately by
			// Producer.clearMessages after RequestTimeout
		}
		return 0
	}

	// We use Redis streams to keep the message until the round ends in
	// case the auctioneer dies mid round. On restart Consume will
	// fetch any messages that weren't used to resolve an auction yet.
	a.unackedBidsMutex.Lock()

	// If the heartbeat is slow, it's possible to re-consume the same
	// bid, so we handle that here.
	if _, ok := a.unackedBids[req.ID]; ok {
		a.unackedBidsMutex.Unlock()
		log.Info("Duplicate bid, skipping", "id", req.ID)
		// Ack() stops the heartbeat goroutine created by the above
		// invocation of Consume. This is OK since the original
		// heartbeat goroutine for the unacked bid is still running,
		// and will be stopped at auction end.
		req.Ack()

		// Importantly we don't want to send duplicate bids to
		// the bidsReceiver since it cares about the ordering.
		return 0
	}

	a.unackedBids[req.ID] = req
	a.unackedBidsMutex.Unlock()

	// Forward the message over a channel for processing elsewhere in
	// another thread, so as to not block this consumption thread.
	a.bidsReceiver <- req.Value

	return 0
}

// updateCoordination manages the primary/secondary status of this auctioneer
func (a *AuctioneerServer) updateCoordination(ctx context.Context) time.Duration {
	var success bool
	candidateValue := fmt.Sprintf("%s:%d", a.myId, time.Now().UnixMilli())
	storedValue, err := a.redisClient.Get(ctx, AUCTIONEER_CHOSEN_KEY).Result()
	if err == nil {
		parts := strings.SplitN(storedValue, ":", 2)
		var storedId string
		var storedTimestamp int64

		if len(parts) == 2 {
			storedId = parts[0]
			storedTimestamp, _ = strconv.ParseInt(parts[1], 10, 64)
		} else {
			log.Error("AUCTIONEER_CHOSEN_KEY in wrong format, deleting it before proceeding", "value", candidateValue)
			_, _ = a.redisClient.Del(ctx, AUCTIONEER_CHOSEN_KEY).Result()
			return a.auctioneerLivenessTimeout / 6
		}

		if storedId == a.myId {
			log.Trace("Refreshing our lock", "id", a.myId)
			err = a.redisClient.Set(ctx, AUCTIONEER_CHOSEN_KEY, candidateValue, a.auctioneerLivenessTimeout).Err()
			success = err == nil
		} else {
			elapsed := time.Now().UnixMilli() - storedTimestamp
			if elapsed > a.auctioneerLivenessTimeout.Milliseconds() {
				log.Trace("Lock is stale, deleting and trying to acquire", "id", a.myId, "storedId", storedId, "elapsedMs", elapsed)
				if delErr := a.redisClient.Del(ctx, AUCTIONEER_CHOSEN_KEY).Err(); delErr != nil {
					log.Error("Error deleting stale lock key",
						"id", a.myId,
						"key", AUCTIONEER_CHOSEN_KEY,
						"error", delErr,
						"storedId", storedId,
						"storedTimestamp", storedTimestamp,
						"elapsedMs", elapsed)
				} else {
					// Try to acquire with SetNX
					success = a.redisClient.SetNX(ctx, AUCTIONEER_CHOSEN_KEY, candidateValue, a.auctioneerLivenessTimeout).Val()
					if success {
						log.Info("Successfully acquired stale lock", "id", a.myId)
					} else {
						log.Info("Failed to acquire after deleting stale lock (lost race)", "id", a.myId)
					}
				}
			} else {
				log.Trace("Lock held by someone else", "id", a.myId, "current", storedId, "remainingMs", a.auctioneerLivenessTimeout.Milliseconds()-elapsed)
			}
		}
	} else if errors.Is(err, redis.Nil) {
		log.Trace("Lock is free, trying to acquire", "id", a.myId)
		success = a.redisClient.SetNX(ctx, AUCTIONEER_CHOSEN_KEY, candidateValue, a.auctioneerLivenessTimeout).Val()
		if success {
			log.Info("Successfully acquired lock", "id", a.myId)
		} else {
			log.Info("Failed to acquire lock (other auctioneer won the race)", "id", a.myId)
		}
	} else {
		log.Warn("Redis error when checking lock", "id", a.myId, "err", err)
	}

	if success != a.lastPrimaryStatus {
		if success {
			log.Info("Became primary auctioneer", "id", a.myId)
		} else {
			log.Info("No longer primary auctioneer", "id", a.myId)
		}
		a.lastPrimaryStatus = success
	}
	a.isPrimary.Store(success)

	// Refresh more frequently than expiry, with defaults is this is every 500ms.
	// Needs to be parameterized rather than hardcoded for tests which run more quickly.
	return a.auctioneerLivenessTimeout / 6
}

func (a *AuctioneerServer) Start(ctx_in context.Context) {
	a.StopWaiter.Start(ctx_in, a)
	// Start S3 storage service to persist validated bids to s3
	if a.s3StorageService != nil {
		a.s3StorageService.Start(ctx_in)
	}

	// Start coordination to manage primary/secondary status
	a.StopWaiter.CallIteratively(a.updateCoordination)

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
				log.Info("Context done while checking redis stream existence", "error", ctx.Err().Error())
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
		a.StopWaiter.CallIteratively(a.consumeNextBid)
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
	if first == nil { // No bids received
		if second == nil {
			log.Info("No bids received for auction resolution", "round", upcomingRound)
			return nil
		} else {
			return errors.New("invalid auctionResult, first place bid is not present but second place bid is") // this should ideally never happen
		}
	}

	// Once resolveAuction returns, we acknowledge all bids to remove them from redis.
	// We remove them unconditionally, since resolveAuction retries until the round ends,
	// and there is no way to use them after the round ends.
	defer a.acknowledgeAllBids(ctx, upcomingRound)

	sequencerRpc, newRpc, err := a.endpointManager.GetSequencerRPC(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sequencer RPC: %w", err)
	}

	if newRpc {
		a.auctionContract, err = express_lane_auctiongen.NewExpressLaneAuction(a.auctionContractAddr, ethclient.NewClient(sequencerRpc))
		if err != nil {
			return fmt.Errorf("failed to recreate ExpressLaneAuction contract bindings with new sequencer endpoint: %w", err)
		}
	}

	makeAuctionResolutionTx := func(onRetry bool) (*types.Transaction, error) {
		opts := copyTxOpts(a.txOpts)
		opts.GasMargin = 2000 // Add a 20% buffer to GasLimit to avoid running out of gas
		opts.NoSend = true

		// Both bids are present
		if second != nil {
			if !onRetry {
				FirstBidValueGauge.Update(first.Amount.Int64())
				SecondBidValueGauge.Update(second.Amount.Int64())
				log.Info("Resolving auction with two bids", "round", upcomingRound)
			}
			return a.auctionContract.ResolveMultiBidAuction(
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
		}

		// Single bid is present
		if !onRetry {
			FirstBidValueGauge.Update(first.Amount.Int64())
			log.Info("Resolving auction with single bid", "round", upcomingRound)
		}
		return a.auctionContract.ResolveSingleBidAuction(
			opts,
			express_lane_auctiongen.Bid{
				ExpressLaneController: first.ExpressLaneController,
				Amount:                first.Amount,
				Signature:             first.Signature,
			},
		)
	}

	var tx *types.Transaction
	var receipt *types.Receipt
	tx, err = makeAuctionResolutionTx(false)
	if err != nil {
		log.Error("Error resolving auction", "error", err)
		return err
	}

	roundEndTime := a.roundTimingInfo.TimeOfNextRound()
	retryInterval := 1 * time.Second

	retryLimit := 5
	for retryCount := 0; ; retryCount++ {
		if err = retryUntil(ctx, func() error {
			if err := sequencerRpc.CallContext(ctx, nil, "auctioneer_submitAuctionResolutionTransaction", tx); err != nil {
				log.Error("Error submitting auction resolution to sequencer endpoint", "error", err)
				return err
			}
			return nil
		}, retryInterval, roundEndTime); err != nil {
			return err
		}

		// Wait for the transaction to be mined until this round ends
		waitMinedCtx, cancel := context.WithTimeout(ctx, time.Until(roundEndTime))
		receipt, err = bind.WaitMined(waitMinedCtx, ethclient.NewClient(sequencerRpc), tx)
		cancel()
		if err != nil { // error is only returned when context expires i.e the current round has ended so no point in retrying
			return fmt.Errorf("error waiting for transaction to be mined: %w", err)
		}

		// Check if the transaction was successful
		if tx != nil && receipt != nil && receipt.Status == types.ReceiptStatusSuccessful {
			break
		}
		if tx != nil {
			log.Warn("Transaction failed or did not finalize successfully", "txHash", tx.Hash().Hex())
		}
		if retryCount == retryLimit {
			return errors.New("could not resolve auction after multiple attempts")
		}
		tx, err = makeAuctionResolutionTx(true)
		if err != nil {
			log.Error("Error resolving auction", "error", err)
			return err
		}
	}

	log.Info("Auction resolved successfully", "txHash", tx.Hash().Hex())
	return nil
}

func (a *AuctioneerServer) acknowledgeAllBids(ctx context.Context, round uint64) {
	a.unackedBidsMutex.Lock()
	defer a.unackedBidsMutex.Unlock()

	var acknowledgedCount int
	for msgID, msg := range a.unackedBids {
		bid := msg.Value
		if uint64(bid.Round) <= round {
			msg.Ack() // Stop the heartbeat goroutine

			// SetResult calls XAck to remove the msg from the consumer group's
			// pending list and then removes it from the stream with XDel.
			if err := a.consumer.SetResult(ctx, msgID, nil); err != nil {
				log.Warn("Error marking bid message as consumed by auctioneer", "msgID", msgID, "error", err)
				// We still need delete that bid from unacked bids since
				// it can't be Ack()ed more than once.
				// It will be cleaned up when it's re-read or by the producer
				// after it expires.
			}
			delete(a.unackedBids, msgID)
			acknowledgedCount++
		}
	}

	log.Info("Acknowledged bids in redis stream", "count", acknowledgedCount)
}

// retryUntil retries a given operation defined by the closure until the specified duration
// has passed or the operation succeeds. It waits for the specified retry interval between
// attempts. The function returns an error if all attempts fail.
func retryUntil(ctx context.Context, operation func() error, retryInterval time.Duration, endTime time.Time) error {
	for {
		if time.Now().After(endTime) {
			break
		}

		// Execute the operation
		if err := operation(); err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
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

// IsPrimary returns whether this auctioneer is currently the primary
func (a *AuctioneerServer) IsPrimary() bool {
	return a.isPrimary.Load()
}

// GetId returns the unique identifier for this auctioneer instance
func (a *AuctioneerServer) GetId() string {
	return a.myId
}

func (a *AuctioneerServer) StopAndWait() {
	// The AUCTIONEER_CHOSEN_KEY lock will be considered expired by other auctioneers after
	// auctioneerLivenessTimeout. This timeout gives time for existing messages to become
	// unclaimed after IdleTimeToAutoclaim before the secondary auctioneer starts consuming
	// messages.
	a.StopWaiter.StopAndWait()
	a.consumer.StopAndWait()
}
