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
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/pkg/errors"
)

type AuctioneerOpt func(*Auctioneer)

type Auctioneer struct {
	txOpts                    *bind.TransactOpts
	chainId                   *big.Int
	client                    Client
	auctionContract           *express_lane_auctiongen.ExpressLaneAuction
	bidsReceiver              chan *Bid
	bidCache                  *bidCache
	initialRoundTimestamp     time.Time
	roundDuration             time.Duration
	auctionClosingDuration    time.Duration
	reserveSubmissionDuration time.Duration
	auctionContractAddr       common.Address
	reservePriceLock          sync.RWMutex
	reservePrice              *big.Int
}

func NewAuctioneer(
	txOpts *bind.TransactOpts,
	chainId *big.Int,
	client Client,
	auctionContractAddr common.Address,
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

	minReservePrice, err := auctionContract.MinReservePrice(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	am := &Auctioneer{
		txOpts:                    txOpts,
		chainId:                   chainId,
		client:                    client,
		auctionContract:           auctionContract,
		bidsReceiver:              make(chan *Bid, 10_000),
		bidCache:                  newBidCache(),
		initialRoundTimestamp:     initialTimestamp,
		auctionContractAddr:       auctionContractAddr,
		roundDuration:             roundDuration,
		reservePrice:              minReservePrice,
		auctionClosingDuration:    auctionClosingDuration,
		reserveSubmissionDuration: reserveSubmissionDuration,
	}
	for _, o := range opts {
		o(am)
	}
	return am, nil
}

func (am *Auctioneer) ReceiveBid(ctx context.Context, b *Bid) error {
	validated, err := am.newValidatedBid(b)
	if err != nil {
		return fmt.Errorf("could not validate bid: %v", err)
	}
	am.bidCache.add(validated)
	return nil
}

func (am *Auctioneer) Start(ctx context.Context) {
	// Receive bids in the background.
	go receiveAsync(ctx, am.bidsReceiver, am.ReceiveBid)

	// Listen for sequencer health in the background and close upcoming auctions if so.
	go am.checkSequencerHealth(ctx)

	// Work on closing auctions.
	ticker := newAuctionCloseTicker(am.roundDuration, am.auctionClosingDuration)
	go ticker.start()
	for {
		select {
		case <-ctx.Done():
			log.Error("Context closed, autonomous auctioneer shutting down")
			return
		case auctionClosingTime := <-ticker.c:
			log.Info("New auction closing time reached", "closingTime", auctionClosingTime, "totalBids", am.bidCache.size())
			if err := am.resolveAuction(ctx); err != nil {
				log.Error("Could not resolve auction for round", "error", err)
			}
		}
	}
}

func (am *Auctioneer) resolveAuction(ctx context.Context) error {
	upcomingRound := CurrentRound(am.initialRoundTimestamp, am.roundDuration) + 1
	// If we have no winner, then we can cancel the auction.
	// Auctioneer can also subscribe to sequencer feed and
	// close auction if sequencer is down.
	result := am.bidCache.topTwoBids()
	first := result.firstPlace
	second := result.secondPlace
	var tx *types.Transaction
	var err error
	hasSingleBid := first != nil && second == nil
	hasBothBids := first != nil && second != nil
	noBids := first == nil && second == nil

	// TODO: Retry a given number of times in case of flakey connection.
	switch {
	case hasBothBids:
		fmt.Printf("First express lane controller: %#x\n", first.expressLaneController)
		tx, err = am.auctionContract.ResolveMultiBidAuction(
			am.txOpts,
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
		log.Info("Resolving auctions, received two bids", "round", upcomingRound)
	case hasSingleBid:
		log.Info("Resolving auctions, received single bids", "round", upcomingRound)
		tx, err = am.auctionContract.ResolveSingleBidAuction(
			am.txOpts,
			express_lane_auctiongen.Bid{
				ExpressLaneController: first.expressLaneController,
				Amount:                first.amount,
				Signature:             first.signature,
			},
		)
	case noBids:
		// TODO: Cancel the upcoming auction.
		log.Info("No bids received for auction resolution")
		return nil
	}
	if err != nil {
		return err
	}
	receipt, err := bind.WaitMined(ctx, am.client, tx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("deposit failed")
	}
	// Clear the bid cache.
	am.bidCache = newBidCache()
	return nil
}

// TODO: Implement. If sequencer is down for some time, cancel the upcoming auction by calling
// the cancel method on the smart contract.
func (am *Auctioneer) checkSequencerHealth(ctx context.Context) {

}

func CurrentRound(initialRoundTimestamp time.Time, roundDuration time.Duration) uint64 {
	return uint64(time.Since(initialRoundTimestamp) / roundDuration)
}
