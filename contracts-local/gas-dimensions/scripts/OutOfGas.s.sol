// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {OutOfGas} from "../src/OutOfGas.sol";

contract OutOfGasScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        OutOfGas outOfGas = new OutOfGas();
        outOfGas.callOutOfGas();
        vm.stopBroadcast();
    }
}
