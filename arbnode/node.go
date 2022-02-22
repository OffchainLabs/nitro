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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/broadcastclient"
	"github.com/offchainlabs/arbstate/broadcaster"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
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

func andTxSucceeded(ctx context.Context, l1client arbutil.L1Interface, txTimeout time.Duration, tx *types.Transaction, err error) error {
	if err != nil {
		return fmt.Errorf("error submitting tx: %w", err)
	}
	_, err = arbutil.EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout)
	if err != nil {
		return fmt.Errorf("error executing tx: %w", err)
	}
	return nil
}

func deployBridgeCreator(ctx context.Context, client arbutil.L1Interface, auth *bind.TransactOpts, txTimeout time.Duration) (common.Address, error) {
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

func deployChallengeFactory(
	ctx context.Context,
	client arbutil.L1Interface,
	auth *bind.TransactOpts,
	txTimeout time.Duration,
) (common.Address, common.Address, error) {
	osp0, tx, _, err := ospgen.DeployOneStepProver0(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("osp0 deploy error: %w", err)
	}

	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMemory deploy error: %w", err)
	}

	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospMath deploy error: %w", err)
	}

	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospHostIo deploy error: %w", err)
	}

	ospEntryAddr, tx, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospEntry deploy error: %w", err)
	}

	challengeManagerAddr, tx, _, err := challengegen.DeployChallengeManager(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return common.Address{}, common.Address{}, fmt.Errorf("ospEntry deploy error: %w", err)
	}

	return ospEntryAddr, challengeManagerAddr, nil
}

func deployRollupCreator(ctx context.Context, client arbutil.L1Interface, auth *bind.TransactOpts, txTimeout time.Duration) (*rollupgen.RollupCreator, common.Address, error) {
	bridgeCreator, err := deployBridgeCreator(ctx, client, auth, txTimeout)
	if err != nil {
		return nil, common.Address{}, err
	}

	ospEntryAddr, challengeManagerAddr, err := deployChallengeFactory(ctx, client, auth, txTimeout)
	if err != nil {
		return nil, common.Address{}, err
	}

	rollupAdminLogic, tx, _, err := rollupgen.DeployRollupAdminLogic(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("rollup admin logic deploy error: %w", err)
	}

	rollupUserLogic, tx, _, err := rollupgen.DeployRollupUserLogic(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	rollupCreatorAddress, tx, rollupCreator, err := rollupgen.DeployRollupCreator(auth, client)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	tx, err = rollupCreator.SetTemplates(
		auth,
		bridgeCreator,
		ospEntryAddr,
		challengeManagerAddr,
		rollupAdminLogic,
		rollupUserLogic,
	)
	err = andTxSucceeded(ctx, client, txTimeout, tx, err)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("rollup user logic deploy error: %w", err)
	}

	return rollupCreator, rollupCreatorAddress, nil
}

func DeployOnL1(ctx context.Context, l1client arbutil.L1Interface, deployAuth *bind.TransactOpts, sequencer common.Address, wasmModuleRoot common.Hash, txTimeout time.Duration) (*RollupAddresses, error) {
	rollupCreator, rollupCreatorAddress, err := deployRollupCreator(ctx, l1client, deployAuth, txTimeout)
	if err != nil {
		return nil, err
	}

	var confirmPeriodBlocks uint64 = 20
	var extraChallengeTimeBlocks uint64 = 200
	seqInboxParams := rollupgen.ISequencerInboxMaxTimeVariation{
		DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
		FutureBlocks:  big.NewInt(12),
		DelaySeconds:  big.NewInt(60 * 60 * 24),
		FutureSeconds: big.NewInt(60 * 60),
	}
	nonce, err := l1client.PendingNonceAt(ctx, rollupCreatorAddress)
	if err != nil {
		return nil, err
	}
	expectedRollupAddr := crypto.CreateAddress(rollupCreatorAddress, nonce+2)
	tx, err := rollupCreator.CreateRollup(
		deployAuth,
		rollupgen.Config{
			ConfirmPeriodBlocks:            confirmPeriodBlocks,
			ExtraChallengeTimeBlocks:       extraChallengeTimeBlocks,
			StakeToken:                     common.Address{},
			BaseStake:                      big.NewInt(params.Ether),
			WasmModuleRoot:                 wasmModuleRoot,
			Owner:                          deployAuth.From,
			LoserStakeEscrow:               common.Address{},
			ChainId:                        big.NewInt(1338),
			SequencerInboxMaxTimeVariation: seqInboxParams,
		},
		expectedRollupAddr,
	)
	if err != nil {
		return nil, fmt.Errorf("error submitting create rollup tx: %w", err)
	}
	receipt, err := arbutil.EnsureTxSucceededWithTimeout(ctx, l1client, tx, txTimeout)
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

func CreateNode(stack *node.Node, chainDb ethdb.Database, config *NodeConfig, l2BlockChain *core.BlockChain, l1client arbutil.L1Interface, deployInfo *RollupAddresses, sequencerTxOpt *bind.TransactOpts) (*Node, error) {
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
		blockValidator, err = validator.NewBlockValidator(inboxTracker, txStreamer, l2BlockChain, &config.BlockValidatorConfig)
		if err != nil {
			return nil, err
		}
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

func ImportBlocksToChainDb(chainDb ethdb.Database, initDataReader statetransfer.StoredBlockReader) (uint64, error) {
	var prevHash common.Hash
	td := big.NewInt(0)
	var blocksInDb uint64
	if initDataReader.More() {
		var err error
		blocksInDb, err = chainDb.Ancients()
		if err != nil {
			return 0, err
		}
	}
	blockNum := uint64(0)
	for ; initDataReader.More(); blockNum++ {
		log.Debug("importing", "blockNum", blockNum)
		storedBlock, err := initDataReader.GetNext()
		if err != nil {
			return blockNum, err
		}
		if blockNum+1 < blocksInDb && initDataReader.More() {
			continue // skip already-imported blocks. Only validate the last.
		}
		storedBlockHash := storedBlock.Header.Hash()
		if blockNum < blocksInDb {
			// validate db and import match
			hashInDb := rawdb.ReadCanonicalHash(chainDb, blockNum)
			if storedBlockHash != hashInDb {
				utils.Fatalf("Import and Database disagree on hashes import: %v, Db: %v", storedBlockHash, hashInDb)
			}
		}
		if blockNum+1 == blocksInDb && blockNum > 0 {
			// we skipped blocks common to DB an import
			prevHash = rawdb.ReadCanonicalHash(chainDb, blockNum-1)
			td = rawdb.ReadTd(chainDb, prevHash, blockNum-1)
		}
		if storedBlock.Header.ParentHash != prevHash {
			utils.Fatalf("Import Block %d, parent hash %v, expected %v", blockNum, storedBlock.Header.ParentHash, prevHash)
		}
		if storedBlock.Header.Number.Cmp(new(big.Int).SetUint64(blockNum)) != 0 {
			panic("unexpected block number in import")
		}
		txs := types.Transactions{}
		for _, txData := range storedBlock.Transactions {
			tx := types.ArbitrumLegacyFromTransactionResult(txData)
			if tx.Hash() != txData.Hash {
				return blockNum, errors.New("bad txHash")
			}
			txs = append(txs, tx)
		}
		receipts := storedBlock.Reciepts
		block := types.NewBlockWithHeader(&storedBlock.Header).WithBody(txs, nil) // don't recalculate hashes
		blockHash := block.Hash()
		if blockHash != storedBlock.Header.Hash() {
			return blockNum, errors.New("bad blockHash")
		}
		_, err = rawdb.WriteAncientBlocks(chainDb, []*types.Block{block}, []types.Receipts{receipts}, td)
		if err != nil {
			return blockNum, err
		}
		prevHash = blockHash
		td.Add(td, storedBlock.Header.Difficulty)
	}
	return blockNum, initDataReader.Close()
}

func WriteOrTestGenblock(chainDb ethdb.Database, initData statetransfer.InitDataReader, blockNumber uint64) error {
	arbstate.RequireHookedGeth()

	EmptyHash := common.Hash{}

	prevHash := EmptyHash
	genDifficulty := big.NewInt(1)
	prevDifficulty := big.NewInt(0)
	storedGenHash := rawdb.ReadCanonicalHash(chainDb, blockNumber)
	timestamp := uint64(0)
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
		Nonce:      types.EncodeNonce(1), // the genesis block reads the init message
		Time:       timestamp,
		ParentHash: prevHash,
		Extra:      []byte("ArbitrumMainnet"),
		GasLimit:   l2pricing.L2GasLimit,
		GasUsed:    0,
		BaseFee:    big.NewInt(l2pricing.InitialBaseFeeWei),
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

func GetBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, config *params.ChainConfig) (*core.BlockChain, error) {
	defaultConf := ethconfig.Defaults

	engine := arbos.Engine{
		IsSequencer: true,
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: defaultConf.EnablePreimageRecording,
	}

	return core.NewBlockChain(chainDb, cacheConfig, config, engine, vmConfig, shouldPreserveFalse, &defaultConf.TxLookupLimit)
}

func WriteOrTestBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, initData statetransfer.InitDataReader, blockNumber uint64, config *params.ChainConfig) (*core.BlockChain, error) {
	err := WriteOrTestGenblock(chainDb, initData, blockNumber)
	if err != nil {
		return nil, err
	}
	err = WriteOrTestChainConfig(chainDb, config)
	if err != nil {
		return nil, err
	}
	return GetBlockChain(chainDb, cacheConfig, config)
}

// TODO: is that right?
func shouldPreserveFalse(block *types.Block) bool {
	return false
}
