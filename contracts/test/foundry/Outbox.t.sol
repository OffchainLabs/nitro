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

    function test_executeTransaction() public {
        // fund bridge with some ether
        vm.deal(address(bridge), 100 ether);

        // store root
        vm.prank(rollup);
        outbox.updateSendRoot(
            0xc86f4eaf8efb31147795fb05564f8777abc3220d4caeb0227c6c69c115931dda,
            0xc86f4eaf8efb31147795fb05564f8777abc3220d4caeb0227c6c69c115931dda
        );

        // create msg receiver on L1
        L2ToL1Target target = new L2ToL1Target();
        target.setOutbox(address(outbox));

        //// execute transaction
        uint256 bridgeBalanceBefore = address(bridge).balance;
        uint256 targetBalanceBefore = address(target).balance;

        bytes32[] memory proof = new bytes32[](5);
        proof[0] = bytes32(0x1216ff070e3c87b032d79b298a3e98009ddd13bf8479b843e225857ca5f950e7);
        proof[1] = bytes32(0x2b5ee8f4bd7664ca0cf31d7ab86119b63f6ff07bb86dbd5af356d0087492f686);
        proof[2] = bytes32(0x0aa797064e0f3768bbac0a02ce031c4f282441a9cd8c669086cf59a083add893);
        proof[3] = bytes32(0xc7aac0aad5108a46ac9879f0b1870fd0cbc648406f733eb9d0b944a18c32f0f8);
        proof[4] = bytes32(0x477ce2b0bc8035ae3052b7339c7496531229bd642bb1871d81618cf93a4d2d1a);

        uint256 withdrawalAmount = 15 ether;
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
        assertEq(uint256(target.outputId()), 12, "Invalid outputId");
        assertEq(target.sender(), user, "Invalid sender");
        assertEq(uint256(target.l1Block()), 20, "Invalid l1Block");
        assertEq(uint256(target.withdrawalAmount()), withdrawalAmount, "Invalid withdrawalAmount");
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
