package arbnode

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
)

type BlockValidatorAPI struct {
	val *staker.BlockValidator
}

func (a *BlockValidatorAPI) LatestValidated(ctx context.Context) (*staker.GlobalStateValidatedInfo, error) {
	return a.val.ReadLastValidatedInfo()
}

type BlockValidatorDebugAPI struct {
	val *staker.StatelessBlockValidator
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
