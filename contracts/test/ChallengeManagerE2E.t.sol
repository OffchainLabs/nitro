// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../src/challengeV2/DataEntities.sol";
import "./MockAssertionChain.sol";
import "../src/challengeV2/ChallengeManagerImpl.sol";
import "../src/osp/IOneStepProofEntry.sol";
import "./challengeV2/Utils.sol";

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
    Random rand = new Random();
    bytes32 genesisHash = rand.hash();
    bytes32 h1 = rand.hash();
    bytes32 h2 = rand.hash();
    uint256 height1 = 11;

    uint256 miniStakeVal = 1 ether;
    uint256 challengePeriodSec = 1000;

    function deploy() internal returns (MockAssertionChain, ChallengeManagerImpl, bytes32) {
        MockAssertionChain assertionChain = new MockAssertionChain();
        ChallengeManagerImpl challengeManager =
            new ChallengeManagerImpl(assertionChain, miniStakeVal, challengePeriodSec, new MockOneStepProofEntry());
        bytes32 genesis = assertionChain.addAssertionUnsafe(0, 0, 0, genesisHash, 0);

        return (assertionChain, challengeManager, genesis);
    }

    function deployAndInitChallenge()
        internal
        returns (MockAssertionChain, ChallengeManagerImpl, bytes32, bytes32, bytes32, bytes32)
    {
        (MockAssertionChain assertionChain, ChallengeManagerImpl challengeManager, bytes32 genesis) = deploy();
        uint256 inboxSeenCount1 = 5;

        bytes32 a1 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h1, 0);
        bytes32 a2 = assertionChain.addAssertion(genesis, height1, inboxSeenCount1, h2, 0);

        bytes32 challengeId = assertionChain.createChallenge(a1, a2, challengeManager);

        return (assertionChain, challengeManager, genesis, a1, a2, challengeId);
    }

    function randomLeavesAndExpansion(uint256 height) internal returns (bytes32[] memory, bytes32[] memory) {
        bytes32[] memory leaves = rand.hashes(height);
        bytes32[] memory exp = MerkleTreeLib.expansionFromLeaves(leaves, 0, height);

        return (leaves, exp);
    }

    function testCanConfirmPs() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1,, bytes32 challengeId) = deployAndInitChallenge();
        (, bytes32[] memory exp) = randomLeavesAndExpansion(height1);

        bytes32 v1Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: challengeId,
                claimId: a1,
                height: height1,
                historyRoot: MerkleTreeLib.root(exp),
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: h1,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(h1),
            abi.encodePacked(uint256(0))
        );

        vm.warp(challengePeriodSec + 2);

        challengeManager.confirmForPsTimer(v1Id);

        assertEq(challengeManager.winningClaim(challengeId), a1);
    }

    function bisect(
        IChallengeManager challengeManager,
        bytes32 currentId,
        bytes32[] memory leaves,
        uint256 bisectionHeight,
        uint256 currentHeight
    ) internal returns (bytes32) {
        bytes32[] memory preExp = MerkleTreeLib.expansionFromLeaves(leaves, 0, bisectionHeight);
        // height 8
        return challengeManager.bisect(
            currentId,
            MerkleTreeLib.root(preExp),
            abi.encode(
                preExp,
                MerkleTreeLib.generatePrefixProof(
                    bisectionHeight, ArrayUtils.slice(leaves, bisectionHeight, currentHeight)
                )
            )
        );
    }

    function testCanConfirmSubChallenge() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1, bytes32 a2, bytes32 blockChallengeId) =
            deployAndInitChallenge();
        (bytes32[] memory leaves1, bytes32[] memory exp1) = randomLeavesAndExpansion(height1);

        bytes32 v1Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: blockChallengeId,
                claimId: a1,
                height: height1,
                historyRoot: MerkleTreeLib.root(exp1),
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: h1,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(h1),
            abi.encodePacked(uint256(0))
        );

        (bytes32[] memory leaves2, bytes32[] memory exp2) = randomLeavesAndExpansion(height1);
        bytes32 v2Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: blockChallengeId,
                claimId: a2,
                height: height1,
                historyRoot: MerkleTreeLib.root(exp2),
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: h2,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(h2),
            abi.encodePacked(uint256(0))
        );

        (bytes32[5] memory b1,) = bisectToRoot(challengeManager, v1Id, v2Id, leaves1, leaves2);

        bytes32 bigStepChallengeId =
            challengeManager.createSubChallenge(challengeManager.getVertex(b1[0]).predecessorId);

        // only add one leaf
        bytes32 bsLeaf1 = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: bigStepChallengeId,
                claimId: b1[0],
                height: height1,
                historyRoot: h1,
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: h1,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(h1),
            abi.encodePacked(uint256(0))
        );

        vm.warp(challengePeriodSec + 2);

        // confirm in the sub challenge by ps
        challengeManager.confirmForPsTimer(bsLeaf1);
        // confirm because of sub challenge
        challengeManager.confirmForSucessionChallengeWin(b1[0]);
        // confirm the rest sequentially by ps
        challengeManager.confirmForPsTimer(b1[1]);
        challengeManager.confirmForPsTimer(b1[2]);
        challengeManager.confirmForPsTimer(b1[3]);
        challengeManager.confirmForPsTimer(b1[4]);

        assertEq(challengeManager.winningClaim(blockChallengeId), a1);
    }

    function bisectToRoot(
        IChallengeManager challengeManager,
        bytes32 winningId,
        bytes32 losingId,
        bytes32[] memory winningLeaves,
        bytes32[] memory losingLeaves
    ) internal returns (bytes32[5] memory, bytes32[5] memory) {
        bytes32[5] memory winningVertices;
        bytes32[5] memory losingVertices;

        winningVertices[4] = winningId;
        losingVertices[4] = losingId;

        // height 8
        winningVertices[3] = bisect(challengeManager, winningVertices[4], winningLeaves, 8, winningLeaves.length);
        losingVertices[3] = bisect(challengeManager, losingVertices[4], losingLeaves, 8, losingLeaves.length);

        // height 4
        winningVertices[2] = bisect(challengeManager, winningVertices[3], winningLeaves, 4, 8);
        losingVertices[2] = bisect(challengeManager, losingVertices[3], losingLeaves, 4, 8);

        // height 2
        winningVertices[1] = bisect(challengeManager, winningVertices[2], winningLeaves, 2, 4);
        losingVertices[1] = bisect(challengeManager, losingVertices[2], losingLeaves, 2, 4);

        // height 1
        winningVertices[0] = bisect(challengeManager, winningVertices[1], winningLeaves, 1, 2);
        losingVertices[0] = bisect(challengeManager, losingVertices[1], losingLeaves, 1, 2);

        return (winningVertices, losingVertices);
    }

    function addLeaf(
        IChallengeManager challengeManager,
        bytes32 challengeId,
        bytes32 claimId,
        bytes32 historyRoot,
        uint256 height,
        bytes memory proof2
    ) internal returns (bytes32) {
        return challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: challengeId,
                claimId: claimId,
                height: height,
                historyRoot: historyRoot,
                firstState: genesisHash,
                firstStatehistoryProof: "",
                lastState: historyRoot,
                lastStatehistoryProof: ""
            }),
            abi.encodePacked(historyRoot),
            proof2
        );
    }

    function addLeafsAndBisectToSubChallenge(
        IChallengeManager challengeManager,
        bytes32 challengeId,
        bytes32 claimId1,
        bytes32 claimId2,
        bytes memory addLeafProof2
    ) internal returns (bytes32[5] memory, bytes32[5] memory) {
        (bytes32[] memory leaves1, bytes32[] memory exp1) = randomLeavesAndExpansion(height1);
        (bytes32[] memory leaves2, bytes32[] memory exp2) = randomLeavesAndExpansion(height1);

        bytes32 blockLeaf1Id =
            addLeaf(challengeManager, challengeId, claimId1, MerkleTreeLib.root(exp1), height1, addLeafProof2);
        bytes32 blockLeaf2Id =
            addLeaf(challengeManager, challengeId, claimId2, MerkleTreeLib.root(exp2), height1, addLeafProof2);
        (bytes32[5] memory challengeWinningVertices, bytes32[5] memory challengeLosingVertices) =
            bisectToRoot(challengeManager, blockLeaf1Id, blockLeaf2Id, leaves1, leaves2);

        return (challengeWinningVertices, challengeLosingVertices);
    }

    function testCanConfirmFromOneStep() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1, bytes32 a2, bytes32 blockChallengeId) =
            deployAndInitChallenge();

        (bytes32[5] memory blockWinners, bytes32[5] memory blockLosers) =
            addLeafsAndBisectToSubChallenge(challengeManager, blockChallengeId, a1, a2, abi.encodePacked(uint256(0)));

        bytes32 bigStepChallengeId =
            challengeManager.createSubChallenge(challengeManager.getVertex(blockWinners[0]).predecessorId);
        (bytes32[5] memory bigStepWinners, bytes32[5] memory bigStepLosers) = addLeafsAndBisectToSubChallenge(
            challengeManager, bigStepChallengeId, blockWinners[0], blockLosers[0], abi.encodePacked(uint256(0))
        );

        bytes32 smallStepChallengeId =
            challengeManager.createSubChallenge(challengeManager.getVertex(bigStepWinners[0]).predecessorId);

        (bytes32[5] memory smallStepWinners,) = addLeafsAndBisectToSubChallenge(
            challengeManager, smallStepChallengeId, bigStepWinners[0], bigStepLosers[0], abi.encodePacked(height1)
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

        vm.warp(challengePeriodSec + 2);
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
