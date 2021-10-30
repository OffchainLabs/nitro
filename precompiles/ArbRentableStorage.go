//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"math/big"
)

var (
	ErrBinNotFound       = errors.New("bin does not exist")
	ErrCannotRenewBinNow = errors.New("cannot renew bin now")
)

type ArbRentableStorage struct {
	Address addr
}

func (con ArbRentableStorage) AllocateBin(c ctx, evm mech, binId *big.Int) error {
	if err := c.burn(5 * params.SstoreSetGas); err != nil {
		return err
	}
	arbos.OpenArbosState(evm.StateDB).RentableState().AllocateBin(c.caller, common.BigToHash(binId), evm.Context.Time.Uint64())
	return nil
}

func (con ArbRentableStorage) GetBinTimeout(c ctx, evm mech, binId *big.Int) (*big.Int, error) {
	return con.GetForeignBinTimeout(c, evm, c.caller, binId)
}

func (con ArbRentableStorage) GetForeignBinTimeout(c ctx, evm mech, binOwner addr, binId *big.Int) (*big.Int, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return nil, err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(binOwner, common.BigToHash(binId), evm.Context.Time.Uint64())
	if bin == nil {
		return nil, ErrBinNotFound
	}
	return big.NewInt(int64(bin.GetTimeout())), nil
}

func (con ArbRentableStorage) GetBinRenewGas(c ctx, evm mech, binId *big.Int) (*big.Int, error) {
	return con.GetForeignBinRenewGas(c, evm, c.caller, binId)
}

func (con ArbRentableStorage) GetForeignBinRenewGas(c ctx, evm mech, binOwner addr, binId *big.Int) (*big.Int, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return nil, err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(binOwner, common.BigToHash(binId), evm.Context.Time.Uint64())
	if bin == nil {
		return nil, ErrBinNotFound
	}
	return big.NewInt(int64(bin.GetRenewGas())), nil
}

func (con ArbRentableStorage) RenewBin(c ctx, evm mech, binId *big.Int) error {
	return con.RenewForeignBin(c, evm, c.caller, binId)
}

func (con ArbRentableStorage) RenewForeignBin(c ctx, evm mech, binOwner addr, binId *big.Int) error {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return err
	}
	currentTimestamp := evm.Context.Time.Uint64()
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(binOwner, common.BigToHash(binId), currentTimestamp)
	if bin == nil {
		return ErrBinNotFound
	}
	if !bin.CanBeRenewedNow(currentTimestamp) {
		return ErrCannotRenewBinNow
	}
	if err := c.burn(bin.GetRenewGas()); err != nil {
		return err
	}
	bin.Renew(currentTimestamp)
	return nil
}

func (con ArbRentableStorage) GetInBin(c ctx, evm mech, binId *big.Int, slot *big.Int) ([]byte, error) {
	return con.GetInForeignBin(c, evm, c.caller, binId, slot)
}

func (con ArbRentableStorage) GetInForeignBin(c ctx, evm mech, binOwner addr, binId *big.Int, slot *big.Int) ([]byte, error) {
	if err := c.burn(5 * params.SloadGas); err != nil {
		return nil, err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(binOwner, common.BigToHash(binId), evm.Context.Time.Uint64())
	if bin == nil {
		return nil, ErrBinNotFound
	}
	return bin.GetSlot(common.BigToHash(slot)), nil
}

func (con ArbRentableStorage) SetInBin(c ctx, evm mech, binId *big.Int, slot *big.Int, data []byte) error {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(c.caller, common.BigToHash(binId), evm.Context.Time.Uint64())
	if bin == nil {
		return ErrBinNotFound
	}
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

func (con ArbRentableStorage) DeleteInBin(c ctx, evm mech, binId *big.Int, slot *big.Int) error {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return err
	}
	bin := arbos.OpenArbosState(evm.StateDB).RentableState().OpenBin(c.caller, common.BigToHash(binId), evm.Context.Time.Uint64())
	if bin == nil {
		return ErrBinNotFound
	}
	slotHash := common.BigToHash(slot)
	oldSize := bin.GetSlotDataSize(slotHash)
	if err := c.burn((1 + oldSize/32) * params.SstoreClearGas); err != nil {
		return err
	}
	bin.DeleteSlot(slotHash)
	return nil
}
