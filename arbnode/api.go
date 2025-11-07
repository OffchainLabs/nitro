package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
)

type BlockValidatorAPI struct {
	val *staker.BlockValidator
}

func NewBlockValidatorAPI(val *staker.BlockValidator) *BlockValidatorAPI {
	return &BlockValidatorAPI{
		val: val,
	}
}

func (a *BlockValidatorAPI) LatestValidated(ctx context.Context) (*staker.GlobalStateValidatedInfo, error) {
	return a.val.ReadLastValidatedInfo()
}

type ArbAPI struct {
	execClient        execution.ExecutionClient
	inboxTracker      *InboxTracker
	parentChainReader *headerreader.HeaderReader
}

func NewArbAPI(
	execClient execution.ExecutionClient,
	inboxTracker *InboxTracker,
	parentChainReader *headerreader.HeaderReader,
) *ArbAPI {
	return &ArbAPI{
		execClient:        execClient,
		inboxTracker:      inboxTracker,
		parentChainReader: parentChainReader,
	}
}

func (a *ArbAPI) GetL1Confirmations(ctx context.Context, blockNum uint64) (uint64, error) {
	// blocks behind genesis are treated as belonging to batch 0
	msgNum, err := a.execClient.BlockNumberToMessageIndex(blockNum).Await(ctx)
	if err != nil {
		if !errors.Is(err, gethexec.BlockNumBeforeGenesis) {
			return 0, err
		}
		msgNum = 0
	}

	// batches not yet posted have 0 confirmations but no error
	batchNum, found, err := a.inboxTracker.FindInboxBatchContainingMessage(msgNum)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}
	parentChainBlockNum, err := a.inboxTracker.GetBatchParentChainBlock(batchNum)
	if err != nil {
		return 0, err
	}

	if a.parentChainReader == nil {
		return 0, nil
	}
	if a.parentChainReader.IsParentChainArbitrum() {
		parentChainClient := a.parentChainReader.Client()
		parentNodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, parentChainClient)
		if err != nil {
			return 0, err
		}
		parentChainBlock, err := parentChainClient.BlockByNumber(ctx, new(big.Int).SetUint64(parentChainBlockNum))
		if err != nil {
			// Hide the parent chain RPC error from the client in case it contains sensitive information.
			// Likely though, this error is just "not found" because the block got reorg'd.
			return 0, fmt.Errorf("failed to get parent chain block %v containing batch", parentChainBlockNum)
		}
		confs, err := parentNodeInterface.GetL1Confirmations(&bind.CallOpts{Context: ctx}, parentChainBlock.Hash())
		if err != nil {
			log.Warn(
				"Failed to get L1 confirmations from parent chain",
				"blockNumber", parentChainBlockNum,
				"blockHash", parentChainBlock.Hash(), "err", err,
			)
			return 0, fmt.Errorf("failed to get L1 confirmations from parent chain for block %v", parentChainBlock.Hash())
		}
		return confs, nil
	}
	latestHeader, err := a.parentChainReader.LastHeaderWithError()
	if err != nil {
		return 0, err
	}
	if latestHeader == nil {
		return 0, errors.New("no headers read from l1")
	}
	latestBlockNum := latestHeader.Number.Uint64()
	if latestBlockNum < parentChainBlockNum {
		return 0, nil
	}
	return (latestBlockNum - parentChainBlockNum), nil
}

func (a *ArbAPI) FindBatchContainingBlock(ctx context.Context, blockNum uint64) (uint64, error) {
	msgIndex, err := a.execClient.BlockNumberToMessageIndex(blockNum).Await(ctx)
	if err != nil {
		if errors.Is(err, gethexec.BlockNumBeforeGenesis) {
			return 0, fmt.Errorf("block %v is part of genesis", blockNum)
		}
		return 0, err
	}

	res, found, err := a.inboxTracker.FindInboxBatchContainingMessage(msgIndex)
	if err == nil && !found {
		return 0, errors.New("block not yet found on any batch")
	}
	return res, err
}

type BlockValidatorDebugAPI struct {
	val *staker.StatelessBlockValidator
}

func NewBlockValidatorDebugAPI(val *staker.StatelessBlockValidator) *BlockValidatorDebugAPI {
	return &BlockValidatorDebugAPI{
		val: val,
	}
}

type ValidateBlockResult struct {
	Valid       bool                    `json:"valid"`
	Latency     string                  `json:"latency"`
	GlobalState validator.GoGlobalState `json:"globalstate"`
}

func (a *BlockValidatorDebugAPI) ValidateMessageNumber(
	ctx context.Context, msgNum hexutil.Uint64, full bool, moduleRootOptional *common.Hash,
) (ValidateBlockResult, error) {
	result := ValidateBlockResult{}

	var moduleRoot common.Hash
	if moduleRootOptional != nil {
		moduleRoot = *moduleRootOptional
	} else {
		moduleRoot = a.val.GetLatestWasmModuleRoot()
	}
	start_time := time.Now()
	valid, gs, err := a.val.ValidateResult(ctx, arbutil.MessageIndex(msgNum), full, moduleRoot)
	result.Latency = fmt.Sprintf("%vms", time.Since(start_time).Milliseconds())
	if gs != nil {
		result.GlobalState = *gs
	}
	result.Valid = valid
	return result, err
}

func (a *BlockValidatorDebugAPI) ValidationInputsAt(ctx context.Context, msgNum hexutil.Uint64, target rawdb.WasmTarget,
) (server_api.InputJSON, error) {
	return a.val.ValidationInputsAt(ctx, arbutil.MessageIndex(msgNum), target)
}
