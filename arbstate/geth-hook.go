// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/util"
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

	core.RenderRPCError = func(data []byte) error {
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

	types.ArbitrumSubmitRetryableTxDataHook = func(tx *types.ArbitrumSubmitRetryableTx) []byte {
		toToEncode := common.Address{}
		if tx.RetryTo != nil {
			toToEncode = *tx.RetryTo
		}
		data, err := util.PackArbRetryableTxSubmitRetryable(
			tx.RequestId,
			tx.L1BaseFee,
			tx.DepositValue,
			tx.Value,
			tx.GasFeeCap,
			tx.Gas,
			tx.MaxSubmissionFee,
			tx.FeeRefundAddr,
			tx.Beneficiary,
			toToEncode,
			tx.RetryData,
		)
		if err != nil {
			log.Error("failed to abi-encode submission data", "err", err)
			return nil
		}
		return data
	}
}

// Does nothing, but forces an import to let the init function run
func RequireHookedGeth() {}
