// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
)

// ArbOwnerPublic precompile provides non-owners with info about the current chain owners.
// The calls to this precompile do not require the sender be a chain owner.
// For those that are, see ArbOwner
type ArbOwnerPublic struct {
	Address addr // 0x6b
}

// GetAllChainOwners retrieves the list of chain owners
func (con ArbOwnerPublic) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.State.ChainOwners().AllMembers(65536)
}

// IsChainOwner checks if the user is a chain owner
func (con ArbOwnerPublic) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.State.ChainOwners().IsMember(addr)
}

// GetNetworkFeeAccount gets the network fee collector
func (con ArbOwnerPublic) GetNetworkFeeAccount(c ctx, evm mech) (addr, error) {
	return c.State.NetworkFeeAccount()
}

// GetInfraFeeAccount gets the infrastructure fee collector
func (con ArbOwnerPublic) GetInfraFeeAccount(c ctx, evm mech) (addr, error) {
	if c.State.ArbOSVersion() < 6 {
		return c.State.NetworkFeeAccount()
	}
	return c.State.InfraFeeAccount()
}
