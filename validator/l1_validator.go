//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/pkg/errors"
)

type Validator struct {
	rollup             *RollupWatcher
	rollupAddress      common.Address
	sequencerInbox     *bridgegen.SequencerInbox
	validatorUtils     *rollupgen.ValidatorUtils
	client             arbutil.L1Interface
	builder            *BuilderBackend
	wallet             *ValidatorWallet
	callOpts           bind.CallOpts
	GasThreshold       *big.Int
	SendThreshold      *big.Int
	BlockThreshold     *big.Int
	genesisBlockNumber uint64

	l2Blockchain   *core.BlockChain
	inboxReader    InboxReaderInterface
	inboxTracker   InboxTrackerInterface
	txStreamer     TransactionStreamerInterface
	blockValidator *BlockValidator
}

func NewValidator(
	ctx context.Context,
	client arbutil.L1Interface,
	wallet *ValidatorWallet,
	validatorUtilsAddress common.Address,
	callOpts bind.CallOpts,
	l2Blockchain *core.BlockChain,
	inboxReader InboxReaderInterface,
	inboxTracker InboxTrackerInterface,
	txStreamer TransactionStreamerInterface,
	blockValidator *BlockValidator,
) (*Validator, error) {
	builder, err := NewBuilderBackend(wallet)
	if err != nil {
		return nil, err
	}
	rollup, err := NewRollupWatcher(ctx, wallet.RollupAddress(), builder, callOpts)
	if err != nil {
		return nil, err
	}
	localCallOpts := callOpts
	localCallOpts.Context = ctx
	sequencerBridgeAddress, err := rollup.SequencerBridge(&localCallOpts)
	if err != nil {
		return nil, err
	}
	sequencerInbox, err := bridgegen.NewSequencerInbox(sequencerBridgeAddress, client)
	if err != nil {
		return nil, err
	}
	validatorUtils, err := rollupgen.NewValidatorUtils(
		validatorUtilsAddress,
		client,
	)
	if err != nil {
		return nil, err
	}
	genesisBlockNumber, err := txStreamer.GetGenesisBlockNumber()
	if err != nil {
		return nil, err
	}
	return &Validator{
		rollup:             rollup,
		rollupAddress:      wallet.RollupAddress(),
		sequencerInbox:     sequencerInbox,
		validatorUtils:     validatorUtils,
		client:             client,
		builder:            builder,
		wallet:             wallet,
		GasThreshold:       big.NewInt(100_000_000_000),
		SendThreshold:      big.NewInt(5),
		BlockThreshold:     big.NewInt(960),
		callOpts:           callOpts,
		genesisBlockNumber: genesisBlockNumber,
		l2Blockchain:       l2Blockchain,
		inboxReader:        inboxReader,
		inboxTracker:       inboxTracker,
		txStreamer:         txStreamer,
		blockValidator:     blockValidator,
	}, nil
}

func (v *Validator) getCallOpts(ctx context.Context) *bind.CallOpts {
	opts := v.callOpts
	opts.Context = ctx
	return &opts
}

// removeOldStakers removes the stakes of all currently staked validators except
// its own if dontRemoveSelf is true
func (v *Validator) removeOldStakers(ctx context.Context, dontRemoveSelf bool) (*types.Transaction, error) {
	stakersToEliminate, err := v.validatorUtils.RefundableStakers(v.getCallOpts(ctx), v.rollupAddress)
	if err != nil {
		return nil, err
	}
	walletAddr := v.wallet.Address()
	if dontRemoveSelf && walletAddr != nil {
		for i, staker := range stakersToEliminate {
			if staker == *walletAddr {
				stakersToEliminate[i] = stakersToEliminate[len(stakersToEliminate)-1]
				stakersToEliminate = stakersToEliminate[:len(stakersToEliminate)-1]
				break
			}
		}
	}

	if len(stakersToEliminate) == 0 {
		return nil, nil
	}
	log.Info("removing old stakers", "count", len(stakersToEliminate))
	return v.wallet.ReturnOldDeposits(ctx, stakersToEliminate)
}

func (v *Validator) resolveTimedOutChallenges(ctx context.Context) (*types.Transaction, error) {
	challengesToEliminate, _, err := v.validatorUtils.TimedOutChallenges(v.getCallOpts(ctx), v.rollupAddress, 0, 10)
	if err != nil {
		return nil, err
	}
	if len(challengesToEliminate) == 0 {
		return nil, nil
	}
	log.Info("timing out challenges", "count", len(challengesToEliminate))
	return v.wallet.TimeoutChallenges(ctx, challengesToEliminate)
}

func (v *Validator) resolveNextNode(ctx context.Context, info *StakerInfo) error {
	callOpts := v.getCallOpts(ctx)
	confirmType, err := v.validatorUtils.CheckDecidableNextNode(callOpts, v.rollupAddress)
	if err != nil {
		return err
	}
	unresolvedNodeIndex, err := v.rollup.FirstUnresolvedNode(callOpts)
	if err != nil {
		return err
	}
	switch ConfirmType(confirmType) {
	case CONFIRM_TYPE_INVALID:
		addr := v.wallet.Address()
		if info == nil || addr == nil || info.LatestStakedNode <= unresolvedNodeIndex {
			// We aren't an example of someone staked on a competitor
			return nil
		}
		log.Info("rejecing node", "node", unresolvedNodeIndex)
		_, err = v.rollup.RejectNextNode(v.builder.Auth(ctx), *addr)
		return err
	case CONFIRM_TYPE_VALID:
		nodeInfo, err := v.rollup.LookupNode(ctx, unresolvedNodeIndex)
		if err != nil {
			return err
		}
		afterGs := nodeInfo.AfterState().GlobalState
		_, err = v.rollup.ConfirmNextNode(v.builder.Auth(ctx), afterGs.BlockHash, afterGs.SendRoot)
		return err
	default:
		return nil
	}
}

func (v *Validator) isRequiredStakeElevated(ctx context.Context) (bool, error) {
	callOpts := v.getCallOpts(ctx)
	requiredStake, err := v.rollup.CurrentRequiredStake(callOpts)
	if err != nil {
		return false, err
	}
	baseStake, err := v.rollup.BaseStake(callOpts)
	if err != nil {
		return false, err
	}
	return requiredStake.Cmp(baseStake) > 0, nil
}

type createNodeAction struct {
	assertion         *Assertion
	prevInboxMaxCount *big.Int
	hash              [32]byte
}

type existingNodeAction struct {
	number uint64
	hash   [32]byte
}

type nodeAction interface{}

type OurStakerInfo struct {
	LatestStakedNode     uint64
	LatestStakedNodeHash [32]byte
	CanProgress          bool
	StakeExists          bool
	*StakerInfo
}

// Returns (block number, global state inbox position is invalid, error).
// If global state is invalid, block number is set to the last of the batch.
func (v *Validator) blockNumberFromGlobalState(gs GoGlobalState) (int64, bool, error) {
	var batchHeight arbutil.MessageIndex
	if gs.Batch > 0 {
		var err error
		batchHeight, err = v.inboxTracker.GetBatchMessageCount(gs.Batch - 1)
		if err != nil {
			return 0, false, err
		}
	}

	nextBatchHeight, err := v.inboxTracker.GetBatchMessageCount(gs.Batch)
	if err != nil {
		return 0, false, err
	}

	if gs.PosInBatch >= uint64(nextBatchHeight-batchHeight) {
		// This PosInBatch would enter the next batch. Return the last block before the next batch.
		// We can be sure that MessageCountToBlockNumber will return a non-negative number as nextBatchHeight must be nonzero.
		return arbutil.MessageCountToBlockNumber(nextBatchHeight, v.genesisBlockNumber), true, nil
	}

	return arbutil.MessageCountToBlockNumber(batchHeight+arbutil.MessageIndex(gs.PosInBatch), v.genesisBlockNumber), false, nil
}

func (v *Validator) generateNodeAction(ctx context.Context, stakerInfo *OurStakerInfo, strategy StakerStrategy) (nodeAction, bool, error) {
	startState, prevInboxMaxCount, startStateProposed, err := lookupNodeStartState(ctx, v.rollup, stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash)
	if err != nil {
		return nil, false, err
	}

	localBatchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return nil, false, err
	}
	if localBatchCount < startState.RequiredBatches() {
		log.Info("catching up to chain batches", "localBatches", localBatchCount, "target", startState.RequiredBatches())
		return nil, false, nil
	}

	startBlock := v.l2Blockchain.GetBlockByHash(startState.GlobalState.BlockHash)
	if startBlock == nil && (startState.GlobalState != GoGlobalState{}) {
		expectedBlockHeight, inboxPositionInvalid, err := v.blockNumberFromGlobalState(startState.GlobalState)
		if err != nil {
			return nil, false, err
		}
		if inboxPositionInvalid {
			log.Error("invalid start global state inbox position", startState.GlobalState.BlockHash, "batch", startState.GlobalState.Batch, "pos", startState.GlobalState.PosInBatch)
			return nil, false, errors.New("invalid start global state inbox position")
		}
		latestHeader := v.l2Blockchain.CurrentHeader()
		if latestHeader.Number.Int64() < expectedBlockHeight {
			log.Info("catching up to chain blocks", "localBlocks", latestHeader.Number, "target", expectedBlockHeight)
			return nil, false, nil
		} else {
			log.Error("unknown start block hash", "hash", startState.GlobalState.BlockHash, "batch", startState.GlobalState.Batch, "pos", startState.GlobalState.PosInBatch)
			return nil, false, errors.New("unknown start block hash")
		}
	}

	var blocksValidated uint64
	if v.blockValidator != nil {
		blocksValidated = v.blockValidator.BlocksValidated()
	} else {
		blocksValidated = v.l2Blockchain.CurrentHeader().Number.Uint64()

		if localBatchCount > 0 {
			messageCount, err := v.inboxTracker.GetBatchMessageCount(localBatchCount - 1)
			if err != nil {
				return nil, false, err
			}
			// Must be non-negative as a batch must contain at least one message
			lastBatchBlock := uint64(arbutil.MessageCountToBlockNumber(messageCount, v.genesisBlockNumber))
			if blocksValidated > lastBatchBlock {
				blocksValidated = lastBatchBlock
			}
		} else {
			blocksValidated = 0
		}
	}

	currentL1Block, err := v.client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, false, err
	}

	minAssertionPeriod, err := v.rollup.MinimumAssertionPeriod(v.getCallOpts(ctx))
	if err != nil {
		return nil, false, err
	}

	timeSinceProposed := new(big.Int).Sub(currentL1Block.Number(), new(big.Int).SetUint64(startStateProposed))
	if timeSinceProposed.Cmp(minAssertionPeriod) < 0 {
		// Too soon to assert
		return nil, false, nil
	}

	// Not necessarily successors
	successorNodes, err := v.rollup.LookupNodeChildren(ctx, stakerInfo.LatestStakedNode)
	if err != nil {
		return nil, false, err
	}

	var correctNode nodeAction
	wrongNodesExist := false
	if len(successorNodes) > 0 {
		log.Info("examining existing potential successors", "count", len(successorNodes))
	}
	for _, nd := range successorNodes {
		if correctNode != nil && wrongNodesExist {
			// We've found everything we could hope to find
			break
		}
		if correctNode == nil {
			afterGs := nd.AfterState().GlobalState
			requiredBatches := nd.AfterState().RequiredBatches()
			if localBatchCount < requiredBatches {
				return nil, false, fmt.Errorf("waiting for validator to catch up to assertion batches: %v/%v", localBatchCount, requiredBatches)
			}
			if requiredBatches > 0 {
				haveAcc, err := v.inboxTracker.GetBatchAcc(requiredBatches - 1)
				if err != nil {
					return nil, false, err
				}
				if haveAcc != nd.AfterInboxBatchAcc {
					return nil, false, fmt.Errorf("missed sequencer batches reorg: at seq num %v have acc %v but assertion has acc %v", requiredBatches-1, haveAcc, nd.AfterInboxBatchAcc)
				}
			}
			lastBlockNum, inboxPositionInvalid, err := v.blockNumberFromGlobalState(afterGs)
			if err != nil {
				return nil, false, err
			}
			if int64(blocksValidated) < lastBlockNum {
				return nil, false, fmt.Errorf("waiting for validator to catch up to assertion blocks: %v/%v", blocksValidated, lastBlockNum)
			}
			var expectedBlockHash common.Hash
			var expectedSendRoot common.Hash
			if lastBlockNum >= 0 {
				lastBlock := v.l2Blockchain.GetBlockByNumber(uint64(lastBlockNum))
				if lastBlock == nil {
					return nil, false, fmt.Errorf("block %v not in database despite being validated", lastBlockNum)
				}
				lastBlockExtra, err := arbos.DeserializeHeaderExtraInformation(lastBlock.Header())
				if err != nil {
					return nil, false, err
				}
				expectedBlockHash = lastBlock.Hash()
				expectedSendRoot = lastBlockExtra.SendRoot
			}

			var expectedNumBlocks uint64
			if startBlock == nil {
				expectedNumBlocks = uint64(lastBlockNum + 1)
			} else {
				expectedNumBlocks = uint64(lastBlockNum) - startBlock.NumberU64()
			}
			valid := !inboxPositionInvalid &&
				nd.Assertion.NumBlocks == expectedNumBlocks &&
				afterGs.BlockHash == expectedBlockHash &&
				afterGs.SendRoot == expectedSendRoot
			if valid {
				log.Info(
					"found correct node",
					"node", nd.NodeNum,
					"blockNum", lastBlockNum,
					"blockHash", afterGs.BlockHash,
				)
				correctNode = existingNodeAction{
					number: nd.NodeNum,
					hash:   nd.NodeHash,
				}
				continue
			} else {
				log.Warn(
					"found node with incorrect assertion",
					"node", nd.NodeNum,
					"inboxPositionInvalid", inboxPositionInvalid,
					"computedBlockNum", lastBlockNum,
					"numBlocks", nd.Assertion.NumBlocks,
					"expectedNumBlocks", expectedNumBlocks,
					"blockHash", afterGs.BlockHash,
					"expectedBlockHash", expectedBlockHash,
					"sendRoot", afterGs.SendRoot,
					"expectedSendRoot", expectedSendRoot,
				)
			}
		} else {
			log.Warn("found younger sibling to correct node", "node", nd.NodeNum)
		}
		// If we've hit this point, the node is "wrong"
		wrongNodesExist = true
	}

	if strategy == WatchtowerStrategy || correctNode != nil || (strategy < MakeNodesStrategy && !wrongNodesExist) {
		return correctNode, wrongNodesExist, nil
	}

	if !prevInboxMaxCount.IsUint64() {
		return nil, false, fmt.Errorf("inbox max count %v isn't a uint64", prevInboxMaxCount)
	}
	minBatchCount := prevInboxMaxCount.Uint64()
	if localBatchCount < minBatchCount {
		// not enough batches in database
		return nil, wrongNodesExist, nil
	}

	if blocksValidated == 0 || localBatchCount == 0 {
		// we haven't validated anything
		return nil, wrongNodesExist, nil
	}
	lastBlockValidated := blocksValidated - 1
	if startBlock != nil && lastBlockValidated <= startBlock.NumberU64() {
		// we haven't validated any new blocks
		return nil, wrongNodesExist, nil
	}
	var assertionCoversBatch uint64
	var afterGsBatch uint64
	var afterGsPosInBatch uint64
	for i := localBatchCount - 1; i+1 >= minBatchCount && i > 0; i-- {
		batchMessageCount, err := v.inboxTracker.GetBatchMessageCount(i)
		if err != nil {
			return nil, false, err
		}
		prevBatchMessageCount, err := v.inboxTracker.GetBatchMessageCount(i - 1)
		if err != nil {
			return nil, false, err
		}
		// Must be non-negative as a batch must contain at least one message
		lastBlockNum := uint64(arbutil.MessageCountToBlockNumber(batchMessageCount, v.genesisBlockNumber))
		prevBlockNum := uint64(arbutil.MessageCountToBlockNumber(prevBatchMessageCount, v.genesisBlockNumber))
		if lastBlockValidated > prevBlockNum && lastBlockValidated <= lastBlockNum {
			// We found the batch containing the last validated block
			if i+1 == minBatchCount && lastBlockValidated < lastBlockNum {
				// We haven't reached the minimum assertion size yet
				break
			}
			assertionCoversBatch = i
			if lastBlockValidated < lastBlockNum {
				afterGsBatch = i
				afterGsPosInBatch = lastBlockValidated - prevBlockNum
			} else {
				afterGsBatch = i + 1
				afterGsPosInBatch = 0
			}
			break
		}
	}
	if assertionCoversBatch == 0 {
		// we haven't validated the next batch completely
		return nil, wrongNodesExist, nil
	}
	validatedBatchAcc, err := v.inboxTracker.GetBatchAcc(assertionCoversBatch)
	if err != nil {
		return nil, false, err
	}

	assertingBlock := v.l2Blockchain.GetBlockByNumber(lastBlockValidated)
	if assertingBlock == nil {
		return nil, false, fmt.Errorf("missing validated block %v", lastBlockValidated)
	}
	assertingBlockExtra, err := arbos.DeserializeHeaderExtraInformation(assertingBlock.Header())
	if err != nil {
		return nil, false, err
	}

	hasSiblingByte := [1]byte{0}
	lastNum := stakerInfo.LatestStakedNode
	lastHash := stakerInfo.LatestStakedNodeHash
	if len(successorNodes) > 0 {
		lastSuccessor := successorNodes[len(successorNodes)-1]
		lastNum = lastSuccessor.NodeNum
		lastHash = lastSuccessor.NodeHash
		hasSiblingByte[0] = 1
	}
	var assertionNumBlocks uint64
	if startBlock == nil {
		assertionNumBlocks = assertingBlock.NumberU64() + 1
	} else {
		assertionNumBlocks = assertingBlock.NumberU64() - startBlock.NumberU64()
	}
	assertion := &Assertion{
		BeforeState: startState,
		AfterState: &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  assertingBlock.Hash(),
				SendRoot:   assertingBlockExtra.SendRoot,
				Batch:      afterGsBatch,
				PosInBatch: afterGsPosInBatch,
			},
			MachineStatus: MachineStatusFinished,
		},
		NumBlocks: assertionNumBlocks,
	}

	executionHash := assertion.ExecutionHash()
	newNodeHash := crypto.Keccak256Hash(hasSiblingByte[:], lastHash[:], executionHash[:], validatedBatchAcc[:])

	action := createNodeAction{
		assertion:         assertion,
		hash:              newNodeHash,
		prevInboxMaxCount: prevInboxMaxCount,
	}
	log.Info("creating node", "hash", newNodeHash, "lastNode", lastNum, "parentNode", stakerInfo.LatestStakedNode)
	return action, wrongNodesExist, nil
}

// Returns (execution state, inbox max count, block proposed, error)
func lookupNodeStartState(ctx context.Context, rollup *RollupWatcher, nodeNum uint64, nodeHash [32]byte) (*ExecutionState, *big.Int, uint64, error) {
	if nodeNum == 0 {
		creationEvent, err := rollup.LookupCreation(ctx)
		if err != nil {
			return nil, nil, 0, err
		}
		return &ExecutionState{
			GlobalState:   GoGlobalState{},
			MachineStatus: MachineStatusFinished,
		}, big.NewInt(1), creationEvent.Raw.BlockNumber, nil
	}
	node, err := rollup.LookupNode(ctx, nodeNum)
	if err != nil {
		return nil, nil, 0, err
	}
	if node.NodeHash != nodeHash {
		return nil, nil, 0, errors.New("looked up starting node but found wrong hash")
	}
	return node.AfterState(), node.InboxMaxCount, node.BlockProposed, nil
}
