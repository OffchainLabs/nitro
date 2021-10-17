pragma solidity >=0.4.21 <0.7.0;

interface ArbosTest {
    function installAccount(address addr, bool isEOA, uint balance, uint nonce, bytes calldata code, bytes calldata initStorage) external; 

    function getMarshalledStorage(address addr) external view;  // returns raw returndata

    function getAccountInfo(address addr) external view;  // returns raw returndata

    function burnArbGas(uint gasAmount) external view;

    function setNonce(address addr, uint nonce) external;
}





