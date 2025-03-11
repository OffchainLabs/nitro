// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type nextRoundListener interface {
	nextRound(round uint64, controller common.Address)
}

// expressLaneTracker knows what round it is
type expressLaneTracker struct {
	stopwaiter.StopWaiter

	roundTimingInfo timeboost.RoundTimingInfo
	pollInterval    time.Duration

	apiBackend      *arbitrum.APIBackend
	auctionContract *express_lane_auctiongen.ExpressLaneAuction

	listener nextRoundListener
}

func newExpressLaneTracker(
	roundTimingInfo timeboost.RoundTimingInfo,
	pollInterval time.Duration,
	apiBackend *arbitrum.APIBackend,
	auctionContract *express_lane_auctiongen.ExpressLaneAuction,
	listener nextRoundListener) *expressLaneTracker {
	return &expressLaneTracker{
		roundTimingInfo: roundTimingInfo,
		pollInterval:    pollInterval,
		apiBackend:      apiBackend,
		auctionContract: auctionContract,
		listener:        listener,
	}
}

func (t *expressLaneTracker) Start(ctxIn context.Context) {
	t.StopWaiter.Start(ctxIn, t)

	t.LaunchThread(func(ctx context.Context) {
		// Monitor for auction resolutions from the auction manager smart contract
		// and set the express lane controller for the upcoming round accordingly.
		log.Info("Monitoring express lane auction contract")

		var fromBlock uint64
		latestBlock, err := t.apiBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
		if err != nil {
			log.Error("ExpressLaneService could not get the latest header", "err", err)
		} else {
			maxBlocksPerRound := t.roundTimingInfo.Round / t.pollInterval
			fromBlock = latestBlock.Number.Uint64()
			// #nosec G115
			if fromBlock > uint64(maxBlocksPerRound) {
				// #nosec G115
				fromBlock -= uint64(maxBlocksPerRound)
			}
		}

		ticker := time.NewTicker(t.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			latestBlock, err := t.apiBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
			if err != nil {
				log.Error("expressLaneTracker could not get the latest header", "err", err)
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock > toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Context: ctx,
				Start:   fromBlock,
				End:     &toBlock,
			}

			it, err := t.auctionContract.FilterAuctionResolved(filterOpts, nil, nil, nil)
			if err != nil {
				log.Error("Could not filter auction resolutions event", "error", err)
				continue
			}
			for it.Next() {
				timeSinceAuctionClose := t.roundTimingInfo.AuctionClosing - t.roundTimingInfo.TimeTilNextRound()
				auctionResolutionLatency.Update(timeSinceAuctionClose.Nanoseconds())
				log.Info(
					"AuctionResolved: New express lane controller assigned",
					"round", it.Event.Round,
					"controller", it.Event.FirstPriceExpressLaneController,
					"timeSinceAuctionClose", timeSinceAuctionClose,
				)
				t.listener.nextRound(it.Event.Round, it.Event.FirstPriceExpressLaneController)
			}
			fromBlock = toBlock + 1
		}
	})
}
