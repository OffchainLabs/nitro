
pragma solidity >=0.4.21 <0.9.0;

/**
* @title A test contract
*/
interface ArbDebug {
    event Basic(bool flag, bytes32 indexed value);
    event Mixed(bool indexed flag, bool not, bytes32 indexed value, address conn, address indexed caller);
    event Store(bool indexed flag, address indexed field, uint24 number, bytes32 value, bytes store);

    function events(bool flag, bytes32 value) external payable returns(address, uint256);
}
