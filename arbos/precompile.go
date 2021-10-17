//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ArbosPrecompile interface {
	GasToCharge(input []byte) uint64

	// Important fields: evm.StateDB and evm.Config.Tracer
	// NOTE: if precompileAddress != actingAsAddress, watch out! This is a delegatecall or callcode, so caller might be wrong. In that case, unless this precompile is pure, it should probably revert.
	Call(
		input []byte,
		precompileAddress common.Address,
		actingAsAddress common.Address,
		caller common.Address,
		value *big.Int,
		readOnly bool,
		evm *vm.EVM,
	) (output []byte, err error)
}

func Precompiles() map[common.Address]ArbosPrecompile {
	return nil
}
