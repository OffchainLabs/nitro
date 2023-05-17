// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "./AbsBridge.t.sol";
import "../../src/bridge/IEthBridge.sol";
import "../../src/libraries/AddressAliasHelper.sol";

contract BridgeTest is AbsBridgeTest {
    IEthBridge public ethBridge;

    // msg details
    uint8 public kind = 7;
    bytes32 public messageDataHash = keccak256(abi.encodePacked("some msg"));
    uint256 public ethAmount = 2 ether;

    function setUp() public {
        inbox = address(1001);

        // deploy eth and bridge
        bridge = Bridge(TestUtil.deployProxy(address(new Bridge())));
        ethBridge = IEthBridge(address(bridge));

        // init bridge
        ethBridge.initialize(IOwnable(rollup));

        // fund user account
        vm.deal(user, 10 ether);
    }

    /* solhint-disable func-name-mixedcase */
    function test_initialize() public {
        assertEq(address(bridge.rollup()), rollup, "Invalid rollup ref");
        assertEq(bridge.activeOutbox(), address(0), "Invalid activeOutbox ref");
    }

    function test_initialize_revert_ReInit() public {
        vm.expectRevert("Initializable: contract is already initialized");
        ethBridge.initialize(IOwnable(rollup));
    }

    function test_initialize_revert_NonDelegated() public {
        IEthBridge noTokenBridge = new Bridge();
        vm.expectRevert("Function must be called through delegatecall");
        noTokenBridge.initialize(IOwnable(rollup));
    }

    function test_enqueueDelayedMessage() public {
        // inbox will move ETH to bridge
        vm.deal(inbox, ethAmount);
        uint256 inboxEthBalanceBefore = address(inbox).balance;
        uint256 bridgeEthBalanceBefore = address(bridge).balance;

        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // expect event
        vm.expectEmit(true, true, true, true);
        vm.fee(70);
        uint256 baseFeeToReport = block.basefee;
        emit MessageDelivered(
            0,
            0,
            inbox,
            kind,
            AddressAliasHelper.applyL1ToL2Alias(user),
            messageDataHash,
            baseFeeToReport,
            uint64(block.timestamp)
        );

        // enqueue msg inbox->bridge
        address userAliased = AddressAliasHelper.applyL1ToL2Alias(user);
        vm.prank(inbox);
        ethBridge.enqueueDelayedMessage{value: ethAmount}(kind, userAliased, messageDataHash);

        //// checks

        uint256 bridgeEthBalanceAfter = address(bridge).balance;
        assertEq(
            bridgeEthBalanceAfter - bridgeEthBalanceBefore,
            ethAmount,
            "Invalid bridge eth balance"
        );

        uint256 inboxEthBalanceAfter = address(inbox).balance;
        assertEq(inboxEthBalanceBefore - inboxEthBalanceAfter, ethAmount, "Invalid inbox balance");

        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_enqueueDelayedMessage_TwoInRow() public {
        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        vm.deal(inbox, ethAmount);
        uint256 inboxEthBalanceBefore = address(inbox).balance;
        uint256 bridgeEthBalanceBefore = address(bridge).balance;

        // 1st enqueue msg
        vm.prank(inbox);
        ethBridge.enqueueDelayedMessage{value: 1 ether}(2, address(400), messageDataHash);

        // expect event
        vm.expectEmit(true, true, true, true);
        emit MessageDelivered(
            1,
            bridge.delayedInboxAccs(0),
            inbox,
            8,
            AddressAliasHelper.applyL1ToL2Alias(user),
            messageDataHash,
            block.basefee,
            uint64(block.timestamp)
        );

        // enqueue msg inbox->bridge
        address userAliased = AddressAliasHelper.applyL1ToL2Alias(user);
        vm.prank(inbox);
        ethBridge.enqueueDelayedMessage{value: 1 ether}(8, userAliased, messageDataHash);

        //// checks

        uint256 bridgeEthBalanceAfter = address(bridge).balance;
        assertEq(
            bridgeEthBalanceAfter - bridgeEthBalanceBefore,
            ethAmount,
            "Invalid bridge eth balance"
        );

        uint256 inboxEthBalanceAfter = address(inbox).balance;
        assertEq(inboxEthBalanceBefore - inboxEthBalanceAfter, ethAmount, "Invalid inbox balance");

        assertEq(bridge.delayedMessageCount(), 2, "Invalid delayed message count");
    }

    function test_enqueueDelayedMessage_revert_UseTokenForFees() public {
        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // enqueue msg
        hoax(inbox);
        vm.expectRevert();
        IERC20Bridge(address(bridge)).enqueueDelayedMessage(kind, user, messageDataHash, 1000);
    }

    function test_enqueueDelayedMessage_revert_NotDelayedInbox() public {
        hoax(inbox);
        vm.expectRevert(abi.encodeWithSelector(NotDelayedInbox.selector, inbox));
        ethBridge.enqueueDelayedMessage{value: ethAmount}(kind, user, messageDataHash);
    }

    function test_executeCall_EmptyCalldata() public {
        // fund bridge with some eth
        vm.deal(address(bridge), 10 ether);
        uint256 bridgeEthBalanceBefore = address(bridge).balance;
        uint256 userEthBalanceBefore = address(user).balance;

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        uint256 withdrawalAmount = 3 ether;

        // expect event
        vm.expectEmit(true, true, true, true);
        emit BridgeCallTriggered(outbox, user, withdrawalAmount, "");

        //// execute call
        vm.prank(outbox);
        (bool success, ) = bridge.executeCall({to: user, value: withdrawalAmount, data: ""});

        //// checks
        assertTrue(success, "Execute call failed");

        uint256 bridgeEthBalanceAfter = address(bridge).balance;
        assertEq(
            bridgeEthBalanceBefore - bridgeEthBalanceAfter,
            withdrawalAmount,
            "Invalid bridge eth balance"
        );

        uint256 userEthBalanceAfter = address(user).balance;
        assertEq(
            userEthBalanceAfter - userEthBalanceBefore,
            withdrawalAmount,
            "Invalid user eth balance"
        );
    }

    function test_executeCall_WithCalldata() public {
        // fund bridge with some eth
        vm.deal(address(bridge), 10 ether);

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // deploy some contract that will be call receiver
        EthVault vault = new EthVault();

        uint256 bridgeEthBalanceBefore = address(bridge).balance;
        uint256 vaultEthBalanceBefore = address(vault).balance;

        // call params
        uint256 newVaultVersion = 7;
        uint256 withdrawalAmount = 3 ether;
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

        uint256 bridgeEthBalanceAfter = address(bridge).balance;
        assertEq(
            bridgeEthBalanceBefore - bridgeEthBalanceAfter,
            withdrawalAmount,
            "Invalid bridge eth balance"
        );

        uint256 vaultEthBalanceAfter = address(vault).balance;
        assertEq(
            vaultEthBalanceAfter - vaultEthBalanceBefore,
            withdrawalAmount,
            "Invalid vault eth balance"
        );
    }

    function test_executeCall_UnsuccessfulCall() public {
        // fund bridge with some eth
        vm.deal(address(bridge), 10 ether);

        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // deploy some contract that will be call receiver
        EthVault vault = new EthVault();

        uint256 bridgeEthBalanceBefore = address(bridge).balance;
        uint256 vaultEthBalanceBefore = address(vault).balance;

        // call params
        uint256 withdrawalAmount = 3 ether;
        bytes memory revertingData = abi.encodeWithSelector(EthVault.justRevert.selector);

        // expect event
        vm.expectEmit(true, true, true, true);
        emit BridgeCallTriggered(outbox, address(vault), withdrawalAmount, revertingData);

        //// execute call - do call which reverts
        vm.prank(outbox);
        (bool success, bytes memory returnData) = bridge.executeCall({
            to: address(vault),
            value: withdrawalAmount,
            data: revertingData
        });

        //// checks
        assertEq(success, false, "Execute shall be unsuccessful");
        assertEq(vault.version(), 0, "Invalid vaultVersion");

        // get revert reason
        assembly {
            returnData := add(returnData, 0x04)
        }
        string memory revertReason = abi.decode(returnData, (string));
        assertEq(revertReason, "bye", "Invalid revert reason");

        uint256 bridgeEthBalanceAfter = address(bridge).balance;
        assertEq(
            bridgeEthBalanceBefore,
            bridgeEthBalanceAfter,
            "Invalid bridge eth balance after unsuccessful call"
        );

        uint256 vaultEthBalanceAfter = address(vault).balance;
        assertEq(
            vaultEthBalanceAfter,
            vaultEthBalanceBefore,
            "Invalid vault eth balance after unsuccessful call"
        );
    }

    function test_executeCall_revert_NotOutbox() public {
        vm.expectRevert(abi.encodeWithSelector(NotOutbox.selector, address(this)));
        bridge.executeCall({to: user, value: 0.1 ether, data: ""});
    }

    function test_executeCall_revert_NotContract() public {
        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // executeCall shall revert when 'to' is not contract
        address to = address(234);
        vm.expectRevert(abi.encodeWithSelector(NotContract.selector, address(to)));
        vm.prank(outbox);
        bridge.executeCall({to: to, value: 0.1 ether, data: "some data"});
    }
}
