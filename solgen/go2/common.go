package precompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func checkCall(
	input []byte,
	precompileAddress common.Address,
	actingAsAddress common.Address,
	value *big.Int,
	readOnly bool,
	evmABI *abi.ABI,
) ([]interface{}, abi.Arguments, error) {
	method, err := evmABI.MethodById(input)
	if err != nil {
		return nil, nil, err
	}
	if method.StateMutability != "pure" && actingAsAddress != precompileAddress {
		// should not access precompile superpowers when not acting as the precompile
		return nil, nil, vm.ErrExecutionReverted
	}
	if !method.IsConstant() && readOnly {
		// tried to write to global state in read-only mode
		return nil, nil, vm.ErrExecutionReverted
	}
	if !method.Payable && value.BitLen() == 0 {
		return nil, nil, errors.New("value sent to non-payable method")
	}
	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, nil, err
	}
	return args, method.Outputs, err
}
