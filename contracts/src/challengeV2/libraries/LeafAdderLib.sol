// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../DataEntitiesOld.sol";
import "./ChallengeVertexLib.sol";
import "./MerkleTreeLib.sol";
import "./PsVerticesLib.sol";

library LeafAdderLib {
    using PsVerticesLib for mapping(bytes32 => ChallengeVertex);

    // CHRIS: TODO: re-arrange the order of args on all these functions - we should use something consistent
    function checkAddLeaf(
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafArgs memory leafData,
        uint256 miniStake
    ) internal view {
        require(leafData.claimId != 0, "Empty claimId");
        require(leafData.historyRoot != 0, "Empty historyRoot");
        // CHRIS: TODO: we should also prove that the height is greater than 1 if we set the root heigt to 1
        require(leafData.height != 0, "Empty height");

        // CHRIS: TODO: comment on why we need the mini stake
        // CHRIS: TODO: also are we using this to refund moves in real-time? would be more expensive if so, but could be necessary?
        // CHRIS: TODO: this can apparently be moved directly to the public goods fund
        // CHRIS: TODO: we need to record who was on the winning leaf
        require(msg.value == miniStake, "Incorrect mini-stake amount");

        // CHRIS: TODO: require that this challenge hasnt declared a winner
        require(challenges[leafData.challengeId].winningClaim == 0, "Winner already declared");

        // CHRIS: TODO: also check the root is in the history at height 0/1?
        MerkleTreeLib.verifyInclusionProof(
            leafData.historyRoot, leafData.lastState, leafData.height, leafData.lastStatehistoryProof
        );

        // CHRIS: TODO: do we need to pass in first state if we can derive it from the root id?
        MerkleTreeLib.verifyInclusionProof(
            leafData.historyRoot, leafData.firstState, 0, leafData.firstStatehistoryProof
        );

        // CHRIS: TODO: we dont know the root id - this is in the challenge itself?
        // CHRIS: TODO: in the below check we're trying to ensure the root id of this challenge is the
        // first state of this commitment, we should do a prefix proof then instead of checking first leaf?
        // Do we know that it will have the correct history commitment?
        // require(
        //     challenges[leafData.challengeId].rootId
        //         == ChallengeVertexLib.id(leafData.challengeId, keccak256(abi.encodePacked(leafData.firstState)), 0),
        //     "First state is not the challenge root"
        // );
    }
}

library BlockLeafAdder {
    // CHRIS: TODO: not all these libs are used
    using ChallengeVertexLib for ChallengeVertex;
    using PsVerticesLib for mapping(bytes32 => ChallengeVertex);

    function initialPsTimeSec(bytes32 claimId, IAssertionChain assertionChain) internal view returns (uint256) {
        bool isFirstChild = assertionChain.isFirstChild(claimId);

        if (isFirstChild) {
            bytes32 predecessorId = assertionChain.getPredecessorId(claimId);
            uint256 firstChildCreationTime = assertionChain.getFirstChildCreationTime(predecessorId);

            return block.timestamp - firstChildCreationTime;
        } else {
            return 0;
        }
    }

    function getBlockHash(bytes32 assertionStateHash, bytes memory proof) internal returns (bytes32) {
        return bytes32(proof);
        // CHRIS: TODO:
        // 1. The assertion state hash contains all the info being asserted - including the block hash
        // 2. Extract the block hash from the assertion state hash using the claim proof and return it
    }

    function getInboxMsgProcessedCount(bytes32 assertionStateHash, bytes memory proof)
        internal
        view
        returns (uint256)
    {
        (GlobalState memory globalState, uint256 inboxMaxCount, MachineStatus machineStatus) =
            abi.decode(proof, (GlobalState, uint256, MachineStatus));

        // CHRIS: TODO: use the stateHash from RollupLib? it would mean importing RollupLib here, is that ok?
        // CHRIS: TODO: try add later after circular dependencies fixed in rollup
        bytes32 stateHash = keccak256(abi.encodePacked(GlobalStateLib.hash(globalState), inboxMaxCount, machineStatus));
        require(stateHash == assertionStateHash, "Invalid state hash from proof");

        return GlobalStateLib.getInboxPosition(globalState);
    }

    function addLeaf(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafLibArgs memory leafLibArgs, // CHRIS: TODO: better name
        IAssertionChain assertionChain
    ) internal returns (bytes32) {
        {
            // check that the predecessor of this claim has registered this contract as it's succession challenge
            bytes32 predecessorId = assertionChain.getPredecessorId(leafLibArgs.leafData.claimId);
            require(
                assertionChain.hasSibling(leafLibArgs.leafData.claimId),
                "Claim predecessor not linked to this challenge"
            );

            uint256 assertionHeight = assertionChain.getHeight(leafLibArgs.leafData.claimId);
            uint256 predecessorAssertionHeight = assertionChain.getHeight(predecessorId);

            uint256 heightDiff = assertionHeight - predecessorAssertionHeight;
            require(heightDiff == leafLibArgs.leafData.height, "Invalid leaf height");

            // bytes32 claimStateHash = assertionChain.getChallengeHash(leafLibArgs.leafData.claimId);

            // RJ: TODO: provide valid inbox proofs from the validator client
            // require(
            //     getInboxMsgProcessedCount(claimStateHash, leafLibArgs.proof2)
            //         == assertionChain.getInboxMsgCountSeen(predecessorId),
            //     "Invalid inbox messages processed"
            // );

            // CHRIS: TODO: Implement the block hash check
            // require(
            //     getBlockHash(claimStateHash, leafLibArgs.proof1) == leafLibArgs.leafData.lastState,
            //     "Last state is not the assertion claim block hash"
            // );

            LeafAdderLib.checkAddLeaf(challenges, leafLibArgs.leafData, leafLibArgs.miniStake);
        }

        ChallengeVertex memory leaf = ChallengeVertexLib.newLeaf(
            leafLibArgs.leafData.challengeId,
            leafLibArgs.leafData.historyRoot,
            leafLibArgs.leafData.height,
            leafLibArgs.leafData.claimId,
            msg.sender,
            initialPsTimeSec(leafLibArgs.leafData.claimId, assertionChain)
        );

        return vertices.addVertex(
            leaf, challenges[leafLibArgs.leafData.challengeId].rootId, leafLibArgs.challengePeriodSec
        );
    }

    // CHRIS: TODO: check exists whenever we access the challenges? also the vertices now have a challenge index
}

library BigStepLeafAdder {
    using ChallengeVertexLib for ChallengeVertex;
    using PsVerticesLib for mapping(bytes32 => ChallengeVertex);

    function getBlockHashFromClaim(bytes32 claimId, bytes memory claimProof) internal returns (bytes32) {
        // CHRIS: TODO:
        // 1. Get the history commitment for this claim
        // 2. Unwrap the last state of the claim using the proof
        // 3. Also use the proof to extract the block hash from the last state
        // 4. Return the block hash
    }

    function getBlockHashProducedByTerminalState(bytes32 state, bytes memory stateProof) internal returns (bytes32) {
        // 1. Hydrate the state using the state proof
        // 2. Show that the state is terminal
        // 3. Extract the block hash that is being produced by this terminal state
    }

    function addLeaf(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafLibArgs memory leafLibArgs // CHRIS: TODO: better name
    ) internal returns (bytes32) {
        {
            // CHRIS: TODO: we should only have the special stuff in here, we can pass in the initial ps timer or something
            // CHRIS: TODO: rename challenge to challenge manager
            require(vertices[leafLibArgs.leafData.claimId].exists(), "Claim does not exist");
            bytes32 predecessorId = vertices[leafLibArgs.leafData.claimId].predecessorId;
            require(vertices[predecessorId].exists(), "Claim predecessor does not exist");
            require(
                vertices[leafLibArgs.leafData.claimId].height - vertices[predecessorId].height == 1,
                "Claim not height one above predecessor"
            );
            require(
                vertices[predecessorId].successionChallenge == leafLibArgs.leafData.challengeId,
                "Claim has invalid succession challenge"
            );

            // CHRIS: TODO: check challenge also exists

            // CHRIS: TODO: also check that the claim is a block hash?

            // in a bigstep challenge the states are wasm states, and the claims are block challenge vertices
            // check that the wasm state is a terminal state, and that it produces the blockhash that's in the claim
            bytes32 lastStateBlockHash =
                getBlockHashProducedByTerminalState(leafLibArgs.leafData.lastState, leafLibArgs.proof1);
            bytes32 claimBlockHash = getBlockHashFromClaim(leafLibArgs.leafData.claimId, leafLibArgs.proof2);

            require(claimBlockHash == lastStateBlockHash, "Claim inconsistent with state");

            LeafAdderLib.checkAddLeaf(challenges, leafLibArgs.leafData, leafLibArgs.miniStake);
        }

        ChallengeVertex memory leaf = ChallengeVertexLib.newLeaf(
            leafLibArgs.leafData.challengeId,
            leafLibArgs.leafData.historyRoot,
            leafLibArgs.leafData.height,
            leafLibArgs.leafData.claimId,
            msg.sender,
            vertices.getCurrentPsTimer(leafLibArgs.leafData.claimId)
        );

        return vertices.addVertex(
            leaf, challenges[leafLibArgs.leafData.challengeId].rootId, leafLibArgs.challengePeriodSec
        );
    }
}

library SmallStepLeafAdder {
    using ChallengeVertexLib for ChallengeVertex;
    using PsVerticesLib for mapping(bytes32 => ChallengeVertex);

    uint256 public constant MAX_STEPS = 2 << 19;

    function getProgramCounter(bytes32 state, bytes memory proof) internal returns (uint256) {
        // CHRIS: TODO:
        // 1. hydrate the wavm state with the proof
        // 2. find the program counter and return it
        return uint256(bytes32(proof));
    }

    function addLeaf(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        mapping(bytes32 => Challenge) storage challenges,
        AddLeafLibArgs memory leafLibArgs
    ) internal returns (bytes32) {
        {
            require(vertices[leafLibArgs.leafData.claimId].exists(), "Claim does not exist");
            bytes32 predecessorId = vertices[leafLibArgs.leafData.claimId].predecessorId;
            require(vertices[predecessorId].exists(), "Claim predecessor does not exist");
            require(
                vertices[leafLibArgs.leafData.claimId].height - vertices[predecessorId].height == 1,
                "Claim not height one above predecessor"
            );
            require(
                vertices[predecessorId].successionChallenge == leafLibArgs.leafData.challengeId,
                "Claim has invalid succession challenge"
            );

            // CHRIS: TODO: should call it "claimChallengeId";

            // the wavm state of the last state should always be exactly the same as the wavm state of the claim
            // regardless of the height
            // CHRIS: TODO: skip the test for now, as I'm not sure it's relevant
            // require(
            //     MerkleTreeLib.verifyInclusionProof(
            //         vertices[leafLibArgs.leafData.claimId].historyRoot,
            //         leafLibArgs.leafData.lastState,
            //         vertices[leafLibArgs.leafData.claimId].height,
            //         leafLibArgs.proof1
            //     ),
            //     "Invalid claim state"
            // );

            // CHRIS: TODO: document and align the proogs
            uint256 lastStateProgramCounter = getProgramCounter(leafLibArgs.leafData.lastState, leafLibArgs.proof2);
            uint256 predecessorSteps = vertices[predecessorId].height * MAX_STEPS;

            // RAUL: TODO: Re-enable this check
            // require(
            //     predecessorSteps + leafLibArgs.leafData.height == lastStateProgramCounter,
            //     "Inconsistent program counter"
            // );

            // CHRIS: TODO: re-enable this leaf check
            // if (!ChallengeVertexLib.isLeaf(vertices[leafLibArgs.leafData.claimId])) {
            //     require(leafLibArgs.leafData.height == MAX_STEPS, "Invalid non-leaf steps");
            // } else {
            //     require(leafLibArgs.leafData.height <= MAX_STEPS, "Invalid leaf steps");
            // }

            LeafAdderLib.checkAddLeaf(challenges, leafLibArgs.leafData, leafLibArgs.miniStake);
        }

        ChallengeVertex memory leaf = ChallengeVertexLib.newLeaf(
            leafLibArgs.leafData.challengeId,
            leafLibArgs.leafData.historyRoot,
            leafLibArgs.leafData.height,
            leafLibArgs.leafData.claimId,
            msg.sender,
            vertices.getCurrentPsTimer(leafLibArgs.leafData.claimId)
        );

        return vertices.addVertex(
            leaf, challenges[leafLibArgs.leafData.challengeId].rootId, leafLibArgs.challengePeriodSec
        );
    }
}
