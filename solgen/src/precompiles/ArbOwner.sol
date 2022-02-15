pragma solidity >=0.4.21 <0.9.0;


/// @title Provides owners with tools for managing the rollup. 
/// @notice Calls by non-owners will always revert.
/// Most of Arbitrum Classic's owner methods have been removed since they no longer make sense in Nitro:
/// - What were once chain parameters are now parts of ArbOS's state, and those that remain are set at genesis. 
/// - ArbOS upgrades happen with the rest of the system rather than being independent
/// - Exemptions to address aliasing are no longer offered. Exemptions were intended to support backward compatibility for contracts deployed before aliasing was introduced, but no exemptions were ever requested.
/// Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000070.
interface ArbOwner {
    /// @notice Add account as a chain owner
    function addChainOwner(address newOwner) external;

    /// @notice Remove account from the list of chain owners
    function removeChainOwner(address ownerToRemove) external;

    /// @notice See if the user is a chain owner
    function isChainOwner(address addr) external view returns(bool);

    /// @notice Retrieves the list of chain owners
    function getAllChainOwners() external view returns(address[] memory);

    /// @notice Sets the L1 gas price estimate directly, bypassing the autoregression
    function setL1GasPriceEstimate(uint priceInWei) external;

    /// @notice Sets the L2 gas price directly, bypassing the pool calculus
    function setL2GasPrice(uint256 priceInWei) external;

    /// @notice Sets the minimum gas price needed for a transaction to succeed
    function setMinimumGasPrice(uint256 priceInWei) external;

    /// @notice Sets the computational speed limit for the chain
    function setSpeedLimit(uint64 limit) external view;

    /// @notice Sets the number of seconds worth of the speed limit the large gas pool contains
    function setGasPoolSeconds(uint64 factor) external view;

    /// @notice Sets the number of seconds worth of the speed limit the small gas pool contains
    function setSmallGasPoolSeconds(uint64 factor) external view;

    /// @notice Sets the maximum size a tx (and block) can be
    function setMaxTxGasLimit(uint64 limit) external view;

    /// @notice Gets the network fee collector
    function getNetworkFeeAccount() external view returns(address);

    /// @notice Sets the network fee collector
    function setNetworkFeeAccount(address newNetworkFeeAccount) external view;
}
