// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
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

// IsNativeTokenOwner checks if the account is a native token owner
func (con ArbOwnerPublic) IsNativeTokenOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.State.NativeTokenOwners().IsMember(addr)
}

// GetAllNativeTokenOwners retrieves the list of native token owners
func (con ArbOwnerPublic) GetAllNativeTokenOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.State.NativeTokenOwners().AllMembers(65536)
}

// GetNativeTokenMangementFrom returns the time in epoch seconds when the
// native token management becomes enabled
func (con ArbOwnerPublic) GetNativeTokenManagementFrom(c ctx, evm mech) (uint64, error) {
	return c.State.NativeTokenManagementFromTime()
}

// GetNetworkFeeAccount gets the network fee collector
func (con ArbOwnerPublic) GetNetworkFeeAccount(c ctx, evm mech) (addr, error) {
	return c.State.NetworkFeeAccount()
}

// GetInfraFeeAccount gets the infrastructure fee collector
func (con ArbOwnerPublic) GetInfraFeeAccount(c ctx, evm mech) (addr, error) {
	if c.State.ArbOSVersion() < params.ArbosVersion_6 {
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

// IsCalldataPriceIncreaseEnabled checks if the increased calldata price feature
// (EIP-7623) is enabled
func (con ArbOwnerPublic) IsCalldataPriceIncreaseEnabled(c ctx, _ mech) (bool, error) {
	return c.State.Features().IsIncreasedCalldataPriceEnabled()
}

// Get how much L1 charges per non-zero byte of calldata
func (con ArbOwnerPublic) GetParentGasFloorPerToken(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().ParentGasFloorPerToken()
}
