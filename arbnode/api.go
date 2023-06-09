package arbnode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/staker"
)

type BlockValidatorAPI struct {
	val *staker.BlockValidator
}

func (a *BlockValidatorAPI) LatestValidatedBlock(ctx context.Context) (hexutil.Uint64, error) {
	block := a.val.LastBlockValidated()
	return hexutil.Uint64(block), nil
}

func (a *BlockValidatorAPI) LatestValidatedBlockHash(ctx context.Context) (common.Hash, error) {
	_, hash, _ := a.val.LastBlockValidatedAndHash()
	return hash, nil
}

type BlockValidatorDebugAPI struct {
	val        *staker.StatelessBlockValidator
	blockchain *core.BlockChain
}

type ValidateBlockResult struct {
	Valid   bool   `json:"valid"`
	Latency string `json:"latency"`
}

func (a *BlockValidatorDebugAPI) ValidateBlock(
	ctx context.Context, blockNum rpc.BlockNumber, full bool, moduleRootOptional *common.Hash,
) (ValidateBlockResult, error) {
	result := ValidateBlockResult{}

	if blockNum < 0 {
		return result, errors.New("this method only accepts absolute block numbers")
	}
	header := a.blockchain.GetHeaderByNumber(uint64(blockNum))
	if header == nil {
		return result, errors.New("block not found")
	}
	if !a.blockchain.Config().IsArbitrumNitro(header.Number) {
		return result, types.ErrUseFallback
	}
	var moduleRoot common.Hash
	if moduleRootOptional != nil {
		moduleRoot = *moduleRootOptional
	} else {
		moduleRoots := a.val.GetModuleRootsToValidate()
		if len(moduleRoots) == 0 {
			return result, errors.New("no current WasmModuleRoot configured, must provide parameter")
		}
		moduleRoot = moduleRoots[0]
	}
	start_time := time.Now()
	valid, err := a.val.ValidateBlock(ctx, header, full, moduleRoot)
	result.Valid = valid
	result.Latency = fmt.Sprintf("%vms", time.Since(start_time).Milliseconds())
	return result, err
}
