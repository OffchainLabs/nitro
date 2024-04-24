// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethhook

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/precompiles"
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

	// Precompiles don't actually enter evm execution like normal calls do,
	// so we need to increment the depth here to simulate the callstack change.
	info.Evm.IncrementDepth()
	defer info.Evm.DecrementDepth()

	return p.inner.Call(
		input, info.PrecompileAddress, info.ActingAsAddress,
		info.Caller, info.Value, info.ReadOnly, gasSupplied, info.Evm,
	)
}

func init() {
	core.ReadyEVMForL2 = func(evm *vm.EVM, msg *core.Message) {
		if evm.ChainConfig().IsArbitrum() {
			evm.ProcessingHook = arbos.NewTxProcessor(evm, msg)
		}
	}

	for k, v := range vm.PrecompiledContractsBerlin {
		vm.PrecompiledAddressesArbitrum = append(vm.PrecompiledAddressesArbitrum, k)
		vm.PrecompiledContractsArbitrum[k] = v
	}

	precompileErrors := make(map[[4]byte]abi.Error)
	for addr, precompile := range precompiles.Precompiles() {
		for _, errABI := range precompile.Precompile().GetErrorABIs() {
			var id [4]byte
			copy(id[:], errABI.ID[:4])
			precompileErrors[id] = errABI
		}
		var wrapped vm.AdvancedPrecompile = ArbosPrecompileWrapper{precompile}
		vm.PrecompiledContractsArbitrum[addr] = wrapped
		vm.PrecompiledAddressesArbitrum = append(vm.PrecompiledAddressesArbitrum, addr)
	}

	for addr, precompile := range vm.PrecompiledContractsArbitrum {
		vm.PrecompiledContractsArbOS30[addr] = precompile
		vm.PrecompiledAddressesArbOS30 = append(vm.PrecompiledAddressesArbOS30, addr)
	}
	for addr, precompile := range vm.PrecompiledContractsP256Verify {
		vm.PrecompiledContractsArbOS30[addr] = precompile
		vm.PrecompiledAddressesArbOS30 = append(vm.PrecompiledAddressesArbOS30, addr)
	}

	core.RenderRPCError = func(data []byte) error {
		if len(data) < 4 {
			return nil
		}
		var id [4]byte
		copy(id[:], data[:4])
		errABI, found := precompileErrors[id]
		if !found {
			return nil
		}
		rendered, err := precompiles.RenderSolError(errABI, data)
		if err != nil {
			log.Warn("failed to render rpc error", "err", err)
			return nil
		}
		return errors.New(rendered)
	}
}

// RequireHookedGeth does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
