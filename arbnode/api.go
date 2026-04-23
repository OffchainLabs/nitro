// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbnode

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
)

var (
	getL1ConfirmationCallsCounter        = metrics.NewRegisteredCounter("arb/consensus_rpc_get_l1_confirmation_calls", nil)
	findBatchContainingBlockCallsCounter = metrics.NewRegisteredCounter("arb/consensus_rpc_find_batch_containing_block_calls", nil)
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
	consensusNode   *Node
	genesisBlockNum uint64
}

func NewArbAPI(consensusNode *Node, genesisBlockNum uint64) *ArbAPI {
	return &ArbAPI{
		consensusNode:   consensusNode,
		genesisBlockNum: genesisBlockNum,
	}
}

func (a *ArbAPI) GetL1Confirmations(ctx context.Context, blockNum uint64) (uint64, error) {
	getL1ConfirmationCallsCounter.Inc(1)

	msgIdx, err := arbutil.BlockNumberToMessageIndex(blockNum, a.genesisBlockNum)
	if err != nil {
		return 0, err
	}
	return a.consensusNode.GetL1Confirmations(msgIdx).Await(ctx)
}

func (a *ArbAPI) FindBatchContainingBlock(ctx context.Context, blockNum uint64) (uint64, error) {
	findBatchContainingBlockCallsCounter.Inc(1)

	msgIdx, err := arbutil.BlockNumberToMessageIndex(blockNum, a.genesisBlockNum)
	if err != nil {
		return 0, err
	}
	return a.consensusNode.FindBatchContainingMessage(msgIdx).Await(ctx)
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
