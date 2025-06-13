// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

interface ArbSys {
    /**
     * @notice Get Arbitrum block number (distinct from L1 block number; Arbitrum genesis block has block number 0)
     * @return block number as int
     */
    function arbBlockNumber() external view returns (uint256);
}

interface ArbWasm {
    function activateProgram(address program) external payable;
}

contract Precompile {

    uint256 public n;
    address constant arbSysAddress = address(0x64);
    address constant arbWasmAddress = address(0x71);


    function testArbSysArbBlockNumber() public {
        n = ArbSys(arbSysAddress).arbBlockNumber();
    }

    function testActivateProgram(address program) public payable {
        n = 1;
        ArbWasm(arbWasmAddress).activateProgram{value: msg.value}(program);
    }
}