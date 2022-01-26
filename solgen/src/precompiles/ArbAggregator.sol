pragma solidity >=0.4.21 <0.9.0;

interface ArbAggregator {

    // DEPRECATED -- use getPreferredAggregators or isDefaultAggregator instead
    // If address has at least one preferred aggregator, this returns one of them chosen arbitrarily (and false)
    // If address has no preferred aggregators, this returns a default aggregator chosen arbitrarily (and true)
    // If address has no preferred aggregator and there is no default aggregator, this returns (address(0), true).
    function getPreferredAggregator(address addr) external view returns (address, bool);

    // DEPRECATED -- use addPreferredAggregator and/or removePreferredAggregator instead
    // If prefAgg is zero, this removes all of the caller's preferred aggregators.
    // Otherwise, this sets caller's preferred aggregator set to a singleton set containing prefAgg.
    function setPreferredAggregator(address prefAgg) external;

    // Add a preferred aggregator for an address
    // This reverts unless called by the address or a chain owner
    function addPreferredAggregator(address newAgg) external;

    // Remove a preferred aggregator for an address, or do nothing if the aggregator was not preferred.
    // This reverts unless called by the address or a chain owner
    function removePreferredAggregator(address aggToRemove) external;

    // Returns true iff if addr's preferred aggregator set contains aggregator.
    function isPreferredAggregator(address addr, address aggregator) external view returns(bool);

    // Get an address's list of preferred aggregators.
    // If there are more than 256 on the list, this will return an arbitrary subset of size 256.
    function getPreferredAggregators(address addr) external view returns (address[] memory);

    // DEPRECATED -- use getDefaultAggregators or isDefaultAggregator instead
    // Return a default aggregator, or zero if there are none.
    // If there are more than one default aggregator, this chooses one to return, arbitrarily.
    function getDefaultAggregator() external view returns (address);

    // DEPRECATED -- use addDefaultAggregator and/or removeDefaultAggregator instead
    // If newDefault is zero, this removes all of the default aggregators.
    // Otherwise, this sets the default aggregator set to a singleton set containing newDefault.
    // This reverts unless called by the default aggregator, its fee collector, or a chain owner
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
