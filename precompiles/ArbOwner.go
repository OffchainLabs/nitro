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

func (con ArbOwner) AddAllowedSender(c ctx, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddChainOwner(c ctx, evm mech, newOwner addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowAllSenders(c ctx, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddMappingException(c ctx, evm mech, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AllowOnlyOwnerToSend(c ctx, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddToReserveFunds(c ctx, evm mech, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) ContinueCodeUpload(c ctx, evm mech, marshalledCode []byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) CreateChainParameter(c ctx, evm mech, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) DeployContract(
	c ctx,
	evm mech,
	value huge,
	constructorData []byte,
	deemedSender addr,
	deemedNonce huge,
) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbOwner) FinishCodeUploadAsArbosUpgrade(c ctx, evm mech, newCodeHash [32]byte, oldCodeHash [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) GetAllAllowedSenders(c ctx, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllChainOwners(c ctx, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllFairGasPriceSenders(c ctx, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllMappingExceptions(c ctx, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetChainParameter(c ctx, evm mech, which [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetTotalOfEthBalances(c ctx, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetLastUpgradeHash(c ctx, evm mech) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) GetUploadedCodeHash(c ctx, evm mech) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbOwner) IsAllowedSender(c ctx, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsFairGasPriceSender(c ctx, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsMappingException(c ctx, evm mech, from huge, to huge) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) RemoveAllowedSender(c ctx, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveChainOwner(c ctx, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveMappingException(c ctx, evm mech, from huge, to huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SerializeAllParameters(c ctx, evm mech) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) SetChainParameter(c ctx, evm mech, which [32]byte, value huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetFairGasPriceSender(c ctx, evm mech, addr addr, isFairGasPriceSender bool) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) SetL1GasPriceEstimate(c ctx, evm mech, priceInGwei huge) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUpload(c ctx, evm mech) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) StartCodeUploadWithCheck(c ctx, evm mech, oldCodeHash [32]byte) error {
	return errors.New("unimplemented")
}
