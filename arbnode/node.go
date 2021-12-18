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
	"github.com/offchainlabs/arbstate/arbstate"
	nitrobroadcaster "github.com/offchainlabs/arbstate/broadcaster"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
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

	tx, err = inboxContract.Initialize(deployAuth, bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("error submitting inbox initialize tx: %w", err)
	}
	if _, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout); err != nil {
		return nil, fmt.Errorf("error executing inbox initialize tx: %w", err)
	}

	tx, err = bridgeContract.Initialize(deployAuth)
	if err != nil {
		return nil, fmt.Errorf("error submitting bridge initialize tx: %w", err)
	}
	if _, err := EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout); err != nil {
		return nil, fmt.Errorf("error executing bridge initialize tx: %w", err)
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
}

var NodeConfigDefault = NodeConfig{arbitrum.DefaultConfig, true, DefaultInboxReaderConfig, DefaultDelayedSequencerConfig, true, DefaultBatchPosterConfig, "", false, validator.DefaultBlockValidatorConfig, false, wsbroadcastserver.DefaultBroadcasterConfig}
var NodeConfigL1Test = NodeConfig{arbitrum.DefaultConfig, true, TestInboxReaderConfig, DefaultDelayedSequencerConfig, true, TestBatchPosterConfig, "", false, validator.DefaultBlockValidatorConfig, false, wsbroadcastserver.DefaultBroadcasterConfig}
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
}

func CreateNode(stack *node.Node, chainDb ethdb.Database, config *NodeConfig, l2BlockChain *core.BlockChain, l1client L1Interface, deployInfo *RollupAddresses, sequencerTxOpt *bind.TransactOpts) (*Node, error) {
	var broadcaster *nitrobroadcaster.Broadcaster
	if config.BatchPoster && config.Broadcaster {
		broadcaster = nitrobroadcaster.NewBroadcaster(config.BroadcasterConfig)
	}
	txStreamer, err := NewTransactionStreamer(chainDb, l2BlockChain, broadcaster)
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

	if !config.L1Reader {
		return &Node{backend, arbInterface, txStreamer, txPublisher, nil, nil, nil, nil, nil, nil}, nil
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
		blockValidator = validator.NewBlockValidator(inboxTracker, txStreamer, &config.BlockValidatorConfig)
	}

	if !config.BatchPoster {
		return &Node{backend, arbInterface, txStreamer, txPublisher, deployInfo, inboxReader, inboxTracker, nil, nil, blockValidator}, nil
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
	return &Node{backend, arbInterface, txStreamer, txPublisher, deployInfo, inboxReader, inboxTracker, delayedSequencer, batchPoster, blockValidator}, nil
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

func CreateDefaultBlockChain(stack *node.Node, genesis *core.Genesis) (ethdb.Database, *core.BlockChain, error) {
	arbstate.RequireHookedGeth()

	defaultConf := ethconfig.Defaults

	engine := arbos.Engine{
		IsSequencer: true,
	}
	chainDb, err := stack.OpenDatabase("l2chaindata", 0, 0, "", false)
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
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
