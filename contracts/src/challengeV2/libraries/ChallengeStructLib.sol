// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntities.sol";

// CHRIS: TODO: rename later when we dont have conflicting names
library ChallengeStructLib {
    // CHRIS: TODO: rename args?
    function id(bytes32 challengeOriginId, ChallengeType cType) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(challengeOriginId, cType));
    }

    function exists(Challenge storage challenge) internal view returns (bool) {
        // CHRIS: TODO: this and the other exists can be tested for invariants
        return challenge.rootId != 0;
    }
}