//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "./ChallengeLib.sol";

contract ChallengeCore {
    bytes32 public challengeStateHash;

    function extractChallengeSegment(
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition
    ) internal view returns (uint256 segmentStart, uint256 segmentLength) {
        require(
            challengeStateHash ==
                ChallengeLib.hashChallengeState(
                    oldSegmentsStart,
                    oldSegmentsLength,
                    oldSegments
                ),
            "BIS_STATE"
        );
        if (
            oldSegments.length < 2 ||
            challengePosition >= oldSegments.length - 1
        ) {
            revert("BAD_CHALLENGE_POS");
        }
        uint256 oldChallengeDegree = oldSegments.length - 1;
        segmentLength = oldSegmentsLength / oldChallengeDegree;
        // Intentionally done before challengeLength is potentially added to for the final segment
        segmentStart = oldSegmentsStart + segmentLength * challengePosition;
        if (challengePosition == oldSegments.length - 2) {
            segmentLength += oldSegmentsLength % oldChallengeDegree;
        }
    }
}
