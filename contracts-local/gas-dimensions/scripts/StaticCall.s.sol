// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {StaticCallee, StaticCaller} from "../src//StaticCall.sol";

contract StaticCallTestScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        StaticCallee callee = new StaticCallee();
        StaticCaller caller = new StaticCaller();
        // this should be warm, to a contract with code
        caller.testStaticCallNonEmptyWarm(address(callee));
        // this should be cold, to a contract with code
        caller.testStaticCallNonEmptyCold(address(callee));
        // this should be cold, to a contract with no code
        caller.testStaticCallEmptyCold(address(0xbeef));
        // this should be warm to a contract with no code
        caller.testStaticCallEmptyWarm(address(0xbeef));
        // warm, code, mem expansion
        caller.testStaticCallNonEmptyWarmMemExpansion(address(callee));
        // warm, no code, mem expansion
        caller.testStaticCallEmptyWarmMemExpansion(address(0xbeef));
        // cold, code, no mem expansion
        caller.testStaticCallNonEmptyColdMemExpansion(address(callee));
        // cold, no code, no mem expansion
        caller.testStaticCallEmptyColdMemExpansion(address(0xbeef));

        vm.stopBroadcast();
    }
}
