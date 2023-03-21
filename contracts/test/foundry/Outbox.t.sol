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
}
