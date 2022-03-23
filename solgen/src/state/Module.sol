// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./ModuleMemory.sol";

struct Module {
    bytes32 globalsMerkleRoot;
    ModuleMemory moduleMemory;
    bytes32 tablesMerkleRoot;
    bytes32 functionsMerkleRoot;
    uint32 internalsOffset;
}

library ModuleLib {
    using ModuleMemoryLib for ModuleMemory;

    function hash(Module memory mod) internal pure returns (bytes32) {
        return
            keccak256(
                abi.encodePacked(
                    "Module:",
                    mod.globalsMerkleRoot,
                    mod.moduleMemory.hash(),
                    mod.tablesMerkleRoot,
                    mod.functionsMerkleRoot,
                    mod.internalsOffset
                )
            );
    }
}
