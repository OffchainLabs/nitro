//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/broadcastclient"
	"github.com/offchainlabs/arbstate/broadcaster"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/solgen/go/ospgen"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/offchainlabs/arbstate/statetransfer"
	"github.com/offchainlabs/arbstate/validator"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

type RollupAddresses struct {
	Bridge         common.Address
	Inbox          common.Address
	SequencerInbox common.Address
	Rollup         common.Address
	DeployedAt     uint64
}

func andTxSucceeded(ctx context.Context, l1client L1Interface, txTimeout time.Duration, tx *types.Transaction, err error) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	_, err = EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout)
	if err != nil {
		return fmt.Errorf("error executing tx: %w", err)
	}
	return nil
}

func deployBridgeCreator(ctx context.Context, client L1Interface, auth *bind.TransactOpts, txTimeout time.Duration) (common.Address, error) {
	bridgeTemplate, tx, _, err := bridgegen.DeployBridge(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge deploy error: %w", err)
	}

	seqInboxTemplate, tx, _, err := bridgegen.DeploySequencerInbox(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("sequencer inbox deploy error: %w", err)
	}

	inboxTemplate, tx, _, err := bridgegen.DeployInbox(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("inbox deploy error: %w", err)
	}

	rollupEventBridgeTemplate, tx, _, err := rollupgen.DeployRollupEventBridge(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("rollup event bridge deploy error: %w", err)
	}

	outboxTemplate, tx, _, err := bridgegen.DeployOutbox(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("outbox deploy error: %w", err)
	}

	bridgeCreatorAddr, tx, bridgeCreator, err := rollupgen.DeployBridgeCreator(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge creator deploy error: %w", err)
	}

	tx, err = bridgeCreator.UpdateTemplates(auth, bridgeTemplate, seqInboxTemplate, inboxTemplate, rollupEventBridgeTemplate, outboxTemplate)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("bridge creator update templates error: %w", err)
	}

	return bridgeCreatorAddr, nil
}

func deployChallengeFactory(ctx context.Context, client L1Interface, auth *bind.TransactOpts, txTimeout time.Duration) (common.Address, error) {
	osp0, tx, _, err := ospgen.DeployOneStepProver0(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("osp0 deploy error: %w", err)
	}

	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("ospMemory deploy error: %w", err)
	}

	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("ospMath deploy error: %w", err)
	}

	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("ospHostIo deploy error: %w", err)
	}

	ospEntryAddr, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, fmt.Errorf("ospEntry deploy error: %w", err)
	}

	return ospEntryAddr, nil
}

func deployRollupCreator(ctx context.Context, client L1Interface, auth *bind.TransactOpts, txTimeout time.Duration) (*rollupgen.RollupCreator, error) {
	bridgeCreator, err := deployBridgeCreator(ctx, client, auth, txTimeout)
	if err != nil {
		return nil, err
	}

	rollupTemplate, tx, _, err := rollupgen.DeployRollup(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, fmt.Errorf("rollup deploy error: %w", err)
	}

	challengeFactory, err := deployChallengeFactory(ctx, client, auth, txTimeout)
	if err != nil {
		return nil, err
	}

	rollupAdminLogic, tx, _, err := rollupgen.DeployRollupAdminLogic(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, fmt.Errorf("rollup admin logic deploy error: %w", err)
	}

	rollupUserLogic, tx, _, err := rollupgen.DeployRollupUserLogic(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	_, tx, rollupCreator, err := rollupgen.DeployRollupCreator(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	tx, err = rollupCreator.SetTemplates(auth, bridgeCreator, rollupTemplate, challengeFactory, rollupAdminLogic, rollupUserLogic)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	return rollupCreator, nil
}

func DeployOnL1(ctx context.Context, l1client L1Interface, deployAuth *bind.TransactOpts, sequencer common.Address, wasmModuleRoot common.Hash, txTimeout time.Duration) (*RollupAddresses, error) {
	rollupCreator, err := deployRollupCreator(ctx, l1client, deployAuth, txTimeout)
	if err != nil {
		return nil, err
	}

	confirmPeriodBlocks := big.NewInt(20)
	extraChallengeTimeBlocks := big.NewInt(20)
	seqInboxParams := [4]*big.Int{
		big.NewInt(60 * 60 * 24 * 15), // maxDelayBlocks
		big.NewInt(12),                // maxFutureBlocks
		big.NewInt(60 * 60 * 24),      // maxDelaySeconds
		big.NewInt(60 * 60),           // maxFutureSeconds
	}
	tx, err := rollupCreator.CreateRollup(
		deployAuth,
		confirmPeriodBlocks,
		extraChallengeTimeBlocks,
		common.Address{},
		big.NewInt(params.Ether),
		wasmModuleRoot,
		deployAuth.From,
		big.NewInt(1338),
		seqInboxParams,
	)
	if err != nil {
		return nil, fmt.Errorf("error submitting create rollup tx: %w", err)
	}
	receipt, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout)
	if err != nil {
		return nil, fmt.Errorf("error executing create rollup tx: %w", err)
	}
	info, err := rollupCreator.ParseRollupCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, fmt.Errorf("error parsing rollup created log: %w", err)
	}

	rollup, err := rollupgen.NewRollupAdminLogic(info.RollupAddress, l1client)
	if err != nil {
		return nil, err
	}
	tx, err = rollup.SetIsBatchPoster(deployAuth, sequencer, true)
	err = andTxSucceeded(ctx, l1client, txTimeout, tx, err)
	if err != nil {
		return nil, fmt.Errorf("error setting is batch poster: %w", err)
	}

	return &RollupAddresses{
		Bridge:         info.DelayedBridge,
		Inbox:          info.InboxAddress,
		SequencerInbox: info.SequencerInbox,
		DeployedAt:     receipt.BlockNumber.Uint64(),
		Rollup:         info.RollupAddress,
	}, nil
}

type NodeConfig struct {
	ArbConfig              arbitrum.Config
	L1Reader               bool
	InboxReaderConfig      InboxReaderConfig
	DelayedSequencerConfig DelayedSequencerConfig
	BatchPoster            bool
	BatchPosterConfig      BatchPosterConfig
	ForwardingTarget       string // "" if not forwarding
	BlockValidator         bool
	BlockValidatorConfig   validator.BlockValidatorConfig
	Broadcaster            bool
	BroadcasterConfig      wsbroadcastserver.BroadcasterConfig
	BroadcastClient        bool
	BroadcastClientConfig  broadcastclient.BroadcastClientConfig
}

var NodeConfigDefault = NodeConfig{arbitrum.DefaultConfig, true, DefaultInboxReaderConfig, DefaultDelayedSequencerConfig, true, DefaultBatchPosterConfig, "", false, validator.DefaultBlockValidatorConfig, false, wsbroadcastserver.DefaultBroadcasterConfig, false, broadcastclient.DefaultBroadcastClientConfig}
var NodeConfigL1Test = NodeConfig{arbitrum.DefaultConfig, true, TestInboxReaderConfig, TestDelayedSequencerConfig, true, TestBatchPosterConfig, "", false, validator.DefaultBlockValidatorConfig, false, wsbroadcastserver.DefaultBroadcasterConfig, false, broadcastclient.DefaultBroadcastClientConfig}
var NodeConfigL2Test = NodeConfig{ArbConfig: arbitrum.DefaultConfig, L1Reader: false}

type Node struct {
	Backend          *arbitrum.Backend
	ArbInterface     *ArbInterface
	TxStreamer       *TransactionStreamer
	TxPublisher      TransactionPublisher
	DeployInfo       *RollupAddresses
	InboxReader      *InboxReader
	InboxTracker     *InboxTracker
	DelayedSequencer *DelayedSequencer
	BatchPoster      *BatchPoster
	BlockValidator   *validator.BlockValidator
	BroadcastServer  *broadcaster.Broadcaster
	BroadcastClient  *broadcastclient.BroadcastClient
}

func CreateNode(stack *node.Node, chainDb ethdb.Database, config *NodeConfig, l2BlockChain *core.BlockChain, l1client L1Interface, deployInfo *RollupAddresses, sequencerTxOpt *bind.TransactOpts) (*Node, error) {
	var broadcastServer *broadcaster.Broadcaster
	if config.Broadcaster {
		broadcastServer = broadcaster.NewBroadcaster(config.BroadcasterConfig)
	}

	txStreamer, err := NewTransactionStreamer(chainDb, l2BlockChain, broadcastServer)
	if err != nil {
		return nil, err
	}
	var txPublisher TransactionPublisher
	if config.ForwardingTarget != "" {
		txPublisher, err = NewForwarder(config.ForwardingTarget)
	} else if config.L1Reader {
		if l1client == nil {
			return nil, errors.New("l1client is nil")
		}
		txPublisher, err = NewSequencer(txStreamer, l1client)
	} else {
		txPublisher, err = NewSequencer(txStreamer, nil)
	}
	if err != nil {
		return nil, err
	}
	arbInterface, err := NewArbInterface(txStreamer, txPublisher)
	if err != nil {
		return nil, err
	}
	backend, err := arbitrum.NewBackend(stack, &config.ArbConfig, chainDb, l2BlockChain, arbInterface)
	if err != nil {
		return nil, err
	}
	var broadcastClient *broadcastclient.BroadcastClient
	if config.BroadcastClient {
		broadcastClient = broadcastclient.NewBroadcastClient(config.BroadcastClientConfig.URL, nil, config.BroadcastClientConfig.Timeout, txStreamer)
	}
	if !config.L1Reader {
		return &Node{backend, arbInterface, txStreamer, txPublisher, nil, nil, nil, nil, nil, nil, broadcastServer, broadcastClient}, nil
	}

	if deployInfo == nil {
		return nil, errors.New("deployinfo is nil")
	}
	delayedBridge, err := NewDelayedBridge(l1client, deployInfo.Bridge, deployInfo.DeployedAt)
	if err != nil {
		return nil, err
	}
	sequencerInbox, err := NewSequencerInbox(l1client, deployInfo.SequencerInbox, int64(deployInfo.DeployedAt))
	if err != nil {
		return nil, err
	}
	inboxTracker, err := NewInboxTracker(chainDb, txStreamer)
	if err != nil {
		return nil, err
	}
	inboxReader, err := NewInboxReader(inboxTracker, l1client, new(big.Int).SetUint64(deployInfo.DeployedAt), delayedBridge, sequencerInbox, &(config.InboxReaderConfig))
	if err != nil {
		return nil, err
	}

	var blockValidator *validator.BlockValidator
	if config.BlockValidator {
		blockValidator = validator.NewBlockValidator(inboxTracker, txStreamer, l2BlockChain, &config.BlockValidatorConfig)
	}

	if !config.BatchPoster {
		return &Node{backend, arbInterface, txStreamer, txPublisher, deployInfo, inboxReader, inboxTracker, nil, nil, blockValidator, broadcastServer, broadcastClient}, nil
	}

	if sequencerTxOpt == nil {
		return nil, errors.New("sequencerTxOpts is nil")
	}
	delayedSequencer, err := NewDelayedSequencer(l1client, inboxReader, txStreamer, &(config.DelayedSequencerConfig))
	if err != nil {
		return nil, err
	}
	batchPoster, err := NewBatchPoster(l1client, inboxTracker, txStreamer, &config.BatchPosterConfig, deployInfo.SequencerInbox, common.Address{}, sequencerTxOpt)
	if err != nil {
		return nil, err
	}
	return &Node{backend, arbInterface, txStreamer, txPublisher, deployInfo, inboxReader, inboxTracker, delayedSequencer, batchPoster, blockValidator, broadcastServer, broadcastClient}, nil
}

func (n *Node) Start(ctx context.Context) error {
	err := n.TxPublisher.Initialize(ctx)
	if err != nil {
		return err
	}
	err = n.TxStreamer.Initialize()
	if err != nil {
		return err
	}
	if n.InboxTracker != nil {
		err = n.InboxTracker.Initialize()
		if err != nil {
			return err
		}
	}
	n.TxStreamer.Start(ctx)
	if n.InboxReader != nil {
		err = n.InboxReader.Start(ctx)
		if err != nil {
			return err
		}
	}
	err = n.TxPublisher.Start(ctx)
	if err != nil {
		return err
	}
	if n.DelayedSequencer != nil {
		n.DelayedSequencer.Start(ctx)
	}
	if n.BatchPoster != nil {
		n.BatchPoster.Start(ctx)
	}
	if n.BlockValidator != nil {
		err = n.BlockValidator.Start(ctx)
		if err != nil {
			return err
		}
	}
	if n.BroadcastServer != nil {
		err = n.BroadcastServer.Start(ctx)
		if err != nil {
			return err
		}
	}
	if n.BroadcastClient != nil {
		n.BroadcastClient.Start(ctx)
	}
	return nil
}

func CreateDefaultStack() (*node.Node, error) {
	stackConf := node.DefaultConfig
	var err error
	stackConf.DataDir = ""
	stackConf.HTTPHost = "localhost"
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stack, err := node.New(&stackConf)
	if err != nil {
		return nil, fmt.Errorf("error creating protocol stack: %w", err)
	}
	return stack, nil
}

func CreateDefaultBlockChain(stack *node.Node, initData *statetransfer.ArbosInitializationInfo, config *params.ChainConfig) (ethdb.Database, *core.BlockChain, error) {
	arbstate.RequireHookedGeth()

	defaultConf := ethconfig.Defaults

	engine := arbos.Engine{
		IsSequencer: true,
	}
	chainDb, err := stack.OpenDatabase("l2chaindata", 0, 0, "", false)
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
	}

	l2GenesysAlloc, err := arbosState.GetGenesisAllocFromArbos(initData)
	if err != nil {
		return nil, nil, err
	}

	genesis := &core.Genesis{
		Config:     config,
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumMainnet"),
		GasLimit:   l2pricing.L2GasLimit,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      l2GenesysAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(l2pricing.InitialGasPriceWei),
	}

	chainConfig, _, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, genesis, defaultConf.OverrideArrowGlacier)
	var configCompatError *params.ConfigCompatError
	if errors.As(genesisErr, &configCompatError) {
		return nil, nil, genesisErr
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: defaultConf.EnablePreimageRecording,
	}
	cacheConfig := &core.CacheConfig{
		TrieCleanLimit:      defaultConf.TrieCleanCache,
		TrieCleanJournal:    stack.ResolvePath(defaultConf.TrieCleanCacheJournal),
		TrieCleanRejournal:  defaultConf.TrieCleanCacheRejournal,
		TrieCleanNoPrefetch: defaultConf.NoPrefetch,
		TrieDirtyLimit:      defaultConf.TrieDirtyCache,
		TrieDirtyDisabled:   defaultConf.NoPruning,
		TrieTimeLimit:       defaultConf.TrieTimeout,
		SnapshotLimit:       defaultConf.SnapshotCache,
		Preimages:           defaultConf.Preimages,
	}

	blockChain, err := core.NewBlockChain(chainDb, cacheConfig, chainConfig, engine, vmConfig, shouldPreserveFalse, &defaultConf.TxLookupLimit)
	if err != nil {
		return nil, nil, err
	}

	// stack.RegisterAPIs(tracers.APIs(backend.APIBackend))

	return chainDb, blockChain, nil
}

// TODO: is that right?
func shouldPreserveFalse(block *types.Block) bool {
	return false
}
