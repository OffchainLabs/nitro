// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
)

// ArbNativeTokenManager precompile enables minting and burning native tokens.
// All calls to this precompile are authorized by the NativeTokenPrecompile wrapper.
type ArbNativeTokenManager struct {
	Address                  addr // 0x73
	NativeTokenBurned        func(ctx, mech, addr, huge) error
	NativeTokenBurnedGasCost func(addr, huge) (uint64, error)
	NativeTokenMinted        func(ctx, mech, addr, huge) error
	NativeTokenMintedGasCost func(addr, huge) (uint64, error)
}

var mintBurnGasCost = params.WarmStorageReadCostEIP2929 + params.CallValueTransferGas

// Mints some amount of the native gas token for this chain to the given address
func (con ArbNativeTokenManager) MintNativeToken(c ctx, evm mech, amount huge) error {
	if !con.hasAccess(c) {
		return c.BurnOut()
	}
	if err := c.Burn(multigas.ResourceKindStorageAccess, mintBurnGasCost); err != nil {
		return err
	}

	evm.StateDB.ExpectBalanceMint(amount)
	evm.StateDB.AddBalance(c.caller, uint256.MustFromBig(amount), tracing.BalanceIncreaseMintNativeToken)
	return con.NativeTokenMinted(c, evm, c.caller, amount)
}

// Burns some amount of the native gas token for this chain from the given address
func (con ArbNativeTokenManager) BurnNativeToken(c ctx, evm mech, amount huge) error {
	if !con.hasAccess(c) {
		return c.BurnOut()
	}
	if err := c.Burn(multigas.ResourceKindStorageAccess, mintBurnGasCost); err != nil {
		return err
	}

	toSub := uint256.MustFromBig(amount)
	if evm.StateDB.GetBalance(c.caller).Cmp(toSub) < 0 {
		return errors.New("burn amount exceeds balance")
	}
	evm.StateDB.ExpectBalanceBurn(amount)
	evm.StateDB.SubBalance(c.caller, toSub, tracing.BalanceDecreaseBurnNativeToken)
	return con.NativeTokenBurned(c, evm, c.caller, amount)
}

func (con ArbNativeTokenManager) hasAccess(c ctx) bool {
	manager, err := c.State.NativeTokenOwners().IsMember(c.caller)
	return manager && err == nil
}
