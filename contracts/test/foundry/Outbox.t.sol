// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/Outbox.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

contract OutboxTest is Test {
    Outbox public outbox;
    ERC20Bridge public bridge;
    IERC20 public nativeToken;

    address public user = address(100);
    address public rollup = address(1000);
    address public seqInbox = address(1001);

    function setUp() public {
        // deploy token, bridge and outbox
        nativeToken = new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this));
        bridge = ERC20Bridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        outbox = Outbox(TestUtil.deployProxy(address(new Outbox())));

        // init bridge and outbox
        bridge.initialize(IOwnable(rollup), address(nativeToken));
        outbox.initialize(IBridge(bridge));

        vm.prank(rollup);
        bridge.setOutbox(address(outbox), true);

        // fund user account
        nativeToken.transfer(user, 1_000);
    }

    /* solhint-disable func-name-mixedcase */
    function test_initialize() public {
        assertEq(address(outbox.bridge()), address(bridge), "Invalid bridge ref");
        assertEq(address(outbox.rollup()), rollup, "Invalid rollup ref");
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
            0xcb920246c89b1654256473a041b5231711799d82dcae749351c758af797598e3,
            0xcb920246c89b1654256473a041b5231711799d82dcae749351c758af797598e3
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
        outbox.executeTransaction({
            proof: proof,
            index: 12,
            l2Sender: user,
            to: user,
            l2Block: 300,
            l1Block: 20,
            l2Timestamp: 1234,
            value: withdrawalAmount,
            data: ""
        });

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
