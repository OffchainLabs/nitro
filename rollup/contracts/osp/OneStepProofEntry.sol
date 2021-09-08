//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Deserialize.sol";
import "../state/Machines.sol";
import "./IOneStepProver.sol";

contract OneStepProofEntry {
	IOneStepProver prover0;

	constructor(IOneStepProver prover0_) {
		prover0 = prover0_;
	}

	function proveOneStep(bytes32 beforeHash, bytes calldata proof) external view returns (bytes32 afterHash) {
		Machine memory mach;
		uint256 offset = 0;
		(mach, offset) = Deserialize.machine(proof, offset);
		require(Machines.hash(mach) == beforeHash, "MACHINE_BEFORE_HASH");

		if (mach.instructions.proved.length == 0 && mach.instructions.remainingHash == 0) {
			mach.halted = true;
		} else {
			uint8 opcode = Instructions.peek(mach.instructions).opcode;
			if (opcode >= 0 && opcode <= 0xFF) {
				mach = prover0.executeOneStep(mach, proof[offset:]);
			} else {
				revert("TODO: instruction prover not specified");
			}
		}

		return Machines.hash(mach);
	}
}
