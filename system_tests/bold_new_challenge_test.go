// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/bold/challenge"
	modes "github.com/offchainlabs/nitro/bold/challenge/types"
	"github.com/offchainlabs/nitro/bold/commitment/history"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/protocol"
	"github.com/offchainlabs/nitro/bold/protocol/sol"
	"github.com/offchainlabs/nitro/bold/state"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/nitro/init"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/server_common"
)

type incorrectBlockStateProvider struct {
	honest              BoldStateProviderInterface
	chain               protocol.AssertionChain
	wrongAtFirstVirtual bool
	wrongAtBlockHeight  uint64
	honestMachineHash   common.Hash
	evilMachineHash     common.Hash
}

func (s *incorrectBlockStateProvider) ExecutionStateAfterPreviousState(
	ctx context.Context,
	maxInboxCount uint64,
	previousGlobalState protocol.GoGlobalState,
) (*protocol.ExecutionState, error) {
	maxNumberOfBlocks := s.chain.SpecChallengeManager().LayerZeroHeights().BlockChallengeHeight.Uint64()
	executionState, err := s.honest.ExecutionStateAfterPreviousState(ctx, maxInboxCount, previousGlobalState)
	if err != nil {
		return nil, err
	}
	evilStates, err := s.L2MessageStatesUpTo(ctx, previousGlobalState, state.Batch(maxInboxCount), option.Some(state.Height(maxNumberOfBlocks)))
	if err != nil {
		return nil, err
	}
	historyCommit, err := history.NewCommitment(evilStates, maxNumberOfBlocks+1)
	if err != nil {
		return nil, err
	}
	executionState.EndHistoryRoot = historyCommit.Merkle
	return executionState, nil
}

func (s *incorrectBlockStateProvider) L2MessageStatesUpTo(
	ctx context.Context,
	fromState protocol.GoGlobalState,
	batchLimit state.Batch,
	toHeight option.Option[state.Height],
) ([]common.Hash, error) {
	states, err := s.honest.L2MessageStatesUpTo(ctx, fromState, batchLimit, toHeight)
	if err != nil {
		return nil, err
	}
	// Double check that virtual blocks aren't being enumerated by the honest impl
	for i := len(states) - 1; i >= 1; i-- {
		if states[i] == states[i-1] {
			panic("Virtual block found repeated in honest impl (test case currently doesn't accomodate this)")
		} else {
			break
		}
	}
	if s.wrongAtFirstVirtual && (toHeight.IsNone() || uint64(len(states)) < uint64(toHeight.Unwrap())) {
		// We've found the first virtual block, now let's make it wrong
		s.wrongAtFirstVirtual = false
		s.wrongAtBlockHeight = uint64(len(states))
	}
	if toHeight.IsNone() || uint64(toHeight.Unwrap()) >= s.wrongAtBlockHeight {
		for uint64(len(states)) <= s.wrongAtBlockHeight {
			states = append(states, states[len(states)-1])
		}
		s.honestMachineHash = states[s.wrongAtBlockHeight]
		states[s.wrongAtBlockHeight][0] ^= 0xFF
		s.evilMachineHash = states[s.wrongAtBlockHeight]
		if uint64(len(states)) == s.wrongAtBlockHeight+1 && (toHeight.IsNone() || uint64(len(states)) < uint64(toHeight.Unwrap())) {
			// don't break the end inclusion proof
			states = append(states, s.honestMachineHash)
		}
	}
	return states, nil
}

func (s *incorrectBlockStateProvider) CollectMachineHashes(
	ctx context.Context, cfg *state.HashCollectorConfig,
) ([]common.Hash, error) {
	honestHashes, err := s.honest.CollectMachineHashes(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if uint64(cfg.BlockChallengeHeight)+1 == s.wrongAtBlockHeight {
		if uint64(len(honestHashes)) < cfg.NumDesiredHashes && honestHashes[len(honestHashes)-1] == s.honestMachineHash {
			honestHashes = append(honestHashes, s.evilMachineHash)
		}
	} else if uint64(cfg.BlockChallengeHeight) >= s.wrongAtBlockHeight {
		panic(fmt.Sprintf("challenge occured at block height %v at or after wrongAtBlockHeight %v", cfg.BlockChallengeHeight, s.wrongAtBlockHeight))
	}
	return honestHashes, nil
}

func (s *incorrectBlockStateProvider) CollectProof(
	ctx context.Context,
	assertionMetadata *state.AssociatedAssertionMetadata,
	blockChallengeHeight state.Height,
	machineIndex state.OpcodeIndex,
) ([]byte, error) {
	return s.honest.CollectProof(ctx, assertionMetadata, blockChallengeHeight, machineIndex)
}

func testChallengeProtocolBOLDVirtualBlocks(t *testing.T, wrongAtFirstVirtual bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithExtraWeight(3)

	// Block validation requires db hash scheme
	builder.RequireScheme(t, rawdb.HashScheme)
	builder.nodeConfig.BlockValidator.Enable = true
	builder.valnodeConfig.UseJit = false

	cleanup := builder.Build(t)
	defer cleanup()

	evilNodeConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	evilNodeConfig.BlockValidator.Enable = true
	evilNode, cleanupEvilNode := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: evilNodeConfig,
	})
	defer cleanupEvilNode()

	go keepChainMoving(t, ctx, builder.L1Info, builder.L1.Client)

	builder.L1Info.GenerateAccount("HonestAsserter")
	fundBoldStaker(t, ctx, builder, "HonestAsserter")
	builder.L1Info.GenerateAccount("EvilAsserter")
	fundBoldStaker(t, ctx, builder, "EvilAsserter")

	l2Params := boldChallengeManagerParams{
		rollupAddr:   builder.addresses.Rollup,
		parentClient: builder.L1.Client,
		parentInfo:   builder.L1Info,
		l1Reader:     builder.L2.ConsensusNode.L1Reader,
		nodeConfig:   builder.nodeConfig,
	}

	assertionChain, cleanupHonestChallengeManager := startBoldChallengeManager(t, ctx, l2Params, builder.L2, "HonestAsserter", nil)
	defer cleanupHonestChallengeManager()

	_, cleanupEvilChallengeManager := startBoldChallengeManager(t, ctx, l2Params, evilNode, "EvilAsserter", func(stateManager BoldStateProviderInterface) BoldStateProviderInterface {
		p := &incorrectBlockStateProvider{
			honest:              stateManager,
			chain:               assertionChain,
			wrongAtFirstVirtual: wrongAtFirstVirtual,
		}
		if !wrongAtFirstVirtual {
			p.wrongAtBlockHeight = blockChallengeLeafHeight - 2
		}
		return p
	})
	defer cleanupEvilChallengeManager()

	TransferBalance(t, "Faucet", "Faucet", common.Big0, builder.L2Info, builder.L2.Client, ctx)

	// Everything's setup, now just wait for the challenge to complete and ensure the honest party won
	waitForHonestOSPWin(t, ctx, builder.L1.Client, assertionChain.SpecChallengeManager().Address(), builder.L1Info.GetAddress("HonestAsserter"), time.Second)
}

func fundBoldStaker(t *testing.T, ctx context.Context, builder *NodeBuilder, name string) {
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", name, balance, builder.L1Info, builder.L1.Client, ctx)

	rollupUserLogic, err := rollupgen.NewRollupUserLogic(builder.addresses.Rollup, builder.L1.Client)
	Require(t, err)
	stakeToken, err := rollupUserLogic.StakeToken(&bind.CallOpts{Context: ctx})
	Require(t, err)
	stakeTokenWeth, err := mocksgen.NewTestWETH9(stakeToken, builder.L1.Client)
	Require(t, err)

	txOpts := builder.L1Info.GetDefaultTransactOpts(name, ctx)

	txOpts.Value = big.NewInt(params.Ether)
	tx, err := stakeTokenWeth.Deposit(&txOpts)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)
	txOpts.Value = nil

	tx, err = stakeTokenWeth.Approve(&txOpts, builder.addresses.Rollup, balance)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)

	challengeManager, err := rollupUserLogic.ChallengeManager(&bind.CallOpts{Context: ctx})
	Require(t, err)
	tx, err = stakeTokenWeth.Approve(&txOpts, challengeManager, balance)
	Require(t, err)
	_, err = builder.L1.EnsureTxSucceeded(tx)
	Require(t, err)
}

func newTestHistoryProvider(sm BoldStateProviderInterface) *state.HistoryCommitmentProvider {
	return state.NewHistoryCommitmentProvider(
		sm, sm, sm,
		[]state.Height{
			state.Height(blockChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(bigStepChallengeLeafHeight),
			state.Height(smallStepChallengeLeafHeight),
		},
		sm,
		nil,
	)
}

func waitForHonestOSPWin(
	t *testing.T,
	ctx context.Context,
	client *ethclient.Client,
	chalManagerAddr common.Address,
	honestAddr common.Address,
	pollInterval time.Duration,
) {
	t.Helper()
	filterer, err := challengeV2gen.NewEdgeChallengeManagerFilterer(chalManagerAddr, client)
	Require(t, err)

	fromBlock := uint64(0)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := client.HeaderByNumber(ctx, nil)
			Require(t, err)
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			it, err := filterer.FilterEdgeConfirmedByOneStepProof(filterOpts, nil, nil)
			Require(t, err)
			for it.Next() {
				if it.Error() != nil {
					t.Fatalf("Error in filter iterator: %v", it.Error())
				}
				t.Log("Received event of OSP confirmation!")
				tx, _, err := client.TransactionByHash(ctx, it.Event.Raw.TxHash)
				Require(t, err)
				signer := types.NewCancunSigner(tx.ChainId())
				address, err := signer.Sender(tx)
				Require(t, err)
				if address == honestAddr {
					t.Log("Honest party won OSP, impossible for evil party to win if honest party continues")
					Require(t, it.Close())
					return
				}
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

func fastChallengeStackOpts(name string, headerProvider challenge.HeaderProvider) []challenge.StackOpt {
	return []challenge.StackOpt{
		challenge.StackWithName(name),
		challenge.StackWithMode(modes.MakeMode),
		challenge.StackWithPostingInterval(200 * time.Millisecond),
		challenge.StackWithPollingInterval(100 * time.Millisecond),
		challenge.StackWithConfirmationInterval(200 * time.Millisecond),
		challenge.StackWithMinimumGapToParentAssertion(0),
		challenge.StackWithAverageBlockCreationTime(100 * time.Millisecond),
		challenge.StackWithHeaderProvider(headerProvider),
	}
}

func createEvilAssertionChain(
	t *testing.T,
	ctx context.Context,
	rollupAddr common.Address,
	chalManagerAddr common.Address,
	l1client *ethclient.Client,
	evilNode *arbnode.Node,
	evilOpts *bind.TransactOpts,
	nodeConfig *arbnode.Config,
) *sol.AssertionChain {
	t.Helper()
	l1ChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(evilNode.ConsensusDB, storage.StakerPrefix),
		evilNode.L1Reader,
		evilOpts,
		NewCommonConfigFetcher(nodeConfig),
		evilNode.SyncMonitor,
		l1ChainId,
	)
	Require(t, err)
	chain, err := sol.NewAssertionChain(
		ctx,
		rollupAddr,
		chalManagerAddr,
		evilOpts,
		l1client,
		bold.NewDataPosterTransactor(dp),
		sol.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
		sol.WithParentChainBlockCreationTime(10*time.Millisecond),
	)
	Require(t, err)
	return chain
}

// createSecondL2Node creates a second L2 node that shares an L1 chain with the first node.
// The addresses parameter determines which rollup contracts the new node connects to.
func createSecondL2Node(
	t *testing.T,
	ctx context.Context,
	first *arbnode.Node,
	l1info *BlockchainTestInfo,
	l1client *ethclient.Client,
	l2InitData *statetransfer.ArbosInitializationInfo,
	addresses *chaininfo.RollupAddresses,
	nodeConfig *arbnode.Config,
	stackConfig *node.Config,
) (*node.Node, *ethclient.Client, *arbnode.Node, *gethexec.ExecutionNode) {
	t.Helper()
	fatalErrChan := make(chan error, 10)

	firstExec, ok := first.ExecutionClient.(*gethexec.ExecutionNode)
	if !ok {
		Fatal(t, "not geth execution node")
	}
	chainConfig := firstExec.ArbInterface.BlockChain().Config()

	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 18
	if stackConfig == nil {
		stackConfig = testhelpers.CreateStackConfigForTest(t.TempDir())
		stackConfig.DBEngine = rawdb.DBPebble
	}
	l2stack, err := node.New(stackConfig)
	Require(t, err)

	l2executionDB, err := l2stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)
	l2consensusDB, err := l2stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)

	AddValNodeIfNeeded(t, ctx, nodeConfig, true, "", "")

	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	txOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	initReader := statetransfer.NewMemoryInitDataReader(l2InitData)
	initMessage, err := nitroinit.GetConsensusParsedInitMsg(ctx, true, chainConfig.ChainID, l1client, first.DeployInfo, chainConfig)
	Require(t, err)

	execConfig := ExecConfigDefaultNonSequencerTest(t, rawdb.HashScheme)
	Require(t, execConfig.Validate())
	coreCacheConfig := gethexec.DefaultCacheConfigFor(&execConfig.Caching)
	l2blockchain, err := gethexec.WriteOrTestBlockChain(l2executionDB, coreCacheConfig, initReader, chainConfig, nil, nil, initMessage, &execConfig.TxIndexer, 0, execConfig.ExposeMultiGas)
	Require(t, err)

	l1ChainId, err := l1client.ChainID(ctx)
	Require(t, err)
	execNode, err := gethexec.CreateExecutionNode(ctx, l2stack, l2executionDB, l2blockchain, l1client, NewCommonConfigFetcher(execConfig), l1ChainId, 0)
	Require(t, err)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	l2node, err := arbnode.CreateConsensusNode(ctx, l2stack, execNode, l2consensusDB, NewCommonConfigFetcher(nodeConfig), l2blockchain.Config(), l1client, addresses, &txOpts, &txOpts, dataSigner, fatalErrChan, l1ChainId, nil /* blob reader */, locator.LatestWasmModuleRoot())
	Require(t, err)

	l2client := ClientForStack(t, l2stack, clientForStackUseHTTP(stackConfig))

	StartWatchChanErr(t, ctx, fatalErrChan, l2node)

	return l2stack, l2client, l2node, execNode
}

func runFastChallengeAndAssertHonestWin(
	t *testing.T,
	ctx context.Context,
	honestChain, evilChain *sol.AssertionChain,
	honestSM, evilSM BoldStateProviderInterface,
	honestNode, evilNode *arbnode.Node,
	parentClient *ethclient.Client,
	honestAddr common.Address,
) {
	t.Helper()
	provider := newTestHistoryProvider(honestSM)
	evilProvider := newTestHistoryProvider(evilSM)

	manager, err := challenge.NewChallengeStack(
		honestChain,
		provider,
		fastChallengeStackOpts("honest", honestNode.L1Reader)...,
	)
	Require(t, err)

	managerB, err := challenge.NewChallengeStack(
		evilChain,
		evilProvider,
		fastChallengeStackOpts("evil", evilNode.L1Reader)...,
	)
	Require(t, err)

	manager.Start(ctx)
	managerB.Start(ctx)

	waitForHonestOSPWin(t, ctx, parentClient, honestChain.SpecChallengeManager().Address(), honestAddr, 50*time.Millisecond)
}

func TestChallengeProtocolBOLDNearLastVirtualBlock(t *testing.T) {
	t.Skip("This test is flaky and needs to be fixed")
	testChallengeProtocolBOLDVirtualBlocks(t, false)
}

func TestChallengeProtocolBOLDFirstVirtualBlock(t *testing.T) {
	t.Skip("This test is flaky and needs to be fixed")
	testChallengeProtocolBOLDVirtualBlocks(t, true)
}

type BoldStateProviderInterface interface {
	state.L2MessageStateCollector
	state.MachineHashCollector
	state.ProofCollector
	state.ExecutionProvider
}

type boldChallengeManagerParams struct {
	rollupAddr   common.Address
	parentClient *ethclient.Client
	parentInfo   *BlockchainTestInfo
	l1Reader     *headerreader.HeaderReader
	nodeConfig   *arbnode.Config
}

func startBoldChallengeManager(t *testing.T, ctx context.Context, params boldChallengeManagerParams, node *TestClient, addressName string, mockStateProvider func(BoldStateProviderInterface) BoldStateProviderInterface) (*sol.AssertionChain, func()) {
	var stateManager BoldStateProviderInterface
	var err error
	cacheDir := t.TempDir()
	stateManager, err = bold.NewBOLDStateProvider(
		node.ConsensusNode.BlockValidator,
		node.ConsensusNode.StatelessBlockValidator,
		state.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{
			ValidatorName:          addressName,
			MachineLeavesCachePath: cacheDir,
			CheckBatchFinality:     false,
		},
		cacheDir,
		node.ConsensusNode.InboxTracker,
		node.ConsensusNode.TxStreamer,
		node.ConsensusNode.InboxReader,
		nil,
	)
	Require(t, err)

	if mockStateProvider != nil {
		stateManager = mockStateProvider(stateManager)
	}

	provider := newTestHistoryProvider(stateManager)

	rollupUserLogic, err := rollupgen.NewRollupUserLogic(params.rollupAddr, params.parentClient)
	Require(t, err)
	chalManagerAddr, err := rollupUserLogic.ChallengeManager(&bind.CallOpts{})
	Require(t, err)

	txOpts := params.parentInfo.GetDefaultTransactOpts(addressName, ctx)

	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(node.ConsensusNode.ConsensusDB, storage.StakerPrefix),
		params.l1Reader,
		&txOpts,
		NewCommonConfigFetcher(params.nodeConfig),
		node.ConsensusNode.SyncMonitor,
		params.parentInfo.Signer.ChainID(),
	)
	Require(t, err)

	assertionChain, err := sol.NewAssertionChain(
		ctx,
		params.rollupAddr,
		chalManagerAddr,
		&txOpts,
		params.parentClient,
		bold.NewDataPosterTransactor(dp),
		sol.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
	)
	Require(t, err)

	stackOpts := []challenge.StackOpt{
		challenge.StackWithName(addressName),
		challenge.StackWithMode(modes.MakeMode),
		challenge.StackWithPostingInterval(time.Second * 3),
		challenge.StackWithPollingInterval(time.Second),
		challenge.StackWithAverageBlockCreationTime(time.Second),
		challenge.StackWithMinimumGapToParentAssertion(0),
	}

	challengeManager, err := challenge.NewChallengeStack(
		assertionChain,
		provider,
		stackOpts...,
	)
	Require(t, err)

	challengeManager.Start(ctx)
	return assertionChain, challengeManager.StopAndWait
}
