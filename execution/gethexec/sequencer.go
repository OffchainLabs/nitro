// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/headerreader"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	sequencerBacklogGauge                   = metrics.NewRegisteredGauge("arb/sequencer/backlog", nil)
	nonceCacheHitCounter                    = metrics.NewRegisteredCounter("arb/sequencer/noncecache/hit", nil)
	nonceCacheMissCounter                   = metrics.NewRegisteredCounter("arb/sequencer/noncecache/miss", nil)
	nonceCacheRejectedCounter               = metrics.NewRegisteredCounter("arb/sequencer/noncecache/rejected", nil)
	nonceCacheClearedCounter                = metrics.NewRegisteredCounter("arb/sequencer/noncecache/cleared", nil)
	nonceFailureCacheSizeGauge              = metrics.NewRegisteredGauge("arb/sequencer/noncefailurecache/size", nil)
	nonceFailureCacheOverflowCounter        = metrics.NewRegisteredGauge("arb/sequencer/noncefailurecache/overflow", nil)
	blockCreationTimer                      = metrics.NewRegisteredTimer("arb/sequencer/block/creation", nil)
	successfulBlocksCounter                 = metrics.NewRegisteredCounter("arb/sequencer/block/successful", nil)
	conditionalTxRejectedBySequencerCounter = metrics.NewRegisteredCounter("arb/sequencer/condtionaltx/rejected", nil)
	conditionalTxAcceptedBySequencerCounter = metrics.NewRegisteredCounter("arb/sequencer/condtionaltx/accepted", nil)
)

type SequencerConfig struct {
	Enable                      bool            `koanf:"enable"`
	MaxBlockSpeed               time.Duration   `koanf:"max-block-speed" reload:"hot"`
	MaxRevertGasReject          uint64          `koanf:"max-revert-gas-reject" reload:"hot"`
	MaxAcceptableTimestampDelta time.Duration   `koanf:"max-acceptable-timestamp-delta" reload:"hot"`
	SenderWhitelist             string          `koanf:"sender-whitelist"`
	Forwarder                   ForwarderConfig `koanf:"forwarder"`
	QueueSize                   int             `koanf:"queue-size"`
	QueueTimeout                time.Duration   `koanf:"queue-timeout" reload:"hot"`
	NonceCacheSize              int             `koanf:"nonce-cache-size" reload:"hot"`
	MaxTxDataSize               int             `koanf:"max-tx-data-size" reload:"hot"`
	NonceFailureCacheSize       int             `koanf:"nonce-failure-cache-size" reload:"hot"`
	NonceFailureCacheExpiry     time.Duration   `koanf:"nonce-failure-cache-expiry" reload:"hot"`
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
	// 95% of the default batch poster limit, leaving 5KB for headers and such
	MaxTxDataSize:           95000,
	NonceFailureCacheSize:   1024,
	NonceFailureCacheExpiry: time.Second,
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
	MaxTxDataSize:               95000,
	NonceFailureCacheSize:       1024,
	NonceFailureCacheExpiry:     time.Second,
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
	f.Int(prefix+".max-tx-data-size", DefaultSequencerConfig.MaxTxDataSize, "maximum transaction size the sequencer will accept")
	f.Int(prefix+".nonce-failure-cache-size", DefaultSequencerConfig.NonceFailureCacheSize, "number of transactions with too high of a nonce to keep in memory while waiting for their predecessor")
	f.Duration(prefix+".nonce-failure-cache-expiry", DefaultSequencerConfig.NonceFailureCacheExpiry, "maximum amount of time to wait for a predecessor before rejecting a tx with nonce too high")
}

type txQueueItem struct {
	tx              *types.Transaction
	options         *arbitrum_types.ConditionalOptions
	resultChan      chan<- error
	returnedResult  bool
	ctx             context.Context
	firstAppearance time.Time
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
	}
	return c.block == header.ParentHash
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
	// Note: we don't use c.matches here because the header will have changed
	if c.block == block.ParentHash() {
		c.block = block.Hash()
		c.dirty = nil
	} else {
		c.Reset(block.Hash())
	}
}

func (c *nonceCache) Caching() bool {
	return c.cache != nil && c.cache.Size() > 0
}

func (c *nonceCache) Resize(newSize int) {
	c.cache.Resize(newSize)
}

type addressAndNonce struct {
	address common.Address
	nonce   uint64
}

type nonceFailure struct {
	queueItem txQueueItem
	nonceErr  error
	expiry    time.Time
	revived   bool
}

type nonceFailureCache struct {
	*containers.LruCache[addressAndNonce, *nonceFailure]
	getExpiry func() time.Duration
}

func (c nonceFailureCache) Contains(err NonceError) bool {
	key := addressAndNonce{err.sender, err.txNonce}
	return c.LruCache.Contains(key)
}

func (c nonceFailureCache) Add(err NonceError, queueItem txQueueItem) {
	key := addressAndNonce{err.sender, err.txNonce}
	val := &nonceFailure{
		queueItem: queueItem,
		nonceErr:  err,
		expiry:    queueItem.firstAppearance.Add(c.getExpiry()),
		revived:   false,
	}
	evicted := c.LruCache.Add(key, val)
	if evicted {
		nonceFailureCacheOverflowCounter.Inc(1)
	}
}

type Sequencer struct {
	stopwaiter.StopWaiter

	execEngine      *ExecutionEngine
	txQueue         chan txQueueItem
	txRetryQueue    containers.Queue[txQueueItem]
	l1Reader        *headerreader.HeaderReader
	config          SequencerConfigFetcher
	senderWhitelist map[common.Address]struct{}
	nonceCache      *nonceCache
	nonceFailures   *nonceFailureCache
	onForwarderSet  chan struct{}

	L1BlockAndTimeMutex sync.Mutex
	l1BlockNumber       uint64
	l1Timestamp         uint64

	// activeMutex manages pauseChan (pauses execution) and forwarder
	// at most one of these is non-nil at any given time
	// both are nil for the active sequencer
	activeMutex sync.Mutex
	pauseChan   chan struct{}
	forwarder   *TxForwarder
}

func NewSequencer(execEngine *ExecutionEngine, l1Reader *headerreader.HeaderReader, configFetcher SequencerConfigFetcher) (*Sequencer, error) {
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
	s := &Sequencer{
		execEngine:      execEngine,
		txQueue:         make(chan txQueueItem, config.QueueSize),
		l1Reader:        l1Reader,
		config:          configFetcher,
		senderWhitelist: senderWhitelist,
		nonceCache:      newNonceCache(config.NonceCacheSize),
		l1BlockNumber:   0,
		l1Timestamp:     0,
		pauseChan:       nil,
		onForwarderSet:  make(chan struct{}, 1),
	}
	s.nonceFailures = &nonceFailureCache{
		containers.NewLruCacheWithOnEvict(config.NonceCacheSize, s.onNonceFailureEvict),
		func() time.Duration { return configFetcher().NonceFailureCacheExpiry },
	}
	execEngine.EnableReorgSequencing()
	return s, nil
}

func (s *Sequencer) onNonceFailureEvict(_ addressAndNonce, failure *nonceFailure) {
	if failure.revived {
		return
	}
	queueItem := failure.queueItem
	err := queueItem.ctx.Err()
	if err != nil {
		queueItem.returnResult(err)
		return
	}
	_, forwarder := s.GetPauseAndForwarder()
	if forwarder != nil {
		// We might not have gotten the predecessor tx because our forwarder did. Let's try there instead.
		// We run this in a background goroutine because LRU eviction needs to be quick.
		// We use an untracked thread for a few reasons:
		//   - It's guaranteed to run even when stopped (we need to return *some* result).
		//   - It acquires mutexes and this might need to happen a lot.
		//   - We don't need the context because queueItem has its own.
		//   - The RPC handler is on a separate StopWaiter anyways -- we should respect its context.
		s.LaunchUntrackedThread(func() {
			err = forwarder.PublishTransaction(queueItem.ctx, queueItem.tx, queueItem.options)
			queueItem.returnResult(err)
		})
	} else {
		queueItem.returnResult(failure.nonceErr)
	}
}

func (s *Sequencer) ctxWithQueueTimeout(inctx context.Context) (context.Context, context.CancelFunc) {
	timeout := s.config().QueueTimeout
	if timeout == time.Duration(0) {
		return context.WithCancel(inctx)
	}
	return context.WithTimeout(inctx, timeout)
}

func (s *Sequencer) PublishTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	sequencerBacklogGauge.Inc(1)
	defer sequencerBacklogGauge.Dec(1)

	_, forwarder := s.GetPauseAndForwarder()
	if forwarder != nil {
		err := forwarder.PublishTransaction(parentCtx, tx, options)
		if !errors.Is(err, ErrNoSequencer) {
			return err
		}
	}

	if len(s.senderWhitelist) > 0 {
		signer := types.LatestSigner(s.execEngine.bc.Config())
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
		options,
		resultChan,
		false,
		ctx,
		time.Now(),
	}
	select {
	case s.txQueue <- queueItem:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case res := <-resultChan:
		return res
	case <-parentCtx.Done():
		// We use parentCtx here and not ctx, because the QueueTimeout only applies to the background queue.
		// We want to give the background queue as much time as possible to make a response.
		return parentCtx.Err()
	}
}

func (s *Sequencer) preTxFilter(_ *params.ChainConfig, header *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, sender common.Address, l1Info *arbos.L1Info) error {
	if s.nonceCache.Caching() {
		stateNonce := s.nonceCache.Get(header, statedb, sender)
		err := MakeNonceError(sender, tx.Nonce(), stateNonce)
		if err != nil {
			nonceCacheRejectedCounter.Inc(1)
			return err
		}
	}
	if options != nil {
		err := options.Check(l1Info.L1BlockNumber(), header.Time, statedb)
		if err != nil {
			conditionalTxRejectedBySequencerCounter.Inc(1)
			return err
		}
		conditionalTxAcceptedBySequencerCounter.Inc(1)
	}
	return nil
}

func (s *Sequencer) postTxFilter(header *types.Header, _ *arbosState.ArbosState, tx *types.Transaction, sender common.Address, dataGas uint64, result *core.ExecutionResult) error {
	if result.Err != nil && result.UsedGas > dataGas && result.UsedGas-dataGas <= s.config().MaxRevertGasReject {
		return arbitrum.NewRevertReason(result)
	}
	newNonce := tx.Nonce() + 1
	s.nonceCache.Update(header, sender, newNonce)
	newAddrAndNonce := addressAndNonce{sender, newNonce}
	nonceFailure, haveNonceFailure := s.nonceFailures.Get(newAddrAndNonce)
	if haveNonceFailure {
		nonceFailure.revived = true // prevent the expiry hook from taking effect
		s.nonceFailures.Remove(newAddrAndNonce)
		// Immediately check if the transaction submission has been canceled
		err := nonceFailure.queueItem.ctx.Err()
		if err != nil {
			nonceFailure.queueItem.returnResult(err)
		} else {
			// Add this transaction (whose nonce is now correct) back into the queue
			s.txRetryQueue.Push(nonceFailure.queueItem)
		}
	}
	return nil
}

func (s *Sequencer) CheckHealth(ctx context.Context) error {
	pauseChan, forwarder := s.GetPauseAndForwarder()
	if forwarder != nil {
		return forwarder.CheckHealth(ctx)
	}
	if pauseChan != nil {
		return nil
	}
	_, err := s.execEngine.consensus.ExpectChosenSequencer().Await(ctx)
	return err
}

func (s *Sequencer) ForwardTarget() string {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	if s.forwarder == nil {
		return ""
	}
	return s.forwarder.target
}

func (s *Sequencer) ForwardTo(url string) containers.PromiseInterface[struct{}] {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	if s.forwarder != nil {
		if s.forwarder.target == url {
			log.Warn("attempted to update sequencer forward target with existing target", "url", url)
			return containers.NewReadyPromise[struct{}](struct{}{}, nil)
		}
		s.forwarder.Disable()
	}
	s.forwarder = NewForwarder(url, &s.config().Forwarder)
	err := s.forwarder.Initialize(s.GetContext())
	if err != nil {
		log.Error("failed to set forward agent", "err", err)
		s.forwarder = nil
	}
	if s.pauseChan != nil {
		close(s.pauseChan)
		s.pauseChan = nil
	}
	if err == nil {
		// If createBlocks is waiting for a new queue item, notify it that it needs to clear the nonceFailures.
		select {
		case s.onForwarderSet <- struct{}{}:
		default:
		}
	}
	return containers.NewReadyPromise[struct{}](struct{}{}, err)
}

func (s *Sequencer) Activate() containers.PromiseInterface[struct{}] {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	if s.forwarder != nil {
		s.forwarder.Disable()
		s.forwarder = nil
	}
	if s.pauseChan != nil {
		close(s.pauseChan)
		s.pauseChan = nil
	}
	return containers.NewReadyPromise[struct{}](struct{}{}, nil)
}

func (s *Sequencer) Pause() containers.PromiseInterface[struct{}] {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	if s.forwarder != nil {
		s.forwarder.Disable()
		s.forwarder = nil
	}
	if s.pauseChan == nil {
		s.pauseChan = make(chan struct{})
	}
	return containers.NewReadyPromise[struct{}](struct{}{}, nil)
}

var ErrNoSequencer = errors.New("sequencer temporarily not available")

func (s *Sequencer) GetPauseAndForwarder() (chan struct{}, *TxForwarder) {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	return s.pauseChan, s.forwarder
}

// only called from createBlock, may be paused
func (s *Sequencer) handleInactive(ctx context.Context, queueItems []txQueueItem) bool {
	var forwarder *TxForwarder
	for {
		var pause chan struct{}
		pause, forwarder = s.GetPauseAndForwarder()
		if pause == nil {
			if forwarder == nil {
				return false
			}
			// if forwarding: jump to next loop
			break
		}
		// if paused: wait till unpaused
		select {
		case <-ctx.Done():
			return true
		case <-pause:
		}
	}
	publishResults := make(chan *txQueueItem, len(queueItems))
	for _, item := range queueItems {
		item := item
		go func() {
			res := forwarder.PublishTransaction(item.ctx, item.tx, item.options)
			if errors.Is(res, ErrNoSequencer) {
				publishResults <- &item
			} else {
				publishResults <- nil
				item.returnResult(res)
			}
		}()
	}
	for range queueItems {
		remainingItem := <-publishResults
		if remainingItem != nil {
			s.txRetryQueue.Push(*remainingItem)
		}
	}
	// Evict any leftover nonce failures, forwarding them
	s.nonceFailures.Clear()
	return true
}

var sequencerInternalError = errors.New("sequencer internal error")

func (s *Sequencer) makeSequencingHooks() *arbos.SequencingHooks {
	return &arbos.SequencingHooks{
		PreTxFilter:             s.preTxFilter,
		PostTxFilter:            s.postTxFilter,
		DiscardInvalidTxsEarly:  true,
		TxErrors:                []error{},
		ConditionalOptionsForTx: nil,
	}
}

func (s *Sequencer) expireNonceFailures() *time.Timer {
	defer nonceFailureCacheSizeGauge.Update(int64(s.nonceFailures.Len()))
	for {
		_, failure, ok := s.nonceFailures.GetOldest()
		if !ok {
			return nil
		}
		untilExpiry := time.Until(failure.expiry)
		if untilExpiry > 0 {
			return time.NewTimer(untilExpiry)
		}
		s.nonceFailures.RemoveOldest()
	}
}

// There's no guarantee that returned tx nonces will be correct
func (s *Sequencer) precheckNonces(queueItems []txQueueItem) []txQueueItem {
	bc := s.execEngine.bc
	latestHeader := bc.CurrentBlock()
	latestState, err := bc.StateAt(latestHeader.Root)
	if err != nil {
		log.Error("failed to get current state to pre-check nonces", "err", err)
		return queueItems
	}
	nextHeaderNumber := arbmath.BigAdd(latestHeader.Number, common.Big1)
	signer := types.MakeSigner(bc.Config(), nextHeaderNumber)
	outputQueueItems := make([]txQueueItem, 0, len(queueItems))
	var nextQueueItem *txQueueItem
	var queueItemsIdx int
	pendingNonces := make(map[common.Address]uint64)
	for {
		var queueItem txQueueItem
		if nextQueueItem != nil {
			queueItem = *nextQueueItem
			nextQueueItem = nil
		} else if queueItemsIdx < len(queueItems) {
			queueItem = queueItems[queueItemsIdx]
			queueItemsIdx++
		} else {
			break
		}
		tx := queueItem.tx
		sender, err := types.Sender(signer, tx)
		if err != nil {
			queueItem.returnResult(err)
			continue
		}
		stateNonce := s.nonceCache.Get(latestHeader, latestState, sender)
		pendingNonce, pending := pendingNonces[sender]
		if !pending {
			pendingNonce = stateNonce
		}
		txNonce := tx.Nonce()
		if txNonce == pendingNonce {
			pendingNonces[sender] = txNonce + 1
			nextKey := addressAndNonce{sender, txNonce + 1}
			revivingFailure, exists := s.nonceFailures.Get(nextKey)
			if exists {
				// This tx was the predecessor to one that had failed its nonce check
				// Re-enqueue the tx whose nonce should now be correct, unless it expired
				revivingFailure.revived = true
				s.nonceFailures.Remove(nextKey)
				err := revivingFailure.queueItem.ctx.Err()
				if err != nil {
					revivingFailure.queueItem.returnResult(err)
				} else {
					nextQueueItem = &revivingFailure.queueItem
				}
			}
		} else if txNonce < stateNonce || txNonce > pendingNonce {
			// It's impossible for this tx to succeed so far,
			// because its nonce is lower than the state nonce
			// or higher than the highest tx nonce we've seen.
			err := MakeNonceError(sender, txNonce, stateNonce)
			if errors.Is(err, core.ErrNonceTooHigh) {
				var nonceError NonceError
				if !errors.As(err, &nonceError) {
					log.Warn("unreachable nonce error is not nonceError")
					continue
				}
				// Retry this transaction if its predecessor appears
				if s.nonceFailures.Contains(nonceError) {
					queueItem.returnResult(err)
				} else {
					s.nonceFailures.Add(nonceError, queueItem)
				}
				continue
			} else if err != nil {
				nonceCacheRejectedCounter.Inc(1)
				queueItem.returnResult(err)
				continue
			} else {
				log.Warn("unreachable nonce err == nil condition hit in precheckNonces")
			}
		}
		// If neither if condition was hit, then txNonce >= stateNonce && txNonce < pendingNonce
		// This tx might still go through if previous txs fail.
		// We'll include it in the output queue in case that happens.
		outputQueueItems = append(outputQueueItems, queueItem)
	}
	nonceFailureCacheSizeGauge.Update(int64(s.nonceFailures.Len()))
	return outputQueueItems
}

func (s *Sequencer) createBlock(ctx context.Context) (returnValue bool) {
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
	defer nonceFailureCacheSizeGauge.Update(int64(s.nonceFailures.Len()))

	config := s.config()

	// Clear out old nonceFailures
	s.nonceFailures.Resize(config.NonceFailureCacheSize)
	nextNonceExpiryTimer := s.expireNonceFailures()
	defer func() {
		// We wrap this in a closure as to not cache the current value of nextNonceExpiryTimer
		if nextNonceExpiryTimer != nil {
			nextNonceExpiryTimer.Stop()
		}
	}()

	for {
		var queueItem txQueueItem
		if s.txRetryQueue.Len() > 0 {
			queueItem = s.txRetryQueue.Pop()
		} else if len(queueItems) == 0 {
			var nextNonceExpiryChan <-chan time.Time
			if nextNonceExpiryTimer != nil {
				nextNonceExpiryChan = nextNonceExpiryTimer.C
			}
			select {
			case queueItem = <-s.txQueue:
			case <-nextNonceExpiryChan:
				// No need to stop the previous timer since it already elapsed
				nextNonceExpiryTimer = s.expireNonceFailures()
				continue
			case <-s.onForwarderSet:
				// Make sure this notification isn't outdated
				_, forwarder := s.GetPauseAndForwarder()
				if forwarder != nil {
					s.nonceFailures.Clear()
				}
				continue
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
		if len(txBytes) > config.MaxTxDataSize {
			// This tx is too large
			queueItem.returnResult(txpool.ErrOversizedData)
			continue
		}
		if totalBatchSize+len(txBytes) > config.MaxTxDataSize {
			// This tx would be too large to add to this batch
			s.txRetryQueue.Push(queueItem)
			// End the batch here to put this tx in the next one
			break
		}
		totalBatchSize += len(txBytes)
		queueItems = append(queueItems, queueItem)
	}

	s.nonceCache.Resize(config.NonceCacheSize) // Would probably be better in a config hook but this is basically free
	s.nonceCache.BeginNewBlock()
	queueItems = s.precheckNonces(queueItems)
	txes := make([]*types.Transaction, len(queueItems))
	hooks := s.makeSequencingHooks()
	hooks.ConditionalOptionsForTx = make([]*arbitrum_types.ConditionalOptions, len(queueItems))
	for i, queueItem := range queueItems {
		txes[i] = queueItem.tx
		hooks.ConditionalOptionsForTx[i] = queueItem.options
	}

	if s.handleInactive(ctx, queueItems) {
		return false
	}

	timestamp := time.Now().Unix()
	s.L1BlockAndTimeMutex.Lock()
	l1Block := s.l1BlockNumber
	l1Timestamp := s.l1Timestamp
	s.L1BlockAndTimeMutex.Unlock()

	if s.l1Reader != nil && (l1Block == 0 || math.Abs(float64(l1Timestamp)-float64(timestamp)) > config.MaxAcceptableTimestampDelta.Seconds()) {
		log.Error(
			"cannot sequence: unknown L1 block or L1 timestamp too far from local clock time",
			"l1Block", l1Block,
			"l1Timestamp", time.Unix(int64(l1Timestamp), 0),
			"localTimestamp", time.Unix(int64(timestamp), 0),
		)
		return false
	}

	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: l1Block,
		Timestamp:   uint64(timestamp),
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	start := time.Now()
	block, err := s.execEngine.SequenceTransactions(ctx, header, txes, hooks)
	elapsed := time.Since(start)
	blockCreationTimer.Update(elapsed)
	if elapsed >= time.Second*5 {
		var blockNum *big.Int
		if block != nil {
			blockNum = block.Number()
		}
		log.Warn("took over 5 seconds to sequence a block", "elapsed", elapsed, "numTxes", len(txes), "success", block != nil, "l2Block", blockNum)
	}
	if err == nil && len(hooks.TxErrors) != len(txes) {
		err = fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(hooks.TxErrors), len(txes))
	}
	if errors.Is(err, execution.ErrRetrySequencer) {
		log.Warn("error sequencing transactions", "err", err)
		// we changed roles
		// forward if we have where to
		if s.handleInactive(ctx, queueItems) {
			return false
		}
		// try to add back to queue otherwise
		for _, item := range queueItems {
			s.txRetryQueue.Push(item)
		}
		return false
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// thread closed. We'll later try to forward these messages.
			for _, item := range queueItems {
				s.txRetryQueue.Push(item)
			}
			return true // don't return failure to avoid retrying immediately
		}
		log.Error("error sequencing transactions", "err", err)
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
			if madeBlock {
				// There was already an earlier tx in the block; retry in a fresh block.
				s.txRetryQueue.Push(queueItem)
				continue
			}
		}
		if errors.Is(err, core.ErrIntrinsicGas) {
			// Strip additional information, as it's incorrect due to L1 data gas.
			err = core.ErrIntrinsicGas
		}
		var nonceError NonceError
		if errors.As(err, &nonceError) && nonceError.txNonce > nonceError.stateNonce {
			s.nonceFailures.Add(nonceError, queueItem)
			continue
		}
		queueItem.returnResult(err)
	}
	return madeBlock
}

func (s *Sequencer) updateLatestParentChainBlock(header *types.Header) {
	s.L1BlockAndTimeMutex.Lock()
	defer s.L1BlockAndTimeMutex.Unlock()

	l1BlockNumber := arbutil.ParentHeaderToL1BlockNumber(header)
	if header.Time > s.l1Timestamp || (header.Time == s.l1Timestamp && l1BlockNumber > s.l1BlockNumber) {
		s.l1Timestamp = header.Time
		s.l1BlockNumber = l1BlockNumber
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
	s.updateLatestParentChainBlock(header)
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
					s.updateLatestParentChainBlock(header)
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
		}
		// If we didn't make a block, try again immediately.
		return 0
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
	_, forwarder := s.GetPauseAndForwarder()
	if forwarder != nil {
		var wg sync.WaitGroup
	emptyqueues:
		for {
			var item txQueueItem
			source := ""
			if s.txRetryQueue.Len() > 0 {
				item = s.txRetryQueue.Pop()
				source = "retryQueue"
			} else if s.nonceFailures.Len() > 0 {
				_, failure, _ := s.nonceFailures.GetOldest()
				failure.revived = true
				item = failure.queueItem
				s.nonceFailures.RemoveOldest()
			} else {
				select {
				case item = <-s.txQueue:
					source = "txQueue"
				default:
					break emptyqueues
				}
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := forwarder.PublishTransaction(item.ctx, item.tx, item.options)
				if err != nil {
					log.Warn("failed to forward transaction while shutting down", "source", source, "err", err)
				}
			}()
		}
		wg.Wait()
	}
}
