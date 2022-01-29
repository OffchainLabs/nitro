//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../libraries/Precompiles.sol";

contract HashProofHelper {
	struct PreimagePart {
		bytes32 fullHash;
		uint64 offset;
		bytes part;
	}

	struct KeccakState {
		uint64 offset;
		bytes part;
		uint64[25] state;
		uint256 length;
	}

	PreimagePart[] public preimageParts;
	mapping(address => KeccakState) public keccakStates;

	event PreimagePartProven(
		bytes32 indexed fullHash,
		uint64 indexed offset,
		uint256 indexed proofNumber,
		bytes part
	);

	uint256 constant MAX_PART_LENGTH = 32;
	uint256 constant KECCAK_ROUND_INPUT = 136;

	function proveWithFullPreimage(bytes calldata data, uint64 offset) external returns (uint256 proofNumber) {
		bytes32 fullHash = keccak256(data);
		bytes memory part;
		if (data.length > offset) {
			uint256 partLength = data.length - offset;
			if (partLength > 32) {
				partLength = 32;
			}
			part = data[offset:(offset + partLength)];
		}
		proofNumber = preimageParts.length;
		preimageParts.push(PreimagePart({
			fullHash: fullHash,
			offset: offset,
			part: part
		}));
		emit PreimagePartProven(
			fullHash,
			offset,
			proofNumber,
			part
		);
	}

	// Flags: a bitset signaling various things about the proof, ordered from least to most significant bits.
	//   0th bit: indicates that this data is the final chunk of preimage data.
	//   1st bit: indicates that the preimage part currently being built should be cleared before this.
	function proveWithSplitPreimage(bytes calldata data, uint64 offset, uint256 flags) external returns (uint256 proofNumber) {
		bool isFinal = (flags & (1 << 0)) != 0;
		if ((flags & (1 << 1)) != 0) {
			delete keccakStates[msg.sender];
		}
		require(isFinal || data.length % KECCAK_ROUND_INPUT == 0, "NOT_BLOCK_ALIGNED");
		KeccakState storage state = keccakStates[msg.sender];
		uint256 startLength = state.length;
		if (startLength == 0) {
			state.offset = offset;
		} else {
			require(state.offset == offset, "DIFF_OFFSET");
		}
		keccakUpdate(state, data, isFinal);
		if (uint256(offset) + MAX_PART_LENGTH > startLength && offset < state.length) {
			uint256 startIdx = 0;
			if (offset > startLength) {
				startIdx = offset - startLength;
			}
			uint256 endIdx = uint256(offset) + MAX_PART_LENGTH - startLength;
			if (endIdx > data.length) {
				endIdx = data.length;
			}
			for (uint256 i = startIdx; i < endIdx; i++) {
				state.part.push(data[i]);
			}
		}
		if (!isFinal) {
			return 0;
		}
		bytes32 fullHash;
		for (uint256 i = 0; i < 32; i++) {
			uint256 stateIdx = i / 8;
			// work around our weird keccakF function state ordering
			stateIdx = 5 * (stateIdx % 5) + stateIdx / 5;
			uint8 b = uint8(state.state[stateIdx] >> ((i % 8) * 8));
			fullHash |= bytes32(uint256(b) << (248 - (i * 8)));
		}
		proofNumber = preimageParts.length;
		preimageParts.push(PreimagePart({
			fullHash: fullHash,
			offset: state.offset,
			part: state.part
		}));
		delete keccakStates[msg.sender];
		emit PreimagePartProven(
			fullHash,
			state.offset,
			proofNumber,
			state.part
		);
	}

	function keccakUpdate(KeccakState storage state, bytes calldata data, bool isFinal) internal {
		state.length += data.length;
		while (true) {
			if (data.length == 0 && !isFinal) {
				break;
			}
			for (uint256 i = 0; i < KECCAK_ROUND_INPUT; i++) {
				uint8 b = 0;
				if (i < data.length) {
					b = uint8(data[i]);
				} else {
					// Padding
					if (i == data.length) {
						b |= uint8(0x01);
					}
					if (i == KECCAK_ROUND_INPUT-1) {
						b |= uint8(0x80);
					}
				}
				uint256 stateIdx = i / 8;
				// work around our weird keccakF function state ordering
				stateIdx = 5 * (stateIdx % 5) + stateIdx / 5;
				state.state[stateIdx] ^= uint64(b) << uint64((i % 8) * 8);
			}
			uint256[25] memory state256;
			for (uint256 i = 0; i < 25; i++) {
				state256[i] = state.state[i];
			}
			state256 = Precompiles.keccakF(state256);
			for (uint256 i = 0; i < 25; i++) {
				state.state[i] = uint64(state256[i]);
			}
			if (data.length < KECCAK_ROUND_INPUT) {
				break;
			}
			data = data[KECCAK_ROUND_INPUT:];
		}
	}

	function clearSplitProof() external {
		delete keccakStates[msg.sender];
	}
}
