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
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

abstract contract AbsInboxTest is Test {
    IInboxBase public inbox;
    IBridge public bridge;

    uint256 public constant MAX_DATA_SIZE = 117_964;

    address public user = address(100);
    address public rollup = address(1000);
    address public seqInbox = address(1001);

    /* solhint-disable func-name-mixedcase */
    function test_getProxyAdmin() public {
        assertFalse(inbox.getProxyAdmin() == address(0), "Invalid proxy admin");
    }

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
        address mockRollupOwner = address(10_000);
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
        address mockRollupOwner = address(10_000);
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
        assertEq(
            (PausableUpgradeable(address(inbox))).paused(),
            false,
            "Invalid initial paused state"
        );

        vm.prank(rollup);
        inbox.pause();

        assertEq((PausableUpgradeable(address(inbox))).paused(), true, "Invalid paused state");
    }

    function test_unpause() public {
        vm.prank(rollup);
        inbox.pause();
        assertEq(
            (PausableUpgradeable(address(inbox))).paused(),
            true,
            "Invalid initial paused state"
        );
        vm.prank(rollup);
        inbox.unpause();

        assertEq((PausableUpgradeable(address(inbox))).paused(), false, "Invalid paused state");
    }

    function test_initialize_revert_ReInit() public {
        vm.expectRevert("Initializable: contract is already initialized");
        inbox.initialize(bridge, ISequencerInbox(seqInbox));
    }

    function test_initialize_revert_NonDelegated() public {
        ERC20Inbox inb = new ERC20Inbox(MAX_DATA_SIZE);
        vm.expectRevert("Function must be called through delegatecall");
        inb.initialize(bridge, ISequencerInbox(seqInbox));
    }

    function test_sendL2MessageFromOrigin_revert() public {
        vm.expectRevert(abi.encodeWithSelector(Deprecated.selector));
        vm.prank(user);
        inbox.sendL2MessageFromOrigin(abi.encodePacked("some msg"));
    }

    function test_sendL2Message() public {
        // L2 msg params
        bytes memory data = abi.encodePacked("some msg");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(0, data);

        // send L2 msg -> tx.origin == msg.sender
        vm.prank(user, user);
        uint256 msgNum = inbox.sendL2Message(data);

        //// checks
        assertEq(msgNum, 0, "Invalid msgNum");
        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_sendL2Message_revert_WhenPaused() public {
        vm.prank(rollup);
        inbox.pause();

        vm.expectRevert("Pausable: paused");
        vm.prank(user);
        inbox.sendL2Message(abi.encodePacked("some msg"));
    }

    function test_sendL2Message_revert_NotAllowed() public {
        vm.prank(rollup);
        inbox.setAllowListEnabled(true);

        vm.expectRevert(abi.encodeWithSelector(NotAllowedOrigin.selector, user));
        vm.prank(user, user);
        inbox.sendL2Message(abi.encodePacked("some msg"));
    }

    function test_sendL2Message_revert_L1Forked() public {
        vm.chainId(10);
        vm.expectRevert(abi.encodeWithSelector(L1Forked.selector));
        vm.prank(user, user);
        inbox.sendL2Message(abi.encodePacked("some msg"));
    }

    function test_sendUnsignedTransaction() public {
        // L2 msg params
        uint256 maxFeePerGas = 0;
        uint256 gasLimit = 10;
        uint256 nonce = 3;
        uint256 value = 300;
        bytes memory data = abi.encodePacked("test data");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(
            0,
            abi.encodePacked(
                L2MessageType_unsignedEOATx,
                gasLimit,
                maxFeePerGas,
                nonce,
                uint256(uint160(user)),
                value,
                data
            )
        );

        // send TX
        vm.prank(user, user);
        uint256 msgNum = inbox.sendUnsignedTransaction(
            gasLimit,
            maxFeePerGas,
            nonce,
            user,
            value,
            data
        );

        //// checks
        assertEq(msgNum, 0, "Invalid msgNum");
        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_sendUnsignedTransaction_revert_WhenPaused() public {
        vm.prank(rollup);
        inbox.pause();

        vm.expectRevert("Pausable: paused");
        vm.prank(user);
        inbox.sendUnsignedTransaction(10, 10, 10, user, 10, abi.encodePacked("test data"));
    }

    function test_sendUnsignedTransaction_revert_NotAllowed() public {
        vm.prank(rollup);
        inbox.setAllowListEnabled(true);

        vm.expectRevert(abi.encodeWithSelector(NotAllowedOrigin.selector, user));
        vm.prank(user, user);
        inbox.sendUnsignedTransaction(10, 10, 10, user, 10, abi.encodePacked("test data"));
    }

    function test_sendUnsignedTransaction_revert_GasLimitTooLarge() public {
        uint256 tooBigGasLimit = uint256(type(uint64).max) + 1;

        vm.expectRevert(GasLimitTooLarge.selector);
        vm.prank(user, user);
        inbox.sendUnsignedTransaction(tooBigGasLimit, 10, 10, user, 10, abi.encodePacked("data"));
    }

    function test_sendContractTransaction() public {
        // L2 msg params
        uint256 maxFeePerGas = 0;
        uint256 gasLimit = 10;
        uint256 value = 300;
        bytes memory data = abi.encodePacked("test data");

        // expect event
        vm.expectEmit(true, true, true, true);
        emit InboxMessageDelivered(
            0,
            abi.encodePacked(
                L2MessageType_unsignedContractTx,
                gasLimit,
                maxFeePerGas,
                uint256(uint160(user)),
                value,
                data
            )
        );

        // send TX
        vm.prank(user);
        uint256 msgNum = inbox.sendContractTransaction(gasLimit, maxFeePerGas, user, value, data);

        //// checks
        assertEq(msgNum, 0, "Invalid msgNum");
        assertEq(bridge.delayedMessageCount(), 1, "Invalid delayed message count");
    }

    function test_sendContractTransaction_revert_WhenPaused() public {
        vm.prank(rollup);
        inbox.pause();

        vm.expectRevert("Pausable: paused");
        inbox.sendContractTransaction(10, 10, user, 10, abi.encodePacked("test data"));
    }

    function test_sendContractTransaction_revert_NotAllowed() public {
        vm.prank(rollup);
        inbox.setAllowListEnabled(true);

        vm.expectRevert(abi.encodeWithSelector(NotAllowedOrigin.selector, user));
        vm.prank(user, user);
        inbox.sendContractTransaction(10, 10, user, 10, abi.encodePacked("test data"));
    }

    function test_sendContractTransaction_revert_GasLimitTooLarge() public {
        uint256 tooBigGasLimit = uint256(type(uint64).max) + 1;

        vm.expectRevert(GasLimitTooLarge.selector);
        vm.prank(user);
        inbox.sendContractTransaction(tooBigGasLimit, 10, user, 10, abi.encodePacked("data"));
    }

    /**
     *
     * Event declarations
     *
     */

    event AllowListAddressSet(address indexed user, bool val);
    event AllowListEnabledUpdated(bool isEnabled);
    event InboxMessageDelivered(uint256 indexed messageNum, bytes data);
    event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum);
}

contract Sender {}
