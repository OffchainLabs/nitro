// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func compileUserWasm(db vm.StateDB, program common.Address, wasm []byte, params *goParams) error {
	return errors.New("unimplemented")
}

func callUserWasm(
	db vm.StateDB, program common.Address, calldata []byte, gas *uint64, params *goParams,
) (uint32, []byte, error) {
	return 0, nil, errors.New("unimplemented")
}
