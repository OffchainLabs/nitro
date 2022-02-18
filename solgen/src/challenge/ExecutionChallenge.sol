//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../libraries/DelegateCallAware.sol";
import "../osp/IOneStepProofEntry.sol";
import "./IChallengeResultReceiver.sol";
import "./ChallengeLib.sol";
import "./ChallengeCore.sol";
import "./IChallenge.sol";
import "@openzeppelin/contracts/proxy/beacon/BeaconProxy.sol";
import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract ExecutionChallenge is ChallengeCore, DelegateCallAware, IChallenge {
    event OneStepProofCompleted();

    IOneStepProofEntry public osp;
    ExecutionContext public execCtx;

    function initialize(
        IOneStepProofEntry osp_,
        IChallengeResultReceiver resultReceiver_,
        ExecutionContext memory execCtx_,
        bytes32[2] memory startAndEndHashes,
        uint256 challenge_length,
        address asserter_,
        address challenger_,
        uint256 asserterTimeLeft_,
        uint256 challengerTimeLeft_
    ) public onlyDelegated {
        require(address(resultReceiver) == address(0), "ALREADY_INIT");
        require(address(resultReceiver_) != address(0), "NO_RESULT_RECEIVER");
        require(challenge_length <= OneStepProofEntryLib.MAX_STEPS, "CHALLENGE_TOO_LONG");
        osp = osp_;
        resultReceiver = resultReceiver_;
        execCtx = execCtx_;
        bytes32[] memory segments = new bytes32[](2);
        segments[0] = startAndEndHashes[0];
        segments[1] = startAndEndHashes[1];
        challengeStateHash = ChallengeLib.hashChallengeState(0, challenge_length, segments);
        asserter = asserter_;
        challenger = challenger_;
        asserterTimeLeft = asserterTimeLeft_;
        challengerTimeLeft = challengerTimeLeft_;
        lastMoveTimestamp = block.timestamp;
        turn = Turn.CHALLENGER;

        emit InitiatedChallenge();
        emit Bisected(
            challengeStateHash,
            0,
            challenge_length,
            segments
        );
    }

    function oneStepProveExecution(
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition,
        bytes calldata proof
    ) external takeTurn {
        (uint256 challengeStart, uint256 challengeLength) = extractChallengeSegment(
            oldSegmentsStart,
            oldSegmentsLength,
            oldSegments,
            challengePosition
        );
        require(challengeLength == 1, "TOO_LONG");

        bytes32 afterHash = osp.proveOneStep(
            execCtx,
            challengeStart,
            oldSegments[challengePosition],
            proof
        );
        require(
            afterHash != oldSegments[challengePosition + 1],
            "SAME_OSP_END"
        );

        emit OneStepProofCompleted();
        _currentWin();
    }

    function clearChallenge() external override {
        require(msg.sender == address(resultReceiver), "NOT_RES_RECEIVER");
        turn = Turn.NO_CHALLENGE;
    }

    function _currentWin() private {
        // As a safety measure, challenges can only be resolved by timeouts during mainnet beta.
        // As state is 0, no move is possible. The other party will lose via timeout
        challengeStateHash = bytes32(0);

        // if (turn == Turn.ASSERTER) {
        //     _asserterWin();
        // } else if (turn == Turn.CHALLENGER) {
        //     _challengerWin();
        // } else {
        // 	   revert(NO_TURN);
        // }
    }
}
