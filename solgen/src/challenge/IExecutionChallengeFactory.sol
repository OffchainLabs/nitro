//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../osp/IOneStepProofEntry.sol";
import "./IExecutionChallenge.sol";
import "./IChallengeResultReceiver.sol";

interface IExecutionChallengeFactory {
    function createChallenge(
        IChallengeResultReceiver resultReceiver_,
        ExecutionContext memory execCtx_,
        bytes32[2] memory startAndEndHashes,
        uint256 challenge_length_,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) external returns (IExecutionChallenge);
}
