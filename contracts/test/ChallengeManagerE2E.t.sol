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

    function genesisStates() internal view returns (bytes32[] memory) {
        bytes32[] memory genStates = new bytes32[](1);
        genStates[0] = genesisHash;
        return genStates;
    }

    bytes32 h1 = rand.hash();
    bytes32 h2 = rand.hash();
    uint256 height1 = 18;

    uint256 miniStakeVal = 1 ether;
    uint256 challengePeriodSec = 1000;

    uint256 genesisHeight = 2;

    function deploy() internal returns (MockAssertionChain, ChallengeManagerImpl, bytes32) {
        MockAssertionChain assertionChain = new MockAssertionChain();
        ChallengeManagerImpl challengeManager =
            new ChallengeManagerImpl(assertionChain, miniStakeVal, challengePeriodSec, new MockOneStepProofEntry());
        bytes32 genesis =
            assertionChain.addAssertionUnsafe(0, genesisHeight, 0, keccak256(abi.encodePacked(genesisHash)), 0);

        return (assertionChain, challengeManager, genesis);
    }

    function deployAndInitChallenge()
        internal
        returns (MockAssertionChain, ChallengeManagerImpl, bytes32, bytes32, bytes32, bytes32)
    {
        (MockAssertionChain assertionChain, ChallengeManagerImpl challengeManager, bytes32 genesis) = deploy();
        uint256 inboxSeenCount1 = 5;

        // add one since heights are zero indexed in the history states
        bytes32 a1 = assertionChain.addAssertion(genesis, genesisHeight + height1 + 1, inboxSeenCount1, h1, 0);
        bytes32 a2 = assertionChain.addAssertion(genesis, genesisHeight + height1 + 1, inboxSeenCount1, h2, 0);

        bytes32 challengeId = assertionChain.createChallenge(a1, a2, challengeManager);

        return (assertionChain, challengeManager, genesis, a1, a2, challengeId);
    }

    function appendRandomStates(bytes32[] memory currentStates, uint256 numStates)
        internal
        returns (bytes32[] memory, bytes32[] memory)
    {
        bytes32[] memory newStates = rand.hashes(numStates);
        bytes32[] memory full = ArrayUtils.concat(currentStates, newStates);
        bytes32[] memory exp = ProofUtils.expansionFromLeaves(full, 0, full.length);

        return (full, exp);
    }

    function testCanConfirmPs() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1,, bytes32 challengeId) = deployAndInitChallenge();
        (bytes32[] memory states, bytes32[] memory exp) = appendRandomStates(genesisStates(), height1);

        bytes32[] memory firstProof = ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), 0);
        bytes32[] memory lastProof = ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), states.length - 1);

        bytes32 v1Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: challengeId,
                claimId: a1,
                height: height1,
                historyRoot: MerkleTreeLib.root(exp),
                firstState: genesisHash,
                firstStatehistoryProof: firstProof,
                lastState: states[states.length - 1],
                lastStatehistoryProof: lastProof
            }),
            abi.encodePacked(states[states.length - 1]),
            abi.encodePacked(uint256(0))
        );

        vm.warp(challengePeriodSec + 2);

        challengeManager.confirmForPsTimer(v1Id);

        assertEq(challengeManager.winningClaim(challengeId), a1);
    }

    function bisect(
        IChallengeManager challengeManager,
        bytes32 currentId,
        bytes32[] memory states,
        uint256 bisectionSize,
        uint256 currentSize
    ) internal returns (bytes32) {
        bytes32[] memory preExp = ProofUtils.expansionFromLeaves(states, 0, bisectionSize);
        bytes32[] memory newStates = ArrayUtils.slice(states, bisectionSize, currentSize);
        return challengeManager.bisect(
            currentId,
            MerkleTreeLib.root(preExp),
            abi.encode(preExp, ProofUtils.generatePrefixProof(bisectionSize, newStates))
        );
    }

    function testCanConfirmSubChallenge() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1, bytes32 a2, bytes32 blockChallengeId) =
            deployAndInitChallenge();
        (bytes32[] memory states1, bytes32[] memory exp1) = appendRandomStates(genesisStates(), height1);

        bytes32 v1Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: blockChallengeId,
                claimId: a1,
                height: height1,
                historyRoot: MerkleTreeLib.root(exp1),
                firstState: genesisHash,
                firstStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states1), 0),
                lastState: states1[states1.length - 1],
                lastStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states1), states1.length - 1)
            }),
            abi.encodePacked(states1[states1.length - 1]),
            abi.encodePacked(uint256(0))
        );

        (bytes32[] memory states2, bytes32[] memory exp2) = appendRandomStates(genesisStates(), height1);
        bytes32 v2Id = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: blockChallengeId,
                claimId: a2,
                height: height1,
                historyRoot: MerkleTreeLib.root(exp2),
                firstState: genesisHash,
                firstStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states2), 0),
                lastState: states2[states2.length - 1],
                lastStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states2), states2.length - 1)
            }),
            abi.encodePacked(states2[states2.length - 1]),
            abi.encodePacked(uint256(0))
        );

        (bytes32[5] memory b1,) = bisectToRoot(challengeManager, v1Id, v2Id, states1, states2);

        bytes32 bigStepChallengeId =
            challengeManager.createSubChallenge(challengeManager.getVertex(b1[0]).predecessorId);

        (bytes32[] memory subStates, bytes32[] memory subExp) = appendRandomStates(genesisStates(), height1);

        // only add one leaf
        bytes32 bsLeaf1 = challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: bigStepChallengeId,
                claimId: b1[0],
                height: height1,
                historyRoot: MerkleTreeLib.root(subExp),
                firstState: subStates[0],
                firstStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(subStates), 0),
                lastState: subStates[subStates.length - 1],
                lastStatehistoryProof: ProofUtils.generateInclusionProof(
                    ProofUtils.rehashed(subStates), subStates.length - 1
                    )
            }),
            abi.encodePacked(subStates[subStates.length - 1]),
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

        // height 16
        winningVertices[3] = bisect(challengeManager, winningVertices[4], winningLeaves, 16, winningLeaves.length);
        losingVertices[3] = bisect(challengeManager, losingVertices[4], losingLeaves, 16, losingLeaves.length);

        // height 8
        winningVertices[2] = bisect(challengeManager, winningVertices[3], winningLeaves, 8, 16);
        losingVertices[2] = bisect(challengeManager, losingVertices[3], losingLeaves, 8, 16);

        // height 4
        winningVertices[1] = bisect(challengeManager, winningVertices[2], winningLeaves, 4, 8);
        losingVertices[1] = bisect(challengeManager, losingVertices[2], losingLeaves, 4, 8);

        // height 4
        winningVertices[0] = bisect(challengeManager, winningVertices[1], winningLeaves, 2, 4);
        losingVertices[0] = bisect(challengeManager, losingVertices[1], losingLeaves, 2, 4);

        return (winningVertices, losingVertices);
    }

    function addLeaf(
        IChallengeManager challengeManager,
        bytes32 challengeId,
        bytes32 claimId,
        bytes32 historyRoot,
        uint256 height,
        bytes32[] memory states,
        bytes memory proof2
    ) internal returns (bytes32) {
        return challengeManager.addLeaf{value: miniStakeVal}(
            AddLeafArgs({
                challengeId: challengeId,
                claimId: claimId,
                height: height,
                historyRoot: historyRoot,
                firstState: genesisHash,
                firstStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), 0),
                lastState: states[states.length - 1],
                lastStatehistoryProof: ProofUtils.generateInclusionProof(ProofUtils.rehashed(states), states.length - 1)
            }),
            abi.encodePacked(states[states.length - 1]),
            proof2
        );
    }

    struct AddLeafAndBisectReturns {
        bytes32[5] winningVertices;
        bytes32[5] losingVertices;
        bytes32[] states1;
        bytes32[] states2;
    }

    function addLeafsAndBisectToSubChallenge(
        IChallengeManager challengeManager,
        bytes32 challengeId,
        bytes32 claimId1,
        bytes32 claimId2,
        bytes memory addLeafProof2
    ) internal returns (AddLeafAndBisectReturns memory) {
        AddLeafAndBisectReturns memory r;
        (bytes32[] memory states1, bytes32[] memory exp1) = appendRandomStates(genesisStates(), height1);
        r.states1 = states1;
        (bytes32[] memory states2, bytes32[] memory exp2) = appendRandomStates(genesisStates(), height1);
        r.states2 = states2;

        bytes32 v1Id =
            addLeaf(challengeManager, challengeId, claimId1, MerkleTreeLib.root(exp1), height1, states1, addLeafProof2);
        bytes32 v2Id =
            addLeaf(challengeManager, challengeId, claimId2, MerkleTreeLib.root(exp2), height1, states2, addLeafProof2);

        (bytes32[5] memory challengeWinningVertices, bytes32[5] memory challengeLosingVertices) =
            bisectToRoot(challengeManager, v1Id, v2Id, r.states1, r.states2);
        r.losingVertices = challengeLosingVertices;
        r.winningVertices = challengeWinningVertices;

        return r;
    }

    function testCanConfirmFromOneStep() public {
        (, ChallengeManagerImpl challengeManager,, bytes32 a1, bytes32 a2, bytes32 blockChallengeId) =
            deployAndInitChallenge();

        AddLeafAndBisectReturns memory blockResult =
            addLeafsAndBisectToSubChallenge(challengeManager, blockChallengeId, a1, a2, abi.encodePacked(uint256(0)));

        bytes32 bigStepChallengeId = challengeManager.createSubChallenge(
            challengeManager.getVertex(blockResult.winningVertices[0]).predecessorId
        );
        AddLeafAndBisectReturns memory bigStepResult = addLeafsAndBisectToSubChallenge(
            challengeManager,
            bigStepChallengeId,
            blockResult.winningVertices[0],
            blockResult.losingVertices[0],
            abi.encodePacked(uint256(0))
        );

        bytes32 smallStepChallengeId = challengeManager.createSubChallenge(
            challengeManager.getVertex(bigStepResult.winningVertices[0]).predecessorId
        );

        AddLeafAndBisectReturns memory smallStepResult = addLeafsAndBisectToSubChallenge(
            challengeManager,
            smallStepChallengeId,
            bigStepResult.winningVertices[0],
            bigStepResult.losingVertices[0],
            abi.encodePacked(height1)
        );

        challengeManager.createSubChallenge(
            challengeManager.getVertex(smallStepResult.winningVertices[0]).predecessorId
        );
        uint256 baseHeight = challengeManager.getVertex(smallStepResult.winningVertices[0]).height - 1;
        // form the states for the history commitment of the winning states
        bytes32[] memory firstStates = new bytes32[](2);
        firstStates[0] = smallStepResult.states1[0];
        firstStates[1] = smallStepResult.states1[1];

        challengeManager.executeOneStep(
            smallStepResult.winningVertices[0],
            OneStepData({
                execCtx: ExecutionContext({maxInboxMessagesRead: 0, bridge: IBridge(address(0))}),
                machineStep: baseHeight,
                beforeHash: smallStepResult.states1[0],
                proof: abi.encodePacked(smallStepResult.states1[1])
            }),
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(genesisStates()), baseHeight),
            ProofUtils.generateInclusionProof(ProofUtils.rehashed(firstStates), baseHeight + 1)
        );

        challengeManager.confirmForSucessionChallengeWin(smallStepResult.winningVertices[0]);

        vm.warp(challengePeriodSec + 2);
        challengeManager.confirmForPsTimer(smallStepResult.winningVertices[1]);
        challengeManager.confirmForPsTimer(smallStepResult.winningVertices[2]);
        challengeManager.confirmForPsTimer(smallStepResult.winningVertices[3]);
        challengeManager.confirmForPsTimer(smallStepResult.winningVertices[4]);

        challengeManager.confirmForSucessionChallengeWin(bigStepResult.winningVertices[0]);
        challengeManager.confirmForPsTimer(bigStepResult.winningVertices[1]);
        challengeManager.confirmForPsTimer(bigStepResult.winningVertices[2]);
        challengeManager.confirmForPsTimer(bigStepResult.winningVertices[3]);
        challengeManager.confirmForPsTimer(bigStepResult.winningVertices[4]);

        challengeManager.confirmForSucessionChallengeWin(blockResult.winningVertices[0]);
        challengeManager.confirmForPsTimer(blockResult.winningVertices[1]);
        challengeManager.confirmForPsTimer(blockResult.winningVertices[2]);
        challengeManager.confirmForPsTimer(blockResult.winningVertices[3]);
        challengeManager.confirmForPsTimer(blockResult.winningVertices[4]);

        assertEq(challengeManager.winningClaim(blockChallengeId), a1);
    }
}
