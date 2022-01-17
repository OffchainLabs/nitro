pragma solidity >=0.4.21 <0.9.0;

interface ArbInfo {
    // Retrieves an account's balance
    function getBalance(address account) external view returns (uint256);

    // Retrieves a contract's source program
    function getCode(address account) external view returns (bytes memory);
}
