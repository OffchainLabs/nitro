// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// ArbGasInfo provides insight into the cost of using the rollup.
type ArbGasInfo struct {
	Address addr // 0x6c
}

var storageArbGas = big.NewInt(int64(storage.StorageWriteCost))

const AssumedSimpleTxSize = 140

// GetPricesInWeiWithAggregator gets  prices in wei when using the provided aggregator
func (con ArbGasInfo) GetPricesInWeiWithAggregator(
	c ctx,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	if c.State.ArbOSVersion() < 4 {
		return con._preVersion4_GetPricesInWeiWithAggregator(c, evm, aggregator)
	}

	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)

	// the cost of a simple tx without calldata
	perL2Tx := arbmath.BigMulByUint(weiForL1Calldata, AssumedSimpleTxSize)

	// nitro's compute-centric l2 gas pricing has no special compute component that rises independently
	perArbGasBase, err := c.State.L2PricingState().MinBaseFeeWei()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	if arbmath.BigLessThan(l2GasPrice, perArbGasBase) {
		perArbGasBase = l2GasPrice
	}
	perArbGasCongestion := arbmath.BigSub(l2GasPrice, perArbGasBase)
	perArbGasTotal := l2GasPrice

	weiForL2Storage := arbmath.BigMul(l2GasPrice, storageArbGas)

	return perL2Tx, weiForL1Calldata, weiForL2Storage, perArbGasBase, perArbGasCongestion, perArbGasTotal, nil
}

func (con ArbGasInfo) _preVersion4_GetPricesInWeiWithAggregator(
	c ctx,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)

	// the cost of a simple tx without calldata
	perL2Tx := arbmath.BigMulByUint(weiForL1Calldata, AssumedSimpleTxSize)

	// nitro's compute-centric l2 gas pricing has no special compute component that rises independently
	perArbGasBase := l2GasPrice
	perArbGasCongestion := common.Big0
	perArbGasTotal := l2GasPrice

	weiForL2Storage := arbmath.BigMul(l2GasPrice, storageArbGas)

	return perL2Tx, weiForL1Calldata, weiForL2Storage, perArbGasBase, perArbGasCongestion, perArbGasTotal, nil
}

// GetPricesInWei gets prices in wei when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInWei(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	return con.GetPricesInWeiWithAggregator(c, evm, addr{})
}

// GetPricesInArbGasWithAggregator gets prices in ArbGas when using the provided aggregator
func (con ArbGasInfo) GetPricesInArbGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	if c.State.ArbOSVersion() < 4 {
		return con._preVersion4_GetPricesInArbGasWithAggregator(c, evm, aggregator)
	}
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	weiPerL2Tx := arbmath.BigMulByUint(weiForL1Calldata, AssumedSimpleTxSize)
	gasForL1Calldata := common.Big0
	gasPerL2Tx := common.Big0
	if l2GasPrice.Sign() > 0 {
		gasForL1Calldata = arbmath.BigDiv(weiForL1Calldata, l2GasPrice)
		gasPerL2Tx = arbmath.BigDiv(weiPerL2Tx, l2GasPrice)
	}

	return gasPerL2Tx, gasForL1Calldata, storageArbGas, nil
}

func (con ArbGasInfo) _preVersion4_GetPricesInArbGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	gasForL1Calldata := common.Big0
	if l2GasPrice.Sign() > 0 {
		gasForL1Calldata = arbmath.BigDiv(weiForL1Calldata, l2GasPrice)
	}

	perL2Tx := big.NewInt(AssumedSimpleTxSize)
	return perL2Tx, gasForL1Calldata, storageArbGas, nil
}

// GetPricesInArbGas gets prices in ArbGas when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInArbGas(c ctx, evm mech) (huge, huge, huge, error) {
	return con.GetPricesInArbGasWithAggregator(c, evm, addr{})
}

// GetGasAccountingParams gets the rollup's speed limit, pool size, and tx gas limit
func (con ArbGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	l2pricing := c.State.L2PricingState()
	speedLimit, _ := l2pricing.SpeedLimitPerSecond()
	maxTxGasLimit, err := l2pricing.PerBlockGasLimit()
	return arbmath.UintToBig(speedLimit), arbmath.UintToBig(maxTxGasLimit), arbmath.UintToBig(maxTxGasLimit), err
}

// GetMinimumGasPrice gets the minimum gas price needed for a transaction to succeed
func (con ArbGasInfo) GetMinimumGasPrice(c ctx, evm mech) (huge, error) {
	return c.State.L2PricingState().MinBaseFeeWei()
}

// GetL1BaseFeeEstimate gets the current estimate of the L1 basefee
func (con ArbGasInfo) GetL1BaseFeeEstimate(c ctx, evm mech) (huge, error) {
	return c.State.L1PricingState().PricePerUnit()
}

// GetL1BaseFeeEstimateInertia gets how slowly ArbOS updates its estimate of the L1 basefee
func (con ArbGasInfo) GetL1BaseFeeEstimateInertia(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().Inertia()
}

// GetL1RewardRate gets the L1 pricer reward rate
func (con ArbGasInfo) GetL1RewardRate(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().PerUnitReward()
}

// GetL1RewardRecipient gets the L1 pricer reward recipient
func (con ArbGasInfo) GetL1RewardRecipient(c ctx, evm mech) (common.Address, error) {
	return c.State.L1PricingState().PayRewardsTo()
}

// GetL1GasPriceEstimate gets the current estimate of the L1 basefee
func (con ArbGasInfo) GetL1GasPriceEstimate(c ctx, evm mech) (huge, error) {
	return con.GetL1BaseFeeEstimate(c, evm)
}

// GetCurrentTxL1GasFees gets the fee paid to the aggregator for posting this tx
func (con ArbGasInfo) GetCurrentTxL1GasFees(c ctx, evm mech) (huge, error) {
	return c.txProcessor.PosterFee, nil
}

// GetGasBacklog gets the backlogged amount of gas burnt in excess of the speed limit
func (con ArbGasInfo) GetGasBacklog(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().GasBacklog()
}

// GetPricingInertia gets the L2 basefee in response to backlogged gas
func (con ArbGasInfo) GetPricingInertia(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().PricingInertia()
}

// GetGasBacklogTolerance gets the forgivable amount of backlogged gas ArbOS will ignore when raising the basefee
func (con ArbGasInfo) GetGasBacklogTolerance(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().BacklogTolerance()
}

func (con ArbGasInfo) GetL1PricingSurplus(c ctx, evm mech) (*big.Int, error) {
	if c.State.ArbOSVersion() < 10 {
		return con._preversion10_GetL1PricingSurplus(c, evm)
	}
	ps := c.State.L1PricingState()
	fundsDueForRefunds, err := ps.BatchPosterTable().TotalFundsDue()
	if err != nil {
		return nil, err
	}
	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return nil, err
	}
	haveFunds, err := ps.L1FeesAvailable()
	if err != nil {
		return nil, err
	}
	needFunds := arbmath.BigAdd(fundsDueForRefunds, fundsDueForRewards)
	return arbmath.BigSub(haveFunds, needFunds), nil
}

func (con ArbGasInfo) _preversion10_GetL1PricingSurplus(c ctx, evm mech) (*big.Int, error) {
	ps := c.State.L1PricingState()
	fundsDueForRefunds, err := ps.BatchPosterTable().TotalFundsDue()
	if err != nil {
		return nil, err
	}
	fundsDueForRewards, err := ps.FundsDueForRewards()
	if err != nil {
		return nil, err
	}
	haveFunds := evm.StateDB.GetBalance(l1pricing.L1PricerFundsPoolAddress)
	needFunds := arbmath.BigAdd(fundsDueForRefunds, fundsDueForRewards)
	return arbmath.BigSub(haveFunds, needFunds), nil
}

func (con ArbGasInfo) GetPerBatchGasCharge(c ctx, evm mech) (int64, error) {
	return c.State.L1PricingState().PerBatchGasCost()
}

func (con ArbGasInfo) GetAmortizedCostCapBips(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().AmortizedCostCapBips()
}

func (con ArbGasInfo) GetL1FeesAvailable(c ctx, evm mech) (huge, error) {
	return c.State.L1PricingState().L1FeesAvailable()
}

func (con ArbGasInfo) GetL1PricingEquilibrationUnits(c ctx, evm mech) (*big.Int, error) {
	return c.State.L1PricingState().EquilibrationUnits()
}

func (con ArbGasInfo) GetLastL1PricingUpdateTime(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().LastUpdateTime()
}

func (con ArbGasInfo) GetL1PricingFundsDueForRewards(c ctx, evm mech) (*big.Int, error) {
	return c.State.L1PricingState().FundsDueForRewards()
}

func (con ArbGasInfo) GetL1PricingUnitsSinceUpdate(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().UnitsSinceUpdate()
}

func (con ArbGasInfo) GetLastL1PricingSurplus(c ctx, evm mech) (*big.Int, error) {
	return c.State.L1PricingState().LastSurplus()
}
