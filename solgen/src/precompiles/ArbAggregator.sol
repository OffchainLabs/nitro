pragma solidity >=0.4.21 <0.9.0;

interface ArbAggregator {
    // Get the preferred aggregator for an address.
    // Returns (preferredAggregatorAddress, isDefault)
    //     isDefault is true if addr is set to prefer the default aggregator
    function getPreferredAggregator(address addr) external view returns (address, bool);

    // Set the caller's preferred aggregator.
    // If prefAgg is zero, this sets the caller to prefer the default aggregator
    function setPreferredAggregator(address prefAgg) external;

    // Get default aggregator.
    function getDefaultAggregator() external view returns (address);

    // Set the preferred aggregator.
    // This reverts unless called by the aggregator, its fee collector, or a chain owner
    function setDefaultAggregator(address newDefault) external;

    // Get the aggregator's compression ratio, as measured in ppm (100% = 1,000,000)
    function getCompressionRatio(address aggregator) external view returns (uint64);

    // Set the aggregator's compression ratio, as measured in ppm (100% = 1,000,000)
    // This reverts unless called by the aggregator, its fee collector, or a chain owner
    function setCompressionRatio(address aggregator, uint64 ratio) external;

    // Get the address where fees to aggregator are sent.
    // This will often but not always be the same as the aggregator's address.
    function getFeeCollector(address aggregator) external view returns (address);

    // Set the address where fees to aggregator are sent.
    // This reverts unless called by the aggregator, its fee collector, or a chain owner
    function setFeeCollector(address aggregator, address newFeeCollector) external;

    // Get the tx base fee (in approximate L1 gas) for aggregator
    function getTxBaseFee(address aggregator) external view returns (uint);

    // Set the tx base fee (in approximate L1 gas) for aggregator
    // Revert unless called by aggregator or the chain owner
    // Revert if feeInL1Gas is outside the chain's allowed bounds
    function setTxBaseFee(address aggregator, uint feeInL1Gas) external;
}
