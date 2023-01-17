// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/NativeTokenBridge.sol";
import "../../src/bridge/NativeTokenInbox.sol";
import "../../src/bridge/IEthBridge.sol";
import "../../src/libraries/AddressAliasHelper.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

import "forge-std/console.sol";

contract NativeTokenBridgeTest is Test {
    NativeTokenBridge public bridge;
    IERC20 public nativeToken;

    address public user = address(100);
    address public userB = address(101);

    address public rollup = address(1000);
    address public inbox = address(1001);
    address public outbox = address(1002);

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

        // fund user account
        nativeToken.transfer(user, 1_000);
    }

    function testInitialization() public {
        assertEq(bridge.nativeToken(), address(nativeToken), "Invalid nativeToken ref");
        assertEq(address(bridge.rollup()), rollup, "Invalid rollup ref");
        assertEq(bridge.activeOutbox(), address(0), "Invalid activeOutbox ref");
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
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));
        uint256 delayedMsgCountBefore = bridge.delayedMessageCount();

        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), tokenFeeAmount);

        // enqueue msg
        address userAliased = AddressAliasHelper.applyL1ToL2Alias(user);
        vm.prank(inbox);
        bridge.enqueueDelayedMessage(kind, userAliased, messageDataHash, tokenFeeAmount);

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceAfter - bridgeTokenBalanceBefore,
            tokenFeeAmount,
            "Invalid bridge token balance"
        );

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(
            userTokenBalanceBefore - userTokenBalanceAfter,
            tokenFeeAmount,
            "Invalid user token balance"
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
        vm.expectRevert();
        IEthBridge(address(bridge)).enqueueDelayedMessage{value: 0.1 ether}(
            kind,
            user,
            messageDataHash
        );
    }

    function testCantEnqueueMsgFromUnregisteredInbox() public {
        vm.prank(inbox);
        vm.expectRevert(abi.encodeWithSelector(NotDelayedInbox.selector, inbox));
        bridge.enqueueDelayedMessage(kind, user, messageDataHash, tokenFeeAmount);
    }

    function testExecuteCallNoData() public {
        // fund bridge with some tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        //// execute call
        vm.prank(outbox);
        uint256 withdrawalAmount = 15;
        bridge.executeCall({to: user, value: withdrawalAmount, data: ""});

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceBefore - bridgeTokenBalanceAfter,
            withdrawalAmount,
            "Invalid bridge token balance"
        );

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(
            userTokenBalanceAfter - userTokenBalanceBefore,
            withdrawalAmount,
            "Invalid user token balance"
        );
    }
}
