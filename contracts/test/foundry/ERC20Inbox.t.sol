// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/ERC20Inbox.sol";
import "../../src/bridge/ISequencerInbox.sol";
import "../../src/libraries/AddressAliasHelper.sol";
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
    function test_setAllowList() public {
        address[] memory users = new address[](2);
        users[0] = address(300);
        users[1] = address(301);

        bool[] memory allowed = new bool[](2);
        allowed[0] = true;
        allowed[0] = false;

        vm.expectEmit(true, true, true, true);
        emit AllowListAddressSet(users[0], allowed[0]);
        emit AllowListAddressSet(users[1], allowed[1]);

        vm.prank(rollup);
        inbox.setAllowList(users, allowed);

        assertEq(inbox.isAllowed(users[0]), allowed[0], "Invalid isAllowed user[0]");
        assertEq(inbox.isAllowed(users[1]), allowed[1], "Invalid isAllowed user[1]");
    }

    function test_setAllowList_revert_InvalidLength() public {
        address[] memory users = new address[](1);
        users[0] = address(300);

        bool[] memory allowed = new bool[](2);
        allowed[0] = true;
        allowed[0] = false;

        vm.expectRevert("INVALID_INPUT");
        vm.prank(rollup);
        inbox.setAllowList(users, allowed);
    }

    function test_setOutbox_revert_NonOwnerCall() public {
        // mock the owner() call on rollup
        address mockRollupOwner = address(10000);
        vm.mockCall(
            rollup,
            abi.encodeWithSelector(IOwnable.owner.selector),
            abi.encode(mockRollupOwner)
        );

        // setAllowList shall revert
        vm.expectRevert(
            abi.encodeWithSelector(
                NotRollupOrOwner.selector,
                address(this),
                rollup,
                mockRollupOwner
            )
        );

        address[] memory users = new address[](2);
        users[0] = address(300);
        bool[] memory allowed = new bool[](2);
        allowed[0] = true;
        inbox.setAllowList(users, allowed);
    }

    function test_setAllowListEnabled_EnableAllowList() public {
        assertEq(inbox.allowListEnabled(), false, "Invalid initial value for allowList");

        vm.expectEmit(true, true, true, true);
        emit AllowListEnabledUpdated(true);

        vm.prank(rollup);
        inbox.setAllowListEnabled(true);

        assertEq(inbox.allowListEnabled(), true, "Invalid allowList");
    }

    function test_setAllowListEnabled_DisableAllowList() public {
        vm.prank(rollup);
        inbox.setAllowListEnabled(true);
        assertEq(inbox.allowListEnabled(), true, "Invalid initial value for allowList");

        vm.expectEmit(true, true, true, true);
        emit AllowListEnabledUpdated(false);

        vm.prank(rollup);
        inbox.setAllowListEnabled(false);

        assertEq(inbox.allowListEnabled(), false, "Invalid allowList");
    }

    function test_setAllowListEnabled_revert_AlreadyEnabled() public {
        vm.prank(rollup);
        inbox.setAllowListEnabled(true);
        assertEq(inbox.allowListEnabled(), true, "Invalid initial value for allowList");

        vm.expectRevert("ALREADY_SET");
        vm.prank(rollup);
        inbox.setAllowListEnabled(true);
    }

    function test_setAllowListEnabled_revert_AlreadyDisabled() public {
        vm.prank(rollup);
        vm.expectRevert("ALREADY_SET");
        inbox.setAllowListEnabled(false);
    }

    function test_setAllowListEnabled_revert_NonOwnerCall() public {
        // mock the owner() call on rollup
        address mockRollupOwner = address(10000);
        vm.mockCall(
            rollup,
            abi.encodeWithSelector(IOwnable.owner.selector),
            abi.encode(mockRollupOwner)
        );

        // setAllowListEnabled shall revert
        vm.expectRevert(
            abi.encodeWithSelector(
                NotRollupOrOwner.selector,
                address(this),
                rollup,
                mockRollupOwner
            )
        );

        inbox.setAllowListEnabled(true);
    }

    function test_pause() public {
        assertEq(inbox.paused(), false, "Invalid initial paused state");

        vm.prank(rollup);
        inbox.pause();

        assertEq(inbox.paused(), true, "Invalid paused state");
    }

    function test_unpause() public {
        vm.prank(rollup);
        inbox.pause();
        assertEq(inbox.paused(), true, "Invalid initial paused state");

        vm.prank(rollup);
        inbox.unpause();

        assertEq(inbox.paused(), false, "Invalid paused state");
    }

    function test_initialize() public {
        assertEq(address(inbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(inbox.sequencerInbox()), seqInbox, "Invalid seqInbox ref");
        assertEq(inbox.allowListEnabled(), false, "Invalid allowListEnabled");
        assertEq(inbox.paused(), false, "Invalid paused state");
    }

    function test_initialize_revert_ReInit() public {
        vm.expectRevert("Initializable: contract is already initialized");
        inbox.initialize(bridge, ISequencerInbox(seqInbox));
    }

    function test_initialize_revert_NonDelegated() public {
        ERC20Inbox inb = new ERC20Inbox();
        vm.expectRevert("Function must be called through delegatecall");
        inb.initialize(bridge, ISequencerInbox(seqInbox));
    }

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

    function test_depositERC20_revert_NativeTokenTransferFails() public {
        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        // deposit tokens
        vm.prank(user);
        uint256 invalidDepositAmount = 1_000_000;
        vm.expectRevert("ERC20: insufficient allowance");
        inbox.depositERC20(invalidDepositAmount);

        //// checks

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(bridgeTokenBalanceAfter, bridgeTokenBalanceBefore, "Invalid bridge token balance");

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(userTokenBalanceBefore, userTokenBalanceAfter, "Invalid user token balance");

        assertEq(bridge.delayedMessageCount(), 0, "Invalid delayed message count");
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

    /****
     **** Event declarations
     ***/

    event AllowListAddressSet(address indexed user, bool val);
    event AllowListEnabledUpdated(bool isEnabled);
    event InboxMessageDelivered(uint256 indexed messageNum, bytes data);
    event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum);
}
