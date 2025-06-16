// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {CounterArray} from "../src//CounterArray.sol";

contract RefunderScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        CounterArray counterArray = CounterArray(0xA6E41fFD769491a42A6e5Ce453259b93983a22EF);
        console.log("CounterArray address:", address(counterArray));
        console.log("CounterArray counters[3]:", counterArray.counters(3));
        //counterArray.refunder(0, 19);
        vm.stopBroadcast();
    }
}
