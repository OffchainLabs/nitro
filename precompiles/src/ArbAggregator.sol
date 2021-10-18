pragma solidity >=0.4.21 <0.7.0;

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
    // Reverts unless called by the chain owner or the current default aggregator.
    function setDefaultAggregator(address newDefault) external;

    // Get the address where fees to aggregator are sent.
    // This will often but not always be the same as the aggregator's address.
    function getFeeCollector(address aggregator) external view returns (address);

    // Set the address where fees to aggregator are sent.
    // This reverts unless called by the address that would be returned by getFeeCollector(aggregator),
    //      or by the chain owner.
    function setFeeCollector(address aggregator, address newFeeCollector) external;

    // Get the tx base fee (in approximate L1 gas) for aggregator
    function getTxBaseFee(address aggregator) external view returns (uint);

    // Set the tx base fee (in approximate L1 gas) for aggregator
    // Revert unless called by aggregator or the chain owner
    // Revert if feeInL1Gas is outside the chain's allowed bounds
    function setTxBaseFee(address aggregator, uint feeInL1Gas) external;
}

