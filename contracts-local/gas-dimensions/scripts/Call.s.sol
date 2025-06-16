// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {Caller, Callee} from "../src/Call.sol";

contract CallTestScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));

        Callee callee1 = new Callee();
        Callee callee2 = new Callee();
        Callee callee3 = new Callee();
        Callee callee4 = new Callee();
        Caller caller = new Caller();
        payable(caller).transfer(1 ether);

        // five axes:
        // warm / cold
        // target has code / target has no code
        // target funded / not funded
        // sending money with the call / no transfer
        // memory expansion / memory unchanged

        // 1. warm + target has code + target not funded + zero value + memory unchanged
        caller.warmNoTransferMemUnchanged(address(callee1));
        // 2. warm + target has code + target not funded + positive value + memory unchanged
        caller.warmPayableMemUnchanged(address(callee2));
        // 3. warm + target has no code + zero value
        caller.warmNoTransferMemExpansion(address(0xbeef));
        // 4. warm + target has no code + positive value
        caller.warmPayableMemExpansion(address(0xbeef));
        // 5. cold + target has code + zero value
        caller.coldNoTransferMemUnchanged(address(callee3));
        // 6. cold + target has code + positive value
        caller.coldPayableMemUnchanged(address(callee4));
        // 7. cold + target has no code + zero value
        caller.coldNoTransferMemExpansion(address(0xbeef));
        // 8. cold + target has no code + positive value
        caller.coldPayableMemExpansion(address(0xbeef));

        vm.stopBroadcast();
    }
}
