// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, VmSafe} from "forge-std/Script.sol";
import {ExtCodeSize} from "../src/ExtCodeSize.sol";

contract ExtCodeSizeScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        ExtCodeSize extCodeSize = new ExtCodeSize();
        extCodeSize.getExtCodeSizeCold();
        extCodeSize.getExtCodeSizeWarm();
        vm.stopBroadcast();
    }
}