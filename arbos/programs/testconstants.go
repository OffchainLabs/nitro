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
	errIfNotEq := func(index int, a RequestType, b uint32) error {
		if uint32(a) != b {
			return fmt.Errorf("constant test %d failed! %d != %d", index, a, b)
		}
		return nil
	}

	if err := errIfNotEq(1, GetBytes32, C.EvmApiMethod_GetBytes32); err != nil {
		return err
	}
	if err := errIfNotEq(2, SetTrieSlots, C.EvmApiMethod_SetTrieSlots); err != nil {
		return err
	}
	if err := errIfNotEq(3, ContractCall, C.EvmApiMethod_ContractCall); err != nil {
		return err
	}
	if err := errIfNotEq(4, DelegateCall, C.EvmApiMethod_DelegateCall); err != nil {
		return err
	}
	if err := errIfNotEq(5, StaticCall, C.EvmApiMethod_StaticCall); err != nil {
		return err
	}
	if err := errIfNotEq(6, Create1, C.EvmApiMethod_Create1); err != nil {
		return err
	}
	if err := errIfNotEq(7, Create2, C.EvmApiMethod_Create2); err != nil {
		return err
	}
	if err := errIfNotEq(8, EmitLog, C.EvmApiMethod_EmitLog); err != nil {
		return err
	}
	if err := errIfNotEq(9, AccountBalance, C.EvmApiMethod_AccountBalance); err != nil {
		return err
	}
	if err := errIfNotEq(10, AccountCode, C.EvmApiMethod_AccountCode); err != nil {
		return err
	}
	if err := errIfNotEq(12, AccountCodeHash, C.EvmApiMethod_AccountCodeHash); err != nil {
		return err
	}
	if err := errIfNotEq(13, AddPages, C.EvmApiMethod_AddPages); err != nil {
		return err
	}
	if err := errIfNotEq(14, CaptureHostIO, C.EvmApiMethod_CaptureHostIO); err != nil {
		return err
	}
	if err := errIfNotEq(15, EvmApiMethodReqOffset, C.EVM_API_METHOD_REQ_OFFSET); err != nil {
		return err
	}

	assertEq := func(index int, a apiStatus, b uint32) error {
		if uint32(a) != b {
			return fmt.Errorf("constant test %d failed! %d != %d", index, a, b)
		}
		return nil
	}

	if err := assertEq(0, Success, C.EvmApiStatus_Success); err != nil {
		return err
	}
	if err := assertEq(1, Failure, C.EvmApiStatus_Failure); err != nil {
		return err
	}
	if err := assertEq(2, OutOfGas, C.EvmApiStatus_OutOfGas); err != nil {
		return err
	}
	if err := assertEq(3, WriteProtection, C.EvmApiStatus_WriteProtection); err != nil {
		return err
	}

	return nil
}
