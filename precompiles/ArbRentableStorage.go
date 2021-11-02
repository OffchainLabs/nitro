//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/rentableStorage"
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
	oldSizeWords := (bin.GetSlotDataSize(slotHash) + 31) / 32
	newSizeWords := (uint64(len(data)) + 31) / 32
	gasToCharge := 3 * params.SstoreSetGas
	if newSizeWords >= oldSizeWords {
		gasToCharge += oldSizeWords*params.SstoreResetGas + (newSizeWords-oldSizeWords)*params.SstoreSetGas
		// charge for storing the added data until this bin's timeout
		gasToCharge += (newSizeWords - oldSizeWords) * rentableStorage.RenewChargePer32Bytes * (evm.Context.Time.Uint64() - bin.GetTimeout()) / rentableStorage.RentableStorageLifetimeSeconds
	} else {
		gasToCharge += newSizeWords * params.SstoreResetGas
		// refund gas that would have paid for the deleted storage until this bin's timeout
		// but don't reduce gasToCharge by more than 20%
		refund := (oldSizeWords - newSizeWords) * rentableStorage.RenewChargePer32Bytes * (evm.Context.Time.Uint64() - bin.GetTimeout()) / rentableStorage.RentableStorageLifetimeSeconds
		if refund > gasToCharge/5 {
			gasToCharge = 4 * gasToCharge / 5
		} else {
			gasToCharge -= refund
		}
	}
	if err := c.burn(gasToCharge); err != nil {
		return err
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
