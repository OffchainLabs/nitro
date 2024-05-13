// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

// This file exists because cgo isn't allowed in tests

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#include "arbitrator.h"
*/
import "C"
import "fmt"

func testConstants() error {

	// this closure exists to avoid polluting the package namespace
	index := 1
	errIfNotEq := func(a RequestType, b uint32) error {
		if uint32(a) != b {
			return fmt.Errorf("constant test %d failed! %d != %d", index, a, b)
		}
		index += 1
		return nil
	}

	if err := errIfNotEq(GetBytes32, C.EvmApiMethod_GetBytes32); err != nil {
		return err
	}
	if err := errIfNotEq(SetTrieSlots, C.EvmApiMethod_SetTrieSlots); err != nil {
		return err
	}
	if err := errIfNotEq(GetTransientBytes32, C.EvmApiMethod_GetTransientBytes32); err != nil {
		return err
	}
	if err := errIfNotEq(SetTransientBytes32, C.EvmApiMethod_SetTransientBytes32); err != nil {
		return err
	}
	if err := errIfNotEq(ContractCall, C.EvmApiMethod_ContractCall); err != nil {
		return err
	}
	if err := errIfNotEq(DelegateCall, C.EvmApiMethod_DelegateCall); err != nil {
		return err
	}
	if err := errIfNotEq(StaticCall, C.EvmApiMethod_StaticCall); err != nil {
		return err
	}
	if err := errIfNotEq(Create1, C.EvmApiMethod_Create1); err != nil {
		return err
	}
	if err := errIfNotEq(Create2, C.EvmApiMethod_Create2); err != nil {
		return err
	}
	if err := errIfNotEq(EmitLog, C.EvmApiMethod_EmitLog); err != nil {
		return err
	}
	if err := errIfNotEq(AccountBalance, C.EvmApiMethod_AccountBalance); err != nil {
		return err
	}
	if err := errIfNotEq(AccountCode, C.EvmApiMethod_AccountCode); err != nil {
		return err
	}
	if err := errIfNotEq(AccountCodeHash, C.EvmApiMethod_AccountCodeHash); err != nil {
		return err
	}
	if err := errIfNotEq(AddPages, C.EvmApiMethod_AddPages); err != nil {
		return err
	}
	if err := errIfNotEq(CaptureHostIO, C.EvmApiMethod_CaptureHostIO); err != nil {
		return err
	}
	if err := errIfNotEq(EvmApiMethodReqOffset, C.EVM_API_METHOD_REQ_OFFSET); err != nil {
		return err
	}

	index = 0
	assertEq := func(a apiStatus, b uint32) error {
		if uint32(a) != b {
			return fmt.Errorf("constant test %d failed! %d != %d", index, a, b)
		}
		index += 1
		return nil
	}

	if err := assertEq(Success, C.EvmApiStatus_Success); err != nil {
		return err
	}
	if err := assertEq(Failure, C.EvmApiStatus_Failure); err != nil {
		return err
	}
	if err := assertEq(OutOfGas, C.EvmApiStatus_OutOfGas); err != nil {
		return err
	}
	if err := assertEq(WriteProtection, C.EvmApiStatus_WriteProtection); err != nil {
		return err
	}
	return nil
}
