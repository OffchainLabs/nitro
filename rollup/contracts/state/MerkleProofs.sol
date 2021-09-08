//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";

struct MerkleProof {
	bytes32[] counterparts;
}

library MerkleProofs {
	function computeRoot(MerkleProof memory proof, uint256 index, Value memory leaf) internal pure returns (bytes32 h) {
		h = Values.hash(leaf);
		string memory prefix = "Value merkle tree:";
		for (uint256 layer = 0; layer < proof.counterparts.length; layer++) {
			if (index & 1 == 0) {
				h = keccak256(abi.encodePacked(prefix, h, proof.counterparts[layer]));
			} else {
				h = keccak256(abi.encodePacked(prefix, proof.counterparts[layer], h));
			}
		}
	}
}
