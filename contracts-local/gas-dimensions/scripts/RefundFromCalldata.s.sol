// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {CounterArray} from "../src//CounterArray.sol";

contract RefundFromCalldataScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));

        bytes32 slotKey = 0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e566;
        CounterArray counterArray = CounterArray(0xA6E41fFD769491a42A6e5Ce453259b93983a22EF);
        counterArray.refundFromCalldata(slotKey);
        vm.stopBroadcast();
    }
}
