//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
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
	gasSupplied uint64,
	info *vm.AdvancedPrecompileCall,
) (ret []byte, gasLeft uint64, err error) {
	return p.inner.Call(
		input, info.PrecompileAddress, info.ActingAsAddress,
		info.Caller, info.Value, info.ReadOnly, gasSupplied, info.Evm,
	)
}

func init() {
	core.ReadyEVMForL2 = func(evm *vm.EVM, msg core.Message) {
		if evm.ChainConfig().IsArbitrum() {
			evm.ProcessingHook = arbos.NewTxProcessor(evm, msg)
		}
	}

	for k, v := range vm.PrecompiledContractsBerlin {
		vm.PrecompiledAddressesArbitrum = append(vm.PrecompiledAddressesArbitrum, k)
		vm.PrecompiledContractsArbitrum[k] = v
	}

	for addr, precompile := range precompiles.Precompiles() {
		var wrapped vm.AdvancedPrecompile = ArbosPrecompileWrapper{precompile}
		vm.PrecompiledContractsArbitrum[addr] = wrapped
		vm.PrecompiledAddressesArbitrum = append(vm.PrecompiledAddressesArbitrum, addr)
	}
}

// Does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
