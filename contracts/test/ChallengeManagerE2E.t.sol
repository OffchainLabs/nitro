// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "../src/DataEntities.sol";
import "./MockAssertionChain.sol";
import "../src/ChallengeManager.sol";

contract ChallengeManagerE2ETest is Test {
    function generateHash(uint256 iterations) internal pure returns (bytes32 h) {
        // seed
        h = 0xf19f64ef5b8c788ff3f087b4f75bc6596a6aaa3c9048bbbbe990fa0870261385;
        for (uint256 i = 0; i < iterations; i++) {
            h = keccak256(abi.encodePacked(h));
        }
    }

    bytes32 genesisHash = generateHash(0);
    bytes32 h1 = generateHash(1);
    bytes32 h2 = generateHash(2);
    uint256 height1 = 10;
    uint256 inboxSeenCount1 = 5;

    uint256 miniStakeVal = 1 ether;
    uint256 challengePeriod = 1000;

    function testCanConfirmPs() public {
        MockAssertionChain assertionChain = new MockAssertionChain();
        ChallengeManager challengeManager = new ChallengeManager(assertionChain, miniStakeVal, challengePeriod);

        bytes32 genesis = assertionChain.addAssertionUnsafe(0, 0, 0, genesisHash, 0);
        bytes32 a1 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h1, 0);
        bytes32 a2 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h2, 0);

        bytes32 challengeId = assertionChain.createChallenge(a1, a2, challengeManager);
        bytes32 v1Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: challengeId,
                claimId: a1,
                height: height1,
                historyCommitment: h1,
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: h1,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(h1),
            abi.encodePacked(uint256(0))
        );

        vm.warp(challengePeriod + 2);

        challengeManager.confirmForPsTimer(v1Id);
    }
}
