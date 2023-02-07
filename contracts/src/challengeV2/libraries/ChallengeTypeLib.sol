// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntities.sol";

// CHRIS: TODO: check that all the lib functions have the correct visibility

library ChallengeTypeLib {
    function nextType(ChallengeType cType) external pure returns (ChallengeType) {
        if (cType == ChallengeType.Block) {
            return ChallengeType.BigStep;
        } else if (cType == ChallengeType.BigStep) {
            return ChallengeType.SmallStep;
            // CHRIS: TODO: everywhere we have a switch we should check we have a revert for everything else
        } else if (cType == ChallengeType.SmallStep) {
            return ChallengeType.OneStep;
        } else {
            revert("Cannot get next challenge type for one step challenge");
        }
    }
}
