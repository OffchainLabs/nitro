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

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
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
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client"
	"github.com/offchainlabs/nitro/validator/client/redis"
	"github.com/offchainlabs/nitro/validator/retry_wrapper"
)

type MELValidator struct {
	stopwaiter.StopWaiter

	config   MELValidatorConfigFetcher
	arbDb    ethdb.KeyValueStore
	l1Client *ethclient.Client

	boldStakerAddr common.Address
	rollupAddr     common.Address
	rollup         *rollupgen.RollupUserLogic

	messageExtractor *melrunner.MessageExtractor
	dapReaders       arbstate.DapReaderSource

	latestValidatedGS               validator.GoGlobalState
	latestValidatedParentChainBlock atomic.Uint64

	latestWasmModuleRoot common.Hash
	redisValidator       *redis.ValidationClient
	executionSpawners    []validator.ExecutionSpawner
	chosenValidator      map[common.Hash]validator.ValidationSpawner

	// wasmModuleRoot
	moduleMutex           sync.Mutex
	currentWasmModuleRoot common.Hash
	pendingWasmModuleRoot common.Hash
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
	BlockValidatorDangerousConfigAddOptions(prefix+".dangerous", f)
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

func NewMELValidator(
	config MELValidatorConfigFetcher,
	arbDb ethdb.KeyValueStore,
	l1Client *ethclient.Client,
	stack *node.Node,
	messageExtractor *melrunner.MessageExtractor,
	dapReaders arbstate.DapReaderSource,
	latestWasmModuleRoot common.Hash,
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
	return &MELValidator{
		config:               config,
		arbDb:                arbDb,
		l1Client:             l1Client,
		messageExtractor:     messageExtractor,
		dapReaders:           dapReaders,
		latestWasmModuleRoot: latestWasmModuleRoot,
		redisValidator:       redisValClient,
		executionSpawners:    executionSpawners,
	}, nil
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
	mv.CallIteratively(func(ctx context.Context) time.Duration {
		latestStaked, err := mv.rollup.LatestStakedAssertion(&bind.CallOpts{}, mv.boldStakerAddr)
		if err != nil {
			log.Error("MEL validator: Error fetching latest staked assertion hash", "err", err)
			return 0
		}
		latestStakedAssertion, err := ReadBoldAssertionCreationInfo(ctx, mv.rollup, mv.l1Client, mv.rollupAddr, latestStaked)
		if err != nil {
			log.Error("MEL validator: Error fetching latest staked assertion creation info", "err", err)
			return 0
		}
		if latestStakedAssertion.InboxMaxCount == nil || !latestStakedAssertion.InboxMaxCount.IsUint64() {
			log.Error("MEL validator: latestStakedAssertion.InboxMaxCount is not uint64")
			return 0
		}

		// Create validation entry
		entry, endGSParentChainBlockNumber, err := mv.CreateNextValidationEntry(ctx, mv.latestValidatedParentChainBlock.Load(), latestStakedAssertion.InboxMaxCount.Uint64())
		if err != nil {
			log.Error("MEL validator: Error creating validation entry", "latestValidatedParentChainBlock", mv.latestValidatedParentChainBlock.Load(), "inboxMaxCount", latestStakedAssertion.InboxMaxCount.Uint64(), "err", err)
			return 0
		}
		if entry == nil { // nothing to create, so lets wait for latestStakedAssertion to progress through blockValidator
			return time.Minute
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
		}
		mv.latestValidatedParentChainBlock.Store(endGSParentChainBlockNumber)
		mv.latestValidatedGS = doneEntry.End
		return 0
	})
}

func (mv *MELValidator) LatestValidatedMELState(ctx context.Context) (*mel.State, error) {
	return mv.messageExtractor.GetState(ctx, mv.latestValidatedParentChainBlock.Load())
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

func (mv *MELValidator) CreateNextValidationEntry(ctx context.Context, lastValidatedParentChainBlock, toValidateMsgExtractionCount uint64) (*validationEntry, uint64, error) {
	if lastValidatedParentChainBlock == 0 { // TODO: last validated.
		// ending position- bold staker latest posted assertion on chain that it agrees with (l1blockhash)-
		return nil, 0, errors.New("trying to create validation entry for zero block number")
	}
	currentState, err := mv.messageExtractor.GetState(ctx, lastValidatedParentChainBlock)
	if err != nil {
		return nil, 0, err
	}
	// We have already validated message extraction of messages till count toValidateMsgExtractionCount, so can return early
	// and wait for block validator to progress the toValidateMsgExtractionCount
	if currentState.MsgCount >= toValidateMsgExtractionCount {
		return nil, 0, nil
	}
	initialState := currentState.Clone()
	encodedInitialState, err := rlp.EncodeToBytes(initialState)
	if err != nil {
		return nil, 0, err
	}
	preimages := make(daprovider.PreimagesMap)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	preimages[arbutil.Keccak256PreimageType][initialState.Hash()] = encodedInitialState
	delayedMsgRecordingDB, err := melrecording.NewDelayedMsgDatabase(mv.arbDb, preimages)
	if err != nil {
		return nil, 0, err
	}
	recordingDAPReaders, err := melrecording.NewDAPReaderSource(ctx, mv.dapReaders, preimages)
	if err != nil {
		return nil, 0, err
	}
	currentState.RecordMsgPreimagesTo(preimages)
	var endState *mel.State
	for i := lastValidatedParentChainBlock + 1; ; i++ {
		header, err := mv.l1Client.HeaderByNumber(ctx, new(big.Int).SetUint64(i))
		if err != nil {
			return nil, 0, err
		}
		encodedHeader, err := rlp.EncodeToBytes(header)
		if err != nil {
			return nil, 0, err
		}
		preimages[arbutil.Keccak256PreimageType][header.Hash()] = encodedHeader
		txsRecorder, err := melrecording.NewTransactionRecorder(mv.l1Client, header.Hash(), preimages)
		if err != nil {
			return nil, 0, err
		}
		if err := txsRecorder.Initialize(ctx); err != nil {
			return nil, 0, err
		}
		recordedLogsFetcher, err := melrecording.RecordReceipts(ctx, mv.l1Client, header.Hash(), preimages)
		if err != nil {
			return nil, 0, err
		}
		endState, _, _, _, err = melextraction.ExtractMessages(ctx, currentState, header, recordingDAPReaders, delayedMsgRecordingDB, txsRecorder, recordedLogsFetcher, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("error calling melextraction.ExtractMessages in recording mode: %w", err)
		}
		wantState, err := mv.messageExtractor.GetState(ctx, i)
		if err != nil {
			return nil, 0, err
		}
		if endState.Hash() != wantState.Hash() {
			return nil, 0, fmt.Errorf("calculated MEL state hash in recording mode doesn't match the one computed in native mode, parentchainBlocknumber: %d", i)
		}
		if endState.MsgCount >= toValidateMsgExtractionCount {
			break
		}
		currentState = endState
	}
	fmt.Printf("Initial state hash: %#x\n", initialState.Hash())
	return &validationEntry{
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
			MELMsgHash:   common.Hash{},
			Batch:        0,
			PosInBatch:   0,
		},
		EndParentChainBlockHash: endState.ParentChainBlockHash,
	}, 0, nil
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
			return nil, fmt.Errorf("%w: error preparing validation", err)
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
	info := GlobalStateValidatedInfo{
		GlobalState: doneEntry.End,
		WasmRoots:   doneEntry.WasmModuleRoots,
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
