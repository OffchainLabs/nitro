//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// This precompile provides owners with tools for managing the rollup.
// All calls to this precompile are authorized by the OwnerPrecompile wrapper,
// which ensures only a chain owner can access these methods. For methods that
// are safe for non-owners to call, see ArbOwnerOld
type ArbOwner struct {
	Address          addr // 0x70
	OwnerActs        func(ctx, mech, bytes4, addr, []byte) error
	OwnerActsGasCost func(bytes4, addr, []byte) (uint64, error)
}

// Add account as a chain owner
func (con ArbOwner) AddChainOwner(c ctx, evm mech, newOwner addr) error {
	return c.state.ChainOwners().Add(newOwner)
}

// Remove account from the list of chain owners
func (con ArbOwner) RemoveChainOwner(c ctx, evm mech, addr addr) error {
	member, _ := con.IsChainOwner(c, evm, addr)
	if !member {
		return errors.New("tried to remove non-owner")
	}
	return c.state.ChainOwners().Remove(addr)
}

// See if the account is a chain owner
func (con ArbOwner) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.state.ChainOwners().IsMember(addr)
}

// Retrieves the list of chain owners
func (con ArbOwner) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.state.ChainOwners().AllMembers()
}

// Sets the L1 basefee estimate directly, bypassing the autoregression
func (con ArbOwner) SetL1BaseFeeEstimate(c ctx, evm mech, priceInWei huge) error {
	return c.state.L1PricingState().SetL1BaseFeeEstimateWei(priceInWei)
}

// Set how slowly ArbOS updates its estimate of the L1 basefee
func (con ArbOwner) SetL1BaseFeeEstimateInertia(c ctx, evm mech, inertia uint64) error {
	return c.state.L1PricingState().SetL1BaseFeeEstimateInertia(inertia)
}

// Sets the L2 gas price directly, bypassing the pool calculus
func (con ArbOwner) SetL2GasPrice(c ctx, evm mech, priceInWei huge) error {
	return c.state.L2PricingState().SetGasPriceWei(priceInWei)
}

// Sets the minimum gas price needed for a transaction to succeed
func (con ArbOwner) SetMinimumGasPrice(c ctx, evm mech, priceInWei huge) error {
	return c.state.L2PricingState().SetMinGasPriceWei(priceInWei)
}

// Sets the computational speed limit for the chain
func (con ArbOwner) SetSpeedLimit(c ctx, evm mech, limit uint64) error {
	return c.state.L2PricingState().SetSpeedLimitPerSecond(limit)
}

// Sets the number of seconds worth of the speed limit the gas pool contains
func (con ArbOwner) SetGasPoolSeconds(c ctx, evm mech, seconds uint64) error {
	return c.state.L2PricingState().SetGasPoolSeconds(seconds)
}

// Set the target fullness in bips the pricing model will try to keep the pool at
func (con ArbOwner) SetGasPoolTarget(c ctx, evm mech, target uint64) error {
	return c.state.L2PricingState().SetGasPoolTarget(arbmath.SaturatingCastToBips(target))
}

// Set the extent in bips to which the pricing model favors filling the pool over increasing speeds
func (con ArbOwner) SetGasPoolWeight(c ctx, evm mech, weight uint64) error {
	return c.state.L2PricingState().SetGasPoolWeight(arbmath.SaturatingCastToBips(weight))
}

// Set how slowly ArbOS updates its estimate the amount of gas being burnt per second
func (con ArbOwner) SetRateEstimateInertia(c ctx, evm mech, inertia uint64) error {
	return c.state.L2PricingState().SetRateEstimateInertia(inertia)
}

// Sets the maximum size a tx (and block) can be
func (con ArbOwner) SetMaxTxGasLimit(c ctx, evm mech, limit uint64) error {
	return c.state.L2PricingState().SetMaxPerBlockGasLimit(limit)
}

// Gets the network fee collector
func (con ArbOwner) GetNetworkFeeAccount(c ctx, evm mech) (addr, error) {
	return c.state.NetworkFeeAccount()
}

// Sets the network fee collector
func (con ArbOwner) SetNetworkFeeAccount(c ctx, evm mech, newNetworkFeeAccount addr) error {
	return c.state.SetNetworkFeeAccount(newNetworkFeeAccount)
}
