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

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/headerreader"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
)

var (
	sequencerBacklogGauge     = metrics.NewRegisteredGauge("arb/sequencer/backlog", nil)
	nonceCacheHitCounter      = metrics.NewRegisteredCounter("arb/sequencer/noncecache/hit", nil)
	nonceCacheMissCounter     = metrics.NewRegisteredCounter("arb/sequencer/noncecache/miss", nil)
	nonceCacheRejectedCounter = metrics.NewRegisteredCounter("arb/sequencer/noncecache/rejected", nil)
	nonceCacheClearedCounter  = metrics.NewRegisteredCounter("arb/sequencer/noncecache/cleared", nil)
	blockCreationTimer        = metrics.NewRegisteredTimer("arb/sequencer/block/creation", nil)
	successfulBlocksCounter   = metrics.NewRegisteredCounter("arb/sequencer/block/successful", nil)
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
	QueueTimeout                time.Duration            `koanf:"queue-timeout" reload:"hot"`
	NonceCacheSize              int                      `koanf:"nonce-cache-size" reload:"hot"`
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
	QueueTimeout:                time.Second * 12,
	NonceCacheSize:              1024,
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
	QueueTimeout:                time.Second * 5,
	NonceCacheSize:              4,
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
	f.Duration(prefix+".queue-timeout", DefaultSequencerConfig.QueueTimeout, "maximum amount of time transaction can wait in queue")
	f.Int(prefix+".nonce-cache-size", DefaultSequencerConfig.NonceCacheSize, "size of the tx sender nonce cache")
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

type nonceCache struct {
	cache *containers.LruCache[common.Address, uint64]
	block common.Hash
	dirty *types.Header
}

func newNonceCache(size int) *nonceCache {
	return &nonceCache{
		cache: containers.NewLruCache[common.Address, uint64](size),
		block: common.Hash{},
		dirty: nil,
	}
}

func (c *nonceCache) matches(header *types.Header) bool {
	if c.dirty != nil {
		// The header is updated as the block is built,
		// so instead of checking its hash, we do a pointer comparison.
		return c.dirty == header
	} else {
		return c.block == header.ParentHash
	}
}

func (c *nonceCache) Reset(block common.Hash) {
	if c.cache.Len() > 0 {
		nonceCacheClearedCounter.Inc(1)
	}
	c.cache.Clear()
	c.block = block
	c.dirty = nil
}

func (c *nonceCache) BeginNewBlock() {
	if c.dirty != nil {
		c.Reset(common.Hash{})
	}
}

func (c *nonceCache) Get(header *types.Header, statedb *state.StateDB, addr common.Address) uint64 {
	if !c.matches(header) {
		c.Reset(header.ParentHash)
	}
	nonce, ok := c.cache.Get(addr)
	if ok {
		nonceCacheHitCounter.Inc(1)
		return nonce
	}
	nonceCacheMissCounter.Inc(1)
	nonce = statedb.GetNonce(addr)
	c.cache.Add(addr, nonce)
	return nonce
}

func (c *nonceCache) Update(header *types.Header, addr common.Address, nonce uint64) {
	if !c.matches(header) {
		c.Reset(header.ParentHash)
	}
	c.dirty = header
	c.cache.Add(addr, nonce)
}

func (c *nonceCache) Finalize(block *types.Block) {
	// Note: we don't use c.Matches here because the header will have changed
	if c.block == block.ParentHash() {
		c.block = block.Hash()
		c.dirty = nil
	} else {
		c.Reset(block.Hash())
	}
}

func (c *nonceCache) GetSize() int {
	return c.cache.GetSize()
}

func (c *nonceCache) Resize(newSize int) {
	c.cache.Resize(newSize)
}

type Sequencer struct {
	stopwaiter.StopWaiter

	txStreamer      *TransactionStreamer
	txQueue         chan txQueueItem
	txRetryQueue    containers.Queue[txQueueItem]
	l1Reader        *headerreader.HeaderReader
	config          SequencerConfigFetcher
	senderWhitelist map[common.Address]struct{}
	nonceCache      *nonceCache

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
		nonceCache:      newNonceCache(config.NonceCacheSize),
		l1BlockNumber:   0,
		l1Timestamp:     0,
	}, nil
}

var ErrRetrySequencer = errors.New("please retry transaction")

func (s *Sequencer) ctxWithQueueTimeout(inctx context.Context) (context.Context, context.CancelFunc) {
	timeout := s.config().QueueTimeout
	if timeout == time.Duration(0) {
		return context.WithCancel(inctx)
	}
	return context.WithTimeout(inctx, timeout)
}

func (s *Sequencer) PublishTransaction(parentCtx context.Context, tx *types.Transaction) error {
	sequencerBacklogGauge.Inc(1)
	defer sequencerBacklogGauge.Dec(1)

	forwarder := s.GetForwarder()
	if forwarder != nil {
		err := forwarder.PublishTransaction(parentCtx, tx)
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

	ctx, cancelFunc := s.ctxWithQueueTimeout(parentCtx)
	defer cancelFunc()

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

func (s *Sequencer) preTxFilter(_ *params.ChainConfig, header *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, sender common.Address) error {
	if s.nonceCache.GetSize() > 0 {
		stateNonce := s.nonceCache.Get(header, statedb, sender)
		err := MakeNonceError(sender, tx.Nonce(), stateNonce)
		if err != nil {
			nonceCacheRejectedCounter.Inc(1)
			return err
		}
	}
	return nil
}

func (s *Sequencer) postTxFilter(header *types.Header, _ *arbosState.ArbosState, tx *types.Transaction, sender common.Address, dataGas uint64, result *core.ExecutionResult) error {
	if result.Err != nil && result.UsedGas > dataGas && result.UsedGas-dataGas <= s.config().MaxRevertGasReject {
		return arbitrum.NewRevertReason(result)
	}
	s.nonceCache.Update(header, sender, tx.Nonce()+1)
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
		panicErr := recover()
		if panicErr != nil {
			log.Error("sequencer block creation panicked", "panic", panicErr, "backtrace", string(debug.Stack()))
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

	s.nonceCache.Resize(s.config().NonceCacheSize) // Would probably be better in a config hook but this is basically free
	s.nonceCache.BeginNewBlock()
	hooks := &arbos.SequencingHooks{
		PreTxFilter:            s.preTxFilter,
		PostTxFilter:           s.postTxFilter,
		DiscardInvalidTxsEarly: true,
		TxErrors:               []error{},
	}
	start := time.Now()
	block, err := s.txStreamer.SequenceTransactions(header, txes, hooks)
	blockCreationTimer.Update(time.Since(start))
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
		if errors.Is(err, context.Canceled) {
			// thread closed. We'll later try to forward these messages.
			for _, item := range queueItems {
				s.requeueOrFail(item, err)
			}
			return true // don't return failure to avoid retrying immediately
		}
		log.Warn("error sequencing transactions", "err", err)
		for _, queueItem := range queueItems {
			queueItem.returnResult(err)
		}
		return false
	}

	if block != nil {
		successfulBlocksCounter.Inc(1)
		s.nonceCache.Finalize(block)
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

func (s *Sequencer) StopAndWait() {
	s.StopWaiter.StopAndWait()
	if s.txRetryQueue.Len() == 0 && len(s.txQueue) == 0 {
		return
	}
	// this usually means that coordinator's safe-shutdown-delay is too low
	log.Warn("sequencer has queued items while shutting down", "txQueue", len(s.txQueue), "retryQueue", s.txRetryQueue.Len())
	forwarder := s.GetForwarder()
	if forwarder != nil {
	emptyqueues:
		for {
			var item txQueueItem
			source := ""
			if s.txRetryQueue.Len() > 0 {
				item = s.txRetryQueue.Pop()
				source = "retryQueue"
			} else {
				select {
				case item = <-s.txQueue:
					source = "txQueue"
				default:
					break emptyqueues
				}
			}
			err := forwarder.PublishTransaction(item.ctx, item.tx)
			if err != nil {
				log.Warn("failed to forward transaction while shutting down", "source", source, "err", err)
			}

		}
	}
}
