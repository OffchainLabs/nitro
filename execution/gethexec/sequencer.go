// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/execution/gethexec/eventfilter"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/ctxhelper"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	sequencerBacklogGauge                   = metrics.NewRegisteredGauge("arb/sequencer/backlog", nil)
	sequencerQueueGauge                     = metrics.NewRegisteredGauge("arb/sequencer/queue/length", nil)
	sequencerQueueHistogram                 = metrics.NewRegisteredHistogram("arb/sequencer/queue/histogram", nil, metrics.NewBoundedHistogramSample())
	nonceCacheHitCounter                    = metrics.NewRegisteredCounter("arb/sequencer/noncecache/hit", nil)
	nonceCacheMissCounter                   = metrics.NewRegisteredCounter("arb/sequencer/noncecache/miss", nil)
	nonceCacheRejectedCounter               = metrics.NewRegisteredCounter("arb/sequencer/noncecache/rejected", nil)
	nonceCacheClearedCounter                = metrics.NewRegisteredCounter("arb/sequencer/noncecache/cleared", nil)
	nonceFailureCacheSizeGauge              = metrics.NewRegisteredGauge("arb/sequencer/noncefailurecache/size", nil)
	nonceFailureCacheOverflowCounter        = metrics.NewRegisteredCounter("arb/sequencer/noncefailurecache/overflow", nil)
	blockCreationTimer                      = metrics.NewRegisteredHistogram("arb/sequencer/block/creation", nil, metrics.NewBoundedHistogramSample())
	successfulBlocksCounter                 = metrics.NewRegisteredCounter("arb/sequencer/block/successful", nil)
	blockTxSizeHistogram                    = metrics.NewRegisteredHistogram("arb/sequencer/block/txsize", nil, metrics.NewBoundedHistogramSample())
	txSizeHistogram                         = metrics.NewRegisteredHistogram("arb/sequencer/transactions/txsize", nil, metrics.NewBoundedHistogramSample())
	conditionalTxRejectedBySequencerCounter = metrics.NewRegisteredCounter("arb/sequencer/conditionaltx/rejected", nil)
	conditionalTxAcceptedBySequencerCounter = metrics.NewRegisteredCounter("arb/sequencer/conditionaltx/accepted", nil)
	l1GasPriceGauge                         = metrics.NewRegisteredGauge("arb/sequencer/l1gasprice", nil)
	callDataUnitsBacklogGauge               = metrics.NewRegisteredGauge("arb/sequencer/calldataunitsbacklog", nil)
	currentSurplusGauge                     = metrics.NewRegisteredGauge("arb/sequencer/currentsurplus", nil)
	expectedSurplusGauge                    = metrics.NewRegisteredGauge("arb/sequencer/expectedsurplus", nil)
	// number of blocks ended because of block gas limit at least one tx wasn't included in block because of gas limit)
	gasLimitedBlocksCounter = metrics.NewRegisteredCounter("arb/sequencer/block/gaslimited", nil)
	// number of blocks ended because of txes data size limit
	dataLimitedBlocksCounter = metrics.NewRegisteredCounter("arb/sequencer/block/datalimited", nil)
	// number of blocks ended because of exhausting the transactions to sequence
	txExhaustedBlocksCounter = metrics.NewRegisteredCounter("arb/sequencer/block/txexhausted", nil)
	// forwarder/pause wait + validation before sequencing an express lane submission
	expressLanePreSequenceWaitHistogram = metrics.NewRegisteredHistogram("arb/sequencer/timeboost/expresslane/presequencewait", nil, metrics.NewBoundedHistogramSample())
)

type SequencerConfig struct {
	Enable                       bool             `koanf:"enable"`
	MaxBlockSpeed                time.Duration    `koanf:"max-block-speed" reload:"hot"`
	MaxRevertGasReject           uint64           `koanf:"max-revert-gas-reject" reload:"hot"`
	MaxAcceptableTimestampDelta  time.Duration    `koanf:"max-acceptable-timestamp-delta" reload:"hot"`
	PollInterval                 time.Duration    `koanf:"poll-interval" reload:"hot"`
	SenderWhitelist              []string         `koanf:"sender-whitelist"`
	Forwarder                    ForwarderConfig  `koanf:"forwarder"`
	QueueSize                    int              `koanf:"queue-size"`
	QueueTimeout                 time.Duration    `koanf:"queue-timeout" reload:"hot"`
	NonceCacheSize               int              `koanf:"nonce-cache-size" reload:"hot"`
	MaxTxDataSize                int              `koanf:"max-tx-data-size" reload:"hot"`
	NonceFailureCacheSize        int              `koanf:"nonce-failure-cache-size" reload:"hot"`
	NonceFailureCacheExpiry      time.Duration    `koanf:"nonce-failure-cache-expiry" reload:"hot"`
	ExpectedSurplusGasPriceMode  string           `koanf:"expected-surplus-gas-price-mode"`
	ExpectedSurplusSoftThreshold string           `koanf:"expected-surplus-soft-threshold" reload:"hot"`
	ExpectedSurplusHardThreshold string           `koanf:"expected-surplus-hard-threshold" reload:"hot"`
	EnableProfiling              bool             `koanf:"enable-profiling" reload:"hot"`
	Timeboost                    timeboost.Config `koanf:"timeboost"`
	ExperimentalPGA              PGAConfig        `koanf:"experimental-pga"`
	Dangerous                    DangerousConfig  `koanf:"dangerous"`
	expectedSurplusSoftThreshold int
	expectedSurplusHardThreshold int
}

type DangerousConfig struct {
	DisableSeqInboxMaxDataSizeCheck bool `koanf:"disable-seq-inbox-max-data-size-check"`
	DisableBlobBaseFeeCheck         bool `koanf:"disable-blob-base-fee-check"`
}

type PGAConfig struct {
	Enable         bool `koanf:"enable"`
	RoundsPerBlock uint `koanf:"rounds-per-block"`
}

const minPGARoundLength = 50 * time.Millisecond

// PGARoundLength returns the length of a PGA round. It is derived from the
// block time rather than configured directly, so MaxBlockSpeed remains the
// single source of truth.
func (c *SequencerConfig) PGARoundLength() time.Duration {
	// RoundsPerBlock is a small round count bounded by Validate; the conversion cannot overflow.
	// #nosec G115
	return c.MaxBlockSpeed / time.Duration(c.ExperimentalPGA.RoundsPerBlock)
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
	if c.ExpectedSurplusGasPriceMode != "CalldataPrice" &&
		c.ExpectedSurplusGasPriceMode != "BlobPrice" &&
		c.ExpectedSurplusGasPriceMode != "CalldataPrice7623" {
		return fmt.Errorf("undefined expected-surplus-gas-price-mode: %s", c.ExpectedSurplusGasPriceMode)
	}

	var err error
	if c.ExpectedSurplusSoftThreshold != "default" {
		if c.expectedSurplusSoftThreshold, err = strconv.Atoi(c.ExpectedSurplusSoftThreshold); err != nil {
			return fmt.Errorf("invalid expected-surplus-soft-threshold value provided in sequencer config %w", err)
		}
	}
	if c.ExpectedSurplusHardThreshold != "default" {
		if c.expectedSurplusHardThreshold, err = strconv.Atoi(c.ExpectedSurplusHardThreshold); err != nil {
			return fmt.Errorf("invalid expected-surplus-hard-threshold value provided in sequencer config %w", err)
		}
	}
	if c.expectedSurplusSoftThreshold < c.expectedSurplusHardThreshold {
		return errors.New("expected-surplus-soft-threshold cannot be lower than expected-surplus-hard-threshold")
	}
	maxTxDataSize := uint64(c.MaxTxDataSize) // #nosec G115
	if err := arbostypes.ValidateMaxTxDataSize(maxTxDataSize); err != nil {
		return err
	}
	if c.Timeboost.Enable {
		if len(c.Timeboost.AuctionContractAddress) > 0 && !common.IsHexAddress(c.Timeboost.AuctionContractAddress) {
			return fmt.Errorf("invalid timeboost.auction-contract-address \"%v\"", c.Timeboost.AuctionContractAddress)
		}
		if c.Enable {
			if c.Timeboost.RedisUrl == timeboost.DefaultConfig.RedisUrl {
				return errors.New("timeboost is enabled but no redis-url was set")
			}
			if c.Timeboost.MaxFutureSequenceDistance == 0 {
				return errors.New("timeboost max-future-sequence-distance option cannot be zero, it should be set to a positive value")
			}
			if len(c.Timeboost.AuctioneerAddress) > 0 && !common.IsHexAddress(c.Timeboost.AuctioneerAddress) {
				return fmt.Errorf("invalid timeboost.auctioneer-address \"%v\"", c.Timeboost.AuctioneerAddress)
			}
		}
	}
	if c.ExperimentalPGA.RoundsPerBlock == 0 {
		return errors.New("experimental-pga.rounds-per-block must be at least 1")
	}
	if c.ExperimentalPGA.Enable {
		if c.Timeboost.Enable {
			return errors.New("experimental-pga.enable and timeboost.enable are mutually exclusive")
		}
		if roundLength := c.PGARoundLength(); roundLength < minPGARoundLength {
			return fmt.Errorf("PGA round length %v (max-block-speed / experimental-pga.rounds-per-block) is below the minimum supported %v", roundLength, minPGARoundLength)
		}
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("sequencer poll-interval must be positive, got %v", c.PollInterval)
	}

	return nil
}

type SequencerConfigFetcher func() *SequencerConfig

var DefaultSequencerConfig = SequencerConfig{
	Enable:                      false,
	MaxBlockSpeed:               time.Millisecond * 250,
	MaxRevertGasReject:          0,
	MaxAcceptableTimestampDelta: time.Hour,
	PollInterval:                10 * time.Millisecond,
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
	ExpectedSurplusGasPriceMode:  "BlobPrice",
	ExpectedSurplusSoftThreshold: "default",
	ExpectedSurplusHardThreshold: "default",
	EnableProfiling:              false,
	Timeboost:                    timeboost.DefaultConfig,
	ExperimentalPGA:              DefaultPGAConfig,
	Dangerous:                    DefaultDangerousConfig,
}

var DefaultDangerousConfig = DangerousConfig{
	DisableSeqInboxMaxDataSizeCheck: false,
}

var DefaultPGAConfig = PGAConfig{
	Enable:         false,
	RoundsPerBlock: 2,
}

func SequencerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSequencerConfig.Enable, "act and post to l1 as sequencer")
	f.Duration(prefix+".max-block-speed", DefaultSequencerConfig.MaxBlockSpeed, "minimum delay between blocks (sets a maximum speed of block production)")
	f.Uint64(prefix+".max-revert-gas-reject", DefaultSequencerConfig.MaxRevertGasReject, "maximum gas executed in a revert for the sequencer to reject the transaction instead of posting it (anti-DOS)")
	f.Duration(prefix+".max-acceptable-timestamp-delta", DefaultSequencerConfig.MaxAcceptableTimestampDelta, "maximum acceptable time difference between the local time and the latest L1 block's timestamp")
	f.Duration(prefix+".poll-interval", DefaultSequencerConfig.PollInterval, "interval the sequencer waits before re-checking for pending work when idle")
	f.StringSlice(prefix+".sender-whitelist", DefaultSequencerConfig.SenderWhitelist, "comma separated whitelist of authorized senders (if empty, everyone is allowed)")
	AddOptionsForSequencerForwarderConfig(prefix+".forwarder", f)
	timeboost.AddOptions(prefix+".timeboost", f)
	PGAAddOptions(prefix+".experimental-pga", f)

	DangerousAddOptions(prefix+".dangerous", f)
	f.Int(prefix+".queue-size", DefaultSequencerConfig.QueueSize, "size of the pending tx queue")
	f.Duration(prefix+".queue-timeout", DefaultSequencerConfig.QueueTimeout, "maximum amount of time transaction can wait in queue")
	f.Int(prefix+".nonce-cache-size", DefaultSequencerConfig.NonceCacheSize, "size of the tx sender nonce cache")
	f.Int(prefix+".max-tx-data-size", DefaultSequencerConfig.MaxTxDataSize, "maximum transaction size the sequencer will accept")
	f.Int(prefix+".nonce-failure-cache-size", DefaultSequencerConfig.NonceFailureCacheSize, "number of transactions with too high of a nonce to keep in memory while waiting for their predecessor")
	f.Duration(prefix+".nonce-failure-cache-expiry", DefaultSequencerConfig.NonceFailureCacheExpiry, "maximum amount of time to wait for a predecessor before rejecting a tx with nonce too high")
	f.String(prefix+".expected-surplus-gas-price-mode", DefaultSequencerConfig.ExpectedSurplusGasPriceMode, "gas price setting to be used in calculating estimated surplus. Allowed values- CalldataPrice, BlobPrice and CalldataPrice7523")
	f.String(prefix+".expected-surplus-soft-threshold", DefaultSequencerConfig.ExpectedSurplusSoftThreshold, "if expected surplus is lower than this value, warnings are posted")
	f.String(prefix+".expected-surplus-hard-threshold", DefaultSequencerConfig.ExpectedSurplusHardThreshold, "if expected surplus is lower than this value, new incoming transactions will be denied")
	f.Bool(prefix+".enable-profiling", DefaultSequencerConfig.EnableProfiling, "enable CPU profiling and tracing")
}

func DangerousAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".disable-seq-inbox-max-data-size-check", DefaultDangerousConfig.DisableSeqInboxMaxDataSizeCheck, "DANGEROUS! disables nitro checks on sequencer MaxTxDataSize against the sequencer inbox MaxDataSize")
	f.Bool(prefix+".disable-blob-base-fee-check", DefaultDangerousConfig.DisableBlobBaseFeeCheck, "DANGEROUS! disables nitro checks on sequencer for blob base fee")
}

func PGAAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultPGAConfig.Enable, "EXPERIMENTAL: enable priority gas auction (PGA) transaction ordering; mutually exclusive with timeboost")
	f.Uint(prefix+".rounds-per-block", DefaultPGAConfig.RoundsPerBlock, "EXPERIMENTAL: number of PGA rounds per block; the round length is max-block-speed divided by this value")
}

func EventFilterAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".path", "", "path to JSON file containing event filter rules")
}

type txQueueItem struct {
	tx                  *types.Transaction
	txSize              int // size in bytes of the marshalled transaction
	options             *arbitrum_types.ConditionalOptions
	resultChan          chan<- error
	returnedResult      *atomic.Bool
	ctx                 context.Context
	firstAppearance     time.Time
	isTimeboosted       bool
	isAuctionResolution bool
	blockStamp          uint64 // block number at which timeboosted tx was added to the txQueue
}

func newTxQueueItem(
	ctx context.Context,
	tx *types.Transaction,
	options *arbitrum_types.ConditionalOptions,
	resultChan chan<- error,
) txQueueItem {
	return txQueueItem{
		tx:              tx,
		txSize:          int(tx.Size()), // #nosec G115
		options:         options,
		resultChan:      resultChan,
		returnedResult:  &atomic.Bool{},
		ctx:             ctx,
		firstAppearance: time.Now(),
	}
}

// newRegularTxQueueItem returns a queue item for a regular (possibly timeboosted) transaction.
func newRegularTxQueueItem(
	ctx context.Context,
	tx *types.Transaction,
	options *arbitrum_types.ConditionalOptions,
	resultChan chan<- error,
	isTimeboosted bool,
	blockStamp uint64,
) txQueueItem {
	item := newTxQueueItem(ctx, tx, options, resultChan)
	item.isTimeboosted = isTimeboosted
	item.blockStamp = blockStamp
	return item
}

// newAuctionResolutionTxQueueItem returns a queue item for an auction resolution transaction.
// Nothing reads its result, so it gets a throwaway buffered channel.
func newAuctionResolutionTxQueueItem(ctx context.Context, tx *types.Transaction) txQueueItem {
	item := newTxQueueItem(ctx, tx, nil, make(chan error, 1))
	item.isAuctionResolution = true
	return item
}

func (i *txQueueItem) returnResultMaybeLog(err error, outputLog bool) {
	if i.returnedResult.Swap(true) {
		if outputLog {
			log.Error("attempting to return result to already finished queue item", "err", err)
		}
		return
	}
	i.resultChan <- err
	close(i.resultChan)
}

func (i *txQueueItem) returnResult(err error) {
	i.returnResultMaybeLog(err, false)
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

type pendingQueueItemsResults struct {
	block      *types.Block
	queueItems []txQueueItem
	hooks      *FullSequencingHooks
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
	expressLaneService *timeboost.ExpressLaneService
	parentChain        *parent.ParentChain

	L1BlockAndTimeMutex sync.Mutex
	l1BlockNumber       atomic.Uint64
	l1Timestamp         uint64

	forwarderMutex sync.Mutex
	isActive       bool
	forwarder      *TxForwarder

	// activeUntil is the consensus chosen-sequencer deadline pushed via
	// SetActiveUntil. Nil means no coordinator signal was ever received
	// (coordinator-less setups), which CheckHealth treats as healthy.
	activeUntil atomic.Pointer[time.Time]

	expectedSurplusMutex              sync.RWMutex
	expectedSurplus                   int64
	expectedSurplusUpdated            bool
	expectedSurplusFailureCount       int
	auctioneerAddr                    common.Address
	timeboostAuctionResolutionTxQueue chan txQueueItem

	pendingQueueItemsResults *pendingQueueItemsResults
	createBlockMutex         sync.Mutex

	pendingDelayedMsgCommit bool

	eventFilter              *eventfilter.EventFilter
	addressFilterService     *addressfilter.FilterService
	pendingFilteredTxReports []addressfilter.FilteredTxReport

	// sequencingState tracks turn alternation between regular tx and delayed
	// message sequencing.
	sequencingState sequencingState
}

func NewSequencer(
	execEngine *ExecutionEngine,
	l1Reader *headerreader.HeaderReader,
	configFetcher SequencerConfigFetcher,
	parentChain *parent.ParentChain,
	eventFilter *eventfilter.EventFilter,
	addressFilterService *addressfilter.FilterService,
) (*Sequencer, error) {
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
		isActive:                          false,
		parentChain:                       parentChain,
		timeboostAuctionResolutionTxQueue: make(chan txQueueItem, 10), // There should never be more than 1 outstanding auction resolutions
		eventFilter:                       eventFilter,
		addressFilterService:              addressFilterService,
	}
	s.nonceFailures = &nonceFailureCache{
		containers.NewLruCacheWithOnEvict(config.NonceCacheSize, s.onNonceFailureEvict),
		func() time.Duration { return configFetcher().NonceFailureCacheExpiry },
	}
	s.Pause()
	execEngine.SetEventFilter(eventFilter)
	return s, nil
}

func (s *Sequencer) FilteringReady() bool {
	if s.addressFilterService == nil {
		return true
	}
	return !s.addressFilterService.GetLoadedAt().IsZero()
}

func (s *Sequencer) buildFilteredTxReport(tx *types.Transaction, header *types.Header, filteredAddresses []filter.FilteredAddressRecord, positionInBlock int) {
	if s.execEngine.filteringReportRPCClient == nil {
		return
	}
	txRLP, err := tx.MarshalBinary()
	if err != nil {
		// MarshalBinary should essentially never fail for a well-formed transaction already
		// in memory. We log instead of returning an error so that the caller can return a
		// plain ErrSeqFilter, avoiding exposure of internal operation errors (e.g.
		// marshalling failures) to end users.
		log.Error("failed to marshal transaction for filtered tx report", "err", err, "txHash", tx.Hash())
		return
	}
	report := addressfilter.FilteredTxReport{
		ID:                uuid.Must(uuid.NewV7()).String(),
		TxHash:            tx.Hash(),
		TxRLP:             txRLP,
		FilteredAddresses: filteredAddresses,
		ChainID:           s.execEngine.bc.Config().ChainID.Uint64(),
		BlockNumber:       header.Number.Uint64(),
		ParentBlockHash:   header.ParentHash,
		PositionInBlock:   uint64(positionInBlock), // #nosec G115
		FilteredAt:        time.Now().UTC(),
		IsDelayed:         false,
		DelayedReportData: nil,
	}
	s.pendingFilteredTxReports = append(s.pendingFilteredTxReports, report)
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
	forwarder := s.getForwarder()
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

func (s *Sequencer) PublishTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	forwarder := s.getForwarder()
	if forwarder != nil {
		err := forwarder.PublishTransaction(parentCtx, tx, options)
		if !errors.Is(err, ErrNoSequencer) {
			return err
		}
	}

	config := s.config()
	queueTimeout := config.QueueTimeout
	queueCtx, cancelFunc := ctxhelper.WithTimeoutOrCancel(parentCtx, queueTimeout+config.Timeboost.ExpressLaneAdvantage) // Include timeboost delay in ctx timeout
	defer cancelFunc()

	resultChan := make(chan error, 1)
	err := s.publishTransactionToQueue(queueCtx, tx, options, resultChan, false /* delay tx if express lane is active */)
	if err != nil {
		return err
	}

	now := time.Now()
	// Just to be safe, make sure we don't run over twice the queue timeout
	abortCtx, cancel := ctxhelper.WithTimeoutOrCancel(parentCtx, queueTimeout*2)
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
	if !s.config().Timeboost.Enable {
		return errors.New("timeboost not enabled")
	}

	forwarder := s.getForwarder()
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
	if *tx.To() != s.expressLaneService.AuctionContractAddr() {
		return fmt.Errorf("transaction recipient %#x is not the auction contract %#x", *tx.To(), s.expressLaneService.AuctionContractAddr())
	}
	signer := types.LatestSigner(s.execEngine.bc.Config())
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return err
	}
	if sender != auctioneerAddr {
		return fmt.Errorf("sender %#x is not the auctioneer address %#x", sender, auctioneerAddr)
	}
	roundTimingInfo := s.expressLaneService.GetRoundTimingInfo()
	if !roundTimingInfo.IsWithinAuctionCloseWindow(arrivalTime) {
		return fmt.Errorf("transaction arrival time not within auction closure window: %v", arrivalTime)
	}
	// The queues are no longer drained after the sequencer is stopped; reject
	// the tx instead of parking it.
	if s.Stopped() {
		return ErrNoSequencer
	}
	log.Info("Prioritizing auction resolution transaction from auctioneer", "txHash", tx.Hash().Hex())
	s.timeboostAuctionResolutionTxQueue <- newAuctionResolutionTxQueueItem(s.GetContext(), tx)
	return nil
}

func (s *Sequencer) PublishExpressLaneTransaction(ctx context.Context, msg *timeboost.ExpressLaneSubmission) error {
	if !s.config().Timeboost.Enable {
		return errors.New("timeboost not enabled")
	}

	preSequenceWaitStart := time.Now()
	forwarder := s.getForwarder()
	if forwarder != nil {
		return forwarder.PublishExpressLaneTransaction(ctx, msg)
	}

	if s.expressLaneService == nil {
		return errors.New("express lane service not enabled")
	}
	if err := s.expressLaneService.ValidateExpressLaneTx(msg); err != nil {
		return err
	}

	forwarder = s.getForwarder()
	if forwarder != nil {
		return forwarder.PublishExpressLaneTransaction(ctx, msg)
	}
	expressLanePreSequenceWaitHistogram.Update(time.Since(preSequenceWaitStart).Microseconds())

	return s.expressLaneService.SequenceExpressLaneSubmission(msg)
}

func (s *Sequencer) PublishTimeboostedTransaction(queueCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	resultChan := make(chan error, 1)
	return s.publishTransactionToQueue(queueCtx, tx, options, resultChan, true)
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

	if s.config().Timeboost.Enable && s.expressLaneService != nil {
		if !isExpressLaneController && s.expressLaneService.CurrentRoundHasController() {
			time.Sleep(s.config().Timeboost.ExpressLaneAdvantage)
		}
	}

	var blockStamp uint64
	if isExpressLaneController && config.Timeboost.QueueTimeoutInBlocks > 0 {
		blockStamp = s.execEngine.bc.CurrentBlock().Number.Uint64()
	}

	// The queues are no longer drained after the sequencer is stopped; reject
	// the tx instead of parking it.
	if s.Stopped() {
		return ErrNoSequencer
	}
	queueItem := newRegularTxQueueItem(queueCtx, tx, options, resultChan, isExpressLaneController, blockStamp)
	select {
	case s.txQueue <- queueItem:
	case <-queueCtx.Done():
		return queueCtx.Err()
	}
	return nil
}

func (s *Sequencer) PreTxFilter(_ *params.ChainConfig, header *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, sender common.Address, l1Info *arbos.L1Info, positionInBlock int) error {
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

	touchAddresses(statedb, tx, sender)
	if statedb.IsTxFiltered() {
		return state.ErrSeqFilter
	}

	addressFiltered, filteredAddresses := statedb.IsAddressFiltered()
	if addressFiltered {
		s.buildFilteredTxReport(tx, header, filteredAddresses, positionInBlock)
		return state.ErrSeqFilter
	}
	return nil
}

func (s *Sequencer) PostTxFilter(header *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, sender common.Address, dataGas uint64, result *core.ExecutionResult, positionInBlock int) error {
	if s.eventFilter != nil {
		logs := statedb.GetCurrentTxLogs()
		for _, l := range logs {
			for _, addr := range s.eventFilter.AddressesForFiltering(l.Topics, l.Data, l.Address) {
				statedb.TouchAddress(&addr)
			}
		}
	}
	if statedb.IsTxFiltered() {
		return state.ErrSeqFilter
	}
	if addressFiltered, filteredAddresses := statedb.IsAddressFiltered(); addressFiltered {
		s.buildFilteredTxReport(tx, header, filteredAddresses, positionInBlock)
		return state.ErrSeqFilter
	}

	// For redeems, skip nonce/revert-gas checks since those
	// don't apply to protocol-scheduled transactions.
	if tx.Type() == types.ArbitrumRetryTxType {
		return nil
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
	forwarder := s.getForwarder()
	if forwarder != nil {
		return forwarder.CheckHealth(ctx)
	}
	if !s.IsActive() {
		return nil
	}
	if activeUntil := s.activeUntil.Load(); activeUntil != nil && time.Now().After(*activeUntil) {
		return ErrNotChosenSequencer
	}
	return nil
}

func (s *Sequencer) SetActiveUntil(deadline time.Time) {
	s.activeUntil.Store(&deadline)
}

func (s *Sequencer) ForwardTarget() string {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder == nil {
		return ""
	}
	return s.forwarder.PrimaryTarget()
}

func (s *Sequencer) ForwardTo(url string) error {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
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
	s.isActive = false
	return err
}

func (s *Sequencer) Activate() {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder != nil {
		s.forwarder.Disable()
		s.forwarder = nil
	}
	if s.expressLaneService != nil {
		s.LaunchThread(func(context.Context) {
			// We launch redis sync (which is best effort) in parallel to avoid blocking sequencer activation
			s.expressLaneService.SyncFromRedis()
			time.Sleep(time.Second)
			s.expressLaneService.SyncFromRedis()
		})
	}
	s.isActive = true
}

func (s *Sequencer) Pause() {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder != nil {
		s.forwarder.Disable()
		s.forwarder = nil
	}
	s.isActive = false
}

var ErrNoSequencer = errors.New("sequencer temporarily not available")

// ErrNotChosenSequencer reports an active sequencer whose consensus lockout
// deadline has passed: it is no longer the chosen sequencer and commits would
// be rejected until the coordinator pauses it or it reacquires the lockout.
var ErrNotChosenSequencer = errors.New("sequencer is not the chosen sequencer")

func (s *Sequencer) getForwarder() *TxForwarder {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	return s.forwarder
}

func (s *Sequencer) IsActive() bool {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	return s.isActive
}

func (s *Sequencer) handleInactive(forwarder *TxForwarder, queueItems []txQueueItem) {
	if forwarder == nil {
		return
	}

	// Drain the parked nonce failures and forward them along with the queue items.
	for {
		_, failure, ok := s.nonceFailures.GetOldest()
		if !ok {
			break
		}
		failure.revived = true // prevent the eviction hook from returning a result
		s.nonceFailures.RemoveOldest()
		queueItems = append(queueItems, failure.queueItem)
	}

	publishResults := make(chan *txQueueItem, len(queueItems))
	for _, item := range queueItems {
		// Skip abandoned submissions: the forwarder ignores the item ctx, so a
		// drained item whose submitter timed out would still be forwarded.
		// Auction resolution items are exempt: their ctx is the sequencer's own
		// lifecycle context, which is already canceled during shutdown.
		if !item.isAuctionResolution {
			if err := item.ctx.Err(); err != nil {
				item.returnResult(err)
				publishResults <- nil
				continue
			}
		}
		go func() {
			var err error
			if item.isAuctionResolution {
				err = forwarder.PublishAuctionResolutionTransaction(item.ctx, item.tx)
			} else {
				err = forwarder.PublishTransaction(item.ctx, item.tx, item.options)
			}
			if errors.Is(err, ErrNoSequencer) {
				publishResults <- &item
			} else {
				if err != nil {
					log.Warn("failed to forward transaction", "txHash", item.tx.Hash(), "err", err)
				}
				publishResults <- nil
				item.returnResult(err)
			}
		}()
	}
	for range queueItems {
		remainingItem := <-publishResults
		if remainingItem != nil {
			s.txRetryQueue.Push(*remainingItem)
		}
	}
}

var sequencerInternalError = errors.New("sequencer internal error")

func (s *Sequencer) expireNonceFailures() {
	defer nonceFailureCacheSizeGauge.Update(int64(s.nonceFailures.Len()))
	for {
		_, failure, ok := s.nonceFailures.GetOldest()
		if !ok {
			return
		}
		if time.Until(failure.expiry) > 0 {
			return
		}

		// Check queueCtx status before notifying client
		queueItem := failure.queueItem
		err := queueItem.ctx.Err()
		if err != nil {
			// queueCtx has already timed out, return that error
			queueItem.returnResultMaybeLog(err, true)
		} else {
			// nonce-failure-cache-expiry timeout, return the original nonce error
			queueItem.returnResultMaybeLog(failure.nonceErr, true)
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
	arbosVersion := types.DeserializeHeaderExtraInformation(latestHeader).ArbOSFormatVersion
	signer := types.MakeSigner(bc.Config(), nextHeaderNumber, latestHeader.Time, arbosVersion)
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

// drainQueueItems snapshots each queue's length and drains up to that many items in priority
// order (auction resolution, then retry, then submitted). Safe for concurrent consumers: items
// taken by another consumer between the snapshot and the read are skipped, and concurrent pushes
// are left for the next drain.
func drainQueueItems(
	txQueue chan txQueueItem,
	txRetryQueue *synchronizedTxQueue,
	auctionResolutionTxQueue chan txQueueItem,
) []txQueueItem {
	var queueItems []txQueueItem
	for range len(auctionResolutionTxQueue) {
		select {
		case queueItem := <-auctionResolutionTxQueue:
			log.Debug("Popped the auction resolution tx", "txHash", queueItem.tx.Hash())
			queueItems = append(queueItems, queueItem)
		default:
		}
	}
	for range txRetryQueue.Len() {
		queueItem := txRetryQueue.Pop()
		if queueItem.tx == nil {
			// Pop returned the zero value: another consumer emptied the queue.
			break
		}
		queueItems = append(queueItems, queueItem)
	}
	for range len(txQueue) {
		select {
		case queueItem := <-txQueue:
			queueItems = append(queueItems, queueItem)
		default:
		}
	}
	return queueItems
}

// validateQueueItem returns the reason a drained item must be dropped (canceled item ctx,
// oversized tx, timeboost block-age expiry, fee cap below basefee), or nil to sequence it.
func validateQueueItem(config *SequencerConfig, currentHeader *types.Header, queueItem txQueueItem) error {
	if err := queueItem.ctx.Err(); err != nil {
		return err
	}
	if queueItem.txSize > config.MaxTxDataSize {
		return txpool.ErrOversizedData
	}
	if queueItem.isTimeboosted && queueItem.blockStamp != 0 &&
		currentHeader.Number.Uint64() >= queueItem.blockStamp+config.Timeboost.QueueTimeoutInBlocks {
		err := fmt.Errorf("timeboosted tx: %s has hit block based timeout. currentBlockNum: %d, blockStamp: %d, blockExpiry: %d",
			queueItem.tx.Hash(),
			currentHeader.Number.Uint64()+1,
			queueItem.blockStamp,
			queueItem.blockStamp+config.Timeboost.QueueTimeoutInBlocks,
		)
		// The timeboost result isn't read by anyone, so we log the error
		log.Info("Error sequencing timeboost tx", "err", err)
		return err
	}
	if arbmath.BigLessThan(queueItem.tx.GasFeeCap(), currentHeader.BaseFee) {
		return fmt.Errorf("%w: maxFeePerGas: %s baseFee: %s", core.ErrFeeCapTooLow, queueItem.tx.GasFeeCap(), currentHeader.BaseFee)
	}
	return nil
}

// drainAndValidateQueueItems drains the queues and filters out the invalid items, returning the
// validation failure to each dropped item's submitter.
func drainAndValidateQueueItems(
	config *SequencerConfig,
	txQueue chan txQueueItem,
	txRetryQueue *synchronizedTxQueue,
	auctionResolutionTxQueue chan txQueueItem,
	currentHeader *types.Header,
) []txQueueItem {
	unvalidatedItems := drainQueueItems(txQueue, txRetryQueue, auctionResolutionTxQueue)
	var queueItems []txQueueItem
	for _, queueItem := range unvalidatedItems {
		if err := validateQueueItem(config, currentHeader, queueItem); err != nil {
			queueItem.returnResult(err)
		} else {
			queueItems = append(queueItems, queueItem)
		}
	}
	return queueItems
}

func (s *Sequencer) createBlockWithRegularTxs(ctx context.Context) (sequencedMsg *execution.SequencedMsg, throttleRegularSequencingFor time.Duration) {
	s.createBlockMutex.Lock()
	defer s.createBlockMutex.Unlock()

	s.pendingQueueItemsResults = nil

	forwarder := s.getForwarder()

	var queueItems []txQueueItem

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
			s.pendingQueueItemsResults = nil
			// Wait for the MaxBlockSpeed until attempting to create a block again
			throttleRegularSequencingFor = s.config().MaxBlockSpeed
		}
	}()
	defer nonceFailureCacheSizeGauge.Update(int64(s.nonceFailures.Len()))

	config := s.config()

	if ctx.Err() != nil {
		// Context canceled; retry promptly.
		return nil, 0
	}
	txQueueLen := int64(len(s.txQueue))
	sequencerQueueGauge.Update(txQueueLen)
	sequencerQueueHistogram.Update(txQueueLen)

	if forwarder != nil {
		// We are forwarding (no longer the active sequencer); forward everything
		// drained, unvalidated, and do not sequence locally.
		queueItems = drainQueueItems(s.txQueue, &s.txRetryQueue, s.timeboostAuctionResolutionTxQueue)
		s.handleInactive(forwarder, queueItems)
		return nil, config.MaxBlockSpeed
	}

	queueItems = drainAndValidateQueueItems(config, s.txQueue, &s.txRetryQueue, s.timeboostAuctionResolutionTxQueue, s.execEngine.bc.CurrentBlock())
	if queueItems == nil {
		// No regular txs to sequence right now; re-check on the idle poll
		// cadence rather than waiting a full block interval. This matches the
		// wait decideSequencingTurn uses when there is no pending work.
		return nil, min(config.PollInterval, config.MaxBlockSpeed)
	}

	s.nonceCache.Resize(config.NonceCacheSize) // Would probably be better in a config hook but this is basically free
	s.nonceCache.BeginNewBlock()
	queueItems = s.precheckNonces(queueItems)
	maxTxDataSize := s.config().MaxTxDataSize
	hooks := MakeSequencingHooks(
		queueItems,
		maxTxDataSize,
		s,
	)

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
		return nil, config.MaxBlockSpeed
	}

	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: l1Block,
		Timestamp:   arbmath.SaturatingUCast[uint64](timestamp),
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	s.pendingFilteredTxReports = nil

	start := time.Now()
	var (
		block *types.Block
		err   error
	)
	if config.EnableProfiling {
		sequencedMsg, block, err = s.execEngine.SequenceTransactionsWithProfiling(header, hooks)
	} else {
		sequencedMsg, block, err = s.execEngine.SequenceTransactions(header, hooks)
	}

	if len(s.pendingFilteredTxReports) > 0 && s.execEngine.filteringReportRPCClient != nil {
		reports := s.pendingFilteredTxReports
		s.LaunchThread(func(ctx context.Context) {
			if _, err := s.execEngine.filteringReportRPCClient.ReportFilteredTransactions(ReportProducerSequencer, reports).Await(ctx); err != nil {
				log.Error("failed to report filtered transactions", "count", len(reports), "err", err)
			}
		})
	}
	s.pendingFilteredTxReports = nil
	elapsed := time.Since(start)
	blockCreationTimer.Update(elapsed.Nanoseconds())
	if elapsed >= time.Second*5 {
		var blockNum *big.Int
		if block != nil {
			blockNum = block.Number()
		}
		log.Warn("took over 5 seconds to sequence a block", "elapsed", elapsed, "numTxes", hooks.sequencedQueueItemsCount, "success", block != nil, "l2Block", blockNum)
	}
	if err == nil {
		if len(hooks.txErrors) != hooks.sequencedQueueItemsCount { // This is not supposed to happen, if so we have a bug
			err = fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(hooks.txErrors), hooks.sequencedQueueItemsCount)
		} else {
			for i := hooks.sequencedQueueItemsCount; i < len(hooks.queueItems); i++ {
				s.txRetryQueue.Push(hooks.queueItems[i])
			}
		}
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// thread closed. We'll later try to forward these messages.
			for _, item := range queueItems {
				s.txRetryQueue.Push(item)
			}
			return nil, config.MaxBlockSpeed // don't return failure to avoid retrying immediately
		}
		log.Error("error sequencing transactions", "err", err)
		for _, queueItem := range queueItems {
			queueItem.returnResult(err)
		}
		return nil, 0
	}

	madeBlock := false
	for _, err := range hooks.txErrors {
		if err == nil {
			madeBlock = true
		}
	}

	s.pendingQueueItemsResults = &pendingQueueItemsResults{
		block:      block,
		hooks:      hooks,
		queueItems: queueItems,
	}

	if madeBlock {
		// Rate-limit block production to at most one block per MaxBlockSpeed.
		return sequencedMsg, config.MaxBlockSpeed
	}
	// Items were present but no block was produced (e.g. all txs failed this
	// round); retry promptly.
	return sequencedMsg, 0
}

func (s *Sequencer) EndSequencing(ctx context.Context, errWhileSequencing error) {
	s.createBlockMutex.Lock()
	defer s.createBlockMutex.Unlock()

	// The staged results are consumed on every path below; clearing them in one
	// place keeps a no-op turn's EndSequencing from re-processing stale results.
	defer func() { s.pendingQueueItemsResults = nil }()

	if s.pendingDelayedMsgCommit {
		s.pendingDelayedMsgCommit = false
		if errWhileSequencing == nil {
			// The block was durably written; remove the delayed message now.
			s.execEngine.popSequencedDelayedMessage()
		}
		// On error (e.g. ErrRetrySequencer) leave the message queued for retry.
		return
	}

	if s.pendingQueueItemsResults == nil {
		return
	}

	if errors.Is(errWhileSequencing, execution.ErrRetrySequencer) {
		forwarder := s.getForwarder()
		if forwarder != nil {
			// forward if we have where to
			s.handleInactive(forwarder, s.pendingQueueItemsResults.queueItems)
			return
		}

		// adds back to queue otherwise
		for _, item := range s.pendingQueueItemsResults.queueItems {
			s.txRetryQueue.Push(item)
		}
		return
	}

	if errWhileSequencing != nil {
		for _, queueItem := range s.pendingQueueItemsResults.queueItems {
			queueItem.returnResult(errWhileSequencing)
		}
	} else {
		if s.pendingQueueItemsResults.block != nil {
			successfulBlocksCounter.Inc(1)
			s.nonceCache.Finalize(s.pendingQueueItemsResults.block)
		}

		madeBlock := false
		var blockTxSize int64
		blockGasLimitReached := false
		for i, err := range s.pendingQueueItemsResults.hooks.txErrors {
			queueItem := s.pendingQueueItemsResults.queueItems[i]
			if err == nil {
				madeBlock = true
				blockTxSize += int64(queueItem.txSize)
				txSizeHistogram.Update(int64(queueItem.txSize))
			}
			if errors.Is(err, core.ErrGasLimitReached) {
				// There's not enough gas left in the block for this tx.
				if madeBlock {
					blockGasLimitReached = true
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

		if madeBlock {
			blockTxSizeHistogram.Update(blockTxSize)
			if s.pendingQueueItemsResults.hooks.txSizeLimitReached {
				dataLimitedBlocksCounter.Inc(1)
			} else if blockGasLimitReached {
				gasLimitedBlocksCounter.Inc(1)
			} else {
				// no transactions were skipped due to block size or gas limit
				txExhaustedBlocksCounter.Inc(1)
			}
		}
	}
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
	auctioneerAddr common.Address,
	roundTimingInfo *timeboost.RoundTimingInfo,
	expressLaneTracker *timeboost.ExpressLaneTracker,
) error {
	configFetcher := func() *timeboost.ExpressLaneServiceConfig {
		c := s.config()
		return &timeboost.ExpressLaneServiceConfig{
			QueueTimeout:                 c.QueueTimeout,
			MaxFutureSequenceDistance:    c.Timeboost.MaxFutureSequenceDistance,
			RedisUrl:                     c.Timeboost.RedisUrl,
			RedisUpdateEventsChannelSize: c.Timeboost.RedisUpdateEventsChannelSize,
		}
	}
	els, err := timeboost.NewExpressLaneService(
		s,
		configFetcher,
		roundTimingInfo,
		expressLaneTracker,
	)
	if err != nil {
		return fmt.Errorf("failed to create express lane service. err: %w", err)
	}
	s.auctioneerAddr = auctioneerAddr
	s.expressLaneService = els
	return nil
}

const maxConsecutiveExpectedSurplusFailures = 20

var (
	usableBytesInBlob    = big.NewInt(int64(len(kzg4844.Blob{}) * 31 / 32))
	blobTxBlobGasPerBlob = big.NewInt(params.BlobTxBlobGasPerBlob)
)

func (s *Sequencer) logExpectedSurplusError(err error) {
	s.expectedSurplusFailureCount++

	logLevel := log.Error
	if s.expectedSurplusFailureCount <= maxConsecutiveExpectedSurplusFailures {
		logLevel = log.Warn
	}

	logLevel("expected surplus soft/hard thresholds are enabled but unable to fetch latest expected surplus, retrying",
		"err", err,
		"consecutiveFailures", s.expectedSurplusFailureCount)
}

func (s *Sequencer) updateExpectedSurplus(ctx context.Context) (int64, error) {
	header, err := s.l1Reader.LastHeader(ctx)
	if err != nil {
		return 0, fmt.Errorf("error encountered getting latest header from l1reader while updating expectedSurplus: %w", err)
	}
	l1GasPrice := header.BaseFee.Int64()

	// #nosec G115
	backlogCallDataUnits := int64(s.execEngine.backlogCallDataUnits())
	var backlogCost int64 // tx's cached calldata units are already scaled by TxDataNonZeroGasEIP2028 = 16, so we divide them by 16 while calculating cost for blobs and for EIP7623 pricing accordingly
	switch s.config().ExpectedSurplusGasPriceMode {
	case "CalldataPrice":
		backlogCost = backlogCallDataUnits * header.BaseFee.Int64()
	case "BlobPrice":
		if s.config().Dangerous.DisableBlobBaseFeeCheck {
			if !s.expectedSurplusUpdated {
				// only print notification once
				log.Info("expected surplus calculation is set to use blob price but --execution.sequencer.dangerous.disable-blob-base-fee-check is set, falling back to calldata price model")
			}
			backlogCost = backlogCallDataUnits * header.BaseFee.Int64()
		} else if header.BlobGasUsed == nil || header.ExcessBlobGas == nil {
			if !s.expectedSurplusUpdated {
				// only print notification once
				log.Info("expected surplus calculation is set to use blob price but latest parent chain header has BlobGasUsed or ExcessBlobGas as nil, falling back to calldata price model")
			}
			backlogCost = backlogCallDataUnits * header.BaseFee.Int64()
		} else {
			blobFeePerByte, err := s.parentChain.BlobFeePerByte(ctx, header)
			if err != nil {
				return 0, fmt.Errorf("error encountered getting blob base fee while updating expectedSurplus: %w", err)
			}

			// We want to calculate the following two values:
			// - l1GasPrice = (blobFeePerByte * blobTxBlobGasPerBlob) / (usableBytesInBlob * 16)
			// - backlogCost = backlogCallDataUnits * (blobFeePerByte * blobTxBlobGasPerBlob) / (usableBytesInBlob * 16)
			// If we divide by usableBytesInBlob too early, the value of blobFeePerByte becomes zero because of rounding.
			// Then even if we multiply with backlogCallDataUnits, the value will still remain zero.
			// This is why we multiply with backlogCallDataUnits before we divide.
			if backlogCallDataUnits == 0 {
				blobFeePerByte.Mul(blobFeePerByte, blobTxBlobGasPerBlob)
				blobFeePerByte.Div(blobFeePerByte, usableBytesInBlob)
				l1GasPrice = blobFeePerByte.Int64() / 16
				backlogCost = 0
			} else {
				// l1GasPrice can be zero because of roundings, hence backlogCost is calculated separately
				backlogFee := big.NewInt(backlogCallDataUnits)
				backlogFee.Mul(backlogFee, blobFeePerByte)
				backlogFee.Mul(backlogFee, blobTxBlobGasPerBlob)
				backlogFee.Div(backlogFee, usableBytesInBlob)
				backlogCost = backlogFee.Int64() / 16
				l1GasPrice = backlogCost / backlogCallDataUnits
			}
		}
	case "CalldataPrice7623":
		l1GasPrice = (header.BaseFee.Int64() * 40) / 16
		backlogCost = (backlogCallDataUnits * header.BaseFee.Int64() * 40) / 16
	default:
		return 0, fmt.Errorf("unrecognized ExpectedSurplusGasPriceMode: %s", s.config().ExpectedSurplusGasPriceMode)
	}

	surplus, err := s.execEngine.getL1PricingSurplus()
	if err != nil {
		return 0, fmt.Errorf("error encountered getting l1 pricing surplus while updating expectedSurplus: %w", err)
	}
	expectedSurplus := surplus - backlogCost

	// update metrics
	l1GasPriceGauge.Update(l1GasPrice)
	callDataUnitsBacklogGauge.Update(backlogCallDataUnits)
	currentSurplusGauge.Update(surplus)
	expectedSurplusGauge.Update(expectedSurplus)
	config := s.config()
	if config.ExpectedSurplusSoftThreshold != "default" && expectedSurplus < int64(config.expectedSurplusSoftThreshold) {
		log.Warn("expected surplus is below soft threshold", "value", expectedSurplus, "threshold", config.expectedSurplusSoftThreshold)
	}
	s.expectedSurplusFailureCount = 0
	return expectedSurplus, nil
}

func (s *Sequencer) StartExpressLaneService(ctx context.Context) {
	if s.expressLaneService != nil {
		s.expressLaneService.Start(ctx)
	}
}

func (s *Sequencer) backgroundForwarder(_ context.Context) time.Duration {
	s.createBlockMutex.Lock()
	defer s.createBlockMutex.Unlock()

	config := s.config()

	// Turns servicing expiry in StartSequencing are not guaranteed to happen
	// (the sequencer may be inactive, or sequencing may be yielding while exec
	// catches up to consensus); expire here so parked nonce-gap txs get their
	// prompt nonce error instead of waiting out the queue-timeout abort.
	s.nonceFailures.Resize(config.NonceFailureCacheSize)
	s.expireNonceFailures()

	forwarder := s.getForwarder()
	if forwarder != nil {
		queueItems := drainQueueItems(s.txQueue, &s.txRetryQueue, s.timeboostAuctionResolutionTxQueue)
		s.handleInactive(forwarder, queueItems)
	}
	return config.PollInterval
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
				s.logExpectedSurplusError(err)
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

	s.CallIteratively(s.backgroundForwarder)

	return nil
}

func (s *Sequencer) hasPendingRegularTxs() bool {
	return s.txRetryQueue.Len() > 0 ||
		len(s.txQueue) > 0 ||
		len(s.timeboostAuctionResolutionTxQueue) > 0
}

func (s *Sequencer) StartSequencing(ctx context.Context) (*execution.SequencedMsg, time.Duration) {
	s.pendingDelayedMsgCommit = false
	config := s.config()

	// Service the nonceFailures cache every turn, regardless of which turn is
	// taken (or none), so stuck entries are expired even when there are no other
	// pending regular txs.
	s.createBlockMutex.Lock()
	s.nonceFailures.Resize(config.NonceFailureCacheSize)
	s.expireNonceFailures()
	s.createBlockMutex.Unlock()

	now := time.Now()
	turn, wait := decideSequencingTurn(
		s.sequencingState,
		s.hasPendingRegularTxs(),
		s.execEngine.hasPendingDelayedMsgs(),
		now,
		min(config.PollInterval, config.MaxBlockSpeed),
	)
	switch turn {
	case regularSequencingTurn:
		sequencedMsg, throttleRegularSequencingFor := s.createBlockWithRegularTxs(ctx)
		s.sequencingState = sequencingStateAfterRegularSequencing(s.sequencingState, throttleRegularSequencingFor, now)
		return sequencedMsg, 0
	case delayedSequencingTurn:
		sequencedMsg, err := s.execEngine.SequenceDelayedMessage()
		producedMsg := err == nil && sequencedMsg != nil
		s.pendingDelayedMsgCommit = producedMsg
		s.sequencingState = sequencingStateAfterDelayedSequencing(s.sequencingState, producedMsg, now, config.MaxBlockSpeed)
		if err != nil {
			return nil, 0
		}
		return sequencedMsg, 0
	default:
		return nil, wait
	}
}

func (s *Sequencer) StopAndWait() {
	// expressLaneService is started by ExecutionNode via StartExpressLaneService,
	// but stopped here because the Sequencer owns it.
	if s.config().Timeboost.Enable && s.expressLaneService != nil {
		s.expressLaneService.StopAndWait()
	}
	s.StopWaiter.StopAndWait()
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
	forwarder := s.getForwarder()
	if forwarder == nil {
		return
	}
	// Bound the whole drain: a hung forwarding target must not hang shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The existing forwarder runs on the sequencer's StopWaiter context, which
	// was canceled above, so its publishes would fail immediately. Create a
	// fresh forwarder on the shutdown context for the final drain.
	forwarder = NewForwarder(forwarder.targets, &s.config().Forwarder)
	if err := forwarder.Initialize(shutdownCtx); err != nil {
		log.Error("failed to initialize the shutdown forwarder; dropping the queued transactions", "err", err)
		return
	}
	defer forwarder.StopAndWait()
	// Drain in a loop: txs may be enqueued while a batch is being forwarded.
	for shutdownCtx.Err() == nil {
		queueItems := drainQueueItems(s.txQueue, &s.txRetryQueue, s.timeboostAuctionResolutionTxQueue)
		if len(queueItems) == 0 && s.nonceFailures.Len() == 0 {
			return
		}
		s.handleInactive(forwarder, queueItems)
	}
	log.Error("gave up draining the queues while shutting down; some transactions were not forwarded",
		"txQueue", len(s.txQueue),
		"retryQueue", s.txRetryQueue.Len(),
		"nonceFailures", s.nonceFailures.Len(),
		"timeboostAuctionResolutionTxQueue", len(s.timeboostAuctionResolutionTxQueue))
}

func (s *Sequencer) MakeSameBlockSequencingHooksAndHeaderForTest(t *testing.T, txes types.Transactions) (*arbostypes.L1IncomingMessageHeader, *FullSequencingHooks) {
	t.Helper()
	hooks := MakeZeroTxSizeSequencingHooksForTesting(txes, s, nil)

	s.L1BlockAndTimeMutex.Lock()
	l1Block := s.l1BlockNumber.Load()
	s.L1BlockAndTimeMutex.Unlock()

	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: l1Block,
		Timestamp:   arbmath.SaturatingUCast[uint64](time.Now().Unix()),
	}

	s.pendingFilteredTxReports = nil

	return header, hooks
}

func (s *Sequencer) DispatchPendingFilteredTxReportsForTest(t *testing.T) {
	t.Helper()
	if len(s.pendingFilteredTxReports) > 0 && s.execEngine.filteringReportRPCClient != nil {
		reports := s.pendingFilteredTxReports
		s.LaunchThread(func(ctx context.Context) {
			if _, err := s.execEngine.filteringReportRPCClient.ReportFilteredTransactions(ReportProducerSequencer, reports).Await(ctx); err != nil {
				log.Error("failed to report filtered transactions", "count", len(reports), "err", err)
			}
		})
	}
	s.pendingFilteredTxReports = nil
}

func (s *Sequencer) SetAddressFilterServiceForTest(t *testing.T, service *addressfilter.FilterService) {
	t.Helper()
	s.addressFilterService = service
}
