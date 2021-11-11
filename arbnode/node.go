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
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

type RollupAddresses struct {
	Bridge         common.Address
	Inbox          common.Address
	SequencerInbox common.Address
	DeployedAt     uint64
}

func DeployOnL1(ctx context.Context, l1client L1Interface, deployAuth *bind.TransactOpts, sequencer common.Address) (*RollupAddresses, error) {
	bridgeAddr, tx, bridgeContract, err := bridgegen.DeployBridge(deployAuth, l1client)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(ctx, l1client, tx); err != nil {
		return nil, err
	}

	inboxAddr, tx, inboxContract, err := bridgegen.DeployInbox(deployAuth, l1client)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(ctx, l1client, tx); err != nil {
		return nil, err
	}

	tx, err = inboxContract.Initialize(deployAuth, bridgeAddr)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(ctx, l1client, tx); err != nil {
		return nil, err
	}

	tx, err = bridgeContract.Initialize(deployAuth)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(ctx, l1client, tx); err != nil {
		return nil, err
	}

	tx, err = bridgeContract.SetInbox(deployAuth, inboxAddr, true)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(ctx, l1client, tx); err != nil {
		return nil, err
	}

	sequencerInboxAddr, tx, _, err := bridgegen.DeploySequencerInbox(deployAuth, l1client, bridgeAddr, sequencer)
	if err != nil {
		return nil, err
	}
	txRes, err := EnsureTxSucceeded(ctx, l1client, tx)
	if err != nil {
		return nil, err
	}

	return &RollupAddresses{
		Bridge:         bridgeAddr,
		Inbox:          inboxAddr,
		SequencerInbox: sequencerInboxAddr,
		DeployedAt:     txRes.BlockNumber.Uint64(),
	}, nil
}

type Node struct {
	Backend          *arbitrum.Backend
	Sequencer        *Sequencer
	DeployInfo       *RollupAddresses
	InboxReader      *InboxReader
	BatchPoster      *BatchPoster
	DelayedSequencer *DelayedSequencer
	TxStreamer       *InboxState
	InboxTracker     *InboxReaderDb
}

func CreateNode(l1client L1Interface, deployInfo *RollupAddresses, l2backend *arbitrum.Backend, sequencerTxOpt *bind.TransactOpts, isTest bool) (*Node, error) {
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
	inboxReaderConfig := *DefaultInboxReaderConfig
	if isTest {
		inboxReaderConfig.CheckDelay = time.Millisecond * 10
		inboxReaderConfig.DelayBlocks = 0
	}
	sequencerObj, ok := l2backend.Publisher().(*Sequencer)
	if !ok {
		return nil, errors.New("l2backend doesn't have a sequencer")
	}
	inbox := sequencerObj.InboxState()
	inboxReader, err := NewInboxReader(l2backend.InboxDb(), inbox, l1client, new(big.Int).SetUint64(deployInfo.DeployedAt), delayedBridge, sequencerInbox, &inboxReaderConfig)
	if err != nil {
		return nil, err
	}
	inboxTracker := inboxReader.Database()
	delayedSequencerConfig := *DefaultDelayedSequencerConfig
	if isTest {
		// not necessary, but should help prevent spurious failures in delayed sequencer test
		delayedSequencerConfig.TimeAggregate = time.Second
	}
	var delayedSequencer *DelayedSequencer
	var batchPoster *BatchPoster
	if sequencerTxOpt != nil {
		delayedSequencer, err = NewDelayedSequencer(l1client, inboxReader, inbox, &delayedSequencerConfig)
		if err != nil {
			return nil, err
		}
		batchPoster, err = NewBatchPoster(l1client, inboxTracker, inbox, &DefaultBatchPosterConfig, deployInfo.SequencerInbox, common.Address{}, sequencerTxOpt)
		if err != nil {
			return nil, err
		}
	}
	return &Node{l2backend, sequencerObj, deployInfo, inboxReader, batchPoster, delayedSequencer, inbox, inboxTracker}, nil
}

func (n *Node) Start(ctx context.Context) {
	if n.DelayedSequencer != nil {
		n.DelayedSequencer.Start(ctx)
	}
	n.InboxReader.Start(ctx)
	if n.BatchPoster != nil {
		n.BatchPoster.Start()
	}
	n.Sequencer.Start(ctx)
}

func (n *Node) Stop() {
	if n.BatchPoster != nil {
		n.BatchPoster.Stop()
	}
}

func CreateStack() (*node.Node, error) {
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

func CreateArbBackend(ctx context.Context, stack *node.Node, genesis *core.Genesis, l1Client L1Interface) (*arbitrum.Backend, error) {
	arbstate.RequireHookedGeth()

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()

	nodeConf.Genesis = genesis

	engine := arbos.Engine{
		IsSequencer: true,
	}
	chainDb, err := stack.OpenDatabase("l2chaindata", 0, 0, "", false)
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
	}
	chainConfig, _, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, nodeConf.Genesis, nodeConf.OverrideLondon)
	var configCompatError *params.ConfigCompatError
	if errors.As(genesisErr, &configCompatError) {
		return nil, genesisErr
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: nodeConf.EnablePreimageRecording,
	}
	cacheConfig := &core.CacheConfig{
		TrieCleanLimit:      nodeConf.TrieCleanCache,
		TrieCleanJournal:    stack.ResolvePath(nodeConf.TrieCleanCacheJournal),
		TrieCleanRejournal:  nodeConf.TrieCleanCacheRejournal,
		TrieCleanNoPrefetch: nodeConf.NoPrefetch,
		TrieDirtyLimit:      nodeConf.TrieDirtyCache,
		TrieDirtyDisabled:   nodeConf.NoPruning,
		TrieTimeLimit:       nodeConf.TrieTimeout,
		SnapshotLimit:       nodeConf.SnapshotCache,
		Preimages:           nodeConf.Preimages,
	}

	blockChain, err := core.NewBlockChain(chainDb, cacheConfig, chainConfig, engine, vmConfig, shouldPreserveFalse, &nodeConf.TxLookupLimit)
	if err != nil {
		return nil, err
	}

	inboxDb, err := stack.OpenDatabase("l2inbox", 0, 0, "", false)
	if err != nil {
		utils.Fatalf("Failed to open inbox database: %v", err)
	}
	inbox, err := NewInboxState(inboxDb, blockChain)
	if err != nil {
		return nil, err
	}

	inbox.Start(ctx)

	sequencer, err := NewSequencer(ctx, inbox, l1Client)
	if err != nil {
		return nil, err
	}

	backend, err := arbitrum.NewBackend(stack, &nodeConf, chainDb, inboxDb, blockChain, arbos.ChainConfig.ChainID, sequencer)
	if err != nil {
		return nil, err
	}

	// stack.RegisterAPIs(tracers.APIs(backend.APIBackend))

	return backend, nil
}

// TODO: is that right?
func shouldPreserveFalse(block *types.Block) bool {
	return false
}
