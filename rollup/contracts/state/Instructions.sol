//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

struct Instruction {
	uint16 opcode;
	uint256 argumentData;
}

struct InstructionWindow {
	Instruction[] proved;
	bytes32 remainingHash;
}

library Instructions {
	uint16 constant UNREACHABLE = 0x00;
	uint16 constant NOP = 0x01;
	uint16 constant BLOCK = 0x02;
	uint16 constant BRANCH = 0x0C;
	uint16 constant BRANCH_IF = 0x0D;
	uint16 constant LOCAL_GET = 0x20;
	uint16 constant LOCAL_SET = 0x21;
	uint16 constant GLOBAL_GET = 0x23;
	uint16 constant GLOBAL_SET = 0x24;
	uint16 constant DROP = 0x1A;
	uint16 constant I32_CONST = 0x41;
	uint16 constant I64_CONST = 0x42;
	uint16 constant F32_CONST = 0x43;
	uint16 constant F64_CONST = 0x44;
	uint16 constant I32_EQZ = 0x45;
	uint16 constant I32_ADD = 0x6A;
	uint16 constant I64_ADD = 0x7C;

	uint16 constant END_BLOCK = 0x8000;
	uint16 constant END_BLOCK_IF = 0x8001;
	uint16 constant INIT_FRAME = 0x8002;
	uint16 constant ARBITRARY_JUMP_IF = 0x8003;

	function hash(Instruction memory inst) internal pure returns (bytes32) {
		return keccak256(abi.encodePacked("Instruction:", inst.opcode, inst.argumentData));
	}

	function hash(InstructionWindow memory window) internal pure returns (bytes32 h) {
		h = window.remainingHash;
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

	function newNop() internal pure returns (Instruction memory) {
		return Instruction({
			opcode: NOP,
			argumentData: 0
		});
	}
}
