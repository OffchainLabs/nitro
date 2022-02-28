//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/l1pricing"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util"
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
	l1GasPrice, err := c.state.L1PricingState().L1GasPriceEstimateWei()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	l2GasPrice, err := c.state.L2PricingState().GasPriceWei()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	ratio, err := c.state.L1PricingState().AggregatorCompressionRatio(aggregator)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := util.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	perL1CalldataUnit := util.BigMulByUfrac(weiForL1Calldata, ratio, 16*l1pricing.DataWasNotCompressed)

	// the cost of a simple tx without calldata
	perL2Tx := util.BigMulByUint(perL1CalldataUnit, 16*l1pricing.TxFixedCost)

	// nitro's compute-centric l2 gas pricing has no special compute component that rises independently
	perArbGasBase := l2GasPrice
	perArbGasCongestion := zero
	perArbGasTotal := l2GasPrice

	weiForL2Storage := util.BigMul(l2GasPrice, storageArbGas)

	return perL2Tx, perL1CalldataUnit, weiForL2Storage, perArbGasBase, perArbGasCongestion, perArbGasTotal, nil
}

// Get prices in wei when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInWei(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	maybeAggregator, err := c.state.L1PricingState().ReimbursableAggregatorForSender(c.caller)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	if maybeAggregator == nil {
		return con.GetPricesInWeiWithAggregator(c, evm, common.Address{})
	}
	return con.GetPricesInWeiWithAggregator(c, evm, *maybeAggregator)
}

// Get prices in ArbGas when using the provided aggregator
func (con ArbGasInfo) GetPricesInArbGasWithAggregator(c ctx, evm mech, aggregator addr) (huge, huge, huge, error) {
	l1GasPrice, err := c.state.L1PricingState().L1GasPriceEstimateWei()
	if err != nil {
		return nil, nil, nil, err
	}
	l2GasPrice, err := c.state.L2PricingState().GasPriceWei()
	if err != nil {
		return nil, nil, nil, err
	}
	ratio, err := c.state.L1PricingState().AggregatorCompressionRatio(aggregator)
	if err != nil {
		return nil, nil, nil, err
	}

	// aggregators compress calldata, so we must estimate accordingly
	weiForL1Calldata := util.BigMulByUint(l1GasPrice, params.TxDataNonZeroGasEIP2028)
	compressedCharge := util.BigMulByUfrac(weiForL1Calldata, ratio, l1pricing.DataWasNotCompressed)
	gasForL1Calldata := util.BigDiv(compressedCharge, l2GasPrice)

	perL2Tx := big.NewInt(l1pricing.TxFixedCost)
	return perL2Tx, gasForL1Calldata, storageArbGas, nil
}

// Get prices in ArbGas when using the caller's preferred aggregator
func (con ArbGasInfo) GetPricesInArbGas(c ctx, evm mech) (huge, huge, huge, error) {
	maybeAggregator, err := c.state.L1PricingState().ReimbursableAggregatorForSender(c.caller)
	if err != nil {
		return nil, nil, nil, err
	}
	if maybeAggregator == nil {
		return con.GetPricesInArbGasWithAggregator(c, evm, common.Address{})
	}
	return con.GetPricesInArbGasWithAggregator(c, evm, *maybeAggregator)
}

// Get the rollup's speed limit, pool size, and tx gas limit
func (con ArbGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	l2pricing := c.state.L2PricingState()
	speedLimit, _ := l2pricing.SpeedLimitPerSecond()
	gasPoolMax, _ := l2pricing.GasPoolMax()
	maxTxGasLimit, err := l2pricing.MaxPerBlockGasLimit()
	return util.UintToBig(speedLimit), big.NewInt(gasPoolMax), util.UintToBig(maxTxGasLimit), err
}

// Get the minimum gas price needed for a transaction to succeed
func (con ArbGasInfo) GetMinimumGasPrice(c ctx, evm mech) (huge, error) {
	return c.state.L2PricingState().MinGasPriceWei()
}

// Get the number of seconds worth of the speed limit the gas pool contains
func (con ArbGasInfo) GetGasPoolSeconds(c ctx, evm mech) (uint64, error) {
	return c.state.L2PricingState().GasPoolSeconds()
}

// Get the target fullness in bips the pricing model will try to keep the pool at
func (con ArbGasInfo) GetGasPoolTarget(c ctx, evm mech) (uint64, error) {
	return c.state.L2PricingState().GasPoolTarget()
}

// Get the extent in bips to which the pricing model favors filling the pool over increasing speeds
func (con ArbGasInfo) GetGasPoolVoice(c ctx, evm mech) (uint64, error) {
	return c.state.L2PricingState().GasPoolVoice()
}

// Get ArbOS's estimate of the amount of gas being burnt per second
func (con ArbGasInfo) GetRateEstimate(c ctx, evm mech) (uint64, error) {
	return c.state.L2PricingState().RateEstimate()
}

// Get how slowly ArbOS updates its estimate the amount of gas being burnt per second
func (con ArbGasInfo) GetRateEstimateInertia(c ctx, evm mech) (uint64, error) {
	return c.state.L2PricingState().RateEstimateInertia()
}

// Get the current estimate of the L1 gas price
func (con ArbGasInfo) GetL1GasPriceEstimate(c ctx, evm mech) (huge, error) {
	return c.state.L1PricingState().L1GasPriceEstimateWei()
}

// Get how slowly ArbOS updates its estimate of the L1 gas price
func (con ArbGasInfo) GetL1GasPriceEstimateInertia(c ctx, evm mech) (uint64, error) {
	return c.state.L1PricingState().L1GasPriceEstimateInertia()
}

// Get the fee paid to the aggregator for posting this tx
func (con ArbGasInfo) GetCurrentTxL1GasFees(c ctx, evm mech) (huge, error) {
	return c.txProcessor.PosterFee, nil
}
