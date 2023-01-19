// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/IBridge.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/Bridge.sol";
import "../../src/bridge/ERC20Inbox.sol";
import "../../src/bridge/IEthBridge.sol";
import "../../src/libraries/AddressAliasHelper.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

import "forge-std/console.sol";

abstract contract AbsBridgeTest is Test {
    IBridge public bridge;

    address public user = address(100);
    address public userB = address(101);

    address public rollup = address(1000);
    address public inbox = address(1001);
    address public outbox = address(1002);
    address public seqInbox = address(1003);

    function test_setSequencerInbox() public {
        vm.prank(address(bridge.rollup()));
        bridge.setSequencerInbox(seqInbox);

        assertEq(bridge.sequencerInbox(), seqInbox, "Invalid seqInbox");
    }

    function test_setDelayedInbox() public {
        assertEq(bridge.allowedDelayedInboxes(inbox), false, "Inbox shouldn't be allowed");

        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);
        assertEq(bridge.allowedDelayedInboxes(inbox), true, "Inbox should be allowed");
    }
}
