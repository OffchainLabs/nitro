//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./Values.sol";
import "./Instructions.sol";

struct MerkleProof {
	bytes32[] counterparts;
}

library MerkleProofs {
	function computeRoot(MerkleProof memory proof, uint256 index, Value memory leaf) internal pure returns (bytes32) {
		return computeRootUnsafe(proof, index, Values.hash(leaf), "Value merkle tree:");
	}

	function computeRoot(MerkleProof memory proof, uint256 index, InstructionWindow memory leaf) internal pure returns (bytes32) {
		// Ensure that this is actually an instruction hash
		require(leaf.proved.length > 0, "MUST_PROVE_WINDOW");
		return computeRootUnsafe(proof, index, Instructions.hash(leaf), "Function merkle tree:");
	}

	function computeRootForMemory(MerkleProof memory proof, uint256 index, bytes32 contents) internal pure returns (bytes32) {
		bytes32 h = keccak256(abi.encodePacked("Memory leaf:", contents));
		return computeRootUnsafe(proof, index, h, "Memory merkle tree:");
	}

	// WARNING: leafHash must be computed in such a way that it cannot be a non-leaf hash.
	function computeRootUnsafe(MerkleProof memory proof, uint256 index, bytes32 leafHash, string memory prefix) internal pure returns (bytes32 h) {
		h = leafHash;
		for (uint256 layer = 0; layer < proof.counterparts.length; layer++) {
			if (index & 1 == 0) {
				h = keccak256(abi.encodePacked(prefix, h, proof.counterparts[layer]));
			} else {
				h = keccak256(abi.encodePacked(prefix, proof.counterparts[layer], h));
			}
			index >>= 1;
		}
	}
}
