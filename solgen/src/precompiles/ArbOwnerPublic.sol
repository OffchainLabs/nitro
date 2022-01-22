pragma solidity >=0.4.21 <0.9.0;

interface ArbOwnerPublic {
    // Inquire about ownership without being an owner

    function isChainOwner(address addr) external view returns(bool);
    function getAllChainOwners() external view returns(address[] memory);
    function getNetworkFeeAccount() external view returns(address);
}
