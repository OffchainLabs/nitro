package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
)


type ArbosAPI interface {
	SplitInboxMessage(bytes []byte) ([]MessageSegment, error)

	// StateDB can be used to read or write storage slots, balances, etc.
	FinalizeBlock(header *types.Header, state *state.StateDB, txs types.Transactions)

	// Can be used to charge aggregator costs
	// Should return ErrIntrinsicGas if there isn't enough gas
	ExtraGasChargingHook(msg core.Message, txGasRemaining *uint64, gasPool *core.GasPool, state vm.StateDB) error

	// Can be used to give burnt gas fees to the fee recipient
	// extraGasCharged is any gas remaining subtracted during the ExtraGasChargingHook, which is also included in totalGasUsed
	EndTxHook(
		msg core.Message,
		totalGasUsed uint64,
		extraGasCharged uint64,
		gasPool *core.GasPool,
		state vm.StateDB,
	) error

	Precompiles() map[common.Address]ArbosPrecompile
}

type MessageSegment interface {
	// StateDB can be used to read *but not write* arbitrary storage slots, balances, etc.
	CreateBlockContents(
		beforeState *state.StateDB,
	) (
		[]*types.Transaction,   // transactions to (try to) put in the block
		*big.Int, 	            // timestamp
		common.Address,         // coinbase address
		error,
	)
}

type ArbosPrecompile interface {
	GasToCharge(input []byte) uint64

	// Important fields: evm.StateDB and evm.Config.Tracer
	// NOTE: if precompileAddress != actingAsAddress, watch out! This is a delegatecall or callcode, so caller might be wrong. In that case, unless this precompile is pure, it should probably revert.
	Call(
		input []byte,
		precompileAddress common.Address,
		actingAsAddress common.Address,
		caller common.Address,
		value common.Address,
		readOnly bool,
		evm *vm.EVM,
	) (output []byte, err error)
}
