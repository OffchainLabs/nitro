package gethexec

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type expressLaneControl struct {
	round      uint64
	sequence   uint64
	controller common.Address
}

type expressLaneService struct {
	stopwaiter.StopWaiter
	sync.RWMutex
	client           arbutil.L1Interface
	control          expressLaneControl
	expressLaneAddr  common.Address
	auctionContract  *express_lane_auctiongen.ExpressLaneAuction
	initialTimestamp time.Time
	roundDuration    time.Duration
	chainConfig      *params.ChainConfig
}

func newExpressLaneService(
	client arbutil.L1Interface,
	auctionContractAddr common.Address,
	chainConfig *params.ChainConfig,
) (*expressLaneService, error) {
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
	return &expressLaneService{
		auctionContract:  auctionContract,
		client:           client,
		chainConfig:      chainConfig,
		initialTimestamp: initialTimestamp,
		control: expressLaneControl{
			controller: common.Address{},
			round:      0,
		},
		expressLaneAddr: common.HexToAddress("0x2424242424242424242424242424242424242424"),
		roundDuration:   roundDuration,
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
				it, err := es.auctionContract.FilterAuctionResolved(filterOpts, nil, nil, nil)
				if err != nil {
					log.Error("Could not filter auction resolutions", "error", err)
					continue
				}
				for it.Next() {
					log.Info(
						"New express lane controller assigned",
						"round", it.Event.Round,
						"controller", it.Event.FirstPriceExpressLaneController,
					)
					es.Lock()
					es.control.round = it.Event.Round
					es.control.controller = it.Event.FirstPriceExpressLaneController
					es.control.sequence = 0 // Sequence resets 0 for the new round.
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

func (es *expressLaneService) currentRoundHasController() bool {
	es.Lock()
	defer es.Unlock()
	return es.control.controller != (common.Address{})
}

// An express lane transaction is valid if it satisfies the following conditions:
// 1. The tx round expressed under `maxPriorityFeePerGas` equals the current round number.
// 2. The tx sequence expressed under `nonce` equals the current round sequence.
// 3. The tx sender equals the current round’s priority controller address.
func (es *expressLaneService) validateExpressLaneTx(msg *arbitrum_types.ExpressLaneSubmission) error {
	es.Lock()
	defer es.Unlock()

	currentRound := timeboost.CurrentRound(es.initialTimestamp, es.roundDuration)
	if msg.Round != currentRound {
		return fmt.Errorf("express lane tx round %d does not match current round %d", msg.Round, currentRound)
	}
	// TODO: recover the sender from the signature and message bytes that are being signed over.
	// signer := types.LatestSigner(es.chainConfig)
	// sender, err := types.Sender(signer, tx)
	// if err != nil {
	// 	return err
	// }
	// if sender != es.control.controller {
	// 	return fmt.Errorf("express lane tx sender %s does not match current round controller %s", sender, es.control.controller)
	// }
	return nil
}

// unwrapExpressLaneTx extracts the inner "wrapped" transaction from the data field of an express lane transaction.
func unwrapExpressLaneTx(tx *types.Transaction) (*types.Transaction, error) {
	encodedInnerTx := tx.Data()
	fmt.Printf("Inner in decoding: %#x\n", encodedInnerTx)
	innerTx := &types.Transaction{}
	if err := innerTx.UnmarshalBinary(encodedInnerTx); err != nil {
		return nil, fmt.Errorf("failed to decode inner transaction: %w", err)
	}
	return innerTx, nil
}
