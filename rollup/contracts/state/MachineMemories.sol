//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./MerkleProofs.sol";
import "./Deserialize.sol";

struct MachineMemory {
	uint64 size;
	bytes32 merkleRoot;
}

library MachineMemories {
	function hash(MachineMemory memory mem) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked("Memory:", mem.size, mem.merkleRoot));
	}

	function proveLeaf(MachineMemory memory mem, uint256 leafIdx, bytes calldata proof, uint256 startOffset) internal pure returns (bytes32 contents, uint256 offset) {
		offset = startOffset;
		MerkleProof memory merkle;
		(contents, offset) = Deserialize.b32(proof, offset);
		(merkle, offset) = Deserialize.merkleProof(proof, offset);
		bytes32 recomputedRoot = MerkleProofs.computeRootForMemory(merkle, leafIdx, contents);
		require(recomputedRoot == mem.merkleRoot, "WRONG_MEM_ROOT");
	}
}
