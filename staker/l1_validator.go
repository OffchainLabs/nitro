// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/validator"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

type ConfirmType uint8

const (
	CONFIRM_TYPE_NONE ConfirmType = iota
	CONFIRM_TYPE_VALID
	CONFIRM_TYPE_INVALID
)

type ConflictType uint8

const (
	CONFLICT_TYPE_NONE ConflictType = iota
	CONFLICT_TYPE_FOUND
	CONFLICT_TYPE_INDETERMINATE
	CONFLICT_TYPE_INCOMPLETE
)

type L1Validator struct {
	rollup             *RollupWatcher
	rollupAddress      common.Address
	validatorUtils     *rollupgen.ValidatorUtils
	client             arbutil.L1Interface
	builder            *ValidatorTxBuilder
	wallet             ValidatorWalletInterface
	callOpts           bind.CallOpts
	genesisBlockNumber uint64

	l2Blockchain       *core.BlockChain
	das                arbstate.DataAvailabilityReader
	inboxTracker       InboxTrackerInterface
	txStreamer         TransactionStreamerInterface
	blockValidator     *BlockValidator
	lastWasmModuleRoot common.Hash
}

func NewL1Validator(
	client arbutil.L1Interface,
	wallet ValidatorWalletInterface,
	validatorUtilsAddress common.Address,
	callOpts bind.CallOpts,
	l2Blockchain *core.BlockChain,
	das arbstate.DataAvailabilityReader,
	inboxTracker InboxTrackerInterface,
	txStreamer TransactionStreamerInterface,
	blockValidator *BlockValidator,
) (*L1Validator, error) {
	builder, err := NewValidatorTxBuilder(wallet)
	if err != nil {
		return nil, err
	}
	rollup, err := NewRollupWatcher(wallet.RollupAddress(), builder, callOpts)
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
	return &L1Validator{
		rollup:             rollup,
		rollupAddress:      wallet.RollupAddress(),
		validatorUtils:     validatorUtils,
		client:             client,
		builder:            builder,
		wallet:             wallet,
		callOpts:           callOpts,
		genesisBlockNumber: genesisBlockNumber,
		l2Blockchain:       l2Blockchain,
		das:                das,
		inboxTracker:       inboxTracker,
		txStreamer:         txStreamer,
		blockValidator:     blockValidator,
	}, nil
}

func (v *L1Validator) getCallOpts(ctx context.Context) *bind.CallOpts {
	opts := v.callOpts
	opts.Context = ctx
	return &opts
}

func (v *L1Validator) Initialize(ctx context.Context) error {
	err := v.rollup.Initialize(ctx)
	if err != nil {
		return err
	}
	return v.updateBlockValidatorModuleRoot(ctx)
}

func (v *L1Validator) updateBlockValidatorModuleRoot(ctx context.Context) error {
	if v.blockValidator == nil {
		return nil
	}
	moduleRoot, err := v.rollup.WasmModuleRoot(v.getCallOpts(ctx))
	if err != nil {
		return err
	}
	if moduleRoot != v.lastWasmModuleRoot {
		err := v.blockValidator.SetCurrentWasmModuleRoot(moduleRoot)
		if err != nil {
			return err
		}
		v.lastWasmModuleRoot = moduleRoot
	} else if (moduleRoot == common.Hash{}) {
		return errors.New("wasmModuleRoot in rollup is zero")
	}
	return nil
}

func (v *L1Validator) resolveTimedOutChallenges(ctx context.Context) (*types.Transaction, error) {
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

func (v *L1Validator) resolveNextNode(ctx context.Context, info *StakerInfo, latestConfirmedNode *uint64) (bool, error) {
	callOpts := v.getCallOpts(ctx)
	confirmType, err := v.validatorUtils.CheckDecidableNextNode(callOpts, v.rollupAddress)
	if err != nil {
		return false, err
	}
	unresolvedNodeIndex, err := v.rollup.FirstUnresolvedNode(callOpts)
	if err != nil {
		return false, err
	}
	switch ConfirmType(confirmType) {
	case CONFIRM_TYPE_INVALID:
		addr := v.wallet.Address()
		if info == nil || addr == nil || info.LatestStakedNode <= unresolvedNodeIndex {
			// We aren't an example of someone staked on a competitor
			return false, nil
		}
		log.Warn("rejecting node", "node", unresolvedNodeIndex)
		_, err = v.rollup.RejectNextNode(v.builder.Auth(ctx), *addr)
		return true, err
	case CONFIRM_TYPE_VALID:
		nodeInfo, err := v.rollup.LookupNode(ctx, unresolvedNodeIndex)
		if err != nil {
			return false, err
		}
		afterGs := nodeInfo.AfterState().GlobalState
		log.Info("confirming node", "node", unresolvedNodeIndex)
		_, err = v.rollup.ConfirmNextNode(v.builder.Auth(ctx), afterGs.BlockHash, afterGs.SendRoot)
		if err != nil {
			return false, err
		}
		*latestConfirmedNode = unresolvedNodeIndex
		return true, nil
	default:
		return false, nil
	}
}

func (v *L1Validator) isRequiredStakeElevated(ctx context.Context) (bool, error) {
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
	hash              common.Hash
}

type existingNodeAction struct {
	number uint64
	hash   [32]byte
}

type nodeAction interface{}

type OurStakerInfo struct {
	LatestStakedNode     uint64
	LatestStakedNodeHash common.Hash
	CanProgress          bool
	StakeExists          bool
	*StakerInfo
}

// Returns (block number, global state inbox position is invalid, error).
// If global state is invalid, block number is set to the last of the batch.
func (v *L1Validator) blockNumberFromGlobalState(gs validator.GoGlobalState) (int64, bool, error) {
	var batchHeight arbutil.MessageIndex
	if gs.Batch > 0 {
		var err error
		batchHeight, err = v.inboxTracker.GetBatchMessageCount(gs.Batch - 1)
		if err != nil {
			return 0, false, err
		}
	}

	// Validate the PosInBatch if it's non-zero
	if gs.PosInBatch > 0 {
		nextBatchHeight, err := v.inboxTracker.GetBatchMessageCount(gs.Batch)
		if err != nil {
			return 0, false, err
		}

		if gs.PosInBatch >= uint64(nextBatchHeight-batchHeight) {
			// This PosInBatch would enter the next batch. Return the last block before the next batch.
			// We can be sure that MessageCountToBlockNumber will return a non-negative number as nextBatchHeight must be nonzero.
			return arbutil.MessageCountToBlockNumber(nextBatchHeight, v.genesisBlockNumber), true, nil
		}
	}

	return arbutil.MessageCountToBlockNumber(batchHeight+arbutil.MessageIndex(gs.PosInBatch), v.genesisBlockNumber), false, nil
}

func (v *L1Validator) generateNodeAction(ctx context.Context, stakerInfo *OurStakerInfo, strategy StakerStrategy, makeAssertionInterval time.Duration) (nodeAction, bool, error) {
	startState, prevInboxMaxCount, startStateProposed, err := lookupNodeStartState(ctx, v.rollup, stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash)
	if err != nil {
		return nil, false, fmt.Errorf("error looking up node %v (hash %v) start state: %w", stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash, err)
	}

	startStateProposedHeader, err := v.client.HeaderByNumber(ctx, new(big.Int).SetUint64(startStateProposed))
	if err != nil {
		return nil, false, fmt.Errorf("error looking up L1 header of block %v of node start state: %w", startStateProposed, err)
	}
	startStateProposedTime := time.Unix(int64(startStateProposedHeader.Time), 0)

	v.txStreamer.PauseReorgs()
	defer v.txStreamer.ResumeReorgs()

	localBatchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return nil, false, fmt.Errorf("error getting batch count from inbox tracker: %w", err)
	}
	if localBatchCount < startState.RequiredBatches() {
		log.Info("catching up to chain batches", "localBatches", localBatchCount, "target", startState.RequiredBatches())
		return nil, false, nil
	}

	startBlock := v.l2Blockchain.GetBlockByHash(startState.GlobalState.BlockHash)
	if startBlock == nil && (startState.GlobalState != validator.GoGlobalState{}) {
		expectedBlockHeight, inboxPositionInvalid, err := v.blockNumberFromGlobalState(startState.GlobalState)
		if err != nil {
			return nil, false, fmt.Errorf("error getting block number from global state: %w", err)
		}
		if inboxPositionInvalid {
			log.Error("invalid start global state inbox position", startState.GlobalState.BlockHash, "batch", startState.GlobalState.Batch, "pos", startState.GlobalState.PosInBatch)
			return nil, false, errors.New("invalid start global state inbox position")
		}
		latestHeader := v.l2Blockchain.CurrentBlock()
		if latestHeader.Number.Int64() < expectedBlockHeight {
			log.Info("catching up to chain blocks", "localBlocks", latestHeader.Number, "target", expectedBlockHeight)
			return nil, false, nil
		}
		log.Error("unknown start block hash", "hash", startState.GlobalState.BlockHash, "batch", startState.GlobalState.Batch, "pos", startState.GlobalState.PosInBatch)
		return nil, false, errors.New("unknown start block hash")
	}

	var lastBlockValidated uint64
	if v.blockValidator != nil {
		var expectedHash common.Hash
		var validRoots []common.Hash
		lastBlockValidated, expectedHash, validRoots = v.blockValidator.LastBlockValidatedAndHash()
		haveHash := v.l2Blockchain.GetCanonicalHash(lastBlockValidated)
		if haveHash != expectedHash {
			return nil, false, fmt.Errorf("block validator validated block %v as hash %v but blockchain has hash %v", lastBlockValidated, expectedHash, haveHash)
		}
		if err := v.updateBlockValidatorModuleRoot(ctx); err != nil {
			return nil, false, fmt.Errorf("error updating block validator module root: %w", err)
		}
		wasmRootValid := false
		for _, root := range validRoots {
			if v.lastWasmModuleRoot == root {
				wasmRootValid = true
				break
			}
		}
		if !wasmRootValid {
			return nil, false, fmt.Errorf("wasmroot doesn't match rollup : %v, valid: %v", v.lastWasmModuleRoot, validRoots)
		}
	} else {
		lastBlockValidated = v.l2Blockchain.CurrentBlock().Number.Uint64()

		if localBatchCount > 0 {
			messageCount, err := v.inboxTracker.GetBatchMessageCount(localBatchCount - 1)
			if err != nil {
				return nil, false, fmt.Errorf("error getting latest batch %v message count: %w", localBatchCount-1, err)
			}
			// Must be non-negative as a batch must contain at least one message
			lastBatchBlock := uint64(arbutil.MessageCountToBlockNumber(messageCount, v.genesisBlockNumber))
			if lastBlockValidated > lastBatchBlock {
				lastBlockValidated = lastBatchBlock
			}
		} else {
			lastBlockValidated = 0
		}
	}

	currentL1BlockNum, err := v.client.BlockNumber(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("error getting latest L1 block number: %w", err)
	}

	parentChainBlockNumber, err := arbutil.CorrespondingL1BlockNumber(ctx, v.client, currentL1BlockNum)
	if err != nil {
		return nil, false, err
	}

	minAssertionPeriod, err := v.rollup.MinimumAssertionPeriod(v.getCallOpts(ctx))
	if err != nil {
		return nil, false, fmt.Errorf("error getting rollup minimum assertion period: %w", err)
	}

	timeSinceProposed := big.NewInt(int64(parentChainBlockNumber) - int64(startStateProposed))
	if timeSinceProposed.Cmp(minAssertionPeriod) < 0 {
		// Too soon to assert
		return nil, false, nil
	}

	successorNodes, err := v.rollup.LookupNodeChildren(ctx, stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash)
	if err != nil {
		return nil, false, fmt.Errorf("error looking up node %v (hash %v) children: %w", stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash, err)
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
					return nil, false, fmt.Errorf("error getting batch %v accumulator: %w", requiredBatches-1, err)
				}
				if haveAcc != nd.AfterInboxBatchAcc {
					return nil, false, fmt.Errorf("missed sequencer batches reorg: at seq num %v have acc %v but assertion has acc %v", requiredBatches-1, haveAcc, nd.AfterInboxBatchAcc)
				}
			}
			lastBlockNum, inboxPositionInvalid, err := v.blockNumberFromGlobalState(afterGs)
			if err != nil {
				return nil, false, fmt.Errorf("error getting block number from global state: %w", err)
			}
			if int64(lastBlockValidated) < lastBlockNum {
				return nil, false, fmt.Errorf("waiting for validator to catch up to assertion blocks: %v/%v", lastBlockValidated, lastBlockNum)
			}
			var expectedBlockHash common.Hash
			var expectedSendRoot common.Hash
			if lastBlockNum >= 0 {
				lastBlock := v.l2Blockchain.GetBlockByNumber(uint64(lastBlockNum))
				if lastBlock == nil {
					return nil, false, fmt.Errorf("block %v not in database despite being validated", lastBlockNum)
				}
				lastBlockExtra := types.DeserializeHeaderExtraInformation(lastBlock.Header())
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
				afterGs.SendRoot == expectedSendRoot &&
				nd.Assertion.AfterState.MachineStatus == validator.MachineStatusFinished
			if valid {
				log.Info(
					"found correct assertion",
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
				log.Error(
					"found incorrect assertion",
					"node", nd.NodeNum,
					"inboxPositionInvalid", inboxPositionInvalid,
					"computedBlockNum", lastBlockNum,
					"numBlocks", nd.Assertion.NumBlocks,
					"expectedNumBlocks", expectedNumBlocks,
					"blockHash", afterGs.BlockHash,
					"expectedBlockHash", expectedBlockHash,
					"sendRoot", afterGs.SendRoot,
					"expectedSendRoot", expectedSendRoot,
					"machineStatus", nd.Assertion.AfterState.MachineStatus,
				)
			}
		} else {
			log.Error("found younger sibling to correct assertion (implicitly invalid)", "node", nd.NodeNum)
		}
		// If we've hit this point, the node is "wrong"
		wrongNodesExist = true
	}

	if correctNode != nil || strategy == WatchtowerStrategy {
		return correctNode, wrongNodesExist, nil
	}

	if wrongNodesExist || (strategy >= MakeNodesStrategy && time.Since(startStateProposedTime) >= makeAssertionInterval) {
		// There's no correct node; create one.
		var lastNodeHashIfExists *common.Hash
		if len(successorNodes) > 0 {
			lastNodeHashIfExists = &successorNodes[len(successorNodes)-1].NodeHash
		}
		action, err := v.createNewNodeAction(ctx, stakerInfo, lastBlockValidated, localBatchCount, prevInboxMaxCount, startBlock, startState, lastNodeHashIfExists)
		if err != nil {
			return nil, wrongNodesExist, fmt.Errorf("error generating create new node action (from start block %v to last block validated %v): %w", startBlock, lastBlockValidated, err)
		}
		return action, wrongNodesExist, nil
	}

	return nil, wrongNodesExist, nil
}

func (v *L1Validator) createNewNodeAction(
	ctx context.Context,
	stakerInfo *OurStakerInfo,
	lastBlockValidated uint64,
	localBatchCount uint64,
	prevInboxMaxCount *big.Int,
	startBlock *types.Block,
	startState *validator.ExecutionState,
	lastNodeHashIfExists *common.Hash,
) (nodeAction, error) {
	if !prevInboxMaxCount.IsUint64() {
		return nil, fmt.Errorf("inbox max count %v isn't a uint64", prevInboxMaxCount)
	}
	minBatchCount := prevInboxMaxCount.Uint64()
	if localBatchCount < minBatchCount {
		// not enough batches in database
		return nil, nil
	}

	if localBatchCount == 0 {
		// we haven't validated anything
		return nil, nil
	}
	if startBlock != nil && lastBlockValidated <= startBlock.NumberU64() {
		// we haven't validated any new blocks
		return nil, nil
	}
	var assertionCoversBatch uint64
	var afterGsBatch uint64
	var afterGsPosInBatch uint64
	for i := localBatchCount - 1; i+1 >= minBatchCount && i > 0; i-- {
		batchMessageCount, err := v.inboxTracker.GetBatchMessageCount(i)
		if err != nil {
			return nil, fmt.Errorf("error getting batch %v message count: %w", i, err)
		}
		prevBatchMessageCount, err := v.inboxTracker.GetBatchMessageCount(i - 1)
		if err != nil {
			return nil, fmt.Errorf("error getting previous batch %v message count: %w", i-1, err)
		}
		// Must be non-negative as a batch must contain at least one message
		lastBlockNum := uint64(arbutil.MessageCountToBlockNumber(batchMessageCount, v.genesisBlockNumber))
		prevBlockNum := uint64(arbutil.MessageCountToBlockNumber(prevBatchMessageCount, v.genesisBlockNumber))
		if lastBlockValidated > lastBlockNum {
			return nil, fmt.Errorf("%v blocks have been validated but only %v appear in the latest batch", lastBlockValidated, lastBlockNum)
		}
		if lastBlockValidated > prevBlockNum {
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
		return nil, nil
	}
	validatedBatchAcc, err := v.inboxTracker.GetBatchAcc(assertionCoversBatch)
	if err != nil {
		return nil, fmt.Errorf("error getting batch %v accumulator: %w", assertionCoversBatch, err)
	}

	assertingBlock := v.l2Blockchain.GetBlockByNumber(lastBlockValidated)
	if assertingBlock == nil {
		return nil, fmt.Errorf("missing validated block %v", lastBlockValidated)
	}
	assertingBlockExtra := types.DeserializeHeaderExtraInformation(assertingBlock.Header())

	hasSiblingByte := [1]byte{0}
	prevNum := stakerInfo.LatestStakedNode
	lastHash := stakerInfo.LatestStakedNodeHash
	if lastNodeHashIfExists != nil {
		lastHash = *lastNodeHashIfExists
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
		AfterState: &validator.ExecutionState{
			GlobalState: validator.GoGlobalState{
				BlockHash:  assertingBlock.Hash(),
				SendRoot:   assertingBlockExtra.SendRoot,
				Batch:      afterGsBatch,
				PosInBatch: afterGsPosInBatch,
			},
			MachineStatus: validator.MachineStatusFinished,
		},
		NumBlocks: assertionNumBlocks,
	}

	wasmModuleRoot := v.lastWasmModuleRoot
	if v.blockValidator == nil {
		wasmModuleRoot, err = v.rollup.WasmModuleRoot(v.getCallOpts(ctx))
		if err != nil {
			return nil, fmt.Errorf("error rollup wasm module root: %w", err)
		}
	}

	executionHash := assertion.ExecutionHash()
	newNodeHash := crypto.Keccak256Hash(hasSiblingByte[:], lastHash[:], executionHash[:], validatedBatchAcc[:], wasmModuleRoot[:])

	action := createNodeAction{
		assertion:         assertion,
		hash:              newNodeHash,
		prevInboxMaxCount: prevInboxMaxCount,
	}
	log.Info("creating node", "hash", newNodeHash, "lastNode", prevNum, "parentNode", stakerInfo.LatestStakedNode)
	return action, nil
}

// Returns (execution state, inbox max count, L1 block proposed, error)
func lookupNodeStartState(ctx context.Context, rollup *RollupWatcher, nodeNum uint64, nodeHash [32]byte) (*validator.ExecutionState, *big.Int, uint64, error) {
	if nodeNum == 0 {
		creationEvent, err := rollup.LookupCreation(ctx)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("error looking up rollup creation event: %w", err)
		}
		parentChainBlockNumber, err := arbutil.CorrespondingL1BlockNumber(ctx, rollup.client, creationEvent.Raw.BlockNumber)
		if err != nil {
			return nil, nil, 0, err
		}
		return &validator.ExecutionState{
			GlobalState:   validator.GoGlobalState{},
			MachineStatus: validator.MachineStatusFinished,
		}, big.NewInt(1), parentChainBlockNumber, nil
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
