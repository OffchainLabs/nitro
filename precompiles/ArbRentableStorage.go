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
	c.burn(5 * params.SstoreSetGas)
	arbos.OpenArbosState(evm.StateDB).RentableState().AllocateBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return nil
}

func (con ArbRentableStorage) GetBinTimeout(c ctx, evm mech, id *big.Int) (*big.Int, error) {
	c.burn(3 * params.SloadGas)
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return bin.GetTimeout(), nil
}

func (con ArbRentableStorage) GetBinRenewGas(c ctx, evm mech, id *big.Int) (*big.Int, error) {
	c.burn(3 * params.SloadGas)
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return bin.GetRenewGas(), nil
}

func (con ArbRentableStorage) GetInBin(c ctx, evm mech, id *big.Int, slot *big.Int) ([]byte, error) {
	c.burn(5 * params.SloadGas)
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	return bin.GetSlot(common.BigToHash(slot)), nil
}

func (con ArbRentableStorage) SetInBin(c ctx, evm mech, id *big.Int, slot *big.Int, data []byte) error {
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	slotHash := common.BigToHash(slot)
	oldSize := bin.GetSlotDataSize(slotHash)
	newSize := uint64(len(data))
	if oldSize > newSize {
		c.burn((3 + oldSize / 32) * params.SstoreResetGas)
	} else {
		c.burn((3 + oldSize / 32) * params.SstoreResetGas + ((newSize + 31 - oldSize)/32) * params.SstoreSetGas)
	}
	bin.SetSlot(slotHash, data)
	return nil
}

func (con ArbRentableStorage) DeleteInBin(c ctx, evm mech, id *big.Int, slot *big.Int) error {
	// TODO: charge gas
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(common.BigToHash(id), evm.Context.Time.Uint64())
	slotHash := common.BigToHash(slot)
	oldSize := bin.GetSlotDataSize(slotHash)
	c.burn((3 + oldSize / 32) * params.SstoreClearGas)
	bin.DeleteSlot(slotHash)
	return nil
}
