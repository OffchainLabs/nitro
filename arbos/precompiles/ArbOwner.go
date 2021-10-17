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

func (con ArbOwner) AddAllowedSender(st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddChainOwner(st *state.StateDB, newOwner common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowAllSenders(st *state.StateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddMappingException(st *state.StateDB, from *big.Int, to *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowOnlyOwnerToSend(st *state.StateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddToReserveFunds(st *state.StateDB, value *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) ContinueCodeUpload(st *state.StateDB, marshalledCode []byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) CreateChainParameter(st *state.StateDB, which [32]byte, value *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) DeployContract(
	st *state.StateDB,
	value *big.Int,
	constructorData []byte,
	deemedSender common.Address,
	deemedNonce *big.Int,
) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbOwner) FinishCodeUploadAsArbosUpgrade(
	st *state.StateDB,
	newCodeHash [32]byte,
	oldCodeHash [32]byte,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) GetAllAllowedSenders(st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllChainOwners(st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllFairGasPriceSenders(st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllMappingExceptions(st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetChainParameter(st *state.StateDB, which [32]byte) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetTotalOfEthBalances(st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetLastUpgradeHash(st *state.StateDB) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetUploadedCodeHash(st *state.StateDB) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) IsAllowedSender(st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsChainOwner(st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsFairGasPriceSender(st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsMappingException(st *state.StateDB, from *big.Int, to *big.Int) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) RemoveAllowedSender(st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveChainOwner(st *state.StateDB, addr common.Address) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveMappingException(st *state.StateDB, from *big.Int, to *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SerializeAllParameters(st *state.StateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) SetChainParameter(st *state.StateDB, which [32]byte, value *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetFairGasPriceSender(st *state.StateDB, addr common.Address, isFairGasPriceSender bool) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetL1GasPriceEstimate(st *state.StateDB, priceInGwei *big.Int) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUpload(st *state.StateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadWithCheck(st *state.StateDB, oldCodeHash [32]byte) error {
	return errors.New("unimplemented")
}
