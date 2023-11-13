// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package precompiles

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/util"
	templates "github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
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

	Precompile() *Precompile
}

type purity uint8

const (
	pure purity = iota
	view
	write
	payable
)

type Precompile struct {
	methods       map[[4]byte]*PrecompileMethod
	methodsByName map[string]*PrecompileMethod
	events        map[string]PrecompileEvent
	errors        map[string]PrecompileError
	name          string
	implementer   reflect.Value
	address       common.Address
	arbosVersion  uint64
}

type PrecompileMethod struct {
	name         string
	template     abi.Method
	purity       purity
	handler      reflect.Method
	arbosVersion uint64
}

type PrecompileEvent struct {
	name     string
	template abi.Event
}

type PrecompileError struct {
	name     string
	template abi.Error
}

type SolError struct {
	data   []byte
	solErr abi.Error
}

func RenderSolError(solErr abi.Error, data []byte) (string, error) {
	vals, err := solErr.Unpack(data)
	if err != nil {
		return "", err
	}
	valsRange, ok := vals.([]interface{})
	if !ok {
		return "", errors.New("unexpected unpack result")
	}
	strVals := make([]string, 0, len(valsRange))
	for _, val := range valsRange {
		strVals = append(strVals, fmt.Sprintf("%v", val))
	}
	return fmt.Sprintf("error %v(%v)", solErr.Name, strings.Join(strVals, ", ")), nil
}

func (e *SolError) Error() string {
	rendered, err := RenderSolError(e.solErr, e.data)
	if err != nil {
		return "unable to decode execution error"
	}
	return rendered
}

// MakePrecompile makes a precompile for the given hardhat-to-geth bindings, ensuring that the implementer
// supports each method.
func MakePrecompile(metadata *bind.MetaData, implementer interface{}) (addr, *Precompile) {
	source, err := abi.JSON(strings.NewReader(metadata.ABI))
	if err != nil {
		log.Crit("Bad ABI")
	}

	implementerType := reflect.TypeOf(implementer)
	contract := implementerType.Elem().Name()

	_, ok := implementerType.Elem().FieldByName("Address")
	if !ok {
		log.Crit("Implementer for precompile ", contract, " is missing an Address field")
	}

	address, ok := reflect.ValueOf(implementer).Elem().FieldByName("Address").Interface().(addr)
	if !ok {
		log.Crit("Implementer for precompile ", contract, "'s Address field has the wrong type")
	}

	gethAbiFuncTypeEquality := func(actual, geth reflect.Type) bool {
		gethIn := geth.NumIn()
		gethOut := geth.NumOut()
		if actual.NumIn() != gethIn || actual.NumOut() != gethOut {
			return false
		}
		for i := 0; i < gethIn; i++ {
			if !geth.In(i).ConvertibleTo(actual.In(i)) {
				return false
			}
		}
		for i := 0; i < gethOut; i++ {
			if !actual.Out(i).ConvertibleTo(geth.Out(i)) {
				return false
			}
		}
		return true
	}

	methods := make(map[[4]byte]*PrecompileMethod)
	methodsByName := make(map[string]*PrecompileMethod)
	events := make(map[string]PrecompileEvent)
	errors := make(map[string]PrecompileError)

	for _, method := range source.Methods {

		name := method.RawName
		capitalize := string(unicode.ToUpper(rune(name[0])))
		name = capitalize + name[1:]

		if len(method.ID) != 4 {
			log.Crit("Method ID isn't 4 bytes")
		}
		id := *(*[4]byte)(method.ID)

		// check that the implementer has a supporting implementation for this method

		handler, ok := implementerType.MethodByName(name)
		if !ok {
			log.Crit("Precompile " + contract + " must implement " + name)
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
			log.Crit("Unknown state mutability ", method.StateMutability)
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

		if !gethAbiFuncTypeEquality(handler.Type, expectedHandlerType) {
			log.Crit(
				"Precompile "+contract+"'s "+name+"'s implementer has the wrong type\n",
				"\texpected:\t", expectedHandlerType, "\n\tbut have:\t", handler.Type,
			)
		}

		method := PrecompileMethod{
			name,
			method,
			purity,
			handler,
			0,
		}
		methods[id] = &method
		methodsByName[name] = &method
	}

	for i := 0; i < implementerType.NumMethod(); i++ {
		method := implementerType.Method(i)
		name := method.Name
		if method.IsExported() && methodsByName[name] == nil {
			log.Crit(contract + " is missing a solidity interface for " + name)
		}
	}

	// provide the implementer mechanisms to emit logs for the solidity events

	supportedIndices := map[string]struct{}{
		// the solidity value types: https://docs.soliditylang.org/en/v0.8.9/types.html
		"address": {},
		"bool":    {},
	}
	for i := 8; i <= 256; i += 8 {
		supportedIndices["int"+strconv.Itoa(i)] = struct{}{}
		supportedIndices["uint"+strconv.Itoa(i)] = struct{}{}
	}
	for i := 1; i <= 32; i += 1 {
		supportedIndices["bytes"+strconv.Itoa(i)] = struct{}{}
	}

	for _, event := range source.Events {
		name := event.RawName

		var needs = []reflect.Type{
			reflect.TypeOf(&Context{}), // where the emit goes
			reflect.TypeOf(&vm.EVM{}),  // where the emit goes
		}
		for _, arg := range event.Inputs {
			needs = append(needs, arg.Type.GetType())

			if arg.Indexed {
				_, ok := supportedIndices[arg.Type.String()]
				if !ok {
					log.Crit(
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
			log.Crit(missing, "event ", name, " of type\n\t", expectedFieldType)
		}
		costField, ok := implementerType.Elem().FieldByName(name + "GasCost")
		if !ok {
			log.Crit(missing, "event ", name, "'s GasCost of type\n\t", expectedCostType)
		}
		if !gethAbiFuncTypeEquality(field.Type, expectedFieldType) {
			log.Crit(
				context, "'s field for event ", name, " has the wrong type\n",
				"\texpected:\t", expectedFieldType, "\n\tbut have:\t", field.Type,
			)
		}
		if !gethAbiFuncTypeEquality(costField.Type, expectedCostType) {
			log.Crit(
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

			version := arbosState.ArbOSVersion(state)
			if callerCtx.readOnly && version >= 11 {
				return []reflect.Value{reflect.ValueOf(vm.ErrWriteProtection)}
			}

			emitCost := gascost(args)
			cost := emitCost[0].Interface().(uint64) //nolint:errcheck
			if !emitCost[1].IsNil() {
				// an error occurred during gascost()
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

	for _, solErr := range source.Errors {
		name := solErr.Name

		var needs []reflect.Type
		for _, arg := range solErr.Inputs {
			needs = append(needs, arg.Type.GetType())
		}

		errorType := reflect.TypeOf((*error)(nil)).Elem()
		expectedFieldType := reflect.FuncOf(needs, []reflect.Type{errorType}, false)

		context := "Precompile " + contract + "'s implementer"
		missing := context + " is missing a field for "

		field, ok := implementerType.Elem().FieldByName(name + "Error")
		if !ok {
			log.Crit(missing, "custom error ", name, "Error of type\n\t", expectedFieldType)
		}
		if field.Type != expectedFieldType {
			log.Crit(
				context, "'s field for error ", name, "Error has the wrong type\n",
				"\texpected:\t", expectedFieldType, "\n\tbut have:\t", field.Type,
			)
		}

		structFields := reflect.ValueOf(implementer).Elem()
		errorReturnPointer := structFields.FieldByName(name + "Error")

		capturedSolErr := solErr
		errorReturn := func(args []reflect.Value) []reflect.Value {
			var dataValues []interface{}
			for i := 0; i < len(args); i++ {
				dataValues = append(dataValues, args[i].Interface())
			}

			data, err := capturedSolErr.Inputs.PackValues(dataValues)
			if err != nil {
				glog.Error(fmt.Sprintf(
					"Couldn't pack values for error %s\nnargs %s\nvalues %s\nerror %s",
					name, args, dataValues, err,
				))
				return []reflect.Value{reflect.ValueOf(err)}
			}

			customErr := &SolError{data: append(capturedSolErr.ID[:4], data...), solErr: capturedSolErr}

			return []reflect.Value{reflect.ValueOf(customErr)}
		}

		errorReturnPointer.Set(reflect.MakeFunc(field.Type, errorReturn))

		errors[name] = PrecompileError{
			name,
			solErr,
		}
	}

	return address, &Precompile{
		methods,
		methodsByName,
		events,
		errors,
		contract,
		reflect.ValueOf(implementer),
		address,
		0,
	}
}

func Precompiles() map[addr]ArbosPrecompile {

	//nolint:gocritic
	hex := func(s string) addr {
		return common.HexToAddress(s)
	}

	contracts := make(map[addr]ArbosPrecompile)

	insert := func(address addr, impl ArbosPrecompile) *Precompile {
		contracts[address] = impl
		return impl.Precompile()
	}

	insert(MakePrecompile(templates.ArbInfoMetaData, &ArbInfo{Address: hex("65")}))
	insert(MakePrecompile(templates.ArbAddressTableMetaData, &ArbAddressTable{Address: hex("66")}))
	insert(MakePrecompile(templates.ArbBLSMetaData, &ArbBLS{Address: hex("67")}))
	insert(MakePrecompile(templates.ArbFunctionTableMetaData, &ArbFunctionTable{Address: hex("68")}))
	insert(MakePrecompile(templates.ArbosTestMetaData, &ArbosTest{Address: hex("69")}))
	ArbGasInfo := insert(MakePrecompile(templates.ArbGasInfoMetaData, &ArbGasInfo{Address: hex("6c")}))
	ArbGasInfo.methodsByName["GetL1FeesAvailable"].arbosVersion = 10
	ArbGasInfo.methodsByName["GetL1RewardRate"].arbosVersion = 11
	ArbGasInfo.methodsByName["GetL1RewardRecipient"].arbosVersion = 11
	insert(MakePrecompile(templates.ArbAggregatorMetaData, &ArbAggregator{Address: hex("6d")}))
	insert(MakePrecompile(templates.ArbStatisticsMetaData, &ArbStatistics{Address: hex("6f")}))

	eventCtx := func(gasLimit uint64, err error) *Context {
		if err != nil {
			glog.Error("call to event's GasCost field failed", "err", err)
		}
		return &Context{
			gasSupplied: gasLimit,
			gasLeft:     gasLimit,
		}
	}

	ArbOwnerPublic := insert(MakePrecompile(templates.ArbOwnerPublicMetaData, &ArbOwnerPublic{Address: hex("6b")}))
	ArbOwnerPublic.methodsByName["GetInfraFeeAccount"].arbosVersion = 5
	ArbOwnerPublic.methodsByName["RectifyChainOwner"].arbosVersion = 11
	ArbOwnerPublic.methodsByName["GetBrotliCompressionLevel"].arbosVersion = 12

	ArbRetryableImpl := &ArbRetryableTx{Address: types.ArbRetryableTxAddress}
	ArbRetryable := insert(MakePrecompile(templates.ArbRetryableTxMetaData, ArbRetryableImpl))
	arbos.ArbRetryableTxAddress = ArbRetryable.address
	arbos.RedeemScheduledEventID = ArbRetryable.events["RedeemScheduled"].template.ID
	arbos.EmitReedeemScheduledEvent = func(
		evm mech, gas, nonce uint64, ticketId, retryTxHash bytes32,
		donor addr, maxRefund *big.Int, submissionFeeRefund *big.Int,
	) error {
		zero := common.Big0
		context := eventCtx(ArbRetryableImpl.RedeemScheduledGasCost(hash{}, hash{}, 0, 0, addr{}, zero, zero))
		return ArbRetryableImpl.RedeemScheduled(
			context, evm, ticketId, retryTxHash, nonce, gas, donor, maxRefund, submissionFeeRefund,
		)
	}
	arbos.EmitTicketCreatedEvent = func(evm mech, ticketId bytes32) error {
		context := eventCtx(ArbRetryableImpl.TicketCreatedGasCost(hash{}))
		return ArbRetryableImpl.TicketCreated(context, evm, ticketId)
	}

	ArbSys := insert(MakePrecompile(templates.ArbSysMetaData, &ArbSys{Address: types.ArbSysAddress}))
	arbos.ArbSysAddress = ArbSys.address
	arbos.L2ToL1TransactionEventID = ArbSys.events["L2ToL1Transaction"].template.ID
	arbos.L2ToL1TxEventID = ArbSys.events["L2ToL1Tx"].template.ID

	ArbOwnerImpl := &ArbOwner{Address: hex("70")}
	emitOwnerActs := func(evm mech, method bytes4, owner addr, data []byte) error {
		context := eventCtx(ArbOwnerImpl.OwnerActsGasCost(method, owner, data))
		return ArbOwnerImpl.OwnerActs(context, evm, method, owner, data)
	}
	_, ArbOwner := MakePrecompile(templates.ArbOwnerMetaData, ArbOwnerImpl)
	ArbOwner.methodsByName["GetInfraFeeAccount"].arbosVersion = 5
	ArbOwner.methodsByName["SetInfraFeeAccount"].arbosVersion = 5
	ArbOwner.methodsByName["ReleaseL1PricerSurplusFunds"].arbosVersion = 10
	ArbOwner.methodsByName["SetChainConfig"].arbosVersion = 11
	ArbOwner.methodsByName["SetBrotliCompressionLevel"].arbosVersion = 12

	insert(ownerOnly(ArbOwnerImpl.Address, ArbOwner, emitOwnerActs))
	insert(debugOnly(MakePrecompile(templates.ArbDebugMetaData, &ArbDebug{Address: hex("ff")})))

	ArbosActs := insert(MakePrecompile(templates.ArbosActsMetaData, &ArbosActs{Address: types.ArbosAddress}))
	arbos.InternalTxStartBlockMethodID = ArbosActs.GetMethodID("StartBlock")
	arbos.InternalTxBatchPostingReportMethodID = ArbosActs.GetMethodID("BatchPostingReport")

	return contracts
}

func (p *Precompile) CloneWithImpl(impl interface{}) *Precompile {
	clone := *p
	clone.implementer = reflect.ValueOf(impl)
	return &clone
}

func (p *Precompile) GetMethodID(name string) bytes4 {
	method, ok := p.methodsByName[name]
	if !ok {
		panic(fmt.Sprintf("Precompile %v does not have a method with the name %v", p.name, name))
	}
	return *(*bytes4)(method.template.ID)
}

// Call a precompile in typed form, deserializing its inputs and serializing its outputs
func (p *Precompile) Call(
	input []byte,
	precompileAddress common.Address,
	actingAsAddress common.Address,
	caller common.Address,
	value *big.Int,
	readOnly bool,
	gasSupplied uint64,
	evm *vm.EVM,
) (output []byte, gasLeft uint64, err error) {
	arbosVersion := arbosState.ArbOSVersion(evm.StateDB)

	if arbosVersion < p.arbosVersion {
		// the precompile isn't yet active, so treat this call as if it were to a contract that doesn't exist
		return []byte{}, gasSupplied, nil
	}

	if len(input) < 4 {
		// ArbOS precompiles always have canonical method selectors
		return nil, 0, vm.ErrExecutionReverted
	}
	id := *(*[4]byte)(input)
	method, ok := p.methods[id]
	if !ok || arbosVersion < method.arbosVersion {
		// method does not exist or hasn't yet been activated
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

	callerCtx := &Context{
		caller:      caller,
		gasSupplied: gasSupplied,
		gasLeft:     gasSupplied,
		readOnly:    method.purity <= view,
		tracingInfo: util.NewTracingInfo(evm, caller, precompileAddress, util.TracingDuringEVM),
	}

	argsCost := params.CopyGas * arbmath.WordsForBytes(uint64(len(input)-4))
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
		callerCtx.State = state
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
		p.implementer,
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
		log.Crit("Unknown state mutability ", method.purity)
	}

	args, err := method.template.Inputs.Unpack(input[4:])
	if err != nil {
		// calldata does not match the method's signature
		return nil, 0, vm.ErrExecutionReverted
	}
	for _, arg := range args {
		converted := reflect.ValueOf(arg).Convert(method.handler.Type.In(len(reflectArgs)))
		reflectArgs = append(reflectArgs, converted)
	}

	reflectResult := method.handler.Func.Call(reflectArgs)
	resultCount := len(reflectResult) - 1
	if !reflectResult[resultCount].IsNil() {
		// the last arg is always the error status
		errRet, ok := reflectResult[resultCount].Interface().(error)
		if !ok {
			log.Error("final precompile return value must be error")
			return nil, callerCtx.gasLeft, vm.ErrExecutionReverted
		}
		var solErr *SolError
		isSolErr := errors.As(errRet, &solErr)
		if isSolErr {
			resultCost := params.CopyGas * arbmath.WordsForBytes(uint64(len(solErr.data)))
			if err := callerCtx.Burn(resultCost); err != nil {
				// user cannot afford the result data returned
				return nil, 0, vm.ErrExecutionReverted
			}
			return solErr.data, callerCtx.gasLeft, vm.ErrExecutionReverted
		}
		if !errors.Is(errRet, vm.ErrOutOfGas) {
			log.Debug("precompile reverted with non-solidity error", "precompile", precompileAddress, "input", input, "err", errRet)
		}
		// nolint:errorlint
		if arbosVersion >= 11 || errRet == vm.ErrExecutionReverted {
			return nil, callerCtx.gasLeft, vm.ErrExecutionReverted
		}
		// Preserve behavior with old versions which would zero out gas on this type of error
		return nil, 0, errRet
	}
	result := make([]interface{}, resultCount)
	for i := 0; i < resultCount; i++ {
		result[i] = reflectResult[i].Interface()
	}

	encoded, err := method.template.Outputs.PackValues(result)
	if err != nil {
		log.Error("could not encode precompile result", "err", err)
		return nil, callerCtx.gasLeft, vm.ErrExecutionReverted
	}

	resultCost := params.CopyGas * arbmath.WordsForBytes(uint64(len(encoded)))
	if err := callerCtx.Burn(resultCost); err != nil {
		// user cannot afford the result data returned
		return nil, 0, vm.ErrExecutionReverted
	}

	return encoded, callerCtx.gasLeft, nil
}

func (p *Precompile) Precompile() *Precompile {
	return p
}

// Get4ByteMethodSignatures is needed for the fuzzing harness
func (p *Precompile) Get4ByteMethodSignatures() [][4]byte {
	ret := make([][4]byte, 0, len(p.methods))
	for sig := range p.methods {
		ret = append(ret, sig)
	}
	return ret
}

func (p *Precompile) GetErrorABIs() []abi.Error {
	ret := make([]abi.Error, 0, len(p.errors))
	for _, solErr := range p.errors {
		ret = append(ret, solErr.template)
	}
	return ret
}
