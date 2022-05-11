// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Provides insight into the cost of using the rollup.
type ArbGasInfo struct {
	Address addr // 0x6c
}

var zero = big.NewInt(0)
var storageArbGas = big.NewInt(int64(storage.StorageWriteCost))

// Get prices in wei when using the provided aggregator
func (con ArbGasInfo) GetPricesInWeiWithAggregator(
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
	perL2Tx := arbmath.BigMulByUint(weiForL1Calldata, l1pricing.TxFixedCost)

	// nitro's compute-centric l2 gas pricing has no special compute component that rises independently
	perArbGasBase := l2GasPrice
	perArbGasCongestion := zero
	perArbGasTotal := l2GasPrice

	weiForL2Storage := arbmath.BigMul(l2GasPrice, storageArbGas)

	return perL2Tx, weiForL1Calldata, weiForL2Storage, perArbGasBase, perArbGasCongestion, perArbGasTotal, nil
}

// Get prices in wei when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInWei(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	maybeAggregator, err := c.State.L1PricingState().Sequencer()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	return con.GetPricesInWeiWithAggregator(c, evm, maybeAggregator)
}

// Get prices in ArbGas when using the provided aggregator
func (con ArbGasInfo) GetPricesInArbGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	l1GasPrice, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, nil, nil, err
	}
	l2GasPrice := evm.Context.BaseFee

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := arbmath.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	gasForL1Calldata := arbmath.BigDiv(weiForL1Calldata, l2GasPrice)

	perL2Tx := big.NewInt(l1pricing.TxFixedCost)
	return perL2Tx, gasForL1Calldata, storageArbGas, nil
}

// Get prices in ArbGas when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInArbGas(c ctx, evm mech) (huge, huge, huge, error) {
	maybeAggregator, err := c.State.L1PricingState().Sequencer()
	if err != nil {
		return nil, nil, nil, err
	}
	return con.GetPricesInArbGasWithAggregator(c, evm, maybeAggregator)
}

// Get the rollup's speed limit, pool size, and tx gas limit
func (con ArbGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	l2pricing := c.State.L2PricingState()
	speedLimit, _ := l2pricing.SpeedLimitPerSecond()
	gasPoolMax, _ := l2pricing.GasPoolMax()
	maxTxGasLimit, err := l2pricing.MaxPerBlockGasLimit()
	return arbmath.UintToBig(speedLimit), big.NewInt(gasPoolMax), arbmath.UintToBig(maxTxGasLimit), err
}

// Get the minimum gas price needed for a transaction to succeed
func (con ArbGasInfo) GetMinimumGasPrice(c ctx, evm mech) (huge, error) {
	return c.State.L2PricingState().MinBaseFeeWei()
}

// Get the number of seconds worth of the speed limit the gas pool contains
func (con ArbGasInfo) GetGasPoolSeconds(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().GasPoolSeconds()
}

// Get the target fullness in bips the pricing model will try to keep the pool at
func (con ArbGasInfo) GetGasPoolTarget(c ctx, evm mech) (uint64, error) {
	target, err := c.State.L2PricingState().GasPoolTarget()
	return uint64(target), err
}

// Get the extent in bips to which the pricing model favors filling the pool over increasing speeds
func (con ArbGasInfo) GetGasPoolWeight(c ctx, evm mech) (uint64, error) {
	weight, err := c.State.L2PricingState().GasPoolWeight()
	return uint64(weight), err
}

// Get ArbOS's estimate of the amount of gas being burnt per second
func (con ArbGasInfo) GetRateEstimate(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().RateEstimate()
}

// Get how slowly ArbOS updates its estimate the amount of gas being burnt per second
func (con ArbGasInfo) GetRateEstimateInertia(c ctx, evm mech) (uint64, error) {
	return c.State.L2PricingState().RateEstimateInertia()
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
	return c.State.L1PricingState().PricePerUnit()
}

// Get the fee paid to the aggregator for posting this tx
func (con ArbGasInfo) GetCurrentTxL1GasFees(c ctx, evm mech) (huge, error) {
	return c.txProcessor.PosterFee, nil
}

// Get the amount of gas remaining in the gas pool
func (con ArbGasInfo) GetGasPool(c ctx, evm mech) (int64, error) {
	return c.State.L2PricingState().GasPool_preExp()
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
