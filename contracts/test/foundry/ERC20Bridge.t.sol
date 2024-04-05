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
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetMinterPauser.sol";

import "forge-std/console.sol";

contract ERC20BridgeTest is AbsBridgeTest {
    IERC20Bridge public erc20Bridge;
    IERC20 public nativeToken;

    uint256 public constant MAX_DATA_SIZE = 117_964;

    // msg details
    uint8 public kind = 7;
    bytes32 public messageDataHash = keccak256(abi.encodePacked("some msg"));
    uint256 public tokenFeeAmount = 30;

    function setUp() public {
        // deploy token and bridge
        nativeToken = new ERC20PresetMinterPauser("Appchain Token", "App");
        bridge = ERC20Bridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        erc20Bridge = IERC20Bridge(address(bridge));

        // init bridge
        erc20Bridge.initialize(IOwnable(rollup), address(nativeToken));

        // deploy inbox
        inbox = address(TestUtil.deployProxy(address(new ERC20Inbox(MAX_DATA_SIZE))));
        IERC20Inbox(address(inbox)).initialize(bridge, ISequencerInbox(seqInbox));
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
        vm.expectRevert(abi.encodeWithSelector(InvalidTokenSet.selector, address(0)));
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
        // add fee tokens to inbox
        ERC20PresetMinterPauser(address(nativeToken)).mint(inbox, tokenFeeAmount);

        // snapshot
        uint256 userNativeTokenBalanceBefore = nativeToken.balanceOf(address(user));
        uint256 bridgeNativeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 inboxNativeTokenBalanceBefore = nativeToken.balanceOf(address(inbox));
        uint256 delayedMsgCountBefore = bridge.delayedMessageCount();

        // allow inbox
        vm.prank(rollup);
        bridge.setDelayedInbox(inbox, true);

        // approve bridge to escrow tokens
        vm.prank(user);
        nativeToken.approve(address(bridge), tokenFeeAmount);

        // expect event
        vm.expectEmit(true, true, true, true);
        vm.fee(70);
        uint256 baseFeeToReport = 0;
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

        // enqueue msg
        address userAliased = AddressAliasHelper.applyL1ToL2Alias(user);
        vm.prank(inbox);
        erc20Bridge.enqueueDelayedMessage(kind, userAliased, messageDataHash, tokenFeeAmount);

        //// checks
        uint256 userNativeTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(
            userNativeTokenBalanceAfter,
            userNativeTokenBalanceBefore,
            "Invalid user token balance"
        );

        uint256 bridgeNativeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeNativeTokenBalanceAfter - bridgeNativeTokenBalanceBefore,
            tokenFeeAmount,
            "Invalid bridge token balance"
        );

        uint256 inboxNativeTokenBalanceAfter = nativeToken.balanceOf(address(inbox));
        assertEq(
            inboxNativeTokenBalanceBefore - inboxNativeTokenBalanceAfter,
            tokenFeeAmount,
            "Invalid inbox token balance"
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
        // fund bridge native tokens
        ERC20PresetMinterPauser(address(nativeToken)).mint(address(bridge), 15);

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
        (bool success, ) = bridge.executeCall(user, withdrawalAmount, data);

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
        ERC20PresetMinterPauser(address(nativeToken)).mint(address(bridge), 15);

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
        ERC20PresetMinterPauser(address(nativeToken)).mint(address(bridge), 15);

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
        ERC20PresetMinterPauser(address(nativeToken)).mint(address(bridge), 15);

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

    function test_executeCall_revert_CallTargetNotAllowed() public {
        // allow outbox
        vm.prank(rollup);
        bridge.setOutbox(outbox, true);

        // executeCall shall revert when 'to' is not contract
        address to = address(nativeToken);
        vm.expectRevert(abi.encodeWithSelector(CallTargetNotAllowed.selector, to));
        vm.prank(outbox);
        bridge.executeCall({to: to, value: 10, data: "some data"});
    }

    function test_executeCall_revert_CallNotAllowed() public {
        // deploy and initi bridge contracts
        address _rollup = makeAddr("rollup");
        address _outbox = makeAddr("outbox");
        address _gateway = address(new MockGateway());
        address _nativeToken = address(new MockBridgedToken(_gateway));
        IERC20Bridge _bridge = IERC20Bridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        _bridge.initialize(IOwnable(_rollup), address(_nativeToken));

        // allow outbox
        vm.prank(_rollup);
        _bridge.setOutbox(_outbox, true);

        // fund bridge
        MockBridgedToken(_nativeToken).transfer(address(_bridge), 100 ether);

        // executeCall shall revert when call changes balance of the bridge
        address to = _gateway;
        uint256 withdrawAmount = 25 ether;
        bytes memory data = abi.encodeWithSelector(
            MockGateway.withdraw.selector,
            MockBridgedToken(_nativeToken),
            withdrawAmount
        );
        vm.expectRevert(abi.encodeWithSelector(CallNotAllowed.selector));
        vm.prank(_outbox);
        _bridge.executeCall({to: to, value: 10, data: data});
    }
}

contract MockBridgedToken is ERC20 {
    address public gateway;

    constructor(address _gateway) ERC20("MockBridgedToken", "TT") {
        gateway = _gateway;
        _mint(msg.sender, 1_000_000 ether);
    }

    function bridgeBurn(address account, uint256 amount) external {
        require(msg.sender == gateway, "ONLY_GATEWAY");
        _burn(account, amount);
    }
}

contract MockGateway {
    function withdraw(MockBridgedToken token, uint256 amount) external {
        token.bridgeBurn(msg.sender, amount);
    }
}
