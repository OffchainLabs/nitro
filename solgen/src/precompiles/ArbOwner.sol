pragma solidity >=0.4.21 <0.9.0;

interface ArbOwner {
    // Support actions that can be taken by the chain's owner.
    // All methods will revert, unless the caller is the chain's owner.

    // Promotes the user to chain owner
    function addChainOwner(address newOwner) external;

    // Demotes the user from chain owner, reverting if user is not an owner
    function removeChainOwner(address ownerToRemove) external;

    // See if the user is a chain owner
    function isChainOwner(address addr) external view returns(bool);

    // Retrieves the list of chain owners
    function getAllChainOwners() external view returns(address[] memory);

    // Sets the L1 gas price estimate directly, bypassing the autoregression
    function setL1GasPriceEstimate(uint priceInWei) external;

    // Sets the L2 gas price directly, bypassing the pool calculus
    function setL2GasPrice(uint256 priceInWei) external;
    function setMinimumGasPrice(uint256 priceInWei) external view;
    function setSpeedLimit(uint64 limit) external view;
    function setPoolMemoryFactor(uint64 factor) external view;
    function setMaxTxGasLimit(uint64 factor) external view;
    function getNetworkFeeAccount() external view returns(address);
    function setNetworkFeeAccount(address newNetworkFeeAccount) external view;
}
