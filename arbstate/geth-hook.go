//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/precompiles"
)

type ArbosPrecompileWrapper struct {
	inner precompiles.ArbosPrecompile
}

func (p ArbosPrecompileWrapper) RequiredGas(input []byte) uint64 {
	panic("Non-advanced precompile method called")
}

func (p ArbosPrecompileWrapper) Run(input []byte) ([]byte, error) {
	panic("Non-advanced precompile method called")
}

func (p ArbosPrecompileWrapper) RunAdvanced(
	input []byte,
	suppliedGas uint64,
	info *vm.AdvancedPrecompileCall,
) (ret []byte, remainingGas uint64, err error) {
	gasUsage := p.inner.GasToCharge(input)
	if gasUsage > suppliedGas {
		return nil, 0, vm.ErrOutOfGas
	}
	output, err := p.inner.Call(input, info.PrecompileAddress, info.ActingAsAddress, info.Caller, info.Value, info.ReadOnly, info.Evm)
	return output, suppliedGas - gasUsage, err
}

func init() {
	core.CreateTxProcessingHook = func(msg core.Message, evm *vm.EVM) core.TxProcessingHook {
		if evm.ChainConfig().IsArbitrum(big.NewInt(0)) {
			return arbos.NewTxProcessor(msg, evm)
		}
		return nil
	}
	for addr, precompile := range precompiles.Precompiles() {
		var wrapped vm.AdvancedPrecompile = ArbosPrecompileWrapper{precompile}
		vm.ExtraPrecompiles[addr] = wrapped
	}
}

// Does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
