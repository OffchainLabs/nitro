package arbstate

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/arbstate/arbos2"
)

type ArbosPrecompileWrapper struct {
	inner arbos2.ArbosPrecompile
}

func (p ArbosPrecompileWrapper) RequiredGas(input []byte) uint64 {
	panic("Non-advanced precompile method called")
}

func (p ArbosPrecompileWrapper) Run(input []byte) ([]byte, error) {
	panic("Non-advanced precompile method called")
}

func (p ArbosPrecompileWrapper) RunAdvanced(input []byte, suppliedGas uint64, info *vm.AdvancedPrecompileCall) (ret []byte, remainingGas uint64, err error) {
	gasUsage := p.inner.GasToCharge(input)
	if gasUsage > suppliedGas {
		return nil, 0, vm.ErrOutOfGas
	}
	output, err := p.inner.Call(input, info.PrecompileAddress, info.ActingAsAddress, info.Caller, info.Value, info.ReadOnly, info.Evm)
	return output, suppliedGas - gasUsage, err
}

func init() {
	core.ExtraGasChargingHook = arbos2.ExtraGasChargingHook
	core.EndTxHook = arbos2.EndTxHook
	for addr, precompile := range arbos2.ArbosPrecompiles {
		var wrapped vm.AdvancedPrecompile = ArbosPrecompileWrapper{precompile}
		vm.ExtraPrecompiles[addr] = wrapped
	}
}

// Does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
