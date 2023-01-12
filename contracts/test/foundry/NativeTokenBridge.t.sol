// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/NativeTokenBridge.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

contract NativeTokenBridgeTest is Test {
    NativeTokenBridge public bridge;
    IERC20 public nativeToken;

    address public zero = address(0);
    address public msgSender = address(100);

    address public rollup = address(1000);
    address public inbox = address(1001);

    // msg details
    uint8 kind = 7;
    bytes32 messageDataHash = keccak256(abi.encodePacked("some msg"));
    uint256 tokenFeeAmount = 30;

    function setUp() public {
        // deploy token and bridge
        nativeToken = new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this));
        bridge = NativeTokenBridge(TestUtil.deployProxy(address(new NativeTokenBridge())));

        // init bridge
        bridge.initialize(IOwnable(rollup), address(nativeToken));

        // fund msgSender account
        nativeToken.transfer(msgSender, 1_000);
    }

    function testInitialization() public {
        assertEq(bridge.nativeToken(), address(nativeToken), "Invalid nativeToken ref");
        assertEq(address(bridge.rollup()), rollup, "Invalid rollup ref");
        assertEq(bridge.activeOutbox(), zero, "Invalid activeOutbox ref");
    }

    function testSetDelayedInbox() public {
        assertEq(bridge.allowedDelayedInboxes(inbox), false, "Inbox shouldn't be allowed");

        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);
        assertEq(bridge.allowedDelayedInboxes(inbox), true, "Inbox should be allowed");
    }

    function testEnqueueDelayedMessage() public {
        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 msgSenderTokenBalanceBefore = nativeToken.balanceOf(address(msgSender));
        uint256 delayedMsgCountBefore = bridge.delayedMessageCount();

        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // approve bridge to escrow tokens
        vm.prank(msgSender);
        nativeToken.approve(address(bridge), tokenFeeAmount);

        // enqueue msg
        vm.prank(inbox);
        bridge.enqueueDelayedMessage(kind, msgSender, messageDataHash, tokenFeeAmount);

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceAfter - bridgeTokenBalanceBefore,
            tokenFeeAmount,
            "Invalid bridge token balance"
        );

        uint256 msgSenderTokenBalanceAfter = nativeToken.balanceOf(address(msgSender));
        assertEq(
            msgSenderTokenBalanceBefore - msgSenderTokenBalanceAfter,
            tokenFeeAmount,
            "Invalid msgSender token balance"
        );

        uint256 delayedMsgCountAfter = bridge.delayedMessageCount();
        assertEq(delayedMsgCountAfter - delayedMsgCountBefore, 1, "Invalid delayed message count");
    }

    function testCantUseEthForFees() public {
        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // enqueue msg
        hoax(inbox);
        vm.expectRevert(NotApplicable.selector);
        bridge.enqueueDelayedMessage{value: 0.1 ether}(kind, msgSender, messageDataHash);
    }

    function testCantEnqueueMsgFromUnregisteredInbox() public {
        vm.prank(inbox);
        vm.expectRevert(abi.encodeWithSelector(NotDelayedInbox.selector, inbox));
        bridge.enqueueDelayedMessage(kind, msgSender, messageDataHash, tokenFeeAmount);
    }
}
