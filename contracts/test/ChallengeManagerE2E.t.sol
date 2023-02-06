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

    function deploy() internal returns (MockAssertionChain, ChallengeManager, bytes32) {
        MockAssertionChain assertionChain = new MockAssertionChain();
        ChallengeManager challengeManager = new ChallengeManager(assertionChain, miniStakeVal, challengePeriod);
        bytes32 genesis = assertionChain.addAssertionUnsafe(0, 0, 0, genesisHash, 0);

        return (assertionChain, challengeManager, genesis);
    }

    function deployAndInitChallenge()
        internal
        returns (MockAssertionChain, ChallengeManager, bytes32, bytes32, bytes32, bytes32)
    {
        (MockAssertionChain assertionChain, ChallengeManager challengeManager, bytes32 genesis) = deploy();

        bytes32 a1 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h1, 0);
        bytes32 a2 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h2, 0);

        bytes32 challengeId = assertionChain.createChallenge(a1, a2, challengeManager);

        return (assertionChain, challengeManager, genesis, a1, a2, challengeId);
    }

    function testCanConfirmPs() public {
        (, ChallengeManager challengeManager,, bytes32 a1,, bytes32 challengeId) = deployAndInitChallenge();

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

        assertEq(challengeManager.winningClaim(challengeId), a1);
    }

    function testCanConfirmSubChallenge() public {
        (, ChallengeManager challengeManager,, bytes32 a1, bytes32 a2, bytes32 blockChallengeId) =
            deployAndInitChallenge();

        bytes32 v1Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: blockChallengeId,
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

        bytes32 v2Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: blockChallengeId,
                claimId: a2,
                height: height1,
                historyCommitment: h2,
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: h2,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(h2),
            abi.encodePacked(uint256(0))
        );

        bytes32 b11;
        bytes32 b12;
        bytes32 b14;
        bytes32 b18;
        {
            // height 8
            b18 = challengeManager.bisect(v1Id, h1, "");
            bytes32 b28 = challengeManager.bisect(v2Id, h2, "");

            // height 4
            b14 = challengeManager.bisect(b18, h1, "");
            bytes32 b24 = challengeManager.bisect(b28, h2, "");

            // height 2
            b12 = challengeManager.bisect(b14, h1, "");
            bytes32 b22 = challengeManager.bisect(b24, h2, "");

            // height 1
            b11 = challengeManager.bisect(b12, h1, "");
            challengeManager.bisect(b22, h2, "");
        }

        bytes32 rootId = challengeManager.getVertex(b11).predecessorId;

        bytes32 bigStepChallengeId = challengeManager.createSubChallenge(rootId);

        // only add one leaf
        bytes32 bsLeaf1 = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: bigStepChallengeId,
                claimId: b11,
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

        // confirm in the sub challenge by ps
        challengeManager.confirmForPsTimer(bsLeaf1);
        // confirm because of sub challenge
        challengeManager.confirmForSucessionChallengeWin(b11);
        // confirm the rest sequentially by ps
        challengeManager.confirmForPsTimer(b12);
        challengeManager.confirmForPsTimer(b14);
        challengeManager.confirmForPsTimer(b18);
        challengeManager.confirmForPsTimer(v1Id);

        assertEq(challengeManager.winningClaim(blockChallengeId), a1);
    }
}
