// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {DelegateCaller, DelegateCallee} from "../src//DelegateCall.sol";

contract DelegateCallTestScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        DelegateCallee callee = new DelegateCallee();
        DelegateCaller caller = new DelegateCaller();
        // this should be warm, to a contract with code
        caller.testDelegateCallNonEmptyWarm(address(callee));
        // this should be cold, to a contract with code
        caller.testDelegateCallNonEmptyCold(address(callee));
        // this should be cold, to a contract with no code
        caller.testDelegateCallEmptyCold(address(0xbeef));
        // this should be warm to a contract with no code
        caller.testDelegateCallEmptyWarm(address(0xbeef));

        // trigger memory expansion
        caller.testDelegateCallEmptyWarmMemExpansion(address(0xbeef));
        caller.testDelegateCallNonEmptyWarmMemExpansion(address(callee));
        caller.testDelegateCallEmptyColdMemExpansion(address(0xbeef));
        caller.testDelegateCallNonEmptyColdMemExpansion(address(callee));

        vm.stopBroadcast();
    }
}
