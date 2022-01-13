//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../challenge/ExecutionChallenge.sol";

contract SingleExecutionChallenge is ExecutionChallenge {
    constructor(
        IOneStepProofEntry osp_,
        IChallengeResultReceiver resultReceiver_,
        ExecutionContext memory execCtx_,
        bytes32[2] memory startAndEndHashes,
        uint256 numSteps_,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) {
        isMasterCopy = false;
        initialize(
            osp_,
            resultReceiver_,
            execCtx_,
            startAndEndHashes,
            numSteps_,
            asserter_,
            challenger_,
            asserterTimeLeft_,
            challengerTimeLeft_
        );
    }
}
