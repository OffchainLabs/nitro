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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/broadcastclient"
	"github.com/offchainlabs/arbstate/broadcaster"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/statetransfer"
	"github.com/offchainlabs/arbstate/validator"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

type RollupAddresses struct {
	Bridge         common.Address
	Inbox          common.Address
	SequencerInbox common.Address
	DeployedAt     uint64
}

func DeployOnL1(ctx context.Context, l1client L1Interface, deployAuth *bind.TransactOpts, sequencer common.Address, txTimeout time.Duration) (*RollupAddresses, error) {
	bridgeAddr, tx, bridgeContract, err := bridgegen.DeployBridge(deployAuth, l1client)
	if err != nil {
		return nil, fmt.Errorf("error submitting bridge deploy tx: %w", err)
	}
	if _, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout); err != nil {
		return nil, fmt.Errorf("error executing bridge deploy tx: %w", err)
	}

	inboxAddr, tx, inboxContract, err := bridgegen.DeployInbox(deployAuth, l1client)
	if err != nil {
		return nil, fmt.Errorf("error executing inbox deploy tx: %w", err)
	}
	if _, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout); err != nil {
		return nil, fmt.Errorf("error executing inbox deploy tx: %w", err)
	}

	tx, err = bridgeContract.Initialize(deployAuth)
	if err != nil {
		return nil, fmt.Errorf("error submitting bridge initialize tx: %w", err)
	}
	if _, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout); err != nil {
		return nil, fmt.Errorf("error executing bridge initialize tx: %w", err)
	}

	tx, err = inboxContract.Initialize(deployAuth, bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("error submitting inbox initialize tx: %w", err)
	}
	if _, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout); err != nil {
		return nil, fmt.Errorf("error executing inbox initialize tx: %w", err)
	}

	tx, err = bridgeContract.SetInbox(deployAuth, inboxAddr, true)
	if err != nil {
		return nil, fmt.Errorf("error submitting set inbox tx: %w", err)
	}
	if _, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout); err != nil {
		return nil, fmt.Errorf("error executing set inbox tx: %w", err)
	}

	sequencerInboxAddr, tx, _, err := bridgegen.DeploySequencerInbox(deployAuth, l1client, bridgeAddr, sequencer)
	if err != nil {
		return nil, fmt.Errorf("error submitting sequencer inbox deploy tx: %w", err)
	}
	txRes, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout)
	if err != nil {
		return nil, fmt.Errorf("error executing sequencer inbox deploy tx: %w", err)
	}

	return &RollupAddresses{
		Bridge:         bridgeAddr,
		Inbox:          inboxAddr,
		SequencerInbox: sequencerInboxAddr,
		DeployedAt:     txRes.BlockNumber.Uint64(),
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
	err = n.TxPublisher.Start(ctx)
	if err != nil {
		return err
	}
	n.TxStreamer.Start(ctx)
	if n.InboxReader != nil {
		n.InboxReader.Start(ctx)
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

func (n *Node) StopAndWait(ctx context.Context) {
	if n.BroadcastClient != nil {
		n.BroadcastClient.StopAndWait()
	}
	if n.BroadcastServer != nil {
		n.BroadcastServer.StopAndWait()
	}
	if n.BlockValidator != nil {
		n.BlockValidator.StopAndWait()
	}
	if n.BatchPoster != nil {
		n.BatchPoster.StopAndWait()
	}
	if n.DelayedSequencer != nil {
		n.DelayedSequencer.StopAndWait()
	}
	if n.InboxReader != nil {
		n.InboxReader.StopAndWait()
	}
	n.TxPublisher.StopAndWait()
	n.TxStreamer.StopAndWait()
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

func DefaultCacheConfigFor(stack *node.Node) *core.CacheConfig {
	defaultConf := ethconfig.Defaults

	return &core.CacheConfig{
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
}

func WriteOrTestGenblock(chainDb ethdb.Database, initData *statetransfer.ArbosInitializationInfo, blockNumber uint64) error {
	arbstate.RequireHookedGeth()

	EmptyHash := common.Hash{}

	prevHash := EmptyHash
	genDifficulty := big.NewInt(1)
	prevDifficulty := big.NewInt(0)
	storedGenHash := rawdb.ReadCanonicalHash(chainDb, blockNumber)
	if blockNumber > 0 {
		prevHash = rawdb.ReadCanonicalHash(chainDb, blockNumber-1)
		if prevHash == EmptyHash {
			return fmt.Errorf("block number %d not found in database", chainDb)
		}
		prevDifficulty = rawdb.ReadTd(chainDb, prevHash, blockNumber-1)
	}
	stateRoot, err := arbosState.InitializeArbosInDatabase(chainDb, initData)
	if err != nil {
		return err
	}
	head := &types.Header{
		Number:     new(big.Int).SetUint64(blockNumber),
		Nonce:      types.EncodeNonce(0),
		Time:       uint64(time.Now().Unix()),
		ParentHash: prevHash,
		Extra:      []byte("ArbitrumMainnet"),
		GasLimit:   l2pricing.L2GasLimit,
		GasUsed:    0,
		BaseFee:    big.NewInt(l2pricing.InitialGasPriceWei),
		Difficulty: genDifficulty,
		MixDigest:  EmptyHash,
		Coinbase:   common.Address{},
		Root:       stateRoot,
	}

	genBlock := types.NewBlock(head, nil, nil, nil, trie.NewStackTrie(nil))
	blockHash := genBlock.Hash()

	if storedGenHash == EmptyHash {
		// chainDb did not have genesis block. Initialize it.
		core.WriteHeadBlock(chainDb, genBlock, prevDifficulty)
	} else if storedGenHash != blockHash {
		return errors.New("database contains data inconsistent with initialization")
	}

	return nil
}

func WriteOrTestChainConfig(chainDb ethdb.Database, config *params.ChainConfig) error {
	EmptyHash := common.Hash{}

	block0Hash := rawdb.ReadCanonicalHash(chainDb, 0)
	if block0Hash == EmptyHash {
		return errors.New("block 0 not found")
	}
	storedConfig := rawdb.ReadChainConfig(chainDb, block0Hash)
	if storedConfig == nil {
		rawdb.WriteChainConfig(chainDb, block0Hash, config)
		return nil
	}
	height := rawdb.ReadHeaderNumber(chainDb, rawdb.ReadHeadHeaderHash(chainDb))
	if height == nil {
		return errors.New("non empty chain config but empty chain")
	}
	err := storedConfig.CheckCompatible(config, *height)
	if err != nil {
		return err
	}
	rawdb.WriteChainConfig(chainDb, block0Hash, config)
	return nil
}

func CreateBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, config *params.ChainConfig) (*core.BlockChain, error) {
	defaultConf := ethconfig.Defaults

	engine := arbos.Engine{
		IsSequencer: true,
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: defaultConf.EnablePreimageRecording,
	}

	return core.NewBlockChain(chainDb, cacheConfig, config, engine, vmConfig, shouldPreserveFalse, &defaultConf.TxLookupLimit)
}

func CreateDefaultBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, initData *statetransfer.ArbosInitializationInfo, blockNumber uint64, config *params.ChainConfig) (*core.BlockChain, error) {
	err := WriteOrTestGenblock(chainDb, initData, blockNumber)
	if err != nil {
		return nil, err
	}
	err = WriteOrTestChainConfig(chainDb, config)
	if err != nil {
		return nil, err
	}
	return CreateBlockChain(chainDb, cacheConfig, config)
}

// TODO: is that right?
func shouldPreserveFalse(block *types.Block) bool {
	return false
}
