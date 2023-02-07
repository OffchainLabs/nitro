// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "../src/challengeV2/DataEntities.sol";
import "./MockAssertionChain.sol";
import "../src/challengeV2/ChallengeManagerImpl.sol";
import "../src/osp/IOneStepProofEntry.sol";

contract MockOneStepProofEntry is IOneStepProofEntry {
    function proveOneStep(ExecutionContext calldata, uint256, bytes32, bytes calldata proof)
        external
        view
        returns (bytes32 afterHash)
    {
        return bytes32(proof);
    }
}

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

    function deploy() internal returns (MockAssertionChain, ChallengeManagerImpl, bytes32) {
        MockAssertionChain assertionChain = new MockAssertionChain();
        ChallengeManagerImpl challengeManager =
            new ChallengeManagerImpl(assertionChain, miniStakeVal, challengePeriod, new MockOneStepProofEntry());
        bytes32 genesis = assertionChain.addAssertionUnsafe(0, 0, 0, genesisHash, 0);

        return (assertionChain, challengeManager, genesis);
    }

    function deployAndInitChallenge()
        internal
        returns (MockAssertionChain, ChallengeManagerImpl, bytes32, bytes32, bytes32, bytes32)
    {
        (MockAssertionChain assertionChain, ChallengeManagerImpl challengeManager, bytes32 genesis) = deploy();

        bytes32 a1 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h1, 0);
        bytes32 a2 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h2, 0);

        bytes32 challengeId = assertionChain.createChallenge(a1, a2, challengeManager);

        return (assertionChain, challengeManager, genesis, a1, a2, challengeId);
    }

    function testCanConfirmPs() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1,, bytes32 challengeId) = deployAndInitChallenge();

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
        (, ChallengeManagerImpl challengeManager,, bytes32 a1, bytes32 a2, bytes32 blockChallengeId) =
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

    function bisectToRoot(IChallengeManager challengeManager, bytes32 winningLeaf, bytes32 losingLeaf)
        internal
        returns (bytes32[5] memory, bytes32[5] memory)
    {
        bytes32[5] memory winningVertices;
        bytes32[5] memory losingVertices;

        winningVertices[4] = winningLeaf;
        losingVertices[4] = losingLeaf;

        // height 8
        winningVertices[3] = challengeManager.bisect(winningVertices[4], h1, "");
        losingVertices[3] = challengeManager.bisect(losingVertices[4], h2, "");

        // height 4
        winningVertices[2] = challengeManager.bisect(winningVertices[3], h1, "");
        losingVertices[2] = challengeManager.bisect(losingVertices[3], h2, "");

        // height 2
        winningVertices[1] = challengeManager.bisect(winningVertices[2], h1, "");
        losingVertices[1] = challengeManager.bisect(losingVertices[2], h2, "");

        // height 1
        winningVertices[0] = challengeManager.bisect(winningVertices[1], h1, "");
        losingVertices[0] = challengeManager.bisect(losingVertices[1], h2, "");

        return (winningVertices, losingVertices);
    }

    function addLeaf(
        IChallengeManager challengeManager,
        bytes32 challengeId,
        bytes32 claimId,
        bytes32 historyCommitment,
        bytes memory proof2
    ) internal returns (bytes32) {
        return challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: challengeId,
                claimId: claimId,
                height: height1,
                historyCommitment: historyCommitment,
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: historyCommitment,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(historyCommitment),
            proof2
        );
    }

    function addLeafsAndBisectToSubChallenge(
        IChallengeManager challengeManager,
        bytes32 challengeId,
        bytes32 claimId1,
        bytes32 historyCommitment1,
        bytes32 claimId2,
        bytes32 historyCommitment2,
        bytes memory addLeafProof2
    ) internal returns (bytes32[5] memory, bytes32[5] memory) {
        bytes32 blockLeaf1Id = addLeaf(challengeManager, challengeId, claimId1, historyCommitment1, addLeafProof2);
        bytes32 blockLeaf2Id = addLeaf(challengeManager, challengeId, claimId2, historyCommitment2, addLeafProof2);
        (bytes32[5] memory challengeWinningVertices, bytes32[5] memory challengeLosingVertices) =
            bisectToRoot(challengeManager, blockLeaf1Id, blockLeaf2Id);

        return (challengeWinningVertices, challengeLosingVertices);
    }

    function testCanConfirmFromOneStep() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1, bytes32 a2, bytes32 blockChallengeId) =
            deployAndInitChallenge();

        (bytes32[5] memory blockWinners, bytes32[5] memory blockLosers) = addLeafsAndBisectToSubChallenge(
            challengeManager, blockChallengeId, a1, h1, a2, h2, abi.encodePacked(uint256(0))
        );

        bytes32 bigStepChallengeId =
            challengeManager.createSubChallenge(challengeManager.getVertex(blockWinners[0]).predecessorId);
        (bytes32[5] memory bigStepWinners, bytes32[5] memory bigStepLosers) = addLeafsAndBisectToSubChallenge(
            challengeManager, bigStepChallengeId, blockWinners[0], h1, blockLosers[0], h2, abi.encodePacked(uint256(0))
        );

        bytes32 smallStepChallengeId =
            challengeManager.createSubChallenge(challengeManager.getVertex(bigStepWinners[0]).predecessorId);

        (bytes32[5] memory smallStepWinners,) = addLeafsAndBisectToSubChallenge(
            challengeManager,
            smallStepChallengeId,
            bigStepWinners[0],
            h1,
            bigStepLosers[0],
            h2,
            abi.encodePacked(height1)
        );

        challengeManager.createSubChallenge(challengeManager.getVertex(smallStepWinners[0]).predecessorId);
        uint256 height = challengeManager.getVertex(smallStepWinners[0]).height - 1;

        challengeManager.executeOneStep(
            smallStepWinners[0],
            OneStepData({
                execCtx: ExecutionContext({maxInboxMessagesRead: 0, bridge: IBridge(address(0))}),
                machineStep: height,
                beforeHash: genesisHash,
                proof: abi.encodePacked(bytes32(smallStepWinners[0]))
            }),
            "",
            ""
        );

        challengeManager.confirmForSucessionChallengeWin(smallStepWinners[0]);

        vm.warp(challengePeriod + 2);
        challengeManager.confirmForPsTimer(smallStepWinners[1]);
        challengeManager.confirmForPsTimer(smallStepWinners[2]);
        challengeManager.confirmForPsTimer(smallStepWinners[3]);
        challengeManager.confirmForPsTimer(smallStepWinners[4]);

        challengeManager.confirmForSucessionChallengeWin(bigStepWinners[0]);
        challengeManager.confirmForPsTimer(bigStepWinners[1]);
        challengeManager.confirmForPsTimer(bigStepWinners[2]);
        challengeManager.confirmForPsTimer(bigStepWinners[3]);
        challengeManager.confirmForPsTimer(bigStepWinners[4]);

        challengeManager.confirmForSucessionChallengeWin(blockWinners[0]);
        challengeManager.confirmForPsTimer(blockWinners[1]);
        challengeManager.confirmForPsTimer(blockWinners[2]);
        challengeManager.confirmForPsTimer(blockWinners[3]);
        challengeManager.confirmForPsTimer(blockWinners[4]);

        assertEq(challengeManager.winningClaim(blockChallengeId), a1);
    }
}
