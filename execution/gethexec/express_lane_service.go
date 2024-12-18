// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type expressLaneControl struct {
	sequence   uint64
	controller common.Address
}

type transactionPublisher interface {
	PublishTimeboostedTransaction(context.Context, *types.Transaction, *arbitrum_types.ConditionalOptions) error
}

type expressLaneService struct {
	stopwaiter.StopWaiter
	sync.RWMutex
	transactionPublisher     transactionPublisher
	auctionContractAddr      common.Address
	apiBackend               *arbitrum.APIBackend
	roundTimingInfo          timeboost.RoundTimingInfo
	earlySubmissionGrace     time.Duration
	chainConfig              *params.ChainConfig
	logs                     chan []*types.Log
	auctionContract          *express_lane_auctiongen.ExpressLaneAuction
	roundControl             *lru.Cache[uint64, *expressLaneControl] // thread safe
	messagesBySequenceNumber map[uint64]*timeboost.ExpressLaneSubmission
}

type contractAdapter struct {
	*filters.FilterAPI
	bind.ContractTransactor // We leave this member unset as it is not used.

	apiBackend *arbitrum.APIBackend
}

func (a *contractAdapter) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	logPointers, err := a.GetLogs(ctx, filters.FilterCriteria(q))
	if err != nil {
		return nil, err
	}
	logs := make([]types.Log, 0, len(logPointers))
	for _, log := range logPointers {
		logs = append(logs, *log)
	}
	return logs, nil
}

func (a *contractAdapter) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	panic("contractAdapter doesn't implement SubscribeFilterLogs - shouldn't be needed")
}

func (a *contractAdapter) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	panic("contractAdapter doesn't implement CodeAt - shouldn't be needed")
}

func (a *contractAdapter) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var num rpc.BlockNumber = rpc.LatestBlockNumber
	if blockNumber != nil {
		num = rpc.BlockNumber(blockNumber.Int64())
	}

	state, header, err := a.apiBackend.StateAndHeaderByNumber(ctx, num)
	if err != nil {
		return nil, err
	}

	msg := &core.Message{
		From:              call.From,
		To:                call.To,
		Value:             big.NewInt(0),
		GasLimit:          math.MaxUint64,
		GasPrice:          big.NewInt(0),
		GasFeeCap:         big.NewInt(0),
		GasTipCap:         big.NewInt(0),
		Data:              call.Data,
		AccessList:        call.AccessList,
		SkipAccountChecks: true,
		TxRunMode:         core.MessageEthcallMode, // Indicate this is an eth_call
		SkipL1Charging:    true,                    // Skip L1 data fees
	}

	evm := a.apiBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true}, nil)
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	return result.ReturnData, nil
}

func newExpressLaneService(
	transactionPublisher transactionPublisher,
	apiBackend *arbitrum.APIBackend,
	filterSystem *filters.FilterSystem,
	auctionContractAddr common.Address,
	bc *core.BlockChain,
	earlySubmissionGrace time.Duration,
) (*expressLaneService, error) {
	chainConfig := bc.Config()

	var contractBackend bind.ContractBackend = &contractAdapter{filters.NewFilterAPI(filterSystem), nil, apiBackend}

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	retries := 0

pending:
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	if err != nil {
		const maxRetries = 5
		if errors.Is(err, bind.ErrNoCode) && retries < maxRetries {
			wait := time.Millisecond * 250 * (1 << retries)
			log.Info("ExpressLaneAuction contract not ready, will retry afer wait", "err", err, "auctionContractAddr", auctionContractAddr, "wait", wait, "maxRetries", maxRetries)
			retries++
			time.Sleep(wait)
			goto pending
		}
		return nil, err
	}
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	if err != nil {
		return nil, err
	}

	return &expressLaneService{
		transactionPublisher:     transactionPublisher,
		auctionContract:          auctionContract,
		apiBackend:               apiBackend,
		chainConfig:              chainConfig,
		roundTimingInfo:          *roundTimingInfo,
		earlySubmissionGrace:     earlySubmissionGrace,
		roundControl:             lru.NewCache[uint64, *expressLaneControl](8), // Keep 8 rounds cached.
		auctionContractAddr:      auctionContractAddr,
		logs:                     make(chan []*types.Log, 10_000),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
	}, nil
}

func (es *expressLaneService) Start(ctxIn context.Context) {
	es.StopWaiter.Start(ctxIn, es)

	// Log every new express lane auction round.
	es.LaunchThread(func(ctx context.Context) {
		log.Info("Watching for new express lane rounds")
		waitTime := es.roundTimingInfo.TimeTilNextRound()
		// Wait until the next round starts
		select {
		case <-ctx.Done():
			return
		case <-time.After(waitTime):
			// First tick happened, now set up regular ticks
		}

		ticker := time.NewTicker(es.roundTimingInfo.Round)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				round := es.roundTimingInfo.RoundNumber()
				// TODO (BUG?) is there a race here where messages for a new round can come
				// in before this tick has been processed?
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
		log.Info("Monitoring express lane auction contract")
		// Monitor for auction resolutions from the auction manager smart contract
		// and set the express lane controller for the upcoming round accordingly.
		latestBlock, err := es.apiBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
		if err != nil {
			// TODO: Should not be a crit.
			log.Crit("Could not get latest header", "err", err)
		}
		fromBlock := latestBlock.Number.Uint64()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond * 250):
				latestBlock, err := es.apiBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
				if err != nil {
					log.Crit("Could not get latest header", "err", err)
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
					log.Error("Could not filter auction resolutions event", "error", err)
					continue
				}
				for it.Next() {
					log.Info(
						"AuctionResolved: New express lane controller assigned",
						"round", it.Event.Round,
						"controller", it.Event.FirstPriceExpressLaneController,
					)
					es.roundControl.Add(it.Event.Round, &expressLaneControl{
						controller: it.Event.FirstPriceExpressLaneController,
						sequence:   0,
					})
				}

				setExpressLaneIterator, err := es.auctionContract.FilterSetExpressLaneController(filterOpts, nil, nil, nil)
				if err != nil {
					log.Error("Could not filter express lane controller transfer event", "error", err)
					continue
				}

				for setExpressLaneIterator.Next() {
					if (setExpressLaneIterator.Event.PreviousExpressLaneController == common.Address{}) {
						// The ExpressLaneAuction contract emits both AuctionResolved and SetExpressLaneController
						// events when an auction is resolved. They contain redundant information so
						// the SetExpressLaneController event can be skipped if it's related to a new round, as
						// indicated by an empty PreviousExpressLaneController field (a new round has no
						// previous controller).
						// It is more explicit and thus clearer to use the AuctionResovled event only for the
						// new round setup logic and SetExpressLaneController event only for transfers, rather
						// than trying to overload everything onto SetExpressLaneController.
						continue
					}
					round := setExpressLaneIterator.Event.Round
					roundInfo, ok := es.roundControl.Get(round)
					if !ok {
						log.Warn("Could not find round info for ExpressLaneConroller transfer event", "round", round)
						continue
					}
					prevController := setExpressLaneIterator.Event.PreviousExpressLaneController
					if roundInfo.controller != prevController {
						log.Warn("Previous ExpressLaneController in SetExpressLaneController event does not match Sequencer previous controller, continuing with transfer to new controller anyway",
							"round", round,
							"sequencerRoundController", roundInfo.controller,
							"previous", setExpressLaneIterator.Event.PreviousExpressLaneController,
							"new", setExpressLaneIterator.Event.NewExpressLaneController)
					}
					if roundInfo.controller == setExpressLaneIterator.Event.NewExpressLaneController {
						log.Warn("SetExpressLaneController: Previous and New ExpressLaneControllers are the same, not transferring control.",
							"round", round,
							"previous", setExpressLaneIterator.Event.PreviousExpressLaneController,
							"new", setExpressLaneIterator.Event.NewExpressLaneController)
						continue
					}

					es.Lock()
					// Changes to roundControl by itself are atomic but we need to udpate both roundControl
					// and messagesBySequenceNumber atomically here.
					es.roundControl.Add(round, &expressLaneControl{
						controller: setExpressLaneIterator.Event.NewExpressLaneController,
						sequence:   0,
					})
					// Since the sequence number for this round has been reset to zero, the map of messages
					// by sequence number must be reset otherwise old messages would be replayed.
					es.messagesBySequenceNumber = make(map[uint64]*timeboost.ExpressLaneSubmission)
					es.Unlock()
				}
				fromBlock = toBlock
			}
		}
	})
}

func (es *expressLaneService) currentRoundHasController() bool {
	control, ok := es.roundControl.Get(es.roundTimingInfo.RoundNumber())
	if !ok {
		return false
	}
	return control.controller != (common.Address{})
}

// Sequence express lane submission skips validation of the express lane message itself,
// as the core validator logic is handled in `validateExpressLaneTx“
func (es *expressLaneService) sequenceExpressLaneSubmission(
	ctx context.Context,
	msg *timeboost.ExpressLaneSubmission,
) error {
	es.Lock()
	defer es.Unlock()
	// Although access to roundControl by itself is thread-safe, when the round control is transferred
	// we need to reset roundControl and messagesBySequenceNumber atomically, so the following access
	// must be within the lock.
	control, ok := es.roundControl.Get(msg.Round)
	if !ok {
		return timeboost.ErrNoOnchainController
	}

	// Check if the submission nonce is too low.
	if msg.SequenceNumber < control.sequence {
		return timeboost.ErrSequenceNumberTooLow
	}
	// Check if a duplicate submission exists already, and reject if so.
	if _, exists := es.messagesBySequenceNumber[msg.SequenceNumber]; exists {
		return timeboost.ErrDuplicateSequenceNumber
	}
	// Log an informational warning if the message's sequence number is in the future.
	if msg.SequenceNumber > control.sequence {
		log.Info("Received express lane submission with future sequence number", "SequenceNumber", msg.SequenceNumber)
	}
	// Put into the sequence number map.
	es.messagesBySequenceNumber[msg.SequenceNumber] = msg

	for {
		// Get the next message in the sequence.
		nextMsg, exists := es.messagesBySequenceNumber[control.sequence]
		if !exists {
			break
		}
		if err := es.transactionPublisher.PublishTimeboostedTransaction(
			ctx,
			nextMsg.Transaction,
			msg.Options,
		); err != nil {
			// If the tx failed, clear it from the sequence map.
			delete(es.messagesBySequenceNumber, msg.SequenceNumber)
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

	for {
		currentRound := es.roundTimingInfo.RoundNumber()
		if msg.Round == currentRound {
			break
		}

		timeTilNextRound := es.roundTimingInfo.TimeTilNextRound()
		// We allow txs to come in for the next round if it is close enough to that round,
		// but we sleep until the round starts.
		if msg.Round == currentRound+1 && timeTilNextRound <= es.earlySubmissionGrace {
			time.Sleep(timeTilNextRound)
		} else {
			return errors.Wrapf(timeboost.ErrBadRoundNumber, "express lane tx round %d does not match current round %d", msg.Round, currentRound)
		}
	}
	if !es.currentRoundHasController() {
		return timeboost.ErrNoOnchainController
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

	// Signature verification expects the last byte of the signature to have 27 subtracted,
	// as it represents the recovery ID. If the last byte is greater than or equal to 27, it indicates a recovery ID that hasn't been adjusted yet,
	// it's needed for internal signature verification logic.
	if sigItem[len(sigItem)-1] >= 27 {
		sigItem[len(sigItem)-1] -= 27
	}
	pubkey, err := crypto.SigToPub(prefixed, sigItem)
	if err != nil {
		return timeboost.ErrMalformedData
	}
	sender := crypto.PubkeyToAddress(*pubkey)
	control, ok := es.roundControl.Get(msg.Round)
	if !ok {
		return timeboost.ErrNoOnchainController
	}
	if sender != control.controller {
		return timeboost.ErrNotExpressLaneController
	}
	return nil
}
