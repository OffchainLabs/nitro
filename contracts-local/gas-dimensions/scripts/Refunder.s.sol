// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {CounterArray} from "../src/CounterArray.sol";

contract RefunderScript is Script {
    CounterArray public counterArray;

    function setUp() public {
        counterArray = new CounterArray();
        uint256[] memory counters = new uint256[](20);
        for (uint256 i = 0; i < 20; i++) {
            counters[i] = i + 1;
        }
        counterArray.setCounters(counters);
    }

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        counterArray.refunder();
        vm.stopBroadcast();
    }
}
