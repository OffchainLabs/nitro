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

	"github.com/ethereum/go-ethereum"
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

type L1Interface interface {
	bind.ContractBackend
	ethereum.ChainReader
	ethereum.TransactionReader
}

// will wait untill tx is in the blockchain. attempts = 0 is infinite
func WaitForTx(client L1Interface, txhash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx := context.Background()
	chanHead := make(chan *types.Header, 20)
	headSubscribe, err := client.SubscribeNewHead(ctx, chanHead)
	if err != nil {
		return nil, err
	}
	defer headSubscribe.Unsubscribe()

	chTimeout := time.After(timeout)
	for {
		reciept, err := client.TransactionReceipt(ctx, txhash)
		if reciept != nil {
			return reciept, err
		}
		select {
		case <-chanHead:
		case <-chTimeout:
			return nil, errors.New("timeout waiting for transaction")
		}
	}
}

func EnsureTxSucceeded(client L1Interface, tx *types.Transaction) error {
	txRes, err := WaitForTx(client, tx.Hash(), time.Second)
	if err != nil {
		return err
	}
	if txRes == nil {
		return errors.New("expected receipt")
	}
	if txRes.Status != types.ReceiptStatusSuccessful {
		return errors.New("expected tx to succeed")
	}
	return nil
}

type RollupAddresses struct {
	Bridge         common.Address
	Inbox          common.Address
	SequencerInbox common.Address
}

func CreateL1WithInbox(l1client L1Interface, deployAuth *bind.TransactOpts, sequencer common.Address) (*RollupAddresses, error) {
	bridgeAddr, tx, bridgeContract, err := bridgegen.DeployBridge(deployAuth, l1client)
	if err != nil {
		return nil, err
	}
	if err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	inboxAddr, tx, inboxContract, err := bridgegen.DeployInbox(deployAuth, l1client)
	if err != nil {
		return nil, err
	}
	if err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	tx, err = inboxContract.Initialize(deployAuth, bridgeAddr)
	if err != nil {
		return nil, err
	}
	if err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	tx, err = bridgeContract.Initialize(deployAuth)
	if err != nil {
		return nil, err
	}
	if err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	tx, err = bridgeContract.SetInbox(deployAuth, inboxAddr, true)
	if err != nil {
		return nil, err
	}
	if err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	sequencerInboxAddr, tx, _, err := bridgegen.DeploySequencerInbox(deployAuth, l1client, bridgeAddr, sequencer)
	if err != nil {
		return nil, err
	}
	if err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	return &RollupAddresses{
		Bridge:         bridgeAddr,
		Inbox:          inboxAddr,
		SequencerInbox: sequencerInboxAddr,
	}, nil
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

func CreateArbBackend(stack *node.Node, genesisAlloc core.GenesisAlloc) (*arbitrum.Backend, error) {
	arbstate.RequireHookedGeth()

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()

	nodeConf.Genesis = &core.Genesis{
		Config:     arbos.ChainConfig,
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumMainnet"),
		GasLimit:   0,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

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

	inbox, err := NewInboxState(chainDb, blockChain)
	if err != nil {
		return nil, err
	}

	inbox.Start(context.Background())

	sequencer := NewSequencer(inbox)

	backend, err := arbitrum.NewBackend(stack, &nodeConf, chainDb, blockChain, arbos.ChainConfig.ChainID, sequencer)
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
