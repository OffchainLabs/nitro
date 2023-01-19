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
    uint8 kind = 7;
    bytes32 messageDataHash = keccak256(abi.encodePacked("some msg"));
    uint256 tokenFeeAmount = 30;

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
        erc20Bridge.enqueueDelayedMessage(kind, userAliased, messageDataHash, tokenFeeAmount);

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

    function test_executeCall_EmptyData() public {
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
