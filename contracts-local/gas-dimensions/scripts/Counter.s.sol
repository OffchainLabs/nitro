// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {Counter} from "../src/Counter.sol";
import {Cloner} from "../src/Cloner.sol";

contract CounterScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        Counter counterImpl = new Counter();
        console.log("counterImpl", address(counterImpl));
        Cloner cloner = new Cloner(address(counterImpl));
        console.log("cloner", address(cloner));

        address counter1Addr = cloner.createCounter();
        address counter2Addr = cloner.create2Counter(bytes32(uint256(1)));

        console.log("counter1 addr", counter1Addr);
        console.log("counter2 addr", counter2Addr);

        Counter counter1 = Counter(counter1Addr);
        counter1.setNumber(1);
        console.log("counter1 set number", counter1.number(), " on ", counter1Addr);
        counter1.increment();
        console.log("counter1 incremented", counter1.number(), " on ", counter1Addr);
        vm.stopBroadcast();
    }
}
