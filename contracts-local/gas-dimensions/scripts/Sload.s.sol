// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {Sload} from "../src//Sload.sol";

contract SloadScript is Script {
    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        Sload sload = new Sload();
        sload.warmSload();
        sload.coldSload();
        vm.stopBroadcast();
    }
}
