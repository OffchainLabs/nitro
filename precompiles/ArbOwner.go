// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/util/arbmath"
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

const NativeTokenEnableDelay = 7 * 24 * 60 * 60

var (
	ErrOutOfBounds         = errors.New("value out of bounds")
	ErrNativeTokenDelay    = errors.New("native token feature must be enabled at least 7 days in the future")
	ErrNativeTokenBackward = errors.New("native token feature cannot be updated to a time earlier than the current time at which it is scheduled to be enabled")
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

// SetNativeTokenManagementFrom sets a time in epoch seconds when the native token
// management becomes enabled. Setting it to 0 disables the feature.
// If the feature is disabled, then the time must be at least 7 days in the
// future.
func (con ArbOwner) SetNativeTokenManagementFrom(c ctx, evm mech, timestamp uint64) error {
	if timestamp == 0 {
		return c.State.SetNativeTokenManagementFromTime(0)
	}
	stored, err := c.State.NativeTokenManagementFromTime()
	if err != nil {
		return err
	}
	now := evm.Context.Time
	// If the feature is disabled, then the time must be at least 7 days in the
	// future.
	// If the feature is scheduled to be enabled more than 7 days in the future,
	// and the new time is also in the future, then it must be at least 7 days
	// in the future.
	if (stored == 0 && timestamp < now+NativeTokenEnableDelay) ||
		(stored > now+NativeTokenEnableDelay && timestamp < now+NativeTokenEnableDelay) {
		return ErrNativeTokenDelay
	}
	// If the feature is scheduled to be enabled earlier than the minimum delay,
	// then the new time to enable it must be only further in the future.
	if stored > now && stored <= now+NativeTokenEnableDelay && timestamp < stored {
		return ErrNativeTokenBackward
	}
	return c.State.SetNativeTokenManagementFromTime(timestamp)
}

// AddNativeTokenOwner adds account as a native token owner
func (con ArbOwner) AddNativeTokenOwner(c ctx, evm mech, newOwner addr) error {
	enabledTime, err := c.State.NativeTokenManagementFromTime()
	if err != nil {
		return err
	}
	if enabledTime == 0 || enabledTime > evm.Context.Time {
		return errors.New("native token feature is not enabled yet")
	}
	return c.State.NativeTokenOwners().Add(newOwner)
}

// RemoveNativeTokenOwner removes account from the list of native token owners
func (con ArbOwner) RemoveNativeTokenOwner(c ctx, evm mech, addr addr) error {
	member, _ := con.IsNativeTokenOwner(c, evm, addr)
	if !member {
		return errors.New("tried to remove non native token owner")
	}
	return c.State.NativeTokenOwners().Remove(addr, c.State.ArbOSVersion())
}

// IsNativeTokenOwner checks if the account is a native token owner
func (con ArbOwner) IsNativeTokenOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.State.NativeTokenOwners().IsMember(addr)
}

// GetAllNativeTokenOwners retrieves the list of native token owners
func (con ArbOwner) GetAllNativeTokenOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.State.NativeTokenOwners().AllMembers(65536)
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
	if limit == 0 {
		return errors.New("speed limit must be nonzero")
	}
	return c.State.L2PricingState().SetSpeedLimitPerSecond(limit)
}

// SetMaxTxGasLimit sets the maximum size a tx can be
func (con ArbOwner) SetMaxTxGasLimit(c ctx, evm mech, limit uint64) error {
	if c.State.ArbOSVersion() < params.ArbosVersion_50 {
		return c.State.L2PricingState().SetMaxPerBlockGasLimit(limit)
	}
	return c.State.L2PricingState().SetMaxPerTxGasLimit(limit)
}

// SetMaxBlockGasLimit sets the maximum size a block can be
func (con ArbOwner) SetMaxBlockGasLimit(c ctx, evm mech, limit uint64) error {
	return c.State.L2PricingState().SetMaxPerBlockGasLimit(limit)
}

// SetL2GasPricingInertia sets the L2 gas pricing inertia
func (con ArbOwner) SetL2GasPricingInertia(c ctx, evm mech, sec uint64) error {
	if sec == 0 {
		return errors.New("price inertia must be nonzero")
	}
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

// SetInfraFeeAccount sets the infrastructure fee collector address
func (con ArbOwner) SetInfraFeeAccount(c ctx, evm mech, newInfraFeeAccount addr) error {
	return c.State.SetInfraFeeAccount(newInfraFeeAccount)
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

// Set how much L1 charges per non-zero byte of calldata
func (con ArbOwner) SetParentGasFloorPerToken(c ctx, evm mech, gasFloorPerToken uint64) error {
	return c.State.L1PricingState().SetParentGasFloorPerToken(gasFloorPerToken)
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
// Available starting in ArbOS version 20, which also raises the default to level 1
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

// Sets the number of free wasm pages a tx receives
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
	params.MinInitGas = arbmath.SaturatingUUCast[uint8](arbmath.DivCeil(gas, programs.MinInitGasUnits))
	params.MinCachedInitGas = arbmath.SaturatingUUCast[uint8](arbmath.DivCeil(cached, programs.MinCachedGasUnits))
	return params.Save()
}

// Sets the linear adjustment made to program init costs
func (con ArbOwner) SetWasmInitCostScalar(c ctx, _ mech, percent uint64) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.InitCostScalar = arbmath.SaturatingUUCast[uint8](arbmath.DivCeil(percent, programs.CostScalarPercent))
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

// SetMaxWasmSize sets the maximum size the wasm code can be in bytes after
// decompression.
func (con ArbOwner) SetWasmMaxSize(c ctx, _ mech, maxWasmSize uint32) error {
	params, err := c.State.Programs().Params()
	if err != nil {
		return err
	}
	params.MaxWasmSize = maxWasmSize
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

// SetCalldataPriceIncrease sets the increased calldata price feature on or off
// (EIP-7623)
func (con ArbOwner) SetCalldataPriceIncrease(c ctx, _ mech, enable bool) error {
	return c.State.Features().SetCalldataPriceIncrease(enable)
}

// SetGasBacklog sets the L2 gas backlog directly (used by single-constraint pricing model only)
func (con ArbOwner) SetGasBacklog(c ctx, evm mech, backlog uint64) error {
	return c.State.L2PricingState().SetGasBacklog(backlog)
}

// SetGasPricingConstraints sets the gas pricing constraints used by the multi-constraint pricing model
func (con ArbOwner) SetGasPricingConstraints(c ctx, evm mech, constraints [][3]uint64) error {
	err := c.State.L2PricingState().ClearGasConstraints()
	if err != nil {
		return fmt.Errorf("failed to clear existing constraints: %w", err)
	}

	if c.State.ArbOSVersion() >= params.ArbosVersion_MultiConstraintFix {
		limit := l2pricing.GasConstraintsMaxNum
		if len(constraints) > limit {
			return fmt.Errorf("too many constraints. Max: %d", limit)
		}
	}

	for _, constraint := range constraints {
		gasTargetPerSecond := constraint[0]
		adjustmentWindowSeconds := constraint[1]
		startingBacklogValue := constraint[2]

		if gasTargetPerSecond == 0 || adjustmentWindowSeconds == 0 {
			return fmt.Errorf("invalid constraint with target %d and adjustment window %d", gasTargetPerSecond, adjustmentWindowSeconds)
		}

		err := c.State.L2PricingState().AddGasConstraint(gasTargetPerSecond, adjustmentWindowSeconds, startingBacklogValue)
		if err != nil {
			return fmt.Errorf("failed to add constraint (target: %d, adjustment window: %d): %w", gasTargetPerSecond, adjustmentWindowSeconds, err)
		}
	}
	return nil
}

// SetMultiGasPricingConstraints configures the multi-dimensional gas pricing model
func (con ArbOwner) SetMultiGasPricingConstraints(
	c ctx,
	evm mech,
	constraints []MultiGasConstraint,
) error {
	limit := l2pricing.MultiGasConstraintsMaxNum
	if len(constraints) > limit {
		return fmt.Errorf("too many constraints. Max: %d", limit)
	}

	if err := c.State.L2PricingState().ClearMultiGasConstraints(); err != nil {
		return fmt.Errorf("failed to clear existing multi-gas constraints: %w", err)
	}

	for _, constraint := range constraints {
		if constraint.TargetPerSec == 0 || constraint.AdjustmentWindowSecs == 0 {
			return fmt.Errorf(
				"invalid constraint: target=%d adjustmentWindow=%d",
				constraint.TargetPerSec, constraint.AdjustmentWindowSecs,
			)
		}

		// Build map of resource weights
		weights := make(map[uint8]uint64, len(constraint.Resources))
		for _, r := range constraint.Resources {
			weights[r.Resource] = r.Weight
		}

		if err := c.State.L2PricingState().AddMultiGasConstraint(
			constraint.TargetPerSec,
			constraint.AdjustmentWindowSecs,
			constraint.Backlog,
			weights,
		); err != nil {
			return fmt.Errorf("failed to add multi-gas constraint: %w", err)
		}

		exps, err := c.State.L2PricingState().CalcMultiGasConstraintsExponents()
		if err != nil {
			return fmt.Errorf("failed to calculate multi-gas constraint exponents: %w", err)
		}

		// Ensure no exponent exceeds the maximum allowed value
		for _, exp := range exps {
			if exp > l2pricing.MaxPricingExponentBips {
				return fmt.Errorf("calculated exponent %d exceeds maximum allowed %d", exp, l2pricing.MaxPricingExponentBips)
			}
		}
	}
	return nil
}
