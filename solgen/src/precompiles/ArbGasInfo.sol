pragma solidity >=0.4.21 <0.9.0;

interface ArbGasInfo {
    // return gas prices in wei, assuming the specified aggregator is used
    //        (
    //            per L2 tx,
    //            per L1 calldata unit, (zero byte = 4 units, nonzero byte = 16 units)
    //            per storage allocation,
    //            per ArbGas base,
    //            per ArbGas congestion,
    //            per ArbGas total
    //        )
    function getPricesInWeiWithAggregator(address aggregator) external view returns (uint, uint, uint, uint, uint, uint);

    // DEPRECATED -- use getPricesInWeiWithAggregator instead
    // return gas prices in wei, as described above, assuming that an aggregator chosen arbitrarily from the
    //     caller's preferred aggregator set is used
    // if the caller's preferred aggregator set is empty, an arbitrarily chosen default aggregator is assumed
    function getPricesInWei() external view returns (uint, uint, uint, uint, uint, uint);

    // return prices in ArbGas (per L2 tx, per L1 calldata unit, per storage allocation),
    //       assuming the specified aggregator is used
    function getPricesInArbGasWithAggregator(address aggregator) external view returns (uint, uint, uint);

    // DEPRECATED -- use getPricesInArbGasWithAggregator instead
    // return gas prices in ArbGas, as described above, assuming that an aggregator chosen arbitrarily from the
    //     caller's preferred aggregator set is used
    // if the caller's preferred aggregator set is empty, an arbitrarily chosen default aggregator is assumed
    function getPricesInArbGas() external view returns (uint, uint, uint);

    // return gas accounting parameters (speedLimitPerSecond, gasPoolMax, maxTxGasLimit)
    function getGasAccountingParams() external view returns (uint, uint, uint);

    // get ArbOS's estimate of the L1 gas price in wei
    function getL1GasPriceEstimate() external view returns(uint);

    // get L1 gas fees paid by the current transaction
    function getCurrentTxL1GasFees() external view returns(uint);

    // get the minimum gas price needed for a transaction to succeed
    function getMinimumGasPrice() external view returns(uint);

    // get the number of seconds worth of the speed limit the large gas pool contains
    function getGasPoolSeconds() external view returns(uint);

    // get the number of seconds worth of the speed limit the small gas pool contains
    function getSmallGasPoolSeconds() external view returns(uint);
}
