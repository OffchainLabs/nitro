//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "./IBridge.sol";

contract Outbox {

    address public rollup;              // the rollup contract
    IBridge public bridge;              // the bridge contract

    mapping(uint256 => bool  ) spent;  // maps leaf number => if spent
    mapping(bytes32 => uint64) roots;  // maps root hashes => tree size

    function initialize(address _rollup, IBridge _bridge) external {
        require(rollup == address(0), "ALREADY_INIT");
        rollup = _rollup;
        bridge = _bridge;
    }

    function addRoot(bytes32 hash, uint64 size) external {
        require(msg.sender == rollup, "ONLY_ROLLUP");
        roots[hash] = size;
    }

    function executeSend(
        bytes32[] memory proof,
        bytes32 send,
        bytes32 root,
        uint64 leaf
    ) external returns (bool) {

        uint64 size = roots[root];

        require(size != 0, "BAD_ROOT");
        require(proof.length < 64, "PROOF_TOO_LONG");
        require(spent[leaf], "ALREADY_SPENT");

        for (uint8 level = 0; level < proof.length; level++) {
            bytes32 sibling = proof[level];
            
            if (leaf ^ 1 == 0) {
                send = keccak256(abi.encodePacked(send, sibling));
            } else {
                send = keccak256(abi.encodePacked(sibling, send));
            }
        }

        require(send == root, "INVALID_PROOF");
        spent[leaf] = true;
        
    }
}
