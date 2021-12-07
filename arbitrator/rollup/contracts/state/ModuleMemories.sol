//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./MerkleProofs.sol";
import "./Deserialize.sol";

struct ModuleMemory {
	uint64 size;
	bytes32 merkleRoot;
}

library ModuleMemories {
	function hash(ModuleMemory memory mem) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked("Memory:", mem.size, mem.merkleRoot));
	}

	function proveLeaf(ModuleMemory memory mem, uint256 leafIdx, bytes calldata proof, uint256 startOffset) internal pure returns (bytes32 contents, uint256 offset, MerkleProof memory merkle) {
		offset = startOffset;
		(contents, offset) = Deserialize.b32(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRootFromMemory(merkle, leafIdx, contents);
		require(recomputedRoot == mem.merkleRoot, "WRONG_MEM_ROOT");
	}
}
