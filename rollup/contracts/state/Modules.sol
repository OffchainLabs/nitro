//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ModuleMemories.sol";

struct Module {
    bytes32 globalsMerkleRoot;
    ModuleMemory moduleMemory;
    bytes32 tablesMerkleRoot;
    bytes32 functionsMerkleRoot;
}

library Modules {
    function hash(Module memory mod) internal pure returns (bytes32) {
        return
            keccak256(
                abi.encodePacked(
                    "Module:",
                    mod.globalsMerkleRoot,
                    ModuleMemories.hash(mod.moduleMemory),
                    mod.tablesMerkleRoot,
                    mod.functionsMerkleRoot
                )
            );
    }
}
