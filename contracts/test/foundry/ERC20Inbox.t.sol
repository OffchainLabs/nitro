// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/ERC20Inbox.sol";
import "../../src/bridge/ISequencerInbox.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

contract ERC20InboxTest is Test {
    ERC20Inbox public inbox;
    ERC20Bridge public bridge;
    IERC20 public nativeToken;

    address public user = address(100);
    address public rollup = address(1000);
    address public seqInbox = address(1001);

    function setUp() public {
        // deploy token, bridge and inbox
        nativeToken = new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this));
        bridge = ERC20Bridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        inbox = ERC20Inbox(TestUtil.deployProxy(address(new ERC20Inbox())));

        // init bridge and inbox
        bridge.initialize(IOwnable(rollup), address(nativeToken));
        inbox.initialize(bridge, ISequencerInbox(seqInbox));
        vm.prank(rollup);
        bridge.setDelayedInbox(address(inbox), true);

        // fund user account
        nativeToken.transfer(user, 1_000);
    }

    /* solhint-disable func-name-mixedcase */
    function test_initialize() public {
        assertEq(address(inbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(inbox.sequencerInbox()), seqInbox, "Invalid seqInbox ref");
        assertEq(inbox.allowListEnabled(), false, "Invalid allowListEnabled");
    }

    function test_depositERC20() public {
        uint256 depositAmount = 300;

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));
        uint256 delayedMsgCountBefore = bridge.delayedMessageCount();

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), depositAmount);

        // deposit tokens
        vm.prank(user);
        inbox.depositERC20(depositAmount);

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceAfter - bridgeTokenBalanceBefore,
            depositAmount,
            "Invalid bridge token balance"
        );

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(
            userTokenBalanceBefore - userTokenBalanceAfter,
            depositAmount,
            "Invalid user token balance"
        );

        uint256 delayedMsgCountAfter = bridge.delayedMessageCount();
        assertEq(delayedMsgCountAfter - delayedMsgCountBefore, 1, "Invalid delayed message count");
    }

    function test_createRetryableTicket() public {
        uint256 tokenTotalFeeAmount = 300;

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));
        uint256 delayedMsgCountBefore = bridge.delayedMessageCount();

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), tokenTotalFeeAmount);

        // create retryable
        vm.prank(user);
        inbox.createRetryableTicket({
            to: address(user),
            l2CallValue: 10,
            maxSubmissionCost: 0,
            excessFeeRefundAddress: address(user),
            callValueRefundAddress: address(user),
            gasLimit: 100,
            maxFeePerGas: 2,
            tokenTotalFeeAmount: tokenTotalFeeAmount,
            data: abi.encodePacked("some msg")
        });

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceAfter - bridgeTokenBalanceBefore,
            tokenTotalFeeAmount,
            "Invalid bridge token balance"
        );

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(
            userTokenBalanceBefore - userTokenBalanceAfter,
            tokenTotalFeeAmount,
            "Invalid user token balance"
        );

        uint256 delayedMsgCountAfter = bridge.delayedMessageCount();
        assertEq(delayedMsgCountAfter - delayedMsgCountBefore, 1, "Invalid delayed message count");
    }
}
