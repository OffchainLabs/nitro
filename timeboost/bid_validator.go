package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type BidValidatorConfigFetcher func() *BidValidatorConfig

type BidValidatorConfig struct {
	Enable         bool                  `koanf:"enable"`
	RedisURL       string                `koanf:"redis-url"`
	ProducerConfig pubsub.ProducerConfig `koanf:"producer-config"`
	// Timeout on polling for existence of each redis stream.
	SequencerEndpoint      string `koanf:"sequencer-endpoint"`
	AuctionContractAddress string `koanf:"auction-contract-address"`
}

var DefaultBidValidatorConfig = BidValidatorConfig{
	Enable:         true,
	RedisURL:       "",
	ProducerConfig: pubsub.DefaultProducerConfig,
}

var TestBidValidatorConfig = BidValidatorConfig{
	Enable:         true,
	RedisURL:       "",
	ProducerConfig: pubsub.TestProducerConfig,
}

func BidValidatorConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBidValidatorConfig.Enable, "enable bid validator")
	f.String(prefix+".redis-url", DefaultBidValidatorConfig.RedisURL, "url of redis server")
	pubsub.ProducerAddConfigAddOptions(prefix+".producer-config", f)
	f.String(prefix+".sequencer-endpoint", DefaultAuctioneerServerConfig.SequencerEndpoint, "sequencer RPC endpoint")
	f.String(prefix+".auction-contract-address", DefaultAuctioneerServerConfig.AuctionContractAddress, "express lane auction contract address")
}

type BidValidator struct {
	stopwaiter.StopWaiter
	sync.RWMutex
	reservePriceLock          sync.RWMutex
	chainId                   *big.Int
	stack                     *node.Node
	producerCfg               *pubsub.ProducerConfig
	producer                  *pubsub.Producer[*JsonValidatedBid, error]
	redisClient               redis.UniversalClient
	domainValue               []byte
	client                    *ethclient.Client
	auctionContract           *express_lane_auctiongen.ExpressLaneAuction
	auctionContractAddr       common.Address
	bidsReceiver              chan *Bid
	bidCache                  *bidCache
	initialRoundTimestamp     time.Time
	roundDuration             time.Duration
	auctionClosingDuration    time.Duration
	reserveSubmissionDuration time.Duration
	reservePrice              *big.Int
	bidsPerSenderInRound      map[common.Address]uint8
	maxBidsPerSenderInRound   uint8
}

func NewBidValidator(
	ctx context.Context,
	stack *node.Node,
	configFetcher BidValidatorConfigFetcher,
) (*BidValidator, error) {
	cfg := configFetcher()
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url cannot be empty")
	}
	if cfg.AuctionContractAddress == "" {
		return nil, fmt.Errorf("auction contract address cannot be empty")
	}
	auctionContractAddr := common.HexToAddress(cfg.AuctionContractAddress)
	redisClient, err := redisutil.RedisClientFromURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	client, err := rpc.DialContext(ctx, cfg.SequencerEndpoint)
	if err != nil {
		return nil, err
	}
	sequencerClient := ethclient.NewClient(client)
	chainId, err := sequencerClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, sequencerClient)
	if err != nil {
		return nil, err
	}
	roundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	initialTimestamp := time.Unix(int64(roundTimingInfo.OffsetTimestamp), 0)
	roundDuration := time.Duration(roundTimingInfo.RoundDurationSeconds) * time.Second
	auctionClosingDuration := time.Duration(roundTimingInfo.AuctionClosingSeconds) * time.Second
	reserveSubmissionDuration := time.Duration(roundTimingInfo.ReserveSubmissionSeconds) * time.Second

	reservePrice, err := auctionContract.ReservePrice(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	bidValidator := &BidValidator{
		chainId:                   chainId,
		client:                    sequencerClient,
		redisClient:               redisClient,
		stack:                     stack,
		auctionContract:           auctionContract,
		auctionContractAddr:       auctionContractAddr,
		bidsReceiver:              make(chan *Bid, 10_000),
		bidCache:                  newBidCache(),
		initialRoundTimestamp:     initialTimestamp,
		roundDuration:             roundDuration,
		auctionClosingDuration:    auctionClosingDuration,
		reserveSubmissionDuration: reserveSubmissionDuration,
		reservePrice:              reservePrice,
		domainValue:               domainValue,
		bidsPerSenderInRound:      make(map[common.Address]uint8),
		maxBidsPerSenderInRound:   5, // 5 max bids per sender address in a round.
		producerCfg:               &cfg.ProducerConfig,
	}
	api := &BidValidatorAPI{bidValidator}
	valAPIs := []rpc.API{{
		Namespace: AuctioneerNamespace,
		Version:   "1.0",
		Service:   api,
		Public:    true,
	}}
	stack.RegisterAPIs(valAPIs)
	return bidValidator, nil
}

func EnsureBidValidatorExposedViaRPC(stackConf *node.Config) {
	found := false
	for _, module := range stackConf.HTTPModules {
		if module == AuctioneerNamespace {
			found = true
			break
		}
	}
	if !found {
		stackConf.HTTPModules = append(stackConf.HTTPModules, AuctioneerNamespace)
	}
}

func (bv *BidValidator) Initialize(ctx context.Context) error {
	if err := pubsub.CreateStream(
		ctx,
		validatedBidsRedisStream,
		bv.redisClient,
	); err != nil {
		return fmt.Errorf("creating redis stream: %w", err)
	}
	p, err := pubsub.NewProducer[*JsonValidatedBid, error](
		bv.redisClient, validatedBidsRedisStream, bv.producerCfg,
	)
	if err != nil {
		return fmt.Errorf("failed to init redis in bid validator: %w", err)
	}
	bv.producer = p
	return nil
}

func (bv *BidValidator) Start(ctx_in context.Context) {
	bv.StopWaiter.Start(ctx_in, bv)
	if bv.producer == nil {
		log.Crit("Bid validator not yet initialized by calling Initialize(ctx)")
	}
	bv.producer.Start(ctx_in)
}

type BidValidatorAPI struct {
	*BidValidator
}

func (bv *BidValidatorAPI) SubmitBid(ctx context.Context, bid *JsonBid) error {
	start := time.Now()
	receivedBidsCounter.Inc(1)
	validatedBid, err := bv.validateBid(
		&Bid{
			ChainId:                bid.ChainId.ToInt(),
			ExpressLaneController:  bid.ExpressLaneController,
			AuctionContractAddress: bid.AuctionContractAddress,
			Round:                  uint64(bid.Round),
			Amount:                 bid.Amount.ToInt(),
			Signature:              bid.Signature,
		},
		bv.auctionContract.BalanceOf,
		bv.fetchReservePrice,
	)
	if err != nil {
		return err
	}
	validatedBidsCounter.Inc(1)
	log.Info("Validated bid", "bidder", validatedBid.Bidder.Hex(), "amount", validatedBid.Amount.String(), "round", validatedBid.Round, "elapsed", time.Since(start))
	_, err = bv.producer.Produce(ctx, validatedBid)
	if err != nil {
		return err
	}
	return nil
}

// TODO(Terence): Set reserve price from the contract.
func (bv *BidValidator) fetchReservePrice() *big.Int {
	bv.reservePriceLock.RLock()
	defer bv.reservePriceLock.RUnlock()
	return new(big.Int).Set(bv.reservePrice)
}

func (bv *BidValidator) validateBid(
	bid *Bid,
	balanceCheckerFn func(opts *bind.CallOpts, addr common.Address) (*big.Int, error),
	fetchReservePriceFn func() *big.Int,
) (*JsonValidatedBid, error) {
	// Check basic integrity.
	if bid == nil {
		return nil, errors.Wrap(ErrMalformedData, "nil bid")
	}
	if bid.AuctionContractAddress != bv.auctionContractAddr {
		return nil, errors.Wrap(ErrMalformedData, "incorrect auction contract address")
	}
	if bid.ExpressLaneController == (common.Address{}) {
		return nil, errors.Wrap(ErrMalformedData, "empty express lane controller address")
	}
	if bid.ChainId == nil {
		return nil, errors.Wrap(ErrMalformedData, "empty chain id")
	}

	// Check if the chain ID is valid.
	if bid.ChainId.Cmp(bv.chainId) != 0 {
		return nil, errors.Wrapf(ErrWrongChainId, "can not auction for chain id: %d", bid.ChainId)
	}

	// Check if the bid is intended for upcoming round.
	upcomingRound := CurrentRound(bv.initialRoundTimestamp, bv.roundDuration) + 1
	if bid.Round != upcomingRound {
		return nil, errors.Wrapf(ErrBadRoundNumber, "wanted %d, got %d", upcomingRound, bid.Round)
	}

	// Check if the auction is closed.
	if isAuctionRoundClosed(
		time.Now(),
		bv.initialRoundTimestamp,
		bv.roundDuration,
		bv.auctionClosingDuration,
	) {
		return nil, errors.Wrap(ErrBadRoundNumber, "auction is closed")
	}

	// Check bid is higher than reserve price.
	reservePrice := fetchReservePriceFn()
	if bid.Amount.Cmp(reservePrice) == -1 {
		return nil, errors.Wrapf(ErrReservePriceNotMet, "reserve price %s, bid %s", reservePrice.String(), bid.Amount.String())
	}

	// Validate the signature.
	packedBidBytes := bid.ToMessageBytes()
	if len(bid.Signature) != 65 {
		return nil, errors.Wrap(ErrMalformedData, "signature length is not 65")
	}
	// Recover the public key.
	sigItem := make([]byte, len(bid.Signature))
	copy(sigItem, bid.Signature)
	if sigItem[len(sigItem)-1] >= 27 {
		sigItem[len(sigItem)-1] -= 27
	}
	pubkey, err := crypto.SigToPub(buildEthereumSignedMessage(packedBidBytes), sigItem)
	if err != nil {
		return nil, ErrMalformedData
	}
	// Check how many bids the bidder has sent in this round and cap according to a limit.
	bidder := crypto.PubkeyToAddress(*pubkey)
	bv.Lock()
	numBids, ok := bv.bidsPerSenderInRound[bidder]
	if !ok {
		bv.bidsPerSenderInRound[bidder] = 1
	}
	if numBids >= bv.maxBidsPerSenderInRound {
		bv.Unlock()
		return nil, errors.Wrapf(ErrTooManyBids, "bidder %s has already sent the maximum allowed bids = %d in this round", bidder.Hex(), numBids)
	}
	bv.bidsPerSenderInRound[bidder]++
	bv.Unlock()

	depositBal, err := balanceCheckerFn(&bind.CallOpts{}, bidder)
	if err != nil {
		return nil, err
	}
	if depositBal.Cmp(new(big.Int)) == 0 {
		return nil, ErrNotDepositor
	}
	if depositBal.Cmp(bid.Amount) < 0 {
		return nil, errors.Wrapf(ErrInsufficientBalance, "onchain balance %#x, bid amount %#x", depositBal, bid.Amount)
	}
	vb := &ValidatedBid{
		ExpressLaneController:  bid.ExpressLaneController,
		Amount:                 bid.Amount,
		Signature:              bid.Signature,
		ChainId:                bid.ChainId,
		AuctionContractAddress: bid.AuctionContractAddress,
		Round:                  bid.Round,
		Bidder:                 bidder,
	}
	return vb.ToJson(), nil
}
