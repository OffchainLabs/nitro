// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {Creator, Createe} from "../src/Create.sol";
import {CreatorTwo} from "../src/Create2.sol";

contract CreateTestScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        Creator creator = new Creator();
        payable(address(creator)).transfer(1 ether);
        CreatorTwo creator2 = new CreatorTwo();
        payable(address(creator2)).transfer(1 ether);
        address createe = creator.createNoTransferMemUnchanged();
        console.log("createe", createe);
        address createe2 = creator2.createTwoNoTransferMemUnchanged(bytes32(uint256(0x1337)));
        console.log("createe2", createe2);
        address createe3 = creator.createPayableMemUnchanged();
        console.log("createe3", createe3);
        address createe4 = creator2.createTwoPayableMemUnchanged(bytes32(uint256(0x1339)));
        console.log("createe4", createe4);
        creator.createNoTransferMemExpansion();
        creator2.createTwoNoTransferMemExpansion(bytes32(uint256(0x1339)));
        creator.createPayableMemExpansion();
        creator2.createTwoPayableMemExpansion(bytes32(uint256(0x1339)));
        vm.stopBroadcast();
    }
}
