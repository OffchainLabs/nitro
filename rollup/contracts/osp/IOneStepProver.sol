//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Machines.sol";
import "../state/Instructions.sol";

abstract contract IOneStepProver {
	function executeOneStep(Machine calldata mach, Instruction calldata instruction, bytes calldata proof) virtual view external returns (Machine memory result);
}
