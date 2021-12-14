//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../state/Machines.sol";

library ChallengeLib {
    function hashChallengeState(
        uint256 segmentsStart,
        uint256 segmentsLength,
        bytes32[] memory segments
    ) internal pure returns (bytes32) {
        return
            keccak256(
                abi.encodePacked(segmentsStart, segmentsLength, segments)
            );
    }

    function extractChallengeSegment(
        bytes32 challengeStateHash,
        uint256 oldSegmentsStart,
        uint256 oldSegmentsLength,
        bytes32[] calldata oldSegments,
        uint256 challengePosition
    ) internal pure returns (uint256 segmentStart, uint256 segmentLength) {
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

    function blockStateHash(MachineStatus status, bytes32 globalStateHash)
        internal
        pure
        returns (bytes32)
    {
        if (status == MachineStatus.FINISHED) {
            return keccak256(abi.encodePacked("Block state:", globalStateHash));
        } else if (status == MachineStatus.ERRORED) {
            return
                keccak256(
                    abi.encodePacked("Block state, errored:", globalStateHash)
                );
        } else if (status == MachineStatus.TOO_FAR) {
            return keccak256(abi.encodePacked("Block state, too far:"));
        } else {
            revert("BAD_BLOCK_STATUS");
        }
    }
}
