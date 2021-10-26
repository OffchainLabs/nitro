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

func (con ArbOwner) AddAllowedSender(b burn, caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddChainOwner(b burn, caller addr, evm mech, newOwner addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowAllSenders(b burn, caller addr, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddMappingException(b burn, caller addr, evm mech, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowOnlyOwnerToSend(b burn, caller addr, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddToReserveFunds(b burn, caller addr, evm mech, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) ContinueCodeUpload(b burn, caller addr, evm mech, marshalledCode []byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) CreateChainParameter(b burn, caller addr, evm mech, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) DeployContract(
	b burn,
	caller addr,
	evm mech,
	value huge,
	constructorData []byte,
	deemedSender addr,
	deemedNonce huge,
) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbOwner) FinishCodeUploadAsArbosUpgrade(
	b burn,
	caller addr,
	evm mech,
	newCodeHash [32]byte,
	oldCodeHash [32]byte,
) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) GetAllAllowedSenders(b burn, caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllChainOwners(b burn, caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllFairGasPriceSenders(b burn, caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllMappingExceptions(b burn, caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetChainParameter(b burn, caller addr, evm mech, which [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetTotalOfEthBalances(b burn, caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetLastUpgradeHash(b burn, caller addr, evm mech) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetUploadedCodeHash(b burn, caller addr, evm mech) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) IsAllowedSender(b burn, caller addr, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsChainOwner(b burn, caller addr, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsFairGasPriceSender(b burn, caller addr, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsMappingException(b burn, caller addr, evm mech, from huge, to huge) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) RemoveAllowedSender(b burn, caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveChainOwner(b burn, caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveMappingException(b burn, caller addr, evm mech, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SerializeAllParameters(b burn, caller addr, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) SetChainParameter(b burn, caller addr, evm mech, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetFairGasPriceSender(b burn, caller addr, evm mech, addr addr, isFairGasPriceSender bool) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetL1GasPriceEstimate(b burn, caller addr, evm mech, priceInGwei huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUpload(b burn, caller addr, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadWithCheck(b burn, caller addr, evm mech, oldCodeHash [32]byte) error {
	return errors.New("unimplemented")
}
