// Copyright 2024, Offchain Labs, Inc.
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
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
	"github.com/offchainlabs/nitro/bold/challenge-manager"
	modes "github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
	"github.com/offchainlabs/nitro/bold/state-commitments/history"
	"github.com/offchainlabs/nitro/bold/util"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/bold"
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
	evilStates, err := s.L2MessageStatesUpTo(ctx, previousGlobalState, l2stateprovider.Batch(maxInboxCount), option.Some(l2stateprovider.Height(maxNumberOfBlocks)))
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
	batchLimit l2stateprovider.Batch,
	toHeight option.Option[l2stateprovider.Height],
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
	ctx context.Context, cfg *l2stateprovider.HashCollectorConfig,
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
	assertionMetadata *l2stateprovider.AssociatedAssertionMetadata,
	blockChallengeHeight l2stateprovider.Height,
	machineIndex l2stateprovider.OpcodeIndex,
) ([]byte, error) {
	return s.honest.CollectProof(ctx, assertionMetadata, blockChallengeHeight, machineIndex)
}

func testChallengeProtocolBOLDVirtualBlocks(t *testing.T, wrongAtFirstVirtual bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)

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

	assertionChain, cleanupHonestChallengeManager := startBoldChallengeManager(t, ctx, builder, builder.L2, "HonestAsserter", nil)
	defer cleanupHonestChallengeManager()

	_, cleanupEvilChallengeManager := startBoldChallengeManager(t, ctx, builder, evilNode, "EvilAsserter", func(stateManager BoldStateProviderInterface) BoldStateProviderInterface {
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

	chalManager := assertionChain.SpecChallengeManager()
	filterer, err := challengeV2gen.NewEdgeChallengeManagerFilterer(chalManager.Address(), builder.L1.Client)
	Require(t, err)

	fromBlock := uint64(0)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := builder.L1.Client.HeaderByNumber(ctx, nil)
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
				tx, _, err := builder.L1.Client.TransactionByHash(ctx, it.Event.Raw.TxHash)
				Require(t, err)
				signer := types.NewCancunSigner(tx.ChainId())
				address, err := signer.Sender(tx)
				Require(t, err)
				if address == builder.L1Info.GetAddress("HonestAsserter") {
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

func TestChallengeProtocolBOLDNearLastVirtualBlock(t *testing.T) {
	t.Skip("This test is flaky and needs to be fixed")
	testChallengeProtocolBOLDVirtualBlocks(t, false)
}

func TestChallengeProtocolBOLDFirstVirtualBlock(t *testing.T) {
	t.Skip("This test is flaky and needs to be fixed")
	testChallengeProtocolBOLDVirtualBlocks(t, true)
}

type BoldStateProviderInterface interface {
	l2stateprovider.L2MessageStateCollector
	l2stateprovider.MachineHashCollector
	l2stateprovider.ProofCollector
	l2stateprovider.ExecutionProvider
}

func startBoldChallengeManager(t *testing.T, ctx context.Context, builder *NodeBuilder, node *TestClient, addressName string, mockStateProvider func(BoldStateProviderInterface) BoldStateProviderInterface) (*solimpl.AssertionChain, func()) {
	if !builder.deployBold {
		t.Fatal("bold deployment not enabled")
	}

	var stateManager BoldStateProviderInterface
	var err error
	cacheDir := t.TempDir()
	stateManager, err = bold.NewBOLDStateProvider(
		node.ConsensusNode.BlockValidator,
		node.ConsensusNode.StatelessBlockValidator,
		l2stateprovider.Height(blockChallengeLeafHeight),
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

	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateManager,
		stateManager,
		stateManager,
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		stateManager,
		nil, // Api db
	)

	rollupUserLogic, err := rollupgen.NewRollupUserLogic(builder.addresses.Rollup, builder.L1.Client)
	Require(t, err)
	chalManagerAddr, err := rollupUserLogic.ChallengeManager(&bind.CallOpts{})
	Require(t, err)

	txOpts := builder.L1Info.GetDefaultTransactOpts(addressName, ctx)

	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(node.ConsensusNode.ArbDB, storage.StakerPrefix),
		node.ConsensusNode.L1Reader,
		&txOpts,
		NewCommonConfigFetcher(builder.nodeConfig),
		node.ConsensusNode.SyncMonitor,
		builder.L1Info.Signer.ChainID(),
	)
	Require(t, err)

	assertionChain, err := solimpl.NewAssertionChain(
		ctx,
		builder.addresses.Rollup,
		chalManagerAddr,
		&txOpts,
		util.NewBackendWrapper(builder.L1.Client, rpc.LatestBlockNumber),
		bold.NewDataPosterTransactor(dp),
		solimpl.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
	)
	Require(t, err)

	stackOpts := []challengemanager.StackOpt{
		challengemanager.StackWithName(addressName),
		challengemanager.StackWithMode(modes.MakeMode),
		challengemanager.StackWithPostingInterval(time.Second * 3),
		challengemanager.StackWithPollingInterval(time.Second),
		challengemanager.StackWithAverageBlockCreationTime(time.Second),
		challengemanager.StackWithMinimumGapToParentAssertion(0),
	}

	challengeManager, err := challengemanager.NewChallengeStack(
		assertionChain,
		provider,
		stackOpts...,
	)
	Require(t, err)

	challengeManager.Start(ctx)
	return assertionChain, challengeManager.StopAndWait
}
