pragma solidity >=0.4.21 <0.9.0;

interface ArbBLS {
    // Associate a BLS public key with the caller's address
    function register(uint x0, uint x1, uint y0, uint y1) external;

    // Get the BLS public key associated with an account (revert if there isn't one)
    function getPublicKey(address account) external view returns (uint, uint, uint, uint);
}
