// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type RoundListener interface {
	NextRound(round uint64, controller common.Address)
}

type HeaderProvider interface {
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
}

// ExpressLaneTracker knows what round it is
type ExpressLaneTracker struct {
	stopwaiter.StopWaiter

	roundTimingInfo timeboost.RoundTimingInfo
	pollInterval    time.Duration
	chainConfig     *params.ChainConfig

	headerProvider       HeaderProvider
	auctionContract      *express_lane_auctiongen.ExpressLaneAuction
	auctionContractAddr  common.Address
	earlySubmissionGrace time.Duration

	roundControl containers.SyncMap[uint64, common.Address] // thread safe
	useLogs      bool
}

func NewExpressLaneTracker(
	roundTimingInfo timeboost.RoundTimingInfo,
	pollInterval time.Duration,
	headerProvider HeaderProvider,
	auctionContract *express_lane_auctiongen.ExpressLaneAuction,
	auctionContractAddr common.Address,
	chainConfig *params.ChainConfig,
	earlySubmissionGrace time.Duration) *ExpressLaneTracker {
	return &ExpressLaneTracker{
		roundTimingInfo:      roundTimingInfo,
		pollInterval:         pollInterval,
		headerProvider:       headerProvider,
		auctionContract:      auctionContract,
		auctionContractAddr:  auctionContractAddr,
		earlySubmissionGrace: earlySubmissionGrace,
		chainConfig:          chainConfig,
		useLogs:              false, // default to use contract polling
	}
}

func (t *ExpressLaneTracker) Start(ctxIn context.Context) {
	if t.useLogs {
		t.startViaLogIterator(ctxIn)
	} else {
		t.startViaContractPolling(ctxIn)
	}
}

func (t *ExpressLaneTracker) RoundController(round uint64) (common.Address, error) {
	controller, ok := t.roundControl.Load(round)
	if !ok {
		return common.Address{}, timeboost.ErrNoOnchainController
	}
	return controller, nil
}

// validateExpressLaneTx checks for the correctness of all fields of msg
func (t *ExpressLaneTracker) ValidateExpressLaneTx(msg *timeboost.ExpressLaneSubmission) error {
	if msg == nil || msg.Transaction == nil || msg.Signature == nil {
		return timeboost.ErrMalformedData
	}
	if msg.ChainId.Cmp(t.chainConfig.ChainID) != 0 {
		return errors.Wrapf(timeboost.ErrWrongChainId, "express lane tx chain ID %d does not match current chain ID %d", msg.ChainId, t.chainConfig.ChainID)
	}
	if msg.AuctionContractAddress != t.auctionContractAddr {
		return errors.Wrapf(timeboost.ErrWrongAuctionContract, "msg auction contract address %#x does not match sequencer auction contract address %#x", msg.AuctionContractAddress, t.auctionContractAddr)
	}

	currentRound := t.roundTimingInfo.RoundNumber()
	if msg.Round != currentRound {
		timeTilNextRound := t.roundTimingInfo.TimeTilNextRound()
		// We allow txs to come in for the next round if it is close enough to that round,
		// but we sleep until the round starts.
		if msg.Round == currentRound+1 && timeTilNextRound <= t.earlySubmissionGrace {
			time.Sleep(timeTilNextRound)
		} else {
			return errors.Wrapf(timeboost.ErrBadRoundNumber, "express lane tx round %d does not match current round %d", msg.Round, currentRound)
		}
	}

	controller, ok := t.roundControl.Load(msg.Round)
	if !ok {
		return timeboost.ErrNoOnchainController
	}
	// Extract sender address and cache it to be later used by sequenceExpressLaneSubmission
	sender, err := msg.Sender()
	if err != nil {
		return err
	}
	if sender != controller {
		return timeboost.ErrNotExpressLaneController
	}
	return nil
}

func (t *ExpressLaneTracker) AuctionContractAddr() common.Address {
	return t.auctionContractAddr
}

// --- internals ---

func (t *ExpressLaneTracker) startViaLogIterator(ctxIn context.Context) {
	t.StopWaiter.Start(ctxIn, t)

	t.LaunchThread(func(ctx context.Context) {
		// Monitor for auction resolutions from the auction manager smart contract
		// and set the express lane controller for the upcoming round accordingly.
		log.Info("Monitoring express lane auction contract via logs")

		var fromBlock uint64
		latestBlock, err := t.headerProvider.HeaderByNumber(ctx, rpc.LatestBlockNumber)
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

			latestBlock, err := t.headerProvider.HeaderByNumber(ctx, rpc.LatestBlockNumber)
			if err != nil {
				log.Error("ExpressLaneTracker could not get the latest header", "err", err)
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

				t.roundControl.Store(it.Event.Round, it.Event.FirstPriceExpressLaneController)

			}

			if it.Error() != nil {
				log.Error("Error occurred while iterating auction resolutions", "error", it.Error())
			}

			fromBlock = toBlock + 1
		}
	})

	t.roundHeartbeatThread()
}

func (t *ExpressLaneTracker) startViaContractPolling(ctxIn context.Context) {
	t.StopWaiter.Start(ctxIn, t)

	// poll contract state via resolvedRounds()
	t.LaunchThread(func(ctx context.Context) {
		log.Info("Monitoring express lane auction contract via resolvedRounds")

		ticker := time.NewTicker(t.pollInterval)
		defer ticker.Stop()

		var highestSeenRound uint64

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			records := t.readResolvedRounds(ctx)
			if len(records) == 0 {
				continue
			}

			for _, r := range records {
				if r.controller == (common.Address{}) || r.round == 0 {
					continue
				}

				if r.round > highestSeenRound {
					highestSeenRound = r.round

					timeSinceAuctionClose := t.roundTimingInfo.AuctionClosing - t.roundTimingInfo.TimeTilNextRound()
					auctionResolutionLatency.Update(timeSinceAuctionClose.Nanoseconds())
					log.Info(
						"AuctionResolved: New express lane controller assigned",
						"round", r.round,
						"controller", r.controller,
						"timeSinceAuctionClose", timeSinceAuctionClose,
					)

					t.roundControl.Store(r.round, r.controller)
				}
			}
		}
	})

	t.roundHeartbeatThread()
}

func (t *ExpressLaneTracker) roundHeartbeatThread() {
	t.LaunchThread(func(ctx context.Context) {
		// Log every new express lane auction round.
		log.Info("Watching for new express lane rounds")

		// Wait until the next round starts
		waitTime := t.roundTimingInfo.TimeTilNextRound()
		select {
		case <-ctx.Done():
			return
		case <-time.After(waitTime):
		}

		// First tick happened, now set up regular ticks
		ticker := time.NewTicker(t.roundTimingInfo.Round)
		defer ticker.Stop()
		for {
			var ti time.Time
			select {
			case <-ctx.Done():
				return
			case ti = <-ticker.C:
			}

			round := t.roundTimingInfo.RoundNumber()
			log.Info(
				"New express lane auction round",
				"round", round,
				"timestamp", ti,
			)

			// Cleanup previous round controller data
			t.roundControl.Delete(round - 1)
		}
	})
}

// resolvedRecord is a helper for parsed resolvedRounds entries
type resolvedRecord struct {
	round      uint64
	controller common.Address
}

// readResolvedRounds calls the contract’s 2-slot buffer:
// ResolvedRounds(opts *bind.CallOpts) (ELCRound, ELCRound, error)
// where ELCRound corresponds to Solidity tuple (address, uint64).
// It deduplicates and returns entries ordered by round ascending.
func (t *ExpressLaneTracker) readResolvedRounds(parentCtx context.Context) []resolvedRecord {
	// Per-call timeout shorter than poll interval to avoid slow node stalling the loop
	timeout := t.pollInterval / 2
	if timeout <= 0 {
		timeout = 2 * time.Second // default timeout - 2 seconds
	}
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	r0, r1, err := t.auctionContract.ResolvedRounds(&bind.CallOpts{Context: ctx})
	if err != nil {
		log.Warn("resolvedRounds call failed", "err", err)
		return nil
	}

	toResolvedRecord := func(r express_lane_auctiongen.ELCRound) (resolvedRecord, bool) {
		controller := r.ExpressLaneController
		round := r.Round

		if controller == (common.Address{}) || round == 0 {
			return resolvedRecord{}, false
		}
		return resolvedRecord{round: round, controller: controller}, true
	}

	var records []resolvedRecord
	if record, ok := toResolvedRecord(r0); ok {
		records = append(records, record)
	}
	if record, ok := toResolvedRecord(r1); ok {
		if len(records) == 0 || records[0].round != record.round {
			// Prepend or append the record based on its round number
			if len(records) > 0 && record.round < records[0].round {
				records = []resolvedRecord{record, records[0]}
			} else {
				records = append(records, record)
			}
		}
	}
	return records
}
