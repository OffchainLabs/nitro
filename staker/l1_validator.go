// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/offchainlabs/nitro/staker/txbuilder"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/validator"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
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
	rollup         *RollupWatcher
	rollupAddress  common.Address
	validatorUtils *rollupgen.ValidatorUtils
	client         arbutil.L1Interface
	builder        *txbuilder.Builder
	wallet         ValidatorWalletInterface
	callOpts       bind.CallOpts

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
	inboxTracker InboxTrackerInterface,
	txStreamer TransactionStreamerInterface,
	blockValidator *BlockValidator,
) (*L1Validator, error) {
	builder, err := txbuilder.NewBuilder(wallet)
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
	return &L1Validator{
		rollup:         rollup,
		rollupAddress:  wallet.RollupAddress(),
		validatorUtils: validatorUtils,
		client:         client,
		builder:        builder,
		wallet:         wallet,
		callOpts:       callOpts,
		inboxTracker:   inboxTracker,
		txStreamer:     txStreamer,
		blockValidator: blockValidator,
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
		auth, err := v.builder.Auth(ctx)
		if err != nil {
			return false, err
		}
		_, err = v.rollup.RejectNextNode(auth, *addr)
		return true, err
	case CONFIRM_TYPE_VALID:
		nodeInfo, err := v.rollup.LookupNode(ctx, unresolvedNodeIndex)
		if err != nil {
			return false, err
		}
		afterGs := nodeInfo.AfterState().GlobalState
		log.Info("confirming node", "node", unresolvedNodeIndex)
		auth, err := v.builder.Auth(ctx)
		if err != nil {
			return false, err
		}
		_, err = v.rollup.ConfirmNextNode(auth, afterGs.BlockHash, afterGs.SendRoot)
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

func (v *L1Validator) generateNodeAction(
	ctx context.Context,
	stakerInfo *OurStakerInfo,
	strategy StakerStrategy,
	stakerConfig *L1ValidatorConfig,
) (nodeAction, bool, error) {
	startState, prevInboxMaxCount, startStateProposedL1, startStateProposedParentChain, err := lookupNodeStartState(
		ctx, v.rollup, stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash,
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"error looking up node %v (hash %v) start state: %w",
			stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash, err,
		)
	}

	startStateProposedHeader, err := v.client.HeaderByNumber(ctx, arbmath.UintToBig(startStateProposedParentChain))
	if err != nil {
		return nil, false, fmt.Errorf(
			"error looking up L1 header of block %v of node start state: %w",
			startStateProposedParentChain, err,
		)
	}
	startStateProposedTime := time.Unix(int64(startStateProposedHeader.Time), 0)

	v.txStreamer.PauseReorgs()
	defer v.txStreamer.ResumeReorgs()

	localBatchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return nil, false, fmt.Errorf("error getting batch count from inbox tracker: %w", err)
	}
	if localBatchCount < startState.RequiredBatches() || localBatchCount == 0 {
		log.Info(
			"catching up to chain batches", "localBatches", localBatchCount,
			"target", startState.RequiredBatches(),
		)
		return nil, false, nil
	}

	caughtUp, startCount, err := GlobalStateToMsgCount(v.inboxTracker, v.txStreamer, startState.GlobalState)
	if err != nil {
		return nil, false, fmt.Errorf("start state not in chain: %w", err)
	}
	if !caughtUp {
		target := GlobalStatePosition{
			BatchNumber: startState.GlobalState.Batch,
			PosInBatch:  startState.GlobalState.PosInBatch,
		}
		var current GlobalStatePosition
		head, err := v.txStreamer.GetProcessedMessageCount()
		if err != nil {
			_, current, err = v.blockValidator.GlobalStatePositionsAtCount(head)
		}
		if err != nil {
			log.Info("catching up to chain messages", "target", target)
		} else {
			log.Info("catching up to chain blocks", "target", target, "current", current)
		}
		return nil, false, nil
	}

	var validatedCount arbutil.MessageIndex
	var validatedGlobalState validator.GoGlobalState
	if v.blockValidator != nil {
		valInfo, err := v.blockValidator.ReadLastValidatedInfo()
		if err != nil || valInfo == nil {
			return nil, false, err
		}
		validatedGlobalState = valInfo.GlobalState
		caughtUp, validatedCount, err = GlobalStateToMsgCount(
			v.inboxTracker, v.txStreamer, valInfo.GlobalState,
		)
		if err != nil {
			return nil, false, fmt.Errorf("%w: not found validated block in blockchain", err)
		}
		if !caughtUp {
			log.Info("catching up to last validated block", "target", valInfo.GlobalState)
			return nil, false, nil
		}
		if err := v.updateBlockValidatorModuleRoot(ctx); err != nil {
			return nil, false, fmt.Errorf("error updating block validator module root: %w", err)
		}
		wasmRootValid := false
		for _, root := range valInfo.WasmRoots {
			if v.lastWasmModuleRoot == root {
				wasmRootValid = true
				break
			}
		}
		if !wasmRootValid {
			if !stakerConfig.Dangerous.IgnoreRollupWasmModuleRoot {
				if len(valInfo.WasmRoots) == 0 {
					return nil, false, fmt.Errorf("block validation is still pending")
				}
				return nil, false, fmt.Errorf(
					"wasmroot doesn't match rollup : %v, valid: %v",
					v.lastWasmModuleRoot, valInfo.WasmRoots,
				)
			}
			log.Warn("wasmroot doesn't match rollup", "rollup", v.lastWasmModuleRoot, "blockValidator", valInfo.WasmRoots)
		}
	} else {
		validatedCount, err = v.txStreamer.GetProcessedMessageCount()
		if err != nil || validatedCount == 0 {
			return nil, false, err
		}
		var batchNum uint64
		messageCount, err := v.inboxTracker.GetBatchMessageCount(localBatchCount - 1)
		if err != nil {
			return nil, false, fmt.Errorf("error getting latest batch %v message count: %w", localBatchCount-1, err)
		}
		if validatedCount >= messageCount {
			batchNum = localBatchCount - 1
			validatedCount = messageCount
		} else {
			var found bool
			batchNum, found, err = v.inboxTracker.FindInboxBatchContainingMessage(validatedCount - 1)
			if err != nil {
				return nil, false, err
			}
			if !found {
				return nil, false, errors.New("batch not found on L1")
			}
		}
		execResult, err := v.txStreamer.ResultAtCount(validatedCount)
		if err != nil {
			return nil, false, err
		}
		_, gsPos, err := GlobalStatePositionsAtCount(v.inboxTracker, validatedCount, batchNum)
		if err != nil {
			return nil, false, fmt.Errorf("%w: failed calculating GSposition for count %d", err, validatedCount)
		}
		validatedGlobalState = buildGlobalState(*execResult, gsPos)
	}

	currentL1BlockNum, err := v.client.BlockNumber(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("error getting latest L1 block number: %w", err)
	}

	l1BlockNumber, err := arbutil.CorrespondingL1BlockNumber(ctx, v.client, currentL1BlockNum)
	if err != nil {
		return nil, false, err
	}

	minAssertionPeriod, err := v.rollup.MinimumAssertionPeriod(v.getCallOpts(ctx))
	if err != nil {
		return nil, false, fmt.Errorf("error getting rollup minimum assertion period: %w", err)
	}

	timeSinceProposed := big.NewInt(int64(l1BlockNumber) - int64(startStateProposedL1))
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
		if correctNode != nil {
			log.Error("found younger sibling to correct assertion (implicitly invalid)", "node", nd.NodeNum)
			wrongNodesExist = true
			continue
		}
		afterGS := nd.AfterState().GlobalState
		requiredBatch := afterGS.Batch
		if afterGS.PosInBatch == 0 && afterGS.Batch > 0 {
			requiredBatch -= 1
		}
		if localBatchCount <= requiredBatch {
			log.Info("staker: waiting for node to catch up to assertion batch", "current", localBatchCount, "target", requiredBatch-1)
			return nil, false, nil
		}
		nodeBatchMsgCount, err := v.inboxTracker.GetBatchMessageCount(requiredBatch)
		if err != nil {
			return nil, false, err
		}
		if validatedCount < nodeBatchMsgCount {
			log.Info("staker: waiting for validator to catch up to assertion batch messages", "current", validatedCount, "target", nodeBatchMsgCount)
			return nil, false, nil
		}
		if nd.Assertion.AfterState.MachineStatus != validator.MachineStatusFinished {
			wrongNodesExist = true
			log.Error("Found incorrect assertion: Machine status not finished", "node", nd.NodeNum, "machineStatus", nd.Assertion.AfterState.MachineStatus)
			continue
		}
		caughtUp, nodeMsgCount, err := GlobalStateToMsgCount(v.inboxTracker, v.txStreamer, afterGS)
		if errors.Is(err, ErrGlobalStateNotInChain) {
			wrongNodesExist = true
			log.Error("Found incorrect assertion", "node", nd.NodeNum, "afterGS", afterGS, "err", err)
			continue
		}
		if err != nil {
			return nil, false, fmt.Errorf("error getting message number from global state: %w", err)
		}
		if !caughtUp {
			return nil, false, fmt.Errorf("unexpected no-caught-up parsing assertion. Current: %d target: %v", validatedCount, afterGS)
		}
		log.Info(
			"found correct assertion",
			"node", nd.NodeNum,
			"count", nodeMsgCount,
			"blockHash", afterGS.BlockHash,
		)
		correctNode = existingNodeAction{
			number: nd.NodeNum,
			hash:   nd.NodeHash,
		}
	}

	if correctNode != nil || strategy == WatchtowerStrategy {
		return correctNode, wrongNodesExist, nil
	}

	makeAssertionInterval := stakerConfig.MakeAssertionInterval
	if wrongNodesExist || (strategy >= MakeNodesStrategy && time.Since(startStateProposedTime) >= makeAssertionInterval) {
		// There's no correct node; create one.
		var lastNodeHashIfExists *common.Hash
		if len(successorNodes) > 0 {
			lastNodeHashIfExists = &successorNodes[len(successorNodes)-1].NodeHash
		}
		action, err := v.createNewNodeAction(ctx, stakerInfo, prevInboxMaxCount, startCount, startState, validatedCount, validatedGlobalState, lastNodeHashIfExists)
		if err != nil {
			return nil, wrongNodesExist, fmt.Errorf("error generating create new node action (from pos %d to %d): %w", startCount, validatedCount, err)
		}
		return action, wrongNodesExist, nil
	}

	return nil, wrongNodesExist, nil
}

func (v *L1Validator) createNewNodeAction(
	ctx context.Context,
	stakerInfo *OurStakerInfo,
	prevInboxMaxCount *big.Int,
	startCount arbutil.MessageIndex,
	startState *validator.ExecutionState,
	validatedCount arbutil.MessageIndex,
	validatedGS validator.GoGlobalState,
	lastNodeHashIfExists *common.Hash,
) (nodeAction, error) {
	if !prevInboxMaxCount.IsUint64() {
		return nil, fmt.Errorf("inbox max count %v isn't a uint64", prevInboxMaxCount)
	}
	if validatedCount <= startCount {
		// we haven't validated any new blocks
		return nil, nil
	}
	if validatedGS.Batch < prevInboxMaxCount.Uint64() {
		// didn't validate enough batches
		log.Info("staker: not enough batches validated to create new assertion", "validated.Batch", validatedGS.Batch, "posInBatch", validatedGS.PosInBatch, "required batch", prevInboxMaxCount)
		return nil, nil
	}
	batchValidated := validatedGS.Batch
	if validatedGS.PosInBatch == 0 {
		batchValidated--
	}
	validatedBatchAcc, err := v.inboxTracker.GetBatchAcc(batchValidated)
	if err != nil {
		return nil, fmt.Errorf("error getting batch %v accumulator: %w", batchValidated, err)
	}

	hasSiblingByte := [1]byte{0}
	prevNum := stakerInfo.LatestStakedNode
	lastHash := stakerInfo.LatestStakedNodeHash
	if lastNodeHashIfExists != nil {
		lastHash = *lastNodeHashIfExists
		hasSiblingByte[0] = 1
	}
	assertionNumBlocks := uint64(validatedCount - startCount)
	assertion := &Assertion{
		BeforeState: startState,
		AfterState: &validator.ExecutionState{
			GlobalState:   validatedGS,
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

// Returns (execution state, inbox max count, L1 block proposed, parent chain block proposed, error)
func lookupNodeStartState(ctx context.Context, rollup *RollupWatcher, nodeNum uint64, nodeHash common.Hash) (*validator.ExecutionState, *big.Int, uint64, uint64, error) {
	if nodeNum == 0 {
		creationEvent, err := rollup.LookupCreation(ctx)
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("error looking up rollup creation event: %w", err)
		}
		l1BlockNumber, err := arbutil.CorrespondingL1BlockNumber(ctx, rollup.client, creationEvent.Raw.BlockNumber)
		if err != nil {
			return nil, nil, 0, 0, err
		}
		return &validator.ExecutionState{
			GlobalState:   validator.GoGlobalState{},
			MachineStatus: validator.MachineStatusFinished,
		}, big.NewInt(1), l1BlockNumber, creationEvent.Raw.BlockNumber, nil
	}
	node, err := rollup.LookupNode(ctx, nodeNum)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	if node.NodeHash != nodeHash {
		return nil, nil, 0, 0, errors.New("looked up starting node but found wrong hash")
	}
	return node.AfterState(), node.InboxMaxCount, node.L1BlockProposed, node.ParentChainBlockProposed, nil
}
