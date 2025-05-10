// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/arbmath"
)

// ArbNativeTokenManager precompile enables minting and burning native tokens.
// All calls to this precompile are authorized by the NativeTokenPrecompile wrapper.
type ArbNativeTokenManager struct {
	Address addr
}

var mintBurnGasCost = arbmath.WordsForBytes(32) * params.SstoreSetGas / 100

// Mints some amount of the native gas token for this chain to the given address
func (con ArbNativeTokenManager) MintNativeToken(c ctx, evm mech, amount huge) error {
	if err := c.Burn(mintBurnGasCost); err != nil {
		return err
	}

	evm.StateDB.ExpectBalanceMint(amount)
	evm.StateDB.AddBalance(c.caller, uint256.MustFromBig(amount), tracing.BalanceIncreaseMintNativeToken)
	return nil
}

// Burns some amount of the native gas token for this chain from the given address
func (con ArbNativeTokenManager) BurnNativeToken(c ctx, evm mech, amount huge) error {
	if err := c.Burn(mintBurnGasCost); err != nil {
		return err
	}

	toSub := uint256.MustFromBig(amount)
	if evm.StateDB.GetBalance(c.caller).Cmp(toSub) < 0 {
		return errors.New("burn amount exceeds balance")
	}
	evm.StateDB.ExpectBalanceBurn(amount)
	evm.StateDB.SubBalance(c.caller, toSub, tracing.BalanceDecreaseBurnNativeToken)
	return nil
}
