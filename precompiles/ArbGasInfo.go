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
	return con.GetPricesInWeiWithAggregator(c, evm, addr{})
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
	ppu, err := c.State.L1PricingState().PricePerUnit()
	if err != nil {
		return nil, err
	}
	return arbmath.BigMulByUint(ppu, params.TxDataNonZeroGasEIP2028), nil
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
