// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethhook

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

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

func (p ArbosPrecompileWrapper) Name() string {
	return p.inner.Name()
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

	addPrecompiles(&vm.PrecompiledAddressesBeforeArbOS30, vm.PrecompiledContractsBeforeArbOS30, vm.PrecompiledContractsBerlin)
	addPrecompiles(&vm.PrecompiledAddressesStartingFromArbOS30, vm.PrecompiledContractsStartingFromArbOS30, vm.PrecompiledContractsCancun)
	addPrecompiles(&vm.PrecompiledAddressesStartingFromArbOS50, vm.PrecompiledContractsStartingFromArbOS50, vm.PrecompiledContractsPrague)
	addPrecompiles(&vm.PrecompiledAddressesStartingFromArbOS50, vm.PrecompiledContractsStartingFromArbOS50, vm.PrecompiledContractsOsaka)

	precompileErrors := make(map[[4]byte]abi.Error)
	for addr, precompile := range precompiles.Precompiles() {
		for _, errABI := range precompile.Precompile().GetErrorABIs() {
			precompileErrors[[4]byte(errABI.ID.Bytes())] = errABI
		}
		var wrapped vm.AdvancedPrecompile = ArbosPrecompileWrapper{precompile}
		vm.PrecompiledContractsStartingFromArbOS30[addr] = wrapped
		vm.PrecompiledAddressesStartingFromArbOS30 = append(vm.PrecompiledAddressesStartingFromArbOS30, addr)

		if precompile.Precompile().ArbosVersion() < params.ArbosVersion_Stylus {
			vm.PrecompiledContractsBeforeArbOS30[addr] = wrapped
			vm.PrecompiledAddressesBeforeArbOS30 = append(vm.PrecompiledAddressesBeforeArbOS30, addr)
		}
	}

	addPrecompiles(&vm.PrecompiledAddressesStartingFromArbOS30, vm.PrecompiledContractsStartingFromArbOS30, vm.PrecompiledContractsBeforeArbOS30)
	addPrecompiles(&vm.PrecompiledAddressesStartingFromArbOS30, vm.PrecompiledContractsStartingFromArbOS30, vm.PrecompiledContractsP256Verify)
	addPrecompiles(&vm.PrecompiledAddressesStartingFromArbOS50, vm.PrecompiledContractsStartingFromArbOS50, vm.PrecompiledContractsStartingFromArbOS30)

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

func addPrecompiles(addresses *[]common.Address, contracts map[common.Address]vm.PrecompiledContract, toAdd map[common.Address]vm.PrecompiledContract) {
	for addr, precompile := range toAdd {
		contracts[addr] = precompile
		*addresses = append(*addresses, addr)
	}
}

// RequireHookedGeth does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
