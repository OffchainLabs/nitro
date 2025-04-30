// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ArbNativeToken precompile enables minting and burning native tokens.
// All calls to this precompile are authorized by the NativeTokenPrecompile wrapper.
type ArbNativeToken struct {
	Address addr
}

// Mints some amount of the native gas token for this chain to the given address
func (con ArbNativeToken) MintNativeToken(c ctx, evm mech, to addr, amount huge) error {
	if c.State.ArbOSVersion() < params.ArbosVersion_41 {
		return fmt.Errorf("minting native token is not supported in ArbOS version %d", c.State.ArbOSVersion())
	}

	evm.StateDB.ExpectBalanceMint(amount)
	evm.StateDB.AddBalance(to, uint256.MustFromBig(amount), tracing.BalanceIncreaseMintNativeToken)
	return nil
}

// Burns some amount of the native gas token for this chain from the given address
func (con ArbNativeToken) BurnNativeToken(c ctx, evm mech, from addr, amount huge) error {
	if c.State.ArbOSVersion() < params.ArbosVersion_41 {
		return fmt.Errorf("burning native token is not supported in ArbOS version %d", c.State.ArbOSVersion())
	}

	toSub := uint256.MustFromBig(amount)
	if evm.StateDB.GetBalance(from).Cmp(toSub) < 0 {
		return errors.New("burn amount exceeds balance")
	}
	evm.StateDB.ExpectBalanceBurn(amount)
	evm.StateDB.SubBalance(from, toSub, tracing.BalanceDecreaseBurnNativeToken)
	return nil
}
