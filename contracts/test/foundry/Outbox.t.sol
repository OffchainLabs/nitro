// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "./AbsOutbox.t.sol";
import "../../src/bridge/Bridge.sol";
import "../../src/bridge/Outbox.sol";

contract OutboxTest is AbsOutboxTest {
    Outbox public ethOutbox;
    Bridge public ethBridge;

    function setUp() public {
        // deploy bridge and outbox
        bridge = IBridge(TestUtil.deployProxy(address(new Bridge())));
        ethBridge = Bridge(address(bridge));
        outbox = IOutbox(TestUtil.deployProxy(address(new Outbox())));
        ethOutbox = Outbox(address(outbox));

        // init bridge and outbox
        ethBridge.initialize(IOwnable(rollup));
        ethOutbox.initialize(IBridge(bridge));

        vm.prank(rollup);
        bridge.setOutbox(address(outbox), true);
    }

    /* solhint-disable func-name-mixedcase */
    function test_executeTransaction() public {
        // fund bridge with some ether
        vm.deal(address(bridge), 100 ether);

        // create msg receiver on L1
        L2ToL1Target target = new L2ToL1Target();
        target.setOutbox(address(outbox));

        //// execute transaction
        uint256 bridgeBalanceBefore = address(bridge).balance;
        uint256 targetBalanceBefore = address(target).balance;

        bytes32[] memory proof = new bytes32[](1);
        proof[0] = bytes32(0);

        uint256 withdrawalAmount = 15 ether;
        bytes memory data = abi.encodeWithSignature("receiveHook()");

        uint256 index = 1;
        bytes32 itemHash = outbox.calculateItemHash({
            l2Sender: user,
            to: address(target),
            l2Block: 300,
            l1Block: 20,
            l2Timestamp: 1234,
            value: withdrawalAmount,
            data: data
        });
        bytes32 root = outbox.calculateMerkleRoot(proof, index, itemHash);
        // store root
        vm.prank(rollup);
        outbox.updateSendRoot(
            root,
            bytes32(uint256(1))
        );

        outbox.executeTransaction({
            proof: proof,
            index: index,
            l2Sender: user,
            to: address(target),
            l2Block: 300,
            l1Block: 20,
            l2Timestamp: 1234,
            value: withdrawalAmount,
            data: data
        });

        uint256 bridgeBalanceAfter = address(bridge).balance;
        assertEq(
            bridgeBalanceBefore - bridgeBalanceAfter,
            withdrawalAmount,
            "Invalid bridge balance"
        );

        uint256 targetBalanceAfter = address(target).balance;
        assertEq(
            targetBalanceAfter - targetBalanceBefore,
            withdrawalAmount,
            "Invalid target balance"
        );

        /// check context was properly set during execution
        assertEq(uint256(target.l2Block()), 300, "Invalid l2Block");
        assertEq(uint256(target.timestamp()), 1234, "Invalid timestamp");
        assertEq(uint256(target.outputId()), index, "Invalid outputId");
        assertEq(target.sender(), user, "Invalid sender");
        assertEq(uint256(target.l1Block()), 20, "Invalid l1Block");
        assertEq(uint256(target.withdrawalAmount()), withdrawalAmount, "Invalid withdrawalAmount");

        vm.expectRevert(abi.encodeWithSignature("AlreadySpent(uint256)", index));
        outbox.executeTransaction({
            proof: proof,
            index: index,
            l2Sender: user,
            to: address(target),
            l2Block: 300,
            l1Block: 20,
            l2Timestamp: 1234,
            value: withdrawalAmount,
            data: data
        });
    }
}

/**
 * Contract for testing L2 to L1 msgs
 */
contract L2ToL1Target {
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
        withdrawalAmount = msg.value;
    }

    function setOutbox(address _outbox) external {
        outbox = _outbox;
    }
}
