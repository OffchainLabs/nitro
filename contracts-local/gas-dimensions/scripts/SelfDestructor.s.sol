// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe} from "forge-std/Script.sol";
import {SelfDestructor} from "../src//SelfDestructor.sol";
import {PayableCounter} from "../src//PayableCounter.sol";

contract SelfDestructorScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        SelfDestructor selfDestructor = new SelfDestructor();
        PayableCounter payableCounter = new PayableCounter();

        payable(address(selfDestructor)).transfer(0.1 ether);
        payable(address(payableCounter)).transfer(0.1 ether);
        // warm, code and value at target
        selfDestructor.warmSelfDestructor(address(payableCounter));

        // warm, but there's no money to send
        SelfDestructor selfDestructor4 = new SelfDestructor();
        selfDestructor4.warmSelfDestructor(address(payableCounter));

        // cold, no code or value at target
        SelfDestructor selfDestructor2 = new SelfDestructor();
        payable(address(selfDestructor2)).transfer(0.1 ether);
        selfDestructor2.selfDestruct(address(0xcafebabe));

        // cold, code and value at target
        SelfDestructor selfDestructor3 = new SelfDestructor();
        payable(address(selfDestructor3)).transfer(0.1 ether);
        PayableCounter payableCounter2 = new PayableCounter();
        payable(address(payableCounter2)).transfer(0.1 ether);
        selfDestructor3.selfDestruct(address(payableCounter2));

        // warm but the target address is empty
        SelfDestructor selfDestructor5 = new SelfDestructor();
        payable(address(selfDestructor5)).transfer(0.1 ether);
        selfDestructor5.warmEmptySelfDestructor(address(0xdeadbeef));

        vm.stopBroadcast();
    }
}
