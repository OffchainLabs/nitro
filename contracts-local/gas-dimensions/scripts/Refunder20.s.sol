// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {CounterArray} from "../src/CounterArray.sol";

contract RefunderScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        CounterArray counterArray = new CounterArray();
        uint256[] memory counters = new uint256[](20);
        for (uint256 i = 0; i < 20; i++) {
            counters[i] = i + 1;
        }
        counterArray.setCounters(counters);
        //counterArray.refunder(0, 19);
        vm.stopBroadcast();
    }
}
