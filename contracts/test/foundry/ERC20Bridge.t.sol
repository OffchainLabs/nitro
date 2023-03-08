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

        // expect event
        vm.expectEmit(true, true, true, true);
        emit MessageDelivered(
            0,
            0,
            inbox,
            kind,
            AddressAliasHelper.applyL1ToL2Alias(user),
            messageDataHash,
            block.basefee,
            uint64(block.timestamp)
        );

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

        // call params
        uint256 withdrawalAmount = 15;
        bytes memory data = "";

        // expect event
        vm.expectEmit(true, true, true, true);
        emit BridgeCallTriggered(outbox, user, withdrawalAmount, data);

        //// execute call
        vm.prank(outbox);
        (bool success, ) = bridge.executeCall({to: user, value: withdrawalAmount, data: data});

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
        EthVault vault = new EthVault();

        // native token balances
        uint256 bridgeNativeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 vaultNativeTokenBalanceBefore = nativeToken.balanceOf(address(vault));

        // call params
        uint256 withdrawalAmount = 15;
        uint256 newVaultVersion = 7;
        bytes memory data = abi.encodeWithSelector(EthVault.setVersion.selector, newVaultVersion);

        // expect event
        vm.expectEmit(true, true, true, true);
        emit BridgeCallTriggered(outbox, address(vault), withdrawalAmount, data);

        //// execute call
        vm.prank(outbox);
        (bool success, ) = bridge.executeCall({
            to: address(vault),
            value: withdrawalAmount,
            data: data
        });

        //// checks
        assertTrue(success, "Execute call failed");
        assertEq(vault.version(), newVaultVersion, "Invalid newVaultVersion");

        uint256 bridgeNativeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeNativeTokenBalanceBefore - bridgeNativeTokenBalanceAfter,
            withdrawalAmount,
            "Invalid bridge native token balance"
        );

        uint256 vaultNativeTokenBalanceAfter = nativeToken.balanceOf(address(vault));
        assertEq(
            vaultNativeTokenBalanceAfter - vaultNativeTokenBalanceBefore,
            withdrawalAmount,
            "Invalid vault native token balance"
        );
    }

    function test_executeCall_UnsuccessfulExtraCall() public {
        // fund bridge with native tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // deploy some contract that will be call receiver
        EthVault vault = new EthVault();

        // native token balances
        uint256 bridgeNativeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 vaultNativeTokenBalanceBefore = nativeToken.balanceOf(address(vault));

        // call params
        uint256 withdrawalAmount = 15;
        bytes memory data = abi.encodeWithSelector(EthVault.justRevert.selector);

        // expect event
        vm.expectEmit(true, true, true, true);
        emit BridgeCallTriggered(outbox, address(vault), withdrawalAmount, data);

        //// execute call - do call which reverts
        vm.prank(outbox);
        (bool success, bytes memory returnData) = bridge.executeCall({
            to: address(vault),
            value: withdrawalAmount,
            data: data
        });

        //// checks
        assertEq(success, false, "Execute shall be unsuccessful");
        assertEq(vault.version(), 0, "Invalid vaultVersion");

        // get and assert revert reason
        assembly {
            returnData := add(returnData, 0x04)
        }
        string memory revertReason = abi.decode(returnData, (string));
        assertEq(revertReason, "bye", "Invalid revert reason");

        // bridge successfully sent native token even though extra call was unsuccessful (we didn't revert it)
        uint256 bridgeNativeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeNativeTokenBalanceBefore - bridgeNativeTokenBalanceAfter,
            withdrawalAmount,
            "Invalid bridge native token balance after unsuccessful extra call"
        );

        // vault successfully recieved native token even though extra call was unsuccessful (we didn't revert it)
        uint256 vaultNativeTokenBalanceAfter = nativeToken.balanceOf(address(vault));
        assertEq(
            vaultNativeTokenBalanceAfter - vaultNativeTokenBalanceBefore,
            withdrawalAmount,
            "Invalid vault native token balance after unsuccessful call"
        );
    }

    function test_executeCall_UnsuccessfulNativeTokenTransfer() public {
        // fund bridge with native tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // deploy some contract that will be call receiver
        EthVault vault = new EthVault();

        // call params
        uint256 withdrawalAmount = 100_000_000;
        uint256 newVaultVersion = 9;
        bytes memory data = abi.encodeWithSelector(EthVault.setVersion.selector, newVaultVersion);

        //// execute call - do call which reverts on native token transfer due to invalid amount
        vm.prank(outbox);
        vm.expectRevert("ERC20: transfer amount exceeds balance");
        bridge.executeCall({to: address(vault), value: withdrawalAmount, data: data});
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
