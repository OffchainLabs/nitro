// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "./AbsOutbox.t.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/ERC20Outbox.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

contract ERC20OutboxTest is AbsOutboxTest {
    ERC20Outbox public erc20Outbox;
    ERC20Bridge public erc20Bridge;
    IERC20 public nativeToken;

    function setUp() public {
        // deploy token, bridge and outbox
        nativeToken = new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this));
        bridge = IBridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        erc20Bridge = ERC20Bridge(address(bridge));
        outbox = IOutbox(TestUtil.deployProxy(address(new ERC20Outbox())));
        erc20Outbox = ERC20Outbox(address(outbox));

        // init bridge and outbox
        erc20Bridge.initialize(IOwnable(rollup), address(nativeToken));
        erc20Outbox.initialize(IBridge(bridge));

        vm.prank(rollup);
        bridge.setOutbox(address(outbox), true);

        // fund user account
        nativeToken.transfer(user, 1_000);
    }

    function test_initialize_WithdrawalAmount() public {
        assertEq(erc20Outbox.l2ToL1WithdrawalAmount(), 0, "Invalid withdrawalAmount");
    }

    function test_executeTransaction() public {
        // fund bridge with some tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // store root
        vm.prank(rollup);
        outbox.updateSendRoot(
            0x7e87df146feb0900d5a441d1d081867190b34395307698f4e879c8164cd9a7f9,
            0x7e87df146feb0900d5a441d1d081867190b34395307698f4e879c8164cd9a7f9
        );

        // create msg receiver on L1
        ERC20L2ToL1Target target = new ERC20L2ToL1Target();
        target.setOutbox(address(outbox));

        //// execute transaction
        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 targetTokenBalanceBefore = nativeToken.balanceOf(address(target));

        bytes32[] memory proof = new bytes32[](5);
        proof[0] = bytes32(0x1216ff070e3c87b032d79b298a3e98009ddd13bf8479b843e225857ca5f950e7);
        proof[1] = bytes32(0x2b5ee8f4bd7664ca0cf31d7ab86119b63f6ff07bb86dbd5af356d0087492f686);
        proof[2] = bytes32(0x0aa797064e0f3768bbac0a02ce031c4f282441a9cd8c669086cf59a083add893);
        proof[3] = bytes32(0xc7aac0aad5108a46ac9879f0b1870fd0cbc648406f733eb9d0b944a18c32f0f8);
        proof[4] = bytes32(0x477ce2b0bc8035ae3052b7339c7496531229bd642bb1871d81618cf93a4d2d1a);

        uint256 withdrawalAmount = 15;
        bytes memory data = abi.encodeWithSignature("receiveHook()");
        outbox.executeTransaction({
            proof: proof,
            index: 12,
            l2Sender: user,
            to: address(target),
            l2Block: 300,
            l1Block: 20,
            l2Timestamp: 1234,
            value: withdrawalAmount,
            data: data
        });

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(
            bridgeTokenBalanceBefore - bridgeTokenBalanceAfter,
            withdrawalAmount,
            "Invalid bridge token balance"
        );

        uint256 targetTokenBalanceAfter = nativeToken.balanceOf(address(target));
        assertEq(
            targetTokenBalanceAfter - targetTokenBalanceBefore,
            withdrawalAmount,
            "Invalid target token balance"
        );

        /// check context was properly set during execution
        assertEq(uint256(target.l2Block()), 300, "Invalid l2Block");
        assertEq(uint256(target.timestamp()), 1234, "Invalid timestamp");
        assertEq(uint256(target.outputId()), 12, "Invalid outputId");
        assertEq(target.sender(), user, "Invalid sender");
        assertEq(uint256(target.l1Block()), 20, "Invalid l1Block");
        assertEq(uint256(target.withdrawalAmount()), withdrawalAmount, "Invalid withdrawalAmount");
    }

    function test_executeTransaction_revert_CallTargetNotAllowed() public {
        // // fund bridge with some tokens
        vm.startPrank(user);
        nativeToken.approve(address(bridge), 100);
        nativeToken.transfer(address(bridge), 100);
        vm.stopPrank();

        // store root
        vm.prank(rollup);
        outbox.updateSendRoot(
            0x5b6cd410f78e45e55eeb02133b8e72e6ca122c59b667eed4f214e374d808058e,
            0x5b6cd410f78e45e55eeb02133b8e72e6ca122c59b667eed4f214e374d808058e
        );

        //// execute transaction
        uint256 bridgeTokenBalanceBefore = nativeToken.balanceOf(address(bridge));
        uint256 userTokenBalanceBefore = nativeToken.balanceOf(address(user));

        bytes32[] memory proof = new bytes32[](5);
        proof[0] = bytes32(0x1216ff070e3c87b032d79b298a3e98009ddd13bf8479b843e225857ca5f950e7);
        proof[1] = bytes32(0x2b5ee8f4bd7664ca0cf31d7ab86119b63f6ff07bb86dbd5af356d0087492f686);
        proof[2] = bytes32(0x0aa797064e0f3768bbac0a02ce031c4f282441a9cd8c669086cf59a083add893);
        proof[3] = bytes32(0xc7aac0aad5108a46ac9879f0b1870fd0cbc648406f733eb9d0b944a18c32f0f8);
        proof[4] = bytes32(0x477ce2b0bc8035ae3052b7339c7496531229bd642bb1871d81618cf93a4d2d1a);

        uint256 withdrawalAmount = 15;

        address invalidTarget = address(nativeToken);

        vm.expectRevert(abi.encodeWithSelector(CallTargetNotAllowed.selector, invalidTarget));
        outbox.executeTransaction({
            proof: proof,
            index: 12,
            l2Sender: user,
            to: invalidTarget,
            l2Block: 300,
            l1Block: 20,
            l2Timestamp: 1234,
            value: withdrawalAmount,
            data: ""
        });

        uint256 bridgeTokenBalanceAfter = nativeToken.balanceOf(address(bridge));
        assertEq(bridgeTokenBalanceBefore, bridgeTokenBalanceAfter, "Invalid bridge token balance");

        uint256 userTokenBalanceAfter = nativeToken.balanceOf(address(user));
        assertEq(userTokenBalanceAfter, userTokenBalanceBefore, "Invalid user token balance");
    }
}

/**
 * Contract for testing L2 to L1 msgs
 */
contract ERC20L2ToL1Target {
    address public outbox;

    uint128 public l2Block;
    uint128 public timestamp;
    bytes32 public outputId;
    address public sender;
    uint96 public l1Block;
    uint256 public withdrawalAmount;

    function receiveHook() external payable {
        l2Block = uint128(IOutbox(outbox).l2ToL1Block());
        timestamp = uint128(IOutbox(outbox).l2ToL1Timestamp());
        outputId = IOutbox(outbox).l2ToL1OutputId();
        sender = IOutbox(outbox).l2ToL1Sender();
        l1Block = uint96(IOutbox(outbox).l2ToL1EthBlock());
        withdrawalAmount = ERC20Outbox(outbox).l2ToL1WithdrawalAmount();
    }

    function setOutbox(address _outbox) external {
        outbox = _outbox;
    }
}
