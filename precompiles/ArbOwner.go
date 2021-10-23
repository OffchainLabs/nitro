//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbOwner struct{}

func (con ArbOwner) AddAllowedSender(caller addr, st *stateDB, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddAllowedSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) AddChainOwner(caller addr, st *stateDB, newOwner addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddChainOwnerGasCost(newOwner addr) uint64 {
	return 0
}

func (con ArbOwner) AllowAllSenders(caller addr, st *stateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowAllSendersGasCost() uint64 {
	return 0
}

func (con ArbOwner) AddMappingException(caller addr, st *stateDB, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddMappingExceptionGasCost(from huge, to huge) uint64 {
	return 0
}

func (con ArbOwner) AllowOnlyOwnerToSend(caller addr, st *stateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowOnlyOwnerToSendGasCost() uint64 {
	return 0
}

func (con ArbOwner) AddToReserveFunds(caller addr, st *stateDB, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddToReserveFundsGasCost() uint64 {
	return 0
}

func (con ArbOwner) ContinueCodeUpload(caller addr, st *stateDB, marshalledCode []byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) ContinueCodeUploadGasCost(marshalledCode []byte) uint64 {
	return 0
}

func (con ArbOwner) CreateChainParameter(caller addr, st *stateDB, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) CreateChainParameterGasCost(which [32]byte, value huge) uint64 {
	return 0
}

func (con ArbOwner) DeployContract(
	caller addr,
	st *stateDB,
	value huge,
	constructorData []byte,
	deemedSender addr,
	deemedNonce huge,
) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbOwner) DeployContractGasCost(constructorData []byte, deemedSender addr, deemedNonce huge) uint64 {
	return 0
}

func (con ArbOwner) FinishCodeUploadAsArbosUpgrade(
	caller addr,
	st *stateDB,
	newCodeHash [32]byte,
	oldCodeHash [32]byte,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) FinishCodeUploadAsArbosUpgradeGasCost(newCodeHash [32]byte, oldCodeHash [32]byte) uint64 {
	return 0
}

func (con ArbOwner) GetAllAllowedSenders(caller addr, st *stateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllAllowedSendersGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetAllChainOwners(caller addr, st *stateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllChainOwnersGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetAllFairGasPriceSenders(caller addr, st *stateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllFairGasPriceSendersGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetAllMappingExceptions(caller addr, st *stateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllMappingExceptionsGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetChainParameter(caller addr, st *stateDB, which [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetChainParameterGasCost(which [32]byte) uint64 {
	return 0
}

func (con ArbOwner) GetTotalOfEthBalances(caller addr, st *stateDB) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetTotalOfEthBalancesGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetLastUpgradeHash(caller addr, st *stateDB) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetLastUpgradeHashGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetUploadedCodeHash(caller addr, st *stateDB) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetUploadedCodeHashGasCost() uint64 {
	return 0
}

func (con ArbOwner) IsAllowedSender(caller addr, st *stateDB, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsAllowedSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) IsChainOwner(caller addr, st *stateDB, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsChainOwnerGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) IsFairGasPriceSender(caller addr, st *stateDB, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsFairGasPriceSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) IsMappingException(caller addr, st *stateDB, from huge, to huge) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsMappingExceptionGasCost(from huge, to huge) uint64 {
	return 0
}

func (con ArbOwner) RemoveAllowedSender(caller addr, st *stateDB, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveAllowedSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) RemoveChainOwner(caller addr, st *stateDB, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveChainOwnerGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) RemoveMappingException(caller addr, st *stateDB, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveMappingExceptionGasCost(from huge, to huge) uint64 {
	return 0
}

func (con ArbOwner) SerializeAllParameters(caller addr, st *stateDB) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) SerializeAllParametersGasCost() uint64 {
	return 0
}

func (con ArbOwner) SetChainParameter(caller addr, st *stateDB, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetChainParameterGasCost(which [32]byte, value huge) uint64 {
	return 0
}

func (con ArbOwner) SetFairGasPriceSender(caller addr, st *stateDB, addr addr, isFairGasPriceSender bool) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetFairGasPriceSenderGasCost(addr addr, isFairGasPriceSender bool) uint64 {
	return 0
}

func (con ArbOwner) SetL1GasPriceEstimate(caller addr, st *stateDB, priceInGwei huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetL1GasPriceEstimateGasCost(priceInGwei huge) uint64 {
	return 0
}

func (con ArbOwner) StartCodeUpload(caller addr, st *stateDB) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadGasCost() uint64 {
	return 0
}

func (con ArbOwner) StartCodeUploadWithCheck(caller addr, st *stateDB, oldCodeHash [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadWithCheckGasCost(oldCodeHash [32]byte) uint64 {
	return 0
}
