package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

// domainValue holds the Keccak256 hash of the string "TIMEBOOST_BID".
// It is intended to be immutable after initialization.
var domainValue []byte

const AuctioneerNamespace = "auctioneer"

func init() {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte("TIMEBOOST_BID"))
	domainValue = hash.Sum(nil)
}

type AuctioneerOpt func(*Auctioneer)

// Auctioneer is a struct that represents an autonomous auctioneer.
// It is responsible for receiving bids, validating them, and resolving auctions.
// Spec: https://github.com/OffchainLabs/timeboost-design/tree/main
type Auctioneer struct {
	txOpts                    *bind.TransactOpts
	chainId                   []*big.Int // Auctioneer could handle auctions on multiple chains.
	domainValue               []byte
	client                    Client
	auctionContract           *express_lane_auctiongen.ExpressLaneAuction
	auctionContractAddr       common.Address
	bidsReceiver              chan *Bid
	bidCache                  *bidCache
	initialRoundTimestamp     time.Time
	roundDuration             time.Duration
	auctionClosingDuration    time.Duration
	reserveSubmissionDuration time.Duration
	reservePriceLock          sync.RWMutex
	reservePrice              *big.Int
	sync.RWMutex
	bidsPerSenderInRound    map[common.Address]uint8
	maxBidsPerSenderInRound uint8
}

func EnsureValidationExposedViaAuthRPC(stackConf *node.Config) {
	found := false
	for _, module := range stackConf.AuthModules {
		if module == AuctioneerNamespace {
			found = true
			break
		}
	}
	if !found {
		stackConf.AuthModules = append(stackConf.AuthModules, AuctioneerNamespace)
	}
}

// NewAuctioneer creates a new autonomous auctioneer struct.
func NewAuctioneer(
	txOpts *bind.TransactOpts,
	chainId []*big.Int,
	stack *node.Node,
	client Client,
	auctionContractAddr common.Address,
	opts ...AuctioneerOpt,
) (*Auctioneer, error) {
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, client)
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
	am := &Auctioneer{
		txOpts:                    txOpts,
		chainId:                   chainId,
		client:                    client,
		auctionContract:           auctionContract,
		auctionContractAddr:       auctionContractAddr,
		bidsReceiver:              make(chan *Bid, 10_000), // TODO(Terence): Is 10000 enough? Make this configurable?
		bidCache:                  newBidCache(),
		initialRoundTimestamp:     initialTimestamp,
		roundDuration:             roundDuration,
		auctionClosingDuration:    auctionClosingDuration,
		reserveSubmissionDuration: reserveSubmissionDuration,
		reservePrice:              reservePrice,
		domainValue:               domainValue,
		bidsPerSenderInRound:      make(map[common.Address]uint8),
		maxBidsPerSenderInRound:   5, // 5 max bids per sender address in a round.
	}
	for _, o := range opts {
		o(am)
	}
	auctioneerApi := &AuctioneerAPI{am}
	valAPIs := []rpc.API{{
		Namespace: AuctioneerNamespace,
		Version:   "1.0",
		Service:   auctioneerApi,
		Public:    true,
	}}
	stack.RegisterAPIs(valAPIs)
	return am, nil
}

// ReceiveBid validates and adds a bid to the bid cache.
func (a *Auctioneer) receiveBid(ctx context.Context, b *Bid) error {
	vb, err := a.validateBid(b, a.auctionContract.BalanceOf, a.fetchReservePrice)
	if err != nil {
		return err
	}
	a.bidCache.add(vb)
	return nil
}

// Start starts the autonomous auctioneer.
func (a *Auctioneer) Start(ctx context.Context) {
	// Receive bids in the background.
	go receiveAsync(ctx, a.bidsReceiver, a.receiveBid)

	// Listen for sequencer health in the background and close upcoming auctions if so.
	go a.checkSequencerHealth(ctx)

	// Work on closing auctions.
	ticker := newAuctionCloseTicker(a.roundDuration, a.auctionClosingDuration)
	go ticker.start()
	for {
		select {
		case <-ctx.Done():
			log.Error("Context closed, autonomous auctioneer shutting down")
			return
		case auctionClosingTime := <-ticker.c:
			log.Info("New auction closing time reached", "closingTime", auctionClosingTime, "totalBids", a.bidCache.size())
			if err := a.resolveAuction(ctx); err != nil {
				log.Error("Could not resolve auction for round", "error", err)
			}
			// Clear the bid cache.
			a.bidCache = newBidCache()
		}
	}
}

// resolveAuction resolves the auction by calling the smart contract with the top two bids.
func (a *Auctioneer) resolveAuction(ctx context.Context) error {
	upcomingRound := CurrentRound(a.initialRoundTimestamp, a.roundDuration) + 1
	result := a.bidCache.topTwoBids()
	first := result.firstPlace
	second := result.secondPlace
	var tx *types.Transaction
	var err error
	switch {
	case first != nil && second != nil: // Both bids are present
		tx, err = a.auctionContract.ResolveMultiBidAuction(
			a.txOpts,
			express_lane_auctiongen.Bid{
				ExpressLaneController: first.expressLaneController,
				Amount:                first.amount,
				Signature:             first.signature,
			},
			express_lane_auctiongen.Bid{
				ExpressLaneController: second.expressLaneController,
				Amount:                second.amount,
				Signature:             second.signature,
			},
		)
		log.Info("Resolving auction with two bids", "round", upcomingRound)

	case first != nil: // Single bid is present
		tx, err = a.auctionContract.ResolveSingleBidAuction(
			a.txOpts,
			express_lane_auctiongen.Bid{
				ExpressLaneController: first.expressLaneController,
				Amount:                first.amount,
				Signature:             first.signature,
			},
		)
		log.Info("Resolving auction with single bid", "round", upcomingRound)

	case second == nil: // No bids received
		log.Info("No bids received for auction resolution", "round", upcomingRound)
		return nil
	}

	if err != nil {
		log.Error("Error resolving auction", "error", err)
		return err
	}

	receipt, err := bind.WaitMined(ctx, a.client, tx)
	if err != nil {
		log.Error("Error waiting for transaction to be mined", "error", err)
		return err
	}

	if tx == nil || receipt == nil || receipt.Status != types.ReceiptStatusSuccessful {
		if tx != nil {
			log.Error("Transaction failed or did not finalize successfully", "txHash", tx.Hash().Hex())
		}
		return errors.New("transaction failed or did not finalize successfully")
	}

	log.Info("Auction resolved successfully", "txHash", tx.Hash().Hex())
	return nil
}

// TODO: Implement. If sequencer is down for some time, cancel the upcoming auction by calling
// the cancel method on the smart contract.
func (a *Auctioneer) checkSequencerHealth(ctx context.Context) {

}

// TODO(Terence): Set reserve price from the contract.
func (a *Auctioneer) fetchReservePrice() *big.Int {
	a.reservePriceLock.RLock()
	defer a.reservePriceLock.RUnlock()
	return new(big.Int).Set(a.reservePrice)
}

func (a *Auctioneer) validateBid(
	bid *Bid,
	balanceCheckerFn func(opts *bind.CallOpts, addr common.Address) (*big.Int, error),
	fetchReservePriceFn func() *big.Int,
) (*validatedBid, error) {
	// Check basic integrity.
	if bid == nil {
		return nil, errors.Wrap(ErrMalformedData, "nil bid")
	}
	if bid.AuctionContractAddress != a.auctionContractAddr {
		return nil, errors.Wrap(ErrMalformedData, "incorrect auction contract address")
	}
	if bid.ExpressLaneController == (common.Address{}) {
		return nil, errors.Wrap(ErrMalformedData, "empty express lane controller address")
	}
	if bid.ChainId == nil {
		return nil, errors.Wrap(ErrMalformedData, "empty chain id")
	}

	// Check if the chain ID is valid.
	chainIdOk := false
	for _, id := range a.chainId {
		if bid.ChainId.Cmp(id) == 0 {
			chainIdOk = true
			break
		}
	}
	if !chainIdOk {
		return nil, errors.Wrapf(ErrWrongChainId, "can not auction for chain id: %d", bid.ChainId)
	}

	// Check if the bid is intended for upcoming round.
	upcomingRound := CurrentRound(a.initialRoundTimestamp, a.roundDuration) + 1
	if bid.Round != upcomingRound {
		return nil, errors.Wrapf(ErrBadRoundNumber, "wanted %d, got %d", upcomingRound, bid.Round)
	}

	// Check if the auction is closed.
	if d, closed := auctionClosed(a.initialRoundTimestamp, a.roundDuration, a.auctionClosingDuration); closed {
		return nil, errors.Wrapf(ErrBadRoundNumber, "auction is closed, %s since closing", d)
	}

	// Check bid is higher than reserve price.
	reservePrice := fetchReservePriceFn()
	if bid.Amount.Cmp(reservePrice) == -1 {
		return nil, errors.Wrapf(ErrReservePriceNotMet, "reserve price %s, bid %s", reservePrice.String(), bid.Amount.String())
	}

	// Validate the signature.
	packedBidBytes, err := encodeBidValues(
		a.domainValue,
		bid.ChainId,
		bid.AuctionContractAddress,
		bid.Round,
		bid.Amount,
		bid.ExpressLaneController,
	)
	if err != nil {
		return nil, ErrMalformedData
	}
	if len(bid.Signature) != 65 {
		return nil, errors.Wrap(ErrMalformedData, "signature length is not 65")
	}
	// Recover the public key.
	prefixed := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(packedBidBytes))), packedBidBytes...))
	sigItem := make([]byte, len(bid.Signature))
	copy(sigItem, bid.Signature)
	if sigItem[len(sigItem)-1] >= 27 {
		sigItem[len(sigItem)-1] -= 27
	}
	pubkey, err := crypto.SigToPub(prefixed, sigItem)
	if err != nil {
		return nil, ErrMalformedData
	}
	if !verifySignature(pubkey, packedBidBytes, sigItem) {
		return nil, ErrWrongSignature
	}
	// Check how many bids the bidder has sent in this round and cap according to a limit.
	bidder := crypto.PubkeyToAddress(*pubkey)
	a.Lock()
	numBids, ok := a.bidsPerSenderInRound[bidder]
	if !ok {
		a.bidsPerSenderInRound[bidder] = 1
	}
	if numBids >= a.maxBidsPerSenderInRound {
		a.Unlock()
		return nil, errors.Wrapf(ErrTooManyBids, "bidder %s has already sent the maximum allowed bids = %d in this round", bidder.Hex(), numBids)
	}
	a.bidsPerSenderInRound[bidder]++
	a.Unlock()

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
	return &validatedBid{
		expressLaneController:  bid.ExpressLaneController,
		amount:                 bid.Amount,
		signature:              bid.Signature,
		chainId:                bid.ChainId,
		auctionContractAddress: bid.AuctionContractAddress,
		round:                  bid.Round,
		bidder:                 bidder,
	}, nil
}

// CurrentRound returns the current round number.
func CurrentRound(initialRoundTimestamp time.Time, roundDuration time.Duration) uint64 {
	if roundDuration == 0 {
		return 0
	}
	return uint64(time.Since(initialRoundTimestamp) / roundDuration)
}

// auctionClosed returns the time since auction was closed and whether the auction is closed.
func auctionClosed(initialRoundTimestamp time.Time, roundDuration time.Duration, auctionClosingDuration time.Duration) (time.Duration, bool) {
	if roundDuration == 0 {
		return 0, true
	}
	d := time.Since(initialRoundTimestamp) % roundDuration
	return d, d > auctionClosingDuration
}
