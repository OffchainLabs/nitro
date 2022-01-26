pragma solidity >=0.4.21 <0.9.0;

interface ArbFunctionTable {
    // This precompile provided aggregator's the ability to manage function tables.
    // Aggregation works differently in Nitro, so these methods have been stubbed and their effects disabled.
    // They are kept for backwards compatibility.

    // Reverts since the table is empty
    function upload(bytes calldata buf) external;

    // Returns the empty table's size, which is 0
    function size(address addr) external view returns(uint);

    // Does nothing
    function get(address addr, uint index) external view returns(uint, bool, uint);
}
