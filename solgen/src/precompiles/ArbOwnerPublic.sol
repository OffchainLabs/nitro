pragma solidity >=0.4.21 <0.9.0;

interface ArbOwnerPublic {
    // Inquire about ownership without being an owner

    // See if the user is a chain owner
    function isChainOwner(address addr) external view returns(bool);

    // Retrieves the list of chain owners
    function getAllChainOwners() external view returns(address[] memory);
}
