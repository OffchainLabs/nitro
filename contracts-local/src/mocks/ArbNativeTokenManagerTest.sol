// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.24;

import "../precompiles/ArbNativeTokenManager.sol";

contract ArbNativeTokenManagerTest {
    function mint(
        uint256 amount
    ) external {
        ArbNativeTokenManager(address(0x73)).mintNativeToken(amount);
    }
}
