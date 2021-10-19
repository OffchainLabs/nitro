pragma solidity >=0.4.21 <0.7.0;

//This functionality has been disabled for now.  Calls to these methods will revert.
interface ArbBLS {
    // Associate a BLS public key with the caller's address
    function register(uint x0, uint x1, uint y0, uint y1) external;

    // Get the BLS public key associated with an address (revert if there isn't one)
    function getPublicKey(address addr) external view returns (uint, uint, uint, uint);
}

