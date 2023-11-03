package arbos

import (
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbosState"
)

// extraTxFilter should be modified by chain operators to enforce additional transaction validity rules
func extraTxFilter(
	chainConfig *params.ChainConfig,
	currentBlockHeader *types.Header,
	statedb *state.StateDB,
	state *arbosState.ArbosState,
	tx *types.Transaction,
	options *arbitrum_types.ConditionalOptions,
	sender common.Address,
	l1Info *L1Info,
	result *core.ExecutionResult,
) error {
	// TODO: implement additional checks
	return nil
}
