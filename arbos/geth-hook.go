package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ArbosPrecompileWrapper struct {
	inner ArbosPrecompile
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
var arbAddress = common.HexToAddress("0xabc")

func init() {
	core.ArbProcessMessage = func(msg core.Message, state vm.StateDB) (*core.ExecutionResult, error) {
		if msg.From() != arbAddress {
			return nil, nil
		}
		// Message is deposit
		state.AddBalance(*msg.To(), msg.Value())
		return &core.ExecutionResult{
			UsedGas:    0,
			Err:        nil,
			ReturnData: nil,
		}, nil
	}
	core.ExtraGasChargingHook = func(msg core.Message, gasRemaining *uint64, gasPool *core.GasPool, stateDb vm.StateDB) error {
		l1Charges, err := Initialize(stateDb).StartTxHook(msg)
		if err != nil {
			return err
		} else if *gasRemaining < l1Charges {
			return vm.ErrOutOfGas
		}
		*gasRemaining -= l1Charges
		*gasPool = *gasPool.AddGas(l1Charges)
		return nil
	}
	core.EndTxHook = func(msg core.Message, totalGasUsed uint64, extraGasCharged uint64, gasPool *core.GasPool, success bool, stateDb vm.StateDB) error {
		return Initialize(stateDb).EndTxHook(msg, totalGasUsed, extraGasCharged)
	}
	for addr, precompile := range Precompiles() {
		var wrapped vm.AdvancedPrecompile = ArbosPrecompileWrapper{precompile}
		vm.ExtraPrecompiles[addr] = wrapped
	}
}

// Does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
