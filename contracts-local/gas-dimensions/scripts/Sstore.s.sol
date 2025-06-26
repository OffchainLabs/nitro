// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {Sstore} from "../src//Sstore.sol";

contract SstoreScript is Script {
    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        Sstore sstore = new Sstore();
        sstore.sstoreColdZeroToZero();
        sstore.sstoreColdZeroToNonZero();
        sstore.sstoreColdNonZeroValueToZero();
        sstore.sstoreColdNonZeroToSameNonZeroValue();
        sstore.sstoreColdNonZeroToDifferentNonZeroValue();
        sstore.sstoreWarmZeroToZero();
        sstore.sstoreWarmZeroToNonZeroValue();
        sstore.sstoreWarmNonZeroValueToZero();
        sstore.sstoreWarmNonZeroToSameNonZeroValue();
        sstore.sstoreWarmNonZeroToDifferentNonZeroValue();
        sstore.sstoreMultipleWarmNonZeroToNonZeroToNonZero();
        sstore.sstoreMultipleWarmNonZeroToNonZeroToSameNonZero();
        sstore.sstoreMultipleWarmNonZeroToZeroToNonZero();
        sstore.sstoreMultipleWarmNonZeroToZeroToSameNonZero();
        sstore.sstoreMultipleWarmZeroToNonZeroToNonZero();
        sstore.sstoreMultipleWarmZeroToNonZeroBackToZero();
        vm.stopBroadcast();
    }
}
