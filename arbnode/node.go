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
	TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error)
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

func EnsureTxSucceeded(client L1Interface, tx *types.Transaction) (*types.Receipt, error) {
	txRes, err := WaitForTx(client, tx.Hash(), time.Second)
	if err != nil {
		return nil, err
	}
	if txRes == nil {
		return nil, errors.New("expected receipt")
	}
	if txRes.Status != types.ReceiptStatusSuccessful {
		// Re-execute the transaction as a call to get a better error
		ctx := context.TODO()
		from, err := client.TransactionSender(ctx, tx, txRes.BlockHash, txRes.TransactionIndex)
		if err != nil {
			return nil, err
		}
		callMsg := ethereum.CallMsg{
			From:       from,
			To:         tx.To(),
			Gas:        tx.Gas(),
			GasPrice:   tx.GasPrice(),
			GasFeeCap:  tx.GasFeeCap(),
			GasTipCap:  tx.GasTipCap(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
		}
		_, err = client.CallContract(ctx, callMsg, txRes.BlockNumber)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("tx failed but call succeeded")
	}
	return txRes, nil
}

type RollupAddresses struct {
	Bridge         common.Address
	Inbox          common.Address
	SequencerInbox common.Address
	DeployedAt     uint64
}

func CreateL1WithInbox(l1client L1Interface, l2backend *arbitrum.Backend, deployAuth *bind.TransactOpts, sequencer common.Address, sequencerTxOpt *bind.TransactOpts, isTest bool) (*RollupAddresses, error) {
	bridgeAddr, tx, bridgeContract, err := bridgegen.DeployBridge(deployAuth, l1client)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	inboxAddr, tx, inboxContract, err := bridgegen.DeployInbox(deployAuth, l1client)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	tx, err = inboxContract.Initialize(deployAuth, bridgeAddr)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	tx, err = bridgeContract.Initialize(deployAuth)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	tx, err = bridgeContract.SetInbox(deployAuth, inboxAddr, true)
	if err != nil {
		return nil, err
	}
	if _, err := EnsureTxSucceeded(l1client, tx); err != nil {
		return nil, err
	}

	sequencerInboxAddr, tx, _, err := bridgegen.DeploySequencerInbox(deployAuth, l1client, bridgeAddr, sequencer)
	if err != nil {
		return nil, err
	}
	txRes, err := EnsureTxSucceeded(l1client, tx)
	if err != nil {
		return nil, err
	}

	blockDeployed := txRes.BlockNumber.Uint64()

	delayedBridge, err := NewDelayedBridge(l1client, bridgeAddr, blockDeployed)
	if err != nil {
		return nil, err
	}
	sequencerInbox, err := NewSequencerInbox(l1client, sequencerInboxAddr, int64(blockDeployed))
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
	inboxReader, err := NewInboxReader(l2backend.InboxDb(), inbox, l1client, new(big.Int).SetUint64(blockDeployed), delayedBridge, sequencerInbox, &inboxReaderConfig)
	if err != nil {
		return nil, err
	}
	inboxReader.Start(context.Background())
	delayedSequencerConfig := *DefaultDelayedSequencerConfig
	if isTest {
		// not necessary, but should help prevent spurious failures in delayed sequencer test
		delayedSequencerConfig.TimeAggregate = time.Second
	}
	delayed_sequencer, err := NewDelayedSequencer(l1client, inboxReader, inbox, &delayedSequencerConfig)
	if err != nil {
		return nil, err
	}
	delayed_sequencer.Start(context.Background())
	if sequencerTxOpt != nil {
		batchPoster, err := NewBatchPoster(l1client, inboxReader.Database(), inbox, &DefaultBatchPosterConfig, sequencerInboxAddr, common.Address{}, sequencerTxOpt)
		if err != nil {
			return nil, err
		}
		batchPoster.Start()
	}
	return &RollupAddresses{
		Bridge:         bridgeAddr,
		Inbox:          inboxAddr,
		SequencerInbox: sequencerInboxAddr,
		DeployedAt:     txRes.BlockNumber.Uint64(),
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
		BaseFee:    big.NewInt(params.InitialBaseFee / 100),
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

	inboxDb, err := stack.OpenDatabase("l2inbox", 0, 0, "", false)
	if err != nil {
		utils.Fatalf("Failed to open inbox database: %v", err)
	}
	inbox, err := NewInboxState(inboxDb, blockChain)
	if err != nil {
		return nil, err
	}

	inbox.Start(context.Background())

	sequencer := NewSequencer(inbox)

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
