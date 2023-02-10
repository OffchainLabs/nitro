// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/validator"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/pkg/errors"
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
	builder        *ValidatorTxBuilder
	wallet         ValidatorWalletInterface
	callOpts       bind.CallOpts

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
	return &L1Validator{
		rollup:         rollup,
		rollupAddress:  wallet.RollupAddress(),
		validatorUtils: validatorUtils,
		client:         client,
		builder:        builder,
		wallet:         wallet,
		callOpts:       callOpts,
		das:            das,
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
	if localBatchCount < startState.RequiredBatches() || localBatchCount == 0 {
		log.Info("catching up to chain batches", "localBatches", localBatchCount, "target", startState.RequiredBatches())
		return nil, false, nil
	}

	caughtUp, startCount, err := GlobalStateToMsgCount(v.inboxTracker, v.txStreamer, startState.GlobalState)
	if err != nil {
		return nil, false, err
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
		return nil, false, err
	}

	var validatedCount arbutil.MessageIndex
	var validatedGlobalState validator.GoGlobalState
	if v.blockValidator != nil {
		valInfo, err := v.blockValidator.ReadLastValidatedInfo()
		if err != nil {
			return nil, false, err
		}
		validatedGlobalState = valInfo.GlobalState
		caughtUp, validatedCount, err = GlobalStateToMsgCount(v.inboxTracker, v.txStreamer, valInfo.GlobalState)
		if err != nil {
			return nil, false, fmt.Errorf("%w: not found validated block in blockchain", err)
		}
		if !caughtUp {
			log.Info("catching up to laste validated block", "target", valInfo.GlobalState)
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
			return nil, false, fmt.Errorf("wasmroot doesn't match rollup : %v, valid: %v", v.lastWasmModuleRoot, valInfo.WasmRoots)
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
			batchNum, err = FindBatchContainingMessageIndex(v.inboxTracker, validatedCount-1, localBatchCount)
			if err != nil {
				return nil, false, err
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

	minAssertionPeriod, err := v.rollup.MinimumAssertionPeriod(v.getCallOpts(ctx))
	if err != nil {
		return nil, false, fmt.Errorf("error getting rollup minimum assertion period: %w", err)
	}

	timeSinceProposed := big.NewInt(int64(currentL1BlockNum) - int64(startStateProposed))
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
		afterGs := nd.AfterState().GlobalState
		requiredBatches := nd.AfterState().RequiredBatches()
		if localBatchCount < requiredBatches {
			return nil, false, fmt.Errorf("waiting for validator to catch up to assertion batches: %v/%v", localBatchCount, requiredBatches)
		}
		if requiredBatches > 0 {
			haveAcc, err := v.inboxTracker.GetBatchAcc(requiredBatches - 1)
			if err != nil {
				return nil, false, fmt.Errorf("%w: error getting batch %v accumulator: localBatchCount: %d", err, requiredBatches-1, localBatchCount)
			}
			if haveAcc != nd.AfterInboxBatchAcc {
				return nil, false, fmt.Errorf("missed sequencer batches reorg: at seq num %v have acc %v but assertion has acc %v", requiredBatches-1, haveAcc, nd.AfterInboxBatchAcc)
			}
		}
		caughtUp, nodeMsgCount, err := GlobalStateToMsgCount(v.inboxTracker, v.txStreamer, startState.GlobalState)
		if errors.Is(err, ErrGlobalStateNotInChain) {
			wrongNodesExist = true
			log.Error("Found incorrect assertion", "err", err)
			continue
		}
		if err != nil {
			return nil, false, fmt.Errorf("error getting block number from global state: %w", err)
		}
		if !caughtUp {
			return nil, false, fmt.Errorf("waiting for validator to catch up to assertion blocks. Current: %d target: %v", validatedCount, startState.GlobalState)
		}
		if validatedCount < nodeMsgCount {
			return nil, false, fmt.Errorf("waiting for validator to catch up to assertion blocks. %d / %d", validatedCount, nodeMsgCount)
		}
		log.Info(
			"found correct assertion",
			"node", nd.NodeNum,
			"count", validatedCount,
			"blockHash", afterGs.BlockHash,
		)
		correctNode = existingNodeAction{
			number: nd.NodeNum,
			hash:   nd.NodeHash,
		}
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
		action, err := v.createNewNodeAction(ctx, stakerInfo, localBatchCount, prevInboxMaxCount, startCount, startState, validatedCount, validatedGlobalState, lastNodeHashIfExists)
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
	localBatchCount uint64,
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
	minBatchCount := prevInboxMaxCount.Uint64()
	if localBatchCount < minBatchCount {
		// not enough batches in database
		return nil, nil
	}

	if localBatchCount == 0 {
		// we haven't validated anything
		return nil, nil
	}
	if validatedCount < startCount {
		// we haven't validated any new blocks
		return nil, nil
	}
	if validatedGS.Batch < minBatchCount {
		// didn't validate enough batches
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

// Returns (execution state, inbox max count, block proposed, error)
func lookupNodeStartState(ctx context.Context, rollup *RollupWatcher, nodeNum uint64, nodeHash [32]byte) (*validator.ExecutionState, *big.Int, uint64, error) {
	if nodeNum == 0 {
		creationEvent, err := rollup.LookupCreation(ctx)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("error looking up rollup creation event: %w", err)
		}
		return &validator.ExecutionState{
			GlobalState:   validator.GoGlobalState{},
			MachineStatus: validator.MachineStatusFinished,
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
