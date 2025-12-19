// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type ArbosPrecompile interface {
	// Important fields: evm.StateDB and evm.Config.Tracer
	// NOTE: if precompileAddress != actingAsAddress, watch out!
	// This is a delegatecall or callcode, so caller might be wrong.
	// In that case, unless this precompile is pure, it should probably revert.
	Call(
		input []byte,
		actingAsAddress common.Address,
		caller common.Address,
		value *big.Int,
		readOnly bool,
		gasSupplied uint64,
		evm *vm.EVM,
	) (output []byte, gasLeft uint64, usedMultiGas multigas.MultiGas, err error)

	Precompile() *Precompile
	Name() string
	Address() common.Address
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
	name            string
	template        abi.Method
	purity          purity
	handler         reflect.Method
	arbosVersion    uint64
	maxArbosVersion uint64
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
	strVals := make([]string, 0, len(vals))
	for _, val := range vals {
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
		panic("Bad ABI")
	}

	implementerType := reflect.TypeOf(implementer)
	contract := implementerType.Elem().Name()

	_, ok := implementerType.Elem().FieldByName("Address")
	if !ok {
		panic("Implementer for precompile " + contract + " is missing an Address field")
	}

	address, ok := reflect.ValueOf(implementer).Elem().FieldByName("Address").Interface().(addr)
	if !ok {
		panic("Implementer for precompile " + contract + "'s Address field has the wrong type")
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
			panic("Method ID isn't 4 bytes")
		}
		id := *(*[4]byte)(method.ID)

		// check that the implementer has a supporting implementation for this method

		handler, ok := implementerType.MethodByName(name)
		if !ok {
			panic("Precompile " + contract + " must implement " + name)
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
			panic("Unknown state mutability " + method.StateMutability)
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
			panic(
				"Precompile " + contract + "'s " + name + "'s implementer has the wrong type\n" +
					"\texpected:\t" + expectedHandlerType.String() + "\n\tbut have:\t" + handler.Type.String())
		}

		method := PrecompileMethod{
			name:            name,
			template:        method,
			purity:          purity,
			handler:         handler,
			arbosVersion:    0,
			maxArbosVersion: 0,
		}
		methods[id] = &method
		methodsByName[name] = &method
	}

	for i := 0; i < implementerType.NumMethod(); i++ {
		method := implementerType.Method(i)
		name := method.Name
		if method.IsExported() && methodsByName[name] == nil {
			panic(contract + " is missing a solidity interface for " + name)
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
					panic(
						"Please change the solidity for precompile " + contract +
							"'s event " + name + ":\n\tEvent indices of type " +
							arg.Type.String() + " are not supported")
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
			panic(missing + "event " + name + " of type\n\t" + expectedFieldType.String())
		}
		costField, ok := implementerType.Elem().FieldByName(name + "GasCost")
		if !ok {
			panic(missing + "event " + name + "'s GasCost of type\n\t" + expectedCostType.String())
		}
		if !gethAbiFuncTypeEquality(field.Type, expectedFieldType) {
			panic(
				context + "'s field for event " + name + " has the wrong type\n" +
					"\texpected:\t" + expectedFieldType.String() + "\n\tbut have:\t" + field.Type.String())
		}
		if !gethAbiFuncTypeEquality(costField.Type, expectedCostType) {
			panic(
				context + "'s field for event " + name + "GasCost has the wrong type\n" +
					"\texpected:\t" + expectedCostType.String() + "\n\tbut have:\t" + costField.Type.String())
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
			// #nosec G115
			cost += params.LogTopicGas * uint64(1+len(topicInputs))

			var dataValues []interface{}

			for i := 0; i < len(args); i++ {
				if !capturedEvent.Inputs[i].Indexed {
					dataValues = append(dataValues, args[i].Interface())
				}
			}

			data, err := dataInputs.PackValues(dataValues)
			if err != nil {
				log.Error(fmt.Sprintf(
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
			if callerCtx.readOnly && version >= params.ArbosVersion_11 {
				return []reflect.Value{reflect.ValueOf(vm.ErrWriteProtection)}
			}

			emitCost := gascost(args)
			cost := emitCost[0].Interface().(uint64) //nolint:errcheck
			if !emitCost[1].IsNil() {
				// an error occurred during gascost()
				return []reflect.Value{emitCost[1]}
			}
			if err := callerCtx.Burn(multigas.ResourceKindHistoryGrowth, cost); err != nil {
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
				log.Error(fmt.Sprintf(
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
					log.Error(fmt.Sprintf(
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
			panic(missing + "custom error " + name + "Error of type\n\t" + expectedFieldType.String())
		}
		if field.Type != expectedFieldType {
			panic(
				context + "'s field for error " + name + "Error has the wrong type\n" +
					"\texpected:\t" + expectedFieldType.String() + "\n\tbut have:\t" + field.Type.String())
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
				log.Error(fmt.Sprintf(
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
		methods:       methods,
		methodsByName: methodsByName,
		events:        events,
		errors:        errors,
		name:          contract,
		implementer:   reflect.ValueOf(implementer),
		address:       address,
		arbosVersion:  0,
	}
}

func Precompiles() map[addr]ArbosPrecompile {
	contracts := make(map[addr]ArbosPrecompile)

	insert := func(address addr, impl ArbosPrecompile) *Precompile {
		contracts[address] = impl
		return impl.Precompile()
	}

	insert(MakePrecompile(precompilesgen.ArbInfoMetaData, &ArbInfo{Address: types.ArbInfoAddress}))
	insert(MakePrecompile(precompilesgen.ArbAddressTableMetaData, &ArbAddressTable{Address: types.ArbAddressTableAddress}))
	insert(MakePrecompile(precompilesgen.ArbBLSMetaData, &ArbBLS{Address: types.ArbBLSAddress}))
	insert(MakePrecompile(precompilesgen.ArbFunctionTableMetaData, &ArbFunctionTable{Address: types.ArbFunctionTableAddress}))
	insert(MakePrecompile(precompilesgen.ArbosTestMetaData, &ArbosTest{Address: types.ArbosTestAddress}))
	ArbGasInfo := insert(MakePrecompile(precompilesgen.ArbGasInfoMetaData, &ArbGasInfo{Address: types.ArbGasInfoAddress}))
	ArbGasInfo.methodsByName["GetL1FeesAvailable"].arbosVersion = params.ArbosVersion_10
	ArbGasInfo.methodsByName["GetL1RewardRate"].arbosVersion = params.ArbosVersion_11
	ArbGasInfo.methodsByName["GetL1RewardRecipient"].arbosVersion = params.ArbosVersion_11
	ArbGasInfo.methodsByName["GetL1PricingEquilibrationUnits"].arbosVersion = params.ArbosVersion_20
	ArbGasInfo.methodsByName["GetLastL1PricingUpdateTime"].arbosVersion = params.ArbosVersion_20
	ArbGasInfo.methodsByName["GetL1PricingFundsDueForRewards"].arbosVersion = params.ArbosVersion_20
	ArbGasInfo.methodsByName["GetL1PricingUnitsSinceUpdate"].arbosVersion = params.ArbosVersion_20
	ArbGasInfo.methodsByName["GetLastL1PricingSurplus"].arbosVersion = params.ArbosVersion_20
	ArbGasInfo.methodsByName["GetMaxTxGasLimit"].arbosVersion = params.ArbosVersion_50
	ArbGasInfo.methodsByName["GetMaxBlockGasLimit"].arbosVersion = params.ArbosVersion_50
	ArbGasInfo.methodsByName["GetGasPricingConstraints"].arbosVersion = params.ArbosVersion_50
	ArbGasInfo.methodsByName["GetMultiGasPricingConstraints"].arbosVersion = params.ArbosVersion_60
	insert(MakePrecompile(precompilesgen.ArbAggregatorMetaData, &ArbAggregator{Address: types.ArbAggregatorAddress}))
	insert(MakePrecompile(precompilesgen.ArbStatisticsMetaData, &ArbStatistics{Address: types.ArbStatisticsAddress}))

	eventCtx := func(gasLimit uint64, err error) *Context {
		if err != nil {
			log.Error("call to event's GasCost field failed", "err", err)
		}
		return &Context{
			gasSupplied: gasLimit,
			gasUsed:     multigas.ZeroGas(),
		}
	}

	ArbOwnerPublicImpl := &ArbOwnerPublic{Address: types.ArbOwnerPublicAddress}
	ArbOwnerPublic := insert(MakePrecompile(precompilesgen.ArbOwnerPublicMetaData, ArbOwnerPublicImpl))
	ArbOwnerPublic.methodsByName["GetInfraFeeAccount"].arbosVersion = params.ArbosVersion_5
	ArbOwnerPublic.methodsByName["RectifyChainOwner"].arbosVersion = params.ArbosVersion_11
	ArbOwnerPublic.methodsByName["GetBrotliCompressionLevel"].arbosVersion = params.ArbosVersion_20
	ArbOwnerPublic.methodsByName["GetScheduledUpgrade"].arbosVersion = params.ArbosVersion_20
	ArbOwnerPublic.methodsByName["IsNativeTokenOwner"].arbosVersion = params.ArbosVersion_41
	ArbOwnerPublic.methodsByName["GetAllNativeTokenOwners"].arbosVersion = params.ArbosVersion_41
	ArbOwnerPublic.methodsByName["GetParentGasFloorPerToken"].arbosVersion = params.ArbosVersion_50

	ArbWasmImpl := &ArbWasm{Address: types.ArbWasmAddress}
	ArbWasm := insert(MakePrecompile(precompilesgen.ArbWasmMetaData, ArbWasmImpl))
	ArbWasm.arbosVersion = params.ArbosVersion_Stylus
	programs.ProgramNotWasmError = ArbWasmImpl.ProgramNotWasmError
	programs.ProgramNotActivatedError = ArbWasmImpl.ProgramNotActivatedError
	programs.ProgramNeedsUpgradeError = ArbWasmImpl.ProgramNeedsUpgradeError
	programs.ProgramExpiredError = ArbWasmImpl.ProgramExpiredError
	programs.ProgramUpToDateError = ArbWasmImpl.ProgramUpToDateError
	programs.ProgramKeepaliveTooSoon = ArbWasmImpl.ProgramKeepaliveTooSoonError
	for _, method := range ArbWasm.methods {
		method.arbosVersion = ArbWasm.arbosVersion
	}

	ArbWasmCacheImpl := &ArbWasmCache{Address: types.ArbWasmCacheAddress}
	ArbWasmCache := insert(MakePrecompile(precompilesgen.ArbWasmCacheMetaData, ArbWasmCacheImpl))
	ArbWasmCache.arbosVersion = params.ArbosVersion_Stylus
	for _, method := range ArbWasmCache.methods {
		method.arbosVersion = ArbWasmCache.arbosVersion
	}
	ArbWasmCache.methodsByName["CacheCodehash"].maxArbosVersion = params.ArbosVersion_Stylus
	ArbWasmCache.methodsByName["CacheProgram"].arbosVersion = params.ArbosVersion_StylusFixes

	ArbRetryableImpl := &ArbRetryableTx{Address: types.ArbRetryableTxAddress}
	ArbRetryable := insert(MakePrecompile(precompilesgen.ArbRetryableTxMetaData, ArbRetryableImpl))
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

	ArbSys := insert(MakePrecompile(precompilesgen.ArbSysMetaData, &ArbSys{Address: types.ArbSysAddress}))
	arbos.ArbSysAddress = ArbSys.address
	arbos.L2ToL1TransactionEventID = ArbSys.events["L2ToL1Transaction"].template.ID
	arbos.L2ToL1TxEventID = ArbSys.events["L2ToL1Tx"].template.ID

	ArbOwnerImpl := &ArbOwner{Address: types.ArbOwnerAddress}
	emitOwnerActs := func(evm mech, method bytes4, owner addr, data []byte) error {
		context := eventCtx(ArbOwnerImpl.OwnerActsGasCost(method, owner, data))
		return ArbOwnerImpl.OwnerActs(context, evm, method, owner, data)
	}
	_, ArbOwner := MakePrecompile(precompilesgen.ArbOwnerMetaData, ArbOwnerImpl)
	ArbOwner.methodsByName["GetInfraFeeAccount"].arbosVersion = params.ArbosVersion_5
	ArbOwner.methodsByName["SetInfraFeeAccount"].arbosVersion = params.ArbosVersion_5
	ArbOwner.methodsByName["ReleaseL1PricerSurplusFunds"].arbosVersion = params.ArbosVersion_10
	ArbOwner.methodsByName["SetChainConfig"].arbosVersion = params.ArbosVersion_11
	ArbOwner.methodsByName["SetBrotliCompressionLevel"].arbosVersion = params.ArbosVersion_20
	ArbOwner.methodsByName["SetGasPricingConstraints"].arbosVersion = params.ArbosVersion_50
	ArbOwner.methodsByName["SetGasBacklog"].arbosVersion = params.ArbosVersion_50
	ArbOwner.methodsByName["SetMultiGasPricingConstraints"].arbosVersion = params.ArbosVersion_60
	stylusMethods := []string{
		"SetInkPrice", "SetWasmMaxStackDepth", "SetWasmFreePages", "SetWasmPageGas",
		"SetWasmPageLimit", "SetWasmMinInitGas", "SetWasmInitCostScalar",
		"SetWasmExpiryDays", "SetWasmKeepaliveDays",
		"SetWasmBlockCacheSize", "AddWasmCacheManager", "RemoveWasmCacheManager",
	}
	for _, method := range stylusMethods {
		ArbOwner.methodsByName[method].arbosVersion = params.ArbosVersion_Stylus
	}

	insert(ownerOnly(ArbOwnerImpl.Address, ArbOwner, emitOwnerActs))
	_, arbDebug := MakePrecompile(precompilesgen.ArbDebugMetaData, &ArbDebug{Address: types.ArbDebugAddress})
	arbDebug.methodsByName["Panic"].arbosVersion = params.ArbosVersion_Stylus
	insert(debugOnly(arbDebug.address, arbDebug))

	ArbosActs := insert(MakePrecompile(precompilesgen.ArbosActsMetaData, &ArbosActs{Address: types.ArbosAddress}))
	arbos.InternalTxStartBlockMethodID = ArbosActs.GetMethodID("StartBlock")
	arbos.InternalTxBatchPostingReportMethodID = ArbosActs.GetMethodID("BatchPostingReport")
	arbos.InternalTxBatchPostingReportV2MethodID = ArbosActs.GetMethodID("BatchPostingReportV2")

	ArbOwner.methodsByName["SetCalldataPriceIncrease"].arbosVersion = params.ArbosVersion_40
	ArbOwnerPublic.methodsByName["IsCalldataPriceIncreaseEnabled"].arbosVersion = params.ArbosVersion_40

	ArbOwner.methodsByName["SetWasmMaxSize"].arbosVersion = params.ArbosVersion_40

	ArbOwner.methodsByName["SetNativeTokenManagementFrom"].arbosVersion = params.ArbosVersion_41
	ArbOwner.methodsByName["AddNativeTokenOwner"].arbosVersion = params.ArbosVersion_41
	ArbOwner.methodsByName["RemoveNativeTokenOwner"].arbosVersion = params.ArbosVersion_41
	ArbOwner.methodsByName["IsNativeTokenOwner"].arbosVersion = params.ArbosVersion_41
	ArbOwner.methodsByName["GetAllNativeTokenOwners"].arbosVersion = params.ArbosVersion_41
	ArbOwner.methodsByName["SetParentGasFloorPerToken"].arbosVersion = params.ArbosVersion_50
	ArbOwner.methodsByName["SetMaxBlockGasLimit"].arbosVersion = params.ArbosVersion_50

	ArbOwnerPublic.methodsByName["GetNativeTokenManagementFrom"].arbosVersion = params.ArbosVersion_50

	ArbNativeTokenManager := insert(MakePrecompile(precompilesgen.ArbNativeTokenManagerMetaData, &ArbNativeTokenManager{Address: types.ArbNativeTokenManagerAddress}))
	ArbNativeTokenManager.arbosVersion = params.ArbosVersion_41
	ArbNativeTokenManager.methodsByName["MintNativeToken"].arbosVersion = params.ArbosVersion_41
	ArbNativeTokenManager.methodsByName["BurnNativeToken"].arbosVersion = params.ArbosVersion_41

	// this should be executed after all precompiles have been inserted
	for _, contract := range contracts {
		precompile := contract.Precompile()
		arbosState.PrecompileMinArbOSVersions[precompile.address] = precompile.arbosVersion
	}

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

func (p *Precompile) ArbosVersion() uint64 {
	return p.arbosVersion
}

func (p *Precompile) Address() common.Address {
	return p.address
}

// Call a precompile in typed form, deserializing its inputs and serializing its outputs
func (p *Precompile) Call(
	input []byte,
	actingAsAddress common.Address,
	caller common.Address,
	value *big.Int,
	readOnly bool,
	gasSupplied uint64,
	evm *vm.EVM,
) (output []byte, gasLeft uint64, multiGasUsed multigas.MultiGas, err error) {
	arbosVersion := arbosState.ArbOSVersion(evm.StateDB)

	if arbosVersion < p.arbosVersion {
		// the precompile isn't yet active, so treat this call as if it were to a contract that doesn't exist
		return []byte{}, gasSupplied, multigas.ZeroGas(), nil
	}

	if len(input) < 4 {
		// ArbOS precompiles always have canonical method selectors
		return nil, 0, multigas.ComputationGas(gasSupplied), vm.ErrExecutionReverted
	}
	id := *(*[4]byte)(input)
	method, ok := p.methods[id]
	if !ok || arbosVersion < method.arbosVersion || (method.maxArbosVersion > 0 && arbosVersion > method.maxArbosVersion) {
		// method does not exist or hasn't yet been activated
		return nil, 0, multigas.ComputationGas(gasSupplied), vm.ErrExecutionReverted
	}

	if method.purity >= view && actingAsAddress != p.address {
		// should not access precompile superpowers when not acting as the precompile
		return nil, 0, multigas.ComputationGas(gasSupplied), vm.ErrExecutionReverted
	}

	if method.purity >= write && readOnly {
		// tried to write to global state in read-only mode
		return nil, 0, multigas.ComputationGas(gasSupplied), vm.ErrExecutionReverted
	}

	if method.purity < payable && value.Sign() != 0 {
		// tried to pay something that's non-payable
		return nil, 0, multigas.ComputationGas(gasSupplied), vm.ErrExecutionReverted
	}

	callerCtx, err := makeContext(p, method, caller, gasSupplied, evm)
	if err != nil {
		return nil, 0, multigas.ComputationGas(gasSupplied), err
	}

	// len(input) must be at least 4 because of the check near the start of this function
	// #nosec G115
	argsCost := params.CopyGas * arbmath.WordsForBytes(uint64(len(input)-4))
	if err := callerCtx.Burn(multigas.ResourceKindL2Calldata, argsCost); err != nil {
		// user cannot afford the argument data supplied
		return nil, 0, multigas.ComputationGas(gasSupplied), vm.ErrExecutionReverted
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
		panic("Unknown state mutability " + strconv.Itoa(int(method.purity)))
	}

	args, err := method.template.Inputs.Unpack(input[4:])
	if err != nil {
		// calldata does not match the method's signature
		return nil, 0, multigas.ComputationGas(gasSupplied), vm.ErrExecutionReverted
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
			return nil, callerCtx.GasLeft(), callerCtx.gasUsed, vm.ErrExecutionReverted
		}
		var solErr *SolError
		isSolErr := errors.As(errRet, &solErr)
		if isSolErr {
			resultCost := params.CopyGas * arbmath.WordsForBytes(uint64(len(solErr.data)))
			if err := callerCtx.Burn(multigas.ResourceKindComputation, resultCost); err != nil {
				// user cannot afford the result data returned
				return nil, 0, callerCtx.gasUsed, vm.ErrExecutionReverted
			}
			return solErr.data, callerCtx.GasLeft(), callerCtx.gasUsed, vm.ErrExecutionReverted
		}
		if errors.Is(errRet, programs.ErrProgramActivation) {
			// Ensure we burn all remaining gas
			callerCtx.BurnOut() //nolint:errcheck
			return nil, 0, callerCtx.gasUsed, errRet
		}
		if !errors.Is(errRet, vm.ErrOutOfGas) {
			log.Debug(
				"precompile reverted with non-solidity error",
				"precompile", p.address, "input", input, "err", errRet,
			)
		}
		// nolint:errorlint
		if arbosVersion >= params.ArbosVersion_11 || errRet == vm.ErrExecutionReverted {
			return nil, callerCtx.GasLeft(), callerCtx.gasUsed, vm.ErrExecutionReverted
		}
		// Preserve behavior with old versions which would zero out gas on this type of error
		callerCtx.BurnOut() //nolint:errcheck
		return nil, 0, callerCtx.gasUsed, errRet
	}
	result := make([]interface{}, resultCount)
	for i := 0; i < resultCount; i++ {
		result[i] = reflectResult[i].Interface()
	}

	encoded, err := method.template.Outputs.PackValues(result)
	if err != nil {
		log.Error("could not encode precompile result", "err", err)
		return nil, callerCtx.GasLeft(), callerCtx.gasUsed, vm.ErrExecutionReverted
	}

	resultCost := params.CopyGas * arbmath.WordsForBytes(uint64(len(encoded)))
	if err := callerCtx.Burn(multigas.ResourceKindComputation, resultCost); err != nil {
		// user cannot afford the result data returned
		return nil, 0, callerCtx.gasUsed, vm.ErrExecutionReverted
	}

	return encoded, callerCtx.GasLeft(), callerCtx.gasUsed, nil
}

func (p *Precompile) Precompile() *Precompile {
	return p
}

// Name returns the name of the precompile.
func (p *Precompile) Name() string {
	return p.name
}

// Get4ByteMethodSignatures is needed for the fuzzing harness
func (p *Precompile) Get4ByteMethodSignatures() [][4]byte {
	ret := make([][4]byte, 0, len(p.methods))
	for sig := range p.methods {
		ret = append(ret, sig)
	}
	sort.Slice(ret, func(i, j int) bool {
		return bytes.Compare(ret[i][:], ret[j][:]) < 0
	})
	return ret
}

func (p *Precompile) GetErrorABIs() []abi.Error {
	ret := make([]abi.Error, 0, len(p.errors))
	for _, solErr := range p.errors {
		ret = append(ret, solErr.template)
	}
	return ret
}
