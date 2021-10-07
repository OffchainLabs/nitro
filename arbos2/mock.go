package arbos2

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type MessageSegment struct {
	contents      []byte
	isL2BatchItem bool
	aggregator    *common.Address
}

func SplitInboxMessage(bytes []byte) ([]MessageSegment, error) {
	return nil, errors.New("TODO")
}

func CreateBlockTemplate(beforeState *state.StateDB, lastHeader *types.Header, segment MessageSegment) (*types.Block, error) {
	return nil, nil
}

func Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) {
}

func ExtraGasChargingHook(msg core.Message, txGasRemaining *uint64, gasPool *core.GasPool, state vm.StateDB) error {
	return nil
}

func EndTxHook(msg core.Message, totalGasUsed uint64, extraGasCharged uint64, gasPool *core.GasPool, success bool, state vm.StateDB) error {
	return nil
}

type ArbosPrecompile interface {
	GasToCharge(input []byte) uint64
	Call(input []byte, precompileAddress common.Address, actingAsAddress common.Address, caller common.Address, value *big.Int, readOnly bool, evm *vm.EVM) (output []byte, err error)
}

var ArbosPrecompiles map[common.Address]ArbosPrecompile
