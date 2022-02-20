pragma solidity >=0.4.21 <0.9.0;

/// @title Deprecated - Provided aggregator's the ability to manage function tables, 
//  this enables one form of transaction compression. 
/// @notice The Nitro aggregator implementation does not use these, 
//  so these methods have been stubbed and their effects disabled. 
/// They are kept for backwards compatibility.
/// Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000068.
interface ArbFunctionTable {
    /// @notice Reverts since the table is empty
    function upload(bytes calldata buf) external;

    /// @notice Returns the empty table's size, which is 0
    function size(address addr) external view returns(uint);

    /// @notice No-op
    function get(address addr, uint index) external view returns(uint, bool, uint);
}
