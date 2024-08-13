package arbnode

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator"
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
		var err error
		moduleRoot, err = a.val.GetLatestWasmModuleRoot(ctx)
		if err != nil {
			return result, fmt.Errorf("no latest WasmModuleRoot configured, must provide parameter: %w", err)
		}
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
