//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/arbstate/arbos/l1pricing"
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util"
)

// Provides insight into the cost of using the rollup.
type ArbGasInfo struct {
	Address addr // 0x6c
}

var zero = big.NewInt(0)
var storageArbGas = big.NewInt(int64(storage.StorageWriteCost))

// Gets prices in wei when using the provided aggregator
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

// Gets prices in wei when using the caller's preferred aggregator
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

// Gets prices in ArbGas when using the provided aggregator
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

// Gets prices in ArbGas when using the caller's preferred aggregator
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

// Gets the rollup's speed limit, pool size, and tx gas limit
func (con ArbGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	l2pricingstate := c.state.L2PricingState()
	speedLimit, _ := l2pricingstate.SpeedLimitPerSecond()
	gasPoolMax, err := l2pricingstate.GasPoolMax()
	return util.UintToBig(speedLimit), big.NewInt(gasPoolMax), util.UintToBig(l2pricing.L2GasLimit), err
}

// Get the minimum gas price needed for a transaction to succeed
func (con ArbGasInfo) GetMinimumGasPrice(c ctx, evm mech) (huge, error) {
	return c.state.L2PricingState().MinGasPriceWei()
}

// Get the number of seconds worth of the speed limit the large gas pool contains
func (con ArbGasInfo) GetGasPoolSeconds(c ctx, evm mech) (huge, error) {
	seconds, err := c.state.L2PricingState().GasPoolSeconds()
	return util.UintToBig(seconds), err
}

// Get the number of seconds worth of the speed limit the small gas pool contains
func (con ArbGasInfo) GetSmallGasPoolSeconds(c ctx, evm mech) (huge, error) {
	seconds, err := c.state.L2PricingState().SmallGasPoolSeconds()
	return util.UintToBig(seconds), err
}

// Gets the current estimate of the L1 gas price
func (con ArbGasInfo) GetL1GasPriceEstimate(c ctx, evm mech) (huge, error) {
	return c.state.L1PricingState().L1GasPriceEstimateWei()
}

// Gets the fee paid to the aggregator for posting this tx
func (con ArbGasInfo) GetCurrentTxL1GasFees(c ctx, evm mech) (huge, error) {
	return c.txProcessor.PosterFee, nil
}
