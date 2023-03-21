// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/IOutbox.sol";
import "../../src/bridge/IBridge.sol";

abstract contract AbsOutboxTest is Test {
    IOutbox public outbox;
    IBridge public bridge;

    address public user = address(100);
    address public rollup = address(1000);
    address public seqInbox = address(1001);

    /* solhint-disable func-name-mixedcase */
    function test_initialize() public {
        assertEq(address(outbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(outbox.rollup()), rollup, "Invalid rollup ref");

        assertEq(outbox.l2ToL1Sender(), address(0), "Invalid l2ToL1Sender");
        assertEq(outbox.l2ToL1Block(), 0, "Invalid l2ToL1Block");
        assertEq(outbox.l2ToL1EthBlock(), 0, "Invalid l2ToL1EthBlock");
        assertEq(outbox.l2ToL1Timestamp(), 0, "Invalid l2ToL1Timestamp");
        assertEq(outbox.l2ToL1OutputId(), bytes32(0), "Invalid l2ToL1OutputId");
        assertEq(outbox.l2ToL1WithdrawalAmount(), 0, "Invalid withdrawalAmount");
    }
}
