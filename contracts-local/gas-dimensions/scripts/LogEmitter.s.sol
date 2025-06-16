// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {LogEmitter} from "../src/LogEmitter.sol";

contract IncrementScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        LogEmitter logEmitter = new LogEmitter();
        logEmitter.emitZeroTopicEmptyData();
        logEmitter.emitZeroTopicNonEmptyData();
        logEmitter.emitOneTopicEmptyData();
        logEmitter.emitOneTopicNonEmptyData();
        logEmitter.emitTwoTopics();
        logEmitter.emitTwoTopicsExtraData();
        logEmitter.emitThreeTopics();
        logEmitter.emitThreeTopicsExtraData();
        logEmitter.emitFourTopics();
        logEmitter.emitFourTopicsExtraData();
        logEmitter.emitZeroTopicNonEmptyDataAndMemExpansion();
        logEmitter.emitOneTopicNonEmptyDataAndMemExpansion();
        logEmitter.emitTwoTopicsExtraDataAndMemExpansion();
        logEmitter.emitThreeTopicsExtraDataAndMemExpansion();
        logEmitter.emitFourTopicsExtraDataAndMemExpansion();
        vm.stopBroadcast();
    }
}
