package gethexec

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/timeboost/bindings"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var _ expressLaneChecker = &expressLaneService{}

type expressLaneChecker interface {
	isExpressLaneTx(sender common.Address) bool
}

type expressLaneControl struct {
	round      uint64
	controller common.Address
}

type expressLaneService struct {
	stopwaiter.StopWaiter
	sync.RWMutex
	client           arbutil.L1Interface
	control          expressLaneControl
	auctionContract  *bindings.ExpressLaneAuction
	initialTimestamp time.Time
	roundDuration    time.Duration
}

func newExpressLaneService(
	client arbutil.L1Interface,
	auctionContractAddr common.Address,
) (*expressLaneService, error) {
	auctionContract, err := bindings.NewExpressLaneAuction(auctionContractAddr, client)
	if err != nil {
		return nil, err
	}
	initialRoundTimestamp, err := auctionContract.InitialRoundTimestamp(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	roundDurationSeconds, err := auctionContract.RoundDurationSeconds(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	initialTimestamp := time.Unix(initialRoundTimestamp.Int64(), 0)
	currRound := timeboost.CurrentRound(initialTimestamp, time.Duration(roundDurationSeconds)*time.Second)
	controller, err := auctionContract.ExpressLaneControllerByRound(&bind.CallOpts{}, big.NewInt(int64(currRound)))
	if err != nil {
		return nil, err
	}
	return &expressLaneService{
		auctionContract:  auctionContract,
		client:           client,
		initialTimestamp: initialTimestamp,
		control: expressLaneControl{
			controller: controller,
			round:      currRound,
		},
		roundDuration: time.Duration(roundDurationSeconds) * time.Second,
	}, nil
}

func (es *expressLaneService) Start(ctxIn context.Context) {
	es.StopWaiter.Start(ctxIn, es)

	// Log every new express lane auction round.
	es.LaunchThread(func(ctx context.Context) {
		log.Info("Watching for new express lane rounds")
		now := time.Now()
		waitTime := es.roundDuration - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())
		time.Sleep(waitTime)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				round := timeboost.CurrentRound(es.initialTimestamp, es.roundDuration)
				log.Info(
					"New express lane auction round",
					"round", round,
					"timestamp", t,
				)
			}
		}
	})
	es.LaunchThread(func(ctx context.Context) {
		log.Info("Monitoring express lane auction contract")
		// Monitor for auction resolutions from the auction manager smart contract
		// and set the express lane controller for the upcoming round accordingly.
		latestBlock, err := es.client.HeaderByNumber(ctx, nil)
		if err != nil {
			log.Crit("Could not get latest header", "err", err)
		}
		fromBlock := latestBlock.Number.Uint64()
		ticker := time.NewTicker(time.Millisecond * 250)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				latestBlock, err := es.client.HeaderByNumber(ctx, nil)
				if err != nil {
					log.Error("Could not get latest header", "err", err)
					continue
				}
				toBlock := latestBlock.Number.Uint64()
				if fromBlock == toBlock {
					continue
				}
				filterOpts := &bind.FilterOpts{
					Context: ctx,
					Start:   fromBlock,
					End:     &toBlock,
				}
				it, err := es.auctionContract.FilterAuctionResolved(filterOpts, nil, nil)
				if err != nil {
					log.Error("Could not filter auction resolutions", "error", err)
					continue
				}
				for it.Next() {
					log.Info(
						"New express lane controller assigned",
						"round", it.Event.WinnerRound,
						"controller", it.Event.WinningBidder,
					)
					es.Lock()
					es.control.round = it.Event.WinnerRound.Uint64()
					es.control.controller = it.Event.WinningBidder
					es.Unlock()
				}
				fromBlock = toBlock
			}
		}
	})
	es.LaunchThread(func(ctx context.Context) {
		// Monitor for auction cancelations.
		// TODO: Implement.
	})
}

func (es *expressLaneService) isExpressLaneTx(sender common.Address) bool {
	es.RLock()
	defer es.RUnlock()
	round := timeboost.CurrentRound(es.initialTimestamp, es.roundDuration)
	log.Info("Current round", "round", round, "controller", es.control.controller, "sender", sender)
	return round == es.control.round && sender == es.control.controller
}
