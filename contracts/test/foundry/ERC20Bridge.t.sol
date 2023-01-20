// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "./AbsBridge.t.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/ERC20Inbox.sol";
import "../../src/bridge/IEthBridge.sol";
import "../../src/libraries/AddressAliasHelper.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

import "forge-std/console.sol";

contract ERC20BridgeTest is AbsBridgeTest {
    IERC20Bridge public erc20Bridge;
    IERC20 public nativeToken;

    // msg details
    uint8 public kind = 7;
    bytes32 public messageDataHash = keccak256(abi.encodePacked("some msg"));
    uint256 public tokenFeeAmount = 30;

    function setUp() public {
        // deploy token and bridge
        nativeToken = new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this));
        bridge = ERC20Bridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        erc20Bridge = IERC20Bridge(address(bridge));

        // init bridge
        erc20Bridge.initialize(IOwnable(rollup), address(nativeToken));

        // fund user account
        nativeToken.transfer(user, 1_000);
    }

    /* solhint-disable func-name-mixedcase */
    function test_initialize() public {
        assertEq(
            address(erc20Bridge.nativeToken()),
            address(nativeToken),
            "Invalid nativeToken ref"
        );
        assertEq(address(bridge.rollup()), rollup, "Invalid rollup ref");
        assertEq(bridge.activeOutbox(), address(0), "Invalid activeOutbox ref");
    }

    function test_initialize_revert_ZeroAddressToken() public {
        IERC20Bridge noTokenBridge = ERC20Bridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        vm.expectRevert(InvalidToken.selector);
        noTokenBridge.initialize(IOwnable(rollup), address(0));
    }

    function test_initialize_revert_ReInit() public {
        vm.expectRevert("Initializable: contract is already initialized");
        erc20Bridge.initialize(IOwnable(rollup), address(nativeToken));
    }

    function test_initialize_revert_NonDelegated() public {
        IERC20Bridge noTokenBridge = new ERC20Bridge();
        vm.expectRevert("Function must be called through delegatecall");
        noTokenBridge.initialize(IOwnable(rollup), address(nativeToken));
    }

    function test_enqueueDelayedMessage() public {
        uint256 bridgeNativeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
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
        erc20Bridge.enqueueDelayedMessage(kind, userAliased, messageDataHash, tokenFeeAmount);

        //// checks

        uint256 bridgeNativeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeNativeTokenBalanceAfter - bridgeNativeTokenBalanceBefore,
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

    function test_enqueueDelayedMessage_revert_UseEthForFees() public {
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

    function test_enqueueDelayedMessage_revert_NotDelayedInbox() public {
        vm.prank(inbox);
        vm.expectRevert(abi.encodeWithSelector(NotDelayedInbox.selector, inbox));
        erc20Bridge.enqueueDelayedMessage(kind, user, messageDataHash, tokenFeeAmount);
    }

    function test_executeCall_EmptyCalldata() public {
        // fund bridge with some tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        uint256 bridgeNativeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        //// execute call
        vm.prank(outbox);
        uint256 withdrawalAmount = 15;
        (bool success, ) = bridge.executeCall({to: user, value: withdrawalAmount, data: ""});

        //// checks
        assertTrue(success, "Execute call failed");

        uint256 bridgeNativeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeNativeTokenBalanceBefore - bridgeNativeTokenBalanceAfter,
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

    function test_executeCall_ExtraCall() public {
        // fund bridge with native tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // deploy some contract that will be call receiver
        IERC20 callReceiver = new ERC20PresetFixedSupply("Rando", "R", 1_000_000, address(this));
        callReceiver.transfer(address(bridge), 3_000);

        uint256 bridgeNativeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));

        address tokenReceiver = address(500);
        uint256 callReceiverNativeTokenBalanceBefore = nativeToken.balanceOf(address(callReceiver));
        uint256 tokenReceiverBalanceBefore = callReceiver.balanceOf(tokenReceiver);

        //// execute call
        vm.prank(outbox);
        uint256 withdrawalAmount = 15;
        uint256 tokenMoveAmount = 3_000;
        (bool success, ) = bridge.executeCall({
            to: address(callReceiver),
            value: withdrawalAmount,
            data: abi.encodeWithSelector(IERC20.transfer.selector, tokenReceiver, tokenMoveAmount)
        });

        //// checks
        assertTrue(success, "Execute call failed");

        uint256 tokenReceiverBalanceAfter = callReceiver.balanceOf(tokenReceiver);
        assertEq(
            tokenReceiverBalanceAfter - tokenReceiverBalanceBefore,
            tokenMoveAmount,
            "Invalid receiver token balance"
        );

        uint256 bridgeNativeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeNativeTokenBalanceBefore - bridgeNativeTokenBalanceAfter,
            withdrawalAmount,
            "Invalid bridge native token balance"
        );

        uint256 callReceiverNativeTokenBalanceAfter = nativeToken.balanceOf(address(callReceiver));
        assertEq(
            callReceiverNativeTokenBalanceAfter - callReceiverNativeTokenBalanceBefore,
            withdrawalAmount,
            "Invalid tokenReceiver native token balance"
        );
    }

    function test_executeCall_revert_FailExtraCall() public {
        // fund bridge with native tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // deploy some contract that will be call receiver
        IERC20 callReceiver = new ERC20PresetFixedSupply("Rando", "R", 1_000_000, address(this));
        callReceiver.transfer(address(bridge), 3_000);

        uint256 bridgeNativeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));

        address tokenReceiver = address(500);
        uint256 callReceiverNativeTokenBalanceBefore = nativeToken.balanceOf(address(callReceiver));
        uint256 tokenReceiverBalanceBefore = callReceiver.balanceOf(tokenReceiver);

        //// execute call - extra call shall be unsuccessful due to too high token amount
        vm.prank(outbox);
        uint256 withdrawalAmount = 15;
        uint256 invalidTokenMoveAmount = 100_000;
        (bool success, ) = bridge.executeCall({
            to: address(callReceiver),
            value: withdrawalAmount,
            data: abi.encodeWithSelector(
                IERC20.transfer.selector,
                tokenReceiver,
                invalidTokenMoveAmount
            )
        });

        //// checks
        assertEq(success, false, "Execute shall be unsuccessful");

        uint256 tokenReceiverBalanceAfter = callReceiver.balanceOf(tokenReceiver);
        assertEq(
            tokenReceiverBalanceAfter,
            tokenReceiverBalanceBefore,
            "Invalid receiver token balance after unsuccessful extra call"
        );

        // bridge successfully sent native token even though extra call was unsuccessful (we didn't revert it)
        uint256 bridgeNativeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeNativeTokenBalanceBefore - bridgeNativeTokenBalanceAfter,
            withdrawalAmount,
            "Invalid bridge native token balance after unsuccessful extra call"
        );

        // bridge successfully recieved native token even though extra call was unsuccessful (we didn't revert it)
        uint256 callReceiverNativeTokenBalanceAfter = nativeToken.balanceOf(address(callReceiver));
        assertEq(
            callReceiverNativeTokenBalanceAfter - callReceiverNativeTokenBalanceBefore,
            withdrawalAmount,
            "Invalid tokenReceiver native token balance after unsuccessful call"
        );
    }

    function test_executeCall_revert_NotOutbox() public {
        vm.expectRevert(abi.encodeWithSelector(NotOutbox.selector, address(this)));
        bridge.executeCall({to: user, value: 10, data: ""});
    }

    function test_executeCall_revert_NotContract() public {
        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // executeCall shall revert when 'to' is not contract
        address to = address(234);
        vm.expectRevert(abi.encodeWithSelector(NotContract.selector, address(to)));
        vm.prank(outbox);
        bridge.executeCall({to: to, value: 10, data: "some data"});
    }
}
