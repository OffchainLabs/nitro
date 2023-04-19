// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../../src/challengeV2/EdgeChallengeManager.sol";

contract MockOneStepProofEntry is IOneStepProofEntry {
    function proveOneStep(
        ExecutionContext calldata,
        uint256,
        bytes32,
        bytes calldata proof
    ) external view returns (bytes32 afterHash) {
        return bytes32(proof);
    }
}
