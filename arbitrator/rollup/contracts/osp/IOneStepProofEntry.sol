//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./IOneStepProver.sol";

interface IOneStepProofEntry {
    function proveOneStep(
        ExecutionContext calldata execCtx,
        bytes32 beforeHash,
        bytes calldata proof
    ) external view returns (bytes32 afterHash);
}
