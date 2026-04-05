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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"

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

	DefaultBidsReceiverBufferSize = 100_000

	defaultPersistBidsBufferSize = 128
	defaultConsumeInterval       = 250 * time.Millisecond
	maxConsumeBackoff            = 30 * time.Second
	maxAuctionResolutionRetries  = 5
)

var (
	receivedBidsCounter             = metrics.NewRegisteredCounter("arb/auctioneer/bids/received", nil)
	validatedBidsCounter            = metrics.NewRegisteredCounter("arb/auctioneer/bids/validated", nil)
	persistBidFailureCounter        = metrics.NewRegisteredCounter("arb/auctioneer/bids/persist_failures", nil)
	persistBidChannelFullCounter    = metrics.NewRegisteredCounter("arb/auctioneer/bids/persist_channel_full", nil)
	auctionResolutionFailureCounter = metrics.NewRegisteredCounter("arb/auctioneer/auction/resolution_failures", nil)
	bidAckFailureCounter            = metrics.NewRegisteredCounter("arb/auctioneer/bids/ack_failures", nil)
	reserveOriginatorBidCounter     = metrics.NewRegisteredCounter("arb/auctioneer/reserve_originator/bid", nil)
	reserveOriginatorWonCounter     = metrics.NewRegisteredCounter("arb/auctioneer/reserve_originator/won", nil)
	FirstBidValueGauge              = metrics.NewRegisteredGauge("arb/auctioneer/bids/firstbidvalue", nil)
	SecondBidValueGauge             = metrics.NewRegisteredGauge("arb/auctioneer/bids/secondbidvalue", nil)
)

func init() {
	domainValue = crypto.Keccak256([]byte("TIMEBOOST_BID"))
}

type AuctioneerServerConfigFetcher func() *AuctioneerServerConfig

type AuctioneerServerConfig struct {
	Enable         bool                  `koanf:"enable"`
	RedisURL       string                `koanf:"redis-url"`
	ConsumerConfig pubsub.ConsumerConfig `koanf:"consumer-config"`
	// Maximum time to wait for the redis stream to exist before treating startup as failed.
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
	ReserveOriginatorAddress  string                   `koanf:"reserve-originator-address"`
}

func (c *AuctioneerServerConfig) Validate() error {
	if c.AuctionContractAddress != "" && !common.IsHexAddress(c.AuctionContractAddress) {
		return errors.New("invalid auctioneer-server.auction-contract-address")
	}

	if c.ReserveOriginatorAddress != "" {
		if !common.IsHexAddress(c.ReserveOriginatorAddress) {
			return errors.New("invalid auctioneer-server.reserve-originator-address")
		}

		if common.HexToAddress(c.ReserveOriginatorAddress) == (common.Address{}) {
			return errors.New("auctioneer-server.reserve-originator-address cannot be the zero address")
		}
	}

	if err := c.S3Storage.Validate(); err != nil {
		return err
	}
	return nil
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
	f.String(prefix+".reserve-originator-address", DefaultAuctioneerServerConfig.ReserveOriginatorAddress, "reserve originator address, on-chain auction resolution is skipped if this address wins")
	S3StorageServiceConfigAddOptions(prefix+".s3-storage", f)
}

// AuctioneerServer is an autonomous auctioneer.
// It consumes validated bids from a Redis stream and resolves auctions on-chain.
type AuctioneerServer struct {
	stopwaiter.StopWaiter
	consumer        *pubsub.Consumer[*JsonValidatedBid, error]
	txOpts          *bind.TransactOpts
	chainId         *big.Int
	endpointManager SequencerEndpointManager
	// auctionContract may be updated by refreshSequencerEndpoint during failover;
	// both reads and writes happen exclusively on the auction resolution thread.
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
	persistBids                    chan *ValidatedBid
	reserveOriginatorAddr          common.Address
	consumeBackoff                 time.Duration // only accessed from consumeNextBid (single-threaded via CallIteratively)
	consecutiveResolutionFailures  int64         // only accessed from the auction resolution thread

	// Coordination fields
	redisClient               redis.UniversalClient
	myId                      string
	isPrimary                 atomic.Bool
	lastPrimaryStatus         bool          // only accessed from updateCoordination (single-threaded via CallIteratively)
	coordinationBackoff       time.Duration // only accessed from updateCoordination (single-threaded via CallIteratively)
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
	success := false
	defer func() {
		if !success {
			database.Close()
		}
	}()
	var s3StorageService *S3StorageService
	if cfg.S3Storage.Enable {
		s3StorageService, err = NewS3StorageService(&cfg.S3Storage, database)
		if err != nil {
			return nil, err
		}
	}
	auctionContractAddr := common.HexToAddress(cfg.AuctionContractAddress)
	reserveOriginatorAddr := common.HexToAddress(cfg.ReserveOriginatorAddress)
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !success {
			redisClient.Close()
		}
	}()
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
	defer func() {
		if !success {
			endpointManager.Close()
		}
	}()

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

	if cfg.ReserveOriginatorAddress != "" {
		log.Info("ReserveOriginator configured, on-chain resolution skipped when this address wins", "address", reserveOriginatorAddr)
	}

	myId := fmt.Sprintf("auctioneer-%s-%d", uuid.New().String()[:8], time.Now().UnixNano())

	log.Info("Auctioneer coordinator initialized", "id", myId)

	success = true
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
		persistBids:                    make(chan *ValidatedBid, defaultPersistBidsBufferSize),
		bidCache:                       newBidCache(domainSeparator),
		roundTimingInfo:                *roundTimingInfo,
		auctionResolutionWaitTime:      cfg.AuctionResolutionWaitTime,
		streamTimeout:                  cfg.StreamTimeout,
		unackedBids:                    make(map[string]*pubsub.Message[*JsonValidatedBid]),
		redisClient:                    redisClient,
		myId:                           myId,
		auctioneerLivenessTimeout:      cfg.ConsumerConfig.IdletimeToAutoclaim * 3,
		reserveOriginatorAddr:          reserveOriginatorAddr,
		consumeBackoff:                 defaultConsumeInterval,
	}, nil
}

func (a *AuctioneerServer) consumeNextBid(ctx context.Context) time.Duration {
	if !a.isPrimary.Load() {
		a.consumeBackoff = defaultConsumeInterval
		return defaultConsumeInterval
	}

	req, err := a.consumer.Consume(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return 0
		}
		log.Error("Error consuming from Redis stream", "stream", validatedBidsRedisStream, "id", a.myId, "error", err)
		backoff := a.consumeBackoff
		a.consumeBackoff = min(backoff*2, maxConsumeBackoff)
		return backoff
	}
	a.consumeBackoff = defaultConsumeInterval
	if req == nil {
		return defaultConsumeInterval
	}

	if err := validateBidTimeConstraints(&a.roundTimingInfo, uint64(req.Value.Round)); err != nil {
		log.Info("Consumed bid that was no longer valid, skipping", "err", err, "msgId", req.ID)
		req.Ack()
		if redisErr := a.consumer.SetError(ctx, req.ID, err.Error()); redisErr != nil {
			// Best-effort: the Consumer will retry or the Producer will clean up after RequestTimeout.
			log.Warn("Error setting error response to bid", "redisErr", redisErr, "originalErr", err, "msgId", req.ID)
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

		return 0
	}

	a.unackedBids[req.ID] = req
	a.unackedBidsMutex.Unlock()

	// Forward to the bid receiver thread to avoid blocking consumption.
	stallTimer := time.NewTimer(5 * time.Second)
	select {
	case a.bidsReceiver <- req.Value:
		stallTimer.Stop()
	case <-stallTimer.C:
		log.Warn("bidsReceiver channel blocked for 5s, bid consumption is stalled",
			"msgId", req.ID, "bidder", req.Value.Bidder, "round", req.Value.Round,
			"channelCap", cap(a.bidsReceiver))
		select {
		case a.bidsReceiver <- req.Value:
		case <-ctx.Done():
			log.Info("Context cancelled after consuming bid, will be re-consumed on restart", "msgId", req.ID, "bidder", req.Value.Bidder, "round", req.Value.Round)
			return 0
		}
	case <-ctx.Done():
		stallTimer.Stop()
		log.Info("Context cancelled after consuming bid, will be re-consumed on restart", "msgId", req.ID, "bidder", req.Value.Bidder, "round", req.Value.Round)
		return 0
	}

	return 0
}

func parseAuctioneerLockValue(value string) (id string, timestamp int64, ok bool) {
	id, tsStr, found := strings.Cut(value, ":")
	if !found || id == "" {
		return "", 0, false
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil || ts <= 0 {
		return "", 0, false
	}
	return id, ts, true
}

func (a *AuctioneerServer) tryAcquireLock(ctx context.Context, candidateValue, reason string) (acquired, hadError bool) {
	ok, err := a.redisClient.SetNX(ctx, AUCTIONEER_CHOSEN_KEY, candidateValue, a.auctioneerLivenessTimeout).Result()
	if err != nil {
		log.Error("Redis error during lock acquisition", "id", a.myId, "reason", reason, "error", err)
		return false, true
	}
	if ok {
		log.Info("Acquired lock", "id", a.myId, "reason", reason)
	} else {
		log.Trace("Lock not acquired", "id", a.myId, "reason", reason)
	}
	return ok, false
}

// updateCoordination manages the primary/secondary status of this auctioneer
func (a *AuctioneerServer) updateCoordination(ctx context.Context) time.Duration {
	if ctx.Err() != nil {
		return 0
	}
	var success bool
	var hadRedisError bool
	candidateValue := fmt.Sprintf("%s:%d", a.myId, time.Now().UnixMilli())
	storedValue, err := a.redisClient.Get(ctx, AUCTIONEER_CHOSEN_KEY).Result()
	if err == nil {
		storedId, storedTimestamp, valid := parseAuctioneerLockValue(storedValue)
		if !valid {
			log.Error("AUCTIONEER_CHOSEN_KEY has invalid format, deleting before proceeding", "value", storedValue)
			if delErr := a.redisClient.Del(ctx, AUCTIONEER_CHOSEN_KEY).Err(); delErr != nil {
				log.Error("Failed to delete invalid AUCTIONEER_CHOSEN_KEY; no auctioneer can become primary until resolved", "value", storedValue, "error", delErr)
				hadRedisError = true
			}
			a.isPrimary.Store(false)
			if a.lastPrimaryStatus {
				log.Info("No longer primary auctioneer (invalid lock format)", "id", a.myId)
				a.lastPrimaryStatus = false
			}
			return a.coordinationInterval(hadRedisError)
		}

		if storedId == a.myId {
			log.Trace("Refreshing our lock", "id", a.myId)
			err = a.redisClient.Set(ctx, AUCTIONEER_CHOSEN_KEY, candidateValue, a.auctioneerLivenessTimeout).Err()
			if err != nil {
				log.Error("Failed to refresh auctioneer lock in Redis", "id", a.myId, "error", err)
				hadRedisError = true
			}
			// Remain primary even if refresh failed: the lock TTL hasn't
			// expired yet, so no other auctioneer can take over until it does.
			success = true
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
					hadRedisError = true
				} else {
					acquired, hadErr := a.tryAcquireLock(ctx, candidateValue, "stale lock deleted")
					success = acquired
					hadRedisError = hadRedisError || hadErr
				}
			} else {
				log.Trace("Lock held by someone else", "id", a.myId, "current", storedId, "remainingMs", a.auctioneerLivenessTimeout.Milliseconds()-elapsed)
			}
		}
	} else if errors.Is(err, redis.Nil) {
		acquired, hadErr := a.tryAcquireLock(ctx, candidateValue, "lock free")
		success = acquired
		hadRedisError = hadRedisError || hadErr
	} else {
		log.Error("Redis error when checking lock", "id", a.myId, "err", err)
		hadRedisError = true
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

	return a.coordinationInterval(hadRedisError)
}

// coordinationInterval returns the next polling interval for coordination,
// applying exponential backoff on Redis errors and resetting on success.
func (a *AuctioneerServer) coordinationInterval(hadRedisError bool) time.Duration {
	baseInterval := a.auctioneerLivenessTimeout / 6
	if hadRedisError {
		backoff := max(a.coordinationBackoff, baseInterval)
		a.coordinationBackoff = min(backoff*2, a.auctioneerLivenessTimeout)
		return backoff
	}
	a.coordinationBackoff = 0
	return baseInterval
}

func (a *AuctioneerServer) Start(ctx_in context.Context) {
	a.StopWaiter.Start(ctx_in, a)
	// ORDERING MATTERS: s3StorageService is tracked before consumer so that
	// on LIFO shutdown, the consumer stops first (ceasing bid consumption),
	// while s3StorageService gets a chance to upload any remaining persisted bids.
	if a.s3StorageService != nil {
		a.StartAndTrackChild(a.s3StorageService)
	}

	a.StopWaiter.CallIteratively(a.updateCoordination)

	// Channel that consumer uses to indicate its readiness.
	readyStream := make(chan struct{}, 1)
	a.StartAndTrackChild(a.consumer)
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
			timer := time.NewTimer(time.Millisecond * 100)
			select {
			case <-ctx.Done():
				timer.Stop()
				log.Info("Context done while checking redis stream existence", "error", ctx.Err().Error())
				return
			case <-timer.C:
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
		streamTimer := time.NewTimer(a.streamTimeout)
		defer streamTimer.Stop()
		for {
			select {
			case <-readyStream:
				log.Trace("At least one stream is ready")
				return // Don't block Start if at least one of the stream is ready.
			case <-streamTimer.C:
				log.Crit("Waiting for redis streams timed out, auctioneer cannot consume bids", "timeout", a.streamTimeout)
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
				converted := JsonValidatedBidToGo(bid)
				a.bidCache.add(converted)
				if a.reserveOriginatorAddr != (common.Address{}) && converted.Bidder == a.reserveOriginatorAddr {
					reserveOriginatorBidCounter.Inc(1)
				}
				select {
				case a.persistBids <- converted:
				default:
					persistBidChannelFullCounter.Inc(1)
					log.Error("Persistence channel full, bid will not be persisted to database", "bidder", converted.Bidder, "round", converted.Round, "amount", converted.Amount.String())
				}
			case <-ctx.Done():
				log.Info("Bid receiver thread shutting down")
				return
			}
		}
	})

	// Bid persistence worker.
	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case bid := <-a.persistBids:
				a.persistValidatedBid(bid)
			case <-ctx.Done():
				// Drain remaining buffered bids before shutting down.
				for {
					select {
					case bid := <-a.persistBids:
						a.persistValidatedBid(bid)
					default:
						log.Info("Bid persistence worker shut down")
						return
					}
				}
			}
		}
	})

	// Auction resolution thread.
	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		ticker := newRoundTicker(a.roundTimingInfo)
		a.StopWaiter.LaunchThread(func(ctx context.Context) {
			ticker.tickAtAuctionClose(ctx)
		})
		for {
			select {
			case <-ctx.Done():
				log.Info("Context closed, autonomous auctioneer shutting down")
				return
			case auctionClosingTime := <-ticker.c:
				log.Info("New auction closing time reached", "closingTime", auctionClosingTime, "totalBids", a.bidCache.size())
				resolutionTimer := time.NewTimer(a.auctionResolutionWaitTime)
				select {
				case <-resolutionTimer.C:
				case <-ctx.Done():
					resolutionTimer.Stop()
					log.Info("Context cancelled during auction resolution wait")
					return
				}
				upcomingRound := a.roundTimingInfo.RoundNumber() + 1
				if err := a.resolveAuction(ctx, upcomingRound); err != nil {
					a.consecutiveResolutionFailures++
					auctionResolutionFailureCounter.Inc(1)
					log.Error("Auction resolution failed; this round's express lane will remain unassigned",
						"round", upcomingRound, "consecutiveFailures", a.consecutiveResolutionFailures, "error", err)
				} else {
					a.consecutiveResolutionFailures = 0
				}
			}
		}
	})
}

// refreshSequencerEndpoint updates a.auctionContract if the sequencer endpoint
// has changed and returns the current RPC client.
func (a *AuctioneerServer) refreshSequencerEndpoint(ctx context.Context) (*rpc.Client, error) {
	sequencerRpc, isNew, err := a.endpointManager.GetSequencerRPC(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting sequencer RPC: %w", err)
	}
	if isNew {
		newContract, err := express_lane_auctiongen.NewExpressLaneAuction(a.auctionContractAddr, ethclient.NewClient(sequencerRpc))
		if err != nil {
			return nil, fmt.Errorf("creating ExpressLaneAuction contract binding for new sequencer endpoint: %w", err)
		}
		a.auctionContract = newContract
	}
	return sequencerRpc, nil
}

// resolveAuction resolves the auction by submitting the winning bid(s) on-chain.
func (a *AuctioneerServer) resolveAuction(ctx context.Context, upcomingRound uint64) error {
	result := a.bidCache.topTwoBidsAndClear()
	first := result.firstPlace
	second := result.secondPlace

	// Always run on return to stop heartbeat goroutines; Redis cleanup
	// is best-effort (degrades gracefully during shutdown).
	defer a.acknowledgeAllBids(ctx, upcomingRound)

	if first == nil {
		if second == nil {
			log.Info("No bids received for auction resolution", "round", upcomingRound)
			return nil
		}

		return errors.New("invalid auctionResult, first place bid is not present but second place bid is") // this should ideally never happen
	}

	FirstBidValueGauge.Update(first.Amount.Int64())
	if second != nil {
		SecondBidValueGauge.Update(second.Amount.Int64())
	}

	// If ReserveOriginator is configured and it won this round, skip on-chain auction resolution.
	if a.reserveOriginatorAddr != (common.Address{}) && a.reserveOriginatorAddr == first.Bidder {
		reserveOriginatorWonCounter.Inc(1)
		if second != nil {
			log.Info("ReserveOriginator won round, skipping on-chain resolution",
				"round", upcomingRound, "firstBid", first.Amount.String(), "firstBidder", first.Bidder.Hex(),
				"secondBid", second.Amount.String(), "secondBidder", second.Bidder.Hex())
		} else {
			log.Info("ReserveOriginator won round (sole bidder), skipping on-chain resolution",
				"round", upcomingRound, "firstBid", first.Amount.String(), "firstBidder", first.Bidder.Hex())
		}
		return nil
	}

	sequencerRpc, err := a.refreshSequencerEndpoint(ctx)
	if err != nil {
		return err
	}

	if second != nil {
		log.Info("Resolving auction with two bids", "round", upcomingRound)
	} else {
		log.Info("Resolving auction with single bid", "round", upcomingRound)
	}

	firstBid := express_lane_auctiongen.Bid{
		ExpressLaneController: first.ExpressLaneController,
		Amount:                first.Amount,
		Signature:             first.Signature,
	}

	makeAuctionResolutionTx := func(contract *express_lane_auctiongen.ExpressLaneAuction) (*types.Transaction, error) {
		opts := copyTxOpts(a.txOpts)
		opts.GasMargin = 2000 // Add a 20% buffer to GasLimit to avoid running out of gas
		opts.NoSend = true

		if second != nil {
			return contract.ResolveMultiBidAuction(
				opts,
				firstBid,
				express_lane_auctiongen.Bid{
					ExpressLaneController: second.ExpressLaneController,
					Amount:                second.Amount,
					Signature:             second.Signature,
				},
			)
		}

		return contract.ResolveSingleBidAuction(opts, firstBid)
	}

	tx, err := makeAuctionResolutionTx(a.auctionContract)
	if err != nil {
		log.Error("Error building initial auction resolution transaction", "round", upcomingRound, "error", err)
		return err
	}

	roundEndTime := a.roundTimingInfo.TimeOfNextRound()
	retryInterval := time.Second

	for attempt := 0; ; attempt++ {
		if err = retryUntil(ctx, func() error {
			err := sequencerRpc.CallContext(ctx, nil, "auctioneer_submitAuctionResolutionTransaction", tx)
			if err != nil {
				log.Error("Error submitting auction resolution to sequencer endpoint", "error", err)
			}
			return err
		}, retryInterval, roundEndTime); err != nil {
			return err
		}

		waitMinedCtx, cancel := context.WithTimeout(ctx, time.Until(roundEndTime))
		receipt, err := bind.WaitMined(waitMinedCtx, ethclient.NewClient(sequencerRpc), tx)
		cancel()
		if err != nil {
			return fmt.Errorf("error waiting for transaction to be mined: %w", err)
		}

		if receipt.Status == types.ReceiptStatusSuccessful {
			break
		}
		log.Warn("Auction resolution transaction reverted on-chain",
			"round", upcomingRound, "txHash", tx.Hash().Hex(), "attempt", attempt+1, "receiptStatus", receipt.Status)
		if attempt >= maxAuctionResolutionRetries {
			return errors.New("could not resolve auction after multiple attempts")
		}
		// Re-acquire sequencer RPC in case of failover since the last attempt.
		sequencerRpc, err = a.refreshSequencerEndpoint(ctx)
		if err != nil {
			return err
		}
		tx, err = makeAuctionResolutionTx(a.auctionContract)
		if err != nil {
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
	shuttingDown := ctx.Err() != nil
	for msgID, msg := range a.unackedBids {
		bid := msg.Value
		if uint64(bid.Round) <= round {
			msg.Ack() // Stop the heartbeat goroutine

			if !shuttingDown {
				// SetResult calls XAck to remove the msg from the consumer group's
				// pending list and then removes it from the stream with XDel.
				if err := a.consumer.SetResult(ctx, msgID, nil); err != nil {
					if ctx.Err() != nil {
						shuttingDown = true
						log.Debug("Skipping remaining bid acknowledgments during shutdown (will be re-consumed on restart)")
					} else {
						bidAckFailureCounter.Inc(1)
						log.Warn("Error marking bid message as consumed by auctioneer", "msgID", msgID, "error", err)
					}
				}
			}
			// Delete from unackedBids unconditionally since Ack() can only be
			// called once. Bids not removed from Redis will be re-consumed on
			// restart via the consumer group's pending entries list.
			delete(a.unackedBids, msgID)
			acknowledgedCount++
		}
	}

	if shuttingDown {
		log.Info("Bid heartbeats stopped during shutdown (bids remain in Redis for re-consumption on restart)", "count", acknowledgedCount)
	} else {
		log.Info("Acknowledged bids in redis stream", "count", acknowledgedCount)
	}
}

// retryUntil retries a given operation until it succeeds, the end time passes, or the
// context is cancelled. It returns nil on success, or an error indicating whether the
// deadline was already past, the context was cancelled, or all attempts failed.
func retryUntil(ctx context.Context, operation func() error, retryInterval time.Duration, endTime time.Time) error {
	if time.Now().After(endTime) {
		return fmt.Errorf("operation not attempted: deadline %v already passed", endTime)
	}

	var lastErr error
	for {
		if time.Now().After(endTime) {
			break
		}
		if lastErr = operation(); lastErr == nil {
			return nil
		}

		remaining := time.Until(endTime)
		if remaining <= 0 {
			break
		}
		timer := time.NewTimer(min(retryInterval, remaining))
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
		}
		if ctx.Err() != nil {
			return fmt.Errorf("%w (last operation error: %w)", ctx.Err(), lastErr)
		}
	}

	return fmt.Errorf("operation failed after retrying until %v: %w", endTime, lastErr)
}

func (a *AuctioneerServer) persistValidatedBid(bid *ValidatedBid) {
	if err := a.database.InsertBid(bid); err != nil {
		persistBidFailureCounter.Inc(1)
		// The bid is still in the bidCache for auction resolution; only the
		// archival record (used by S3 upload) is lost.
		log.Error("Could not persist validated bid to database", "err", err, "bidder", bid.Bidder, "amount", bid.Amount.String(), "round", bid.Round)
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
	a.StopWaiter.StopAndWait()
	a.endpointManager.Close()
	if err := a.database.Close(); err != nil {
		log.Warn("Error closing database during AuctioneerServer shutdown", "error", err)
	}
	if err := a.redisClient.Close(); err != nil {
		log.Warn("Error closing Redis client during AuctioneerServer shutdown", "error", err)
	}
}
