// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Provides insight into the cost of using the rollup.
type ArbGasInfo struct {
	Address addr // 0x6c
}

var storageArbGas = big.NewInt(int64(storage.StorageWriteCost))

// Get prices in wei when using the provided aggregator
func (con ArbGasInfo) GetPricesInWeiWithAggregator(
	c ctx,
	evm mech,
	aggregator addr,
) (huge, huge, huge, huge, huge, huge, error) {
	if c.State.FormatVersion() < 4 {
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
	perL2Tx := arbmath.BigMulByUint(weiForL1Calldata, l1pricing.TxFixedCostEstimate)

	// nitro's compute-centric l2 gas pricing has no special compute component that rises independently
	perArbGasBase, err := c.State.L2PricingState().MinBaseFeeWei()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
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
	perL2Tx := arbmath.BigMulByUint(weiForL1Calldata, l1pricing.TxFixedCostEstimate)

	// nitro's compute-centric l2 gas pricing has no special compute component that rises independently
	perArbGasBase := l2GasPrice
	perArbGasCongestion := common.Big0
	perArbGasTotal := l2GasPrice

	weiForL2Storage := arbmath.BigMul(l2GasPrice, storageArbGas)

	return perL2Tx, weiForL1Calldata, weiForL2Storage, perArbGasBase, perArbGasCongestion, perArbGasTotal, nil
}

// Get prices in wei when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInWei(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	return con.GetPricesInWeiWithAggregator(c, evm, addr{})
}

// Get prices in ArbGas when using the provided aggregator
func (con ArbGasInfo) GetPricesInArbGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	if c.State.FormatVersion() < 4 {
		return con._preVersion4_GetPricesInArbGasWithAggregator(c, evm, aggregator)
	}
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	weiPerL2Tx := arbmath.BigMulByUint(weiForL1Calldata, l1pricing.TxFixedCostEstimate)
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

	perL2Tx := big.NewInt(l1pricing.TxFixedCostEstimate)
	return perL2Tx, gasForL1Calldata, storageArbGas, nil
}

// Get prices in ArbGas when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInArbGas(c ctx, evm mech) (huge, huge, huge, error) {
	return con.GetPricesInArbGasWithAggregator(c, evm, addr{})
}

// Get the rollup's speed limit, pool size, and tx gas limit
func (con ArbGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	l2pricing := c.State.L2PricingState()
	speedLimit, _ := l2pricing.SpeedLimitPerSecond()
	maxTxGasLimit, err := l2pricing.PerBlockGasLimit()
	return arbmath.UintToBig(speedLimit), arbmath.UintToBig(maxTxGasLimit), arbmath.UintToBig(maxTxGasLimit), err
}

// Get the minimum gas price needed for a transaction to succeed
func (con ArbGasInfo) GetMinimumGasPrice(c ctx, evm mech) (huge, error) {
	return c.State.L2PricingState().MinBaseFeeWei()
}

// Get the current estimate of the L1 basefee
func (con ArbGasInfo) GetL1BaseFeeEstimate(c ctx, evm mech) (huge, error) {
	return c.State.L1PricingState().PricePerUnit()
}

// Get how slowly ArbOS updates its estimate of the L1 basefee
func (con ArbGasInfo) GetL1BaseFeeEstimateInertia(c ctx, evm mech) (uint64, error) {
	return c.State.L1PricingState().Inertia()
}

// Get the current estimate of the L1 basefee
func (con ArbGasInfo) GetL1GasPriceEstimate(c ctx, evm mech) (huge, error) {
	return con.GetL1BaseFeeEstimate(c, evm)
}

// Get the fee paid to the aggregator for posting this tx
func (con ArbGasInfo) GetCurrentTxL1GasFees(c ctx, evm mech) (huge, error) {
	return c.txProcessor.PosterFee, nil
}

// Get the backlogged amount of gas burnt in excess of the speed limit
func (con ArbGasInfo) GetGasBacklog(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().GasBacklog()
}

// Get how slowly ArbOS updates the L2 basefee in response to backlogged gas
func (con ArbGasInfo) GetPricingInertia(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().PricingInertia()
}

// Get the forgivable amount of backlogged gas ArbOS will ignore when raising the basefee
func (con ArbGasInfo) GetGasBacklogTolerance(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().BacklogTolerance()
}

func (con ArbGasInfo) GetL1PricingSurplus(c ctx, evm mech) (*big.Int, error) {
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
