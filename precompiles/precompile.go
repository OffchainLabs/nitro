//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	templates "github.com/offchainlabs/arbstate/solgen/go/precompilesgen"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	glog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type ArbosPrecompile interface {
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
		gasSupplied uint64,
		evm *vm.EVM,
	) (output []byte, gasLeft uint64, err error)

	Precompile() Precompile
}

type purity uint8

const (
	pure purity = iota
	view
	write
	payable
)

type Precompile struct {
	methods     map[[4]byte]PrecompileMethod
	events      map[string]PrecompileEvent
	implementer reflect.Value
	address     common.Address
}

type PrecompileMethod struct {
	name        string
	template    abi.Method
	purity      purity
	handler     reflect.Method
	implementer reflect.Value
}

type PrecompileEvent struct {
	name     string
	template abi.Event
}

// Make a precompile for the given hardhat-to-geth bindings, ensuring that the implementer
// supports each method.
func makePrecompile(metadata *bind.MetaData, implementer interface{}) (addr, ArbosPrecompile) {
	source, err := abi.JSON(strings.NewReader(metadata.ABI))
	if err != nil {
		log.Fatal("Bad ABI")
	}

	implementerType := reflect.TypeOf(implementer)
	contract := implementerType.Elem().Name()

	_, ok := implementerType.Elem().FieldByName("Address")
	if !ok {
		log.Fatal("Implementer for precompile ", contract, " is missing an Address field")
	}

	address, ok := reflect.ValueOf(implementer).Elem().FieldByName("Address").Interface().(addr)
	if !ok {
		log.Fatal("Implementer for precompile ", contract, "'s Address field has the wrong type")
	}

	methods := make(map[[4]byte]PrecompileMethod)
	events := make(map[string]PrecompileEvent)

	for _, method := range source.Methods {

		name := method.RawName
		capitalize := string(unicode.ToUpper(rune(name[0])))
		name = capitalize + name[1:]

		if len(method.ID) != 4 {
			log.Fatal("Method ID isn't 4 bytes")
		}
		id := *(*[4]byte)(method.ID)

		// check that the implementer has a supporting implementation for this method

		handler, ok := implementerType.MethodByName(name)
		if !ok {
			log.Fatal("Precompile ", contract, " must implement ", name)
		}

		var needs = []reflect.Type{
			implementerType,            // the contract itself
			reflect.TypeOf((ctx)(nil)), // this call's context
		}

		var purity purity

		switch method.StateMutability {
		case "pure":
			purity = pure
		case "view":
			needs = append(needs, reflect.TypeOf(&vm.EVM{}))
			purity = view
		case "nonpayable":
			needs = append(needs, reflect.TypeOf(&vm.EVM{}))
			purity = write
		case "payable":
			needs = append(needs, reflect.TypeOf(&vm.EVM{}))
			needs = append(needs, reflect.TypeOf(&big.Int{}))
			purity = payable
		default:
			log.Fatal("Unknown state mutability ", method.StateMutability)
		}

		for _, arg := range method.Inputs {
			needs = append(needs, arg.Type.GetType())
		}

		var outputs = []reflect.Type{}
		for _, out := range method.Outputs {
			outputs = append(outputs, out.Type.GetType())
		}
		outputs = append(outputs, reflect.TypeOf((*error)(nil)).Elem())

		expectedHandlerType := reflect.FuncOf(needs, outputs, false)

		if handler.Type != expectedHandlerType {
			log.Fatal(
				"Precompile "+contract+"'s "+name+"'s implementer has the wrong type\n",
				"\texpected:\t", expectedHandlerType, "\n\tbut have:\t", handler.Type,
			)
		}

		methods[id] = PrecompileMethod{
			name,
			method,
			purity,
			handler,
			reflect.ValueOf(implementer),
		}
	}

	// provide the implementer mechanisms to emit logs for the solidity events

	supportedIndices := map[string]struct{}{
		// the solidity value types: https://docs.soliditylang.org/en/v0.8.9/types.html
		"address": {},
		"bytes32": {},
		"bool":    {},
	}
	for i := 8; i <= 256; i += 8 {
		supportedIndices["int"+strconv.Itoa(i)] = struct{}{}
		supportedIndices["uint"+strconv.Itoa(i)] = struct{}{}
	}

	for _, event := range source.Events {
		name := event.RawName

		var needs = []reflect.Type{
			reflect.TypeOf(&context{}), // where the emit goes
			reflect.TypeOf(&vm.EVM{}),  // where the emit goes
		}
		for _, arg := range event.Inputs {
			needs = append(needs, arg.Type.GetType())

			if arg.Indexed {
				_, ok := supportedIndices[arg.Type.String()]
				if !ok {
					log.Fatal(
						"Please change the solidity for precompile ", contract,
						"'s event ", name, ":\n\tEvent indices of type ",
						arg.Type.String(), " are not supported",
					)
				}
			}
		}

		uint64Type := reflect.TypeOf(uint64(0))
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		expectedFieldType := reflect.FuncOf(needs, []reflect.Type{errorType}, false)
		expectedCostType := reflect.FuncOf(needs[2:], []reflect.Type{uint64Type, errorType}, false)

		context := "Precompile " + contract + "'s implementer"
		missing := context + " is missing a field for "

		field, ok := implementerType.Elem().FieldByName(name)
		if !ok {
			log.Fatal(missing, "event ", name, " of type\n\t", expectedFieldType)
		}
		costField, ok := implementerType.Elem().FieldByName(name + "GasCost")
		if !ok {
			log.Fatal(missing, "event ", name, "'s GasCost of type\n\t", expectedCostType)
		}
		if field.Type != expectedFieldType {
			log.Fatal(
				context, "'s field for event ", name, " has the wrong type\n",
				"\texpected:\t", expectedFieldType, "\n\tbut have:\t", field.Type,
			)
		}
		if costField.Type != expectedCostType {
			log.Fatal(
				context, "'s field for event ", name, "GasCost has the wrong type\n",
				"\texpected:\t", expectedCostType, "\n\tbut have:\t", costField.Type,
			)
		}

		structFields := reflect.ValueOf(implementer).Elem()
		fieldPointer := structFields.FieldByName(name)
		costPointer := structFields.FieldByName(name + "GasCost")

		dataInputs := make(abi.Arguments, 0)
		topicInputs := make(abi.Arguments, 0)

		for _, input := range event.Inputs {
			if input.Indexed {
				topicInputs = append(topicInputs, input)
			} else {
				dataInputs = append(dataInputs, input)
			}
		}

		// we can't capture `event` since the for loop will change its value
		capturedEvent := event
		nilError := reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())

		gascost := func(args []reflect.Value) []reflect.Value {

			cost := params.LogGas
			cost += params.LogTopicGas * uint64(1+len(topicInputs))

			var dataValues []interface{}

			for i := 0; i < len(args); i++ {
				if !capturedEvent.Inputs[i].Indexed {
					dataValues = append(dataValues, args[i].Interface())
				}
			}

			data, err := dataInputs.PackValues(dataValues)
			if err != nil {
				glog.Error(fmt.Sprintf(
					"Could not pack values for event %s's GasCost\nerror %s", name, err,
				))
				return []reflect.Value{reflect.ValueOf(0), reflect.ValueOf(err)}
			}

			// charge for the number of bytes
			cost += params.LogDataGas * uint64(len(data))
			return []reflect.Value{reflect.ValueOf(cost), nilError}
		}

		emit := func(args []reflect.Value) []reflect.Value {

			callerCtx := args[0].Interface().(ctx) //nolint:errcheck
			evm := args[1].Interface().(*vm.EVM)   //nolint:errcheck
			state := evm.StateDB
			args = args[2:]

			emitCost := gascost(args)
			cost := emitCost[0].Interface().(uint64) //nolint:errcheck
			if !emitCost[1].IsNil() {
				// an error occured during gascost()
				return []reflect.Value{emitCost[1]}
			}
			if err := callerCtx.Burn(cost); err != nil {
				// the user has run out of gas
				return []reflect.Value{reflect.ValueOf(vm.ErrOutOfGas)}
			}

			// Filter by index'd into data and topics. Indexed values, even if ultimately hashed,
			// aren't supposed to have their contents stored in the general-purpose data portion.
			var dataValues []interface{}
			var topicValues []interface{}

			for i := 0; i < len(args); i++ {
				if capturedEvent.Inputs[i].Indexed {
					topicValues = append(topicValues, args[i].Interface())
				} else {
					dataValues = append(dataValues, args[i].Interface())
				}
			}

			data, err := dataInputs.PackValues(dataValues)
			if err != nil {
				glog.Error(fmt.Sprintf(
					"Couldn't pack values for event %s\nnargs %s\nvalues %s\ntopics %s\nerror %s",
					name, args, dataValues, topicValues, err,
				))
				return []reflect.Value{reflect.ValueOf(err)}
			}

			topics := []common.Hash{capturedEvent.ID}

			for i, input := range topicInputs {
				// Geth provides infrastructure for packing arrays of values,
				// so we create an array with just the value we want to pack.

				packable := []interface{}{topicValues[i]}
				bytes, err := abi.Arguments{input}.PackValues(packable)
				if err != nil {
					glog.Error(fmt.Sprintf(
						"Packing error for event %s\nargs %s\nvalues %s\ntopics %s\nerror %s",
						name, args, dataValues, topicValues, err,
					))
					return []reflect.Value{reflect.ValueOf(err)}
				}

				var topic [32]byte

				if len(bytes) > 32 {
					topic = *(*[32]byte)(crypto.Keccak256(bytes))
				} else {
					offset := 32 - len(bytes)
					copy(topic[offset:], bytes)
				}

				topics = append(topics, topic)
			}

			event := &types.Log{
				Address:     address,
				Topics:      topics,
				Data:        data,
				BlockNumber: evm.Context.BlockNumber.Uint64(),
				// Geth will set all other fields, which include
				//   TxHash, TxIndex, Index, and Removed
			}

			state.AddLog(event)
			return []reflect.Value{nilError}
		}

		fieldPointer.Set(reflect.MakeFunc(field.Type, emit))
		costPointer.Set(reflect.MakeFunc(costField.Type, gascost))

		events[name] = PrecompileEvent{
			name,
			event,
		}
	}

	return address, Precompile{
		methods,
		events,
		reflect.ValueOf(implementer),
		address,
	}
}

func Precompiles() map[addr]ArbosPrecompile {

	//nolint:gocritic
	hex := func(s string) addr {
		return common.HexToAddress(s)
	}

	contracts := make(map[addr]ArbosPrecompile)

	insert := func(address addr, impl ArbosPrecompile) Precompile {
		contracts[address] = impl
		return impl.Precompile()
	}

	insert(makePrecompile(templates.ArbInfoMetaData, &ArbInfo{Address: hex("65")}))
	insert(makePrecompile(templates.ArbAddressTableMetaData, &ArbAddressTable{Address: hex("66")}))
	insert(makePrecompile(templates.ArbBLSMetaData, &ArbBLS{Address: hex("67")}))
	insert(makePrecompile(templates.ArbFunctionTableMetaData, &ArbFunctionTable{Address: hex("68")}))
	insert(makePrecompile(templates.ArbosTestMetaData, &ArbosTest{Address: hex("69")}))
	insert(makePrecompile(templates.ArbOwnerPublicMetaData, &ArbOwnerPublic{Address: hex("6b")}))
	insert(makePrecompile(templates.ArbGasInfoMetaData, &ArbGasInfo{Address: hex("6c")}))
	insert(makePrecompile(templates.ArbAggregatorMetaData, &ArbAggregator{Address: hex("6d")}))
	insert(makePrecompile(templates.ArbStatisticsMetaData, &ArbStatistics{Address: hex("6f")}))

	insert(ownerOnly(makePrecompile(templates.ArbOwnerMetaData, &ArbOwner{Address: hex("70")})))
	insert(debugOnly(makePrecompile(templates.ArbDebugMetaData, &ArbDebug{Address: hex("ff")})))

	ArbRetryable := insert(makePrecompile(templates.ArbRetryableTxMetaData, &ArbRetryableTx{Address: hex("6e")}))
	arbos.ArbRetryableTxAddress = ArbRetryable.address
	arbos.RedeemScheduledEventID = ArbRetryable.events["RedeemScheduled"].template.ID

	ArbSys := insert(makePrecompile(templates.ArbSysMetaData, &ArbSys{Address: hex("64")}))
	arbos.ArbSysAddress = ArbSys.address
	arbos.L2ToL1TransactionEventID = ArbSys.events["L2ToL1Transaction"].template.ID

	return contracts
}

// call a precompile in typed form, deserializing its inputs and serializing its outputs
func (p Precompile) Call(
	input []byte,
	precompileAddress common.Address,
	actingAsAddress common.Address,
	caller common.Address,
	value *big.Int,
	readOnly bool,
	gasSupplied uint64,
	evm *vm.EVM,
) (output []byte, gasLeft uint64, err error) {

	if len(input) < 4 {
		// ArbOS precompiles always have canonical method selectors
		return nil, 0, vm.ErrExecutionReverted
	}
	id := *(*[4]byte)(input)
	method, ok := p.methods[id]
	if !ok {
		// method does not exist
		return nil, 0, vm.ErrExecutionReverted
	}

	if method.purity >= view && actingAsAddress != precompileAddress {
		// should not access precompile superpowers when not acting as the precompile
		return nil, 0, vm.ErrExecutionReverted
	}

	if method.purity >= write && readOnly {
		// tried to write to global state in read-only mode
		return nil, 0, vm.ErrExecutionReverted
	}

	if method.purity < payable && value.Sign() != 0 {
		// tried to pay something that's non-payable
		return nil, 0, vm.ErrExecutionReverted
	}

	callerCtx := &context{
		caller:      caller,
		gasSupplied: gasSupplied,
		gasLeft:     gasSupplied,
	}

	argsCost := params.CopyGas * uint64(len(input)-4)
	if err := callerCtx.Burn(argsCost); err != nil {
		// user cannot afford the argument data supplied
		return nil, 0, vm.ErrExecutionReverted
	}

	if method.purity != pure {
		// impure methods may need the ArbOS state, so open & update the call context now
		state, err := arbosState.OpenArbosState(evm.StateDB, callerCtx)
		if err != nil {
			return nil, 0, err
		}
		callerCtx.state = state
	}

	switch txProcessor := evm.ProcessingHook.(type) {
	case *arbos.TxProcessor:
		callerCtx.txProcessor = txProcessor
	case *vm.DefaultTxProcessor:
		glog.Error("processing hook not set")
		return nil, 0, vm.ErrExecutionReverted
	default:
		glog.Error("unknown processing hook")
		return nil, 0, vm.ErrExecutionReverted
	}

	reflectArgs := []reflect.Value{
		method.implementer,
		reflect.ValueOf(callerCtx),
	}

	switch method.purity {
	case pure:
	case view:
		reflectArgs = append(reflectArgs, reflect.ValueOf(evm))
	case write:
		reflectArgs = append(reflectArgs, reflect.ValueOf(evm))
	case payable:
		reflectArgs = append(reflectArgs, reflect.ValueOf(evm))
		reflectArgs = append(reflectArgs, reflect.ValueOf(value))
	default:
		log.Fatal("Unknown state mutability ", method.purity)
	}

	args, err := method.template.Inputs.Unpack(input[4:])
	if err != nil {
		// calldata does not match the method's signature
		return nil, 0, vm.ErrExecutionReverted
	}
	for _, arg := range args {
		reflectArgs = append(reflectArgs, reflect.ValueOf(arg))
	}

	reflectResult := method.handler.Func.Call(reflectArgs)
	resultCount := len(reflectResult) - 1
	if !reflectResult[resultCount].IsNil() {
		// the last arg is always the error status
		return nil, 0, reflectResult[resultCount].Interface().(error)
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

	resultCost := params.CopyGas * uint64(len(encoded))
	if err := callerCtx.Burn(resultCost); err != nil {
		// user cannot afford the result data returned
		return nil, 0, vm.ErrExecutionReverted
	}

	return encoded, callerCtx.gasLeft, nil
}

func (p Precompile) Precompile() Precompile {
	return p
}
