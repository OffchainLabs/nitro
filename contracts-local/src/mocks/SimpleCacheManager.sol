// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../precompiles/ArbWasmCache.sol";

contract SimpleCacheManager {
    function cacheProgram(
        address program
    ) external {
        ArbWasmCache(address(0x72)).cacheProgram(program);
    }

    function evictProgram(
        address program
    ) external {
        ArbWasmCache(address(0x72)).evictCodehash(codehash(program));
    }

    function codehash(
        address program
    ) internal view returns (bytes32 hash) {
        assembly {
            hash := extcodehash(program)
        }
    }
}
