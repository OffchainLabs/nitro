package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#include "arbitrator.h"
*/
import "C"
import "fmt"

func errIfNotEq(index int, a RequestType, b uint32) error {
	if uint32(a) != b {
		return fmt.Errorf("constant test %d failed! %d != %d", index, a, b)
	}
	return nil
}

func testConstants() error {
	if err := errIfNotEq(1, GetBytes32, C.EvmApiMethod_GetBytes32); err != nil {
		return err
	}
	if err := errIfNotEq(2, SetBytes32, C.EvmApiMethod_SetBytes32); err != nil {
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
	return errIfNotEq(15, EvmApiMethodReqOffset, C.EVM_API_METHOD_REQ_OFFSET)
}
