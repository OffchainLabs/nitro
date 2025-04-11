package arbos

import (
	"errors"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
)

// senderBlacklist contains addresses that are blocked from sending transactions.
// Chain operators can populate this list.
var senderBlacklist = map[common.Address]struct{}{
	// Example: common.HexToAddress("0x123..."): {},
}

// maxTxGasLimit defines the maximum gas a single transaction is allowed to consume.
// Chain operators can adjust this value.
const maxTxGasLimit uint64 = 50_000_000 // Example limit, adjust as needed

// extraPreTxFilter should be modified by chain operators to enforce additional pre-transaction validity rules
func extraPreTxFilter(
	chainConfig *params.ChainConfig,
	currentBlockHeader *types.Header,
	statedb *state.StateDB,
	state *arbosState.ArbosState,
	tx *types.Transaction,
	options *arbitrum_types.ConditionalOptions,
	sender common.Address,
	l1Info *L1Info,
) error {
	// Check if the sender is in the blacklist
	if _, blocked := senderBlacklist[sender]; blocked {
		return errors.New("sender is blocked")
	}

	return nil
}

// extraPostTxFilter should be modified by chain operators to enforce additional post-transaction validity rules
func extraPostTxFilter(
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
	// Check if the transaction exceeded the gas limit
	if result.UsedGas > maxTxGasLimit {
		return errors.New("transaction exceeded gas limit")
	}

	return nil
}
