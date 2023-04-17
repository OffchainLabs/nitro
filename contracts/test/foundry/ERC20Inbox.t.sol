// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "./AbsInbox.t.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/ERC20Inbox.sol";
import "../../src/bridge/ISequencerInbox.sol";
import "../../src/libraries/AddressAliasHelper.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

contract ERC20InboxTest is AbsInboxTest {
    IERC20 public nativeToken;
    IERC20Inbox public erc20Inbox;

    function setUp() public {
        // deploy token, bridge and inbox
        nativeToken = new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this));
        bridge = IBridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        inbox = IInbox(TestUtil.deployProxy(address(new ERC20Inbox())));
        erc20Inbox = IERC20Inbox(address(inbox));

        // init bridge and inbox
        IERC20Bridge(address(bridge)).initialize(IOwnable(rollup), address(nativeToken));
        inbox.initialize(bridge, ISequencerInbox(seqInbox));
        vm.prank(rollup);
        bridge.setDelayedInbox(address(inbox), true);

        // fund user account
        nativeToken.transfer(user, 1_000);
    }

    /* solhint-disable func-name-mixedcase */
    function test_depositERC20_FromEOA() public {
        uint256 depositAmount = 300;

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));
        uint256 delayedMsgCountBefore = bridge.delayedMessageCount();

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), depositAmount);

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(0, abi.encodePacked(user, depositAmount));

        // deposit tokens -> tx.origin == msg.sender
        vm.prank(user, user);
        erc20Inbox.depositERC20(depositAmount);

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

    function test_depositERC20_FromContract() public {
        uint256 depositAmount = 300;

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));
        uint256 delayedMsgCountBefore = bridge.delayedMessageCount();

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), depositAmount);

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(
            0,
            abi.encodePacked(AddressAliasHelper.applyL1ToL2Alias(user), depositAmount)
        );

        // deposit tokens -> tx.origin != msg.sender
        vm.prank(user);
        erc20Inbox.depositERC20(depositAmount);

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

    function test_depositERC20_revert_NativeTokenTransferFails() public {
        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        // deposit tokens
        vm.prank(user);
        uint256 invalidDepositAmount = 1_000_000;
        vm.expectRevert("ERC20: insufficient allowance");
        erc20Inbox.depositERC20(invalidDepositAmount);

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(bridgeTokenBalanceAfter, bridgeTokenBalanceBefore, "Invalid bridge token balance");

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(userTokenBalanceBefore, userTokenBalanceAfter, "Invalid user token balance");

        assertEq(bridge.delayedMessageCount(), 0, "Invalid delayed message count");
    }

    function test_createRetryableTicket_FromEOA() public {
        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        uint256 tokenTotalFeeAmount = 300;

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), tokenTotalFeeAmount);

        // retyrable params
        uint256 l2CallValue = 10;
        uint256 maxSubmissionCost = 0;
        uint256 gasLimit = 100;
        uint256 maxFeePerGas = 2;
        bytes memory data = abi.encodePacked("some msg");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(
            0,
            abi.encodePacked(
                uint256(uint160(user)),
                l2CallValue,
                tokenTotalFeeAmount,
                maxSubmissionCost,
                uint256(uint160(user)),
                uint256(uint160(user)),
                gasLimit,
                maxFeePerGas,
                data.length,
                data
            )
        );

        // create retryable -> tx.origin == msg.sender
        vm.prank(user, user);
        erc20Inbox.createRetryableTicket({
            to: address(user),
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tokenTotalFeeAmount,
            data: data
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

        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_createRetryableTicket_FromContract() public {
        address sender = address(new Sender());
        nativeToken.transfer(address(sender), 1_000);

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 senderTokenBalanceBefore = nativeToken.balanceOf(address(sender));

        uint256 tokenTotalFeeAmount = 300;

        // approve bridge to escrow tokens
        vm.prank(sender);
        nativeToken.approve(address(bridge), tokenTotalFeeAmount);

        // retyrable params
        uint256 l2CallValue = 10;
        uint256 maxSubmissionCost = 0;
        uint256 gasLimit = 100;
        uint256 maxFeePerGas = 2;
        bytes memory data = abi.encodePacked("some msg");

        // expect event
        uint256 uintAlias = uint256(uint160(AddressAliasHelper.applyL1ToL2Alias(sender)));
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(
            0,
            abi.encodePacked(
                uint256(uint160(sender)),
                l2CallValue,
                tokenTotalFeeAmount,
                maxSubmissionCost,
                uintAlias,
                uintAlias,
                gasLimit,
                maxFeePerGas,
                data.length,
                data
            )
        );

        // create retryable
        vm.prank(sender);
        erc20Inbox.createRetryableTicket({
            to: sender,
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: sender,
            callValueRefundAddress: sender,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tokenTotalFeeAmount,
            data: data
        });

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceAfter - bridgeTokenBalanceBefore,
            tokenTotalFeeAmount,
            "Invalid bridge token balance"
        );

        uint256 senderTokenBalanceAfter = nativeToken.balanceOf(sender);
        assertEq(
            senderTokenBalanceBefore - senderTokenBalanceAfter,
            tokenTotalFeeAmount,
            "Invalid sender token balance"
        );

        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_createRetryableTicket_revert_WhenPaused() public {
        vm.prank(rollup);
        inbox.pause();

        vm.expectRevert("Pausable: paused");
        erc20Inbox.createRetryableTicket({
            to: user,
            l2CallValue: 100,
            maxSubmissionCost: 0,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: 10,
            maxFeePerGas: 1,
            tokenTotalFeeAmount: 200,
            data: abi.encodePacked("data")
        });
    }

    function test_createRetryableTicket_revert_OnlyAllowed() public {
        vm.prank(rollup);
        inbox.setAllowListEnabled(true);

        vm.prank(user, user);
        vm.expectRevert(abi.encodeWithSelector(NotAllowedOrigin.selector, user));
        erc20Inbox.createRetryableTicket({
            to: user,
            l2CallValue: 100,
            maxSubmissionCost: 0,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: 10,
            maxFeePerGas: 1,
            tokenTotalFeeAmount: 200,
            data: abi.encodePacked("data")
        });
    }

    function test_createRetryableTicket_revert_InsufficientValue() public {
        uint256 tooSmallTokenTotalFeeAmount = 3;
        uint256 l2CallValue = 100;
        uint256 maxSubmissionCost = 0;
        uint256 gasLimit = 10;
        uint256 maxFeePerGas = 1;

        vm.prank(user, user);
        vm.expectRevert(
            abi.encodeWithSelector(
                InsufficientValue.selector,
                maxSubmissionCost + l2CallValue + gasLimit * maxFeePerGas,
                tooSmallTokenTotalFeeAmount
            )
        );
        erc20Inbox.createRetryableTicket({
            to: user,
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tooSmallTokenTotalFeeAmount,
            data: abi.encodePacked("data")
        });
    }

    function test_createRetryableTicket_revert_RetryableDataTracer() public {
        uint256 tokenTotalFeeAmount = 300;
        uint256 l2CallValue = 100;
        uint256 maxSubmissionCost = 0;
        uint256 gasLimit = 10;
        uint256 maxFeePerGas = 1;
        bytes memory data = abi.encodePacked("xy");

        // revert as maxFeePerGas == 1 is magic value
        vm.prank(user, user);
        vm.expectRevert(
            abi.encodeWithSelector(
                RetryableData.selector,
                user,
                user,
                l2CallValue,
                tokenTotalFeeAmount,
                maxSubmissionCost,
                user,
                user,
                gasLimit,
                maxFeePerGas,
                data
            )
        );
        erc20Inbox.createRetryableTicket({
            to: user,
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tokenTotalFeeAmount,
            data: data
        });

        gasLimit = 1;
        maxFeePerGas = 2;

        // revert as gasLimit == 1 is magic value
        vm.prank(user, user);
        vm.expectRevert(
            abi.encodeWithSelector(
                RetryableData.selector,
                user,
                user,
                l2CallValue,
                tokenTotalFeeAmount,
                maxSubmissionCost,
                user,
                user,
                gasLimit,
                maxFeePerGas,
                data
            )
        );
        erc20Inbox.createRetryableTicket({
            to: user,
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tokenTotalFeeAmount,
            data: data
        });
    }

    function test_createRetryableTicket_revert_GasLimitTooLarge() public {
        uint256 tooBigGasLimit = uint256(type(uint64).max) + 1;

        vm.prank(user, user);
        vm.expectRevert(GasLimitTooLarge.selector);
        erc20Inbox.createRetryableTicket({
            to: user,
            l2CallValue: 100,
            maxSubmissionCost: 0,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: tooBigGasLimit,
            maxFeePerGas: 2,
            tokenTotalFeeAmount: uint256(type(uint64).max) * 3,
            data: abi.encodePacked("data")
        });
    }

    function test_unsafeCreateRetryableTicket_FromEOA() public {
        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        uint256 tokenTotalFeeAmount = 300;

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), tokenTotalFeeAmount);

        // retyrable params
        uint256 l2CallValue = 10;
        uint256 maxSubmissionCost = 0;
        uint256 gasLimit = 100;
        uint256 maxFeePerGas = 2;
        bytes memory data = abi.encodePacked("some msg");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(
            0,
            abi.encodePacked(
                uint256(uint160(user)),
                l2CallValue,
                tokenTotalFeeAmount,
                maxSubmissionCost,
                uint256(uint160(user)),
                uint256(uint160(user)),
                gasLimit,
                maxFeePerGas,
                data.length,
                data
            )
        );

        // create retryable -> tx.origin == msg.sender
        vm.prank(user, user);
        erc20Inbox.unsafeCreateRetryableTicket({
            to: address(user),
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tokenTotalFeeAmount,
            data: data
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

        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_unsafeCreateRetryableTicket_FromContract() public {
        address sender = address(new Sender());
        nativeToken.transfer(address(sender), 1_000);

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 senderTokenBalanceBefore = nativeToken.balanceOf(address(sender));

        uint256 tokenTotalFeeAmount = 300;

        // approve bridge to escrow tokens
        vm.prank(sender);
        nativeToken.approve(address(bridge), tokenTotalFeeAmount);

        // retyrable params
        uint256 l2CallValue = 10;
        uint256 maxSubmissionCost = 0;
        uint256 gasLimit = 100;
        uint256 maxFeePerGas = 2;
        bytes memory data = abi.encodePacked("some msg");

        // expect event (address shall not be aliased)
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(
            0,
            abi.encodePacked(
                uint256(uint160(sender)),
                l2CallValue,
                tokenTotalFeeAmount,
                maxSubmissionCost,
                uint256(uint160(sender)),
                uint256(uint160(sender)),
                gasLimit,
                maxFeePerGas,
                data.length,
                data
            )
        );

        // create retryable
        vm.prank(sender);
        erc20Inbox.unsafeCreateRetryableTicket({
            to: sender,
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: sender,
            callValueRefundAddress: sender,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tokenTotalFeeAmount,
            data: data
        });

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceAfter - bridgeTokenBalanceBefore,
            tokenTotalFeeAmount,
            "Invalid bridge token balance"
        );

        uint256 senderTokenBalanceAfter = nativeToken.balanceOf(sender);
        assertEq(
            senderTokenBalanceBefore - senderTokenBalanceAfter,
            tokenTotalFeeAmount,
            "Invalid sender token balance"
        );

        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_unsafeCreateRetryableTicket_NotRevertingOnInsufficientValue() public {
        uint256 tooSmallTokenTotalFeeAmount = 3;
        uint256 l2CallValue = 100;
        uint256 maxSubmissionCost = 0;
        uint256 gasLimit = 10;
        uint256 maxFeePerGas = 2;

        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), tooSmallTokenTotalFeeAmount);

        vm.prank(user, user);
        erc20Inbox.unsafeCreateRetryableTicket({
            to: user,
            l2CallValue: l2CallValue,
            maxSubmissionCost: maxSubmissionCost,
            excessFeeRefundAddress: user,
            callValueRefundAddress: user,
            gasLimit: gasLimit,
            maxFeePerGas: maxFeePerGas,
            tokenTotalFeeAmount: tooSmallTokenTotalFeeAmount,
            data: abi.encodePacked("data")
        });

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceAfter - bridgeTokenBalanceBefore,
            tooSmallTokenTotalFeeAmount,
            "Invalid bridge token balance"
        );

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(
            userTokenBalanceBefore - userTokenBalanceAfter,
            tooSmallTokenTotalFeeAmount,
            "Invalid user token balance"
        );

        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_calculateRetryableSubmissionFee() public {
        assertEq(inbox.calculateRetryableSubmissionFee(1, 2), 0, "Invalid ERC20 submission fee");
    }
}
