// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/redislock"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	batchPosterWalletBalance      = metrics.NewRegisteredGaugeFloat64("arb/batchposter/wallet/balanceether", nil)
	batchPosterGasRefunderBalance = metrics.NewRegisteredGaugeFloat64("arb/batchposter/gasrefunder/balanceether", nil)
	batchPosterSimpleRedisLockKey = "node.batch-poster.redis-lock.simple-lock-key"
)

type batchPosterPosition struct {
	MessageCount        arbutil.MessageIndex
	DelayedMessageCount uint64
	NextSeqNum          uint64
}

type BatchPoster struct {
	stopwaiter.StopWaiter
	l1Reader         *headerreader.HeaderReader
	inbox            *InboxTracker
	streamer         *TransactionStreamer
	config           BatchPosterConfigFetcher
	seqInbox         *bridgegen.SequencerInbox
	bridge           *bridgegen.Bridge
	syncMonitor      *SyncMonitor
	seqInboxABI      *abi.ABI
	seqInboxAddr     common.Address
	bridgeAddr       common.Address
	gasRefunderAddr  common.Address
	building         *buildingBatch
	daWriter         das.DataAvailabilityServiceWriter
	dataPoster       *dataposter.DataPoster
	redisLock        *redislock.Simple
	messagesPerBatch *arbmath.MovingAverage[uint64]
	// This is an atomic variable that should only be accessed atomically.
	// An estimate of the number of batches we want to post but haven't yet.
	// This doesn't include batches which we don't want to post yet due to the L1 bounds.
	backlog         uint64
	lastHitL1Bounds time.Time // The last time we wanted to post a message but hit the L1 bounds

	batchReverted        atomic.Bool // indicates whether data poster batch was reverted
	nextRevertCheckBlock int64       // the last parent block scanned for reverting batches

	accessList func(SequencerInboxAccs, AfterDelayedMessagesRead int) types.AccessList
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

type BatchPosterConfig struct {
	Enable                             bool `koanf:"enable"`
	DisableDasFallbackStoreDataOnChain bool `koanf:"disable-das-fallback-store-data-on-chain" reload:"hot"`
	// Max batch size.
	MaxSize int `koanf:"max-size" reload:"hot"`
	// Max batch post delay.
	MaxDelay time.Duration `koanf:"max-delay" reload:"hot"`
	// Wait for max BatchPost delay.
	WaitForMaxDelay bool `koanf:"wait-for-max-delay" reload:"hot"`
	// Batch post polling interval.
	PollInterval time.Duration `koanf:"poll-interval" reload:"hot"`
	// Batch posting error delay.
	ErrorDelay         time.Duration               `koanf:"error-delay" reload:"hot"`
	CompressionLevel   int                         `koanf:"compression-level" reload:"hot"`
	DASRetentionPeriod time.Duration               `koanf:"das-retention-period" reload:"hot"`
	GasRefunderAddress string                      `koanf:"gas-refunder-address" reload:"hot"`
	DataPoster         dataposter.DataPosterConfig `koanf:"data-poster" reload:"hot"`
	RedisUrl           string                      `koanf:"redis-url"`
	RedisLock          redislock.SimpleCfg         `koanf:"redis-lock" reload:"hot"`
	ExtraBatchGas      uint64                      `koanf:"extra-batch-gas" reload:"hot"`
	ParentChainWallet  genericconf.WalletConfig    `koanf:"parent-chain-wallet"`
	L1BlockBound       string                      `koanf:"l1-block-bound" reload:"hot"`
	L1BlockBoundBypass time.Duration               `koanf:"l1-block-bound-bypass" reload:"hot"`
	UseAccessLists     bool                        `koanf:"use-access-lists" reload:"hot"`

	gasRefunder  common.Address
	l1BlockBound l1BlockBound
}

func (c *BatchPosterConfig) Validate() error {
	if len(c.GasRefunderAddress) > 0 && !common.IsHexAddress(c.GasRefunderAddress) {
		return fmt.Errorf("invalid gas refunder address \"%v\"", c.GasRefunderAddress)
	}
	c.gasRefunder = common.HexToAddress(c.GasRefunderAddress)
	if c.MaxSize <= 40 {
		return errors.New("MaxBatchSize too small")
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

func BatchPosterConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Bool(prefix+".disable-das-fallback-store-data-on-chain", DefaultBatchPosterConfig.DisableDasFallbackStoreDataOnChain, "If unable to batch to DAS, disable fallback storing data on chain")
	f.Int(prefix+".max-size", DefaultBatchPosterConfig.MaxSize, "maximum batch size")
	f.Duration(prefix+".max-delay", DefaultBatchPosterConfig.MaxDelay, "maximum batch posting delay")
	f.Bool(prefix+".wait-for-max-delay", DefaultBatchPosterConfig.WaitForMaxDelay, "wait for the max batch delay, even if the batch is full")
	f.Duration(prefix+".poll-interval", DefaultBatchPosterConfig.PollInterval, "how long to wait after no batches are ready to be posted before checking again")
	f.Duration(prefix+".error-delay", DefaultBatchPosterConfig.ErrorDelay, "how long to delay after error posting batch")
	f.Int(prefix+".compression-level", DefaultBatchPosterConfig.CompressionLevel, "batch compression level")
	f.Duration(prefix+".das-retention-period", DefaultBatchPosterConfig.DASRetentionPeriod, "In AnyTrust mode, the period which DASes are requested to retain the stored batches.")
	f.String(prefix+".gas-refunder-address", DefaultBatchPosterConfig.GasRefunderAddress, "The gas refunder contract address (optional)")
	f.Uint64(prefix+".extra-batch-gas", DefaultBatchPosterConfig.ExtraBatchGas, "use this much more gas than estimation says is necessary to post batches")
	f.String(prefix+".redis-url", DefaultBatchPosterConfig.RedisUrl, "if non-empty, the Redis URL to store queued transactions in")
	f.String(prefix+".l1-block-bound", DefaultBatchPosterConfig.L1BlockBound, "only post messages to batches when they're within the max future block/timestamp as of this L1 block tag (\"safe\", \"finalized\", \"latest\", or \"ignore\" to ignore this check)")
	f.Duration(prefix+".l1-block-bound-bypass", DefaultBatchPosterConfig.L1BlockBoundBypass, "post batches even if not within the layer 1 future bounds if we're within this margin of the max delay")
	f.Bool(prefix+".use-access-lists", DefaultBatchPosterConfig.UseAccessLists, "post batches with access lists to reduce gas usage (disabled for L3s)")
	redislock.AddConfigOptions(prefix+".redis-lock", f)
	dataposter.DataPosterConfigAddOptions(prefix+".data-poster", f, dataposter.DefaultDataPosterConfig)
	genericconf.WalletConfigAddOptions(prefix+".parent-chain-wallet", f, DefaultBatchPosterConfig.ParentChainWallet.Pathname)
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	Enable:                             false,
	DisableDasFallbackStoreDataOnChain: false,
	// This default is overridden for L3 chains in applyChainParameters in cmd/nitro/nitro.go
	MaxSize:            100000,
	PollInterval:       time.Second * 10,
	ErrorDelay:         time.Second * 10,
	MaxDelay:           time.Hour,
	WaitForMaxDelay:    false,
	CompressionLevel:   brotli.BestCompression,
	DASRetentionPeriod: time.Hour * 24 * 15,
	GasRefunderAddress: "",
	ExtraBatchGas:      50_000,
	DataPoster:         dataposter.DefaultDataPosterConfig,
	ParentChainWallet:  DefaultBatchPosterL1WalletConfig,
	L1BlockBound:       "",
	L1BlockBoundBypass: time.Hour,
	UseAccessLists:     true,
	RedisLock:          redislock.DefaultCfg,
}

var DefaultBatchPosterL1WalletConfig = genericconf.WalletConfig{
	Pathname:      "batch-poster-wallet",
	Password:      genericconf.WalletConfigDefault.Password,
	PrivateKey:    genericconf.WalletConfigDefault.PrivateKey,
	Account:       genericconf.WalletConfigDefault.Account,
	OnlyCreateKey: genericconf.WalletConfigDefault.OnlyCreateKey,
}

var TestBatchPosterConfig = BatchPosterConfig{
	Enable:             true,
	MaxSize:            100000,
	PollInterval:       time.Millisecond * 10,
	ErrorDelay:         time.Millisecond * 10,
	MaxDelay:           0,
	WaitForMaxDelay:    false,
	CompressionLevel:   2,
	DASRetentionPeriod: time.Hour * 24 * 15,
	GasRefunderAddress: "",
	ExtraBatchGas:      10_000,
	DataPoster:         dataposter.TestDataPosterConfig,
	ParentChainWallet:  DefaultBatchPosterL1WalletConfig,
	L1BlockBound:       "",
	L1BlockBoundBypass: time.Hour,
	UseAccessLists:     true,
}

type BatchPosterOpts struct {
	DataPosterDB  ethdb.Database
	L1Reader      *headerreader.HeaderReader
	Inbox         *InboxTracker
	Streamer      *TransactionStreamer
	SyncMonitor   *SyncMonitor
	Config        BatchPosterConfigFetcher
	DeployInfo    *chaininfo.RollupAddresses
	TransactOpts  *bind.TransactOpts
	DAWriter      das.DataAvailabilityServiceWriter
	ParentChainID *big.Int
}

func NewBatchPoster(ctx context.Context, opts *BatchPosterOpts) (*BatchPoster, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(opts.DeployInfo.SequencerInbox, opts.L1Reader.Client())
	if err != nil {
		return nil, err
	}
	bridge, err := bridgegen.NewBridge(opts.DeployInfo.Bridge, opts.L1Reader.Client())
	if err != nil {
		return nil, err
	}
	if err = opts.Config().Validate(); err != nil {
		return nil, err
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
		l1Reader:        opts.L1Reader,
		inbox:           opts.Inbox,
		streamer:        opts.Streamer,
		syncMonitor:     opts.SyncMonitor,
		config:          opts.Config,
		bridge:          bridge,
		seqInbox:        seqInbox,
		seqInboxABI:     seqInboxABI,
		seqInboxAddr:    opts.DeployInfo.SequencerInbox,
		gasRefunderAddr: opts.Config().gasRefunder,
		bridgeAddr:      opts.DeployInfo.Bridge,
		daWriter:        opts.DAWriter,
		redisLock:       redisLock,
	}
	b.messagesPerBatch, err = arbmath.NewMovingAverage[uint64](20)
	if err != nil {
		return nil, err
	}
	dataPosterConfigFetcher := func() *dataposter.DataPosterConfig {
		return &(opts.Config().DataPoster)
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
	b.accessList = func(SequencerInboxAccs, AfterDelayedMessagesRead int) types.AccessList {
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

type AccessListOpts struct {
	SequencerInboxAddr       common.Address
	BridgeAddr               common.Address
	DataPosterAddr           common.Address
	GasRefunderAddr          common.Address
	SequencerInboxAccs       int
	AfterDelayedMessagesRead int
}

// AccessList returns access list (contracts, storage slots) for batchposter.
func AccessList(opts *AccessListOpts) types.AccessList {
	l := types.AccessList{
		types.AccessTuple{
			Address: opts.SequencerInboxAddr,
			StorageKeys: []common.Hash{
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), // totalDelayedMessagesRead
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), // bridge
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"), // maxTimeVariation.delayBlocks
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000005"), // maxTimeVariation.futureBlocks
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000006"), // maxTimeVariation.delaySeconds
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000007"), // maxTimeVariation.futureSeconds
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

	for _, v := range []struct{ slotIdx, val int }{
		{7, opts.SequencerInboxAccs - 1},       // - sequencerInboxAccs[sequencerInboxAccs.length - 1]; (keccak256(7, sequencerInboxAccs.length - 1))
		{7, opts.SequencerInboxAccs},           // - sequencerInboxAccs.push(...); (keccak256(7, sequencerInboxAccs.length))
		{6, opts.AfterDelayedMessagesRead - 1}, // - delayedInboxAccs[afterDelayedMessagesRead - 1]; (keccak256(6, afterDelayedMessagesRead - 1))
	} {
		sb := arbutil.SumBytes(arbutil.PaddedKeccak256([]byte{byte(v.slotIdx)}), big.NewInt(int64(v.val)).Bytes())
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

// checkRevert checks blocks with number in range [from, to] whether they
// contain reverted batch_poster transaction.
// It returns true if it finds batch posting needs to halt, which is true if a batch reverts
// unless the data poster is configured with noop storage which can tolerate reverts.
func (b *BatchPoster) checkReverts(ctx context.Context, to int64) (bool, error) {
	if b.nextRevertCheckBlock > to {
		return false, fmt.Errorf("wrong range, from: %d > to: %d", b.nextRevertCheckBlock, to)
	}
	for ; b.nextRevertCheckBlock <= to; b.nextRevertCheckBlock++ {
		number := big.NewInt(b.nextRevertCheckBlock)
		block, err := b.l1Reader.Client().BlockByNumber(ctx, number)
		if err != nil {
			return false, fmt.Errorf("getting block: %v by number: %w", number, err)
		}
		for idx, tx := range block.Transactions() {
			from, err := b.l1Reader.Client().TransactionSender(ctx, tx, block.Hash(), uint(idx))
			if err != nil {
				return false, fmt.Errorf("getting sender of transaction tx: %v, %w", tx.Hash(), err)
			}
			if from == b.dataPoster.Sender() {
				r, err := b.l1Reader.Client().TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					return false, fmt.Errorf("getting a receipt for transaction: %v, %w", tx.Hash(), err)
				}
				if r.Status == types.ReceiptStatusFailed {
					shouldHalt := !b.config().DataPoster.UseNoOpStorage
					logLevel := log.Warn
					if shouldHalt {
						logLevel = log.Error
					}
					logLevel("Transaction from batch poster reverted", "nonce", tx.Nonce(), "txHash", tx.Hash(), "blockNumber", r.BlockNumber, "blockHash", r.BlockHash)
					return shouldHalt, nil
				}
			}
		}
	}
	return false, nil
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
	segments          *batchSegments
	startMsgCount     arbutil.MessageIndex
	msgCount          arbutil.MessageIndex
	haveUsefulMessage bool
}

func newBatchSegments(firstDelayed uint64, config *BatchPosterConfig, backlog uint64) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, config.MaxSize*2))
	if config.MaxSize <= 40 {
		panic("MaxBatchSize too small")
	}
	compressionLevel := config.CompressionLevel
	recompressionLevel := config.CompressionLevel
	if backlog > 20 {
		compressionLevel = arbmath.MinInt(compressionLevel, brotli.DefaultCompression)
	}
	if backlog > 40 {
		recompressionLevel = arbmath.MinInt(recompressionLevel, brotli.DefaultCompression)
	}
	if backlog > 60 {
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
		sizeLimit:          config.MaxSize - 40, // TODO
		recompressionLevel: recompressionLevel,
		rawSegments:        make([][]byte, 0, 128),
		delayedMsg:         firstDelayed,
	}
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
		return true, nil
	}
	// we've reached the max number of segments
	if len(s.rawSegments) >= arbstate.MaxSegmentsPerSequencerMessage {
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
	fullMsg[0] = arbstate.BrotliMessageHeaderByte
	fullMsg = append(fullMsg, compressedBytes...)
	return fullMsg, nil
}

func (b *BatchPoster) encodeAddBatch(seqNum *big.Int, prevMsgNum arbutil.MessageIndex, newMsgNum arbutil.MessageIndex, message []byte, delayedMsg uint64) ([]byte, error) {
	method, ok := b.seqInboxABI.Methods["addSequencerL2BatchFromOrigin0"]
	if !ok {
		return nil, errors.New("failed to find add batch method")
	}
	inputData, err := method.Inputs.Pack(
		seqNum,
		message,
		new(big.Int).SetUint64(delayedMsg),
		b.config().gasRefunder,
		new(big.Int).SetUint64(uint64(prevMsgNum)),
		new(big.Int).SetUint64(uint64(newMsgNum)),
	)
	if err != nil {
		return nil, err
	}
	fullData := append([]byte{}, method.ID...)
	fullData = append(fullData, inputData...)
	return fullData, nil
}

var ErrNormalGasEstimationFailed = errors.New("normal gas estimation failed")

func (b *BatchPoster) estimateGas(ctx context.Context, sequencerMessage []byte, delayedMessages uint64, realData []byte, realNonce uint64, realAccessList types.AccessList) (uint64, error) {
	config := b.config()
	useNormalEstimation := b.dataPoster.MaxMempoolTransactions() == 1
	if !useNormalEstimation {
		// Check if we can use normal estimation anyways because we're at the latest nonce
		latestNonce, err := b.l1Reader.Client().NonceAt(ctx, b.dataPoster.Sender(), nil)
		if err != nil {
			return 0, err
		}
		useNormalEstimation = latestNonce == realNonce
	}
	if useNormalEstimation {
		// If we're at the latest nonce, we can skip the special future tx estimate stuff
		gas, err := b.l1Reader.Client().EstimateGas(ctx, ethereum.CallMsg{
			From:       b.dataPoster.Sender(),
			To:         &b.seqInboxAddr,
			Data:       realData,
			AccessList: realAccessList,
		})
		if err != nil {
			return 0, fmt.Errorf("%w: %w", ErrNormalGasEstimationFailed, err)
		}
		return gas + config.ExtraBatchGas, nil
	}

	// Here we set seqNum to MaxUint256, and prevMsgNum to 0, because it disables the smart contracts' consistency checks.
	// However, we set nextMsgNum to 1 because it is necessary for a correct estimation for the final to be non-zero.
	// Because we're likely estimating against older state, this might not be the actual next message,
	// but the gas used should be the same.
	data, err := b.encodeAddBatch(abi.MaxUint256, 0, 1, sequencerMessage, delayedMessages)
	if err != nil {
		return 0, err
	}
	gas, err := b.l1Reader.Client().EstimateGas(ctx, ethereum.CallMsg{
		From: b.dataPoster.Sender(),
		To:   &b.seqInboxAddr,
		Data: data,
		// This isn't perfect because we're probably estimating the batch at a different sequence number,
		// but it should overestimate rather than underestimate which is fine.
		AccessList: realAccessList,
	})
	if err != nil {
		sequencerMessageHeader := sequencerMessage
		if len(sequencerMessageHeader) > 33 {
			sequencerMessageHeader = sequencerMessageHeader[:33]
		}
		log.Warn(
			"error estimating gas for batch",
			"err", err,
			"delayedMessages", delayedMessages,
			"sequencerMessageHeader", hex.EncodeToString(sequencerMessageHeader),
			"sequencerMessageLen", len(sequencerMessage),
		)
		return 0, fmt.Errorf("error estimating gas for batch: %w", err)
	}
	return gas + config.ExtraBatchGas, nil
}

const ethPosBlockTime = 12 * time.Second

var errAttemptLockFailed = errors.New("failed to acquire lock; either another batch poster posted a batch or this node fell behind")

func (b *BatchPoster) maybePostSequencerBatch(ctx context.Context) (bool, error) {
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
		b.building = &buildingBatch{
			segments:      newBatchSegments(batchPosition.DelayedMessageCount, b.config(), b.GetBacklogEstimate()),
			msgCount:      batchPosition.MessageCount,
			startMsgCount: batchPosition.MessageCount,
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
	firstMsg, err := b.streamer.GetMessage(batchPosition.MessageCount)
	if err != nil {
		return false, err
	}
	firstMsgTime := time.Unix(int64(firstMsg.Message.Header.Timestamp), 0)

	lastPotentialMsg, err := b.streamer.GetMessage(msgCount - 1)
	if err != nil {
		return false, err
	}

	config := b.config()
	forcePostBatch := time.Since(firstMsgTime) >= config.MaxDelay

	var l1BoundMaxBlockNumber uint64 = math.MaxUint64
	var l1BoundMaxTimestamp uint64 = math.MaxUint64
	var l1BoundMinBlockNumber uint64
	var l1BoundMinTimestamp uint64
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

		if config.L1BlockBoundBypass > 0 {
			latestHeader, err := b.l1Reader.LastHeader(ctx)
			if err != nil {
				return false, err
			}
			latestBlockNumber := arbutil.ParentHeaderToL1BlockNumber(latestHeader)
			blockNumberWithPadding := arbmath.SaturatingUAdd(latestBlockNumber, uint64(config.L1BlockBoundBypass/ethPosBlockTime))
			timestampWithPadding := arbmath.SaturatingUAdd(latestHeader.Time, uint64(config.L1BlockBoundBypass/time.Second))

			l1BoundMinBlockNumber = arbmath.SaturatingUSub(blockNumberWithPadding, arbmath.BigToUintSaturating(maxTimeVariationDelayBlocks))
			l1BoundMinTimestamp = arbmath.SaturatingUSub(timestampWithPadding, arbmath.BigToUintSaturating(maxTimeVariationDelaySeconds))
		}
	}

	for b.building.msgCount < msgCount {
		msg, err := b.streamer.GetMessage(b.building.msgCount)
		if err != nil {
			log.Error("error getting message from streamer", "error", err)
			break
		}
		if msg.Message.Header.BlockNumber < l1BoundMinBlockNumber || msg.Message.Header.Timestamp < l1BoundMinTimestamp {
			log.Error(
				"disabling L1 bound as batch posting message is close to the maximum delay",
				"blockNumber", msg.Message.Header.BlockNumber,
				"l1BoundMinBlockNumber", l1BoundMinBlockNumber,
				"timestamp", msg.Message.Header.Timestamp,
				"l1BoundMinTimestamp", l1BoundMinTimestamp,
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
			break
		}
		if msg.Message.Header.Kind != arbostypes.L1MessageType_BatchPostingReport {
			b.building.haveUsefulMessage = true
		}
		b.building.msgCount++
	}

	if !forcePostBatch || !b.building.haveUsefulMessage {
		// the batch isn't full yet and we've posted a batch recently
		// don't post anything for now
		return false, nil
	}
	sequencerMsg, err := b.building.segments.CloseAndGetBytes()
	if err != nil {
		return false, err
	}
	if sequencerMsg == nil {
		log.Debug("BatchPoster: batch nil", "sequence nr.", batchPosition.NextSeqNum, "from", batchPosition.MessageCount, "prev delayed", batchPosition.DelayedMessageCount)
		b.building = nil // a closed batchSegments can't be reused
		return false, nil
	}

	if b.daWriter != nil {
		if !b.redisLock.AttemptLock(ctx) {
			return false, errAttemptLockFailed
		}

		gotNonce, gotMeta, err := b.dataPoster.GetNextNonceAndMeta(ctx)
		if err != nil {
			return false, err
		}
		if nonce != gotNonce || !bytes.Equal(batchPositionBytes, gotMeta) {
			return false, fmt.Errorf("%w: nonce changed from %d to %d while creating batch", storage.ErrStorageRace, nonce, gotNonce)
		}

		cert, err := b.daWriter.Store(ctx, sequencerMsg, uint64(time.Now().Add(config.DASRetentionPeriod).Unix()), []byte{}) // b.daWriter will append signature if enabled
		if errors.Is(err, das.BatchToDasFailed) {
			if config.DisableDasFallbackStoreDataOnChain {
				return false, errors.New("unable to batch to DAS and fallback storing data on chain is disabled")
			}
			log.Warn("Falling back to storing data on chain", "err", err)
		} else if err != nil {
			return false, err
		} else {
			sequencerMsg = das.Serialize(cert)
		}
	}

	data, err := b.encodeAddBatch(new(big.Int).SetUint64(batchPosition.NextSeqNum), batchPosition.MessageCount, b.building.msgCount, sequencerMsg, b.building.segments.delayedMsg)
	if err != nil {
		return false, err
	}
	accessList := b.accessList(int(batchPosition.NextSeqNum), int(b.building.segments.delayedMsg))
	// On restart, we may be trying to estimate gas for a batch whose successor has
	// already made it into pending state, if not latest state.
	// In that case, we might get a revert with `DelayedBackwards()`.
	// To avoid that, we artificially increase the delayed messages to `lastPotentialMsg.DelayedMessagesRead`.
	// In theory, this might reduce gas usage, but only by a factor that's already
	// accounted for in `config.ExtraBatchGas`, as that same factor can appear if a user
	// posts a new delayed message that we didn't see while gas estimating.
	gasLimit, err := b.estimateGas(ctx, sequencerMsg, lastPotentialMsg.DelayedMessagesRead, data, nonce, accessList)
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
	tx, err := b.dataPoster.PostTransaction(ctx,
		firstMsgTime,
		nonce,
		newMeta,
		b.seqInboxAddr,
		data,
		gasLimit,
		new(big.Int),
		accessList,
	)
	if err != nil {
		return false, err
	}
	log.Info(
		"BatchPoster: batch sent",
		"sequence nr.", batchPosition.NextSeqNum,
		"from", batchPosition.MessageCount,
		"to", b.building.msgCount,
		"prev delayed", batchPosition.DelayedMessageCount,
		"current delayed", b.building.segments.delayedMsg,
		"total segments", len(b.building.segments.rawSegments),
	)
	recentlyHitL1Bounds := time.Since(b.lastHitL1Bounds) < config.PollInterval*3
	postedMessages := b.building.msgCount - batchPosition.MessageCount
	b.messagesPerBatch.Update(uint64(postedMessages))
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
	atomic.StoreUint64(&b.backlog, backlog)
	b.building = nil

	// If we aren't queueing up transactions, wait for the receipt before moving on to the next batch.
	if config.DataPoster.UseNoOpStorage {
		receipt, err := b.l1Reader.WaitForTxApproval(ctx, tx)
		if err != nil {
			return false, fmt.Errorf("error waiting for tx receipt: %w", err)
		}
		log.Info("Got successful receipt from batch poster transaction", "txHash", tx.Hash(), "blockNumber", receipt.BlockNumber, "blockHash", receipt.BlockHash)
	}

	return true, nil
}

func (b *BatchPoster) GetBacklogEstimate() uint64 {
	return atomic.LoadUint64(&b.backlog)
}

func (b *BatchPoster) Start(ctxIn context.Context) {
	b.dataPoster.Start(ctxIn)
	b.redisLock.Start(ctxIn)
	b.StopWaiter.Start(ctxIn, b)
	b.LaunchThread(b.pollForReverts)
	commonEphemeralErrorHandler := util.NewEphemeralErrorHandler(time.Minute, "", 0)
	exceedMaxMempoolSizeEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, dataposter.ErrExceedsMaxMempoolSize.Error(), time.Minute)
	storageRaceEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, storage.ErrStorageRace.Error(), time.Minute)
	normalGasEstimationFailedEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, ErrNormalGasEstimationFailed.Error(), time.Minute)
	accumulatorNotFoundEphemeralErrorHandler := util.NewEphemeralErrorHandler(5*time.Minute, AccumulatorNotFoundErr.Error(), time.Minute)
	resetAllEphemeralErrs := func() {
		commonEphemeralErrorHandler.Reset()
		exceedMaxMempoolSizeEphemeralErrorHandler.Reset()
		storageRaceEphemeralErrorHandler.Reset()
		normalGasEstimationFailedEphemeralErrorHandler.Reset()
		accumulatorNotFoundEphemeralErrorHandler.Reset()
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
		posted, err := b.maybePostSequencerBatch(ctx)
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
			logLevel("error posting batch", "err", err)
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
