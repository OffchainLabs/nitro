
pragma solidity >=0.4.21 <0.8.0;

/**
* @title A test contract 
*/
interface ArbDebug {
    event Basic(bool flag, bytes32 indexed value);
    event Spill(bool flag, bytes32[2] indexed hashable);
    event Mixed(bool indexed flag, bool not, bytes32 indexed value, address conn, address indexed caller);

    function events(bool flag, bytes32 value) external payable returns(address, uint256);
}
