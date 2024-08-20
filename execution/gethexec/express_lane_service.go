package gethexec

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
)

type expressLaneControl struct {
	sequence   uint64
	controller common.Address
}

type expressLaneService struct {
	stopwaiter.StopWaiter
	sync.RWMutex
	auctionContractAddr      common.Address
	initialTimestamp         time.Time
	roundDuration            time.Duration
	auctionClosing           time.Duration
	bc                       *core.BlockChain
	chainConfig              *params.ChainConfig
	logs                     chan []*types.Log
	seqClient                *ethclient.Client
	roundControl             lru.BasicLRU[uint64, *expressLaneControl]
	messagesBySequenceNumber map[uint64]*timeboost.ExpressLaneSubmission
	auctionAbi               *abi.ABI
}

func newExpressLaneService(
	auctionContractAddr common.Address,
	bc *core.BlockChain,
	roundDuration time.Duration,
	initialTimestamp time.Time,
	auctionClosingDuration time.Duration,
) (*expressLaneService, error) {
	chainConfig := bc.Config()
	eabi, err := express_lane_auctiongen.ExpressLaneAuctionMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &expressLaneService{
		auctionAbi:               eabi,
		chainConfig:              chainConfig,
		bc:                       bc,
		initialTimestamp:         initialTimestamp,
		auctionClosing:           auctionClosingDuration,
		roundControl:             lru.NewBasicLRU[uint64, *expressLaneControl](8), // Keep 8 rounds cached.
		auctionContractAddr:      auctionContractAddr,
		roundDuration:            roundDuration,
		logs:                     make(chan []*types.Log, 10_000),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
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
				es.Lock()
				// Reset the sequence numbers map for the new round.
				es.messagesBySequenceNumber = make(map[uint64]*timeboost.ExpressLaneSubmission)
				es.Unlock()
			}
		}
	})
	es.LaunchThread(func(ctx context.Context) {
		log.Info("Monitoring express lane auction contract", "contract", es.auctionContractAddr)
		logs := make(chan []*types.Log)
		sub := es.bc.SubscribeLogsEvent(logs)
		defer sub.Unsubscribe()
		for {
			select {
			case <-ctx.Done():
				log.Info("Closed log processing context")
				return
			case logs := <-es.logs:
				go es.processLogs(ctx, logs)
			}
		}
	})
}

func (es *expressLaneService) processLogs(ctx context.Context, logs []*types.Log) {
	resolvedTopic := es.auctionAbi.Events["AuctionResolved"].ID
	for _, lg := range logs {
		if ctx.Err() != nil {
			log.Error("Context is done, stopping log processing")
			return
		}
		if lg.Address != es.auctionContractAddr || lg.Removed {
			log.Info("Skipping log that is not targeting auction contract")
			continue
		}
		if !slices.Contains(lg.Topics, resolvedTopic) {
			log.Info("Skipping log that does not contain topic")
			continue
		}
		auctionResolvedLog := new(express_lane_auctiongen.ExpressLaneAuctionAuctionResolved)
		if err := unpackLog(auctionResolvedLog, "AuctionResolved", *lg, es.auctionAbi); err != nil {
			log.Error("Could not unpack log", "error", err)
			continue
		}
		log.Info(
			"New express lane controller assigned",
			"round", auctionResolvedLog.Round,
			"controller", auctionResolvedLog.FirstPriceExpressLaneController,
		)
		es.Lock()
		es.roundControl.Add(auctionResolvedLog.Round, &expressLaneControl{
			controller: auctionResolvedLog.FirstPriceExpressLaneController,
			sequence:   0,
		})
		es.Unlock()
	}
}

func (es *expressLaneService) currentRoundHasController() bool {
	es.Lock()
	defer es.Unlock()
	currRound := timeboost.CurrentRound(es.initialTimestamp, es.roundDuration)
	control, ok := es.roundControl.Get(currRound)
	if !ok {
		return false
	}
	return control.controller != (common.Address{})
}

func (es *expressLaneService) isWithinAuctionCloseWindow(arrivalTime time.Time) bool {
	// Calculate the next round start time
	elapsedTime := arrivalTime.Sub(es.initialTimestamp)
	elapsedRounds := elapsedTime / es.roundDuration
	nextRoundStart := es.initialTimestamp.Add((elapsedRounds + 1) * es.roundDuration)
	// Calculate the time to the next round
	timeToNextRound := nextRoundStart.Sub(arrivalTime)
	// Check if the arrival timestamp is within AUCTION_CLOSING_DURATION of TIME_TO_NEXT_ROUND
	return timeToNextRound <= es.auctionClosing
}

func (es *expressLaneService) sequenceExpressLaneSubmission(
	ctx context.Context,
	msg *timeboost.ExpressLaneSubmission,
	publishTxFn func(
		parentCtx context.Context,
		tx *types.Transaction,
		options *arbitrum_types.ConditionalOptions,
		delay bool,
	) error,
) error {
	es.Lock()
	defer es.Unlock()
	control, ok := es.roundControl.Get(msg.Round)
	if !ok {
		return timeboost.ErrNoOnchainController
	}
	// Check if the submission nonce is too low.
	if msg.Sequence < control.sequence {
		return timeboost.ErrSequenceNumberTooLow
	}
	// Check if a duplicate submission exists already, and reject if so.
	if _, exists := es.messagesBySequenceNumber[msg.Sequence]; exists {
		return timeboost.ErrDuplicateSequenceNumber
	}
	// Log an informational warning if the message's sequence number is in the future.
	if msg.Sequence > control.sequence {
		log.Warn("Received express lane submission with future sequence number", "sequence", msg.Sequence)
	}
	// Put into the the sequence number map.
	es.messagesBySequenceNumber[msg.Sequence] = msg

	for {
		// Get the next message in the sequence.
		nextMsg, exists := es.messagesBySequenceNumber[control.sequence]
		if !exists {
			break
		}
		if err := publishTxFn(
			ctx,
			nextMsg.Transaction,
			msg.Options,
			false, /* no delay, as it should go through express lane */
		); err != nil {
			// If the tx failed, clear it from the sequence map.
			delete(es.messagesBySequenceNumber, msg.Sequence)
			return err
		}
		// Increase the global round sequence number.
		control.sequence += 1
	}
	es.roundControl.Add(msg.Round, control)
	return nil
}

func (es *expressLaneService) validateExpressLaneTx(msg *timeboost.ExpressLaneSubmission) error {
	if msg == nil || msg.Transaction == nil || msg.Signature == nil {
		return timeboost.ErrMalformedData
	}
	if msg.ChainId.Cmp(es.chainConfig.ChainID) != 0 {
		return errors.Wrapf(timeboost.ErrWrongChainId, "express lane tx chain ID %d does not match current chain ID %d", msg.ChainId, es.chainConfig.ChainID)
	}
	if msg.AuctionContractAddress != es.auctionContractAddr {
		return errors.Wrapf(timeboost.ErrWrongAuctionContract, "msg auction contract address %s does not match sequencer auction contract address %s", msg.AuctionContractAddress, es.auctionContractAddr)
	}
	if !es.currentRoundHasController() {
		return timeboost.ErrNoOnchainController
	}
	currentRound := timeboost.CurrentRound(es.initialTimestamp, es.roundDuration)
	if msg.Round != currentRound {
		return errors.Wrapf(timeboost.ErrBadRoundNumber, "express lane tx round %d does not match current round %d", msg.Round, currentRound)
	}
	// Reconstruct the message being signed over and recover the sender address.
	signingMessage, err := msg.ToMessageBytes()
	if err != nil {
		return timeboost.ErrMalformedData
	}
	if len(msg.Signature) != 65 {
		return errors.Wrap(timeboost.ErrMalformedData, "signature length is not 65")
	}
	// Recover the public key.
	prefixed := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(signingMessage))), signingMessage...))
	sigItem := make([]byte, len(msg.Signature))
	copy(sigItem, msg.Signature)
	if sigItem[len(sigItem)-1] >= 27 {
		sigItem[len(sigItem)-1] -= 27
	}
	pubkey, err := crypto.SigToPub(prefixed, sigItem)
	if err != nil {
		return timeboost.ErrMalformedData
	}
	sender := crypto.PubkeyToAddress(*pubkey)
	es.RLock()
	defer es.RUnlock()
	control, ok := es.roundControl.Get(msg.Round)
	if !ok {
		return timeboost.ErrNoOnchainController
	}
	if sender != control.controller {
		return timeboost.ErrNotExpressLaneController
	}
	return nil
}

func unpackLog(out any, event string, log types.Log, eabi *abi.ABI) error {
	if len(log.Topics) == 0 {
		return errors.New("no topics")
	}
	if log.Topics[0] != eabi.Events[event].ID {
		return errors.New("wrong topic")
	}
	if len(log.Data) > 0 {
		if err := eabi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range eabi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(out, indexed, log.Topics[1:])
}
