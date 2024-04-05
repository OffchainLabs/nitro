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

contract Random {
    bytes32 seed = bytes32(uint256(0x137));

    function Bytes32() public returns (bytes32) {
        seed = keccak256(abi.encodePacked(seed));
        return seed;
    }

    function Bytes(uint256 length) public returns (bytes memory) {
        require(length > 0, "Length must be greater than 0");
        bytes memory randomBytes = new bytes(length);

        for (uint256 i = 0; i < length; i++) {
            Bytes32();
            randomBytes[i] = bytes1(uint8(uint256(seed) % 256));
        }

        return randomBytes;
    }
}
