// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {CounterArray} from "../src/CounterArray.sol";

contract GetSlotKeyScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        CounterArray counterArray = CounterArray(0xA6E41fFD769491a42A6e5Ce453259b93983a22EF);
        bytes32 slotKey = counterArray.getSlotKey(3);
        console.logBytes32(slotKey);
        vm.stopBroadcast();
    }
}
