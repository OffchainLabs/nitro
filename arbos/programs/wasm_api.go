// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"math/big"
	"syscall/js"
)

type apiWrapper struct {
	getBytes32      js.Func
	setBytes32      js.Func
	contractCall    js.Func
	delegateCall    js.Func
	staticCall      js.Func
	create1         js.Func
	create2         js.Func
	getReturnData   js.Func
	emitLog         js.Func
	addressBalance  js.Func
	addressCodeHash js.Func
	evmBlockHash    js.Func
	funcs           []byte
}

func newApi(
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	scope *vm.ScopeContext,
) *apiWrapper {
	closures := newApiClosures(interpreter, tracingInfo, scope)
	global := js.Global()
	uint8Array := global.Get("Uint8Array")

	const (
		preU32 = iota
		preU64
		preBytes
		preBytes20
		preBytes32
		preString
		preNil
	)

	jsRead := func(value js.Value, kind u8) []u8 {
		length := value.Length()
		data := make([]u8, length)
		js.CopyBytesToGo(data, value)
		if data[0] != kind {
			panic(fmt.Sprintf("not a %v", kind))
		}
		return data[1:]
	}
	jsU32 := func(value js.Value) u32 {
		return arbmath.BytesToUint32(jsRead(value, preU32))
	}
	jsU64 := func(value js.Value) u64 {
		return arbmath.BytesToUint(jsRead(value, preU64))
	}
	jsBytes := func(value js.Value) []u8 {
		return jsRead(value, preBytes)
	}
	jsAddress := func(value js.Value) common.Address {
		return common.BytesToAddress(jsRead(value, preBytes20))
	}
	jsHash := func(value js.Value) common.Hash {
		return common.BytesToHash(jsRead(value, preBytes32))
	}
	jsBig := func(value js.Value) *big.Int {
		return jsHash(value).Big()
	}

	toJs := func(prefix u8, data []byte) js.Value {
		value := append([]byte{prefix}, data...)
		array := uint8Array.New(len(value))
		js.CopyBytesToJS(array, value)
		return array
	}
	write := func(stylus js.Value, results ...any) any {
		array := make([]interface{}, 0)
		for _, result := range results {
			var value js.Value
			switch result := result.(type) {
			case uint32:
				value = toJs(preU32, arbmath.Uint32ToBytes(result))
			case uint64:
				value = toJs(preU64, arbmath.UintToBytes(result))
			case []u8:
				value = toJs(preBytes, result[:])
			case common.Address:
				value = toJs(preBytes20, result[:])
			case common.Hash:
				value = toJs(preBytes32, result[:])
			case error:
				if result == nil {
					value = toJs(preNil, []byte{})
				} else {
					value = toJs(preString, []byte(result.Error()))
				}
			case nil:
				value = toJs(preNil, []byte{})
			default:
				panic("Unable to coerce value")
			}
			array = append(array, value)
		}
		stylus.Set("result", js.ValueOf(array))
		return nil
	}
	maybe := func(value interface{}, err error) interface{} {
		if err != nil {
			return err
		}
		return value
	}

	getBytes32 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		key := jsHash(args[0])
		value, cost := closures.getBytes32(key)
		return write(stylus, value, cost)
	})
	setBytes32 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		key := jsHash(args[0])
		value := jsHash(args[1])
		cost, err := closures.setBytes32(key, value)
		return write(stylus, maybe(cost, err))
	})
	contractCall := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		contract := jsAddress(args[0])
		input := jsBytes(args[1])
		gas := jsU64(args[2])
		value := jsBig(args[3])
		len, cost, status := closures.contractCall(contract, input, gas, value)
		return write(stylus, len, cost, status)
	})
	delegateCall := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		contract := jsAddress(args[0])
		input := jsBytes(args[1])
		gas := jsU64(args[2])
		len, cost, status := closures.delegateCall(contract, input, gas)
		return write(stylus, len, cost, status)
	})
	staticCall := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		contract := jsAddress(args[0])
		input := jsBytes(args[1])
		gas := jsU64(args[2])
		len, cost, status := closures.staticCall(contract, input, gas)
		return write(stylus, len, cost, status)
	})
	create1 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		code := jsBytes(args[0])
		endowment := jsBig(args[1])
		gas := jsU64(args[2])
		addr, len, cost, err := closures.create1(code, endowment, gas)
		return write(stylus, maybe(addr, err), len, cost)
	})
	create2 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		code := jsBytes(args[0])
		endowment := jsBig(args[1])
		salt := jsBig(args[2])
		gas := jsU64(args[3])
		addr, len, cost, err := closures.create2(code, endowment, salt, gas)
		return write(stylus, maybe(addr, err), len, cost)
	})
	getReturnData := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		data := closures.getReturnData()
		return write(stylus, data)
	})
	emitLog := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		data := jsBytes(args[0])
		topics := jsU32(args[1])
		err := closures.emitLog(data, topics)
		return write(stylus, err)
	})
	addressBalance := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		address := jsAddress(args[0])
		value, cost := closures.addressBalance(address)
		return write(stylus, value, cost)
	})
	addressCodeHash := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		address := jsAddress(args[0])
		value, cost := closures.addressCodeHash(address)
		return write(stylus, value, cost)
	})
	evmBlockHash := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		block := jsHash(args[0])
		value, cost := closures.evmBlockHash(block)
		return write(stylus, value, cost)
	})

	ids := make([]byte, 0, 10*4)
	funcs := js.Global().Get("stylus").Call("setCallbacks",
		getBytes32, setBytes32, contractCall, delegateCall,
		staticCall, create1, create2, getReturnData, emitLog,
		addressBalance, addressCodeHash, evmBlockHash,
	)
	for i := 0; i < funcs.Length(); i++ {
		ids = append(ids, arbmath.Uint32ToBytes(u32(funcs.Index(i).Int()))...)
	}
	return &apiWrapper{
		getBytes32:      getBytes32,
		setBytes32:      setBytes32,
		contractCall:    contractCall,
		delegateCall:    delegateCall,
		staticCall:      staticCall,
		create1:         create1,
		create2:         create2,
		getReturnData:   getReturnData,
		emitLog:         emitLog,
		addressBalance:  addressBalance,
		addressCodeHash: addressCodeHash,
		evmBlockHash:    evmBlockHash,
		funcs:           ids,
	}
}

func (api *apiWrapper) drop() {
	api.getBytes32.Release()
	api.setBytes32.Release()
	api.contractCall.Release()
	api.delegateCall.Release()
	api.staticCall.Release()
	api.create1.Release()
	api.create2.Release()
	api.getReturnData.Release()
	api.emitLog.Release()
	api.addressBalance.Release()
	api.addressCodeHash.Release()
	api.evmBlockHash.Release()
}
