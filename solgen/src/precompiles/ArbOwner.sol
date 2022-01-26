pragma solidity >=0.4.21 <0.9.0;

interface ArbOwner {
    // Support actions that can be taken by the chain's owner.
    // All methods will revert, unless the caller is the chain's owner.

    // Add account as a chain owner
    function addChainOwner(address newOwner) external;

    // Remove account from the list of chain owners
    function removeChainOwner(address ownerToRemove) external;

    // See if the user is a chain owner
    function isChainOwner(address addr) external view returns(bool);

    // Retrieves the list of chain owners
    function getAllChainOwners() external view returns(address[] memory);

    // Sets the L1 gas price estimate directly, bypassing the autoregression
    function setL1GasPriceEstimate(uint priceInWei) external;

    // Sets the L2 gas price directly, bypassing the pool calculus
    function setL2GasPrice(uint256 priceInWei) external;

    // Sets the minimum gas price needed for a transaction to succeed
    function setMinimumGasPrice(uint256 priceInWei) external view;

    // Sets the computational speed limit for the chain
    function setSpeedLimit(uint64 limit) external view;

    // Sets the number of seconds worth of the speed limit the large gas pool contains
    function setGasPoolSeconds(uint64 factor) external view;

    // Sets the number of seconds worth of the speed limit the small gas pool contains
    function setSmallGasPoolSeconds(uint64 factor) external view;

    // Sets the maximum size a tx (and block) can be
    function setMaxTxGasLimit(uint64 limit) external view;

    // Gets the network fee collector
    function getNetworkFeeAccount() external view returns(address);

    // Sets the network fee collector
    function setNetworkFeeAccount(address newNetworkFeeAccount) external view;
}
