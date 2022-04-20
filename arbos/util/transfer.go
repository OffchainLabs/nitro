//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package util

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Represents a balance change occuring aside from a call.
// While most uses will be transfers, setting `from` or `to` to nil will mint or burn funds, respectively.
func TransferBalance(from, to *common.Address, amount *big.Int, evm *vm.EVM, scenario TracingScenario) error {
	if from != nil {
		balance := evm.StateDB.GetBalance(*from)
		if arbmath.BigLessThan(balance, amount) {
			return fmt.Errorf("%w: addr %v have %v want %v", vm.ErrInsufficientBalance, *from, balance, amount)
		}
		evm.StateDB.SubBalance(*from, amount)
	}
	if to != nil {
		evm.StateDB.AddBalance(*to, amount)
	}
	if evm.Config.Debug {
		tracer := evm.Config.Tracer

		if evm.Depth() != 0 && scenario != TracingDuringEVM {
			// A non-zero depth implies this transfer is occuring inside EVM execution
			log.Error("Tracing scenario mismatch", "scenario", scenario, "depth", evm.Depth())
			return errors.New("Tracing scenario mismatch")
		}

		if scenario != TracingDuringEVM {
			tracer.CaptureArbitrumTransfer(evm, from, to, amount, scenario == TracingBeforeEVM)
			return nil
		}

		if from == nil {
			from = &common.Address{}
		}
		if to == nil {
			to = &common.Address{}
		}

		info := &TracingInfo{
			Tracer:   evm.Config.Tracer,
			Scenario: scenario,
			Contract: vm.NewContract(addressHolder{*to}, addressHolder{*from}, big.NewInt(0), 0),
			Depth:    evm.Depth(),
		}
		info.MockCall([]byte{}, 0, *from, *to, amount)
	}
	return nil
}

// Mints funds for the user and adds them to their balance
func MintBalance(to *common.Address, amount *big.Int, evm *vm.EVM, scenario TracingScenario) {
	err := TransferBalance(nil, to, amount, evm, scenario)
	if err != nil {
		panic(fmt.Sprintf("impossible error: %v", err))
	}
}
