// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {Balance} from "../src//Balance.sol";

contract BalanceScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        payable(address(0xdeadbeef)).transfer(1000000000000000000);
        Balance balance = new Balance();
        balance.callBalanceCold();
        balance.callBalanceWarm();
        vm.stopBroadcast();
    }
}
