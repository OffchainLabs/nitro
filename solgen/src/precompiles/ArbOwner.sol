pragma solidity >=0.4.21 <0.9.0;

interface ArbOwner {
    // Support actions that can be taken by the chain's owner.
    // All methods will revert, unless the caller is the chain's owner.

    function addChainOwner(address newOwner) external;
    function removeChainOwner(address ownerToRemove) external;    // revert if ownerToRemove is not an owner
    function isChainOwner(address addr) external view returns(bool);
    function getAllChainOwners() external view returns(address[] memory);
    function setL1GasPriceEstimate(uint priceInWei) external;
    function setL2GasPrice(uint256 priceInWei) external;
    function setMinimumGasPrice(uint256 priceInWei) external view;
    function getNetworkFeeAccount() external view returns(address);
    function setNetworkFeeAccount(address newNetworkFeeAccount) external view;
}
