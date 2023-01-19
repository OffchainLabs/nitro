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
    function test_submitBatchSpendingReport() public {
        address sender = address(250);
        bytes32 messageDataHash = keccak256(abi.encode("msg"));

        // expect event
        vm.expectEmit(true, true, true, true);
        emit MessageDelivered(
            0,
            0,
            seqInbox,
            13,
            sender,
            messageDataHash,
            block.basefee,
            uint64(block.timestamp)
        );

        // submit report
        vm.prank(rollup);
        bridge.setSequencerInbox(seqInbox);
        vm.prank(seqInbox);
        uint256 count = bridge.submitBatchSpendingReport(sender, messageDataHash);

        // checks
        assertEq(count, 0, "Invalid count");
        assertEq(bridge.delayedMessageCount(), 1, "Invalid msg count");
    }

    function test_submitBatchSpendingReport_TwoInRow() public {
        address sender = address(250);
        bytes32 messageDataHash = keccak256(abi.encode("msg"));

        // submit 1st report
        vm.prank(rollup);
        bridge.setSequencerInbox(seqInbox);
        vm.prank(seqInbox);
        bridge.submitBatchSpendingReport(sender, messageDataHash);

        // expect event
        vm.expectEmit(true, true, true, true);
        emit MessageDelivered(
            1,
            bridge.delayedInboxAccs(0),
            seqInbox,
            13,
            sender,
            messageDataHash,
            block.basefee,
            uint64(block.timestamp)
        );

        // submit 2nd report
        vm.prank(seqInbox);
        uint256 count = bridge.submitBatchSpendingReport(sender, messageDataHash);

        // checks
        assertEq(count, 1, "Invalid count");
        assertEq(bridge.delayedMessageCount(), 2, "Invalid msg count");
    }

    function test_submitBatchSpendingReport_revert_NonSeqInboxCall() public {
        // submitBatchSpendingReport shall revert
        vm.expectRevert(abi.encodeWithSelector(NotSequencerInbox.selector, address(this)));
        bridge.submitBatchSpendingReport(address(2), keccak256("msg"));
    }

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

    function test_setSequencerInbox_revert_NonOwnerCall() public {
        // mock the owner() call on rollup
        address mockRollupOwner = address(10000);
        vm.mockCall(
            rollup,
            abi.encodeWithSelector(IOwnable.owner.selector),
            abi.encode(mockRollupOwner)
        );

        // setSequencerInbox shall revert
        vm.expectRevert(
            abi.encodeWithSelector(
                NotRollupOrOwner.selector,
                address(this),
                rollup,
                mockRollupOwner
            )
        );
        bridge.setSequencerInbox(seqInbox);
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
        // mock the owner() call on rollup
        address mockRollupOwner = address(10000);
        vm.mockCall(
            rollup,
            abi.encodeWithSelector(IOwnable.owner.selector),
            abi.encode(mockRollupOwner)
        );

        // setDelayedInbox shall revert
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

    function test_setOutbox_EnableOutbox() public {
        assertEq(bridge.allowedOutboxes(outbox), false, "Invalid allowedOutboxes");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit OutboxToggle(outbox, true);

        // enable outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // checks
        assertEq(bridge.allowedOutboxes(outbox), true, "Invalid allowedOutboxes");
        assertEq(outbox, bridge.allowedOutboxList(0), "Invalid allowedOutboxList");
    }

    function test_setOutbox_DisableOutbox() public {
        // initially enable outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);
        assertEq(bridge.allowedOutboxes(outbox), true, "Invalid allowedOutboxes");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit OutboxToggle(outbox, false);

        //  disable outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, false);

        // checks
        assertEq(bridge.allowedOutboxes(outbox), false, "Invalid allowedOutboxes");
        vm.expectRevert();
        bridge.allowedOutboxList(0);
    }

    function test_setOutbox_ReEnableOutbox() public {
        // initially enable outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);
        assertEq(bridge.allowedOutboxes(outbox), true, "Invalid allowedOutboxes");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit OutboxToggle(outbox, true);

        // enable outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // checks
        assertEq(bridge.allowedOutboxes(outbox), true, "Invalid allowedOutboxes");
        assertEq(outbox, bridge.allowedOutboxList(0), "Invalid allowedOutboxList");
    }

    function test_setOutbox_ReDisableOutbox() public {
        // expect event
        vm.expectEmit(true, true, true, true);
        emit OutboxToggle(outbox, false);

        //  disable outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, false);

        // checks
        assertEq(bridge.allowedOutboxes(outbox), false, "Invalid allowedOutboxes");
        vm.expectRevert();
        bridge.allowedOutboxList(0);
    }

    function test_setOutbox_revert_NonOwnerCall() public {
        // mock the owner() call on rollup
        address mockRollupOwner = address(10000);
        vm.mockCall(
            rollup,
            abi.encodeWithSelector(IOwnable.owner.selector),
            abi.encode(mockRollupOwner)
        );

        // setOutbox shall revert
        vm.expectRevert(
            abi.encodeWithSelector(
                NotRollupOrOwner.selector,
                address(this),
                rollup,
                mockRollupOwner
            )
        );
        bridge.setOutbox(outbox, true);
    }

    function test_setOutbox_revert_InvalidOutboxSet() public {
        address invalidOutbox = address(type(uint160).max);

        // setOutbox shall revert
        vm.expectRevert(abi.encodeWithSelector(InvalidOutboxSet.selector, invalidOutbox));
        vm.prank(rollup);
        bridge.setOutbox(invalidOutbox, true);
    }

    function test_setSequencerReportedSubMessageCount() public {
        uint256 newCount = 1234;

        vm.prank(rollup);
        AbsBridge(address(bridge)).setSequencerReportedSubMessageCount(newCount);

        assertEq(
            bridge.sequencerReportedSubMessageCount(),
            newCount,
            "Invalid sequencerReportedSubMessageCount"
        );
    }

    function test_setSequencerReportedSubMessageCount_revert_NonOwnerCall() public {
        // mock the owner() call on rollup
        address mockRollupOwner = address(10000);
        vm.mockCall(
            rollup,
            abi.encodeWithSelector(IOwnable.owner.selector),
            abi.encode(mockRollupOwner)
        );

        // setOutbox shall revert
        vm.expectRevert(
            abi.encodeWithSelector(
                NotRollupOrOwner.selector,
                address(this),
                rollup,
                mockRollupOwner
            )
        );
        AbsBridge(address(bridge)).setSequencerReportedSubMessageCount(123);
    }

    /****
     **** Event declarations
     ***/

    event SequencerInboxUpdated(address newSequencerInbox);
    event InboxToggle(address indexed inbox, bool enabled);
    event OutboxToggle(address indexed outbox, bool enabled);
    event MessageDelivered(
        uint256 indexed messageIndex,
        bytes32 indexed beforeInboxAcc,
        address inbox,
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 baseFeeL1,
        uint64 timestamp
    );
}
