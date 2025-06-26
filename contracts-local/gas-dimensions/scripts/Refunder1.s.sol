// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {CounterArray} from "../src/CounterArray.sol";

contract RefunderScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        CounterArray counterArray = CounterArray(0x525c2aBA45F66987217323E8a05EA400C65D06DC);
        counterArray.refund1();
        vm.stopBroadcast();
    }
}
