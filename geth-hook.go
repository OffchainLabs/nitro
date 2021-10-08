package arbstate

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/arbstate/arbos"
)

type ArbosPrecompileWrapper struct {
	inner arbos.ArbosPrecompile
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
	core.ExtraGasChargingHook = func(msg core.Message, gasRemaining *uint64, gasPool *core.GasPool, stateInterface vm.StateDB) error {
		stateDb := stateInterface.(*state.StateDB)
		l1Charges, err := arbos.Initialize(stateDb).StartTxHook(msg, stateInterface)
		if err != nil {
			return err
		} else if *gasRemaining < l1Charges {
			return vm.ErrOutOfGas
		}
		*gasRemaining -= l1Charges
		*gasPool = *gasPool.AddGas(l1Charges)
		return nil
	}
	core.EndTxHook = func(msg core.Message, totalGasUsed uint64, extraGasCharged uint64, gasPool *core.GasPool, success bool, stateInterface vm.StateDB) error {
		stateDb := stateInterface.(*state.StateDB)
		return arbos.Initialize(stateDb).EndTxHook(msg, totalGasUsed, extraGasCharged, stateInterface)
	}
	for addr, precompile := range arbos.Precompiles() {
		var wrapped vm.AdvancedPrecompile = ArbosPrecompileWrapper{precompile}
		vm.ExtraPrecompiles[addr] = wrapped
	}
}

// Does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
