// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"
	"encoding/json"
	"time"

	"github.com/offchainlabs/nitro/espresso"
	"github.com/offchainlabs/nitro/util/stopwaiter"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
)

var (
	retryTime = time.Second * 5
)

type HotShotState struct {
	client          espresso.Client
	nextSeqBlockNum uint64
}

func NewHotShotState(log log.Logger, url string) *HotShotState {
	return &HotShotState{
		client: *espresso.NewClient(log, url),
		// TODO: Load this from the inbox reader so that new sequencers don't read redundant blocks
		// https://github.com/EspressoSystems/espresso-sequencer/issues/734
		nextSeqBlockNum: 0,
	}
}

func (s *HotShotState) advance() {
	s.nextSeqBlockNum += 1
}

type EspressoSequencer struct {
	stopwaiter.StopWaiter

	execEngine   *ExecutionEngine
	config       SequencerConfigFetcher
	hotShotState *HotShotState
}

func NewEspressoSequencer(execEngine *ExecutionEngine, configFetcher SequencerConfigFetcher) (*EspressoSequencer, error) {
	config := configFetcher()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &EspressoSequencer{
		execEngine:   execEngine,
		config:       configFetcher,
		hotShotState: NewHotShotState(log.New(), config.HotShotUrl),
	}, nil
}

func (s *EspressoSequencer) makeSequencingHooks() *arbos.SequencingHooks {
	return &arbos.SequencingHooks{
		PreTxFilter:             s.preTxFilter,
		PostTxFilter:            s.postTxFilter,
		DiscardInvalidTxsEarly:  false,
		TxErrors:                []error{},
		ConditionalOptionsForTx: nil,
	}
}

func (s *EspressoSequencer) createBlock(ctx context.Context) (returnValue bool) {
	nextSeqBlockNum := s.hotShotState.nextSeqBlockNum
	log.Info("Attempting to sequence Espresso block", "block_num", nextSeqBlockNum)
	header, err := s.hotShotState.client.FetchHeader(ctx, nextSeqBlockNum)
	namespace := s.config().EspressoNamespace
	if err != nil {
		log.Warn("Unable to fetch header for block number, will retry", "block_num", nextSeqBlockNum)
		return false
	}
	arbTxns, err := s.hotShotState.client.FetchTransactionsInBlock(ctx, nextSeqBlockNum, &header, namespace)
	if err != nil {
		log.Error("Error fetching transactions", "err", err)
		return false

	}
	var txes types.Transactions
	for _, tx := range arbTxns.Transactions {
		var out types.Transaction
		if err := json.Unmarshal(tx, &out); err != nil {
			log.Error("Failed to serialize")
			return false
		}
		txes = append(txes, &out)

	}

	arbHeader := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: header.L1Head,
		Timestamp:   header.Timestamp,
		RequestId:   nil,
		L1BaseFee:   nil,
		// TODO: add justification https://github.com/EspressoSystems/espresso-sequencer/issues/733
	}

	hooks := s.makeSequencingHooks()
	_, err = s.execEngine.SequenceTransactions(arbHeader, txes, hooks)
	if err != nil {
		log.Error("Sequencing error for block number", "block_num", nextSeqBlockNum, "err", err)
		return false
	}

	s.hotShotState.advance()

	return true

}

func (s *EspressoSequencer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)
	s.CallIteratively(func(ctx context.Context) time.Duration {
		retryBlockTime := time.Now().Add(retryTime)
		madeBlock := s.createBlock(ctx)
		if madeBlock {
			// Allow the sequencer to catch up to HotShot
			return 0
		}
		// If we didn't make a block, try again in a bit
		return time.Until(retryBlockTime)
	})

	return nil
}

// Required methods for the TransactionPublisher interface
func (s *EspressoSequencer) PublishTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return nil
}

func (s *EspressoSequencer) CheckHealth(ctx context.Context) error {
	return nil
}

func (s *EspressoSequencer) Initialize(ctx context.Context) error {
	return nil
}

// ArbOS expects some preTxFilter, postTxFilter
func (s *EspressoSequencer) preTxFilter(_ *params.ChainConfig, _ *types.Header, _ *state.StateDB, _ *arbosState.ArbosState, _ *types.Transaction, _ *arbitrum_types.ConditionalOptions, _ common.Address, _ *arbos.L1Info) error {
	return nil
}

func (s *EspressoSequencer) postTxFilter(_ *types.Header, _ *arbosState.ArbosState, _ *types.Transaction, _ common.Address, _ uint64, _ *core.ExecutionResult) error {
	return nil
}
