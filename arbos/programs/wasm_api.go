// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"syscall/js"
)

type apiWrapper struct {
	getBytes32 js.Func
	setBytes32 js.Func
	funcs      []byte
}

func newApi(
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	scope *vm.ScopeContext,
) *apiWrapper {
	closures := newApiClosures(interpreter, tracingInfo, scope)
	global := js.Global()
	uint8Array := global.Get("Uint8Array")

	assert := func(cond bool) {
		if !cond {
			panic("assertion failed")
		}
	}

	const (
		preU64 = iota
		preBytes32
		preString
		preNil
	)

	jsHash := func(value js.Value) common.Hash {
		hash := common.Hash{}
		assert(value.Index(0).Int() == preBytes32)
		for i := 0; i < 32; i++ {
			hash[i] = byte(value.Index(i + 1).Int())
		}
		return hash
	}

	toJs := func(prefix u8, data []byte) js.Value {
		value := append([]byte{prefix}, data...)
		array := uint8Array.New(len(value))
		js.CopyBytesToJS(array, value)
		return array
	}
	array := func(results ...any) js.Value {
		array := make([]interface{}, 0)
		for _, result := range results {
			var value js.Value
			switch result := result.(type) {
			case uint64:
				value = toJs(preU64, arbmath.UintToBytes(result))
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
		return js.ValueOf(array)
	}

	getBytes32 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		key := jsHash(args[0])
		value, cost := closures.getBytes32(key)
		stylus.Set("result", array(value, cost))
		return nil
	})
	setBytes32 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		key := jsHash(args[0])
		value := jsHash(args[1])
		cost, err := closures.setBytes32(key, value)
		if err != nil {
			stylus.Set("result", array(err))
		} else {
			stylus.Set("result", array(cost))
		}
		return nil
	})

	ids := make([]byte, 0, 4*2)
	funcs := js.Global().Get("stylus").Call("setCallbacks", getBytes32, setBytes32)
	for i := 0; i < funcs.Length(); i++ {
		ids = append(ids, arbmath.Uint32ToBytes(u32(funcs.Index(i).Int()))...)
	}
	return &apiWrapper{
		getBytes32: getBytes32,
		setBytes32: setBytes32,
		funcs:      ids,
	}
}

func (api *apiWrapper) drop() {
	api.getBytes32.Release()
	api.setBytes32.Release()
}
