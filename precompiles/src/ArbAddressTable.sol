pragma solidity >=0.4.21 <0.7.0;

/** @title Precompiled contract that exists in every Arbitrum chain at 0x0000000000000000000000000000000000000066. 
* Allows registering / retrieving addresses at uint indices, saving calldata.
*/ 
interface ArbAddressTable {
    /**
    * @notice Register an address in the address table
    * @param addr address to register
    * @return index of the address (existing index, or newly created index if not already registered)
    */
    function register(address addr) external returns(uint);

    /** 
    * @param addr address to lookup
    * @return index of an address in the address table (revert if address isn't in the table)
    */
    function lookup(address addr) external view returns(uint);

    /** 
    * @notice Check whether an address exists in the address table
    * @param addr address to check for presence in table
    * @return true if address is in table
    */ 
    function addressExists(address addr) external view returns(bool);

    /**
    * @return size of address table (= first unused index)
     */
    function size() external view returns(uint);

    /** 
    * @param index index to lookup address
    * @return address at a given index in address table (revert if index is beyond end of table)
    */
    function lookupIndex(uint index) external view returns(address);

    /** 
    * @notice read a compressed address from a bytes buffer
    * @param buf bytes buffer containing an address 
    * @param offset offset of target address
    * @return resulting address and updated offset into the buffer (revert if buffer is too short)
    */
    function decompress(bytes calldata buf, uint offset) external pure returns(address, uint);

    /**
    * @notice compress an address and return the result, possibly modifying the compression/decompression state
    * @param addr address to compress
    * @return compressed address bytes
    */
    function compress(address addr) external returns(bytes memory);
}

