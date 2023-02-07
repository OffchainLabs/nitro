// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import {MerkleProofTooLong} from "./Error.sol";

library MerkleLib {
    function generateRoot(bytes32[] memory _hashes) internal pure returns (bytes32) {
        bytes32[] memory prevLayer = _hashes;
        while (prevLayer.length > 1) {
            bytes32[] memory nextLayer = new bytes32[]((prevLayer.length + 1) / 2);
            for (uint256 i = 0; i < nextLayer.length; i++) {
                if (2 * i + 1 < prevLayer.length) {
                    nextLayer[i] = keccak256(
                        abi.encodePacked(prevLayer[2 * i], prevLayer[2 * i + 1])
                    );
                } else {
                    nextLayer[i] = prevLayer[2 * i];
                }
            }
            prevLayer = nextLayer;
        }
        return prevLayer[0];
    }

    function calculateRoot(
        bytes32[] memory nodes,
        uint256 route,
        bytes32 item
    ) internal pure returns (bytes32) {
        uint256 proofItems = nodes.length;
        if (proofItems > 256) revert MerkleProofTooLong(proofItems, 256);
        bytes32 h = item;
        for (uint256 i = 0; i < proofItems; ) {
            bytes32 node = nodes[i];
            if ((route & (1 << i)) == 0) {
                assembly {
                    mstore(0x00, h)
                    mstore(0x20, node)
                    h := keccak256(0x00, 0x40)
                }
            } else {
                assembly {
                    mstore(0x00, node)
                    mstore(0x20, h)
                    h := keccak256(0x00, 0x40)
                }
            }
            unchecked {
                ++i;
            }
        }
        return h;
    }
}
