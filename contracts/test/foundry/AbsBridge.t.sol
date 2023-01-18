// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/bridge/ERC20Bridge.sol";
import "../../src/bridge/Bridge.sol";
import "../../src/bridge/ERC20Inbox.sol";
import "../../src/bridge/IEthBridge.sol";
import "../../src/libraries/AddressAliasHelper.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

import "forge-std/console.sol";

contract AbsBridgeTest is Test {
    IBridge public ethBridge;

    IBridge public erc20Bridge;
    IERC20 public nativeToken;

    IBridge[] public bridges;

    address public ethRollup = address(1000);
    address public erc20Rollup = address(1001);
    address public seqInbox = address(1002);

    address public owner = address(10);
    address public user = address(11);

    function setUp() public {
        vm.startPrank(owner);

        // deploy Eth Bridge
        ethBridge = IBridge(TestUtil.deployProxy(address(new Bridge())));
        IEthBridge(address(ethBridge)).initialize(IOwnable(ethRollup));

        // deploy erc20 Bridge
        nativeToken = new ERC20PresetFixedSupply("Appchain Token", "App", 1_000_000, address(this));
        erc20Bridge = IBridge(TestUtil.deployProxy(address(new ERC20Bridge())));
        IERC20Bridge(address(erc20Bridge)).initialize(IOwnable(erc20Rollup), address(nativeToken));
        vm.stopPrank();

        // fund user account
        nativeToken.transfer(user, 100_000);

        bridges.push(ethBridge);
        bridges.push(erc20Bridge);

    }

    function testSetSequencerInbox() public {
        for (uint256 i = 0; i < bridges.length; i++) {
            vm.prank(address(bridges[i].rollup()));
            bridges[i].setSequencerInbox(seqInbox);

            assertEq(bridges[i].sequencerInbox(), seqInbox, "Invalid seqInbox");
        }
    }
}
