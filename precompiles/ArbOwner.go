// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/util/arbmath"
	am "github.com/offchainlabs/nitro/util/arbmath"
)

// ArbOwner precompile provides owners with tools for managing the rollup.
// All calls to this precompile are authorized by the OwnerPrecompile wrapper,
// which ensures only a chain owner can access these methods. For methods that
// are safe for non-owners to call, see ArbOwnerOld
type ArbOwner struct {
	Address          addr // 0x70
	OwnerActs        func(ctx, mech, bytes4, addr, []byte) error
	OwnerActsGasCost func(bytes4, addr, []byte) (uint64, error)
}

var (
	ErrOutOfBounds = errors.New("value out of bounds")
)

// AddChainOwner adds account as a chain owner
func (con ArbOwner) AddChainOwner(c ctx, evm mech, newOwner addr) error {
	return c.State.ChainOwners().Add(newOwner)
}

// RemoveChainOwner removes account from the list of chain owners
func (con ArbOwner) RemoveChainOwner(c ctx, evm mech, addr addr) error {
	member, _ := con.IsChainOwner(c, evm, addr)
	if !member {
		return errors.New("tried to remove non-owner")
	}
	return c.State.ChainOwners().Remove(addr, c.State.ArbOSVersion())
}

// IsChainOwner checks if the account is a chain owner
func (con ArbOwner) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.State.ChainOwners().IsMember(addr)
}

// GetAllChainOwners retrieves the list of chain owners
func (con ArbOwner) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.State.ChainOwners().AllMembers(65536)
}

// SetL1BaseFeeEstimateInertia sets how slowly ArbOS updates its estimate of the L1 basefee
func (con ArbOwner) SetL1BaseFeeEstimateInertia(c ctx, evm mech, inertia uint64) error {
	return c.State.L1PricingState().SetInertia(inertia)
}

// SetL2BaseFee sets the L2 gas price directly, bypassing the pool calculus
func (con ArbOwner) SetL2BaseFee(c ctx, evm mech, priceInWei huge) error {
	return c.State.L2PricingState().SetBaseFeeWei(priceInWei)
}

// SetMinimumL2BaseFee sets the minimum base fee needed for a transaction to succeed
func (con ArbOwner) SetMinimumL2BaseFee(c ctx, evm mech, priceInWei huge) error {
	if c.txProcessor.MsgIsNonMutating() && priceInWei.Sign() == 0 {
		return errors.New("minimum base fee must be nonzero")
	}
	return c.State.L2PricingState().SetMinBaseFeeWei(priceInWei)
}

// SetSpeedLimit sets the computational speed limit for the chain
func (con ArbOwner) SetSpeedLimit(c ctx, evm mech, limit uint64) error {
	return c.State.L2PricingState().SetSpeedLimitPerSecond(limit)
}

// SetMaxTxGasLimit sets the maximum size a tx (and block) can be
func (con ArbOwner) SetMaxTxGasLimit(c ctx, evm mech, limit uint64) error {
	return c.State.L2PricingState().SetMaxPerBlockGasLimit(limit)
}

// SetL2GasPricingInertia sets the L2 gas pricing inertia
func (con ArbOwner) SetL2GasPricingInertia(c ctx, evm mech, sec uint64) error {
	return c.State.L2PricingState().SetPricingInertia(sec)
}

// SetL2GasBacklogTolerance sets the L2 gas backlog tolerance
func (con ArbOwner) SetL2GasBacklogTolerance(c ctx, evm mech, sec uint64) error {
	return c.State.L2PricingState().SetBacklogTolerance(sec)
}

// GetNetworkFeeAccount gets the network fee collector
func (con ArbOwner) GetNetworkFeeAccount(c ctx, evm mech) (addr, error) {
	return c.State.NetworkFeeAccount()
}

// GetInfraFeeAccount gets the infrastructure fee collector
func (con ArbOwner) GetInfraFeeAccount(c ctx, evm mech) (addr, error) {
	return c.State.InfraFeeAccount()
}

// SetNetworkFeeAccount sets the network fee collector to the new network fee account
func (con ArbOwner) SetNetworkFeeAccount(c ctx, evm mech, newNetworkFeeAccount addr) error {
	return c.State.SetNetworkFeeAccount(newNetworkFeeAccount)
}

// SetInfraFeeAccount sets the infra fee collector to the new network fee account
func (con ArbOwner) SetInfraFeeAccount(c ctx, evm mech, newNetworkFeeAccount addr) error {
	return c.State.SetInfraFeeAccount(newNetworkFeeAccount)
}

// ScheduleArbOSUpgrade to the requested version at the requested timestamp
func (con ArbOwner) ScheduleArbOSUpgrade(c ctx, evm mech, newVersion uint64, timestamp uint64) error {
	return c.State.ScheduleArbOSUpgrade(newVersion, timestamp)
}

// Sets equilibration units parameter for L1 price adjustment algorithm
func (con ArbOwner) SetL1PricingEquilibrationUnits(c ctx, evm mech, equilibrationUnits huge) error {
	return c.State.L1PricingState().SetEquilibrationUnits(equilibrationUnits)
}

// Sets inertia parameter for L1 price adjustment algorithm
func (con ArbOwner) SetL1PricingInertia(c ctx, evm mech, inertia uint64) error {
	return c.State.L1PricingState().SetInertia(inertia)
}

// Sets reward recipient address for L1 price adjustment algorithm
func (con ArbOwner) SetL1PricingRewardRecipient(c ctx, evm mech, recipient addr) error {
	return c.State.L1PricingState().SetPayRewardsTo(recipient)
}

// Sets reward amount for L1 price adjustment algorithm, in wei per unit
func (con ArbOwner) SetL1PricingRewardRate(c ctx, evm mech, weiPerUnit uint64) error {
	return c.State.L1PricingState().SetPerUnitReward(weiPerUnit)
}

// Set how much ArbOS charges per L1 gas spent on transaction data.
func (con ArbOwner) SetL1PricePerUnit(c ctx, evm mech, pricePerUnit *big.Int) error {
	return c.State.L1PricingState().SetPricePerUnit(pricePerUnit)
}

// Sets the base charge (in L1 gas) attributed to each data batch in the calldata pricer
func (con ArbOwner) SetPerBatchGasCharge(c ctx, evm mech, cost int64) error {
	return c.State.L1PricingState().SetPerBatchGasCost(cost)
}

// Sets the cost amortization cap in basis points
func (con ArbOwner) SetAmortizedCostCapBips(c ctx, evm mech, cap uint64) error {
	return c.State.L1PricingState().SetAmortizedCostCapBips(cap)
}

// Sets the Brotli compression level used for fast compression
// Available in ArbOS version 12 with default level as 1
func (con ArbOwner) SetBrotliCompressionLevel(c ctx, evm mech, level uint64) error {
	return c.State.SetBrotliCompressionLevel(level)
}

// Releases surplus funds from L1PricerFundsPoolAddress for use
func (con ArbOwner) ReleaseL1PricerSurplusFunds(c ctx, evm mech, maxWeiToRelease huge) (huge, error) {
	balance := evm.StateDB.GetBalance(l1pricing.L1PricerFundsPoolAddress)
	l1p := c.State.L1PricingState()
	recognized, err := l1p.L1FeesAvailable()
	if err != nil {
		return nil, err
	}
	weiToTransfer := new(big.Int).Sub(balance.ToBig(), recognized)
	if weiToTransfer.Sign() < 0 {
		return common.Big0, nil
	}
	if weiToTransfer.Cmp(maxWeiToRelease) > 0 {
		weiToTransfer = maxWeiToRelease
	}
	if _, err := l1p.AddToL1FeesAvailable(weiToTransfer); err != nil {
		return nil, err
	}
	return weiToTransfer, nil
}

// Sets the amount of ink 1 gas buys
func (con ArbOwner) SetInkPrice(c ctx, evm mech, inkPrice uint32) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	ink, err := arbmath.IntToUint24(inkPrice)
	if err != nil || ink == 0 {
		return errors.New("ink price must be a positive uint24")
	}
	params.InkPrice = ink
	return params.Save()
}

// Sets the maximum depth (in wasm words) a wasm stack may grow
func (con ArbOwner) SetWasmMaxStackDepth(c ctx, evm mech, depth uint32) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.MaxStackDepth = depth
	return params.Save()
}

// Gets the number of free wasm pages a tx gets
func (con ArbOwner) SetWasmFreePages(c ctx, evm mech, pages uint16) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.FreePages = pages
	return params.Save()
}

// Sets the base cost of each additional wasm page
func (con ArbOwner) SetWasmPageGas(c ctx, evm mech, gas uint16) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.PageGas = gas
	return params.Save()
}

// Sets the initial number of pages a wasm may allocate
func (con ArbOwner) SetWasmPageLimit(c ctx, evm mech, limit uint16) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.PageLimit = limit
	return params.Save()
}

// Sets the minimum costs to invoke a program
func (con ArbOwner) SetWasmMinInitGas(c ctx, _ mech, gas, cached uint64) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.MinInitGas = am.SaturatingUUCast[uint8](am.DivCeil(gas, programs.MinInitGasUnits))
	params.MinCachedInitGas = am.SaturatingUUCast[uint8](am.DivCeil(cached, programs.MinCachedGasUnits))
	return params.Save()
}

// Sets the linear adjustment made to program init costs
func (con ArbOwner) SetWasmInitCostScalar(c ctx, _ mech, percent uint64) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.InitCostScalar = am.SaturatingUUCast[uint8](am.DivCeil(percent, programs.CostScalarPercent))
	return params.Save()
}

// Sets the number of days after which programs deactivate
func (con ArbOwner) SetWasmExpiryDays(c ctx, _ mech, days uint16) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.ExpiryDays = days
	return params.Save()
}

// Sets the age a program must be to perform a keepalive
func (con ArbOwner) SetWasmKeepaliveDays(c ctx, _ mech, days uint16) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.KeepaliveDays = days
	return params.Save()
}

// Sets the number of extra programs ArbOS caches during a given block
func (con ArbOwner) SetWasmBlockCacheSize(c ctx, _ mech, count uint16) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.BlockCacheSize = count
	return params.Save()
}

// Adds account as a wasm cache manager
func (con ArbOwner) AddWasmCacheManager(c ctx, _ mech, manager addr) error {
	return c.State.Programs().CacheManagers().Add(manager)
}

// Removes account from the list of wasm cache managers
func (con ArbOwner) RemoveWasmCacheManager(c ctx, _ mech, manager addr) error {
	managers := c.State.Programs().CacheManagers()
	isMember, err := managers.IsMember(manager)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("tried to remove non-manager")
	}
	return managers.Remove(manager, c.State.ArbOSVersion())
}

// Sets serialized chain config in ArbOS state
func (con ArbOwner) SetChainConfig(c ctx, evm mech, serializedChainConfig []byte) error {
	if c == nil {
		return errors.New("nil context")
	}
	if c.txProcessor == nil {
		return errors.New("uninitialized tx processor")
	}
	if c.txProcessor.MsgIsNonMutating() {
		var newConfig params.ChainConfig
		err := json.Unmarshal(serializedChainConfig, &newConfig)
		if err != nil {
			return fmt.Errorf("invalid chain config, can't deserialize: %w", err)
		}
		if newConfig.ChainID == nil {
			return errors.New("invalid chain config, missing chain id")
		}
		chainId, err := c.State.ChainId()
		if err != nil {
			return fmt.Errorf("failed to get chain id from ArbOS state: %w", err)
		}
		if newConfig.ChainID.Cmp(chainId) != 0 {
			return fmt.Errorf("invalid chain config, chain id mismatch, want: %v, have: %v", chainId, newConfig.ChainID)
		}
		oldSerializedConfig, err := c.State.ChainConfig()
		if err != nil {
			return fmt.Errorf("failed to get old chain config from ArbOS state: %w", err)
		}
		if bytes.Equal(oldSerializedConfig, serializedChainConfig) {
			return errors.New("new chain config is the same as old one in ArbOS state")
		}
		if len(oldSerializedConfig) != 0 {
			var oldConfig params.ChainConfig
			err = json.Unmarshal(oldSerializedConfig, &oldConfig)
			if err != nil {
				return fmt.Errorf("failed to deserialize old chain config: %w", err)
			}
			if err := oldConfig.CheckCompatible(&newConfig, evm.Context.BlockNumber.Uint64(), evm.Context.Time); err != nil {
				return fmt.Errorf("invalid chain config, not compatible with previous: %w", err)
			}
		}
		currentConfig := evm.ChainConfig()
		if err := currentConfig.CheckCompatible(&newConfig, evm.Context.BlockNumber.Uint64(), evm.Context.Time); err != nil {
			return fmt.Errorf("invalid chain config, not compatible with EVM's chain config: %w", err)
		}
	}
	return c.State.SetChainConfig(serializedChainConfig)
}
