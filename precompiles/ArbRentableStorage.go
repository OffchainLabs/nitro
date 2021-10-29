//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"math/big"
)

type ArbRentableStorage struct {
	Address addr
}

func (con ArbRentableStorage) AllocateBin(c ctx, evm mech, id *big.Int) error {
	if err := c.burn(5 * params.SstoreSetGas); err != nil {
		return err
	}
	arbos.OpenArbosState(evm.StateDB).RentableState().AllocateBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return nil
}

func (con ArbRentableStorage) GetBinTimeout(c ctx, evm mech, id *big.Int) (*big.Int, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return nil, err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return bin.GetTimeout(), nil
}

func (con ArbRentableStorage) GetBinRenewGas(c ctx, evm mech, id *big.Int) (*big.Int, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return nil, err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return bin.GetRenewGas(), nil
}

func (con ArbRentableStorage) GetInBin(c ctx, evm mech, id *big.Int, slot *big.Int) ([]byte, error) {
	if err := c.burn(5 * params.SloadGas); err != nil {
		return nil, err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return bin.GetSlot(common.BigToHash(slot)), nil
}

func (con ArbRentableStorage) SetInBin(c ctx, evm mech, id *big.Int, slot *big.Int, data []byte) error {
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	slotHash := common.BigToHash(slot)
	oldSize := bin.GetSlotDataSize(slotHash)
	newSize := uint64(len(data))
	if oldSize > newSize {
		if err := c.burn((3 + oldSize/32) * params.SstoreResetGas); err != nil {
			return err
		}
	} else {
		if err := c.burn((3+oldSize/32)*params.SstoreResetGas + ((newSize+31-oldSize)/32)*params.SstoreSetGas); err != nil {
			return err
		}
	}
	bin.SetSlot(slotHash, data)
	return nil
}

func (con ArbRentableStorage) DeleteInBin(c ctx, evm mech, id *big.Int, slot *big.Int) error {
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	slotHash := common.BigToHash(slot)
	oldSize := bin.GetSlotDataSize(slotHash)
	if err := c.burn((3 + oldSize/32) * params.SstoreClearGas); err != nil {
		return err
	}
	bin.DeleteSlot(slotHash)
	return nil
}
