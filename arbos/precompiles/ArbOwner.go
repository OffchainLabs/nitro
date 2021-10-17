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

type ArbOwner struct{}

func (con ArbOwner) AddAllowedSender(caller common.Address, st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddChainOwner(caller common.Address, st *state.StateDB, newOwner common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowAllSenders(caller common.Address, st *state.StateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddMappingException(caller common.Address, st *state.StateDB, from *big.Int, to *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowOnlyOwnerToSend(caller common.Address, st *state.StateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddToReserveFunds(caller common.Address, st *state.StateDB, value *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) ContinueCodeUpload(caller common.Address, st *state.StateDB, marshalledCode []byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) CreateChainParameter(
	caller common.Address,
	st *state.StateDB,
	which [32]byte,
	value *big.Int,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) DeployContract(
	caller common.Address,
	st *state.StateDB,
	value *big.Int,
	constructorData []byte,
	deemedSender common.Address,
	deemedNonce *big.Int,
) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbOwner) FinishCodeUploadAsArbosUpgrade(
	caller common.Address,
	st *state.StateDB,
	newCodeHash [32]byte,
	oldCodeHash [32]byte,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) GetAllAllowedSenders(caller common.Address, st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllChainOwners(caller common.Address, st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllFairGasPriceSenders(caller common.Address, st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllMappingExceptions(caller common.Address, st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetChainParameter(caller common.Address, st *state.StateDB, which [32]byte) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetTotalOfEthBalances(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetLastUpgradeHash(caller common.Address, st *state.StateDB) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetUploadedCodeHash(caller common.Address, st *state.StateDB) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) IsAllowedSender(caller common.Address, st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsChainOwner(caller common.Address, st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsFairGasPriceSender(caller common.Address, st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsMappingException(
	caller common.Address,
	st *state.StateDB,
	from *big.Int,
	to *big.Int,
) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) RemoveAllowedSender(caller common.Address, st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveChainOwner(caller common.Address, st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveMappingException(
	caller common.Address,
	st *state.StateDB,
	from *big.Int,
	to *big.Int,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SerializeAllParameters(caller common.Address, st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) SetChainParameter(caller common.Address, st *state.StateDB, which [32]byte, value *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetFairGasPriceSender(
	caller common.Address,
	st *state.StateDB,
	addr common.Address,
	isFairGasPriceSender bool,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetL1GasPriceEstimate(caller common.Address, st *state.StateDB, priceInGwei *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUpload(caller common.Address, st *state.StateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadWithCheck(caller common.Address, st *state.StateDB, oldCodeHash [32]byte) error {
	return errors.New("unimplemented")
}
