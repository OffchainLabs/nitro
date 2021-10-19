//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"log"
	"math/big"
	"reflect"
	"strings"
	"unicode"

	pre "github.com/offchainlabs/arbstate/arbos/precompiles"
	templates "github.com/offchainlabs/arbstate/precompiles/go"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ArbosPrecompile interface {
	GasToCharge(input []byte) uint64

	// Important fields: evm.StateDB and evm.Config.Tracer
	// NOTE: if precompileAddress != actingAsAddress, watch out!
	// This is a delegatecall or callcode, so caller might be wrong.
	// In that case, unless this precompile is pure, it should probably revert.
	Call(
		input []byte,
		precompileAddress common.Address,
		actingAsAddress common.Address,
		caller common.Address,
		value *big.Int,
		readOnly bool,
		evm *vm.EVM,
	) (output []byte, err error)
}

type purity uint8

const (
	pure purity = iota
	view
	write
	payable
)

type Precompile struct {
	methods map[[4]byte]PrecompileMethod
}

type PrecompileMethod struct {
	name        string
	template    abi.Method
	purity      purity
	handler     reflect.Method
	gascost     reflect.Method
	implementer reflect.Value
}

// Make a precompile for the given hardhat-to-geth bindings, ensuring that the implementer
// supports each method.
func makePrecompile(metadata *bind.MetaData, implementer interface{}) ArbosPrecompile {
	source, err := abi.JSON(strings.NewReader(metadata.ABI))
	if err != nil {
		log.Fatal("Bad ABI")
	}

	contract := reflect.TypeOf(implementer).Name()
	methods := make(map[[4]byte]PrecompileMethod)

	for _, method := range source.Methods {

		name := method.RawName
		capitalize := string(unicode.ToUpper(rune(name[0])))
		name = capitalize + name[1:]
		context := "Precompile " + contract + "'s " + name + "'s implementer "

		if len(method.ID) != 4 {
			log.Fatal("Method ID isn't 4 bytes")
		}
		id := *(*[4]byte)(method.ID)

		// check that the implementer has a supporting implementation for this method

		handler, ok := reflect.TypeOf(implementer).MethodByName(name)
		if !ok {
			log.Fatal("Precompile ", contract, " must implement ", name)
		}

		var needs = []reflect.Type{
			reflect.TypeOf(implementer),      // the contract itself
			reflect.TypeOf(common.Address{}), // the method's caller
		}

		var purity purity

		switch method.StateMutability {
		case "pure":
			purity = pure
		case "view":
			needs = append(needs, reflect.TypeOf(&state.StateDB{}))
			purity = view
		case "nonpayable":
			needs = append(needs, reflect.TypeOf(&state.StateDB{}))
			purity = write
		case "payable":
			needs = append(needs, reflect.TypeOf(&state.StateDB{}))
			needs = append(needs, reflect.TypeOf(&big.Int{}))
			purity = payable
		default:
			log.Fatal("Unknown state mutability ", method.StateMutability)
		}

		for _, arg := range method.Inputs {
			needs = append(needs, arg.Type.GetType())
		}

		signature := handler.Type

		if signature.NumIn() != len(needs) {
			log.Fatal(context, "doesn't have the args\n\t", needs)
		}
		for i, arg := range needs {
			if signature.In(i) != arg {
				log.Fatal(
					context, "doesn't have the args\n\t", needs, "\n",
					"\tArg ", i, " is ", signature.In(i), " instead of ", arg,
				)
			}
		}

		var outputs = []reflect.Type{}
		for _, out := range method.Outputs {
			outputs = append(outputs, out.Type.GetType())
		}
		outputs = append(outputs, reflect.TypeOf((*error)(nil)).Elem())

		if signature.NumOut() != len(outputs) {
			log.Fatal("Precompile ", contract, "'s ", name, " implementer doesn't return ", outputs)
		}
		for i, out := range outputs {
			if signature.Out(i) != out {
				log.Fatal(
					context, "doesn't have the outputs\n\t", outputs, "\n",
					"\tReturn value ", i+1, " is ", signature.Out(i), " instead of ", out,
				)
			}
		}

		// ensure we have a matching gascost func

		gascost, ok := reflect.TypeOf(implementer).MethodByName(name + "GasCost")
		if !ok {
			log.Fatal("Precompile ", contract, " must implement ", name+"GasCost")
		}

		needs = []reflect.Type{
			reflect.TypeOf(implementer), // the contract itself
		}
		for _, arg := range method.Inputs {
			needs = append(needs, arg.Type.GetType())
		}

		signature = gascost.Type
		context = "Precompile " + contract + "'s " + name + "GasCost's implementer "

		if signature.NumIn() != len(needs) {
			log.Fatal(context, "doesn't have the args\n\t", needs)
		}
		for i, arg := range needs {
			if signature.In(i) != arg {
				log.Fatal(
					context, "doesn't have the args\n\t", needs, "\n",
					"\tArg ", i, " is ", signature.In(i), " instead of ", arg,
				)
			}
		}
		if signature.NumOut() != 1 || signature.Out(0) != reflect.TypeOf(uint64(0)) {
			log.Fatal(context, "must return a uint64")
		}

		methods[id] = PrecompileMethod{
			name,
			method,
			purity,
			handler,
			gascost,
			reflect.ValueOf(implementer),
		}
	}

	return Precompile{
		methods,
	}
}

func addr(s string) common.Address {
	return common.HexToAddress(s)
}

func Precompiles() map[common.Address]ArbosPrecompile {
	return map[common.Address]ArbosPrecompile{
		addr("0x64"): makePrecompile(templates.ArbSysMetaData, pre.ArbSys{}),
		addr("0x65"): makePrecompile(templates.ArbInfoMetaData, pre.ArbInfo{}),
		addr("0x66"): makePrecompile(templates.ArbAddressTableMetaData, pre.ArbAddressTable{}),
		addr("0x67"): makePrecompile(templates.ArbBLSMetaData, pre.ArbBLS{}),
		addr("0x68"): makePrecompile(templates.ArbFunctionTableMetaData, pre.ArbFunctionTable{}),
		addr("0x69"): makePrecompile(templates.ArbosTestMetaData, pre.ArbosTest{}),
		addr("0x6b"): makePrecompile(templates.ArbOwnerMetaData, pre.ArbOwner{}),
		addr("0x6c"): makePrecompile(templates.ArbGasInfoMetaData, pre.ArbGasInfo{}),
		addr("0x6d"): makePrecompile(templates.ArbAggregatorMetaData, pre.ArbAggregator{}),
		addr("0x6e"): makePrecompile(templates.ArbRetryableTxMetaData, pre.ArbRetryableTx{}),
		addr("0x6f"): makePrecompile(templates.ArbStatisticsMetaData, pre.ArbStatistics{}),
	}
}

// determine the amount of gas to charge for calling a precompile
func (p Precompile) GasToCharge(input []byte) uint64 {

	if len(input) < 4 {
		// ArbOS precompiles always have canonical method selectors
		return 0
	}
	id := *(*[4]byte)(input)
	method, ok := p.methods[id]
	if !ok {
		// method does not exist
		return 0
	}

	args, err := method.template.Inputs.Unpack(input[4:])
	if err != nil {
		// calldata does not match the method's signature
		return 0
	}

	reflectArgs := []reflect.Value{
		method.implementer,
	}
	for _, arg := range args {
		reflectArgs = append(reflectArgs, reflect.ValueOf(arg))
	}

	// we checked earlier that gascost() returns a uint64
	return method.gascost.Func.Call(reflectArgs)[0].Interface().(uint64)
}

// call a precompile in typed form, deserializing its inputs and serializing its outputs
func (p Precompile) Call(
	input []byte,
	precompileAddress common.Address,
	actingAsAddress common.Address,
	caller common.Address,
	value *big.Int,
	readOnly bool,
	evm *vm.EVM,
) (output []byte, err error) {

	if len(input) < 4 {
		// ArbOS precompiles always have canonical method selectors
		return nil, vm.ErrExecutionReverted
	}
	id := *(*[4]byte)(input)
	method, ok := p.methods[id]
	if !ok {
		// method does not exist
		return nil, vm.ErrExecutionReverted
	}

	if method.purity >= view && actingAsAddress != precompileAddress {
		// should not access precompile superpowers when not acting as the precompile
		return nil, vm.ErrExecutionReverted
	}

	if method.purity >= write && readOnly {
		// tried to write to global state in read-only mode
		return nil, vm.ErrExecutionReverted
	}

	if method.purity < payable && value.Sign() != 0 {
		// tried to pay something that's non-payable
		return nil, vm.ErrExecutionReverted
	}

	reflectArgs := []reflect.Value{
		method.implementer,
		reflect.ValueOf(caller),
	}

	state := evm.StateDB.(*state.StateDB)

	switch method.purity {
	case pure:
	case view:
		reflectArgs = append(reflectArgs, reflect.ValueOf(state))
	case write:
		reflectArgs = append(reflectArgs, reflect.ValueOf(state))
	case payable:
		reflectArgs = append(reflectArgs, reflect.ValueOf(state))
		reflectArgs = append(reflectArgs, reflect.ValueOf(value))
	default:
		log.Fatal("Unknown state mutability ", method.purity)
	}

	args, err := method.template.Inputs.Unpack(input[4:])
	if err != nil {
		// calldata does not match the method's signature
		return nil, vm.ErrExecutionReverted
	}
	for _, arg := range args {
		reflectArgs = append(reflectArgs, reflect.ValueOf(arg))
	}

	reflectResult := method.handler.Func.Call(reflectArgs)
	resultCount := len(reflectResult) - 1
	if !reflectResult[resultCount].IsNil() {
		// the last arg is always the error status
		return nil, vm.ErrExecutionReverted
	}
	result := make([]interface{}, resultCount)
	for i := 0; i < resultCount; i++ {
		result[i] = reflectResult[i].Interface()
	}

	encoded, err := method.template.Outputs.PackValues(result)
	if err != nil {
		// in production we'll just revert, but for now this
		// will catch implementation errors
		log.Fatal("Could not encode precompile result ", err)
	}
	return encoded, nil
}
