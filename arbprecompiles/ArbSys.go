//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbprecompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
)

type ArbSys struct{}

func (con ArbSys) ArbBlockNumber(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) ArbBlockNumberGasCost() uint64 {
	return 0
}

func (con ArbSys) ArbChainID(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return big.NewInt(412345), nil
}

func (con ArbSys) ArbChainIDGasCost() uint64 {
	return 0
}

func (con ArbSys) ArbOSVersion(caller common.Address) (*big.Int, error) {
	return big.NewInt(1000), nil
}

func (con ArbSys) ArbOSVersionGasCost() uint64 {
	return 0
}

func (con ArbSys) GetStorageAt(
	caller common.Address,
	st *state.StateDB,
	address common.Address,
	index *big.Int,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) GetStorageAtGasCost(address common.Address, index *big.Int) uint64 {
	return 0
}

func (con ArbSys) GetStorageGasAvailable(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) GetStorageGasAvailableGasCost() uint64 {
	return 0
}

func (con ArbSys) GetTransactionCount(
	caller common.Address,
	st *state.StateDB,
	account common.Address,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) GetTransactionCountGasCost(account common.Address) uint64 {
	return 0
}

func (con ArbSys) IsTopLevelCall(caller common.Address, st *state.StateDB) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbSys) IsTopLevelCallGasCost() uint64 {
	return 0
}

func (con ArbSys) MapL1SenderContractAddressToL2Alias(
	caller common.Address,
	sender common.Address,
	dest common.Address,
) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbSys) MapL1SenderContractAddressToL2AliasGasCost(sender common.Address, dest common.Address) uint64 {
	return 0
}

func (con ArbSys) MyCallersAddressWithoutAliasing(caller common.Address, st *state.StateDB) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbSys) MyCallersAddressWithoutAliasingGasCost() uint64 {
	return 0
}

func (con ArbSys) SendTxToL1(
	caller common.Address,
	st *state.StateDB,
	value *big.Int,
	destination common.Address,
	calldataForL1 []byte,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) SendTxToL1GasCost(destination common.Address, calldataForL1 []byte) uint64 {
	return 0
}

func (con ArbSys) WasMyCallersAddressAliased(caller common.Address, st *state.StateDB) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbSys) WasMyCallersAddressAliasedGasCost() uint64 {
	return 0
}

func (con ArbSys) WithdrawEth(
	caller common.Address,
	st *state.StateDB,
	value *big.Int,
	destination common.Address,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) WithdrawEthGasCost(destination common.Address) uint64 {
	return 0
}
