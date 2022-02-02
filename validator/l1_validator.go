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
	fromBlock int64,
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
	rollup, err := NewRollupWatcher(wallet.RollupAddress(), fromBlock, builder, callOpts)
	_ = rollup
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

func (v *Validator) resolveNextNode(ctx context.Context, info *StakerInfo, fromBlock int64) error {
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
	*StakerInfo
}

func (v *Validator) blockNumberFromBatchCount(batch uint64) (uint64, error) {
	var height uint64
	if batch > 0 {
		var err error
		height, err = v.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return 0, err
		}
	}
	return height + v.genesisBlockNumber, nil
}

func (v *Validator) generateNodeAction(ctx context.Context, stakerInfo *OurStakerInfo, strategy StakerStrategy, fromBlock int64) (nodeAction, bool, error) {
	startState, startStateProposed, err := lookupNodeStartState(ctx, v.rollup, stakerInfo.LatestStakedNode, stakerInfo.LatestStakedNodeHash)
	if err != nil {
		return nil, false, err
	}

	localBatchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return nil, false, err
	}
	if localBatchCount < startState.GlobalState.Batch {
		log.Info("catching up to chain batches", "localBatches", localBatchCount, "target", startState.GlobalState.Batch)
		return nil, false, nil
	}

	startBlock := v.l2Blockchain.GetBlockByHash(startState.GlobalState.BlockHash)
	if startBlock == nil {
		expectedBlockHeight, err := v.blockNumberFromBatchCount(startState.GlobalState.Batch)
		if err != nil {
			return nil, false, err
		}
		latestHeader := v.l2Blockchain.CurrentHeader()
		if latestHeader.Number.Uint64() < expectedBlockHeight {
			log.Info("catching up to chain blocks", "localBlocks", latestHeader.Number, "target", expectedBlockHeight)
			return nil, false, errors.New("unknown start block hash")
		} else {
			log.Info("unknown start block hash", "hash", startState.GlobalState.BlockHash, "batch", startState.GlobalState.Batch)
			return nil, false, errors.New("unknown start block hash")
		}
	}

	blocksValidated := v.blockValidator.BlocksValidated()

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
	successorNodes, err := v.rollup.LookupNodeChildren(ctx, stakerInfo.LatestStakedNodeHash, startStateProposed)
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
			afterGs := nd.Assertion.AfterState.GlobalState
			if afterGs.PosInBatch != 0 {
				return nil, false, fmt.Errorf("non-zero position in batch in assertion: batch %v pos %v", afterGs.Batch, afterGs.PosInBatch)
			}
			lastBlockNum, err := v.blockNumberFromBatchCount(afterGs.Batch)
			if err != nil {
				return nil, false, err
			}
			if blocksValidated < lastBlockNum {
				return nil, false, fmt.Errorf("waiting for validator to catch up to assertion: %v/%v", blocksValidated, lastBlockNum)
			}
			lastBlock := v.l2Blockchain.GetBlockByNumber(lastBlockNum)
			if lastBlock == nil {
				return nil, false, fmt.Errorf("block %v not in database despite being validated", lastBlockNum)
			}
			lastBlockExtra, err := arbos.DeserializeHeaderExtraInformation(lastBlock.Header())
			if err != nil {
				return nil, false, err
			}

			valid := nd.Assertion.NumBlocks == lastBlockNum-startBlock.NumberU64() &&
				afterGs.BlockHash == lastBlock.Hash() &&
				afterGs.SendRoot == lastBlockExtra.SendRoot
			if valid {
				log.Info("found correct node", "node", nd.NodeNum)
				correctNode = existingNodeAction{
					number: nd.NodeNum,
					hash:   nd.NodeHash,
				}
				continue
			} else {
				log.Warn("found node with incorrect assertion", "node", nd.NodeNum)
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

	if new(big.Int).SetUint64(localBatchCount).Cmp(startState.InboxMaxCount) < 0 {
		// not enough batches in database
		return nil, wrongNodesExist, nil
	}

	var validatedBatchCount uint64
	var validatedBatchBlockNum uint64
	for i := localBatchCount; i > startState.GlobalState.Batch; i-- {
		if i == 0 {
			break
		}
		blockNum, err := v.inboxTracker.GetBatchMessageCount(i - 1)
		if err != nil {
			return nil, false, err
		}
		blockNum += v.genesisBlockNumber
		if blockNum > blocksValidated {
			continue
		}
		validatedBatchCount = i
		validatedBatchBlockNum = blockNum
		break
	}
	if validatedBatchCount == 0 {
		// we haven't validated any new batches
		return nil, wrongNodesExist, nil
	}
	validatedBatchAcc, err := v.inboxTracker.GetBatchAcc(validatedBatchCount - 1)
	if err != nil {
		return nil, false, err
	}

	assertingBlock := v.l2Blockchain.GetBlockByNumber(validatedBatchBlockNum)
	if assertingBlock == nil {
		return nil, false, fmt.Errorf("missing validated block %v", validatedBatchBlockNum)
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
	assertion := &Assertion{
		BeforeState: startState.ExecutionState,
		AfterState: &ExecutionState{
			GlobalState: GoGlobalState{
				BlockHash:  assertingBlock.Hash(),
				SendRoot:   assertingBlockExtra.SendRoot,
				Batch:      localBatchCount,
				PosInBatch: 0,
			},
			MachineStatus: MachineStatusFinished,
		},
		NumBlocks: assertingBlock.NumberU64() - startBlock.NumberU64(),
	}

	executionHash := assertion.ExecutionHash()
	newNodeHash := crypto.Keccak256Hash(hasSiblingByte[:], lastHash[:], executionHash[:], validatedBatchAcc[:])

	action := createNodeAction{
		assertion:         assertion,
		hash:              newNodeHash,
		prevInboxMaxCount: startState.InboxMaxCount,
	}
	log.Info("creating node", "hash", newNodeHash, "lastNode", lastNum, "parentNode", stakerInfo.LatestStakedNode)
	return action, wrongNodesExist, nil
}

func lookupNodeStartState(ctx context.Context, rollup *RollupWatcher, nodeNum uint64, nodeHash [32]byte) (*NodeState, uint64, error) {
	if nodeNum == 0 {
		creationEvent, err := rollup.LookupCreation(ctx)
		if err != nil {
			return nil, 0, err
		}
		return &NodeState{
			InboxMaxCount: big.NewInt(1),
			ExecutionState: &ExecutionState{
				GlobalState:   GoGlobalState{},
				MachineStatus: MachineStatusFinished,
			},
		}, creationEvent.Raw.BlockNumber, nil
	}
	node, err := rollup.LookupNode(ctx, nodeNum)
	if err != nil {
		return nil, 0, err
	}
	if node.NodeHash != nodeHash {
		return nil, 0, errors.New("looked up starting node but found wrong hash")
	}
	return node.AfterState(), node.BlockProposed, nil
}
