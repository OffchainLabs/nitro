
pragma solidity >=0.4.21 <0.9.0;

/**
* @title A test contract whose methods are only accessible in debug mode
*/
interface ArbDebug {
    // Caller becomes a chain owner
    function becomeChainOwner() external;
    
    // Emit events with values based on the args provided
    function events(bool flag, bytes32 value) external payable returns(address, uint256);

    // Events that exist for testing log creation and pricing
    event Basic(bool flag, bytes32 indexed value);
    event Mixed(bool indexed flag, bool not, bytes32 indexed value, address conn, address indexed caller);
    event Store(bool indexed flag, address indexed field, uint24 number, bytes32 value, bytes store);
}
