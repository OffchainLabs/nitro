// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/rollup/ERC20BridgeCreator.sol";
import "../../src/bridge/ISequencerInbox.sol";

contract ERC20BridgeCreatorTest is Test {
    ERC20BridgeCreator public creator;
    address public owner = address(100);

    function setUp() public {
        vm.prank(owner);
        creator = new ERC20BridgeCreator();
    }

    /* solhint-disable func-name-mixedcase */
    function test_constructor() public {
        assertTrue(address(creator.bridgeTemplate()) != address(0), "Bridge not created");
        assertTrue(address(creator.sequencerInboxTemplate()) != address(0), "SeqInbox not created");
        assertTrue(address(creator.inboxTemplate()) != address(0), "Inbox not created");
        assertTrue(
            address(creator.rollupEventInboxTemplate()) != address(0),
            "Event inbox not created"
        );
        assertTrue(address(creator.outboxTemplate()) != address(0), "Outbox not created");
    }

    function test_updateTemplates() public {
        address bridge = address(200);
        address sequencerInbox = address(201);
        address inbox = address(202);
        address rollupEventInbox = address(203);
        address outbox = address(204);

        vm.prank(owner);
        creator.updateTemplates(bridge, sequencerInbox, inbox, rollupEventInbox, outbox);

        assertEq(address(creator.bridgeTemplate()), bridge, "Invalid bridge");
        assertEq(address(creator.sequencerInboxTemplate()), sequencerInbox, "Invalid seqInbox");
        assertEq(address(creator.inboxTemplate()), inbox, "Invalid inbox");
        assertEq(
            address(creator.rollupEventInboxTemplate()),
            rollupEventInbox,
            "Invalid rollup event inbox"
        );
        assertEq(address(creator.outboxTemplate()), outbox, "Invalid outbox");
    }

    function test_createBridge() public {
        address proxyAdmin = address(300);
        address rollup = address(301);
        address nativeToken = address(302);
        ISequencerInbox.MaxTimeVariation memory timeVars = ISequencerInbox.MaxTimeVariation(
            10,
            20,
            30,
            40
        );
        timeVars.delayBlocks;

        (
            ERC20Bridge bridge,
            SequencerInbox seqInbox,
            ERC20Inbox inbox,
            RollupEventInbox eventInbox,
            Outbox outbox
        ) = creator.createBridge(proxyAdmin, rollup, nativeToken, timeVars);

        // bridge
        assertEq(address(bridge.rollup()), rollup, "Invalid rollup ref");
        assertEq(address(bridge.nativeToken()), nativeToken, "Invalid nativeToken ref");
        assertEq(bridge.activeOutbox(), address(0), "Invalid activeOutbox ref");

        // seqInbox
        assertEq(address(seqInbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(seqInbox.rollup()), rollup, "Invalid rollup ref");
        (
            uint256 _delayBlocks,
            uint256 _futureBlocks,
            uint256 _delaySeconds,
            uint256 _futureSeconds
        ) = seqInbox.maxTimeVariation();
        assertEq(_delayBlocks, timeVars.delayBlocks, "Invalid delayBlocks");
        assertEq(_futureBlocks, timeVars.futureBlocks, "Invalid futureBlocks");
        assertEq(_delaySeconds, timeVars.delaySeconds, "Invalid delaySeconds");
        assertEq(_futureSeconds, timeVars.futureSeconds, "Invalid futureSeconds");

        // inbox
        assertEq(address(inbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(inbox.sequencerInbox()), address(seqInbox), "Invalid seqInbox ref");
        assertEq(inbox.allowListEnabled(), false, "Invalid allowListEnabled");
        assertEq(inbox.paused(), false, "Invalid paused status");

        // rollup event inbox
        assertEq(address(eventInbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(eventInbox.rollup()), rollup, "Invalid rollup ref");

        // outbox
        assertEq(address(outbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(outbox.rollup()), rollup, "Invalid rollup ref");
    }
}
