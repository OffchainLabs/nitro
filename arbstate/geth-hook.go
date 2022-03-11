//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/precompiles"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/util/arbmath"
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

	nodeInterface, err := abi.JSON(strings.NewReader(node_interfacegen.NodeInterfaceABI))
	if err != nil {
		panic(err)
	}
	core.InterceptRPCMessage = func(msg types.Message) (types.Message, error) {
		to := msg.To()
		if to == nil || *to != common.HexToAddress("0xc8") {
			return msg, nil
		}
		return ApplyNodeInterface(msg, nodeInterface)
	}

	core.InterceptRPCGasCap = func(gascap *uint64, msg types.Message, header *types.Header, statedb *state.StateDB) {
		arbosVersion := arbosState.ArbOSVersion(statedb)
		if arbosVersion == 0 {
			// ArbOS hasn't been installed, so use the vanilla gas cap
			return
		}
		state, err := arbosState.OpenSystemArbosState(statedb, true)
		if err != nil {
			log.Error("failed to open ArbOS state", "err", err)
			return
		}
		poster, _ := state.L1PricingState().ReimbursableAggregatorForSender(msg.From())
		if poster == nil || header.BaseFee.Sign() == 0 {
			// if gas is free or there's no reimbursable poster, the user won't pay for L1 data costs
			return
		}
		posterCost, _ := state.L1PricingState().PosterDataCost(msg, msg.From(), *poster)
		posterCostInL2Gas := arbmath.BigToUintSaturating(arbmath.BigDiv(posterCost, header.BaseFee))
		*gascap = arbmath.SaturatingUAdd(*gascap, posterCostInL2Gas)
	}
}

// Does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
