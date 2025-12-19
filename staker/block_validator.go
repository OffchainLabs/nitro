// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package staker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/resourcemanager"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client/redis"
	"github.com/offchainlabs/nitro/validator/inputs"
	"github.com/offchainlabs/nitro/validator/retry_wrapper"
)

var (
	validatorPendingValidationsGauge         = metrics.NewRegisteredGauge("arb/validator/validations/pending", nil)
	validatorValidValidationsCounter         = metrics.NewRegisteredCounter("arb/validator/validations/valid", nil)
	validatorFailedValidationsCounter        = metrics.NewRegisteredCounter("arb/validator/validations/failed", nil)
	validatorProfileWaitToRecordHist         = metrics.NewRegisteredHistogram("arb/validator/profile/wait_to_record", nil, metrics.NewBoundedHistogramSample())
	validatorProfileRecordingHist            = metrics.NewRegisteredHistogram("arb/validator/profile/recording", nil, metrics.NewBoundedHistogramSample())
	validatorProfileWaitToLaunchHist         = metrics.NewRegisteredHistogram("arb/validator/profile/wait_to_launch", nil, metrics.NewBoundedHistogramSample())
	validatorProfileLaunchingHist            = metrics.NewRegisteredHistogram("arb/validator/profile/launching", nil, metrics.NewBoundedHistogramSample())
	validatorProfileRunningHist              = metrics.NewRegisteredHistogram("arb/validator/profile/running", nil, metrics.NewBoundedHistogramSample())
	validatorMsgCountCurrentBatch            = metrics.NewRegisteredGauge("arb/validator/msg_count_current_batch", nil)
	validatorMsgCountCreatedGauge            = metrics.NewRegisteredGauge("arb/validator/msg_count_created", nil)
	validatorMsgCountRecordSentGauge         = metrics.NewRegisteredGauge("arb/validator/msg_count_record_sent", nil)
	validatorMsgCountValidatedGauge          = metrics.NewRegisteredGauge("arb/validator/msg_count_validated", nil)
	validatorMsgCountLastValidationSentGauge = metrics.NewRegisteredGauge("arb/validator/msg_count_last_validation_sent", nil)
)

// WorkerThrottler tracks concurrent validation executions for a spawner
// Uses simple atomic counter - no retry logic, just increment/decrement
type WorkerThrottler struct {
	maxWorkers     int
	currentRunning atomic.Int64
}

// HasCapacity checks if there's available capacity
func (t *WorkerThrottler) HasCapacity() bool {
	return t.currentRunning.Load() < int64(t.maxWorkers)
}

func (t *WorkerThrottler) Acquire() {
	t.currentRunning.Add(1)
}

func (t *WorkerThrottler) Release() {
	t.currentRunning.Add(-1)
}

type ThrottledValidationSpawner struct {
	Spawner   validator.ValidationSpawner
	Throttler *WorkerThrottler
}

func NewThrottledValidationSpawner(spawner validator.ValidationSpawner) *ThrottledValidationSpawner {
	return &ThrottledValidationSpawner{
		Spawner:   spawner,
		Throttler: &WorkerThrottler{maxWorkers: spawner.Capacity()},
	}
}

type BlockValidator struct {
	stopwaiter.StopWaiter
	*StatelessBlockValidator

	reorgMutex sync.RWMutex

	chainCaughtUp bool

	// can only be accessed from creation thread or if holding reorg-write
	nextCreateBatch       *FullBatchInfo
	nextCreateBatchReread bool
	prevBatchCache        map[uint64][]byte

	nextCreateStartGS     validator.GoGlobalState
	nextCreatePrevDelayed uint64

	// can only be accessed from validation thread or if holding reorg-write
	lastValidGS     validator.GoGlobalState
	legacyValidInfo *legacyLastBlockValidatedDbInfo

	// only from logger thread
	lastValidInfoPrinted *GlobalStateValidatedInfo

	// can be read (atomic.Load) by anyone holding reorg-read
	// written (atomic.Set) by appropriate thread or (any way) holding reorg-write
	createdA            atomic.Uint64
	recordSentA         atomic.Uint64
	validatedA          atomic.Uint64
	lastValidationSentA atomic.Uint64
	validations         containers.SyncMap[arbutil.MessageIndex, *validationStatus]

	config BlockValidatorConfigFetcher

	createNodesChan         chan struct{}
	sendRecordChan          chan struct{}
	sendValidationsChan     chan struct{}
	progressValidationsChan chan struct{}

	chosenValidator map[common.Hash]*ThrottledValidationSpawner

	// wasmModuleRoot
	moduleMutex           sync.Mutex
	currentWasmModuleRoot common.Hash
	pendingWasmModuleRoot common.Hash

	// for testing only
	testingProgressMadeChan  chan struct{}
	testingProgressMadeMutex sync.Mutex

	// For troubleshooting failed validations
	validationInputsWriter *inputs.Writer

	fatalErr chan<- error

	MemoryFreeLimitChecker resourcemanager.LimitChecker
}

type BlockValidatorConfig struct {
	Enable                            bool                          `koanf:"enable"`
	RedisValidationClientConfig       redis.ValidationClientConfig  `koanf:"redis-validation-client-config"`
	ValidationServer                  rpcclient.ClientConfig        `koanf:"validation-server" reload:"hot"`
	ValidationServerConfigs           []rpcclient.ClientConfig      `koanf:"validation-server-configs"`
	ValidationPoll                    time.Duration                 `koanf:"validation-poll" reload:"hot"`
	PrerecordedBlocks                 uint64                        `koanf:"prerecorded-blocks" reload:"hot"`
	RecordingIterLimit                uint64                        `koanf:"recording-iter-limit"`
	ValidationSentLimit               uint64                        `koanf:"validation-sent-limit"`
	ForwardBlocks                     uint64                        `koanf:"forward-blocks" reload:"hot"`
	BatchCacheLimit                   uint32                        `koanf:"batch-cache-limit"`
	CurrentModuleRoot                 string                        `koanf:"current-module-root"`         // TODO(magic) requires reinitialization on hot reload
	PendingUpgradeModuleRoot          string                        `koanf:"pending-upgrade-module-root"` // TODO(magic) requires StatelessBlockValidator recreation on hot reload
	FailureIsFatal                    bool                          `koanf:"failure-is-fatal" reload:"hot"`
	Dangerous                         BlockValidatorDangerousConfig `koanf:"dangerous"`
	MemoryFreeLimit                   string                        `koanf:"memory-free-limit" reload:"hot"`
	ValidationServerConfigsList       string                        `koanf:"validation-server-configs-list"`
	ValidationSpawningAllowedAttempts uint64                        `koanf:"validation-spawning-allowed-attempts" reload:"hot"`
	// The directory to which the BlockValidator will write the
	// block_inputs_<id>.json files when WriteToFile() is called.
	BlockInputsFilePath string `koanf:"block-inputs-file-path"`

	memoryFreeLimit int
}

func (c *BlockValidatorConfig) Validate() error {
	if c.MemoryFreeLimit == "default" {
		c.memoryFreeLimit = 1073741824 // 1GB
	} else if c.MemoryFreeLimit != "" {
		limit, err := resourcemanager.ParseMemLimit(c.MemoryFreeLimit)
		if err != nil {
			return fmt.Errorf("failed to parse block-validator config memory-free-limit string: %w", err)
		}
		c.memoryFreeLimit = limit
	}
	if err := c.RedisValidationClientConfig.Validate(); err != nil {
		return fmt.Errorf("failed to validate redis validation client config: %w", err)
	}
	streamsEnabled := c.RedisValidationClientConfig.Enabled()
	if len(c.ValidationServerConfigs) == 0 {
		c.ValidationServerConfigs = []rpcclient.ClientConfig{c.ValidationServer}
		if c.ValidationServerConfigsList != "default" {
			var executionServersConfigs []rpcclient.ClientConfig
			if err := json.Unmarshal([]byte(c.ValidationServerConfigsList), &executionServersConfigs); err != nil && !streamsEnabled {
				return fmt.Errorf("failed to parse block-validator validation-server-configs-list string: %w", err)
			}
			c.ValidationServerConfigs = executionServersConfigs
		}
	}
	for i := range c.ValidationServerConfigs {
		if err := c.ValidationServerConfigs[i].Validate(); err != nil {
			return fmt.Errorf("failed to validate one of the block-validator validation-server-configs. url: %s, err: %w", c.ValidationServerConfigs[i].URL, err)
		}
		serverUrl := c.ValidationServerConfigs[i].URL
		if len(serverUrl) > 0 && serverUrl != "self" && serverUrl != "self-auth" {
			u, err := url.Parse(serverUrl)
			if err != nil {
				return fmt.Errorf("failed parsing validation server's url:%s err: %w", serverUrl, err)
			}
			if u.Scheme != "ws" && u.Scheme != "wss" {
				return fmt.Errorf("validation server's url scheme is unsupported, it should either be ws or wss, url:%s", serverUrl)
			}
		}
	}
	if c.Dangerous.Revalidation.EndBlock > 0 && c.Dangerous.Revalidation.EndBlock < c.Dangerous.Revalidation.StartBlock {
		return fmt.Errorf("revalidation end block %d is before start block %d", c.Dangerous.Revalidation.EndBlock, c.Dangerous.Revalidation.StartBlock)
	}
	return nil
}

type BlockValidatorDangerousConfig struct {
	ResetBlockValidation bool               `koanf:"reset-block-validation"`
	Revalidation         RevalidationConfig `koanf:"revalidation"`
}

type RevalidationConfig struct {
	StartBlock            uint64 `koanf:"start-block"`
	EndBlock              uint64 `koanf:"end-block"`
	QuitAfterRevalidation bool   `koanf:"quit-after-revalidation"`
}

type BlockValidatorConfigFetcher func() *BlockValidatorConfig

func BlockValidatorConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockValidatorConfig.Enable, "enable block-by-block validation")
	rpcclient.RPCClientAddOptions(prefix+".validation-server", f, &DefaultBlockValidatorConfig.ValidationServer)
	redis.ValidationClientConfigAddOptions(prefix+".redis-validation-client-config", f)
	f.String(prefix+".validation-server-configs-list", DefaultBlockValidatorConfig.ValidationServerConfigsList, "array of execution rpc configs given as a json string. time duration should be supplied in number indicating nanoseconds")
	f.Duration(prefix+".validation-poll", DefaultBlockValidatorConfig.ValidationPoll, "poll time to check validations")
	f.Uint64(prefix+".forward-blocks", DefaultBlockValidatorConfig.ForwardBlocks, "prepare entries for up to that many blocks ahead of validation (stores batch-copy per block)")
	f.Uint64(prefix+".prerecorded-blocks", DefaultBlockValidatorConfig.PrerecordedBlocks, "record that many blocks ahead of validation (larger footprint)")
	f.Uint32(prefix+".batch-cache-limit", DefaultBlockValidatorConfig.BatchCacheLimit, "limit number of old batches to keep in block-validator")
	f.String(prefix+".current-module-root", DefaultBlockValidatorConfig.CurrentModuleRoot, "current wasm module root ('current' read from chain, 'latest' from machines/latest dir, or provide hash)")
	f.Uint64(prefix+".recording-iter-limit", DefaultBlockValidatorConfig.RecordingIterLimit, "limit on block recordings sent per iteration")
	f.Uint64(prefix+".validation-sent-limit", DefaultBlockValidatorConfig.ValidationSentLimit, "limit on block validations to keep in validation sent state")
	f.String(prefix+".pending-upgrade-module-root", DefaultBlockValidatorConfig.PendingUpgradeModuleRoot, "pending upgrade wasm module root to additionally validate (hash, 'latest' or empty)")
	f.Bool(prefix+".failure-is-fatal", DefaultBlockValidatorConfig.FailureIsFatal, "failing a validation is treated as a fatal error")
	BlockValidatorDangerousConfigAddOptions(prefix+".dangerous", f)
	f.String(prefix+".memory-free-limit", DefaultBlockValidatorConfig.MemoryFreeLimit, "minimum free-memory limit after reaching which the blockvalidator pauses validation. Enabled by default as 1GB, to disable provide empty string")
	f.String(prefix+".block-inputs-file-path", DefaultBlockValidatorConfig.BlockInputsFilePath, "directory to write block validation inputs files")
	f.Uint64(prefix+".validation-spawning-allowed-attempts", DefaultBlockValidatorConfig.ValidationSpawningAllowedAttempts, "number of attempts allowed when trying to spawn a validation before erroring out")
}

func BlockValidatorDangerousConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".reset-block-validation", DefaultBlockValidatorDangerousConfig.ResetBlockValidation, "resets block-by-block validation, starting again at genesis")
	RevalidationConfigAddOptions(prefix+".revalidation", f)
}

func RevalidationConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint64(prefix+".start-block", DefaultBlockValidatorDangerousConfig.Revalidation.StartBlock, "start revalidation from this block")
	f.Uint64(prefix+".end-block", DefaultBlockValidatorDangerousConfig.Revalidation.EndBlock, "end revalidation at this block")
	f.Bool(prefix+".quit-after-revalidation", DefaultBlockValidatorDangerousConfig.Revalidation.QuitAfterRevalidation, "exit node after revalidation is done")
}

var DefaultBlockValidatorConfig = BlockValidatorConfig{
	Enable:                            false,
	ValidationServerConfigsList:       "default",
	ValidationServer:                  rpcclient.DefaultClientConfig,
	RedisValidationClientConfig:       redis.DefaultValidationClientConfig,
	ValidationPoll:                    time.Second,
	ForwardBlocks:                     128,
	PrerecordedBlocks:                 uint64(2 * util.GoMaxProcs()),
	BatchCacheLimit:                   20,
	CurrentModuleRoot:                 "current",
	PendingUpgradeModuleRoot:          "latest",
	FailureIsFatal:                    true,
	Dangerous:                         DefaultBlockValidatorDangerousConfig,
	BlockInputsFilePath:               "./target/validation_inputs",
	MemoryFreeLimit:                   "default",
	RecordingIterLimit:                20,
	ValidationSentLimit:               1024,
	ValidationSpawningAllowedAttempts: 1,
}

var TestBlockValidatorConfig = BlockValidatorConfig{
	Enable:                            false,
	ValidationServer:                  rpcclient.TestClientConfig,
	ValidationServerConfigs:           []rpcclient.ClientConfig{rpcclient.TestClientConfig},
	RedisValidationClientConfig:       redis.TestValidationClientConfig,
	ValidationPoll:                    100 * time.Millisecond,
	ForwardBlocks:                     128,
	BatchCacheLimit:                   20,
	PrerecordedBlocks:                 uint64(2 * util.GoMaxProcs()),
	RecordingIterLimit:                20,
	ValidationSentLimit:               1024,
	CurrentModuleRoot:                 "latest",
	PendingUpgradeModuleRoot:          "latest",
	FailureIsFatal:                    true,
	Dangerous:                         DefaultBlockValidatorDangerousConfig,
	BlockInputsFilePath:               "./target/validation_inputs",
	MemoryFreeLimit:                   "default",
	ValidationSpawningAllowedAttempts: 1,
}

var DefaultBlockValidatorDangerousConfig = BlockValidatorDangerousConfig{
	ResetBlockValidation: false,
	Revalidation:         DefaultRevalidationConfig,
}

var DefaultRevalidationConfig = RevalidationConfig{
	StartBlock:            0,
	EndBlock:              0,
	QuitAfterRevalidation: false,
}

type valStatusField uint32

const (
	Created valStatusField = iota
	RecordSent
	RecordFailed
	Prepared
	SendingValidation
	ValidationDone
)

type validationStatus struct {
	Status    atomic.Uint32        // atomic: value is one of validationStatus*
	Cancel    func()               // non-atomic: only read/written to with reorg mutex
	Entry     *validationEntry     // non-atomic: only read if Status >= validationStatusPrepared
	DoneEntry *validationDoneEntry // non-atomic: only read if status == ValidationDone
	profileTS int64                // time-stamp for profiling
}

type validationDoneEntry struct {
	Success         bool
	Start           validator.GoGlobalState
	End             validator.GoGlobalState
	WasmModuleRoots []common.Hash
}

func (s *validationStatus) getStatus() valStatusField {
	uintStat := s.Status.Load()
	return valStatusField(uintStat)
}

func (s *validationStatus) replaceStatus(old, new valStatusField) bool {
	return s.Status.CompareAndSwap(uint32(old), uint32(new))
}

// gets how many milliseconds last step took, and starts measuring a new step
func (s *validationStatus) profileStep() int64 {
	start := s.profileTS
	s.profileTS = time.Now().UnixMilli()
	return s.profileTS - start
}

func NewBlockValidator(
	statelessBlockValidator *StatelessBlockValidator,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	config BlockValidatorConfigFetcher,
	fatalErr chan<- error,
) (*BlockValidator, error) {
	ret := &BlockValidator{
		StatelessBlockValidator: statelessBlockValidator,
		createNodesChan:         make(chan struct{}, 1),
		sendRecordChan:          make(chan struct{}, 1),
		sendValidationsChan:     make(chan struct{}, 1),
		progressValidationsChan: make(chan struct{}, 1),
		config:                  config,
		fatalErr:                fatalErr,
		prevBatchCache:          make(map[uint64][]byte),
	}
	valInputsWriter, err := inputs.NewWriter(
		inputs.WithBaseDir(ret.stack.InstanceDir()),
		inputs.WithSlug("BlockValidator"))
	if err != nil {
		return nil, err
	}
	ret.validationInputsWriter = valInputsWriter
	if !config().Dangerous.ResetBlockValidation {
		validated, err := ret.ReadLastValidatedInfo()
		if err != nil {
			return nil, err
		}
		if validated != nil {
			ret.lastValidGS = validated.GlobalState
		} else {
			legacyInfo, err := ret.legacyReadLastValidatedInfo()
			if err != nil {
				return nil, err
			}
			ret.legacyValidInfo = legacyInfo
		}
	}
	// genesis block is impossible to validate unless genesis state is empty
	if ret.lastValidGS.Batch == 0 && ret.legacyValidInfo == nil {
		genesis, err := streamer.ResultAtMessageIndex(0)
		if err != nil {
			return nil, err
		}
		ret.lastValidGS = validator.GoGlobalState{
			BlockHash:  genesis.BlockHash,
			SendRoot:   genesis.SendRoot,
			Batch:      1,
			PosInBatch: 0,
		}
	}
	if config().Dangerous.Revalidation.StartBlock > 0 {
		startBlock := config().Dangerous.Revalidation.StartBlock
		messageCount, err := inbox.GetBatchMessageCount(startBlock - 1)
		if err != nil {
			return nil, err
		}
		res := &execution.MessageResult{}
		if messageCount > 0 {
			res, err = streamer.ResultAtMessageIndex(messageCount - 1)
			if err != nil {
				return nil, err
			}
		}
		_, endPos, err := statelessBlockValidator.GlobalStatePositionsAtCount(messageCount)
		if err != nil {
			return nil, err
		}
		gs := BuildGlobalState(*res, endPos)
		err = ret.writeLastValidated(gs, nil)
		if err != nil {
			return nil, err
		}
	}
	streamer.SetBlockValidator(ret)
	inbox.SetBlockValidator(ret)
	if config().MemoryFreeLimit != "" {
		limitchecker, err := resourcemanager.NewCgroupsMemoryLimitCheckerIfSupported(config().memoryFreeLimit)
		if err != nil {
			if config().MemoryFreeLimit == "default" {
				log.Warn("Cgroups V1 or V2 is unsupported, memory-free-limit feature inside block-validator is disabled")
			} else {
				return nil, fmt.Errorf("failed to create MemoryFreeLimitChecker, Cgroups V1 or V2 is unsupported")
			}
		} else {
			ret.MemoryFreeLimitChecker = limitchecker
		}
	}
	return ret, nil
}

func atomicStorePos(addr *atomic.Uint64, val arbutil.MessageIndex, metr *metrics.Gauge) {
	addr.Store(uint64(val))
	// #nosec G115
	metr.Update(int64(val))
}

func atomicLoadPos(addr *atomic.Uint64) arbutil.MessageIndex {
	return arbutil.MessageIndex(addr.Load())
}

func (v *BlockValidator) created() arbutil.MessageIndex {
	return atomicLoadPos(&v.createdA)
}

func (v *BlockValidator) recordSent() arbutil.MessageIndex {
	return atomicLoadPos(&v.recordSentA)
}

func (v *BlockValidator) validated() arbutil.MessageIndex {
	return atomicLoadPos(&v.validatedA)
}

func (v *BlockValidator) lastValidationSent() arbutil.MessageIndex {
	return atomicLoadPos(&v.lastValidationSentA)
}

func (v *BlockValidator) Validated(t *testing.T) arbutil.MessageIndex {
	return v.validated()
}

func (v *BlockValidator) possiblyFatal(err error) {
	if v.Stopped() {
		return
	}
	if err == nil {
		return
	}
	log.Error("Error during validation", "err", err)
	if v.config().FailureIsFatal {
		select {
		case v.fatalErr <- err:
		default:
		}
	}
}

func nonBlockingTrigger(channel chan struct{}) {
	select {
	case channel <- struct{}{}:
	default:
	}
}

func (v *BlockValidator) GetModuleRootsToValidate() []common.Hash {
	v.moduleMutex.Lock()
	defer v.moduleMutex.Unlock()

	validatingModuleRoots := []common.Hash{v.currentWasmModuleRoot}
	if v.currentWasmModuleRoot != v.pendingWasmModuleRoot && v.pendingWasmModuleRoot != (common.Hash{}) {
		validatingModuleRoots = append(validatingModuleRoots, v.pendingWasmModuleRoot)
	}
	return validatingModuleRoots
}

// called from NewBlockValidator, doesn't need to catch locks
func ReadLastValidatedInfo(db ethdb.Database) (*GlobalStateValidatedInfo, error) {
	exists, err := db.Has(lastGlobalStateValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	var validated GlobalStateValidatedInfo
	if !exists {
		return nil, nil
	}
	gsBytes, err := db.Get(lastGlobalStateValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	err = rlp.DecodeBytes(gsBytes, &validated)
	if err != nil {
		return nil, err
	}
	return &validated, nil
}

func (v *BlockValidator) ReadLastValidatedInfo() (*GlobalStateValidatedInfo, error) {
	return ReadLastValidatedInfo(v.db)
}

func (v *BlockValidator) legacyReadLastValidatedInfo() (*legacyLastBlockValidatedDbInfo, error) {
	exists, err := v.db.Has(legacyLastBlockValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	var validated legacyLastBlockValidatedDbInfo
	if !exists {
		return nil, nil
	}
	gsBytes, err := v.db.Get(legacyLastBlockValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	err = rlp.DecodeBytes(gsBytes, &validated)
	if err != nil {
		return nil, err
	}
	return &validated, nil
}

var ErrGlobalStateNotInChain = errors.New("globalstate not in chain")

// false if chain not caught up to globalstate
// error is ErrGlobalStateNotInChain if globalstate not in chain (and chain caught up)
func GlobalStateToMsgCount(tracker InboxTrackerInterface, streamer TransactionStreamerInterface, gs validator.GoGlobalState) (bool, arbutil.MessageIndex, error) {
	batchCount, err := tracker.GetBatchCount()
	if err != nil {
		return false, 0, err
	}
	requiredBatchCount := gs.Batch + 1
	if gs.PosInBatch == 0 {
		requiredBatchCount -= 1
	}
	if batchCount < requiredBatchCount {
		return false, 0, nil
	}
	var prevBatchMsgCount arbutil.MessageIndex
	if gs.Batch > 0 {
		prevBatchMsgCount, err = tracker.GetBatchMessageCount(gs.Batch - 1)
		if err != nil {
			return false, 0, err
		}
	}
	count := prevBatchMsgCount
	if gs.PosInBatch > 0 {
		curBatchMsgCount, err := tracker.GetBatchMessageCount(gs.Batch)
		if err != nil {
			return false, 0, fmt.Errorf("%w: getBatchMsgCount %d batchCount %d", err, gs.Batch, batchCount)
		}
		count += arbutil.MessageIndex(gs.PosInBatch)
		if curBatchMsgCount < count {
			return false, 0, fmt.Errorf("%w: batch %d posInBatch %d, maxPosInBatch %d", ErrGlobalStateNotInChain, gs.Batch, gs.PosInBatch, curBatchMsgCount-prevBatchMsgCount)
		}
	}
	processed, err := streamer.GetProcessedMessageCount()
	if err != nil {
		return false, 0, err
	}
	if processed < count {
		return false, 0, nil
	}
	res := &execution.MessageResult{}
	if count > 0 {
		res, err = streamer.ResultAtMessageIndex(count - 1)
		if err != nil {
			return false, 0, err
		}
	}
	if res.BlockHash != gs.BlockHash || res.SendRoot != gs.SendRoot {
		return false, 0, fmt.Errorf("%w: count %d hash %v expected %v, sendroot %v expected %v", ErrGlobalStateNotInChain, count, gs.BlockHash, res.BlockHash, gs.SendRoot, res.SendRoot)
	}
	return true, count, nil
}

func (v *BlockValidator) sendRecord(s *validationStatus) error {
	if !v.Started() {
		return nil
	}
	if !s.replaceStatus(Created, RecordSent) {
		return fmt.Errorf("failed status check for send record. Status: %v", s.getStatus())
	}

	validatorProfileWaitToRecordHist.Update(s.profileStep())
	v.LaunchThread(func(ctx context.Context) {
		err := v.ValidationEntryRecord(ctx, s.Entry)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			s.replaceStatus(RecordSent, RecordFailed) // after that - could be removed from validations map
			log.Error("Error while recording", "err", err, "status", s.getStatus())
			return
		}
		validatorProfileRecordingHist.Update(s.profileStep())
		if !s.replaceStatus(RecordSent, Prepared) {
			log.Error("Fault trying to update validation with recording", "entry", s.Entry, "status", s.getStatus())
			return
		}
		nonBlockingTrigger(v.sendValidationsChan)
	})
	return nil
}

func (v *BlockValidator) SetCurrentWasmModuleRoot(hash common.Hash) error {
	v.moduleMutex.Lock()
	defer v.moduleMutex.Unlock()

	if (hash == common.Hash{}) {
		return errors.New("trying to set zero as wasmModuleRoot")
	}
	if hash == v.currentWasmModuleRoot {
		return nil
	}
	if (v.currentWasmModuleRoot == common.Hash{}) {
		v.currentWasmModuleRoot = hash
		return nil
	}
	if v.pendingWasmModuleRoot == hash {
		log.Info("Block validator: detected progressing to pending machine", "hash", hash)
		v.currentWasmModuleRoot = hash
		return nil
	}
	if v.config().CurrentModuleRoot != "current" {
		return nil
	}
	return fmt.Errorf(
		"unexpected wasmModuleRoot! cannot validate! found %v, current %v, pending %v",
		hash, v.currentWasmModuleRoot, v.pendingWasmModuleRoot,
	)
}

func (v *BlockValidator) createNextValidationEntry(ctx context.Context) (bool, error) {
	v.reorgMutex.RLock()
	defer v.reorgMutex.RUnlock()
	pos := v.created()
	if pos > v.recordSent()+arbutil.MessageIndex(v.config().ForwardBlocks) {
		log.Trace("create validation entry: nothing to do", "pos", pos, "validated", v.validated())
		return false, nil
	}
	streamerMsgCount, err := v.streamer.GetProcessedMessageCount()
	if err != nil {
		return false, err
	}
	if pos >= streamerMsgCount {
		log.Trace("create validation entry: nothing to do", "pos", pos, "streamerMsgCount", streamerMsgCount)
		return false, nil
	}
	msg, err := v.streamer.GetMessage(pos)
	if err != nil {
		return false, err
	}
	endRes, err := v.streamer.ResultAtMessageIndex(pos)
	if err != nil {
		return false, err
	}
	if v.nextCreateStartGS.PosInBatch == 0 || v.nextCreateBatchReread {
		// new batch
		found, fullBatchInfo, err := v.readFullBatch(ctx, v.nextCreateStartGS.Batch)
		if !found {
			return false, err
		}
		if v.nextCreateBatch != nil {
			v.prevBatchCache[v.nextCreateBatch.Number] = v.nextCreateBatch.PostedData
		}
		v.nextCreateBatch = fullBatchInfo
		// #nosec G115
		validatorMsgCountCurrentBatch.Update(int64(fullBatchInfo.MsgCount))
		batchCacheLimit := v.config().BatchCacheLimit
		if len(v.prevBatchCache) > int(batchCacheLimit) {
			for num := range v.prevBatchCache {
				if num+uint64(batchCacheLimit) < v.nextCreateStartGS.Batch {
					delete(v.prevBatchCache, num)
				}
			}
		}
		v.nextCreateBatchReread = false
	}
	endGS := validator.GoGlobalState{
		BlockHash: endRes.BlockHash,
		SendRoot:  endRes.SendRoot,
	}
	if pos+1 < v.nextCreateBatch.MsgCount {
		endGS.Batch = v.nextCreateStartGS.Batch
		endGS.PosInBatch = v.nextCreateStartGS.PosInBatch + 1
	} else if pos+1 == v.nextCreateBatch.MsgCount {
		endGS.Batch = v.nextCreateStartGS.Batch + 1
		endGS.PosInBatch = 0
	} else {
		return false, fmt.Errorf("illegal batch msg count %d pos %d batch %d", v.nextCreateBatch.MsgCount, pos, endGS.Batch)
	}
	chainConfig := v.streamer.ChainConfig()
	prevBatchNums, err := msg.Message.PastBatchesRequired()
	if err != nil {
		return false, err
	}
	prevBatches := make([]validator.BatchInfo, 0, len(prevBatchNums))
	// prevBatchNums are only used for batch reports, each is only used once
	for _, batchNum := range prevBatchNums {
		data, found := v.prevBatchCache[batchNum]
		if found {
			delete(v.prevBatchCache, batchNum)
		} else {
			data, err = v.readPostedBatch(ctx, batchNum)
			if err != nil {
				return false, err
			}
		}
		prevBatches = append(prevBatches, validator.BatchInfo{
			Number: batchNum,
			Data:   data,
		})
	}
	entry, err := newValidationEntry(
		pos, v.nextCreateStartGS, endGS, msg, v.nextCreateBatch, prevBatches, v.nextCreatePrevDelayed, chainConfig,
	)
	if err != nil {
		return false, err
	}
	status := &validationStatus{
		Entry:     entry,
		profileTS: time.Now().UnixMilli(),
	}
	status.Status.Store(uint32(Created))
	v.validations.Store(pos, status)
	v.nextCreateStartGS = endGS
	v.nextCreatePrevDelayed = msg.DelayedMessagesRead
	atomicStorePos(&v.createdA, pos+1, validatorMsgCountCreatedGauge)
	log.Trace("create validation entry: created", "pos", pos)
	return true, nil
}

func (v *BlockValidator) iterativeValidationEntryCreator(ctx context.Context, ignored struct{}) time.Duration {
	moreWork, err := v.createNextValidationEntry(ctx)
	if err != nil {
		processed, processedErr := v.streamer.GetProcessedMessageCount()
		log.Error("error trying to create validation node", "err", err, "created", v.created()+1, "processed", processed, "processedErr", processedErr)
	}
	if moreWork {
		return 0
	}
	return v.config().ValidationPoll
}

func (v *BlockValidator) isMemoryLimitExceeded() bool {
	if v.MemoryFreeLimitChecker == nil {
		return false
	}
	exceeded, err := v.MemoryFreeLimitChecker.IsLimitExceeded()
	if err != nil {
		log.Error("error checking if free-memory limit exceeded using MemoryFreeLimitChecker", "err", err)
	}
	return exceeded
}

func (v *BlockValidator) sendNextRecordRequests(ctx context.Context) (bool, error) {
	if v.isMemoryLimitExceeded() {
		log.Error("sendNextRecordRequests: aborting due to running low on memory")
		return false, nil
	}
	v.reorgMutex.RLock()
	pos := v.recordSent()
	created := v.created()
	validationSent := v.lastValidationSent()
	v.reorgMutex.RUnlock()

	recordUntil := validationSent + arbutil.MessageIndex(v.config().PrerecordedBlocks) - 1
	if recordUntil > created-1 {
		recordUntil = created - 1
	}
	if recordUntil < pos {
		return false, nil
	}
	recordUntilLimit := pos + arbutil.MessageIndex(v.config().RecordingIterLimit)
	if recordUntil > recordUntilLimit {
		recordUntil = recordUntilLimit
	}
	log.Trace("preparing to record", "pos", pos, "until", recordUntil)
	// prepare could take a long time so we do it without a lock
	err := v.recorder.PrepareForRecord(ctx, pos, recordUntil)
	if err != nil {
		return false, err
	}

	v.reorgMutex.RLock()
	defer v.reorgMutex.RUnlock()
	createdNew := v.created()
	recordSentNew := v.recordSent()
	if createdNew < created || recordSentNew < pos {
		// there was a relevant reorg - quit and restart
		return true, nil
	}
	for pos <= recordUntil {
		if v.isMemoryLimitExceeded() {
			log.Error("sendNextRecordRequests: aborting due to running low on memory")
			return false, nil
		}
		validationStatus, found := v.validations.Load(pos)
		if !found {
			return false, fmt.Errorf("not found entry for pos %d", pos)
		}
		currentStatus := validationStatus.getStatus()
		if currentStatus != Created {
			return false, fmt.Errorf("bad status trying to send recordings for pos %d status: %v", pos, currentStatus)
		}
		err := v.sendRecord(validationStatus)
		if err != nil {
			return false, err
		}
		pos += 1
		atomicStorePos(&v.recordSentA, pos, validatorMsgCountRecordSentGauge)
		log.Trace("next record request: sent", "pos", pos)
	}

	return true, nil
}

func (v *BlockValidator) iterativeValidationEntryRecorder(ctx context.Context, ignored struct{}) time.Duration {
	moreWork, err := v.sendNextRecordRequests(ctx)
	if err != nil {
		log.Error("error trying to record for validation node", "err", err)
	}
	if moreWork {
		return 0
	}
	return v.config().ValidationPoll
}

func (v *BlockValidator) iterativeValidationPrint(ctx context.Context) time.Duration {
	validated, err := v.ReadLastValidatedInfo()
	if err != nil {
		log.Error("cannot read last validated data from database", "err", err)
		return time.Second * 30
	}
	if validated == nil {
		return time.Second
	}
	if v.lastValidInfoPrinted != nil {
		if v.lastValidInfoPrinted.GlobalState.BlockHash == validated.GlobalState.BlockHash {
			return time.Second
		}
	}
	var batchMsgs arbutil.MessageIndex
	var printedCount int64
	if validated.GlobalState.Batch > 0 {
		batchMsgs, err = v.inboxTracker.GetBatchMessageCount(validated.GlobalState.Batch - 1)
	}
	if err != nil {
		printedCount = -1
	} else {
		// #nosec G115
		printedCount = int64(batchMsgs) + int64(validated.GlobalState.PosInBatch)
	}
	log.Info("validated execution", "messageCount", printedCount, "globalstate", validated.GlobalState, "WasmRoots", validated.WasmRoots)
	v.lastValidInfoPrinted = validated
	revalidationConfig := v.config().Dangerous.Revalidation
	if revalidationConfig.EndBlock > 0 && validated.GlobalState.Batch >= revalidationConfig.EndBlock {
		log.Info("revalidation done", "startBlock", revalidationConfig.StartBlock, "endBlock", revalidationConfig.EndBlock)
		if revalidationConfig.QuitAfterRevalidation {
			// Sending nil to fatalErr channel to stop the node, but not report as an error
			// since this is expected shutdown of the node.
			v.fatalErr <- nil
		} else {
			v.StopOnly()
		}
	}
	return time.Second
}

// return val:
// *MessageIndex - pointer to bad entry if there is one (requires reorg)
func (v *BlockValidator) advanceValidations(ctx context.Context) (*arbutil.MessageIndex, error) {
	v.reorgMutex.RLock()
	defer v.reorgMutex.RUnlock()

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		v.reorgMutex.RUnlock()
		v.reorgMutex.RLock()
		pos := v.validated()
		if pos >= v.recordSent() {
			log.Trace("advanceValidations: nothing to validate", "pos", pos)
			return nil, nil
		}
		validationStatus, found := v.validations.Load(pos)
		if !found {
			return nil, fmt.Errorf("not found entry for pos %d", pos)
		}
		if validationStatus.getStatus() != ValidationDone {
			log.Trace("advanceValidations: validation not done", "pos", pos, "status", validationStatus.getStatus())
			return nil, nil
		}
		if validationStatus.DoneEntry.Start != v.lastValidGS {
			log.Warn("Validation entry has wrong start state", "pos", pos, "start", validationStatus.DoneEntry.Start, "expected", v.lastValidGS)
			validationStatus.Cancel()
			return &pos, nil
		}
		if !validationStatus.DoneEntry.Success {
			v.possiblyFatal(fmt.Errorf("validation: failed entry pos %d, start %v", pos, validationStatus.DoneEntry.Start))
			return &pos, nil // if not fatal - retry
		}
		err := v.writeLastValidated(validationStatus.DoneEntry.End, validationStatus.DoneEntry.WasmModuleRoots)
		if err != nil {
			log.Error("failed writing new validated to database", "pos", pos, "err", err)
		}
		go v.recorder.MarkValid(pos, v.lastValidGS.BlockHash)
		atomicStorePos(&v.validatedA, pos+1, validatorMsgCountValidatedGauge)
		v.validations.Delete(pos)
		nonBlockingTrigger(v.createNodesChan)
		nonBlockingTrigger(v.sendRecordChan)
		nonBlockingTrigger(v.sendValidationsChan)
		v.testingProgressMadeMutex.Lock()
		if v.testingProgressMadeChan != nil {
			nonBlockingTrigger(v.testingProgressMadeChan)
		}
		v.testingProgressMadeMutex.Unlock()

		log.Trace("result validated", "count", v.validated(), "blockHash", v.lastValidGS.BlockHash)
	}
}

// return val:
// *MessageIndex - pointer to bad entry if there is one (requires reorg)
func (v *BlockValidator) sendValidations(ctx context.Context) (*arbutil.MessageIndex, error) {
	v.reorgMutex.RLock()
	defer v.reorgMutex.RUnlock()

	wasmRoots := v.GetModuleRootsToValidate()
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		v.reorgMutex.RUnlock()
		v.reorgMutex.RLock()
		pos := v.lastValidationSent()
		if pos >= v.validated()+arbutil.MessageIndex(v.config().ValidationSentLimit) {
			return nil, nil
		}
		if pos >= v.recordSent() {
			return nil, nil
		}
		validationStatus, found := v.validations.Load(pos)
		if !found {
			return nil, fmt.Errorf("not found entry for pos %d", pos)
		}
		currentStatus := validationStatus.getStatus()
		if currentStatus == RecordFailed {
			// retry
			log.Warn("Recording for validation failed, retrying..", "pos", pos)
			return &pos, nil
		}
		if currentStatus != Prepared {
			log.Trace("sendValidations: validation not prepared", "pos", pos, "status", currentStatus)
			return nil, nil
		}
		for _, moduleRoot := range wasmRoots {
			throttledSpawner := v.chosenValidator[moduleRoot]
			if throttledSpawner == nil {
				notFoundErr := fmt.Errorf("did not find spawner for moduleRoot :%v", moduleRoot)
				v.possiblyFatal(notFoundErr)
				return nil, notFoundErr
			}
			if !throttledSpawner.Throttler.HasCapacity() {
				log.Trace("sendValidations: no more capacity", "moduleRoot", moduleRoot, "spawner", throttledSpawner.Spawner.Name())
				return nil, nil
			}
		}
		if v.isMemoryLimitExceeded() {
			log.Error("sendValidations: aborting due to running low on memory")
			return nil, nil
		}
		replaced := validationStatus.replaceStatus(Prepared, SendingValidation)
		if !replaced {
			v.possiblyFatal(errors.New("failed to set SendingValidation status"))
		}
		validatorProfileWaitToLaunchHist.Update(validationStatus.profileStep())
		validatorPendingValidationsGauge.Inc(1)
		// Acquire workers for all module roots
		for _, moduleRoot := range wasmRoots {
			v.chosenValidator[moduleRoot].Throttler.Acquire()
		}
		var runs []validator.ValidationRun
		for _, moduleRoot := range wasmRoots {
			throttledSpawner := v.chosenValidator[moduleRoot]
			spawner := retry_wrapper.NewValidationSpawnerRetryWrapper(throttledSpawner.Spawner)
			spawner.StopWaiter.Start(ctx, v)
			input, err := validationStatus.Entry.ToInput(spawner.StylusArchs())
			if err != nil && ctx.Err() == nil {
				v.possiblyFatal(fmt.Errorf("%w: error preparing validation", err))
				throttledSpawner.Throttler.Release()
				continue
			}
			if ctx.Err() != nil {
				// Release all acquired capacity on cancellation
				for _, moduleRoot := range wasmRoots {
					v.chosenValidator[moduleRoot].Throttler.Release()
				}
				return nil, ctx.Err()
			}
			run := spawner.LaunchWithNAllowedAttempts(input, moduleRoot, v.config().ValidationSpawningAllowedAttempts)
			log.Trace("sendValidations: launched", "pos", validationStatus.Entry.Pos, "moduleRoot", moduleRoot)
			runs = append(runs, run)
		}
		validationStatus.DoneEntry = &validationDoneEntry{
			Success:         false,
			Start:           validationStatus.Entry.Start,
			End:             validationStatus.Entry.End,
			WasmModuleRoots: wasmRoots,
		}
		validationStatus.Entry = nil // no longer needed
		validatorProfileLaunchingHist.Update(validationStatus.profileStep())
		validationCtx, cancel := context.WithCancel(ctx)
		validationStatus.Cancel = cancel
		v.LaunchUntrackedThread(func() {
			defer validatorPendingValidationsGauge.Dec(1)
			defer cancel()
			// Release capacity when validations complete
			defer func() {
				for _, run := range runs {
					v.chosenValidator[run.WasmModuleRoot()].Throttler.Release()
				}
			}()
			markSuccess := len(runs) > 0

			// validationStatus might be removed from under us
			// trigger validation progress when done
			for _, run := range runs {
				runEnd, err := run.Await(validationCtx)
				if err == nil && runEnd != validationStatus.DoneEntry.End {
					err = fmt.Errorf("validation failed: got %v", runEnd)
				}
				if err != nil {
					validatorFailedValidationsCounter.Inc(1)
					markSuccess = false
					log.Error("error while validating", "err", err, "start", validationStatus.DoneEntry.Start, "end", validationStatus.DoneEntry.End)
					break
				}
				validatorValidValidationsCounter.Inc(1)
			}
			validationStatus.DoneEntry.Success = markSuccess
			validatorProfileRunningHist.Update(validationStatus.profileStep())
			replaced := validationStatus.replaceStatus(SendingValidation, ValidationDone)
			if !replaced {
				v.possiblyFatal(errors.New("failed to set SendingValidation status"))
			}
			nonBlockingTrigger(v.progressValidationsChan)
		})
		pos += 1
		atomicStorePos(&v.lastValidationSentA, pos, validatorMsgCountLastValidationSentGauge)
		log.Trace("validation sent", "pos", pos)
	}
}

func (v *BlockValidator) iterativeValidationProgress(ctx context.Context, ignored struct{}) time.Duration {
	reorg, err := v.advanceValidations(ctx)
	if err != nil {
		log.Error("error trying to record for validation node", "err", err)
	} else if reorg != nil {
		err := v.Reorg(ctx, *reorg)
		if err != nil {
			log.Error("error trying to reorg validation", "pos", *reorg-1, "err", err)
			v.possiblyFatal(err)
		}
	}
	return v.config().ValidationPoll
}

func (v *BlockValidator) iterativeValidationSentProgress(ctx context.Context, ignored struct{}) time.Duration {
	reorg, err := v.sendValidations(ctx)
	if err != nil {
		log.Error("error trying to send validation node", "err", err)
	} else if reorg != nil {
		err := v.Reorg(ctx, *reorg)
		if err != nil {
			log.Error("error trying to reorg validation", "pos", *reorg-1, "err", err)
			v.possiblyFatal(err)
		}
	}
	return v.config().ValidationPoll
}

var ErrValidationCanceled = errors.New("validation of block cancelled")

func (v *BlockValidator) writeLastValidated(gs validator.GoGlobalState, wasmRoots []common.Hash) error {
	v.lastValidGS = gs
	info := GlobalStateValidatedInfo{
		GlobalState: gs,
		WasmRoots:   wasmRoots,
	}
	encoded, err := rlp.EncodeToBytes(info)
	if err != nil {
		return err
	}
	err = v.db.Put(lastGlobalStateValidatedInfoKey, encoded)
	if err != nil {
		return err
	}
	return nil
}

func (v *BlockValidator) validGSIsNew(globalState validator.GoGlobalState) bool {
	if v.legacyValidInfo != nil {
		if v.legacyValidInfo.AfterPosition.BatchNumber > globalState.Batch {
			return false
		}
		if v.legacyValidInfo.AfterPosition.BatchNumber == globalState.Batch && v.legacyValidInfo.AfterPosition.PosInBatch >= globalState.PosInBatch {
			return false
		}
		return true
	}
	if v.lastValidGS.Batch > globalState.Batch {
		return false
	}
	if v.lastValidGS.Batch == globalState.Batch && v.lastValidGS.PosInBatch >= globalState.PosInBatch {
		return false
	}
	return true
}

// this accepts globalstate even if not caught up
func (v *BlockValidator) InitAssumeValid(globalState validator.GoGlobalState) error {
	if v.Started() {
		return fmt.Errorf("cannot handle InitAssumeValid while running")
	}

	// don't do anything if we already validated past that
	if !v.validGSIsNew(globalState) {
		return nil
	}

	v.legacyValidInfo = nil

	err := v.writeLastValidated(globalState, nil)
	if err != nil {
		log.Error("failed writing new validated to database", "pos", v.lastValidGS, "err", err)
	}

	return nil
}

func (v *BlockValidator) UpdateLatestStaked(count arbutil.MessageIndex, globalState validator.GoGlobalState) {

	if count <= v.validated() {
		return
	}

	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()

	if count <= v.validated() {
		return
	}

	if !v.chainCaughtUp {
		if !v.validGSIsNew(globalState) {
			return
		}
		v.legacyValidInfo = nil
		err := v.writeLastValidated(globalState, nil)
		if err != nil {
			log.Error("error writing last validated", "err", err)
		}
		return
	}

	countUint64 := uint64(count)
	msg, err := v.streamer.GetMessage(count - 1)
	if err != nil {
		log.Error("getMessage error", "err", err, "count", count)
		return
	}
	// delete no-longer relevant entries
	for iPos := v.validated(); iPos < count && iPos < v.created(); iPos++ {
		status, found := v.validations.Load(iPos)
		if found && status != nil && status.Cancel != nil {
			status.Cancel()
		}
		v.validations.Delete(iPos)
	}
	if v.created() < count {
		v.nextCreateStartGS = globalState
		v.nextCreatePrevDelayed = msg.DelayedMessagesRead
		v.nextCreateBatchReread = true
		if v.nextCreateBatch != nil {
			v.prevBatchCache[v.nextCreateBatch.Number] = v.nextCreateBatch.PostedData
		}
		v.createdA.Store(countUint64)
	}
	// under the reorg mutex we don't need atomic access
	if v.recordSentA.Load() < countUint64 {
		v.recordSentA.Store(countUint64)
	}

	if v.lastValidationSentA.Load() < countUint64 {
		v.lastValidationSentA.Store(countUint64)
	}
	// #nosec G115
	v.validatedA.Store(countUint64)
	// #nosec G115
	validatorMsgCountValidatedGauge.Update(int64(countUint64))
	err = v.writeLastValidated(globalState, nil) // we don't know which wasm roots were validated
	if err != nil {
		log.Error("failed writing valid state after reorg", "err", err)
	}
	nonBlockingTrigger(v.createNodesChan)
}

// Because batches and blocks are handled at separate layers in the node,
// and because block generation from messages is asynchronous,
// this call is different than Reorg, which is currently called later.
func (v *BlockValidator) ReorgToBatchCount(count uint64) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	if v.nextCreateStartGS.Batch >= count {
		v.nextCreateBatchReread = true
		v.prevBatchCache = make(map[uint64][]byte)
	}
}

func (v *BlockValidator) Reorg(ctx context.Context, count arbutil.MessageIndex) error {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	if count <= 1 {
		return errors.New("cannot reorg out genesis")
	}
	if !v.chainCaughtUp {
		return nil
	}
	if v.created() < count {
		return nil
	}
	_, endPosition, err := v.GlobalStatePositionsAtCount(count)
	if err != nil {
		v.possiblyFatal(err)
		return err
	}
	res, err := v.streamer.ResultAtMessageIndex(count - 1)
	if err != nil {
		v.possiblyFatal(err)
		return err
	}
	msg, err := v.streamer.GetMessage(count - 1)
	if err != nil {
		v.possiblyFatal(err)
		return err
	}
	for iPos := count; iPos < v.created(); iPos++ {
		status, found := v.validations.Load(iPos)
		if found && status != nil && status.Cancel != nil {
			status.Cancel()
		}
		v.validations.Delete(iPos)
	}
	v.nextCreateStartGS = BuildGlobalState(*res, endPosition)
	v.nextCreatePrevDelayed = msg.DelayedMessagesRead
	v.nextCreateBatchReread = true
	v.prevBatchCache = make(map[uint64][]byte)
	countUint64 := uint64(count)
	v.createdA.Store(countUint64)
	// under the reorg mutex we don't need atomic access
	if v.recordSentA.Load() > countUint64 {
		v.recordSentA.Store(countUint64)
	}
	if v.lastValidationSentA.Load() > countUint64 {
		v.lastValidationSentA.Store(countUint64)
		// #nosec G115
		validatorMsgCountLastValidationSentGauge.Update(int64(countUint64))
	}
	if v.validatedA.Load() > countUint64 {
		v.validatedA.Store(countUint64)
		// #nosec G115
		validatorMsgCountValidatedGauge.Update(int64(countUint64))
		err := v.writeLastValidated(v.nextCreateStartGS, nil) // we don't know which wasm roots were validated
		if err != nil {
			log.Error("failed writing valid state after reorg", "err", err)
		}
	}
	nonBlockingTrigger(v.createNodesChan)
	return nil
}

// Initialize must be called after SetCurrentWasmModuleRoot sets the current one
func (v *BlockValidator) Initialize(ctx context.Context) error {
	config := v.config()

	currentModuleRoot := config.CurrentModuleRoot
	switch currentModuleRoot {
	case "latest":
		v.currentWasmModuleRoot = v.GetLatestWasmModuleRoot()
	case "current":
		if (v.currentWasmModuleRoot == common.Hash{}) {
			return errors.New("wasmModuleRoot set to 'current' - but info not set from chain")
		}
	default:
		v.currentWasmModuleRoot = common.HexToHash(currentModuleRoot)
		if (v.currentWasmModuleRoot == common.Hash{}) {
			return errors.New("current-module-root config value illegal")
		}
	}
	pendingModuleRoot := config.PendingUpgradeModuleRoot
	if pendingModuleRoot != "" {
		if pendingModuleRoot == "latest" {
			v.pendingWasmModuleRoot = v.GetLatestWasmModuleRoot()
		} else {
			valid, _ := regexp.MatchString("(0x)?[0-9a-fA-F]{64}", pendingModuleRoot)
			v.pendingWasmModuleRoot = common.HexToHash(pendingModuleRoot)
			if (!valid || v.pendingWasmModuleRoot == common.Hash{}) {
				return errors.New("pending-upgrade-module-root config value illegal")
			}
		}
	}
	log.Info("BlockValidator initialized", "current", v.currentWasmModuleRoot, "pending", v.pendingWasmModuleRoot)
	moduleRoots := []common.Hash{v.currentWasmModuleRoot}
	if v.pendingWasmModuleRoot != v.currentWasmModuleRoot && v.pendingWasmModuleRoot != (common.Hash{}) {
		moduleRoots = append(moduleRoots, v.pendingWasmModuleRoot)
	}
	// First spawner is always RedisValidationClient if RedisStreams are enabled.
	if v.redisValidator != nil {
		err := v.redisValidator.Initialize(ctx, moduleRoots)
		if err != nil {
			return err
		}
	}
	v.chosenValidator = make(map[common.Hash]*ThrottledValidationSpawner)
	for _, root := range moduleRoots {
		if v.redisValidator != nil && validator.SpawnerSupportsModule(v.redisValidator, root) {
			v.chosenValidator[root] = NewThrottledValidationSpawner(v.redisValidator)
			log.Info("validator chosen", "WasmModuleRoot", root, "chosen", "redis", "maxWorkers", v.redisValidator.Capacity())
		} else {
			for _, spawner := range v.execSpawners {
				if validator.SpawnerSupportsModule(spawner, root) {
					v.chosenValidator[root] = NewThrottledValidationSpawner(spawner)
					log.Info("validator chosen", "WasmModuleRoot", root, "chosen", spawner.Name(), "maxWorkers", spawner.Capacity())
					break
				}
			}
			if v.chosenValidator[root] == nil {
				return fmt.Errorf("cannot validate WasmModuleRoot %v", root)
			}
		}
	}
	return nil
}

func (v *BlockValidator) checkLegacyValid() error {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	if v.legacyValidInfo == nil {
		return nil
	}
	batchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return err
	}
	requiredBatchCount := v.legacyValidInfo.AfterPosition.BatchNumber + 1
	if v.legacyValidInfo.AfterPosition.PosInBatch == 0 {
		requiredBatchCount -= 1
	}
	if batchCount < requiredBatchCount {
		log.Warn("legacy valid batch ahead of db", "current", batchCount, "required", requiredBatchCount)
		return nil
	}
	var msgCount arbutil.MessageIndex
	if v.legacyValidInfo.AfterPosition.BatchNumber > 0 {
		msgCount, err = v.inboxTracker.GetBatchMessageCount(v.legacyValidInfo.AfterPosition.BatchNumber - 1)
		if err != nil {
			return err
		}
	}
	msgCount += arbutil.MessageIndex(v.legacyValidInfo.AfterPosition.PosInBatch)
	processedCount, err := v.streamer.GetProcessedMessageCount()
	if err != nil {
		return err
	}
	if processedCount < msgCount {
		log.Warn("legacy valid message count ahead of db", "current", processedCount, "required", msgCount)
		return nil
	}

	result := &execution.MessageResult{}
	if msgCount > 0 {
		result, err = v.streamer.ResultAtMessageIndex(msgCount - 1)
		if err != nil {
			return err
		}
	}

	if result.BlockHash != v.legacyValidInfo.BlockHash {
		log.Error("legacy validated blockHash does not fit chain", "info.BlockHash", v.legacyValidInfo.BlockHash, "chain", result.BlockHash, "count", msgCount)
		return fmt.Errorf("legacy validated blockHash does not fit chain")
	}
	validGS := validator.GoGlobalState{
		BlockHash:  result.BlockHash,
		SendRoot:   result.SendRoot,
		Batch:      v.legacyValidInfo.AfterPosition.BatchNumber,
		PosInBatch: v.legacyValidInfo.AfterPosition.PosInBatch,
	}
	err = v.writeLastValidated(validGS, nil)
	if err == nil {
		err = v.db.Delete(legacyLastBlockValidatedInfoKey)
		if err != nil {
			err = fmt.Errorf("deleting legacy: %w", err)
		}
	}
	if err != nil {
		log.Error("failed writing initial lastValid on upgrade from legacy", "new-info", v.lastValidGS, "err", err)
	} else {
		log.Info("updated last-valid from legacy", "lastValid", v.lastValidGS)
	}
	v.legacyValidInfo = nil
	return nil
}

// checks that the chain caught up to lastValidGS, used in startup
func (v *BlockValidator) checkValidatedGSCaughtUp() (bool, error) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	if v.chainCaughtUp {
		return true, nil
	}
	if v.legacyValidInfo != nil {
		return false, nil
	}
	if v.lastValidGS.Batch == 0 {
		return false, errors.New("lastValid not initialized. cannot validate genesis")
	}
	caughtUp, count, err := GlobalStateToMsgCount(v.inboxTracker, v.streamer, v.lastValidGS)
	if err != nil {
		return false, err
	}
	if !caughtUp {
		batchCount, err := v.inboxTracker.GetBatchCount()
		if err != nil {
			log.Error("failed reading batch count", "err", err)
			batchCount = 0
		}
		var batchMsgCount arbutil.MessageIndex
		if batchCount == 0 {
			batchMsgCount = 0
		} else {
			batchMsgCount, err = v.inboxTracker.GetBatchMessageCount(batchCount - 1)
			if err != nil {
				log.Error("failed reading batchMsgCount", "err", err)
				batchMsgCount = 0
			}
		}
		processedMsgCount, err := v.streamer.GetProcessedMessageCount()
		if err != nil {
			log.Error("failed reading processedMsgCount", "err", err)
			processedMsgCount = 0
		}
		log.Info("validator catching up to last valid", "lastValid.Batch", v.lastValidGS.Batch, "lastValid.PosInBatch", v.lastValidGS.PosInBatch, "batchCount", batchCount, "batchMsgCount", batchMsgCount, "processedMsgCount", processedMsgCount)
		return false, nil
	}
	msg, err := v.streamer.GetMessage(count - 1)
	if err != nil {
		return false, err
	}
	v.nextCreateBatchReread = true
	v.nextCreateStartGS = v.lastValidGS
	v.nextCreatePrevDelayed = msg.DelayedMessagesRead
	atomicStorePos(&v.createdA, count, validatorMsgCountCreatedGauge)
	atomicStorePos(&v.recordSentA, count, validatorMsgCountRecordSentGauge)
	atomicStorePos(&v.validatedA, count, validatorMsgCountValidatedGauge)
	atomicStorePos(&v.lastValidationSentA, count, validatorMsgCountLastValidationSentGauge)
	// #nosec G115
	validatorMsgCountValidatedGauge.Update(int64(count))
	v.chainCaughtUp = true
	return true, nil
}

func (v *BlockValidator) LaunchWorkthreadsWhenCaughtUp(ctx context.Context) {
	for {
		err := v.checkLegacyValid()
		if err != nil {
			log.Error("validator got error updating legacy validated info. Consider restarting with dangerous.reset-block-validation", "err", err)
		}
		caughtUp, err := v.checkValidatedGSCaughtUp()
		if err != nil {
			log.Error("validator got error waiting for chain to catch up. Consider restarting with dangerous.reset-block-validation", "err", err)
		}
		if caughtUp {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(v.config().ValidationPoll):
		}
	}
	err := stopwaiter.CallIterativelyWith[struct{}](&v.StopWaiterSafe, v.iterativeValidationEntryCreator, v.createNodesChan)
	if err != nil {
		v.possiblyFatal(err)
	}
	err = stopwaiter.CallIterativelyWith[struct{}](&v.StopWaiterSafe, v.iterativeValidationEntryRecorder, v.sendRecordChan)
	if err != nil {
		v.possiblyFatal(err)
	}
	err = stopwaiter.CallIterativelyWith[struct{}](&v.StopWaiterSafe, v.iterativeValidationSentProgress, v.sendValidationsChan)
	if err != nil {
		v.possiblyFatal(err)
	}
	err = stopwaiter.CallIterativelyWith[struct{}](&v.StopWaiterSafe, v.iterativeValidationProgress, v.progressValidationsChan)
	if err != nil {
		v.possiblyFatal(err)
	}
}

func (v *BlockValidator) Start(ctxIn context.Context) error {
	v.StopWaiter.Start(ctxIn, v)
	v.LaunchThread(v.LaunchWorkthreadsWhenCaughtUp)
	v.CallIteratively(v.iterativeValidationPrint)
	return nil
}

func (v *BlockValidator) StopAndWait() {
	v.StopWaiter.StopAndWait()
}

// WaitForPos can only be used from One thread
func (v *BlockValidator) WaitForPos(t *testing.T, ctx context.Context, pos arbutil.MessageIndex, timeout time.Duration) bool {
	triggerchan := make(chan struct{})
	v.testingProgressMadeMutex.Lock()
	v.testingProgressMadeChan = triggerchan
	v.testingProgressMadeMutex.Unlock()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	lastLoop := false
	for {
		if v.validated() > pos {
			return true
		}
		if lastLoop {
			return false
		}
		select {
		case <-timer.C:
			lastLoop = true
		case <-triggerchan:
		case <-ctx.Done():
			lastLoop = true
		}
	}
}

func (v *BlockValidator) GetValidated() arbutil.MessageIndex {
	v.reorgMutex.RLock()
	defer v.reorgMutex.RUnlock()
	return v.validated()
}
