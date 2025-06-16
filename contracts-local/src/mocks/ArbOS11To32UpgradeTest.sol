// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.24;

import "../precompiles/ArbSys.sol";

contract ArbOS11To32UpgradeTest {
    function mcopy() external returns (bytes32 x) {
        assembly {
            mstore(0x20, 0x9) // Store 0x9 at word 1 in memory
            mcopy(0, 0x20, 0x20) // Copies 0x9 to word 0 in memory
            x := mload(0) // Returns 32 bytes "0x9"
        }
        require(ArbSys(address(0x64)).arbOSVersion() == 55 + 32, "EXPECTED_ARBOS_32");
    }
}
