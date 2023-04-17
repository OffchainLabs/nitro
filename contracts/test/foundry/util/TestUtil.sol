// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.4;

import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

library TestUtil {
    function deployProxy(address logic) public returns (address) {
        ProxyAdmin pa = new ProxyAdmin();
        return address(new TransparentUpgradeableProxy(address(logic), address(pa), ""));
    }
}
