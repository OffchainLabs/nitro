// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/arbnode/redislock"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	batchPosterWalletBalance      = metrics.NewRegisteredGaugeFloat64("arb/batchposter/wallet/eth", nil)
	batchPosterGasRefunderBalance = metrics.NewRegisteredGaugeFloat64("arb/batchposter/gasrefunder/eth", nil)
	baseFeeGauge                  = metrics.NewRegisteredGauge("arb/batchposter/basefee", nil)
	blobFeeGauge                  = metrics.NewRegisteredGauge("arb/batchposter/blobfee", nil)
	l1GasPriceGauge               = metrics.NewRegisteredGauge("arb/batchposter/l1gasprice", nil)
	blockGasUsedGauge             = metrics.NewRegisteredGauge("arb/batchposter/blockgas/used", nil)
	blockGasLimitGauge            = metrics.NewRegisteredGauge("arb/batchposter/blockgas/limit", nil)
	blobGasUsedGauge              = metrics.NewRegisteredGauge("arb/batchposter/blobgas/used", nil)
	blobGasLimitGauge             = metrics.NewRegisteredGauge("arb/batchposter/blobgas/limit", nil)
	suggestedTipCapGauge          = metrics.NewRegisteredGauge("arb/batchposter/suggestedtipcap", nil)

	batchPosterEstimatedBatchBacklogGauge = metrics.NewRegisteredGauge("arb/batchposter/estimated_batch_backlog", nil)

	batchPosterDALastSuccessfulActionGauge = metrics.NewRegisteredGauge("arb/batchPoster/action/da_last_success", nil)
	batchPosterDASuccessCounter            = metrics.NewRegisteredCounter("arb/batchPoster/action/da_success", nil)
	batchPosterDAFailureCounter            = metrics.NewRegisteredCounter("arb/batchPoster/action/da_failure", nil)

	batchPosterFailureCounter = metrics.NewRegisteredCounter("arb/batchPoster/action/failure", nil)

	usableBytesInBlob    = big.NewInt(int64(len(kzg4844.Blob{}) * 31 / 32))
	blobTxBlobGasPerBlob = big.NewInt(params.BlobTxBlobGasPerBlob)
)

const (
	batchPosterSimpleRedisLockKey = "node.batch-poster.redis-lock.simple-lock-key"

	sequencerBatchPostMethodName                    = "addSequencerL2BatchFromOrigin0"
	sequencerBatchPostWithBlobsMethodName           = "addSequencerL2BatchFromBlobs"
	sequencerBatchPostDelayProofMethodName          = "addSequencerL2BatchFromOriginDelayProof"
	sequencerBatchPostWithBlobsDelayProofMethodName = "addSequencerL2BatchFromBlobsDelayProof"

	// Overhead/safety margin for 4844 blob batch encoding (subtracted from max blob capacity)
	blobBatchEncodingOverhead = 2000
	// Size of the L1 sequencer message header (5 uint64 fields: min/max timestamp, min/max block number, after delayed messages read)
	SequencerMessageHeaderSize = 40
)

type batchPosterPosition struct {
	MessageCount        arbutil.MessageIndex
	DelayedMessageCount uint64
	NextSeqNum          uint64
}

type BatchPoster struct {
	stopwaiter.StopWaiter
	l1Reader           *headerreader.HeaderReader
	inbox              *InboxTracker
	streamer           *TransactionStreamer
	arbOSVersionGetter execution.ArbOSVersionGetter
	config             BatchPosterConfigFetcher
	seqInbox           *bridgegen.SequencerInbox
	syncMonitor        *SyncMonitor
	seqInboxABI        *abi.ABI
	seqInboxAddr       common.Address
	bridgeAddr         common.Address
	gasRefunderAddr    common.Address
	building           *buildingBatch
	dapWriters         []daprovider.Writer
	dapReaders         *daprovider.DAProviderRegistry
	dataPoster         *dataposter.DataPoster
	redisLock          *redislock.Simple
	messagesPerBatch   *arbmath.MovingAverage[uint64]
	non4844BatchCount  int // Count of consecutive non-4844 batches posted
	// This is an atomic variable that should only be accessed atomically.
	// An estimate of the number of batches we want to post but haven't yet.
	// This doesn't include batches which we don't want to post yet due to the L1 bounds.
	backlog         atomic.Uint64
	lastHitL1Bounds time.Time // The last time we wanted to post a message but hit the L1 bounds

	batchReverted          atomic.Bool // indicates whether data poster batch was reverted
	nextRevertCheckBlock   int64       // the last parent block scanned for reverting batches
	postedFirstBatch       bool        // indicates if batch poster has posted the first batch
	ethDAFallbackRemaining int         // when >0, use EthDA and decrement; when 0, use altDA
	currentWriterIndex     int         // index of DA writer to use (reset to 0 after success)

	accessList   func(SequencerInboxAccs, AfterDelayedMessagesRead uint64) types.AccessList
	parentChain  *parent.ParentChain
	checkEip7623 bool
	useEip7623   bool
}

type l1BlockBound int

// This enum starts at 1 to avoid the empty initialization of 0 being valid
const (
	// Default is Safe if the L1 reader has finality data enabled, otherwise Latest
	l1BlockBoundDefault l1BlockBound = iota + 1
	l1BlockBoundSafe
	l1BlockBoundFinalized
	l1BlockBoundLatest
	l1BlockBoundIgnore
)

type BatchPosterDangerousConfig struct {
	AllowPostingFirstBatchWhenSequencerMessageCountMismatch bool   `koanf:"allow-posting-first-batch-when-sequencer-message-count-mismatch"`
	FixedGasLimit                                           uint64 `koanf:"fixed-gas-limit"`
}

type BatchPosterConfig struct {
	Enable                             bool `koanf:"enable"`
	DisableDapFallbackStoreDataOnChain bool `koanf:"disable-dap-fallback-store-data-on-chain" reload:"hot"`
	// Number of batches to post to EthDA before retrying AltDA after a fallback.
	EthDAFallbackBatchCount int `koanf:"ethda-fallback-batch-count" reload:"hot"`
	// Deprecated: use MaxCalldataBatchSize instead. Will be removed in next version.
	MaxSize int `koanf:"max-size" reload:"hot"`
	// Maximum calldata batch size for EthDA.
	MaxCalldataBatchSize int `koanf:"max-calldata-batch-size" reload:"hot"`
	// Maximum 4844 blob enabled batch size.
	Max4844BatchSize int `koanf:"max-4844-batch-size" reload:"hot"`
	// Max batch post delay.
	MaxDelay time.Duration `koanf:"max-delay" reload:"hot"`
	// Wait for max BatchPost delay.
	WaitForMaxDelay bool `koanf:"wait-for-max-delay" reload:"hot"`
	// Batch post polling interval.
	PollInterval time.Duration `koanf:"poll-interval" reload:"hot"`
	// Batch posting error delay.
	ErrorDelay                     time.Duration               `koanf:"error-delay" reload:"hot"`
	CompressionLevel               int                         `koanf:"compression-level" reload:"hot"`
	DASRetentionPeriod             time.Duration               `koanf:"das-retention-period" reload:"hot"`
	GasRefunderAddress             string                      `koanf:"gas-refunder-address" reload:"hot"`
	DataPoster                     dataposter.DataPosterConfig `koanf:"data-poster" reload:"hot"`
	RedisUrl                       string                      `koanf:"redis-url"`
	RedisLock                      redislock.SimpleCfg         `koanf:"redis-lock" reload:"hot"`
	ExtraBatchGas                  uint64                      `koanf:"extra-batch-gas" reload:"hot"`
	Post4844Blobs                  bool                        `koanf:"post-4844-blobs" reload:"hot"`
	IgnoreBlobPrice                bool                        `koanf:"ignore-blob-price" reload:"hot"`
	ParentChainWallet              genericconf.WalletConfig    `koanf:"parent-chain-wallet"`
	L1BlockBound                   string                      `koanf:"l1-block-bound" reload:"hot"`
	L1BlockBoundBypass             time.Duration               `koanf:"l1-block-bound-bypass" reload:"hot"`
	UseAccessLists                 bool                        `koanf:"use-access-lists" reload:"hot"`
	GasEstimateBaseFeeMultipleBips arbmath.UBips               `koanf:"gas-estimate-base-fee-multiple-bips"`
	Dangerous                      BatchPosterDangerousConfig  `koanf:"dangerous"`
	ReorgResistanceMargin          time.Duration               `koanf:"reorg-resistance-margin" reload:"hot"`
	CheckBatchCorrectness          bool                        `koanf:"check-batch-correctness"`
	// MaxEmptyBatchDelay defines how long the batch poster waits before submitting a batch
	// that contains no new useful transactions (a “report-only” or “empty” batch). Set to 0 to disable it.
	MaxEmptyBatchDelay         time.Duration `koanf:"max-empty-batch-delay"`
	DelayBufferThresholdMargin uint64        `koanf:"delay-buffer-threshold-margin"`
	DelayBufferAlwaysUpdatable bool          `koanf:"delay-buffer-always-updatable"`
	ParentChainEip7623         string        `koanf:"parent-chain-eip7623"`

	gasRefunder  common.Address
	l1BlockBound l1BlockBound
}

func (c *BatchPosterConfig) Validate() error {
	if len(c.GasRefunderAddress) > 0 && !common.IsHexAddress(c.GasRefunderAddress) {
		return fmt.Errorf("invalid gas refunder address \"%v\"", c.GasRefunderAddress)
	}
	c.gasRefunder = common.HexToAddress(c.GasRefunderAddress)
	if c.MaxSize != 0 {
		log.Error("max-size is deprecated; use max-calldata-batch-size for calldata batches, or data-availability.max-batch-size for AnyTrust; max-size will be removed in a future release")
		if c.MaxCalldataBatchSize == DefaultBatchPosterConfig.MaxCalldataBatchSize {
			c.MaxCalldataBatchSize = c.MaxSize
		} else {
			return errors.New("both max-size (deprecated) and max-calldata-batch-size are set; please use only max-calldata-batch-size")
		}
	}
	if c.MaxCalldataBatchSize <= SequencerMessageHeaderSize {
		return errors.New("MaxCalldataBatchSize too small")
	}
	if c.L1BlockBound == "" {
		c.l1BlockBound = l1BlockBoundDefault
	} else if c.L1BlockBound == "safe" {
		c.l1BlockBound = l1BlockBoundSafe
	} else if c.L1BlockBound == "finalized" {
		c.l1BlockBound = l1BlockBoundFinalized
	} else if c.L1BlockBound == "latest" {
		c.l1BlockBound = l1BlockBoundLatest
	} else if c.L1BlockBound == "ignore" {
		c.l1BlockBound = l1BlockBoundIgnore
	} else {
		return fmt.Errorf("invalid L1 block bound tag \"%v\" (see --help for options)", c.L1BlockBound)
	}
	return nil
}

type BatchPosterConfigFetcher func() *BatchPosterConfig

func DangerousBatchPosterConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".allow-posting-first-batch-when-sequencer-message-count-mismatch", DefaultBatchPosterConfig.Dangerous.AllowPostingFirstBatchWhenSequencerMessageCountMismatch, "allow posting the first batch even if sequence number doesn't match chain (useful after force-inclusion)")
	f.Uint64(prefix+".fixed-gas-limit", DefaultBatchPosterConfig.Dangerous.FixedGasLimit, "use this gas limit for batch posting instead of estimating it")
}

func BatchPosterConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Bool(prefix+".disable-dap-fallback-store-data-on-chain", DefaultBatchPosterConfig.DisableDapFallbackStoreDataOnChain, "If unable to batch to DA provider, disable fallback storing data on chain")
	f.Int(prefix+".ethda-fallback-batch-count", DefaultBatchPosterConfig.EthDAFallbackBatchCount, "number of batches to post to EthDA before retrying AltDA after a fallback")
	f.Int(prefix+".max-size", DefaultBatchPosterConfig.MaxSize, "DEPRECATED: use "+prefix+".max-calldata-batch-size instead")
	f.Int(prefix+".max-calldata-batch-size", DefaultBatchPosterConfig.MaxCalldataBatchSize, "maximum estimated compressed calldata batch size")
	f.Int(prefix+".max-4844-batch-size", DefaultBatchPosterConfig.Max4844BatchSize, "maximum estimated compressed 4844 blob enabled batch size")
	f.Duration(prefix+".max-delay", DefaultBatchPosterConfig.MaxDelay, "maximum batch posting delay")
	f.Bool(prefix+".wait-for-max-delay", DefaultBatchPosterConfig.WaitForMaxDelay, "wait for the max batch delay, even if the batch is full")
	f.Duration(prefix+".poll-interval", DefaultBatchPosterConfig.PollInterval, "how long to wait after no batches are ready to be posted before checking again")
	f.Duration(prefix+".error-delay", DefaultBatchPosterConfig.ErrorDelay, "how long to delay after error posting batch")
	f.Int(prefix+".compression-level", DefaultBatchPosterConfig.CompressionLevel, "batch compression level")
	f.Duration(prefix+".das-retention-period", DefaultBatchPosterConfig.DASRetentionPeriod, "In AnyTrust mode, the period which DASes are requested to retain the stored batches.")
	f.String(prefix+".gas-refunder-address", DefaultBatchPosterConfig.GasRefunderAddress, "The gas refunder contract address (optional)")
	f.Uint64(prefix+".extra-batch-gas", DefaultBatchPosterConfig.ExtraBatchGas, "use this much more gas than estimation says is necessary to post batches")
	f.Bool(prefix+".post-4844-blobs", DefaultBatchPosterConfig.Post4844Blobs, "if the parent chain supports 4844 blobs and they're well priced, post EIP-4844 blobs")
	f.Bool(prefix+".ignore-blob-price", DefaultBatchPosterConfig.IgnoreBlobPrice, "if the parent chain supports 4844 blobs and ignore-blob-price is true, post 4844 blobs even if it's not price efficient")
	f.String(prefix+".redis-url", DefaultBatchPosterConfig.RedisUrl, "if non-empty, the Redis URL to store queued transactions in")
	f.String(prefix+".l1-block-bound", DefaultBatchPosterConfig.L1BlockBound, "only post messages to batches when they're within the max future block/timestamp as of this L1 block tag (\"safe\", \"finalized\", \"latest\", or \"ignore\" to ignore this check)")
	f.Duration(prefix+".l1-block-bound-bypass", DefaultBatchPosterConfig.L1BlockBoundBypass, "post batches even if not within the layer 1 future bounds if we're within this margin of the max delay")
	f.Bool(prefix+".use-access-lists", DefaultBatchPosterConfig.UseAccessLists, "post batches with access lists to reduce gas usage (disabled for L3s)")
	f.Uint64(prefix+".gas-estimate-base-fee-multiple-bips", uint64(DefaultBatchPosterConfig.GasEstimateBaseFeeMultipleBips), "for gas estimation, use this multiple of the basefee (measured in basis points) as the max fee per gas")
	f.Duration(prefix+".reorg-resistance-margin", DefaultBatchPosterConfig.ReorgResistanceMargin, "do not post batch if its within this duration from layer 1 minimum bounds. Requires l1-block-bound option not be set to \"ignore\"")
	f.Bool(prefix+".check-batch-correctness", DefaultBatchPosterConfig.CheckBatchCorrectness, "setting this to true will run the batch against an inbox multiplexer and verifies that it produces the correct set of messages")
	f.Duration(prefix+".max-empty-batch-delay", DefaultBatchPosterConfig.MaxEmptyBatchDelay, "maximum empty batch posting delay, batch poster will only be able to post an empty batch if this time period building a batch has passed; if 0, disable automatic empty batch posting")
	f.Uint64(prefix+".delay-buffer-threshold-margin", DefaultBatchPosterConfig.DelayBufferThresholdMargin, "the number of blocks to post the batch before reaching the delay buffer threshold")
	f.String(prefix+".parent-chain-eip7623", DefaultBatchPosterConfig.ParentChainEip7623, "if parent chain uses EIP7623 (\"yes\", \"no\", \"auto\")")
	f.Bool(prefix+".delay-buffer-always-updatable", DefaultBatchPosterConfig.DelayBufferAlwaysUpdatable, "always treat delay buffer as updatable")
	redislock.AddConfigOptions(prefix+".redis-lock", f)
	dataposter.DataPosterConfigAddOptions(prefix+".data-poster", f, dataposter.DefaultDataPosterConfig, dataposter.DataPosterUsageBatchPoster)
	genericconf.WalletConfigAddOptions(prefix+".parent-chain-wallet", f, DefaultBatchPosterConfig.ParentChainWallet.Pathname)
	DangerousBatchPosterConfigAddOptions(prefix+".dangerous", f)
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	Enable:                             false,
	DisableDapFallbackStoreDataOnChain: false,
	EthDAFallbackBatchCount:            10,
	MaxSize:                            0, // Deprecated
	// This default is overridden for L3 chains in applyChainParameters in cmd/nitro/nitro.go
	MaxCalldataBatchSize: 100000,
	// The Max4844BatchSize should be calculated from the values from L1 chain configs
	// using the eip4844 utility package from go-ethereum.
	// The default value of 0 causes the batch poster to use the value from go-ethereum.
	Max4844BatchSize:               0,
	PollInterval:                   time.Second * 10,
	ErrorDelay:                     time.Second * 10,
	MaxDelay:                       time.Hour,
	WaitForMaxDelay:                false,
	CompressionLevel:               brotli.BestCompression,
	DASRetentionPeriod:             daprovider.DefaultDASRetentionPeriod,
	GasRefunderAddress:             "",
	ExtraBatchGas:                  50_000,
	Post4844Blobs:                  false,
	IgnoreBlobPrice:                false,
	DataPoster:                     dataposter.DefaultDataPosterConfig,
	ParentChainWallet:              DefaultBatchPosterL1WalletConfig,
	L1BlockBound:                   "",
	L1BlockBoundBypass:             time.Hour,
	UseAccessLists:                 true,
	RedisLock:                      redislock.DefaultCfg,
	GasEstimateBaseFeeMultipleBips: arbmath.OneInUBips * 3 / 2,
	ReorgResistanceMargin:          10 * time.Minute,
	CheckBatchCorrectness:          true,
	MaxEmptyBatchDelay:             3 * 24 * time.Hour,
	DelayBufferThresholdMargin:     25, // 5 minutes considering 12-second blocks
	DelayBufferAlwaysUpdatable:     true,
	ParentChainEip7623:             "auto",
}

var DefaultBatchPosterL1WalletConfig = genericconf.WalletConfig{
	Pathname:      "batch-poster-wallet",
	Password:      genericconf.WalletConfigDefault.Password,
	PrivateKey:    genericconf.WalletConfigDefault.PrivateKey,
	Account:       genericconf.WalletConfigDefault.Account,
	OnlyCreateKey: genericconf.WalletConfigDefault.OnlyCreateKey,
}

var TestBatchPosterConfig = BatchPosterConfig{
	Enable:                             true,
	DisableDapFallbackStoreDataOnChain: true,
	EthDAFallbackBatchCount:            1,
	MaxCalldataBatchSize:               100000,
	Max4844BatchSize:                   DefaultBatchPosterConfig.Max4844BatchSize,
	PollInterval:                       time.Millisecond * 10,
	ErrorDelay:                         time.Millisecond * 10,
	MaxDelay:                           0,
	WaitForMaxDelay:                    false,
	CompressionLevel:                   2,
	DASRetentionPeriod:                 daprovider.DefaultDASRetentionPeriod,
	GasRefunderAddress:                 "",
	ExtraBatchGas:                      10_000,
	Post4844Blobs:                      false,
	IgnoreBlobPrice:                    false,
	DataPoster:                         dataposter.TestDataPosterConfig,
	ParentChainWallet:                  DefaultBatchPosterL1WalletConfig,
	L1BlockBound:                       "",
	L1BlockBoundBypass:                 time.Hour,
	UseAccessLists:                     true,
	RedisLock:                          redislock.TestCfg,
	GasEstimateBaseFeeMultipleBips:     arbmath.OneInUBips * 3 / 2,
	CheckBatchCorrectness:              true,
	DelayBufferThresholdMargin:         0,
	DelayBufferAlwaysUpdatable:         true,
	ParentChainEip7623:                 "auto",
}

type BatchPosterOpts struct {
	DataPosterDB  ethdb.Database
	L1Reader      *headerreader.HeaderReader
	Inbox         *InboxTracker
	Streamer      *TransactionStreamer
	VersionGetter execution.ArbOSVersionGetter
	SyncMonitor   *SyncMonitor
	Config        BatchPosterConfigFetcher
	DeployInfo    *chaininfo.RollupAddresses
	TransactOpts  *bind.TransactOpts
	DAPWriters    []daprovider.Writer
	ParentChainID *big.Int
	DAPReaders    *daprovider.DAProviderRegistry
}

func NewBatchPoster(ctx context.Context, opts *BatchPosterOpts) (*BatchPoster, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(opts.DeployInfo.SequencerInbox, opts.L1Reader.Client())
	if err != nil {
		return nil, err
	}

	if err = opts.Config().Validate(); err != nil {
		return nil, err
	}
	var checkEip7623 bool
	var useEip7623 bool
	switch opts.Config().ParentChainEip7623 {
	case "no":
		checkEip7623 = false
		useEip7623 = false
	case "yes":
		checkEip7623 = false
		useEip7623 = true
	case "auto":
		checkEip7623 = true
		useEip7623 = false
	}
	seqInboxABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	redisClient, err := redisutil.RedisClientFromURL(opts.Config().RedisUrl)
	if err != nil {
		return nil, err
	}
	redisLockConfigFetcher := func() *redislock.SimpleCfg {
		simpleRedisLockConfig := opts.Config().RedisLock
		simpleRedisLockConfig.Key = batchPosterSimpleRedisLockKey
		return &simpleRedisLockConfig
	}
	redisLock, err := redislock.NewSimple(redisClient, redisLockConfigFetcher, func() bool { return opts.SyncMonitor.Synced() })
	if err != nil {
		return nil, err
	}
	b := &BatchPoster{
		l1Reader:           opts.L1Reader,
		inbox:              opts.Inbox,
		streamer:           opts.Streamer,
		arbOSVersionGetter: opts.VersionGetter,
		syncMonitor:        opts.SyncMonitor,
		config:             opts.Config,
		seqInbox:           seqInbox,
		seqInboxABI:        seqInboxABI,
		seqInboxAddr:       opts.DeployInfo.SequencerInbox,
		gasRefunderAddr:    opts.Config().gasRefunder,
		bridgeAddr:         opts.DeployInfo.Bridge,
		dapWriters:         opts.DAPWriters,
		redisLock:          redisLock,
		dapReaders:         opts.DAPReaders,
		parentChain:        &parent.ParentChain{ChainID: opts.ParentChainID, L1Reader: opts.L1Reader},
		checkEip7623:       checkEip7623,
		useEip7623:         useEip7623,
	}
	b.messagesPerBatch, err = arbmath.NewMovingAverage[uint64](20)
	if err != nil {
		return nil, err
	}
	dataPosterConfigFetcher := func() *dataposter.DataPosterConfig {
		dpCfg := opts.Config().DataPoster
		dpCfg.Post4844Blobs = opts.Config().Post4844Blobs
		return &dpCfg
	}
	b.dataPoster, err = dataposter.NewDataPoster(ctx,
		&dataposter.DataPosterOpts{
			Database:          opts.DataPosterDB,
			HeaderReader:      opts.L1Reader,
			Auth:              opts.TransactOpts,
			RedisClient:       redisClient,
			Config:            dataPosterConfigFetcher,
			MetadataRetriever: b.getBatchPosterPosition,
			ExtraBacklog:      b.GetBacklogEstimate,
			RedisKey:          "data-poster.queue",
			ParentChainID:     opts.ParentChainID,
		})
	if err != nil {
		return nil, err
	}
	// Dataposter sender may be external signer address, so we should initialize
	// access list after initializing dataposter.
	b.accessList = func(SequencerInboxAccs, AfterDelayedMessagesRead uint64) types.AccessList {
		if !b.config().UseAccessLists || opts.L1Reader.IsParentChainArbitrum() {
			// Access lists cost gas instead of saving gas when posting to L2s,
			// because data is expensive in comparison to computation.
			return nil
		}
		return AccessList(&AccessListOpts{
			SequencerInboxAddr:       opts.DeployInfo.SequencerInbox,
			DataPosterAddr:           b.dataPoster.Sender(),
			BridgeAddr:               opts.DeployInfo.Bridge,
			GasRefunderAddr:          opts.Config().gasRefunder,
			SequencerInboxAccs:       SequencerInboxAccs,
			AfterDelayedMessagesRead: AfterDelayedMessagesRead,
		})
	}
	return b, nil
}

type simulatedBlobReader struct {
	blobs []kzg4844.Blob
}

func (b *simulatedBlobReader) GetBlobs(ctx context.Context, batchBlockHash common.Hash, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	return b.blobs, nil
}

func (b *simulatedBlobReader) Initialize(ctx context.Context) error { return nil }

type simulatedMuxBackend struct {
	batchSeqNum           uint64
	positionWithinMessage uint64
	seqMsg                []byte
	allMsgs               map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata
	delayedInboxStart     uint64
	delayedInbox          []*arbostypes.MessageWithMetadata
}

func (b *simulatedMuxBackend) PeekSequencerInbox() ([]byte, common.Hash, error) {
	return b.seqMsg, common.Hash{}, nil
}

func (b *simulatedMuxBackend) GetSequencerInboxPosition() uint64   { return b.batchSeqNum }
func (b *simulatedMuxBackend) AdvanceSequencerInbox()              {}
func (b *simulatedMuxBackend) GetPositionWithinMessage() uint64    { return b.positionWithinMessage }
func (b *simulatedMuxBackend) SetPositionWithinMessage(pos uint64) { b.positionWithinMessage = pos }

func (b *simulatedMuxBackend) ReadDelayedInbox(seqNum uint64) (*arbostypes.L1IncomingMessage, error) {
	pos := arbmath.SaturatingUSub(seqNum, b.delayedInboxStart)
	if pos < uint64(len(b.delayedInbox)) {
		return b.delayedInbox[pos].Message, nil
	}
	return nil, fmt.Errorf("error serving ReadDelayedInbox, all delayed messages were read. Requested delayed message position:%d, Total delayed messages: %d", pos, len(b.delayedInbox))
}

type AccessListOpts struct {
	SequencerInboxAddr       common.Address
	BridgeAddr               common.Address
	DataPosterAddr           common.Address
	GasRefunderAddr          common.Address
	SequencerInboxAccs       uint64
	AfterDelayedMessagesRead uint64
}

// AccessList returns access list (contracts, storage slots) for batchposter.
func AccessList(opts *AccessListOpts) types.AccessList {
	l := types.AccessList{
		types.AccessTuple{
			Address: opts.SequencerInboxAddr,
			StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), // totalDelayedMessagesRead
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), // bridge
				common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000a"), // maxTimeVariation
				// ADMIN_SLOT from OpenZeppelin, keccak-256 hash of
				// "eip1967.proxy.admin" subtracted by 1.
				common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"),
				// IMPLEMENTATION_SLOT from OpenZeppelin,  keccak-256 hash
				// of "eip1967.proxy.implementation" subtracted by 1.
				common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"),
				// isBatchPoster[batchPosterAddr]; for mainnnet it's: "0xa10aa54071443520884ed767b0684edf43acec528b7da83ab38ce60126562660".
				common.Hash(arbutil.PaddedKeccak256(opts.DataPosterAddr.Bytes(), []byte{3})),
			},
		},
		types.AccessTuple{
			Address: opts.BridgeAddr,
			StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000006"), // delayedInboxAccs.length
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000007"), // sequencerInboxAccs.length
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000009"), // sequencerInbox
				common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000a"), // sequencerReportedSubMessageCount
				// ADMIN_SLOT from OpenZeppelin, keccak-256 hash of
				// "eip1967.proxy.admin" subtracted by 1.
				common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"),
				// IMPLEMENTATION_SLOT from OpenZeppelin,  keccak-256 hash
				// of "eip1967.proxy.implementation" subtracted by 1.
				common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"),
				// These below may change when transaction is actually executed:
				// - delayedInboxAccs[delayedInboxAccs.length - 1]
				// - delayedInboxAccs.push(...);
			},
		},
	}

	for _, v := range []struct{ slotIdx, val uint64 }{
		{7, opts.SequencerInboxAccs - 1},       // - sequencerInboxAccs[sequencerInboxAccs.length - 1]; (keccak256(7, sequencerInboxAccs.length - 1))
		{7, opts.SequencerInboxAccs},           // - sequencerInboxAccs.push(...); (keccak256(7, sequencerInboxAccs.length))
		{6, opts.AfterDelayedMessagesRead - 1}, // - delayedInboxAccs[afterDelayedMessagesRead - 1]; (keccak256(6, afterDelayedMessagesRead - 1))
	} {
		sb := arbutil.SumBytes(arbutil.PaddedKeccak256([]byte{byte(v.slotIdx)}), new(big.Int).SetUint64(v.val).Bytes())
		l[1].StorageKeys = append(l[1].StorageKeys, common.Hash(sb))
	}

	if (opts.GasRefunderAddr != common.Address{}) {
		l = append(l, types.AccessTuple{
			Address: opts.GasRefunderAddr,
			StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"), // CommonParameters.{maxRefundeeBalance, extraGasMargin, calldataCost, maxGasTip}
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000005"), // CommonParameters.{maxGasCost, maxSingleGasUsage}
				// allowedContracts[msg.sender]; for mainnet it's: "0x7686888b19bb7b75e46bb1aa328b65150743f4899443d722f0adf8e252ccda41".
				common.Hash(arbutil.PaddedKeccak256(opts.SequencerInboxAddr.Bytes(), []byte{1})),
				// allowedRefundees[refundee]; for mainnet it's: "0xe85fd79f89ff278fc57d40aecb7947873df9f0beac531c8f71a98f630e1eab62".
				common.Hash(arbutil.PaddedKeccak256(opts.DataPosterAddr.Bytes(), []byte{2})),
			},
		})
	}
	return l
}

type txInfo struct {
	Hash      common.Hash       `json:"hash"`
	Nonce     hexutil.Uint64    `json:"nonce"`
	From      common.Address    `json:"from"`
	To        *common.Address   `json:"to"`
	Gas       hexutil.Uint64    `json:"gas"`
	GasPrice  *hexutil.Big      `json:"gasPrice"`
	GasFeeCap *hexutil.Big      `json:"maxFeePerGas,omitempty"`
	GasTipCap *hexutil.Big      `json:"maxPriorityFeePerGas,omitempty"`
	Input     hexutil.Bytes     `json:"input"`
	Value     *hexutil.Big      `json:"value"`
	Accesses  *types.AccessList `json:"accessList,omitempty"`
}

func (b *BatchPoster) ParentChainIsUsingEIP7623(ctx context.Context, latestHeader *types.Header) (bool, error) {
	// Before EIP-7623 tx.gasUsed is defined as:
	// tx.gasUsed = (
	//     21000
	//     + STANDARD_TOKEN_COST * tokens_in_calldata
	//     + execution_gas_used
	//     + isContractCreation * (32000 + INITCODE_WORD_COST * words(calldata))
	// )
	//
	// With EIP-7623 tx.gasUsed is defined as:
	// tx.gasUsed = (
	//     21000
	//     +
	//     max(
	//         STANDARD_TOKEN_COST * tokens_in_calldata
	//         + execution_gas_used
	//         + isContractCreation * (32000 + INITCODE_WORD_COST * words(calldata)),
	//         TOTAL_COST_FLOOR_PER_TOKEN * tokens_in_calldata
	//     )
	// )
	//
	// STANDARD_TOKEN_COST = 4
	// TOTAL_COST_FLOOR_PER_TOKEN = 10
	//
	// To infer whether the parent chain is using EIP-7623 we estimate gas usage of two parent chain native token transfer transactions,
	// that then have equal execution_gas_used.
	// Also, in both transactions isContractCreation is zero, and tokens_in_calldata is big enough so
	// (TOTAL_COST_FLOOR_PER_TOKEN * tokens_in_calldata > STANDARD_TOKEN_COST * tokens_in_calldata + execution_gas_used).
	// Also, the used calldatas only have non-zero bytes, so tokens_in_calldata is defined as length(calldata) * 4.
	//
	// The difference between the transactions is:
	// length(calldata_tx_2) == length(calldata_tx_1) + 1
	//
	// So, if parent chain is not running EIP-7623:
	// tx_2.gasUsed - tx_1.gasUsed =
	// STANDARD_TOKEN_COST * 4 * (length(calldata_tx_2) - length(calldata_tx_1)) =
	// 16
	//
	// And if the parent chain is running EIP-7623:
	// tx_2.gasUsed - tx_1.gasUsed =
	// TOTAL_COST_FLOOR_PER_TOKEN * 4 * (length(calldata_tx_2) - length(calldata_tx_1)) =
	// 40

	if !b.checkEip7623 {
		return b.useEip7623, nil
	}
	rpcClient := b.l1Reader.Client()
	config := b.config()
	to := b.dataPoster.Sender()

	data := []byte{}
	for i := 0; i < 100_000; i++ {
		data = append(data, 1)
	}

	// Rather than checking the latest block, we're going to check a recent
	// block (5 blocks back) to avoid reorgs.
	targetBlockNumber := latestHeader.Number.Sub(latestHeader.Number, big.NewInt(5))
	targetHeader, err := rpcClient.HeaderByNumber(ctx, targetBlockNumber)
	if err != nil {
		return false, err
	}
	maxFeePerGas := arbmath.BigMulByUBips(targetHeader.BaseFee, config.GasEstimateBaseFeeMultipleBips)
	blockHex := hexutil.Uint64(targetBlockNumber.Uint64()).String()

	gasParams := estimateGasParams{
		From:         b.dataPoster.Sender(),
		To:           &to,
		Data:         data,
		MaxFeePerGas: (*hexutil.Big)(maxFeePerGas),
	}

	gas1, err := estimateGas(rpcClient.Client(), ctx, gasParams, blockHex)
	if err != nil {
		log.Warn("Failed to estimate gas for EIP-7623 check 1", "err", err)
		return false, err
	}

	gasParams.Data = append(gasParams.Data, 1)
	gas2, err := estimateGas(rpcClient.Client(), ctx, gasParams, blockHex)
	if err != nil {
		log.Warn("Failed to estimate gas for EIP-7623 check 2", "err", err)
		return false, err
	}

	// Takes into consideration that eth_estimateGas is an approximation.
	// As an example, go-ethereum can only return an estimate that is equal
	// or bigger than the true estimate, and currently defines the allowed error ratio as 0.015
	var parentChainIsUsingEIP7623 bool
	diffIsClose := func(gas1, gas2, lowerTargetDiff, upperTargetDiff uint64) bool {
		diff := gas2 - gas1
		return diff >= lowerTargetDiff && diff <= upperTargetDiff
	}
	if diffIsClose(gas1, gas2, 14, 18) {
		// targetDiff is 16
		parentChainIsUsingEIP7623 = false
	} else if diffIsClose(gas1, gas2, 36, 44) {
		// targetDiff is 40
		parentChainIsUsingEIP7623 = true
	} else {
		return false, fmt.Errorf("unexpected gas difference, gas1: %d, gas2: %d", gas1, gas2)
	}
	b.useEip7623 = parentChainIsUsingEIP7623
	if parentChainIsUsingEIP7623 {
		// Once the parent chain is using EIP-7623, we don't need to check it again.
		b.checkEip7623 = false
	}
	return parentChainIsUsingEIP7623, nil
}

// getTxsInfoByBlock fetches all the transactions inside block of id 'number' using json rpc
// and returns an array of txInfo which has fields that are necessary in checking for batch reverts
func (b *BatchPoster) getTxsInfoByBlock(ctx context.Context, number int64) ([]txInfo, error) {
	blockNrStr := rpc.BlockNumber(number).String()
	rawRpcClient := b.l1Reader.Client().Client()
	var blk struct {
		Transactions []txInfo `json:"transactions"`
	}
	err := rawRpcClient.CallContext(ctx, &blk, "eth_getBlockByNumber", blockNrStr, true)
	if err != nil {
		return nil, fmt.Errorf("error fetching block %d : %w", number, err)
	}
	return blk.Transactions, nil
}

// checkReverts checks blocks with number in range [from, to] whether they
// contain reverted batch_poster transaction.
// It returns true if it finds batch posting needs to halt, which is true if a batch reverts
// unless the data poster is configured with noop storage which can tolerate reverts.
func (b *BatchPoster) checkReverts(ctx context.Context, to int64) (bool, error) {
	if b.nextRevertCheckBlock > to {
		return false, fmt.Errorf("wrong range, from: %d > to: %d", b.nextRevertCheckBlock, to)
	}
	for ; b.nextRevertCheckBlock <= to; b.nextRevertCheckBlock++ {
		txs, err := b.getTxsInfoByBlock(ctx, b.nextRevertCheckBlock)
		if err != nil {
			return false, fmt.Errorf("error getting transactions data of block %d: %w", b.nextRevertCheckBlock, err)
		}
		for _, tx := range txs {
			if tx.From == b.dataPoster.Sender() {
				r, err := b.l1Reader.Client().TransactionReceipt(ctx, tx.Hash)
				if err != nil {
					return false, fmt.Errorf("getting a receipt for transaction: %v, %w", tx.Hash, err)
				}
				if r.Status == types.ReceiptStatusFailed {
					shouldHalt := !b.dataPoster.UsingNoOpStorage()
					logLevel := log.Warn
					if shouldHalt {
						logLevel = log.Error
					}
					al := types.AccessList{}
					if tx.Accesses != nil {
						al = *tx.Accesses
					}
					txErr := arbutil.DetailTxErrorUsingCallMsg(ctx, b.l1Reader.Client(), tx.Hash, r, ethereum.CallMsg{
						From:       tx.From,
						To:         tx.To,
						Gas:        uint64(tx.Gas),
						GasPrice:   tx.GasPrice.ToInt(),
						GasFeeCap:  tx.GasFeeCap.ToInt(),
						GasTipCap:  tx.GasTipCap.ToInt(),
						Value:      tx.Value.ToInt(),
						Data:       tx.Input,
						AccessList: al,
					})
					logLevel("Transaction from batch poster reverted", "nonce", tx.Nonce, "txHash", tx.Hash, "blockNumber", r.BlockNumber, "blockHash", r.BlockHash, "txErr", txErr)
					return shouldHalt, nil
				}
			}
		}
	}
	return false, nil
}

func (b *BatchPoster) pollForL1PriceData(ctx context.Context) {
	headerCh, unsubscribe := b.l1Reader.Subscribe(false)
	defer unsubscribe()

	if b.config().Post4844Blobs {
		results, err := b.parentChain.MaxBlobGasPerBlock(ctx, nil)
		if err != nil {
			log.Error("Error getting max blob gas per block", "err", err)
		}
		// #nosec G115
		blobGasLimitGauge.Update(int64(results))
	}

	for {
		select {
		case h, ok := <-headerCh:
			if !ok {
				log.Info("L1 headers channel checking for l1 price data has been closed")
				return
			}
			baseFeeGauge.Update(h.BaseFee.Int64())
			l1GasPrice := h.BaseFee.Uint64()
			if b.config().Post4844Blobs && h.BlobGasUsed != nil {
				if h.ExcessBlobGas != nil {
					blobFeePerByte, err := b.parentChain.BlobFeePerByte(ctx, h)
					if err != nil {
						log.Error("Error getting blob fee per byte", "err", err)
						continue
					}
					blobFeePerByte.Mul(blobFeePerByte, blobTxBlobGasPerBlob)
					blobFeePerByte.Div(blobFeePerByte, usableBytesInBlob)
					blobFeeGauge.Update(blobFeePerByte.Int64())
					if l1GasPrice > blobFeePerByte.Uint64()/16 {
						l1GasPrice = blobFeePerByte.Uint64() / 16
					}
				}
				// #nosec G115
				blobGasUsedGauge.Update(int64(*h.BlobGasUsed))
			}
			// #nosec G115
			blockGasUsedGauge.Update(int64(h.GasUsed))
			// #nosec G115
			blockGasLimitGauge.Update(int64(h.GasLimit))
			suggestedTipCap, err := b.l1Reader.Client().SuggestGasTipCap(ctx)
			if err != nil {
				log.Warn("unable to fetch suggestedTipCap from l1 client to update arb/batchposter/suggestedtipcap metric", "err", err)
			} else {
				suggestedTipCapGauge.Update(suggestedTipCap.Int64())
			}
			// #nosec G115
			l1GasPriceGauge.Update(int64(l1GasPrice))
		case <-ctx.Done():
			return
		}
	}
}

// pollForReverts runs a gouroutine that listens to l1 block headers, checks
// if any transaction made by batch poster was reverted.
func (b *BatchPoster) pollForReverts(ctx context.Context) {
	headerCh, unsubscribe := b.l1Reader.Subscribe(false)
	defer unsubscribe()

	for {
		// Poll until:
		// - L1 headers reader channel is closed, or
		// - polling is through context, or
		// - we see a transaction in the block from dataposter that was reverted.
		select {
		case h, ok := <-headerCh:
			if !ok {
				log.Info("L1 headers channel checking for batch poster reverts has been closed")
				return
			}
			blockNum := h.Number.Int64()
			// If this is the first block header, set last seen as number-1.
			// We may see same block number again if there is L1 reorg, in that
			// case we check the block again.
			if b.nextRevertCheckBlock == 0 || b.nextRevertCheckBlock > blockNum {
				b.nextRevertCheckBlock = blockNum
			}
			if blockNum-b.nextRevertCheckBlock > 100 {
				log.Warn("Large gap between last seen and current block number, skipping check for reverts", "last", b.nextRevertCheckBlock, "current", blockNum)
				b.nextRevertCheckBlock = blockNum
				continue
			}

			reverted, err := b.checkReverts(ctx, blockNum)
			if err != nil {
				logLevel := log.Warn
				if strings.Contains(err.Error(), "not found") {
					// Just parent chain node inconsistency
					// One node sent us a block, but another didn't have it
					// We'll try to check this block again next loop
					logLevel = log.Debug
				}
				logLevel("Error checking batch reverts", "err", err)
				continue
			}
			if reverted {
				b.batchReverted.Store(true)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *BatchPoster) getBatchPosterPosition(ctx context.Context, blockNum *big.Int) ([]byte, error) {
	bigInboxBatchCount, err := b.seqInbox.BatchCount(&bind.CallOpts{Context: ctx, BlockNumber: blockNum})
	if err != nil {
		return nil, fmt.Errorf("error getting latest batch count: %w", err)
	}
	inboxBatchCount := bigInboxBatchCount.Uint64()
	var prevBatchMeta BatchMetadata
	if inboxBatchCount > 0 {
		var err error
		prevBatchMeta, err = b.inbox.GetBatchMetadata(inboxBatchCount - 1)
		if err != nil {
			return nil, fmt.Errorf("error getting latest batch metadata: %w", err)
		}
	}
	return rlp.EncodeToBytes(batchPosterPosition{
		MessageCount:        prevBatchMeta.MessageCount,
		DelayedMessageCount: prevBatchMeta.DelayedMessageCount,
		NextSeqNum:          inboxBatchCount,
	})
}

var errBatchAlreadyClosed = errors.New("batch segments already closed")

type batchSegments struct {
	compressedBuffer      *bytes.Buffer
	compressedWriter      *brotli.Writer
	rawSegments           [][]byte
	timestamp             uint64
	blockNum              uint64
	delayedMsg            uint64
	sizeLimit             int
	recompressionLevel    int
	newUncompressedSize   int
	totalUncompressedSize int
	lastCompressedSize    int
	trailingHeaders       int // how many trailing segments are headers
	isDone                bool
}

type buildingBatch struct {
	segments           *batchSegments
	startMsgCount      arbutil.MessageIndex
	msgCount           arbutil.MessageIndex
	haveUsefulMessage  bool
	use4844            bool
	muxBackend         *simulatedMuxBackend
	firstDelayedMsg    *arbostypes.MessageWithMetadata
	firstNonDelayedMsg *arbostypes.MessageWithMetadata
	firstUsefulMsg     *arbostypes.MessageWithMetadata
}

func (b *BatchPoster) newBatchSegments(ctx context.Context, firstDelayed uint64, use4844 bool, usingAltDA bool) (*batchSegments, error) {
	config := b.config()
	var maxSize int

	if use4844 {
		// Building 4844 blobs for EthDA
		if config.Max4844BatchSize != 0 {
			maxSize = config.Max4844BatchSize
		} else {
			maxBlobGasPerBlock, err := b.parentChain.MaxBlobGasPerBlock(ctx, nil)
			if err != nil {
				return nil, err
			}
			// Try to fill under half of the parent chain's max blobs.
			// #nosec G115
			maxSize = blobs.BlobEncodableData*(int(maxBlobGasPerBlock)/params.BlobTxBlobGasPerBlob)/2 - blobBatchEncodingOverhead
		}
	} else if usingAltDA {
		// Query the currently selected DA writer to get its max batch size
		if len(b.dapWriters) == 0 {
			return nil, fmt.Errorf("using AltDA but no DA writers configured")
		}
		if b.currentWriterIndex >= len(b.dapWriters) {
			return nil, fmt.Errorf("currentWriterIndex %d exceeds number of writers %d", b.currentWriterIndex, len(b.dapWriters))
		}
		writerMaxSize, err := b.dapWriters[b.currentWriterIndex].GetMaxMessageSize().Await(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get max message size from DA writer %d: %w", b.currentWriterIndex, err)
		}
		if writerMaxSize <= 0 {
			return nil, fmt.Errorf("DA writer %d returned invalid max message size: %d", b.currentWriterIndex, writerMaxSize)
		}
		maxSize = writerMaxSize
	} else {
		// Using calldata for EthDA
		maxSize = config.MaxCalldataBatchSize
		if maxSize <= SequencerMessageHeaderSize {
			return nil, fmt.Errorf("maximum calldata batch size too small: %d", maxSize)
		}
		maxSize -= SequencerMessageHeaderSize
	}
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, maxSize*2))
	compressionLevel := b.config().CompressionLevel
	recompressionLevel := b.config().CompressionLevel
	if b.GetBacklogEstimate() > 20 {
		compressionLevel = arbmath.MinInt(compressionLevel, brotli.DefaultCompression)
	}
	if b.GetBacklogEstimate() > 40 {
		recompressionLevel = arbmath.MinInt(recompressionLevel, brotli.DefaultCompression)
	}
	if b.GetBacklogEstimate() > 60 {
		compressionLevel = arbmath.MinInt(compressionLevel, 4)
	}
	if recompressionLevel < compressionLevel {
		// This should never be possible
		log.Warn(
			"somehow the recompression level was lower than the compression level",
			"recompressionLevel", recompressionLevel,
			"compressionLevel", compressionLevel,
		)
		recompressionLevel = compressionLevel
	}
	return &batchSegments{
		compressedBuffer:   compressedBuffer,
		compressedWriter:   brotli.NewWriterLevel(compressedBuffer, compressionLevel),
		sizeLimit:          maxSize,
		recompressionLevel: recompressionLevel,
		rawSegments:        make([][]byte, 0, 128),
		delayedMsg:         firstDelayed,
	}, nil
}

func (s *batchSegments) recompressAll() error {
	s.compressedBuffer = bytes.NewBuffer(make([]byte, 0, s.sizeLimit*2))
	s.compressedWriter = brotli.NewWriterLevel(s.compressedBuffer, s.recompressionLevel)
	s.newUncompressedSize = 0
	s.totalUncompressedSize = 0
	for _, segment := range s.rawSegments {
		err := s.addSegmentToCompressed(segment)
		if err != nil {
			return err
		}
	}
	if s.totalUncompressedSize > arbstate.MaxDecompressedLen {
		return fmt.Errorf("batch size %v exceeds maximum decompressed length %v", s.totalUncompressedSize, arbstate.MaxDecompressedLen)
	}
	if len(s.rawSegments) >= arbstate.MaxSegmentsPerSequencerMessage {
		return fmt.Errorf("number of raw segments %v excees maximum number %v", len(s.rawSegments), arbstate.MaxSegmentsPerSequencerMessage)
	}
	return nil
}

func (s *batchSegments) testForOverflow(isHeader bool) (bool, error) {
	// we've reached the max decompressed size
	if s.totalUncompressedSize > arbstate.MaxDecompressedLen {
		log.Info("Batch full: max decompressed length exceeded",
			"current", s.totalUncompressedSize,
			"max", arbstate.MaxDecompressedLen,
			"isHeader", isHeader)
		return true, nil
	}
	// we've reached the max number of segments
	if len(s.rawSegments) >= arbstate.MaxSegmentsPerSequencerMessage {
		log.Info("Batch overflow: max segments exceeded",
			"segments", len(s.rawSegments),
			"max", arbstate.MaxSegmentsPerSequencerMessage,
			"isHeader", isHeader)
		return true, nil
	}
	// there is room, no need to flush
	if (s.lastCompressedSize + s.newUncompressedSize) < s.sizeLimit {
		return false, nil
	}
	// don't want to flush for headers or the first message
	if isHeader || len(s.rawSegments) == s.trailingHeaders {
		return false, nil
	}
	err := s.compressedWriter.Flush()
	if err != nil {
		return true, err
	}
	s.lastCompressedSize = s.compressedBuffer.Len()
	s.newUncompressedSize = 0
	if s.lastCompressedSize >= s.sizeLimit {
		log.Info("Batch overflow: compressed size limit exceeded",
			"compressedSize", s.lastCompressedSize,
			"limit", s.sizeLimit,
			"isHeader", isHeader)
		return true, nil
	}
	return false, nil
}

func (s *batchSegments) close() error {
	s.rawSegments = s.rawSegments[:len(s.rawSegments)-s.trailingHeaders]
	s.trailingHeaders = 0
	err := s.recompressAll()
	if err != nil {
		return err
	}
	s.isDone = true
	return nil
}

func (s *batchSegments) addSegmentToCompressed(segment []byte) error {
	encoded, err := rlp.EncodeToBytes(segment)
	if err != nil {
		return err
	}
	lenWritten, err := s.compressedWriter.Write(encoded)
	s.newUncompressedSize += lenWritten
	s.totalUncompressedSize += lenWritten
	return err
}

// returns false if segment was too large, error in case of real error
func (s *batchSegments) addSegment(segment []byte, isHeader bool) (bool, error) {
	if s.isDone {
		return false, errBatchAlreadyClosed
	}
	err := s.addSegmentToCompressed(segment)
	if err != nil {
		return false, err
	}
	// Force include headers because we don't want to re-compress and we can just trim them later if necessary
	overflow, err := s.testForOverflow(isHeader)
	if err != nil {
		return false, err
	}
	if overflow {
		return false, s.close()
	}
	s.rawSegments = append(s.rawSegments, segment)
	if isHeader {
		s.trailingHeaders++
	} else {
		s.trailingHeaders = 0
	}
	return true, nil
}

func (s *batchSegments) addL2Msg(l2msg []byte) (bool, error) {
	segment := make([]byte, 1, len(l2msg)+1)
	segment[0] = arbstate.BatchSegmentKindL2Message
	segment = append(segment, l2msg...)
	return s.addSegment(segment, false)
}

func (s *batchSegments) prepareIntSegment(val uint64, segmentHeader byte) ([]byte, error) {
	segment := make([]byte, 1, 16)
	segment[0] = segmentHeader
	enc, err := rlp.EncodeToBytes(val)
	if err != nil {
		return nil, err
	}
	return append(segment, enc...), nil
}

func (s *batchSegments) maybeAddDiffSegment(base *uint64, newVal uint64, segmentHeader byte) (bool, error) {
	if newVal == *base {
		return true, nil
	}
	diff := newVal - *base
	seg, err := s.prepareIntSegment(diff, segmentHeader)
	if err != nil {
		return false, err
	}
	success, err := s.addSegment(seg, true)
	if success {
		*base = newVal
	}
	return success, err
}

func (s *batchSegments) addDelayedMessage() (bool, error) {
	segment := []byte{arbstate.BatchSegmentKindDelayedMessages}
	success, err := s.addSegment(segment, false)
	if (err == nil) && success {
		s.delayedMsg += 1
	}
	return success, err
}

func (s *batchSegments) AddMessage(msg *arbostypes.MessageWithMetadata) (bool, error) {
	if s.isDone {
		return false, errBatchAlreadyClosed
	}
	if msg.DelayedMessagesRead > s.delayedMsg {
		if msg.DelayedMessagesRead != s.delayedMsg+1 {
			return false, fmt.Errorf("attempted to add delayed msg %d after %d", msg.DelayedMessagesRead, s.delayedMsg)
		}
		return s.addDelayedMessage()
	}
	success, err := s.maybeAddDiffSegment(&s.timestamp, msg.Message.Header.Timestamp, arbstate.BatchSegmentKindAdvanceTimestamp)
	if !success {
		return false, err
	}
	success, err = s.maybeAddDiffSegment(&s.blockNum, msg.Message.Header.BlockNumber, arbstate.BatchSegmentKindAdvanceL1BlockNumber)
	if !success {
		return false, err
	}
	return s.addL2Msg(msg.Message.L2msg)
}

func (s *batchSegments) IsDone() bool {
	return s.isDone
}

// Returns nil (as opposed to []byte{}) if there's no segments to put in the batch
func (s *batchSegments) CloseAndGetBytes() ([]byte, error) {
	if !s.isDone {
		err := s.close()
		if err != nil {
			return nil, err
		}
	}
	if len(s.rawSegments) == 0 {
		return nil, nil
	}
	err := s.compressedWriter.Close()
	if err != nil {
		return nil, err
	}
	compressedBytes := s.compressedBuffer.Bytes()
	fullMsg := make([]byte, 1, len(compressedBytes)+1)

	fullMsg[0] = daprovider.BrotliMessageHeaderByte

	fullMsg = append(fullMsg, compressedBytes...)
	return fullMsg, nil
}

func (b *BatchPoster) encodeAddBatch(
	seqNum *big.Int,
	prevMsgNum arbutil.MessageIndex,
	newMsgNum arbutil.MessageIndex,
	l2MessageData []byte,
	delayedMsg uint64,
	use4844 bool,
	delayProof *bridgegen.DelayProof,
) ([]byte, []kzg4844.Blob, error) {
	var methodName string
	if use4844 {
		if delayProof != nil {
			methodName = sequencerBatchPostWithBlobsDelayProofMethodName
		} else {
			methodName = sequencerBatchPostWithBlobsMethodName
		}
	} else if delayProof != nil {
		methodName = sequencerBatchPostDelayProofMethodName
	} else {
		methodName = sequencerBatchPostMethodName
	}
	method, ok := b.seqInboxABI.Methods[methodName]
	if !ok {
		return nil, nil, errors.New("failed to find add batch method")
	}
	var args []any
	var kzgBlobs []kzg4844.Blob
	var err error
	args = append(args, seqNum)
	if use4844 {
		kzgBlobs, err = blobs.EncodeBlobs(l2MessageData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to encode blobs: %w", err)
		}
	} else {
		// EIP4844 transactions to the sequencer inbox will not use transaction calldata for L2 info.
		args = append(args, l2MessageData)
	}
	args = append(args, new(big.Int).SetUint64(delayedMsg))
	args = append(args, b.config().gasRefunder)
	args = append(args, new(big.Int).SetUint64(uint64(prevMsgNum)))
	args = append(args, new(big.Int).SetUint64(uint64(newMsgNum)))
	if delayProof != nil {
		args = append(args, delayProof)
	}
	calldata, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, nil, err
	}
	fullCalldata := append([]byte{}, method.ID...)
	fullCalldata = append(fullCalldata, calldata...)
	return fullCalldata, kzgBlobs, nil
}

var ErrNormalGasEstimationFailed = errors.New("normal gas estimation failed")

type estimateGasParams struct {
	From         common.Address   `json:"from"`
	To           *common.Address  `json:"to"`
	Data         hexutil.Bytes    `json:"data"`
	MaxFeePerGas *hexutil.Big     `json:"maxFeePerGas"`
	AccessList   types.AccessList `json:"accessList"`
	BlobHashes   []common.Hash    `json:"blobVersionedHashes,omitempty"`
}

type OverrideAccount struct {
	StateDiff map[common.Hash]common.Hash `json:"stateDiff"`
}

type StateOverride map[common.Address]OverrideAccount

func estimateGas(client rpc.ClientInterface, ctx context.Context, params estimateGasParams, blockHex string) (uint64, error) {
	var gas hexutil.Uint64
	err := client.CallContext(ctx, &gas, "eth_estimateGas", params, blockHex)
	// If eth_estimateGas fails due to a revert, we try again with eth_call to get a detailed error.
	if err != nil && headerreader.IsExecutionReverted(err) {
		err = client.CallContext(ctx, nil, "eth_call", params, blockHex)
	}
	return uint64(gas), err
}

func (b *BatchPoster) estimateGasSimple(
	ctx context.Context,
	realData []byte,
	realBlobs []kzg4844.Blob,
	realAccessList types.AccessList,
) (uint64, error) {

	config := b.config()
	rpcClient := b.l1Reader.Client()
	rawRpcClient := rpcClient.Client()
	latestHeader, err := rpcClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	maxFeePerGas := arbmath.BigMulByUBips(latestHeader.BaseFee, config.GasEstimateBaseFeeMultipleBips)
	_, realBlobHashes, err := blobs.ComputeCommitmentsAndHashes(realBlobs)
	if err != nil {
		return 0, fmt.Errorf("failed to compute real blob commitments: %w", err)
	}
	// If we're at the latest nonce, we can skip the special future tx estimate stuff
	gas, err := estimateGas(rawRpcClient, ctx, estimateGasParams{
		From:         b.dataPoster.Sender(),
		To:           &b.seqInboxAddr,
		Data:         realData,
		MaxFeePerGas: (*hexutil.Big)(maxFeePerGas),
		BlobHashes:   realBlobHashes,
		AccessList:   realAccessList,
	}, "latest")
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrNormalGasEstimationFailed, err)
	}
	return gas + config.ExtraBatchGas, nil
}

// This estimates gas for a batch with future nonce
// a prev. batch is already pending in the parent chain's mempool
func (b *BatchPoster) estimateGasForFutureTx(
	ctx context.Context,
	sequencerMessage []byte,
	delayedMessagesBefore uint64,
	delayedMessagesAfter uint64,
	realAccessList types.AccessList,
	usingBlobs bool,
	delayProof *bridgegen.DelayProof,
) (uint64, error) {
	config := b.config()
	rpcClient := b.l1Reader.Client()
	rawRpcClient := rpcClient.Client()
	latestHeader, err := rpcClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	maxFeePerGas := arbmath.BigMulByUBips(latestHeader.BaseFee, config.GasEstimateBaseFeeMultipleBips)

	// Here we set seqNum to MaxUint256, and prevMsgNum to 0, because it disables the smart contracts' consistency checks.
	// However, we set nextMsgNum to 1 because it is necessary for a correct estimation for the final to be non-zero.
	// Because we're likely estimating against older state, this might not be the actual next message,
	// but the gas used should be the same.
	data, kzgBlobs, err := b.encodeAddBatch(abi.MaxUint256, 0, 1, sequencerMessage, delayedMessagesAfter, usingBlobs, delayProof)
	if err != nil {
		return 0, err
	}
	_, blobHashes, err := blobs.ComputeCommitmentsAndHashes(kzgBlobs)
	if err != nil {
		return 0, fmt.Errorf("failed to compute blob commitments: %w", err)
	}
	gasParams := estimateGasParams{
		From:         b.dataPoster.Sender(),
		To:           &b.seqInboxAddr,
		Data:         data,
		MaxFeePerGas: (*hexutil.Big)(maxFeePerGas),
		BlobHashes:   blobHashes,
		// This isn't perfect because we're probably estimating the batch at a different sequence number,
		// but it should overestimate rather than underestimate which is fine.
		AccessList: realAccessList,
	}
	// slot 0 in the SequencerInbox smart contract holds totalDelayedMessagesRead -
	// This is the number of delayed messages that sequencer knows were processed
	// SequencerInbox checks this value to make sure delayed inbox isn't going backward,
	// And it makes it know if a delayProof is needed
	// Both are required for successful batch posting
	stateOverride := StateOverride{
		b.seqInboxAddr: {
			StateDiff: map[common.Hash]common.Hash{
				// slot 0
				{}: common.Hash(arbmath.Uint64ToU256Bytes(delayedMessagesBefore)),
			},
		},
	}
	var gas hexutil.Uint64
	err = rawRpcClient.CallContext(ctx, &gas, "eth_estimateGas", gasParams, rpc.PendingBlockNumber, stateOverride)
	if err != nil {
		sequencerMessageHeader := sequencerMessage
		if len(sequencerMessageHeader) > 33 {
			sequencerMessageHeader = sequencerMessageHeader[:33]
		}
		// If eth_estimateGas fails due to a revert, we try again with eth_call to get a detailed error.
		if headerreader.IsExecutionReverted(err) {
			err = rawRpcClient.CallContext(ctx, nil, "eth_call", gasParams, rpc.PendingBlockNumber, stateOverride)
		}
		log.Warn(
			"error estimating gas for batch",
			"err", err,
			"delayedMessagesBefore", delayedMessagesBefore,
			"delayedMessagesAfter", delayedMessagesAfter,
			"sequencerMessageHeader", hex.EncodeToString(sequencerMessageHeader),
			"sequencerMessageLen", len(sequencerMessage),
		)
		return 0, fmt.Errorf("error estimating gas for batch: %w", err)
	}
	return uint64(gas) + config.ExtraBatchGas, nil
}

const ethPosBlockTime = 12 * time.Second

var errAttemptLockFailed = errors.New("failed to acquire lock; either another batch poster posted a batch or this node fell behind")

func (b *BatchPoster) MaybePostSequencerBatch(ctx context.Context) (bool, error) {
	if b.batchReverted.Load() {
		return false, fmt.Errorf("batch was reverted, not posting any more batches")
	}
	nonce, batchPositionBytes, err := b.dataPoster.GetNextNonceAndMeta(ctx)
	if err != nil {
		return false, err
	}
	var batchPosition batchPosterPosition
	if err := rlp.DecodeBytes(batchPositionBytes, &batchPosition); err != nil {
		return false, fmt.Errorf("decoding batch position: %w", err)
	}

	dbBatchCount, err := b.inbox.GetBatchCount()
	if err != nil {
		return false, err
	}
	if dbBatchCount > batchPosition.NextSeqNum {
		return false, fmt.Errorf("attempting to post batch %v, but the local inbox tracker database already has %v batches", batchPosition.NextSeqNum, dbBatchCount)
	}
	if b.building == nil || b.building.startMsgCount != batchPosition.MessageCount {
		latestHeader, err := b.l1Reader.LastHeader(ctx)
		if err != nil {
			return false, err
		}
		config := b.config()
		buildingForEthDA := len(b.dapWriters) == 0 || b.ethDAFallbackRemaining > 0
		// Determine if we should use 4844 blobs (only relevant when posting to EthDA)
		var use4844 bool
		if buildingForEthDA &&
			config.Post4844Blobs &&
			latestHeader.ExcessBlobGas != nil &&
			latestHeader.BlobGasUsed != nil {
			arbOSVersion, err := b.arbOSVersionGetter.ArbOSVersionForMessageIndex(arbutil.MessageIndex(arbmath.SaturatingUSub(uint64(batchPosition.MessageCount), 1))).Await(ctx)
			if err != nil {
				return false, err
			}
			if arbOSVersion >= params.ArbosVersion_20 {
				if config.IgnoreBlobPrice {
					use4844 = true
				} else {
					backlog := b.backlog.Load()
					// Logic to prevent switching from non-4844 batches to 4844 batches too often,
					// so that blocks can be filled efficiently. The geth txpool rejects txs for
					// accounts that already have the other type of txs in the pool with
					// "address already reserved". This logic makes sure that, if there is a backlog,
					// that enough non-4844 batches have been posted to fill a block before switching.
					if backlog == 0 ||
						b.non4844BatchCount == 0 ||
						b.non4844BatchCount > 16 {
						blobFeePerByte, err := b.parentChain.BlobFeePerByte(ctx, latestHeader)
						if err != nil {
							return false, err
						}
						blobFeePerByte.Mul(blobFeePerByte, blobTxBlobGasPerBlob)
						blobFeePerByte.Div(blobFeePerByte, usableBytesInBlob)

						// STANDARD_TOKEN_COST = 4
						// TOTAL_COST_FLOOR_PER_TOKEN = 10
						//
						// The following analysis is applied for transactions unrelated to contract creation.
						//
						// Before EIP-7623, gas used related to calldata is defined as
						// STANDARD_TOKEN_COST * (zero_bytes_in_calldata + nonzero_bytes_in_calldata * 4).
						// Considering the worst case scenario regarding gas used per calldata byte,
						// in which calldata only has non-zero bytes, each calldata byte will consume STANDARD_TOKEN * 4, which is 16 gas.
						//
						// With EIP-7623, considering the worst case scenario regarding gas used per calldata byte,
						// in which calldata is also composed only of non-zero bytes,
						// and that (TOTAL_COST_FLOOR_PER_TOKEN * tokens_in_calldata > STANDARD_TOKEN_COST * tokens_in_calldata + execution_gas_used),
						// each calldata byte will consume TOTAL_COST_FLOOR_PER_TOKEN * 4, which is 40 gas.
						calldataFeePerByteMultiplier := uint64(16)
						parentChainIsUsingEIP7623, err := b.ParentChainIsUsingEIP7623(ctx, latestHeader)
						if err != nil {
							log.Error("ParentChainIsUsingEIP7623 failed", "err", err)
						} else if parentChainIsUsingEIP7623 {
							calldataFeePerByteMultiplier = uint64(40)
						}

						calldataFeePerByte := arbmath.BigMulByUint(latestHeader.BaseFee, calldataFeePerByteMultiplier)
						use4844 = arbmath.BigLessThan(blobFeePerByte, calldataFeePerByte)
					}
				}
			}
		}

		if b.ethDAFallbackRemaining > 0 {
			log.Info("Building batch for EthDA due to previous altDA failure", "use4844", use4844, "fallbackRemaining", b.ethDAFallbackRemaining)
		}

		// Only use 4844 batching when posting to EthDA
		use4844 = use4844 && buildingForEthDA
		usingAltDA := !buildingForEthDA
		segments, err := b.newBatchSegments(ctx, batchPosition.DelayedMessageCount, use4844, usingAltDA)
		if err != nil {
			return false, err
		}
		b.building = &buildingBatch{
			segments:      segments,
			msgCount:      batchPosition.MessageCount,
			startMsgCount: batchPosition.MessageCount,
			use4844:       use4844,
		}
		if b.config().CheckBatchCorrectness {
			b.building.muxBackend = &simulatedMuxBackend{
				batchSeqNum: batchPosition.NextSeqNum,
				allMsgs:     make(map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata),
			}
		}
	}
	msgCount, err := b.streamer.GetMessageCount()
	if err != nil {
		return false, err
	}
	if msgCount <= batchPosition.MessageCount {
		// There's nothing after the newest batch, therefore batch posting was not required
		return false, nil
	}

	config := b.config()
	forcePostBatch := config.MaxDelay <= 0

	var l1BoundMaxBlockNumber uint64 = math.MaxUint64
	var l1BoundMaxTimestamp uint64 = math.MaxUint64
	var l1BoundMinBlockNumber uint64
	var l1BoundMinTimestamp uint64
	var l1BoundMinBlockNumberWithBypass uint64
	var l1BoundMinTimestampWithBypass uint64
	hasL1Bound := config.l1BlockBound != l1BlockBoundIgnore
	if hasL1Bound {
		var l1Bound *types.Header
		var err error
		if config.l1BlockBound == l1BlockBoundLatest {
			l1Bound, err = b.l1Reader.LastHeader(ctx)
		} else if config.l1BlockBound == l1BlockBoundSafe || config.l1BlockBound == l1BlockBoundDefault {
			l1Bound, err = b.l1Reader.LatestSafeBlockHeader(ctx)
			if errors.Is(err, headerreader.ErrBlockNumberNotSupported) && config.l1BlockBound == l1BlockBoundDefault {
				// If getting the latest safe block is unsupported, and the L1BlockBound configuration is the default,
				// fall back to using the latest block instead of the safe block.
				l1Bound, err = b.l1Reader.LastHeader(ctx)
			}
		} else {
			if config.l1BlockBound != l1BlockBoundFinalized {
				log.Error(
					"unknown L1 block bound config value; falling back on using finalized",
					"l1BlockBoundString", config.L1BlockBound,
					"l1BlockBoundEnum", config.l1BlockBound,
				)
			}
			l1Bound, err = b.l1Reader.LatestFinalizedBlockHeader(ctx)
		}
		if err != nil {
			return false, fmt.Errorf("error getting L1 bound block: %w", err)
		}

		maxTimeVariationDelayBlocks, maxTimeVariationFutureBlocks, maxTimeVariationDelaySeconds, maxTimeVariationFutureSeconds, err := b.seqInbox.MaxTimeVariation(&bind.CallOpts{
			Context:     ctx,
			BlockNumber: l1Bound.Number,
		})
		if err != nil {
			// This might happen if the latest finalized block is old enough that our L1 node no longer has its state
			log.Warn("error getting max time variation on L1 bound block; falling back on latest block", "err", err)
			maxTimeVariationDelayBlocks, maxTimeVariationFutureBlocks, maxTimeVariationDelaySeconds, maxTimeVariationFutureSeconds, err = b.seqInbox.MaxTimeVariation(&bind.CallOpts{Context: ctx})
			if err != nil {
				return false, fmt.Errorf("error getting max time variation: %w", err)
			}
		}

		l1BoundBlockNumber := arbutil.ParentHeaderToL1BlockNumber(l1Bound)
		l1BoundMaxBlockNumber = arbmath.SaturatingUAdd(l1BoundBlockNumber, arbmath.BigToUintSaturating(maxTimeVariationFutureBlocks))
		l1BoundMaxTimestamp = arbmath.SaturatingUAdd(l1Bound.Time, arbmath.BigToUintSaturating(maxTimeVariationFutureSeconds))

		latestHeader, err := b.l1Reader.LastHeader(ctx)
		if err != nil {
			return false, err
		}
		latestBlockNumber := arbutil.ParentHeaderToL1BlockNumber(latestHeader)
		l1BoundMinBlockNumber = arbmath.SaturatingUSub(latestBlockNumber, arbmath.BigToUintSaturating(maxTimeVariationDelayBlocks))
		l1BoundMinTimestamp = arbmath.SaturatingUSub(latestHeader.Time, arbmath.BigToUintSaturating(maxTimeVariationDelaySeconds))

		if config.L1BlockBoundBypass > 0 {
			// #nosec G115
			blockNumberWithPadding := arbmath.SaturatingUAdd(latestBlockNumber, uint64(config.L1BlockBoundBypass/ethPosBlockTime))
			// #nosec G115
			timestampWithPadding := arbmath.SaturatingUAdd(latestHeader.Time, uint64(config.L1BlockBoundBypass/time.Second))
			l1BoundMinBlockNumberWithBypass = arbmath.SaturatingUSub(blockNumberWithPadding, arbmath.BigToUintSaturating(maxTimeVariationDelayBlocks))
			l1BoundMinTimestampWithBypass = arbmath.SaturatingUSub(timestampWithPadding, arbmath.BigToUintSaturating(maxTimeVariationDelaySeconds))
		}
	}

	for b.building.msgCount < msgCount {
		msg, err := b.streamer.GetMessage(b.building.msgCount)
		if err != nil {
			log.Error("error getting message from streamer", "error", err)
			break
		}
		if msg.Message.Header.BlockNumber < l1BoundMinBlockNumberWithBypass || msg.Message.Header.Timestamp < l1BoundMinTimestampWithBypass {
			log.Error(
				"disabling L1 bound as batch posting message is close to the maximum delay",
				"blockNumber", msg.Message.Header.BlockNumber,
				"l1BoundMinBlockNumberWithBypass", l1BoundMinBlockNumberWithBypass,
				"timestamp", msg.Message.Header.Timestamp,
				"l1BoundMinTimestampWithBypass", l1BoundMinTimestampWithBypass,
				"l1BlockBoundBypass", config.L1BlockBoundBypass,
			)
			l1BoundMaxBlockNumber = math.MaxUint64
			l1BoundMaxTimestamp = math.MaxUint64
		}
		if msg.Message.Header.BlockNumber > l1BoundMaxBlockNumber || msg.Message.Header.Timestamp > l1BoundMaxTimestamp {
			b.lastHitL1Bounds = time.Now()
			log.Info(
				"not posting more messages because block number or timestamp exceed L1 bounds",
				"blockNumber", msg.Message.Header.BlockNumber,
				"l1BoundMaxBlockNumber", l1BoundMaxBlockNumber,
				"timestamp", msg.Message.Header.Timestamp,
				"l1BoundMaxTimestamp", l1BoundMaxTimestamp,
			)
			break
		}
		isDelayed := msg.DelayedMessagesRead > b.building.segments.delayedMsg
		success, err := b.building.segments.AddMessage(msg)
		if err != nil {
			// Clear our cache
			b.building = nil
			return false, fmt.Errorf("error adding message to batch: %w", err)
		}
		if !success {
			// this batch is full
			if !config.WaitForMaxDelay {
				forcePostBatch = true
			}
			b.building.haveUsefulMessage = true
			if b.building.firstUsefulMsg == nil {
				b.building.firstUsefulMsg = msg
			}
			break
		}
		if config.CheckBatchCorrectness {
			b.building.muxBackend.allMsgs[b.building.msgCount] = msg
			if isDelayed {
				b.building.muxBackend.delayedInbox = append(b.building.muxBackend.delayedInbox, msg)
			}
		}
		// #nosec G115
		timeSinceMsg := time.Since(time.Unix(int64(msg.Message.Header.Timestamp), 0))
		if (msg.Message.Header.Kind != arbostypes.L1MessageType_BatchPostingReport) ||
			(config.MaxEmptyBatchDelay > 0 && timeSinceMsg >= config.MaxEmptyBatchDelay) {
			b.building.haveUsefulMessage = true
			if b.building.firstUsefulMsg == nil {
				b.building.firstUsefulMsg = msg
			}
		}
		if isDelayed {
			if b.building.firstDelayedMsg == nil {
				b.building.firstDelayedMsg = msg
			}
		} else if b.building.firstNonDelayedMsg == nil {
			b.building.firstNonDelayedMsg = msg
		}
		b.building.msgCount++
	}

	feeEscalationBaseTime := time.Now()
	if b.building.firstUsefulMsg != nil {
		// #nosec G115
		feeEscalationBaseTime = time.Unix(int64(b.building.firstUsefulMsg.Message.Header.Timestamp), 0)
		if time.Since(feeEscalationBaseTime) >= config.MaxDelay {
			forcePostBatch = true
		}
	} else if b.building.firstDelayedMsg != nil && config.MaxEmptyBatchDelay > 0 {
		// #nosec G115
		feeEscalationBaseTime = time.Unix(int64(b.building.firstDelayedMsg.Message.Header.Timestamp), 0)
		if time.Since(feeEscalationBaseTime) >= config.MaxEmptyBatchDelay {
			forcePostBatch = true
			b.building.haveUsefulMessage = true
		}
	}

	var delayBufferConfig *DelayBufferConfig
	if b.building.firstDelayedMsg != nil { // Only fetch delayBufferConfig config when needed
		delayBufferConfig, err = GetDelayBufferConfig(ctx, b.seqInbox)
		if err != nil {
			return false, err
		}
		if delayBufferConfig.Enabled {
			latestHeader, err := b.l1Reader.LastHeader(ctx)
			if err != nil {
				return false, err
			}
			latestBlock := latestHeader.Number.Uint64()
			firstDelayedMsgBlock := b.building.firstDelayedMsg.Message.Header.BlockNumber
			thresholdLimit := firstDelayedMsgBlock + delayBufferConfig.Threshold - b.config().DelayBufferThresholdMargin
			if latestBlock >= thresholdLimit {
				log.Info("force post batch because of the delay buffer",
					"firstDelayedMsgBlock", firstDelayedMsgBlock,
					"threshold", delayBufferConfig.Threshold,
					"latestBlock", latestBlock)
				forcePostBatch = true
			}
		}
	}

	if b.building.firstNonDelayedMsg != nil && hasL1Bound && config.ReorgResistanceMargin > 0 {
		firstMsgBlockNumber := b.building.firstNonDelayedMsg.Message.Header.BlockNumber
		firstMsgTimeStamp := b.building.firstNonDelayedMsg.Message.Header.Timestamp
		// #nosec G115
		batchNearL1BoundMinBlockNumber := firstMsgBlockNumber <= arbmath.SaturatingUAdd(l1BoundMinBlockNumber, uint64(config.ReorgResistanceMargin/ethPosBlockTime))
		// #nosec G115
		batchNearL1BoundMinTimestamp := firstMsgTimeStamp <= arbmath.SaturatingUAdd(l1BoundMinTimestamp, uint64(config.ReorgResistanceMargin/time.Second))
		if batchNearL1BoundMinTimestamp || batchNearL1BoundMinBlockNumber {
			log.Error(
				"Disabling batch posting due to batch being within reorg resistance margin from layer 1 minimum block or timestamp bounds",
				"reorgResistanceMargin", config.ReorgResistanceMargin,
				"firstMsgTimeStamp", firstMsgTimeStamp,
				"l1BoundMinTimestamp", l1BoundMinTimestamp,
				"firstMsgBlockNumber", firstMsgBlockNumber,
				"l1BoundMinBlockNumber", l1BoundMinBlockNumber,
			)
			return false, errors.New("batch is within reorg resistance margin from layer 1 minimum block or timestamp bounds")
		}
	}

	if !forcePostBatch || !b.building.haveUsefulMessage {
		// the batch isn't full yet and we've posted a batch recently
		// don't post anything for now
		return false, nil
	}

	batchData, err := b.building.segments.CloseAndGetBytes()
	defer func() {
		b.building = nil // a closed batchSegments can't be reused
	}()
	if err != nil {
		return false, err
	}
	if batchData == nil {
		log.Debug("BatchPoster: batch nil", "sequence nr.", batchPosition.NextSeqNum, "from", batchPosition.MessageCount, "prev delayed", batchPosition.DelayedMessageCount)
		return false, nil
	}
	var sequencerMsg []byte

	// Try DA writers if not forced to EthDA
	if len(b.dapWriters) > 0 && b.ethDAFallbackRemaining == 0 {
		if !b.redisLock.AttemptLock(ctx) {
			return false, errAttemptLockFailed
		}

		gotNonce, gotMeta, err := b.dataPoster.GetNextNonceAndMeta(ctx)
		if err != nil {
			batchPosterDAFailureCounter.Inc(1)
			return false, err
		}
		if nonce != gotNonce {
			batchPosterDAFailureCounter.Inc(1)
			return false, fmt.Errorf("%w: nonce changed from %d to %d while creating batch", storage.ErrStorageRace, nonce, gotNonce)
		}
		if !bytes.Equal(batchPositionBytes, gotMeta) {
			batchPosterDAFailureCounter.Inc(1)
			var actualBatchPosition batchPosterPosition
			if err := rlp.DecodeBytes(gotMeta, &actualBatchPosition); err != nil {
				return false, fmt.Errorf("%w: received unexpected batch position bytes", err)
			}
			return false, fmt.Errorf("%w: batch position changed from %v to %v while creating batch", storage.ErrStorageRace, batchPosition, actualBatchPosition)
		}

		// Try the DA writer at currentWriterIndex
		writerIndex := b.currentWriterIndex
		writer := b.dapWriters[writerIndex]

		log.Debug("Attempting to store batch with DA writer", "writerIndex", writerIndex, "numWriters", len(b.dapWriters), "batchSize", len(batchData))
		storeStart := time.Now()
		// #nosec G115
		sequencerMsg, err = writer.Store(batchData, uint64(time.Now().Add(config.DASRetentionPeriod).Unix())).Await(ctx)
		storeDuration := time.Since(storeStart)

		if err != nil {
			if errors.Is(err, daprovider.ErrMessageTooLarge) {
				log.Info("DA writer reports message too large, will rebuild batch", "writerIndex", writerIndex, "error", err, "duration", storeDuration, "batchSize", len(batchData))
				b.building = nil
				return true, nil // Trigger immediate rebuild with same writer
			}
			if errors.Is(err, daprovider.ErrFallbackRequested) {
				log.Warn("DA writer explicitly requested fallback", "writerIndex", writerIndex, "error", err, "duration", storeDuration)
				// Check if there's a next writer to try
				if writerIndex+1 < len(b.dapWriters) {
					b.currentWriterIndex = writerIndex + 1
					b.building = nil
					log.Info("Will rebuild batch for next DA writer", "nextWriterIndex", b.currentWriterIndex)
					return true, nil // Trigger rebuild with next writer's size
				}
				// No more writers - fall back to EthDA
				batchPosterDAFailureCounter.Inc(1)
				if config.DisableDapFallbackStoreDataOnChain {
					log.Error("DA fallback to EthDA is disabled, cannot post batch", "error", err)
					return false, fmt.Errorf("all DA writers failed: %w", err)
				}
				log.Info("DA writers exhausted, will rebuild for EthDA", "error", err, "batchSize", len(batchData), "fallbackBatches", config.EthDAFallbackBatchCount)
				b.ethDAFallbackRemaining = config.EthDAFallbackBatchCount
				b.currentWriterIndex = 0 // Reset for next batch after EthDA fallback period
				b.building = nil
				return true, nil // Trigger rebuild for EthDA
			}
			// Non-fallback error - fail immediately
			log.Error("DA writer failed, operator action required", "writerIndex", writerIndex, "error", err, "duration", storeDuration)
			batchPosterDAFailureCounter.Inc(1)
			return false, fmt.Errorf("DA writer %d failed: %w", writerIndex, err)
		}

		log.Debug("DA writer succeeded", "writerIndex", writerIndex, "duration", storeDuration)
		batchPosterDASuccessCounter.Inc(1)
		batchPosterDALastSuccessfulActionGauge.Update(time.Now().Unix())
	} else {
		// No DA writers or forced to EthDA
		sequencerMsg = batchData
	}

	prevMessageCount := batchPosition.MessageCount
	if b.config().Dangerous.AllowPostingFirstBatchWhenSequencerMessageCountMismatch && !b.postedFirstBatch {
		// AllowPostingFirstBatchWhenSequencerMessageCountMismatch can be used when the
		// message count stored in batch poster's database gets out
		// of sync with the sequencerReportedSubMessageCount stored in the parent chain.
		//
		// An example of when this out of sync issue can happen:
		// 1. Batch poster is running fine, but then it shutdowns for more than 24h.
		// 2. While the batch poster is down, someone sends a transaction to the parent chain
		// smart contract to move a message from the delayed inbox to the main inbox.
		// This will not update sequencerReportedSubMessageCount in the parent chain.
		// 3. When batch poster starts again, the inbox reader will update the
		// message count that is maintained in the batch poster's database to be equal to
		// (sequencerReportedSubMessageCount that is stored in parent chain) +
		// (the amount of delayed messages that were moved from the delayed inbox to the main inbox).
		// At this moment the message count stored on batch poster's database gets out of sync with
		// the sequencerReportedSubMessageCount stored in the parent chain.

		// When the first batch is posted, sequencerReportedSubMessageCount in
		// the parent chain will be updated to be equal to the new message count provided
		// by the batch poster, which will make this out of sync issue disappear.
		// That is why this strategy is only applied for the first batch posted after
		// startup.

		// If prevMessageCount is set to zero, sequencer inbox's smart contract allows
		// to post a batch even if sequencerReportedSubMessageCount is not equal
		// to the provided prevMessageCount
		prevMessageCount = 0
	}

	var delayProof *bridgegen.DelayProof
	latestHeader, err := b.l1Reader.LastHeader(ctx)
	if err != nil {
		return false, err
	}
	delayProofNeeded := b.building.firstDelayedMsg != nil && delayBufferConfig != nil && delayBufferConfig.Enabled // checking if delayBufferConfig is non-nil isn't needed, but better to be safe
	delayProofNeeded = delayProofNeeded && (config.DelayBufferAlwaysUpdatable || delayBufferConfig.isUpdatable(latestHeader.Number.Uint64()))
	if delayProofNeeded {
		delayProof, err = GenDelayProof(ctx, b.building.firstDelayedMsg, b.inbox)
		if err != nil {
			return false, fmt.Errorf("failed to generate delay proof: %w", err)
		}
	}

	data, kzgBlobs, err := b.encodeAddBatch(new(big.Int).SetUint64(batchPosition.NextSeqNum), prevMessageCount, b.building.msgCount, sequencerMsg, b.building.segments.delayedMsg, b.building.use4844, delayProof)
	if err != nil {
		return false, err
	}
	if len(kzgBlobs) > 0 {
		maxBlobGasPerBlock, err := b.parentChain.MaxBlobGasPerBlock(ctx, latestHeader)
		if err != nil {
			return false, err
		}
		// #nosec G115
		if len(kzgBlobs)*params.BlobTxBlobGasPerBlob > int(maxBlobGasPerBlock) {
			// #nosec G115
			return false, fmt.Errorf("produced %v blobs for batch but a block can only hold %v (compressed batch was %v bytes long)", len(kzgBlobs), int(maxBlobGasPerBlock)/params.BlobTxBlobGasPerBlob, len(sequencerMsg))
		}
	}
	accessList := b.accessList(batchPosition.NextSeqNum, b.building.segments.delayedMsg)
	var gasLimit uint64
	if b.config().Dangerous.FixedGasLimit != 0 {
		gasLimit = b.config().Dangerous.FixedGasLimit
	} else {
		useSimpleEstimation := b.dataPoster.MaxMempoolTransactions() == 1
		if !useSimpleEstimation {
			// Check if we can use normal estimation anyways because we're at the latest nonce
			latestNonce, err := b.l1Reader.Client().NonceAt(ctx, b.dataPoster.Sender(), nil)
			if err != nil {
				return false, err
			}
			useSimpleEstimation = latestNonce == nonce
		}

		if useSimpleEstimation {
			gasLimit, err = b.estimateGasSimple(ctx, data, kzgBlobs, accessList)
		} else {
			// When there are previous batches queued up in the dataPoster, we override the delayed message count in the sequencer inbox
			// so it accepts the corresponding delay proof. Otherwise, the gas estimation would revert.
			var delayedMsgBefore uint64
			if b.building.firstDelayedMsg != nil {
				delayedMsgBefore = b.building.firstDelayedMsg.DelayedMessagesRead - 1
			} else if b.building.firstNonDelayedMsg != nil {
				delayedMsgBefore = b.building.firstNonDelayedMsg.DelayedMessagesRead
			}
			gasLimit, err = b.estimateGasForFutureTx(ctx, sequencerMsg, delayedMsgBefore, b.building.segments.delayedMsg, accessList, len(kzgBlobs) > 0, delayProof)
		}
	}
	if err != nil {
		return false, err
	}
	newMeta, err := rlp.EncodeToBytes(batchPosterPosition{
		MessageCount:        b.building.msgCount,
		DelayedMessageCount: b.building.segments.delayedMsg,
		NextSeqNum:          batchPosition.NextSeqNum + 1,
	})
	if err != nil {
		return false, err
	}

	if config.CheckBatchCorrectness {
		// For batch correctness checking, we use a wrapper that overrides blob reads
		// with a simulated reader for the local kzgBlobs (which haven't been posted yet).
		// All other DA reads pass through to the original registry.
		// Explicit nil check needed: a typed nil (*DAProviderRegistry) assigned to an interface is not nil.
		var baseDapReaders arbstate.DapReaderSource
		if b.dapReaders != nil {
			baseDapReaders = b.dapReaders
		}
		dapReaders := arbstate.NewBlobReaderOverride(
			baseDapReaders,
			daprovider.NewReaderForBlobReader(&simulatedBlobReader{kzgBlobs}),
		)
		seqMsg := binary.BigEndian.AppendUint64([]byte{}, l1BoundMinTimestamp)
		seqMsg = binary.BigEndian.AppendUint64(seqMsg, l1BoundMaxTimestamp)
		seqMsg = binary.BigEndian.AppendUint64(seqMsg, l1BoundMinBlockNumber)
		seqMsg = binary.BigEndian.AppendUint64(seqMsg, l1BoundMaxBlockNumber)
		seqMsg = binary.BigEndian.AppendUint64(seqMsg, b.building.segments.delayedMsg)
		seqMsg = append(seqMsg, sequencerMsg...)
		b.building.muxBackend.seqMsg = seqMsg
		b.building.muxBackend.delayedInboxStart = batchPosition.DelayedMessageCount
		b.building.muxBackend.SetPositionWithinMessage(0)
		simMux := arbstate.NewInboxMultiplexer(b.building.muxBackend, batchPosition.DelayedMessageCount, dapReaders, daprovider.KeysetValidate)
		log.Debug("Begin checking the correctness of batch against inbox multiplexer", "startMsgSeqNum", batchPosition.MessageCount, "endMsgSeqNum", b.building.msgCount-1)
		for i := batchPosition.MessageCount; i < b.building.msgCount; i++ {
			msg, err := simMux.Pop(ctx)
			if err != nil {
				return false, fmt.Errorf("error getting message from simulated inbox multiplexer (Pop) when testing correctness of batch: %w", err)
			}
			if msg.DelayedMessagesRead != b.building.muxBackend.allMsgs[i].DelayedMessagesRead {
				return false, fmt.Errorf("simulated inbox multiplexer failed to produce correct delayedMessagesRead field for msg with seqNum: %d. Got: %d, Want: %d", i, msg.DelayedMessagesRead, b.building.muxBackend.allMsgs[i].DelayedMessagesRead)
			}
			if !msg.Message.Equals(b.building.muxBackend.allMsgs[i].Message) {
				return false, fmt.Errorf("simulated inbox multiplexer failed to produce correct message field for msg with seqNum: %d", i)
			}
		}
		log.Debug("Successfully checked that the batch produces correct messages when ran through inbox multiplexer", "sequenceNumber", batchPosition.NextSeqNum)
	}

	if !b.redisLock.AttemptLock(ctx) {
		return false, errAttemptLockFailed
	}

	tx, err := b.dataPoster.PostTransaction(ctx,
		feeEscalationBaseTime,
		nonce,
		newMeta,
		b.seqInboxAddr,
		data,
		gasLimit,
		new(big.Int),
		kzgBlobs,
		accessList,
	)
	if err != nil {
		return false, err
	}
	b.postedFirstBatch = true
	b.currentWriterIndex = 0 // Reset to first writer after successful batch
	log.Info(
		"BatchPoster: batch sent",
		"sequenceNumber", batchPosition.NextSeqNum,
		"from", batchPosition.MessageCount,
		"to", b.building.msgCount,
		"prevDelayed", batchPosition.DelayedMessageCount,
		"currentDelayed", b.building.segments.delayedMsg,
		"totalSegments", len(b.building.segments.rawSegments),
		"numBlobs", len(kzgBlobs),
	)

	recentlyHitL1Bounds := time.Since(b.lastHitL1Bounds) < config.PollInterval*3
	postedMessages := b.building.msgCount - batchPosition.MessageCount
	b.messagesPerBatch.Update(uint64(postedMessages))
	if b.building.use4844 {
		b.non4844BatchCount = 0
	} else {
		b.non4844BatchCount++
	}
	unpostedMessages := msgCount - b.building.msgCount
	messagesPerBatch := b.messagesPerBatch.Average()
	if messagesPerBatch == 0 {
		// This should be impossible because we always post at least one message in a batch.
		// That said, better safe than sorry, as we would panic if this remained at 0.
		log.Warn(
			"messagesPerBatch is somehow zero",
			"postedMessages", postedMessages,
			"buildingFrom", batchPosition.MessageCount,
			"buildingTo", b.building.msgCount,
		)
		messagesPerBatch = 1
	}
	backlog := uint64(unpostedMessages) / messagesPerBatch
	// #nosec G115
	batchPosterEstimatedBatchBacklogGauge.Update(int64(backlog))
	if backlog > 10 {
		logLevel := log.Warn
		if recentlyHitL1Bounds {
			logLevel = log.Info
		} else if backlog > 30 {
			logLevel = log.Error
		}
		logLevel(
			"a large batch posting backlog exists",
			"recentlyHitL1Bounds", recentlyHitL1Bounds,
			"currentPosition", b.building.msgCount,
			"messageCount", msgCount,
			"messagesPerBatch", messagesPerBatch,
			"postedMessages", postedMessages,
			"unpostedMessages", unpostedMessages,
			"batchBacklogEstimate", backlog,
		)
	}
	if recentlyHitL1Bounds {
		// This backlog isn't "real" in that we don't want to post any more messages.
		// Setting the backlog to 0 here ensures that we don't lower compression as a result.
		backlog = 0
	}
	b.backlog.Store(backlog)

	// If we aren't queueing up transactions, wait for the receipt before moving on to the next batch.
	if config.DataPoster.UseNoOpStorage {
		receipt, err := b.l1Reader.WaitForTxApproval(ctx, tx)
		if err != nil {
			return false, fmt.Errorf("error waiting for tx receipt: %w", err)
		}
		log.Info("Got successful receipt from batch poster transaction", "txHash", tx.Hash(), "blockNumber", receipt.BlockNumber, "blockHash", receipt.BlockHash)
	}

	// After successful EthDA batch post in fallback mode, decrement counter and potentially retry AltDA
	if b.ethDAFallbackRemaining > 0 {
		b.ethDAFallbackRemaining--
		if b.ethDAFallbackRemaining == 0 {
			log.Info("EthDA fallback period complete, will retry AltDA")
		} else {
			log.Info("Successful EthDA batch post, continuing fallback mode",
				"fallbackRemaining", b.ethDAFallbackRemaining)
		}
	}

	return true, nil
}

func (b *BatchPoster) GetBacklogEstimate() uint64 {
	return b.backlog.Load()
}

func (b *BatchPoster) Start(ctxIn context.Context) {
	b.dataPoster.Start(ctxIn)
	b.redisLock.Start(ctxIn)
	b.StopWaiter.Start(ctxIn, b)
	b.LaunchThread(b.pollForReverts)
	b.LaunchThread(b.pollForL1PriceData)
	commonEphemeralErrorHandler := util.NewEphemeralErrorHandler(time.Minute, "", 0)
	exceedMaxMempoolSizeEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, dataposter.ErrExceedsMaxMempoolSize.Error(), time.Minute)
	storageRaceEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, storage.ErrStorageRace.Error(), time.Minute)
	normalGasEstimationFailedEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, ErrNormalGasEstimationFailed.Error(), time.Minute)
	accumulatorNotFoundEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, AccumulatorNotFoundErr.Error(), time.Minute)
	nonceTooHighEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, core.ErrNonceTooHigh.Error(), time.Minute)
	resetAllEphemeralErrs := func() {
		commonEphemeralErrorHandler.Reset()
		exceedMaxMempoolSizeEphemeralErrorHandler.Reset()
		storageRaceEphemeralErrorHandler.Reset()
		normalGasEstimationFailedEphemeralErrorHandler.Reset()
		accumulatorNotFoundEphemeralErrorHandler.Reset()
		nonceTooHighEphemeralErrorHandler.Reset()
	}
	b.CallIteratively(func(ctx context.Context) time.Duration {
		var err error
		if common.HexToAddress(b.config().GasRefunderAddress) != (common.Address{}) {
			gasRefunderBalance, err := b.l1Reader.Client().BalanceAt(ctx, common.HexToAddress(b.config().GasRefunderAddress), nil)
			if err != nil {
				log.Warn("error fetching batch poster gas refunder balance", "err", err)
			} else {
				batchPosterGasRefunderBalance.Update(arbmath.BalancePerEther(gasRefunderBalance))
			}
		}
		if b.dataPoster.Sender() != (common.Address{}) {
			walletBalance, err := b.l1Reader.Client().BalanceAt(ctx, b.dataPoster.Sender(), nil)
			if err != nil {
				log.Warn("error fetching batch poster wallet balance", "err", err)
			} else {
				batchPosterWalletBalance.Update(arbmath.BalancePerEther(walletBalance))
			}
		}
		couldLock, err := b.redisLock.CouldAcquireLock(ctx)
		if err != nil {
			log.Warn("Error checking if we could acquire redis lock", "err", err)
			// Might as well try, worst case we fail to lock
			couldLock = true
		}
		if !couldLock {
			log.Debug("Not posting batches right now because another batch poster has the lock or this node is behind")
			b.building = nil
			resetAllEphemeralErrs()
			return b.config().PollInterval
		}
		posted, err := b.MaybePostSequencerBatch(ctx)
		if err == nil {
			resetAllEphemeralErrs()
		}
		if err != nil {
			if ctx.Err() != nil {
				// Shutting down. No need to print the context canceled error.
				return 0
			}
			b.building = nil
			logLevel := log.Error
			// Likely the inbox tracker just isn't caught up.
			// Let's see if this error disappears naturally.
			logLevel = commonEphemeralErrorHandler.LogLevel(err, logLevel)
			// If the error matches one of these, it's only logged at debug for the first minute,
			// then at warn for the next 4 minutes, then at error. If the error isn't one of these,
			// it'll be logged at warn for the first minute, then at error.
			logLevel = exceedMaxMempoolSizeEphemeralErrorHandler.LogLevel(err, logLevel)
			logLevel = storageRaceEphemeralErrorHandler.LogLevel(err, logLevel)
			logLevel = normalGasEstimationFailedEphemeralErrorHandler.LogLevel(err, logLevel)
			logLevel = accumulatorNotFoundEphemeralErrorHandler.LogLevel(err, logLevel)
			logLevel = nonceTooHighEphemeralErrorHandler.LogLevel(err, logLevel)
			logLevel("error posting batch", "err", err)
			// Only increment batchPosterFailureCounter metric in cases of non-ephemeral errors
			if util.CompareLogLevels(logLevel, log.Error) {
				batchPosterFailureCounter.Inc(1)
			}
			return b.config().ErrorDelay
		} else if posted {
			return 0
		} else {
			return b.config().PollInterval
		}
	})
}

func (b *BatchPoster) StopAndWait() {
	b.StopWaiter.StopAndWait()
	b.dataPoster.StopAndWait()
	b.redisLock.StopAndWait()
}

type BoolRing struct {
	buffer         []bool
	bufferPosition int
}

func NewBoolRing(size int) *BoolRing {
	return &BoolRing{
		buffer: make([]bool, 0, size),
	}
}

func (b *BoolRing) Update(value bool) {
	period := cap(b.buffer)
	if period == 0 {
		return
	}
	if len(b.buffer) < period {
		b.buffer = append(b.buffer, value)
	} else {
		b.buffer[b.bufferPosition] = value
	}
	b.bufferPosition = (b.bufferPosition + 1) % period
}

func (b *BoolRing) Empty() bool {
	return len(b.buffer) == 0
}

// Peek returns the most recently inserted value.
// Assumes not empty, check Empty() first
func (b *BoolRing) Peek() bool {
	lastPosition := b.bufferPosition - 1
	if lastPosition < 0 {
		// This is the case where we have wrapped around, since Peek() shouldn't
		// be called without checking Empty(), so we can just use capactity.
		lastPosition = cap(b.buffer) - 1
	}
	return b.buffer[lastPosition]
}

// All returns true if the BoolRing is full and all values equal value.
func (b *BoolRing) All(value bool) bool {
	if len(b.buffer) < cap(b.buffer) {
		return false
	}
	for _, v := range b.buffer {
		if v != value {
			return false
		}
	}
	return true
}
