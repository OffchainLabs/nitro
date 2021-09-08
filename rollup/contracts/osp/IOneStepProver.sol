//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Machines.sol";

abstract contract IOneStepProver {
	function executeOneStep(Machine calldata mach, bytes calldata proof) virtual view external returns (Machine memory result);
}
