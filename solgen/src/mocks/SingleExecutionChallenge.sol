//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../challenge/ExecutionChallenge.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract SingleExecutionChallenge is ERC1967Proxy {
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
    ) ERC1967Proxy(
        address(new ExecutionChallenge()),
        abi.encodeWithSelector(
            ExecutionChallenge.initialize.selector,
            osp_,
            resultReceiver_,
            execCtx_,
            startAndEndHashes,
            numSteps_,
            asserter_,
            challenger_,
            asserterTimeLeft_,
            challengerTimeLeft_
        )
    ) {}
}
