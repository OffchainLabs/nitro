package timeboost

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type BidValidatorConfigFetcher func() *BidValidatorConfig

type BidValidatorConfig struct {
	Enable         bool                  `koanf:"enable"`
	RedisURL       string                `koanf:"redis-url"`
	ProducerConfig pubsub.ProducerConfig `koanf:"producer-config"`
	// Timeout on polling for existence of each redis stream.
	RpcEndpoint             string `koanf:"rpc-endpoint"`
	AuctionContractAddress  string `koanf:"auction-contract-address"`
	MaxBidsPerSender        uint8  `koanf:"max-bids-per-sender"`
	EnableEthcallValidation bool   `koanf:"enable-ethcall-validation"`
	AuctioneerAddress       string `koanf:"auctioneer-address"`
}

var DefaultBidValidatorConfig = BidValidatorConfig{
	Enable:                  true,
	RedisURL:                "",
	ProducerConfig:          pubsub.DefaultProducerConfig,
	MaxBidsPerSender:        5,
	EnableEthcallValidation: true,
}

var TestBidValidatorConfig = BidValidatorConfig{
	Enable:                  true,
	RedisURL:                "",
	ProducerConfig:          pubsub.TestProducerConfig,
	MaxBidsPerSender:        5,
	EnableEthcallValidation: true,
}

func BidValidatorConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBidValidatorConfig.Enable, "enable bid validator")
	f.String(prefix+".redis-url", DefaultBidValidatorConfig.RedisURL, "url of redis server")
	pubsub.ProducerAddConfigAddOptions(prefix+".producer-config", f)
	f.String(prefix+".rpc-endpoint", DefaultBidValidatorConfig.RpcEndpoint, "url of rpc endpoint")
	f.String(prefix+".auction-contract-address", DefaultBidValidatorConfig.AuctionContractAddress, "express lane auction contract address")
	f.Uint8(prefix+".max-bids-per-sender", DefaultBidValidatorConfig.MaxBidsPerSender, "maximum number of bids a sender can submit per round")
	f.Bool(prefix+".enable-ethcall-validation", DefaultBidValidatorConfig.EnableEthcallValidation, "enable eth_call validation of bids")
	f.String(prefix+".auctioneer-address", DefaultBidValidatorConfig.AuctioneerAddress, "Address of the Timeboost Autonomous Auctioneer required for eth_call validation of bids")

}

type BidValidator struct {
	stopwaiter.StopWaiter
	sync.RWMutex
	chainId                        *big.Int
	stack                          *node.Node
	producerCfg                    *pubsub.ProducerConfig
	producer                       *pubsub.Producer[*JsonValidatedBid, error]
	redisClient                    redis.UniversalClient
	domainValue                    []byte
	client                         *ethclient.Client
	auctionContract                *express_lane_auctiongen.ExpressLaneAuction
	auctionContractAddr            common.Address
	auctionContractDomainSeparator [32]byte
	bidsReceiver                   chan *Bid
	roundTimingInfo                RoundTimingInfo
	reservePriceLock               sync.RWMutex
	reservePrice                   *big.Int
	bidsPerSenderInRound           map[common.Address]uint8
	maxBidsPerSenderInRound        uint8
	enableEthcallValidation        bool
	auctioneerAddr                 common.Address
	auctionContractAbi             *abi.ABI
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

	client, err := rpc.DialContext(ctx, cfg.RpcEndpoint)
	if err != nil {
		return nil, err
	}
	rpcClient := ethclient.NewClient(client)
	chainId, err := rpcClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, rpcClient)
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

	reservePrice, err := auctionContract.ReservePrice(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	domainSeparator, err := auctionContract.DomainSeparator(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	var auctioneerAddr common.Address
	var auctionContractAbi *abi.ABI
	if cfg.EnableEthcallValidation {
		if cfg.AuctioneerAddress == "" {
			return nil, fmt.Errorf("auctioneer address cannot be empty, used for eth_call validation of bids")
		}
		auctioneerAddr = common.HexToAddress(cfg.AuctioneerAddress)
		auctionContractAbi, err = express_lane_auctiongen.ExpressLaneAuctionMetaData.GetAbi()
		if err != nil {
			return nil, errors.Wrap(err, "getting ExpressLaneAuctionABI")
		}
	}

	bidValidator := &BidValidator{
		chainId:                        chainId,
		client:                         rpcClient,
		redisClient:                    redisClient,
		stack:                          stack,
		auctionContract:                auctionContract,
		auctionContractAddr:            auctionContractAddr,
		auctionContractDomainSeparator: domainSeparator,
		bidsReceiver:                   make(chan *Bid, 10_000),
		roundTimingInfo:                *roundTimingInfo,
		reservePrice:                   reservePrice,
		domainValue:                    domainValue,
		bidsPerSenderInRound:           make(map[common.Address]uint8),
		maxBidsPerSenderInRound:        cfg.MaxBidsPerSender,
		producerCfg:                    &cfg.ProducerConfig,
		enableEthcallValidation:        cfg.EnableEthcallValidation,
		auctioneerAddr:                 auctioneerAddr,
		auctionContractAbi:             auctionContractAbi,
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

	// Thread to set reserve price and clear per-round map of bid count per account.
	bv.StopWaiter.LaunchThread(func(ctx context.Context) {
		reservePriceTicker := newRoundTicker(bv.roundTimingInfo)
		go reservePriceTicker.tickAtReserveSubmissionDeadline()
		auctionCloseTicker := newRoundTicker(bv.roundTimingInfo)
		go auctionCloseTicker.tickAtAuctionClose()

		for {
			select {
			case <-ctx.Done():
				log.Error("Context closed, autonomous auctioneer shutting down")
				return
			case <-reservePriceTicker.c:
				rp, err := bv.auctionContract.ReservePrice(&bind.CallOpts{})
				if err != nil {
					log.Error("Could not get reserve price", "error", err)
					continue
				}

				currentReservePrice := bv.fetchReservePrice()
				if currentReservePrice.Cmp(rp) == 0 {
					continue
				}

				log.Info("Reserve price updated", "old", currentReservePrice.String(), "new", rp.String())
				bv.SetReservePrice(rp)

			case <-auctionCloseTicker.c:
				bv.Lock()
				bv.bidsPerSenderInRound = make(map[common.Address]uint8)
				bv.Unlock()
			}
		}
	})
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

// SetReservePrice is exported for testing eth_call validation
func (bv *BidValidator) SetReservePrice(p *big.Int) {
	bv.reservePriceLock.Lock()
	defer bv.reservePriceLock.Unlock()
	bv.reservePrice = p
}

func (bv *BidValidator) fetchReservePrice() *big.Int {
	bv.reservePriceLock.RLock()
	defer bv.reservePriceLock.RUnlock()
	return bv.reservePrice
}

// Check time-related constraints for bid.
// It's useful to split out to be able to re-check just these constraints after
// time has elapsed.
func validateBidTimeConstraints(roundTimingInfo *RoundTimingInfo, bidRound uint64) error {
	// Check if the bid is intended for upcoming round.
	upcomingRound := roundTimingInfo.RoundNumber() + 1
	if bidRound != upcomingRound {
		return errors.Wrapf(ErrBadRoundNumber, "wanted %d, got %d", upcomingRound, bidRound)
	}

	// Check if the auction is closed.
	if roundTimingInfo.isAuctionRoundClosed() {
		return errors.Wrap(ErrBadRoundNumber, "auction is closed")
	}
	return nil
}

func (bv *BidValidator) validateBid(
	bid *Bid,
	balanceCheckerFn func(opts *bind.CallOpts, account common.Address) (*big.Int, error)) (*JsonValidatedBid, error) {
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

	if err := validateBidTimeConstraints(&bv.roundTimingInfo, (uint64)(bid.Round)); err != nil {
		return nil, err
	}

	// Check bid is higher than or equal to reserve price.
	if bid.Amount.Cmp(bv.reservePrice) == -1 {
		return nil, errors.Wrapf(ErrReservePriceNotMet, "reserve price %s, bid %s", bv.reservePrice.String(), bid.Amount.String())
	}

	// Validate the signature.
	if len(bid.Signature) != 65 {
		return nil, errors.Wrap(ErrMalformedData, "signature length is not 65")
	}

	// Recover the public key.
	sigItem := make([]byte, len(bid.Signature))
	copy(sigItem, bid.Signature)

	// Signature verification expects the last byte of the signature to have 27 subtracted,
	// as it represents the recovery ID. If the last byte is greater than or equal to 27, it indicates a recovery ID that hasn't been adjusted yet,
	// it's needed for internal signature verification logic.
	if sigItem[len(sigItem)-1] != 27 && sigItem[len(sigItem)-1] != 28 {
		return nil, errors.New("invalid Ethereum signature (V is not 27 or 28)")
	}
	sigItem[len(sigItem)-1] -= 27

	bidHash, err := bid.ToEIP712Hash(bv.auctionContractDomainSeparator)
	if err != nil {
		return nil, err
	}
	pubkey, err := crypto.SigToPub(bidHash[:], sigItem)
	if err != nil {
		return nil, ErrMalformedData
	}
	// Check how many bids the bidder has sent in this round and cap according to a limit.
	bidder := crypto.PubkeyToAddress(*pubkey)
	if !crypto.VerifySignature(crypto.CompressPubkey(pubkey), bidHash[:], sigItem[:64]) {
		return nil, errors.New("invalid signature")
	}
	bv.Lock()
	numBids, ok := bv.bidsPerSenderInRound[bidder]
	if !ok {
		bv.bidsPerSenderInRound[bidder] = 0
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
		return nil, errors.Wrapf(ErrNotDepositor, "bidder %s", bidder.Hex())
	}
	if depositBal.Cmp(bid.Amount) < 0 {
		return nil, errors.Wrapf(ErrInsufficientBalance, "bidder %s, onchain balance %#x, bid amount %#x", bidder.Hex(), depositBal, bid.Amount)
	}

	if bv.enableEthcallValidation {
		var timeOverride map[string]interface{}
		if !bv.roundTimingInfo.IsWithinAuctionCloseWindow(time.Now()) {
			newTimestamp := bv.roundTimingInfo.TimeOfNextRound().Add(-1 * bv.roundTimingInfo.AuctionClosing)
			timeOverride = map[string]interface{}{
				"time": fmt.Sprintf("0x%x", newTimestamp.Unix()),
			}
		}
		calldata, err := bv.auctionContractAbi.Pack("resolveSingleBidAuction", bid)
		if err != nil {
			return nil, fmt.Errorf("error creating calldata for eth_call bid validation: %w", err)
		}
		params := []interface{}{
			map[string]interface{}{
				"from": bv.auctioneerAddr.Hex(),
				"to":   bv.auctionContractAddr.Hex(),
				"data": "0x" + hex.EncodeToString(calldata),
			}, nil, nil, timeOverride,
		}
		var result string
		err = bv.client.Client().CallContext(bv.GetContext(), &result, "eth_call", params...)
		if err != nil {
			return nil, fmt.Errorf("error validating bid via eth_call auction resolution: %w", err)
		}
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
