//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbOwner struct {
	Address addr
}

func (con ArbOwner) AddAllowedSender(caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddAllowedSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) AddChainOwner(caller addr, evm mech, newOwner addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddChainOwnerGasCost(newOwner addr) uint64 {
	return 0
}

func (con ArbOwner) AllowAllSenders(caller addr, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowAllSendersGasCost() uint64 {
	return 0
}

func (con ArbOwner) AddMappingException(caller addr, evm mech, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddMappingExceptionGasCost(from huge, to huge) uint64 {
	return 0
}

func (con ArbOwner) AllowOnlyOwnerToSend(caller addr, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowOnlyOwnerToSendGasCost() uint64 {
	return 0
}

func (con ArbOwner) AddToReserveFunds(caller addr, evm mech, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddToReserveFundsGasCost() uint64 {
	return 0
}

func (con ArbOwner) ContinueCodeUpload(caller addr, evm mech, marshalledCode []byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) ContinueCodeUploadGasCost(marshalledCode []byte) uint64 {
	return 0
}

func (con ArbOwner) CreateChainParameter(caller addr, evm mech, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) CreateChainParameterGasCost(which [32]byte, value huge) uint64 {
	return 0
}

func (con ArbOwner) DeployContract(
	caller addr,
	evm mech,
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
	evm mech,
	newCodeHash [32]byte,
	oldCodeHash [32]byte,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) FinishCodeUploadAsArbosUpgradeGasCost(newCodeHash [32]byte, oldCodeHash [32]byte) uint64 {
	return 0
}

func (con ArbOwner) GetAllAllowedSenders(caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllAllowedSendersGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetAllChainOwners(caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllChainOwnersGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetAllFairGasPriceSenders(caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllFairGasPriceSendersGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetAllMappingExceptions(caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllMappingExceptionsGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetChainParameter(caller addr, evm mech, which [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetChainParameterGasCost(which [32]byte) uint64 {
	return 0
}

func (con ArbOwner) GetTotalOfEthBalances(caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetTotalOfEthBalancesGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetLastUpgradeHash(caller addr, evm mech) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetLastUpgradeHashGasCost() uint64 {
	return 0
}

func (con ArbOwner) GetUploadedCodeHash(caller addr, evm mech) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetUploadedCodeHashGasCost() uint64 {
	return 0
}

func (con ArbOwner) IsAllowedSender(caller addr, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsAllowedSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) IsChainOwner(caller addr, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsChainOwnerGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) IsFairGasPriceSender(caller addr, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsFairGasPriceSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) IsMappingException(caller addr, evm mech, from huge, to huge) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsMappingExceptionGasCost(from huge, to huge) uint64 {
	return 0
}

func (con ArbOwner) RemoveAllowedSender(caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveAllowedSenderGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) RemoveChainOwner(caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveChainOwnerGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) RemoveMappingException(caller addr, evm mech, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveMappingExceptionGasCost(from huge, to huge) uint64 {
	return 0
}

func (con ArbOwner) SerializeAllParameters(caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) SerializeAllParametersGasCost() uint64 {
	return 0
}

func (con ArbOwner) SetChainParameter(caller addr, evm mech, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetChainParameterGasCost(which [32]byte, value huge) uint64 {
	return 0
}

func (con ArbOwner) SetFairGasPriceSender(caller addr, evm mech, addr addr, isFairGasPriceSender bool) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetFairGasPriceSenderGasCost(addr addr, isFairGasPriceSender bool) uint64 {
	return 0
}

func (con ArbOwner) SetL1GasPriceEstimate(caller addr, evm mech, priceInGwei huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetL1GasPriceEstimateGasCost(priceInGwei huge) uint64 {
	return 0
}

func (con ArbOwner) StartCodeUpload(caller addr, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadGasCost() uint64 {
	return 0
}

func (con ArbOwner) StartCodeUploadWithCheck(caller addr, evm mech, oldCodeHash [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadWithCheckGasCost(oldCodeHash [32]byte) uint64 {
	return 0
}
