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
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/pkg/errors"
)

// domainValue is the Keccak256 hash of the string "TIMEBOOST_BID".
// This variable represents a fixed domain identifier used in the express lane auction.
var domainValue = []byte{
	0xc7, 0xf4, 0x5f, 0x6f, 0x1b, 0x1e, 0x1d, 0xfc,
	0x22, 0xe1, 0xb9, 0xf6, 0x9c, 0xda, 0x8e, 0x4e,
	0x86, 0xf4, 0x84, 0x81, 0xf0, 0xc5, 0xe0, 0x19,
	0x7c, 0x3f, 0x09, 0x1b, 0x89, 0xe8, 0xeb, 0x12,
}

type AuctioneerOpt func(*Auctioneer)

// Auctioneer is a struct that represents an autonomous auctioneer.
// It is responsible for receiving bids, validating them, and resolving auctions.
// Spec: https://github.com/OffchainLabs/timeboost-design/tree/main
type Auctioneer struct {
	txOpts                    *bind.TransactOpts
	chainId                   []uint64 // Auctioneer could handle auctions on multiple chains.
	domainValue               []byte
	client                    Client
	auctionContract           *express_lane_auctiongen.ExpressLaneAuction
	bidsReceiver              chan *Bid
	bidCache                  *bidCache
	initialRoundTimestamp     time.Time
	roundDuration             time.Duration
	auctionClosingDuration    time.Duration
	reserveSubmissionDuration time.Duration
	reservePriceLock          sync.RWMutex
	reservePrice              *big.Int
}

// NewAuctioneer creates a new autonomous auctioneer struct.
func NewAuctioneer(
	txOpts *bind.TransactOpts,
	chainId []uint64,
	client Client,
	auctionContract *express_lane_auctiongen.ExpressLaneAuction,
	opts ...AuctioneerOpt,
) (*Auctioneer, error) {
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
		bidsReceiver:              make(chan *Bid, 10_000), // TODO(Terence): Is 10000 enough? Make this configurable?
		bidCache:                  newBidCache(),
		initialRoundTimestamp:     initialTimestamp,
		roundDuration:             roundDuration,
		auctionClosingDuration:    auctionClosingDuration,
		reserveSubmissionDuration: reserveSubmissionDuration,
		reservePrice:              reservePrice,
		domainValue:               domainValue,
	}
	for _, o := range opts {
		o(am)
	}
	return am, nil
}

// ReceiveBid validates and adds a bid to the bid cache.
func (a *Auctioneer) receiveBid(ctx context.Context, b *Bid) error {
	vb, err := a.validateBid(b)
	if err != nil {
		return errors.Wrap(err, "could not validate bid")
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

func (a *Auctioneer) validateBid(bid *Bid) (*validatedBid, error) {
	// Check basic integrity.
	if bid == nil {
		return nil, errors.Wrap(ErrMalformedData, "nil bid")
	}
	if bid.Bidder == (common.Address{}) {
		return nil, errors.Wrap(ErrMalformedData, "empty bidder address")
	}
	if bid.ExpressLaneController == (common.Address{}) {
		return nil, errors.Wrap(ErrMalformedData, "empty express lane controller address")
	}

	// Check if the chain ID is valid.
	chainIdOk := false
	for _, id := range a.chainId {
		if bid.ChainId == id {
			chainIdOk = true
			break
		}
	}
	if !chainIdOk {
		return nil, errors.Wrapf(ErrWrongChainId, "can not aucution for chain id: %d", bid.ChainId)
	}

	// Check if the bid is intended for upcoming round.
	upcomingRound := CurrentRound(a.initialRoundTimestamp, a.roundDuration) + 1
	if bid.Round != upcomingRound {
		return nil, errors.Wrapf(ErrBadRoundNumber, "wanted %d, got %d", upcomingRound, bid.Round)
	}

	// Check if the auction is closed.
	if d, closed := auctionClosed(a.initialRoundTimestamp, a.roundDuration, a.auctionClosingDuration); closed {
		return nil, fmt.Errorf("auction is closed, %d seconds into the round", d)
	}

	// Check bid is higher than reserve price.
	reservePrice := a.fetchReservePrice()
	if bid.Amount.Cmp(reservePrice) == -1 {
		return nil, errors.Wrapf(ErrInsufficientBid, "reserve price %s, bid %s", reservePrice, bid.Amount)
	}

	// Validate the signature.
	packedBidBytes, err := encodeBidValues(
		a.domainValue,
		new(big.Int).SetUint64(bid.ChainId),
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
	prefixed := crypto.Keccak256(append([]byte("\x19Ethereum Signed Message:\n112"), packedBidBytes...))
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
	// Validate if the user if a depositor in the contract and has enough balance for the bid.
	// TODO: Retry some number of times if flakey connection.
	// TODO: Validate reserve price against amount of bid.
	// TODO: No need to do anything expensive if the bid coming is in invalid.
	// Cache this if the received time of the bid is too soon. Include the arrival timestamp.
	depositBal, err := a.auctionContract.BalanceOf(&bind.CallOpts{}, bid.Bidder)
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
		expressLaneController: bid.ExpressLaneController,
		amount:                bid.Amount,
		signature:             bid.Signature,
	}, nil
}

// CurrentRound returns the current round number.
func CurrentRound(initialRoundTimestamp time.Time, roundDuration time.Duration) uint64 {
	return uint64(time.Since(initialRoundTimestamp) / roundDuration)
}

// auctionClosed returns the time since auction was closed and whether the auction is closed.
func auctionClosed(initialRoundTimestamp time.Time, roundDuration time.Duration, auctionClosingDuration time.Duration) (time.Duration, bool) {
	d := time.Since(initialRoundTimestamp) % roundDuration
	return d, d > auctionClosingDuration
}
