//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos/l1pricing"
	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util"
)

var zero = big.NewInt(0)
var storageArbGas = big.NewInt(int64(storage.StorageWriteCost))

type ArbGasInfo struct {
	Address addr
}

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

func (con ArbGasInfo) GetPricesInWei(c ctx, evm mech) (huge, huge, huge, huge, huge, huge, error) {
	aggregator, _, err := c.state.L1PricingState().PreferredAggregator(c.caller)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	return con.GetPricesInWeiWithAggregator(c, evm, aggregator)
}

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

func (con ArbGasInfo) GetPricesInArbGas(c ctx, evm mech) (huge, huge, huge, error) {
	aggregator, _, err := c.state.L1PricingState().PreferredAggregator(c.caller)
	if err != nil {
		return nil, nil, nil, err
	}
	return con.GetPricesInArbGasWithAggregator(c, evm, aggregator)
}

func (con ArbGasInfo) GetGasAccountingParams(c ctx, evm mech) (huge, huge, huge, error) {
	speedLimit := big.NewInt(l2pricing.SpeedLimitPerSecond)
	gasPoolMax := big.NewInt(l2pricing.GasPoolMax)
	maxTxGasLimit := new(big.Int).SetUint64(l2pricing.MaxPerBlockGasLimit())
	return speedLimit, gasPoolMax, maxTxGasLimit, nil
}

func (con ArbGasInfo) GetL1GasPriceEstimate(c ctx, evm mech) (huge, error) {
	return c.state.L1PricingState().L1GasPriceEstimateWei()
}

func (con ArbGasInfo) GetCurrentTxL1GasFees(c ctx, evm mech) (huge, error) {
	return c.txProcessor.PosterFee, nil
}
