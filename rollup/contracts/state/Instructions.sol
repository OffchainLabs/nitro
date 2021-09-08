//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

struct Instruction {
	uint8 opcode;
	uint256 argument_data;
}

struct InstructionWindow {
	Instruction[] proved;
	bytes32 remaining_hash;
}

library Instructions {
	uint8 constant UNREACHABLE = 0x00;
	uint8 constant NOP = 0x01;
	uint8 constant BLOCK = 0x02;
	uint8 constant END_BLOCK = 0x03;
	uint8 constant END_BLOCK_IF = 0x04;
	uint8 constant INIT_FRAME = 0x05;
	uint8 constant BRANCH = 0x0C;
	uint8 constant BRANCH_IF = 0x0D;
	uint8 constant DROP = 0x1A;
	uint8 constant I32_CONST = 0x41;
	uint8 constant I64_CONST = 0x42;
	uint8 constant F32_CONST = 0x43;
	uint8 constant F64_CONST = 0x44;
	uint8 constant I32_EQZ = 0x45;
	uint8 constant I32_ADD = 0x6A;
	uint8 constant I64_ADD = 0x7C;

	function hash(Instruction memory inst) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked("Instruction:", inst.opcode, inst.argument_data));
	}

	function hash(InstructionWindow memory window) internal pure returns (bytes32 h) {
		h = window.remaining_hash;
		for (uint256 i = 0; i < window.proved.length; i++) {
			h = keccak256(abi.encodePacked("Instruction stack:", hash(window.proved[i]), h));
		}
	}

	function peek(InstructionWindow memory window) internal pure returns (Instruction memory) {
		require(window.proved.length == 1, "BAD_WINDOW_LENGTH");
		return window.proved[0];
	}

	function pop(InstructionWindow memory window) internal pure returns (Instruction memory inst) {
		require(window.proved.length == 1, "BAD_WINDOW_LENGTH");
		inst = window.proved[0];
		window.proved = new Instruction[](0);
	}

	function maxOpcode() internal pure returns (uint8) {
		return 0xFF;
	}

	function newNop() internal pure returns (Instruction memory) {
		return Instruction({
			opcode: NOP,
			argument_data: 0
		});
	}
}
