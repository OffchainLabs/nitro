// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

interface ArbSys {
    /**
     * @notice Get Arbitrum block number (distinct from L1 block number; Arbitrum genesis block has block number 0)
     * @return block number as int
     */
    function arbBlockNumber() external view returns (uint256);
}

contract Precompile {

    uint256 public n;
    address constant arbSysAddress = address(0x64);


    function testArbSysArbBlockNumber() public {
        n = ArbSys(arbSysAddress).arbBlockNumber();
    }
}