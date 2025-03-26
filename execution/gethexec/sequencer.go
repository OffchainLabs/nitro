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
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/headerreader"
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
	conditionalTxRejectedBySequencerCounter = metrics.NewRegisteredCounter("arb/sequencer/conditionaltx/rejected", nil)
	conditionalTxAcceptedBySequencerCounter = metrics.NewRegisteredCounter("arb/sequencer/conditionaltx/accepted", nil)
	l1GasPriceGauge                         = metrics.NewRegisteredGauge("arb/sequencer/l1gasprice", nil)
	callDataUnitsBacklogGauge               = metrics.NewRegisteredGauge("arb/sequencer/calldataunitsbacklog", nil)
	unusedL1GasChargeGauge                  = metrics.NewRegisteredGauge("arb/sequencer/unusedl1gascharge", nil)
	currentSurplusGauge                     = metrics.NewRegisteredGauge("arb/sequencer/currentsurplus", nil)
	expectedSurplusGauge                    = metrics.NewRegisteredGauge("arb/sequencer/expectedsurplus", nil)
)

type SequencerConfig struct {
	Enable                       bool            `koanf:"enable"`
	MaxBlockSpeed                time.Duration   `koanf:"max-block-speed" reload:"hot"`
	MaxRevertGasReject           uint64          `koanf:"max-revert-gas-reject" reload:"hot"`
	MaxAcceptableTimestampDelta  time.Duration   `koanf:"max-acceptable-timestamp-delta" reload:"hot"`
	SenderWhitelist              []string        `koanf:"sender-whitelist"`
	Forwarder                    ForwarderConfig `koanf:"forwarder"`
	QueueSize                    int             `koanf:"queue-size"`
	QueueTimeout                 time.Duration   `koanf:"queue-timeout" reload:"hot"`
	NonceCacheSize               int             `koanf:"nonce-cache-size" reload:"hot"`
	MaxTxDataSize                int             `koanf:"max-tx-data-size" reload:"hot"`
	NonceFailureCacheSize        int             `koanf:"nonce-failure-cache-size" reload:"hot"`
	NonceFailureCacheExpiry      time.Duration   `koanf:"nonce-failure-cache-expiry" reload:"hot"`
	ExpectedSurplusSoftThreshold string          `koanf:"expected-surplus-soft-threshold" reload:"hot"`
	ExpectedSurplusHardThreshold string          `koanf:"expected-surplus-hard-threshold" reload:"hot"`
	EnableProfiling              bool            `koanf:"enable-profiling" reload:"hot"`
	Dangerous                    DangerousConfig `koanf:"dangerous"`
	expectedSurplusSoftThreshold int
	expectedSurplusHardThreshold int
}

type DangerousConfig struct {
	Timeboost TimeboostConfig `koanf:"timeboost"`
}

type TimeboostConfig struct {
	Enable                    bool          `koanf:"enable"`
	AuctionContractAddress    string        `koanf:"auction-contract-address"`
	AuctioneerAddress         string        `koanf:"auctioneer-address"`
	ExpressLaneAdvantage      time.Duration `koanf:"express-lane-advantage"`
	SequencerHTTPEndpoint     string        `koanf:"sequencer-http-endpoint"`
	EarlySubmissionGrace      time.Duration `koanf:"early-submission-grace"`
	MaxQueuedTxCount          int           `koanf:"max-queued-tx-count"`
	MaxFutureSequenceDistance uint64        `koanf:"max-future-sequence-distance"`
	RedisUrl                  string        `koanf:"redis-url"`
}

var DefaultTimeboostConfig = TimeboostConfig{
	Enable:                    false,
	AuctionContractAddress:    "",
	AuctioneerAddress:         "",
	ExpressLaneAdvantage:      time.Millisecond * 200,
	SequencerHTTPEndpoint:     "http://localhost:8547",
	EarlySubmissionGrace:      time.Second * 2,
	MaxQueuedTxCount:          10,
	MaxFutureSequenceDistance: 100,
	RedisUrl:                  "unset",
}

func (c *SequencerConfig) Validate() error {
	for _, address := range c.SenderWhitelist {
		if len(address) == 0 {
			continue
		}
		if !common.IsHexAddress(address) {
			return fmt.Errorf("sequencer sender whitelist entry \"%v\" is not a valid address", address)
		}
	}
	var err error
	if c.ExpectedSurplusSoftThreshold != "default" {
		if c.expectedSurplusSoftThreshold, err = strconv.Atoi(c.ExpectedSurplusSoftThreshold); err != nil {
			return fmt.Errorf("invalid expected-surplus-soft-threshold value provided in batchposter config %w", err)
		}
	}
	if c.ExpectedSurplusHardThreshold != "default" {
		if c.expectedSurplusHardThreshold, err = strconv.Atoi(c.ExpectedSurplusHardThreshold); err != nil {
			return fmt.Errorf("invalid expected-surplus-hard-threshold value provided in batchposter config %w", err)
		}
	}
	if c.expectedSurplusSoftThreshold < c.expectedSurplusHardThreshold {
		return errors.New("expected-surplus-soft-threshold cannot be lower than expected-surplus-hard-threshold")
	}
	if c.MaxTxDataSize > arbostypes.MaxL2MessageSize-50000 {
		return errors.New("max-tx-data-size too large for MaxL2MessageSize")
	}
	return c.Dangerous.Timeboost.Validate()
}

func (c *TimeboostConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.RedisUrl == DefaultTimeboostConfig.RedisUrl {
		return errors.New("timeboost is enabled but no redis-url was set")
	}
	if len(c.AuctionContractAddress) > 0 && !common.IsHexAddress(c.AuctionContractAddress) {
		return fmt.Errorf("invalid timeboost.auction-contract-address \"%v\"", c.AuctionContractAddress)
	}
	if len(c.AuctioneerAddress) > 0 && !common.IsHexAddress(c.AuctioneerAddress) {
		return fmt.Errorf("invalid timeboost.auctioneer-address \"%v\"", c.AuctioneerAddress)
	}
	if c.MaxFutureSequenceDistance == 0 {
		return errors.New("timeboost max-future-sequence-distance option cannot be zero, it should be set to a positive value")
	}
	return nil
}

type SequencerConfigFetcher func() *SequencerConfig

var DefaultSequencerConfig = SequencerConfig{
	Enable:                      false,
	MaxBlockSpeed:               time.Millisecond * 250,
	MaxRevertGasReject:          0,
	MaxAcceptableTimestampDelta: time.Hour,
	SenderWhitelist:             []string{},
	Forwarder:                   DefaultSequencerForwarderConfig,
	QueueSize:                   1024,
	QueueTimeout:                time.Second * 12,
	NonceCacheSize:              1024,
	// 95% of the default batch poster limit, leaving 5KB for headers and such
	// This default is overridden for L3 chains in applyChainParameters in cmd/nitro/nitro.go
	MaxTxDataSize:                95000,
	NonceFailureCacheSize:        1024,
	NonceFailureCacheExpiry:      time.Second,
	ExpectedSurplusSoftThreshold: "default",
	ExpectedSurplusHardThreshold: "default",
	EnableProfiling:              false,
	Dangerous:                    DefaultDangerousConfig,
}

var DefaultDangerousConfig = DangerousConfig{
	Timeboost: DefaultTimeboostConfig,
}

func SequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSequencerConfig.Enable, "act and post to l1 as sequencer")
	f.Duration(prefix+".max-block-speed", DefaultSequencerConfig.MaxBlockSpeed, "minimum delay between blocks (sets a maximum speed of block production)")
	f.Uint64(prefix+".max-revert-gas-reject", DefaultSequencerConfig.MaxRevertGasReject, "maximum gas executed in a revert for the sequencer to reject the transaction instead of posting it (anti-DOS)")
	f.Duration(prefix+".max-acceptable-timestamp-delta", DefaultSequencerConfig.MaxAcceptableTimestampDelta, "maximum acceptable time difference between the local time and the latest L1 block's timestamp")
	f.StringSlice(prefix+".sender-whitelist", DefaultSequencerConfig.SenderWhitelist, "comma separated whitelist of authorized senders (if empty, everyone is allowed)")
	AddOptionsForSequencerForwarderConfig(prefix+".forwarder", f)
	DangerousAddOptions(prefix+".dangerous", f)

	f.Int(prefix+".queue-size", DefaultSequencerConfig.QueueSize, "size of the pending tx queue")
	f.Duration(prefix+".queue-timeout", DefaultSequencerConfig.QueueTimeout, "maximum amount of time transaction can wait in queue")
	f.Int(prefix+".nonce-cache-size", DefaultSequencerConfig.NonceCacheSize, "size of the tx sender nonce cache")
	f.Int(prefix+".max-tx-data-size", DefaultSequencerConfig.MaxTxDataSize, "maximum transaction size the sequencer will accept")
	f.Int(prefix+".nonce-failure-cache-size", DefaultSequencerConfig.NonceFailureCacheSize, "number of transactions with too high of a nonce to keep in memory while waiting for their predecessor")
	f.Duration(prefix+".nonce-failure-cache-expiry", DefaultSequencerConfig.NonceFailureCacheExpiry, "maximum amount of time to wait for a predecessor before rejecting a tx with nonce too high")
	f.String(prefix+".expected-surplus-soft-threshold", DefaultSequencerConfig.ExpectedSurplusSoftThreshold, "if expected surplus is lower than this value, warnings are posted")
	f.String(prefix+".expected-surplus-hard-threshold", DefaultSequencerConfig.ExpectedSurplusHardThreshold, "if expected surplus is lower than this value, new incoming transactions will be denied")
	f.Bool(prefix+".enable-profiling", DefaultSequencerConfig.EnableProfiling, "enable CPU profiling and tracing")
}

func TimeboostAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultTimeboostConfig.Enable, "enable timeboost based on express lane auctions")
	f.String(prefix+".auction-contract-address", DefaultTimeboostConfig.AuctionContractAddress, "Address of the proxy pointing to the ExpressLaneAuction contract")
	f.String(prefix+".auctioneer-address", DefaultTimeboostConfig.AuctioneerAddress, "Address of the Timeboost Autonomous Auctioneer")
	f.Duration(prefix+".express-lane-advantage", DefaultTimeboostConfig.ExpressLaneAdvantage, "specify the express lane advantage")
	f.String(prefix+".sequencer-http-endpoint", DefaultTimeboostConfig.SequencerHTTPEndpoint, "this sequencer's http endpoint")
	f.Duration(prefix+".early-submission-grace", DefaultTimeboostConfig.EarlySubmissionGrace, "period of time before the next round where submissions for the next round will be queued")
	f.Int(prefix+".max-queued-tx-count", DefaultTimeboostConfig.MaxQueuedTxCount, "maximum allowed number of express lane txs with future sequence number to be queued. Set 0 to disable this check and a negative value to prevent queuing of any future sequence number transactions")
	f.Uint64(prefix+".max-future-sequence-distance", DefaultTimeboostConfig.MaxFutureSequenceDistance, "maximum allowed difference (in terms of sequence numbers) between a future express lane tx and the current sequence count of a round")
	f.String(prefix+".redis-url", DefaultTimeboostConfig.RedisUrl, "the Redis URL for expressLaneService to coordinate via")
}

func DangerousAddOptions(prefix string, f *flag.FlagSet) {
	TimeboostAddOptions(prefix+".timeboost", f)
}

type txQueueItem struct {
	tx              *types.Transaction
	txSize          int // size in bytes of the marshalled transaction
	options         *arbitrum_types.ConditionalOptions
	resultChan      chan<- error
	returnedResult  *atomic.Bool
	ctx             context.Context
	firstAppearance time.Time
	isTimeboosted   bool
}

func (i *txQueueItem) returnResult(err error) {
	if i.returnedResult.Swap(true) {
		log.Error("attempting to return result to already finished queue item", "err", err)
		return
	}
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
		// Note, even though the of the header changes, c.dirty points to the
		// same header, hence hashes will be the same and this check will pass.
		return headerreader.HeadersEqual(c.dirty, header)
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
	expiry := queueItem.firstAppearance.Add(c.getExpiry())
	if c.Contains(err) || time.Now().After(expiry) {
		queueItem.returnResult(err)
		return
	}
	key := addressAndNonce{err.sender, err.txNonce}
	val := &nonceFailure{
		queueItem: queueItem,
		nonceErr:  err,
		expiry:    expiry,
		revived:   false,
	}
	evicted := c.LruCache.Add(key, val)
	if evicted {
		nonceFailureCacheOverflowCounter.Inc(1)
	}
}

type synchronizedTxQueue struct {
	queue containers.Queue[txQueueItem]
	mutex sync.RWMutex
}

func (q *synchronizedTxQueue) Push(item txQueueItem) {
	q.mutex.Lock()
	q.queue.Push(item)
	q.mutex.Unlock()
}

func (q *synchronizedTxQueue) Pop() txQueueItem {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.queue.Pop()

}

func (q *synchronizedTxQueue) Len() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.queue.Len()
}

type Sequencer struct {
	stopwaiter.StopWaiter

	execEngine         *ExecutionEngine
	txQueue            chan txQueueItem
	txRetryQueue       synchronizedTxQueue
	l1Reader           *headerreader.HeaderReader
	config             SequencerConfigFetcher
	senderWhitelist    map[common.Address]struct{}
	nonceCache         *nonceCache
	nonceFailures      *nonceFailureCache
	expressLaneService *expressLaneService
	onForwarderSet     chan struct{}

	L1BlockAndTimeMutex sync.Mutex
	l1BlockNumber       atomic.Uint64
	l1Timestamp         uint64

	// activeMutex manages pauseChan (pauses execution) and forwarder
	// at most one of these is non-nil at any given time
	// both are nil for the active sequencer
	activeMutex sync.Mutex
	pauseChan   chan struct{}
	forwarder   *TxForwarder

	expectedSurplusMutex              sync.RWMutex
	expectedSurplus                   int64
	expectedSurplusUpdated            bool
	auctioneerAddr                    common.Address
	timeboostAuctionResolutionTxQueue chan txQueueItem
}

func NewSequencer(execEngine *ExecutionEngine, l1Reader *headerreader.HeaderReader, configFetcher SequencerConfigFetcher) (*Sequencer, error) {
	config := configFetcher()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	senderWhitelist := make(map[common.Address]struct{})
	for _, address := range config.SenderWhitelist {
		if len(address) == 0 {
			continue
		}
		senderWhitelist[common.HexToAddress(address)] = struct{}{}
	}
	s := &Sequencer{
		execEngine:                        execEngine,
		txQueue:                           make(chan txQueueItem, config.QueueSize),
		l1Reader:                          l1Reader,
		config:                            configFetcher,
		senderWhitelist:                   senderWhitelist,
		nonceCache:                        newNonceCache(config.NonceCacheSize),
		l1Timestamp:                       0,
		pauseChan:                         nil,
		onForwarderSet:                    make(chan struct{}, 1),
		timeboostAuctionResolutionTxQueue: make(chan txQueueItem, 10), // There should never be more than 1 outstanding auction resolutions
	}
	s.nonceFailures = &nonceFailureCache{
		containers.NewLruCacheWithOnEvict(config.NonceCacheSize, s.onNonceFailureEvict),
		func() time.Duration { return configFetcher().NonceFailureCacheExpiry },
	}
	s.Pause()
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

// ctxWithTimeout is like context.WithTimeout except a timeout of 0 means unlimited instead of instantly expired.
func ctxWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == time.Duration(0) {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *Sequencer) PublishTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	_, forwarder := s.GetPauseAndForwarder()
	if forwarder != nil {
		err := forwarder.PublishTransaction(parentCtx, tx, options)
		if !errors.Is(err, ErrNoSequencer) {
			return err
		}
	}

	config := s.config()
	queueTimeout := config.QueueTimeout
	queueCtx, cancelFunc := ctxWithTimeout(parentCtx, queueTimeout+config.Dangerous.Timeboost.ExpressLaneAdvantage) // Include timeboost delay in ctx timeout
	defer cancelFunc()

	resultChan := make(chan error, 1)
	err := s.publishTransactionToQueue(queueCtx, tx, options, resultChan, false /* delay tx if express lane is active */)
	if err != nil {
		return err
	}

	now := time.Now()
	// Just to be safe, make sure we don't run over twice the queue timeout
	abortCtx, cancel := ctxWithTimeout(parentCtx, queueTimeout*2)
	defer cancel()

	select {
	case res := <-resultChan:
		return res
	case <-abortCtx.Done():
		// We use abortCtx here and not queueCtx, because the QueueTimeout only applies to the background queue.
		// We want to give the background queue as much time as possible to make a response.
		err := abortCtx.Err()
		if parentCtx.Err() == nil {
			// If we've hit the abort deadline (as opposed to parentCtx being canceled), something went wrong.
			log.Warn("Transaction sequencing hit abort deadline", "err", err, "submittedAt", now, "queueTimeout", queueTimeout*2, "txHash", tx.Hash())
		}
		return err
	}
}

func (s *Sequencer) PublishAuctionResolutionTransaction(ctx context.Context, tx *types.Transaction) error {
	if !s.config().Dangerous.Timeboost.Enable {
		return errors.New("timeboost not enabled")
	}

	forwarder, err := s.getForwarder(ctx)
	if err != nil {
		return err
	}
	if forwarder != nil {
		err := forwarder.PublishAuctionResolutionTransaction(ctx, tx)
		if !errors.Is(err, ErrNoSequencer) {
			return err
		}
	}

	arrivalTime := time.Now()
	auctioneerAddr := s.auctioneerAddr
	if auctioneerAddr == (common.Address{}) {
		return errors.New("invalid auctioneer address")
	}
	if tx.To() == nil {
		return errors.New("transaction has no recipient")
	}
	if *tx.To() != s.expressLaneService.auctionContractAddr {
		return errors.New("transaction recipient is not the auction contract")
	}
	signer := types.LatestSigner(s.execEngine.bc.Config())
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return err
	}
	if sender != auctioneerAddr {
		return fmt.Errorf("sender %#x is not the auctioneer address %#x", sender, auctioneerAddr)
	}
	if !s.expressLaneService.roundTimingInfo.IsWithinAuctionCloseWindow(arrivalTime) {
		return fmt.Errorf("transaction arrival time not within auction closure window: %v", arrivalTime)
	}
	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	log.Info("Prioritizing auction resolution transaction from auctioneer", "txHash", tx.Hash().Hex())
	s.timeboostAuctionResolutionTxQueue <- txQueueItem{
		tx:              tx,
		txSize:          len(txBytes),
		options:         nil,
		resultChan:      make(chan error, 1),
		returnedResult:  &atomic.Bool{},
		ctx:             s.GetContext(),
		firstAppearance: time.Now(),
		isTimeboosted:   true,
	}
	return nil
}

func (s *Sequencer) PublishExpressLaneTransaction(ctx context.Context, msg *timeboost.ExpressLaneSubmission) error {
	if !s.config().Dangerous.Timeboost.Enable {
		return errors.New("timeboost not enabled")
	}

	forwarder, err := s.getForwarder(ctx)
	if err != nil {
		return err
	}
	if forwarder != nil {
		return forwarder.PublishExpressLaneTransaction(ctx, msg)
	}

	if s.expressLaneService == nil {
		return errors.New("express lane service not enabled")
	}
	if err := s.expressLaneService.validateExpressLaneTx(msg); err != nil {
		return err
	}

	forwarder, err = s.getForwarder(ctx)
	if err != nil {
		return err
	}
	if forwarder != nil {
		return forwarder.PublishExpressLaneTransaction(ctx, msg)
	}

	return s.expressLaneService.sequenceExpressLaneSubmission(ctx, msg)
}

func (s *Sequencer) PublishTimeboostedTransaction(queueCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, resultChan chan error) {
	if err := s.publishTransactionToQueue(queueCtx, tx, options, resultChan, true); err != nil {
		resultChan <- err
	}
}

func (s *Sequencer) publishTransactionToQueue(queueCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, resultChan chan error, isExpressLaneController bool) error {
	config := s.config()
	// Only try to acquire Rlock and check for hard threshold if l1reader is not nil
	// And hard threshold was enabled, this prevents spamming of read locks when not needed
	if s.l1Reader != nil && config.ExpectedSurplusHardThreshold != "default" {
		s.expectedSurplusMutex.RLock()
		if s.expectedSurplusUpdated && s.expectedSurplus < int64(config.expectedSurplusHardThreshold) {
			return errors.New("currently not accepting transactions due to expected surplus being below threshold")
		}
		s.expectedSurplusMutex.RUnlock()
	}

	sequencerBacklogGauge.Inc(1)
	defer sequencerBacklogGauge.Dec(1)

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
	if tx.Type() >= types.ArbitrumDepositTxType || tx.Type() == types.BlobTxType {
		// Should be unreachable for Arbitrum types due to UnmarshalBinary not accepting Arbitrum internal txs
		// and we want to disallow BlobTxType since Arbitrum doesn't support EIP-4844 txs yet.
		return types.ErrTxTypeNotSupported
	}

	txBytes, err := tx.MarshalBinary()
	if err != nil {
		return err
	}

	if s.config().Dangerous.Timeboost.Enable && s.expressLaneService != nil {
		if !isExpressLaneController && s.expressLaneService.currentRoundHasController() {
			time.Sleep(s.config().Dangerous.Timeboost.ExpressLaneAdvantage)
		}
	}

	queueItem := txQueueItem{
		tx,
		len(txBytes),
		options,
		resultChan,
		&atomic.Bool{},
		queueCtx,
		time.Now(),
		isExpressLaneController,
	}
	select {
	case s.txQueue <- queueItem:
	case <-queueCtx.Done():
		return queueCtx.Err()
	}

	return nil
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

func (s *Sequencer) postTxFilter(header *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, sender common.Address, dataGas uint64, result *core.ExecutionResult) error {
	if statedb.IsTxFiltered() {
		return state.ErrArbTxFilter
	}
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
	return s.execEngine.consensus.ExpectChosenSequencer()
}

func (s *Sequencer) ForwardTarget() string {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	if s.forwarder == nil {
		return ""
	}
	return s.forwarder.PrimaryTarget()
}

func (s *Sequencer) ForwardTo(url string) error {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	if s.forwarder != nil {
		if s.forwarder.PrimaryTarget() == url {
			log.Warn("attempted to update sequencer forward target with existing target", "url", url)
			return nil
		}
		s.forwarder.Disable()
	}
	s.forwarder = NewForwarder([]string{url}, &s.config().Forwarder)
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
	return err
}

func (s *Sequencer) Activate() {
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
	if s.expressLaneService != nil {
		s.LaunchThread(func(context.Context) {
			// We launch redis sync (which is best effort) in parallel to avoid blocking sequencer activation
			s.expressLaneService.syncFromRedis()
			time.Sleep(time.Second)
			s.expressLaneService.syncFromRedis()
		})
	}
}

func (s *Sequencer) Pause() {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	if s.forwarder != nil {
		s.forwarder.Disable()
		s.forwarder = nil
	}
	if s.pauseChan == nil {
		s.pauseChan = make(chan struct{})
	}
}

var ErrNoSequencer = errors.New("sequencer temporarily not available")

func (s *Sequencer) GetPauseAndForwarder() (chan struct{}, *TxForwarder) {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	return s.pauseChan, s.forwarder
}

// getForwarder returns accurate forwarder and pauses if needed.
// Required for processing timeboost txs, as just checking forwarder==nil doesn't imply the sequencer to be chosen
func (s *Sequencer) getForwarder(ctx context.Context) (*TxForwarder, error) {
	for {
		pause, forwarder := s.GetPauseAndForwarder()
		if pause == nil {
			return forwarder, nil
		}
		// if paused: wait till unpaused
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pause:
		}
	}
}

// only called from createBlock, may be paused
func (s *Sequencer) handleInactive(ctx context.Context, queueItems []txQueueItem) bool {
	forwarder, err := s.getForwarder(ctx)
	if err != nil {
		return true
	}
	if forwarder == nil {
		return false
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
func (s *Sequencer) precheckNonces(queueItems []txQueueItem, totalBlockSize int) []txQueueItem {
	config := s.config()
	bc := s.execEngine.bc
	latestHeader := bc.CurrentBlock()
	latestState, err := bc.StateAt(latestHeader.Root)
	if err != nil {
		log.Error("failed to get current state to pre-check nonces", "err", err)
		return queueItems
	}
	nextHeaderNumber := arbmath.BigAdd(latestHeader.Number, common.Big1)
	signer := types.MakeSigner(bc.Config(), nextHeaderNumber, latestHeader.Time)
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
					if arbmath.SaturatingAdd(totalBlockSize, revivingFailure.queueItem.txSize) > config.MaxTxDataSize {
						// This tx would be too large to add to this block
						s.txRetryQueue.Push(revivingFailure.queueItem)
					} else {
						nextQueueItem = &revivingFailure.queueItem
						totalBlockSize += revivingFailure.queueItem.txSize
					}
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
				s.nonceFailures.Add(nonceError, queueItem)
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
	var totalBlockSize int

	defer func() {
		panicErr := recover()
		if panicErr != nil {
			log.Error("sequencer block creation panicked", "panic", panicErr, "backtrace", string(debug.Stack()))
			// Return an internal error to any queue items we were trying to process
			for _, item := range queueItems {
				// This can race, but that's alright, worst case is a log line in returnResult
				if !item.returnedResult.Load() {
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
			select {
			case queueItem = <-s.timeboostAuctionResolutionTxQueue:
				log.Debug("Popped the auction resolution tx", "txHash", queueItem.tx.Hash())
			default:
				// The txRetryQueue is not modeled as a channel because it is only added to from
				// this function (Sequencer.createBlock). So it is sufficient to check its
				// len at the start of this loop, since items can't be added to it asynchronously,
				// which is not true for the main txQueue or timeboostAuctionResolutionQueue.
				queueItem = s.txRetryQueue.Pop()
			}
		} else if len(queueItems) == 0 {
			var nextNonceExpiryChan <-chan time.Time
			if nextNonceExpiryTimer != nil {
				nextNonceExpiryChan = nextNonceExpiryTimer.C
			}
			select {
			case queueItem = <-s.timeboostAuctionResolutionTxQueue:
				log.Debug("Popped the auction resolution tx", "txHash", queueItem.tx.Hash())
			default:
				select {
				case queueItem = <-s.txQueue:
				case queueItem = <-s.timeboostAuctionResolutionTxQueue:
					log.Debug("Popped the auction resolution tx", "txHash", queueItem.tx.Hash())
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
			}
		} else {
			done := false
			select {
			case queueItem = <-s.timeboostAuctionResolutionTxQueue:
				log.Debug("Popped the auction resolution tx", "txHash", queueItem.tx.Hash())
			default:
				select {
				case queueItem = <-s.txQueue:
				case queueItem = <-s.timeboostAuctionResolutionTxQueue:
					log.Debug("Popped the auction resolution tx", "txHash", queueItem.tx.Hash())
				default:
					done = true
				}
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
		if queueItem.txSize > config.MaxTxDataSize {
			// This tx is too large
			queueItem.returnResult(txpool.ErrOversizedData)
			continue
		}
		if totalBlockSize+queueItem.txSize > config.MaxTxDataSize {
			// This tx would be too large to add to this batch
			s.txRetryQueue.Push(queueItem)
			// End the batch here to put this tx in the next one
			break
		}
		totalBlockSize += queueItem.txSize
		queueItems = append(queueItems, queueItem)
	}

	s.nonceCache.Resize(config.NonceCacheSize) // Would probably be better in a config hook but this is basically free
	s.nonceCache.BeginNewBlock()
	queueItems = s.precheckNonces(queueItems, totalBlockSize)
	txes := make([]*types.Transaction, len(queueItems))
	timeboostedTxs := make(map[common.Hash]struct{})
	hooks := s.makeSequencingHooks()
	hooks.ConditionalOptionsForTx = make([]*arbitrum_types.ConditionalOptions, len(queueItems))
	totalBlockSize = 0 // recompute the totalBlockSize to double check it
	for i, queueItem := range queueItems {
		txes[i] = queueItem.tx
		totalBlockSize = arbmath.SaturatingAdd(totalBlockSize, queueItem.txSize)
		hooks.ConditionalOptionsForTx[i] = queueItem.options
		if queueItem.isTimeboosted {
			timeboostedTxs[queueItem.tx.Hash()] = struct{}{}
		}
	}

	if totalBlockSize > config.MaxTxDataSize {
		for _, queueItem := range queueItems {
			s.txRetryQueue.Push(queueItem)
		}
		log.Error(
			"put too many transactions in a block",
			"numTxes", len(queueItems),
			"totalBlockSize", totalBlockSize,
			"maxTxDataSize", config.MaxTxDataSize,
		)
		return false
	}

	if s.handleInactive(ctx, queueItems) {
		return false
	}

	timestamp := time.Now().Unix()
	s.L1BlockAndTimeMutex.Lock()
	l1Block := s.l1BlockNumber.Load()
	l1Timestamp := s.l1Timestamp
	s.L1BlockAndTimeMutex.Unlock()

	if s.l1Reader != nil && (l1Block == 0 || math.Abs(float64(l1Timestamp)-float64(timestamp)) > config.MaxAcceptableTimestampDelta.Seconds()) {
		for _, queueItem := range queueItems {
			s.txRetryQueue.Push(queueItem)
		}
		// #nosec G115
		log.Error(
			"cannot sequence: unknown L1 block or L1 timestamp too far from local clock time",
			"l1Block", l1Block,
			"l1Timestamp", time.Unix(int64(l1Timestamp), 0),
			"localTimestamp", time.Unix(timestamp, 0),
		)
		return true
	}

	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: l1Block,
		Timestamp:   arbmath.SaturatingUCast[uint64](timestamp),
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	start := time.Now()
	var (
		block *types.Block
		err   error
	)
	if config.EnableProfiling {
		block, err = s.execEngine.SequenceTransactionsWithProfiling(header, txes, hooks, timeboostedTxs)
	} else {
		block, err = s.execEngine.SequenceTransactions(header, txes, hooks, timeboostedTxs)
	}
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
	if header.Time > s.l1Timestamp || (header.Time == s.l1Timestamp && l1BlockNumber > s.l1BlockNumber.Load()) {
		s.l1Timestamp = header.Time
		s.l1BlockNumber.Store(l1BlockNumber)
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

func (s *Sequencer) InitializeExpressLaneService(
	apiBackend *arbitrum.APIBackend,
	filterSystem *filters.FilterSystem,
	auctionContractAddr common.Address,
	auctioneerAddr common.Address,
	earlySubmissionGrace time.Duration,
) error {
	els, err := newExpressLaneService(
		s,
		s.config,
		apiBackend,
		filterSystem,
		auctionContractAddr,
		s.execEngine.bc,
		earlySubmissionGrace,
	)
	if err != nil {
		return fmt.Errorf("failed to create express lane service. auctionContractAddr: %v err: %w", auctionContractAddr, err)
	}
	s.auctioneerAddr = auctioneerAddr
	s.expressLaneService = els
	return nil
}

var (
	usableBytesInBlob    = big.NewInt(int64(len(kzg4844.Blob{}) * 31 / 32))
	blobTxBlobGasPerBlob = big.NewInt(params.BlobTxBlobGasPerBlob)
)

func (s *Sequencer) updateExpectedSurplus(ctx context.Context) (int64, error) {
	header, err := s.l1Reader.LastHeader(ctx)
	if err != nil {
		return 0, fmt.Errorf("error encountered getting latest header from l1reader while updating expectedSurplus: %w", err)
	}
	l1GasPrice := header.BaseFee.Uint64()
	if header.BlobGasUsed != nil {
		if header.ExcessBlobGas != nil {
			blobFeePerByte := eip4844.CalcBlobFee(eip4844.CalcExcessBlobGas(*header.ExcessBlobGas, *header.BlobGasUsed))
			blobFeePerByte.Mul(blobFeePerByte, blobTxBlobGasPerBlob)
			blobFeePerByte.Div(blobFeePerByte, usableBytesInBlob)
			if l1GasPrice > blobFeePerByte.Uint64()/16 {
				l1GasPrice = blobFeePerByte.Uint64() / 16
			}
		}
	}
	surplus, err := s.execEngine.getL1PricingSurplus()
	if err != nil {
		return 0, fmt.Errorf("error encountered getting l1 pricing surplus while updating expectedSurplus: %w", err)
	}
	// #nosec G115
	backlogL1GasCharged := int64(s.execEngine.backlogL1GasCharged())
	// #nosec G115
	backlogCallDataUnits := int64(s.execEngine.backlogCallDataUnits())
	// #nosec G115
	expectedSurplus := int64(surplus) + backlogL1GasCharged - backlogCallDataUnits*int64(l1GasPrice)
	// update metrics
	// #nosec G115
	l1GasPriceGauge.Update(int64(l1GasPrice))
	callDataUnitsBacklogGauge.Update(backlogCallDataUnits)
	unusedL1GasChargeGauge.Update(backlogL1GasCharged)
	currentSurplusGauge.Update(surplus)
	expectedSurplusGauge.Update(expectedSurplus)
	config := s.config()
	if config.ExpectedSurplusSoftThreshold != "default" && expectedSurplus < int64(config.expectedSurplusSoftThreshold) {
		log.Warn("expected surplus is below soft threshold", "value", expectedSurplus, "threshold", config.expectedSurplusSoftThreshold)
	}
	return expectedSurplus, nil
}

func (s *Sequencer) StartExpressLaneService(ctx context.Context) {
	if s.expressLaneService != nil {
		s.expressLaneService.Start(ctx)
	}
}

func (s *Sequencer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)
	config := s.config()
	if (config.ExpectedSurplusHardThreshold != "default" || config.ExpectedSurplusSoftThreshold != "default") && s.l1Reader == nil {
		return errors.New("expected surplus soft/hard thresholds are enabled but l1Reader is nil")
	}

	if s.l1Reader != nil {
		initialBlockNr := s.l1BlockNumber.Load()
		if initialBlockNr == 0 {
			return errors.New("sequencer not initialized")
		}

		expectedSurplus, err := s.updateExpectedSurplus(ctxIn)
		if err != nil {
			if config.ExpectedSurplusHardThreshold != "default" {
				return fmt.Errorf("expected-surplus-hard-threshold is enabled but error fetching initial expected surplus value: %w", err)
			}
			log.Error("expected-surplus-soft-threshold is enabled but error fetching initial expected surplus value", "err", err)
		} else {
			s.expectedSurplus = expectedSurplus
			s.expectedSurplusUpdated = true
		}
		s.CallIteratively(func(ctx context.Context) time.Duration {
			expectedSurplus, err := s.updateExpectedSurplus(ctxIn)
			s.expectedSurplusMutex.Lock()
			defer s.expectedSurplusMutex.Unlock()
			if err != nil {
				s.expectedSurplusUpdated = false
				log.Error("expected surplus soft/hard thresholds are enabled but unable to fetch latest expected surplus, retrying", "err", err)
				return 0
			}
			s.expectedSurplusUpdated = true
			s.expectedSurplus = expectedSurplus
			return 5 * time.Second
		})

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
		if s.createBlock(ctx) {
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
	if s.config().Dangerous.Timeboost.Enable && s.expressLaneService != nil {
		s.expressLaneService.StopAndWait()
	}
	if s.txRetryQueue.Len() == 0 &&
		len(s.txQueue) == 0 &&
		s.nonceFailures.Len() == 0 &&
		len(s.timeboostAuctionResolutionTxQueue) == 0 {
		return
	}
	// this usually means that coordinator's safe-shutdown-delay is too low
	log.Warn("Sequencer has queued items while shutting down",
		"txQueue", len(s.txQueue),
		"retryQueue", s.txRetryQueue.Len(),
		"nonceFailures", s.nonceFailures.Len(),
		"timeboostAuctionResolutionTxQueue", len(s.timeboostAuctionResolutionTxQueue))
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
				source = "nonceFailures"
				s.nonceFailures.RemoveOldest()
			} else {
				select {
				case item = <-s.txQueue:
					source = "txQueue"
				case item = <-s.timeboostAuctionResolutionTxQueue:
					source = "timeboostAuctionResolutionTxQueue"
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
