// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
)

// 95% of the SequencerInbox limit, leaving ~5KB for headers and such
const maxTxDataSize = 112065

type SequencerConfig struct {
	Enable                      bool                     `koanf:"enable"`
	MaxBlockSpeed               time.Duration            `koanf:"max-block-speed" reload:"hot"`
	MaxRevertGasReject          uint64                   `koanf:"max-revert-gas-reject" reload:"hot"`
	MaxAcceptableTimestampDelta time.Duration            `koanf:"max-acceptable-timestamp-delta" reload:"hot"`
	SenderWhitelist             string                   `koanf:"sender-whitelist"`
	Forwarder                   ForwarderConfig          `koanf:"forwarder"`
	QueueSize                   int                      `koanf:"queue-size"`
	Dangerous                   DangerousSequencerConfig `koanf:"dangerous"`
}

func (c *SequencerConfig) Validate() error {
	entries := strings.Split(c.SenderWhitelist, ",")
	for _, address := range entries {
		if len(address) == 0 {
			continue
		}
		if !common.IsHexAddress(address) {
			return fmt.Errorf("sequencer sender whitelist entry \"%v\" is not a valid address", address)
		}
	}
	return nil
}

type SequencerConfigFetcher func() *SequencerConfig

var DefaultSequencerConfig = SequencerConfig{
	Enable:                      false,
	MaxBlockSpeed:               time.Millisecond * 100,
	MaxRevertGasReject:          params.TxGas + 10000,
	MaxAcceptableTimestampDelta: time.Hour,
	Forwarder:                   DefaultSequencerForwarderConfig,
	QueueSize:                   1024,
	Dangerous:                   DefaultDangerousSequencerConfig,
}

var TestSequencerConfig = SequencerConfig{
	Enable:                      true,
	MaxBlockSpeed:               time.Millisecond * 10,
	MaxRevertGasReject:          params.TxGas + 10000,
	MaxAcceptableTimestampDelta: time.Hour,
	SenderWhitelist:             "",
	Forwarder:                   DefaultTestForwarderConfig,
	QueueSize:                   128,
	Dangerous:                   TestDangerousSequencerConfig,
}

func SequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSequencerConfig.Enable, "act and post to l1 as sequencer")
	f.Duration(prefix+".max-block-speed", DefaultSequencerConfig.MaxBlockSpeed, "minimum delay between blocks (sets a maximum speed of block production)")
	f.Uint64(prefix+".max-revert-gas-reject", DefaultSequencerConfig.MaxRevertGasReject, "maximum gas executed in a revert for the sequencer to reject the transaction instead of posting it (anti-DOS)")
	f.Duration(prefix+".max-acceptable-timestamp-delta", DefaultSequencerConfig.MaxAcceptableTimestampDelta, "maximum acceptable time difference between the local time and the latest L1 block's timestamp")
	f.String(prefix+".sender-whitelist", DefaultSequencerConfig.SenderWhitelist, "comma separated whitelist of authorized senders (if empty, everyone is allowed)")
	AddOptionsForSequencerForwarderConfig(prefix+".forwarder", f)
	f.Int(prefix+".queue-size", DefaultSequencerConfig.QueueSize, "size of the pending tx queue")
	DangerousSequencerConfigAddOptions(prefix+".dangerous", f)
}

type txQueueItem struct {
	tx             *types.Transaction
	resultChan     chan<- error
	returnedResult bool
	ctx            context.Context
}

func (i *txQueueItem) returnResult(err error) {
	if i.returnedResult {
		log.Error("attempting to return result to already finished queue item", "err", err)
		return
	}
	i.returnedResult = true
	i.resultChan <- err
	close(i.resultChan)
}

type Sequencer struct {
	stopwaiter.StopWaiter

	txStreamer      *TransactionStreamer
	txQueue         chan txQueueItem
	txRetryQueue    arbutil.Queue[txQueueItem]
	l1Reader        *headerreader.HeaderReader
	config          SequencerConfigFetcher
	senderWhitelist map[common.Address]struct{}

	L1BlockAndTimeMutex sync.Mutex
	l1BlockNumber       uint64
	l1Timestamp         uint64

	forwarderMutex sync.Mutex
	forwarder      *TxForwarder
}

func NewSequencer(txStreamer *TransactionStreamer, l1Reader *headerreader.HeaderReader, configFetcher SequencerConfigFetcher) (*Sequencer, error) {
	config := configFetcher()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	senderWhitelist := make(map[common.Address]struct{})
	entries := strings.Split(config.SenderWhitelist, ",")
	for _, address := range entries {
		if len(address) == 0 {
			continue
		}
		senderWhitelist[common.HexToAddress(address)] = struct{}{}
	}
	return &Sequencer{
		txStreamer:      txStreamer,
		txQueue:         make(chan txQueueItem, config.QueueSize),
		l1Reader:        l1Reader,
		config:          configFetcher,
		senderWhitelist: senderWhitelist,
		l1BlockNumber:   0,
		l1Timestamp:     0,
	}, nil
}

var ErrRetrySequencer = errors.New("please retry transaction")

func (s *Sequencer) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	forwarder := s.GetForwarder()
	if forwarder != nil {
		err := forwarder.PublishTransaction(ctx, tx)
		if !errors.Is(err, ErrNoSequencer) {
			return err
		}
	}

	if len(s.senderWhitelist) > 0 {
		signer := types.LatestSigner(s.txStreamer.bc.Config())
		sender, err := types.Sender(signer, tx)
		if err != nil {
			return err
		}
		_, authorized := s.senderWhitelist[sender]
		if !authorized {
			return errors.New("transaction sender is not on the whitelist")
		}
	}
	if tx.Type() >= types.ArbitrumDepositTxType {
		// Should be unreachable due to UnmarshalBinary not accepting Arbitrum internal txs
		return types.ErrTxTypeNotSupported
	}

	resultChan := make(chan error, 1)
	queueItem := txQueueItem{
		tx,
		resultChan,
		false,
		ctx,
	}
	select {
	case s.txQueue <- queueItem:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case res := <-resultChan:
		return res
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Sequencer) preTxFilter(_ *params.ChainConfig, _ *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, _ *types.Transaction) error {
	return nil
}

func (s *Sequencer) postTxFilter(_ *arbosState.ArbosState, _ *types.Transaction, _ common.Address, dataGas uint64, result *core.ExecutionResult) error {
	if result.Err != nil && result.UsedGas > dataGas && result.UsedGas-dataGas <= s.config().MaxRevertGasReject {
		return arbitrum.NewRevertReason(result)
	}
	return nil
}

func (s *Sequencer) CheckHealth(ctx context.Context) error {
	s.forwarderMutex.Lock()
	forwarder := s.forwarder
	s.forwarderMutex.Unlock()
	if forwarder != nil {
		return forwarder.CheckHealth(ctx)
	}

	if s.txStreamer.coordinator != nil && !s.txStreamer.coordinator.CurrentlyChosen() {
		return ErrNoSequencer
	}
	return nil
}

func (s *Sequencer) ForwardTarget() string {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder == nil {
		return ""
	}
	return s.forwarder.target
}

func (s *Sequencer) ForwardTo(url string) error {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder != nil {
		if s.forwarder.target == url {
			log.Warn("attempted to update sequencer forward target with existing target", "url", url)
			return nil
		}
		s.forwarder.Disable()
	}
	s.forwarder = NewForwarder(url, &s.config().Forwarder)
	err := s.forwarder.Initialize(s.GetContext())
	if err != nil {
		log.Error("failed to set forward agent", "err", err)
		s.forwarder = nil
	}
	return err
}

func (s *Sequencer) DontForward() {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder != nil {
		s.forwarder.Disable()
	}
	s.forwarder = nil
}

var ErrNoSequencer = errors.New("sequencer temporarily not available")

func (s *Sequencer) requeueOrFail(queueItem txQueueItem, err error) {
	select {
	case s.txQueue <- queueItem:
	default:
		queueItem.returnResult(err)
	}
}

func (s *Sequencer) GetForwarder() *TxForwarder {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	return s.forwarder
}

func (s *Sequencer) forwardIfSet(queueItems []txQueueItem) bool {
	forwarder := s.GetForwarder()
	if forwarder == nil {
		return false
	}
	for _, item := range queueItems {
		res := forwarder.PublishTransaction(item.ctx, item.tx)
		if errors.Is(res, ErrNoSequencer) {
			s.requeueOrFail(item, ErrNoSequencer)
		} else {
			item.returnResult(res)
		}
	}
	return true
}

var sequencerInternalError = errors.New("sequencer internal error")

func (s *Sequencer) createBlock(ctx context.Context) (returnValue bool) {
	var txes types.Transactions
	var queueItems []txQueueItem
	var totalBatchSize int

	defer func() {
		panic := recover()
		if panic != nil {
			log.Error("sequencer block creation panicked", "panic", panic, "backtrace", string(debug.Stack()))
			// Return an internal error to any queue items we were trying to process
			for _, item := range queueItems {
				if !item.returnedResult {
					item.returnResult(sequencerInternalError)
				}
			}
			// Wait for the MaxBlockSpeed until attempting to create a block again
			returnValue = true
		}
	}()

	for {
		var queueItem txQueueItem
		if s.txRetryQueue.Len() > 0 {
			queueItem = s.txRetryQueue.Pop()
		} else if len(txes) == 0 {
			select {
			case queueItem = <-s.txQueue:
			case <-ctx.Done():
				return false
			}
		} else {
			done := false
			select {
			case queueItem = <-s.txQueue:
			default:
				done = true
			}
			if done {
				break
			}
		}
		err := queueItem.ctx.Err()
		if err != nil {
			queueItem.returnResult(err)
			continue
		}
		txBytes, err := queueItem.tx.MarshalBinary()
		if err != nil {
			queueItem.returnResult(err)
			continue
		}
		if len(txBytes) > maxTxDataSize {
			// This tx is too large
			queueItem.returnResult(core.ErrOversizedData)
			continue
		}
		if totalBatchSize+len(txBytes) > maxTxDataSize {
			// This tx would be too large to add to this batch
			s.txRetryQueue.Push(queueItem)
			// End the batch here to put this tx in the next one
			break
		}
		totalBatchSize += len(txBytes)
		txes = append(txes, queueItem.tx)
		queueItems = append(queueItems, queueItem)
	}

	if s.forwardIfSet(queueItems) {
		return false
	}

	timestamp := time.Now().Unix()
	s.L1BlockAndTimeMutex.Lock()
	l1Block := s.l1BlockNumber
	l1Timestamp := s.l1Timestamp
	s.L1BlockAndTimeMutex.Unlock()

	if s.l1Reader != nil && (l1Block == 0 || math.Abs(float64(l1Timestamp)-float64(timestamp)) > s.config().MaxAcceptableTimestampDelta.Seconds()) {
		log.Error(
			"cannot sequence: unknown L1 block or L1 timestamp too far from local clock time",
			"l1Block", l1Block,
			"l1Timestamp", l1Timestamp,
			"localTimestamp", timestamp,
		)
		return false
	}

	header := &arbos.L1IncomingMessageHeader{
		Kind:        arbos.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: l1Block,
		Timestamp:   uint64(timestamp),
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	hooks := &arbos.SequencingHooks{
		PreTxFilter:            s.preTxFilter,
		PostTxFilter:           s.postTxFilter,
		DiscardInvalidTxsEarly: true,
		TxErrors:               []error{},
	}
	err := s.txStreamer.SequenceTransactions(header, txes, hooks)
	if err == nil && len(hooks.TxErrors) != len(txes) {
		err = fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(hooks.TxErrors), len(txes))
	}
	if errors.Is(err, ErrRetrySequencer) {
		// we changed roles
		// forward if we have where to
		if s.forwardIfSet(queueItems) {
			return false
		}
		// try to add back to queue otherwise
		for _, item := range queueItems {
			s.requeueOrFail(item, ErrNoSequencer)
		}
		return false
	}
	if err != nil {
		log.Warn("error sequencing transactions", "err", err)
		for _, queueItem := range queueItems {
			queueItem.returnResult(err)
		}
		return false
	}

	madeBlock := false
	for i, err := range hooks.TxErrors {
		if err == nil {
			madeBlock = true
		}
		queueItem := queueItems[i]
		if errors.Is(err, core.ErrGasLimitReached) {
			// There's not enough gas left in the block for this tx.
			if madeBlock && !errors.Is(err, arbos.ErrMaxGasLimitReached) {
				// There was already an earlier tx in the block; retry in a fresh block.
				s.txRetryQueue.Push(queueItem)
				continue
			}
		}
		if errors.Is(err, core.ErrIntrinsicGas) {
			// Strip additional information, as it's incorrect due to L1 data gas.
			err = core.ErrIntrinsicGas
		}
		queueItem.returnResult(err)
	}
	return madeBlock
}

func (s *Sequencer) updateLatestL1Block(header *types.Header) {
	s.L1BlockAndTimeMutex.Lock()
	defer s.L1BlockAndTimeMutex.Unlock()
	if s.l1BlockNumber < header.Number.Uint64() {
		s.l1BlockNumber = header.Number.Uint64()
		s.l1Timestamp = header.Time
	}
}

func (s *Sequencer) Initialize(ctx context.Context) error {
	if s.l1Reader == nil {
		return nil
	}

	header, err := s.l1Reader.LastHeader(ctx)
	if err != nil {
		return err
	}
	s.updateLatestL1Block(header)
	return nil
}

func (s *Sequencer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)
	if s.l1Reader != nil {
		initialBlockNr := atomic.LoadUint64(&s.l1BlockNumber)
		if initialBlockNr == 0 {
			return errors.New("sequencer not initialized")
		}

		headerChan, cancel := s.l1Reader.Subscribe(false)

		s.LaunchThread(func(ctx context.Context) {
			defer cancel()
			for {
				select {
				case header, ok := <-headerChan:
					if !ok {
						return
					}
					s.updateLatestL1Block(header)
				case <-ctx.Done():
					return
				}
			}
		})

	}

	s.CallIteratively(func(ctx context.Context) time.Duration {
		nextBlock := time.Now().Add(s.config().MaxBlockSpeed)
		madeBlock := s.createBlock(ctx)
		if madeBlock {
			// Note: this may return a negative duration, but timers are fine with that (they treat negative durations as 0).
			return time.Until(nextBlock)
		} else {
			// If we didn't make a block, try again immediately.
			return 0
		}
	})

	return nil
}
