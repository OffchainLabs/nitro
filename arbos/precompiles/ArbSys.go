//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

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

func (con ArbSys) ArbChainID(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) ArbOSVersion(caller common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) GetStorageAt(
	caller common.Address,
	st *state.StateDB,
	address common.Address,
	index *big.Int,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) GetStorageGasAvailable(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) GetTransactionCount(
	caller common.Address,
	st *state.StateDB,
	account common.Address,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbSys) IsTopLevelCall(caller common.Address, st *state.StateDB) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbSys) MapL1SenderContractAddressToL2Alias(
	caller common.Address,
	sender common.Address,
	dest common.Address,
) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbSys) MyCallersAddressWithoutAliasing(caller common.Address, st *state.StateDB) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
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

func (con ArbSys) WasMyCallersAddressAliased(caller common.Address, st *state.StateDB) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbSys) WithdrawEth(
	caller common.Address,
	st *state.StateDB,
	value *big.Int,
	destination common.Address,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}
