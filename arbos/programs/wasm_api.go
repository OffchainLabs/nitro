// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"syscall/js"
)

type apiWrapper struct {
	getBytes32 js.Func
	setBytes32 js.Func
	funcs      []byte
}

func wrapGoApi(id usize) (*apiWrapper, usize) {
	println("Wrap", id)

	closures := getApi(id)

	toAny := func(data []byte) []interface{} {
		cast := []interface{}{}
		for _, b := range data {
			cast = append(cast, b)
		}
		return cast
	}
	array := func(results ...any) js.Value {
		array := make([]interface{}, 0)
		for _, value := range results {
			switch value := value.(type) {
			case common.Hash:
				array = append(array, toAny(value[:]))
			case uint64:
				array = append(array, toAny(arbmath.UintToBytes(value)))
			case error:
				if value == nil {
					array = append(array, nil)
				} else {
					array = append(array, toAny([]byte(value.Error())))
				}
			case nil:
				array = append(array, nil)
			default:
				panic("Unable to coerce value")
			}
		}
		return js.ValueOf(array)
	}

	getBytes32 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		colors.PrintPink("Go: getBytes32 with ", len(args), " args ", args)
		key := jsHash(args[0])
		value, cost := closures.getBytes32(key)
		stylus.Set("result", array(value, cost))
		return nil
	})
	setBytes32 := js.FuncOf(func(stylus js.Value, args []js.Value) any {
		println("Go: setBytes32 with ", len(args), " args ", args)
		key := jsHash(args[0])
		value := jsHash(args[1])
		cost, err := closures.setBytes32(key, value)
		stylus.Set("result", array(cost, err))
		println("Go: done with setBytes32!")
		return nil
	})

	ids := make([]byte, 0, 4*2)
	funcs := js.Global().Get("stylus").Call("setCallbacks", getBytes32, setBytes32)
	for i := 0; i < funcs.Length(); i++ {
		ids = append(ids, arbmath.Uint32ToBytes(u32(funcs.Index(i).Int()))...)
	}

	api := &apiWrapper{
		getBytes32: getBytes32,
		setBytes32: setBytes32,
		funcs:      ids,
	}
	return api, id
}

func (api *apiWrapper) drop() {
	println("wasm_api: Dropping Funcs")
	api.getBytes32.Release()
	api.setBytes32.Release()
}

func jsHash(value js.Value) common.Hash {
	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = byte(value.Index(i).Int())
	}
	return hash
}
