package staker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbnode/mel/recording"
	"github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client"
	"github.com/offchainlabs/nitro/validator/client/redis"
	"github.com/offchainlabs/nitro/validator/retry_wrapper"
)

type MsgPreimagesAndRelevantState struct {
	msgPreimages  daprovider.PreimagesMap
	relevantState *mel.State
}

type MELValidator struct {
	stopwaiter.StopWaiter

	config   MELValidatorConfigFetcher
	arbDb    ethdb.KeyValueStore
	l1Client *ethclient.Client

	messageExtractor *melrunner.MessageExtractor
	dapReaders       arbstate.DapReaderSource

	validateMsgExtractionTill atomic.Uint64

	melReorgDetector                chan uint64
	rewindMutex                     sync.Mutex
	latestValidatedGS               validator.GoGlobalState
	latestValidatedParentChainBlock uint64

	latestWasmModuleRoot common.Hash
	redisValidator       *redis.ValidationClient
	executionSpawners    []validator.ExecutionSpawner
	chosenValidator      map[common.Hash]validator.ValidationSpawner

	// wasmModuleRoot
	moduleMutex           sync.Mutex
	currentWasmModuleRoot common.Hash
	pendingWasmModuleRoot common.Hash

	// msgPreimagesAndStateCache is an LRU mapping a MEL state's parentChainBlockNumber to
	// the message preimages recorded during validation of the corresponding state
	msgPreimagesAndStateCacheMutex sync.RWMutex
	msgPreimagesAndStateCache      map[arbutil.MessageIndex]*MsgPreimagesAndRelevantState
}

type MELValidatorConfig struct {
	Enable                            bool                         `koanf:"enable"`
	RedisValidationClientConfig       redis.ValidationClientConfig `koanf:"redis-validation-client-config"`
	ValidationServer                  rpcclient.ClientConfig       `koanf:"validation-server" reload:"hot"`
	ValidationServerConfigs           []rpcclient.ClientConfig     `koanf:"validation-server-configs"`
	ValidationPoll                    time.Duration                `koanf:"validation-poll" reload:"hot"`
	CurrentModuleRoot                 string                       `koanf:"current-module-root"`
	PendingUpgradeModuleRoot          string                       `koanf:"pending-upgrade-module-root"`
	ValidationServerConfigsList       string                       `koanf:"validation-server-configs-list"`
	ValidationSpawningAllowedAttempts uint64                       `koanf:"validation-spawning-allowed-attempts" reload:"hot"`
}

func (c *MELValidatorConfig) Validate() error {
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
	return nil
}

type MELValidatorConfigFetcher func() *MELValidatorConfig

func MELValidatorConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMELValidatorConfig.Enable, "enable MEL state validation")
	rpcclient.RPCClientAddOptions(prefix+".validation-server", f, &DefaultMELValidatorConfig.ValidationServer)
	redis.ValidationClientConfigAddOptions(prefix+".redis-validation-client-config", f)
	f.String(prefix+".validation-server-configs-list", DefaultMELValidatorConfig.ValidationServerConfigsList, "array of execution rpc configs given as a json string. time duration should be supplied in number indicating nanoseconds")
	f.Duration(prefix+".validation-poll", DefaultMELValidatorConfig.ValidationPoll, "poll time to check validations")
	f.String(prefix+".current-module-root", DefaultMELValidatorConfig.CurrentModuleRoot, "current wasm module root ('current' read from chain, 'latest' from machines/latest dir, or provide hash)")
	f.String(prefix+".pending-upgrade-module-root", DefaultMELValidatorConfig.PendingUpgradeModuleRoot, "pending upgrade wasm module root to additionally validate (hash, 'latest' or empty)")
	f.Uint64(prefix+".validation-spawning-allowed-attempts", DefaultMELValidatorConfig.ValidationSpawningAllowedAttempts, "number of attempts allowed when trying to spawn a validation before erroring out")
}

var DefaultMELValidatorConfig = MELValidatorConfig{
	Enable:                            false,
	ValidationServerConfigsList:       "default",
	ValidationServer:                  rpcclient.DefaultClientConfig,
	RedisValidationClientConfig:       redis.DefaultValidationClientConfig,
	ValidationPoll:                    time.Second,
	CurrentModuleRoot:                 "current",
	PendingUpgradeModuleRoot:          "latest",
	ValidationSpawningAllowedAttempts: 1,
}

var TestMELValidatorConfig = MELValidatorConfig{
	Enable:                            false,
	ValidationServerConfigs:           []rpcclient.ClientConfig{rpcclient.TestClientConfig},
	ValidationServer:                  rpcclient.DefaultClientConfig,
	RedisValidationClientConfig:       redis.DefaultValidationClientConfig,
	ValidationPoll:                    100 * time.Millisecond,
	CurrentModuleRoot:                 "latest",
	PendingUpgradeModuleRoot:          "latest",
	ValidationSpawningAllowedAttempts: 1,
}

func NewMELValidator(
	config MELValidatorConfigFetcher,
	arbDb ethdb.KeyValueStore,
	l1Client *ethclient.Client,
	stack *node.Node,
	initialState *mel.State,
	messageExtractor *melrunner.MessageExtractor,
	dapReaders arbstate.DapReaderSource,
	latestWasmModuleRoot common.Hash,
	melReorgDetector chan uint64,
) (*MELValidator, error) {
	var executionSpawners []validator.ExecutionSpawner
	configs := config().ValidationServerConfigs
	for i := range configs {
		confFetcher := func() *rpcclient.ClientConfig { return &config().ValidationServerConfigs[i] }
		executionSpawner := client.NewExecutionClient(confFetcher, stack)
		executionSpawners = append(executionSpawners, executionSpawner)
	}
	if len(executionSpawners) == 0 {
		return nil, errors.New("no enabled execution servers")
	}
	var redisValClient *redis.ValidationClient
	if config().RedisValidationClientConfig.Enabled() {
		var err error
		redisValClient, err = redis.NewValidationClient(&config().RedisValidationClientConfig)
		if err != nil {
			return nil, fmt.Errorf("creating new redis validation client: %w", err)
		}
	}
	if latestWasmModuleRoot == (common.Hash{}) {
		return nil, errors.New("latestWasmModuleRoot not set")
	}
	if melReorgDetector == nil {
		return nil, errors.New("melReorgDetector not set")
	}
	mv := &MELValidator{
		config:                    config,
		arbDb:                     arbDb,
		l1Client:                  l1Client,
		messageExtractor:          messageExtractor,
		dapReaders:                dapReaders,
		latestWasmModuleRoot:      latestWasmModuleRoot,
		redisValidator:            redisValClient,
		executionSpawners:         executionSpawners,
		msgPreimagesAndStateCache: make(map[arbutil.MessageIndex]*MsgPreimagesAndRelevantState),
		melReorgDetector:          melReorgDetector,
	}
	info, err := ReadLastMELValidatedInfo(arbDb)
	if err != nil {
		return nil, err
	}
	if info != nil {
		mv.latestValidatedParentChainBlock = info.ParentChainBlockNumber
		mv.latestValidatedGS = info.GlobalState
	} else {
		if initialState == nil {
			return nil, errors.New("initialState is nil when starting out from scratch")
		}
		// Cannot validate genesis message
		mv.latestValidatedParentChainBlock = initialState.ParentChainBlockNumber
	}
	return mv, nil
}

func ReadLastMELValidatedInfo(db ethdb.KeyValueStore) (*MELGlobalStateValidatedInfo, error) {
	exists, err := db.Has(lastMELGlobalStateValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	var validated MELGlobalStateValidatedInfo
	if !exists {
		return nil, nil
	}
	infoBytes, err := db.Get(lastMELGlobalStateValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	err = rlp.DecodeBytes(infoBytes, &validated)
	if err != nil {
		return nil, err
	}
	return &validated, nil
}

func (mv *MELValidator) Initialize(ctx context.Context) error {
	config := mv.config()
	currentModuleRoot := config.CurrentModuleRoot
	switch currentModuleRoot {
	case "latest":
		mv.currentWasmModuleRoot = mv.latestWasmModuleRoot
	case "current":
		if (mv.currentWasmModuleRoot == common.Hash{}) {
			return errors.New("wasmModuleRoot set to 'current' - but info not set from chain")
		}
	default:
		mv.currentWasmModuleRoot = common.HexToHash(currentModuleRoot)
		if (mv.currentWasmModuleRoot == common.Hash{}) {
			return errors.New("current-module-root config value illegal")
		}
	}
	pendingModuleRoot := config.PendingUpgradeModuleRoot
	if pendingModuleRoot != "" {
		if pendingModuleRoot == "latest" {
			mv.pendingWasmModuleRoot = mv.latestWasmModuleRoot
		} else {
			valid, _ := regexp.MatchString("(0x)?[0-9a-fA-F]{64}", pendingModuleRoot)
			mv.pendingWasmModuleRoot = common.HexToHash(pendingModuleRoot)
			if (!valid || mv.pendingWasmModuleRoot == common.Hash{}) {
				return errors.New("pending-upgrade-module-root config value illegal")
			}
		}
	}
	log.Info("MELValidator initialized", "current", mv.currentWasmModuleRoot, "pending", mv.pendingWasmModuleRoot)
	if mv.redisValidator != nil {
		if err := mv.redisValidator.Start(ctx); err != nil {
			return fmt.Errorf("starting execution spawner: %w", err)
		}
	}
	for _, spawner := range mv.executionSpawners {
		if err := spawner.Start(ctx); err != nil {
			return err
		}
	}
	moduleRoots := mv.GetModuleRootsToValidate()
	// First spawner is always RedisValidationClient if RedisStreams are enabled.
	if mv.redisValidator != nil {
		err := mv.redisValidator.Initialize(ctx, moduleRoots)
		if err != nil {
			return err
		}
	}
	mv.chosenValidator = make(map[common.Hash]validator.ValidationSpawner)
	for _, root := range moduleRoots {
		if mv.redisValidator != nil && validator.SpawnerSupportsModule(mv.redisValidator, root) {
			mv.chosenValidator[root] = mv.redisValidator
			log.Info("validator chosen", "WasmModuleRoot", root, "chosen", "redis", "maxWorkers", mv.redisValidator.Capacity())
		} else {
			for _, spawner := range mv.executionSpawners {
				if validator.SpawnerSupportsModule(spawner, root) {
					mv.chosenValidator[root] = spawner
					log.Info("validator chosen", "WasmModuleRoot", root, "chosen", spawner.Name(), "maxWorkers", spawner.Capacity())
					break
				}
			}
			if mv.chosenValidator[root] == nil {
				return fmt.Errorf("cannot validate WasmModuleRoot %v", root)
			}
		}
	}
	return nil
}

func (mv *MELValidator) Start(ctx context.Context) {
	mv.StopWaiter.Start(ctx, mv)
	if mv.melReorgDetector != nil {
		mv.LaunchThread(mv.rewindOnMELReorgs)
	}
	mv.CallIteratively(func(ctx context.Context) time.Duration {
		mv.rewindMutex.Lock()
		defer mv.rewindMutex.Unlock()

		// Create validation entry
		entry, endMELState, err := mv.CreateNextValidationEntry(ctx, mv.latestValidatedParentChainBlock, mv.validateMsgExtractionTill.Load())
		if err != nil {
			log.Error("MEL validator: Error creating validation entry", "latestValidatedParentChainBlock", mv.latestValidatedParentChainBlock, "validateMsgExtractionTill", mv.validateMsgExtractionTill.Load(), "err", err)
			return time.Second
		}
		if entry == nil { // nothing to create, so lets wait for parentChain or blockValidator to make progress
			return time.Second
		}

		// Send validation entry to validation nodes
		doneEntry, err := mv.SendValidationEntry(ctx, entry)
		if err != nil {
			log.Error("MEL validator: Error sending validation entry", "err", err)
			return 0
		}

		// Advance validations
		if err := mv.AdvanceValidations(ctx, doneEntry); err != nil {
			log.Error("MEL validator: Error advancing validation status", "err", err)
			return 0
		}
		mv.latestValidatedParentChainBlock = endMELState.ParentChainBlockNumber
		mv.latestValidatedGS = doneEntry.End
		log.Info("Successfully validated Message extraction", "latestValidatedParentChainBlock", mv.latestValidatedParentChainBlock, "validatedMsgCount", endMELState.MsgCount)
		return 0
	})
}

func (mv *MELValidator) UpdateValidationTarget(target arbutil.MessageIndex) {
	mv.validateMsgExtractionTill.Store(uint64(target))
}

// rewindOnMELReorgs handles MEL related reorgs and will always be the first to receive a reorg event i.e before the blockvalidator,
// either L1 (parent chain reorg) or L2 (InitConfig.ReorgToMessageBatch) or both (parent chain reorg causing L2 reorg)
func (mv *MELValidator) rewindOnMELReorgs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case parentChainBlockNumber := <-mv.melReorgDetector:
			log.Info("MEL Validator: receieved a reorg event from message extractor", "parentChainBlockNumber", parentChainBlockNumber)
			mv.rewindMutex.Lock()
			if parentChainBlockNumber < mv.latestValidatedParentChainBlock {
				mv.latestValidatedParentChainBlock = parentChainBlockNumber
				mv.UpdateValidationTarget(0) // This makes the MEL validator wait until block validator asks it to validate msg extraction
				// Remove stale msg preimages
				mv.msgPreimagesAndStateCacheMutex.Lock()
				for key, val := range mv.msgPreimagesAndStateCache {
					if val.relevantState.ParentChainBlockNumber > parentChainBlockNumber {
						delete(mv.msgPreimagesAndStateCache, key)
					}
				}
				mv.msgPreimagesAndStateCacheMutex.Unlock()
			}
			mv.rewindMutex.Unlock()
		}
	}
}

func (mv *MELValidator) LatestValidatedMELState(ctx context.Context) (*mel.State, error) {
	mv.rewindMutex.Lock()
	defer mv.rewindMutex.Unlock()
	return mv.messageExtractor.GetState(mv.latestValidatedParentChainBlock)
}

func (mv *MELValidator) SetCurrentWasmModuleRoot(hash common.Hash) error {
	mv.moduleMutex.Lock()
	defer mv.moduleMutex.Unlock()

	if (hash == common.Hash{}) {
		return errors.New("trying to set zero as wasmModuleRoot")
	}
	if hash == mv.currentWasmModuleRoot {
		return nil
	}
	if (mv.currentWasmModuleRoot == common.Hash{}) {
		mv.currentWasmModuleRoot = hash
		return nil
	}
	if mv.pendingWasmModuleRoot == hash {
		log.Info("Block validator: detected progressing to pending machine", "hash", hash)
		mv.currentWasmModuleRoot = hash
		return nil
	}
	if mv.config().CurrentModuleRoot != "current" {
		return nil
	}
	return fmt.Errorf(
		"unexpected wasmModuleRoot! cannot validate! found %v, current %v, pending %v",
		hash, mv.currentWasmModuleRoot, mv.pendingWasmModuleRoot,
	)
}

func (mv *MELValidator) GetModuleRootsToValidate() []common.Hash {
	mv.moduleMutex.Lock()
	defer mv.moduleMutex.Unlock()

	validatingModuleRoots := []common.Hash{mv.currentWasmModuleRoot}
	if mv.currentWasmModuleRoot != mv.pendingWasmModuleRoot && mv.pendingWasmModuleRoot != (common.Hash{}) {
		validatingModuleRoots = append(validatingModuleRoots, mv.pendingWasmModuleRoot)
	}
	return validatingModuleRoots
}

func (mv *MELValidator) CreateNextValidationEntry(ctx context.Context, lastValidatedParentChainBlock, validateMsgExtractionTill uint64) (*validationEntry, *mel.State, error) {
	if lastValidatedParentChainBlock == 0 { // TODO: last validated.
		// ending position- bold staker latest posted assertion on chain that it agrees with (l1blockhash)-
		return nil, nil, errors.New("trying to create validation entry for zero block number")
	}
	currentState, err := mv.messageExtractor.GetState(lastValidatedParentChainBlock)
	if err != nil {
		return nil, nil, err
	}
	// In case this is the first state, set delayedMessageBacklog to read InitMsg
	currentState.SetDelayedMessageBacklog(&mel.DelayedMessageBacklog{})
	// We have already validated message extraction of messages till count toValidateMsgExtractionCount, so can return early
	// and wait for block validator to progress the toValidateMsgExtractionCount
	if currentState.MsgCount > validateMsgExtractionTill {
		return nil, nil, nil
	}
	initialState := currentState.Clone()
	encodedInitialState, err := rlp.EncodeToBytes(initialState)
	if err != nil {
		return nil, nil, err
	}
	preimages := make(daprovider.PreimagesMap)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	preimages[arbutil.Keccak256PreimageType][initialState.Hash()] = encodedInitialState
	delayedMsgRecordingDB, err := melrecording.NewDelayedMsgDatabase(mv.arbDb, preimages)
	if err != nil {
		return nil, nil, err
	}
	recordingDAPReaders, err := melrecording.NewDAPReaderSource(ctx, mv.dapReaders, preimages)
	if err != nil {
		return nil, nil, err
	}
	melMsgHash := common.Hash{}
	var endState *mel.State
	for i := lastValidatedParentChainBlock + 1; ; i++ {
		header, err := mv.l1Client.HeaderByNumber(ctx, new(big.Int).SetUint64(i))
		if err != nil {
			if errors.Is(err, ethereum.NotFound) { // Wait for parent chain to progress
				return nil, nil, nil
			}
			return nil, nil, err
		}
		encodedHeader, err := rlp.EncodeToBytes(header)
		if err != nil {
			return nil, nil, err
		}
		preimages[arbutil.Keccak256PreimageType][header.Hash()] = encodedHeader
		txsRecorder, err := melrecording.NewTransactionRecorder(mv.l1Client, header.Hash(), preimages)
		if err != nil {
			return nil, nil, err
		}
		if err := txsRecorder.Initialize(ctx); err != nil {
			return nil, nil, err
		}
		recordedLogsFetcher, err := melrecording.RecordReceipts(ctx, mv.l1Client, header.Hash(), preimages)
		if err != nil {
			return nil, nil, err
		}
		// Record msg preimages separately in order to make it available for block validation later
		msgPreimages := make(daprovider.PreimagesMap)
		if err := currentState.RecordMsgPreimagesTo(msgPreimages); err != nil {
			return nil, nil, err
		}
		var l2Msgs []*arbostypes.MessageWithMetadata
		endState, l2Msgs, _, _, err = melextraction.ExtractMessages(ctx, currentState, header, recordingDAPReaders, delayedMsgRecordingDB, txsRecorder, recordedLogsFetcher, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("error calling melextraction.ExtractMessages in recording mode: %w", err)
		}
		if len(l2Msgs) > 0 && (melMsgHash == common.Hash{}) {
			melMsgHash = l2Msgs[0].Hash()
		}
		wantState, err := mv.messageExtractor.GetState(i)
		if err != nil {
			return nil, nil, err
		}
		if endState.Hash() != wantState.Hash() {
			return nil, nil, fmt.Errorf("calculated MEL state hash in recording mode doesn't match the one computed in native mode, parentchainBlocknumber: %d", i)
		}
		if len(msgPreimages[arbutil.Keccak256PreimageType]) > 0 {
			mv.msgPreimagesAndStateCacheMutex.Lock()
			preimagesAndState := &MsgPreimagesAndRelevantState{
				msgPreimages:  msgPreimages,
				relevantState: endState,
			}
			for msgIndex := currentState.MsgCount; msgIndex < endState.MsgCount; msgIndex++ {
				mv.msgPreimagesAndStateCache[arbutil.MessageIndex(msgIndex)] = preimagesAndState
			}
			mv.msgPreimagesAndStateCacheMutex.Unlock()
			validator.CopyPreimagesInto(preimages, msgPreimages)
		}
		if endState.MsgCount > validateMsgExtractionTill {
			break
		}
		currentState = endState
	}
	return &validationEntry{
		Stage:     Ready,
		Preimages: preimages,
		Start: validator.GoGlobalState{
			BlockHash:    common.Hash{},
			MELStateHash: initialState.Hash(),
			MELMsgHash:   common.Hash{},
			Batch:        0,
			PosInBatch:   initialState.MsgCount,
		},
		End: validator.GoGlobalState{
			BlockHash:    common.Hash{},
			MELStateHash: endState.Hash(),
			MELMsgHash:   melMsgHash,
			Batch:        0,
			PosInBatch:   initialState.MsgCount,
		},
		EndParentChainBlockHash: endState.ParentChainBlockHash,
	}, endState, nil
}

// FetchMsgPreimagesAndRelevantState is to be only called after validating the extraction of l2BlockNum message by comparing with LatestValidatedMELState
func (mv *MELValidator) FetchMsgPreimagesAndRelevantState(ctx context.Context, msgIndex arbutil.MessageIndex) (*MsgPreimagesAndRelevantState, error) {
	mv.msgPreimagesAndStateCacheMutex.RLock()
	defer mv.msgPreimagesAndStateCacheMutex.RUnlock()
	preimagesAndRelevantState, found := mv.msgPreimagesAndStateCache[msgIndex]
	if !found {
		return nil, fmt.Errorf("Couldn't find msg preimages, msgIndex: %d", msgIndex)
	}
	return preimagesAndRelevantState, nil
}

// FetchMessageOriginMELStateHash returns the hash of the MEL state that extracted the message corresponding to the given position
func (mv *MELValidator) FetchMessageOriginMELStateHash(pos arbutil.MessageIndex) (common.Hash, error) {
	mv.msgPreimagesAndStateCacheMutex.RLock()
	preimagesAndRelevantState, found := mv.msgPreimagesAndStateCache[pos]
	mv.msgPreimagesAndStateCacheMutex.RUnlock()
	if !found {
		state, err := mv.messageExtractor.FindMessageOriginMELState(pos)
		if err != nil {
			return common.Hash{}, err
		}
		return state.Hash(), nil
	}
	return preimagesAndRelevantState.relevantState.Hash(), nil
}

// ClearValidatedMsgPreimages trims the msgPreimagesAndStateCache by clearing out entries with
// keys lower than the last validated l2 block
func (mv *MELValidator) ClearValidatedMsgPreimages(lastValidatedL2BlockNumber arbutil.MessageIndex) {
	mv.msgPreimagesAndStateCacheMutex.Lock()
	defer mv.msgPreimagesAndStateCacheMutex.Unlock()
	for key := range mv.msgPreimagesAndStateCache {
		if key < lastValidatedL2BlockNumber {
			delete(mv.msgPreimagesAndStateCache, key)
		}
	}
}

func (mv *MELValidator) SendValidationEntry(ctx context.Context, entry *validationEntry) (*validationDoneEntry, error) {
	wasmRoots := mv.GetModuleRootsToValidate()
	var runs []validator.ValidationRun
	for _, moduleRoot := range wasmRoots {
		chosenSpawner := mv.chosenValidator[moduleRoot]
		spawner := retry_wrapper.NewValidationSpawnerRetryWrapper(chosenSpawner)
		spawner.StopWaiter.Start(ctx, mv)
		input, err := entry.ToInput(nil)
		if err != nil && ctx.Err() == nil {
			return nil, fmt.Errorf("error preparing validation: %w", err)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		run := spawner.LaunchWithNAllowedAttempts(input, moduleRoot, mv.config().ValidationSpawningAllowedAttempts)
		log.Trace("sendValidations: launched", "pos", entry.Pos, "moduleRoot", moduleRoot)
		runs = append(runs, run)
	}
	for _, run := range runs {
		runEnd, err := run.Await(ctx)
		if err == nil && runEnd != entry.End {
			err = fmt.Errorf("validation failed: got %v", runEnd)
		}
		if err != nil {
			return nil, fmt.Errorf("MEL validator: error while validating: %w", err)
		}
	}
	return &validationDoneEntry{
		Success:         true,
		Start:           entry.Start,
		End:             entry.End,
		WasmModuleRoots: wasmRoots,
	}, nil
}

func (mv *MELValidator) AdvanceValidations(ctx context.Context, doneEntry *validationDoneEntry) error {
	info := MELGlobalStateValidatedInfo{
		ParentChainBlockNumber: mv.latestValidatedParentChainBlock, // rewindMutex already held, no need to acquire
		GlobalState:            doneEntry.End,
		WasmRoots:              doneEntry.WasmModuleRoots,
	}
	encoded, err := rlp.EncodeToBytes(info)
	if err != nil {
		return err
	}
	err = mv.arbDb.Put(lastMELGlobalStateValidatedInfoKey, encoded)
	if err != nil {
		return err
	}
	return nil
}
