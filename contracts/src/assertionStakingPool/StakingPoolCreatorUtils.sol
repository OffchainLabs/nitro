// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/utils/Create2.sol";
import "@openzeppelin/contracts/utils/Address.sol";

library StakingPoolCreatorUtils {
    error PoolDoesntExist();
    function getPool(bytes memory creationCode, bytes memory args) internal view returns (address) {
        bytes32 bytecodeHash = keccak256(abi.encodePacked(creationCode, args));
        address pool = Create2.computeAddress(0, bytecodeHash, address(this));
        if (Address.isContract(pool)) {
            return pool;
        } else {
            revert PoolDoesntExist();
        }
    }
}
