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

    /* solhint-disable func-name-mixedcase */
    function test_setSequencerInbox() public {
        // expect event
        vm.expectEmit(true, true, true, true);
        emit SequencerInboxUpdated(seqInbox);

        // set seqInbox
        vm.prank(address(bridge.rollup()));
        bridge.setSequencerInbox(seqInbox);

        // checks
        assertEq(bridge.sequencerInbox(), seqInbox, "Invalid seqInbox");
    }

    function test_setDelayedInbox_enableInbox() public {
        assertEq(bridge.allowedDelayedInboxes(inbox), false, "Invalid allowedDelayedInboxes");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxToggle(inbox, true);

        // enable inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // checks
        assertEq(bridge.allowedDelayedInboxes(inbox), true, "Invalid allowedDelayedInboxes");
        assertEq(inbox, bridge.allowedDelayedInboxList(0), "Invalid allowedDelayedInboxList");
    }

    function test_setDelayedInbox_disableInbox() public {
        // initially enable inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);
        assertEq(bridge.allowedDelayedInboxes(inbox), true, "Invalid allowedDelayedInboxes");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxToggle(inbox, false);

        // disable inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, false);

        // checks
        assertEq(bridge.allowedDelayedInboxes(inbox), false, "Invalid allowedDelayedInboxes");
        vm.expectRevert();
        bridge.allowedDelayedInboxList(0);
    }

    function test_setDelayedInbox_ReEnableInbox() public {
        // initially enable inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);
        assertEq(bridge.allowedDelayedInboxes(inbox), true, "Invalid allowedDelayedInboxes");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxToggle(inbox, true);

        // enable again inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // checks
        assertEq(bridge.allowedDelayedInboxes(inbox), true, "Invalid allowedDelayedInboxes");
        assertEq(inbox, bridge.allowedDelayedInboxList(0), "Invalid allowedDelayedInboxList");
    }

    function test_setDelayedInbox_ReDisableInbox() public {
        assertEq(bridge.allowedDelayedInboxes(inbox), false, "Invalid allowedDelayedInboxes");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxToggle(inbox, false);

        // disable again inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, false);

        // checks
        assertEq(bridge.allowedDelayedInboxes(inbox), false, "Invalid allowedDelayedInboxes");
        vm.expectRevert();
        bridge.allowedDelayedInboxList(0);
    }

    function test_setDelayedInbox_revert_NonOwnerCall() public {
        address mockRollupOwner = address(10000);
        vm.mockCall(
            rollup,
            abi.encodeWithSelector(IOwnable.owner.selector),
            abi.encode(mockRollupOwner)
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                NotRollupOrOwner.selector,
                address(this),
                rollup,
                mockRollupOwner
            )
        );
        bridge.setDelayedInbox(inbox, true);
    }

    /****
     **** Event declarations
     ***/

    event SequencerInboxUpdated(address newSequencerInbox);
    event InboxToggle(address indexed inbox, bool enabled);
}
