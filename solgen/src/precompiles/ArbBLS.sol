pragma solidity >=0.4.21 <0.9.0;

//This functionality has been disabled for now.  Calls to these methods will revert.
interface ArbBLS {
    // Deprecated -- equivalent to registerAltBN128
    function register(uint x0, uint x1, uint y0, uint y1) external;  // DEPRECATED

    // Deprecated -- equivalent to getAltBN128
    function getPublicKey(address addr) external view returns (uint, uint, uint, uint);  // DEPRECATED

    // Associate an AltBN128 public key with the caller's address
    function registerAltBN128(uint x0, uint x1, uint y0, uint y1) external;

    // Get the AltBN128 public key associated with an address (revert if there isn't one)
    function getAltBN128(address addr) external view returns (uint, uint, uint, uint);

    // Associate a BLS 12-381 public key with the caller's address
    function registerBLS12381(bytes calldata key) external;

    // Get the BLS 12-381 public key associated with an address (revert if there isn't one)
    function getBLS12381(address addr) external view returns (bytes memory);
}
