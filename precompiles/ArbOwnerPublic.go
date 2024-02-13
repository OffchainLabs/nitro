// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
)

// ArbOwnerPublic precompile provides non-owners with info about the current chain owners.
// The calls to this precompile do not require the sender be a chain owner.
// For those that are, see ArbOwner
type ArbOwnerPublic struct {
	Address                    addr // 0x6b
	ChainOwnerRectified        func(ctx, mech, addr) error
	ChainOwnerRectifiedGasCost func(addr) (uint64, error)
}

// GetAllChainOwners retrieves the list of chain owners
func (con ArbOwnerPublic) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.State.ChainOwners().AllMembers(65536)
}

// RectifyChainOwner checks if the account is a chain owner
func (con ArbOwnerPublic) RectifyChainOwner(c ctx, evm mech, addr addr) error {
	err := c.State.ChainOwners().RectifyMapping(addr)
	if err != nil {
		return err
	}
	return con.ChainOwnerRectified(c, evm, addr)
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

// GetBrotliCompressionLevel gets the current brotli compression level used for fast compression
func (con ArbOwnerPublic) GetBrotliCompressionLevel(c ctx, evm mech) (uint64, error) {
	return c.State.BrotliCompressionLevel()
}

// GetScheduledUpgrade gets the next scheduled ArbOS version upgrade and its activation timestamp.
// Returns (0, 0, nil) if no ArbOS upgrade is scheduled.
func (con ArbOwnerPublic) GetScheduledUpgrade(c ctx, evm mech) (uint64, uint64, error) {
	version, timestamp, err := c.State.GetScheduledUpgrade()
	if err != nil {
		return 0, 0, err
	}
	if c.State.ArbOSVersion() >= version {
		return 0, 0, nil
	}
	return version, timestamp, nil
}
