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
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type TracingScenario uint64

const (
	TracingBeforeEVM TracingScenario = iota
	TracingDuringEVM
	TracingAfterEVM
)

// holds an address to satisfy core/vm's ContractRef() interface
type addressHolder struct {
	addr common.Address
}

func (a addressHolder) Address() common.Address {
	return a.addr
}

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

		input := []byte("Transfer Balance")
		inputLen := uint64(len(input))
		memory := vm.NewMemory()
		memory.Resize(inputLen)
		memory.Set(0, inputLen, input)
		stack := &vm.Stack{}
		stack.SetData([]uint256.Int{
			*uint256.NewInt(0),                          // return size
			*uint256.NewInt(0),                          // return offset
			*uint256.NewInt(inputLen),                   // memory length
			*uint256.NewInt(0),                          // memory offset
			*uint256.NewInt(0).SetBytes(amount.Bytes()), // call value
			*uint256.NewInt(0).SetBytes(to.Bytes()),     // to address
			*uint256.NewInt(0),                          // 0 gas
		})
		contract := vm.NewContract(addressHolder{*to}, addressHolder{*from}, big.NewInt(0), 0)
		scope := &vm.ScopeContext{
			Memory:   memory,
			Stack:    stack,
			Contract: contract,
		}
		tracer.CaptureState(0, vm.CALL, 0, 0, scope, []byte{}, evm.Depth(), nil)
		tracer.CaptureEnter(vm.INVALID, *from, *to, input, 0, amount)

		retStack := &vm.Stack{}
		retStack.SetData([]uint256.Int{
			*uint256.NewInt(0), // return size
			*uint256.NewInt(0), // return offset
		})
		retScope := &vm.ScopeContext{
			Memory:   vm.NewMemory(),
			Stack:    retStack,
			Contract: contract,
		}
		tracer.CaptureState(0, vm.RETURN, 0, 0, retScope, []byte{}, evm.Depth()+1, nil)
		tracer.CaptureExit(nil, 0, nil)

		popStack := &vm.Stack{}
		popStack.SetData([]uint256.Int{
			*uint256.NewInt(1), // CALL result success
		})
		popScope := &vm.ScopeContext{
			Memory:   vm.NewMemory(),
			Stack:    popStack,
			Contract: contract,
		}
		tracer.CaptureState(0, vm.POP, 0, 0, popScope, []byte{}, evm.Depth(), nil)
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
